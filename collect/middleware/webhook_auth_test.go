/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/collect/configuration"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestWebhookAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		secret         string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "valid token",
			secret:         "test-secret-123",
			authHeader:     "Bearer test-secret-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing authorization header",
			secret:         "test-secret-123",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			secret:         "test-secret-123",
			authHeader:     "Bearer wrong-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "basic auth instead of bearer",
			secret:         "test-secret-123",
			authHeader:     "Basic dXNlcjpwYXNz",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "bearer prefix missing",
			secret:         "test-secret-123",
			authHeader:     "test-secret-123",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "secret not configured",
			secret:         "",
			authHeader:     "Bearer any-token",
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "empty bearer token",
			secret:         "test-secret-123",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configuration.Config.AlertmanagerWebhookSecret = tt.secret

			router := gin.New()
			router.Use(WebhookAuthMiddleware())
			router.POST("/webhook", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestWebhookAuthMiddleware_TimingSafe(t *testing.T) {
	// Verify that a token differing by one character is rejected
	configuration.Config.AlertmanagerWebhookSecret = "correct-secret-token"

	router := gin.New()
	router.Use(WebhookAuthMiddleware())
	router.POST("/webhook", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("Authorization", "Bearer correct-secret-tokeN") // last char differs

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWebhookAuthMiddleware_ResponseFormat(t *testing.T) {
	configuration.Config.AlertmanagerWebhookSecret = "test-secret"

	router := gin.New()
	router.Use(WebhookAuthMiddleware())
	router.POST("/webhook", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test 401 response body
	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, float64(http.StatusUnauthorized), resp["code"])
	assert.Equal(t, "authentication required", resp["message"])

	// Test 503 response body (unconfigured)
	configuration.Config.AlertmanagerWebhookSecret = ""
	req2 := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req2.Header.Set("Authorization", "Bearer any")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	var resp2 map[string]interface{}
	err = json.Unmarshal(w2.Body.Bytes(), &resp2)
	assert.NoError(t, err)
	assert.Equal(t, float64(http.StatusServiceUnavailable), resp2["code"])
}
