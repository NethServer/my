/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGetAuthenticatedSystemID(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(c *gin.Context)
		wantID string
		wantOK bool
	}{
		{
			name: "valid system_id in context",
			setup: func(c *gin.Context) {
				c.Set("system_id", "abc-123-def")
			},
			wantID: "abc-123-def",
			wantOK: true,
		},
		{
			name:   "missing system_id",
			setup:  func(c *gin.Context) {},
			wantID: "",
			wantOK: false,
		},
		{
			name: "system_id is not a string",
			setup: func(c *gin.Context) {
				c.Set("system_id", 12345)
			},
			wantID: "",
			wantOK: false,
		},
		{
			name: "system_id is nil",
			setup: func(c *gin.Context) {
				c.Set("system_id", nil)
			},
			wantID: "",
			wantOK: false,
		},
		{
			name: "empty string system_id",
			setup: func(c *gin.Context) {
				c.Set("system_id", "")
			},
			wantID: "",
			wantOK: true,
		},
		{
			name: "uuid format system_id",
			setup: func(c *gin.Context) {
				c.Set("system_id", "943d3ea6-fa69-4f14-8be0-a7c51250510f")
			},
			wantID: "943d3ea6-fa69-4f14-8be0-a7c51250510f",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

			tt.setup(c)

			gotID, gotOK := getAuthenticatedSystemID(c)
			assert.Equal(t, tt.wantID, gotID)
			assert.Equal(t, tt.wantOK, gotOK)
		})
	}
}

func TestGetAuthenticatedSystemIDIntegration(t *testing.T) {
	router := gin.New()

	// Middleware that sets system_id (simulates BasicAuthMiddleware)
	router.Use(func(c *gin.Context) {
		c.Set("system_id", "test-system-integration")
		c.Next()
	})

	var capturedID string
	var capturedOK bool

	router.GET("/test", func(c *gin.Context) {
		capturedID, capturedOK = getAuthenticatedSystemID(c)
		c.JSON(http.StatusOK, gin.H{"id": capturedID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test-system-integration", capturedID)
	assert.True(t, capturedOK)
}
