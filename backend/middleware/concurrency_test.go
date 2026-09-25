/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimitConcurrency(t *testing.T) {
	gin.SetMode(gin.TestMode)

	release := make(chan struct{})
	started := make(chan struct{}, 10)
	r := gin.New()
	r.GET("/slow", LimitConcurrency("test", 2, 5*time.Second), func(c *gin.Context) {
		started <- struct{}{}
		<-release
		c.Status(http.StatusOK)
	})

	// Two requests occupy both slots and block inside the handler.
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/slow", nil))
			codes <- w.Code
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("handlers did not start")
		}
	}

	// The third is rejected at once with 429 and a Retry-After hint.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/slow", nil))
	require.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "5", w.Header().Get("Retry-After"))
	assert.Contains(t, w.Body.String(), "retry_after_seconds")

	// Once a slot frees up, requests pass again.
	close(release)
	wg.Wait()
	for i := 0; i < 2; i++ {
		assert.Equal(t, http.StatusOK, <-codes)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/slow", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLimitExportsIsShared(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Every export route gets the same limiter, so saturating it through one
	// collection rejects an export of another collection too.
	release := make(chan struct{})
	started := make(chan struct{}, 10)
	r := gin.New()
	block := func(c *gin.Context) {
		started <- struct{}{}
		<-release
		c.Status(http.StatusOK)
	}
	r.GET("/systems/export", LimitExports(), block)
	r.GET("/customers/export", LimitExports(), block)

	var wg sync.WaitGroup
	for i := 0; i < exportConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/systems/export", nil))
		}()
	}
	for i := 0; i < exportConcurrency; i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("handlers did not start")
		}
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/customers/export", nil))
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	close(release)
	wg.Wait()
}
