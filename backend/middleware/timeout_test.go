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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A browser accepts gzip, so the gzip middleware wraps c.Writer with a type
// that has no Unwrap: the deadline must still reach the connection.
func TestExtendDeadlineReachesTheConnectionThroughGzip(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var viaCurrent, viaRemembered error
	r := gin.New()
	r.Use(RememberRawWriter())
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.GET("/export", ExtendDeadline(time.Minute), func(c *gin.Context) {
		deadline := time.Now().Add(time.Minute)
		viaCurrent = http.NewResponseController(c.Writer).SetWriteDeadline(deadline)
		viaRemembered = http.NewResponseController(deadlineWriter(c)).SetWriteDeadline(deadline)
		c.String(http.StatusOK, "ok")
	})

	srv := httptest.NewServer(r)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/export", nil)
	require.NoError(t, err)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"), "the response went through the gzip wrapper")
	assert.True(t, errors.Is(viaCurrent, http.ErrNotSupported), "the gzip wrapper hides the connection: %v", viaCurrent)
	assert.NoError(t, viaRemembered, "the remembered writer reaches it")
}

func TestDeadlineWriterFallsBackToCurrentWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	assert.Equal(t, c.Writer, deadlineWriter(c))
}
