/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package middleware

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/collect/configuration"
	"github.com/stretchr/testify/assert"
)

func TestBasicAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set required environment variables for configuration init
	_ = os.Setenv("DATABASE_URL", "postgres://localhost:5432/testdb")
	defer func() { _ = os.Unsetenv("DATABASE_URL") }()

	// Initialize configuration for testing
	configuration.Init()

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedAuth   bool
	}{
		{
			name:           "missing authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedAuth:   false,
		},
		{
			name:           "invalid authorization format",
			authHeader:     "Bearer token123",
			expectedStatus: http.StatusUnauthorized,
			expectedAuth:   false,
		},
		{
			name:           "invalid base64 encoding",
			authHeader:     "Basic invalid-base64!",
			expectedStatus: http.StatusUnauthorized,
			expectedAuth:   false,
		},
		{
			name:           "invalid credentials format - no colon",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("invalidformat")),
			expectedStatus: http.StatusUnauthorized,
			expectedAuth:   false,
		},
		{
			name:           "valid format but password too short",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("system1:short")),
			expectedStatus: http.StatusUnauthorized,
			expectedAuth:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(BasicAuthMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusUnauthorized {
				assert.Contains(t, w.Header().Get("WWW-Authenticate"), "Basic realm")
			}
		})
	}
}

func TestHashSystemSecret(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		expected string
	}{
		{
			name:     "empty secret",
			secret:   "",
			expected: fmt.Sprintf("%x", sha256.Sum256([]byte(""))),
		},
		{
			name:     "simple secret",
			secret:   "password123",
			expected: fmt.Sprintf("%x", sha256.Sum256([]byte("password123"))),
		},
		{
			name:     "long secret",
			secret:   "this-is-a-very-long-secret-key-for-testing-purposes",
			expected: fmt.Sprintf("%x", sha256.Sum256([]byte("this-is-a-very-long-secret-key-for-testing-purposes"))),
		},
		{
			name:     "secret with special characters",
			secret:   "secret!@#$%^&*()_+-=[]{}|;:,.<>?",
			expected: fmt.Sprintf("%x", sha256.Sum256([]byte("secret!@#$%^&*()_+-=[]{}|;:,.<>?"))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hashSystemSecret(tt.secret)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashSystemSecretConsistency(t *testing.T) {
	secret := "consistent-secret-key"

	// Hash the same secret multiple times
	hash1 := hashSystemSecret(secret)
	hash2 := hashSystemSecret(secret)
	hash3 := hashSystemSecret(secret)

	// All hashes should be identical
	assert.Equal(t, hash1, hash2)
	assert.Equal(t, hash2, hash3)
	assert.Equal(t, hash1, hash3)
}

func TestHashSystemSecretDifferentInputs(t *testing.T) {
	secret1 := "secret1"
	secret2 := "secret2"

	hash1 := hashSystemSecret(secret1)
	hash2 := hashSystemSecret(secret2)

	// Different inputs should produce different hashes
	assert.NotEqual(t, hash1, hash2)
}

func TestHashSystemSecretFormat(t *testing.T) {
	secret := "test-secret"
	hash := hashSystemSecret(secret)

	// Hash should be 64 characters long (256 bits / 4 bits per hex char)
	assert.Equal(t, 64, len(hash))

	// Hash should only contain hex characters
	for _, char := range hash {
		assert.True(t, (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f'),
			"Hash should only contain hex characters, found: %c", char)
	}
}

func TestBasicAuthMiddlewareHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(BasicAuthMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	tests := []struct {
		name          string
		authHeader    string
		expectWWWAuth bool
	}{
		{
			name:          "missing auth header",
			authHeader:    "",
			expectWWWAuth: true,
		},
		{
			name:          "invalid auth format",
			authHeader:    "Bearer token",
			expectWWWAuth: true,
		},
		{
			name:          "invalid base64",
			authHeader:    "Basic invalid!",
			expectWWWAuth: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tt.expectWWWAuth {
				wwwAuth := w.Header().Get("WWW-Authenticate")
				assert.Contains(t, wwwAuth, "Basic realm")
				assert.Contains(t, wwwAuth, "System Authentication")
			}
		})
	}
}

func TestMiddlewareContextSetting(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	router.Use(func(c *gin.Context) {
		// Simulate successful authentication
		c.Set("system_id", "test-system-001")
		c.Set("authenticated_system", true)
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		systemID, exists := c.Get("system_id")
		authStatus, authExists := c.Get("authenticated_system")

		assert.True(t, exists)
		assert.True(t, authExists)
		assert.Equal(t, "test-system-001", systemID)
		assert.Equal(t, true, authStatus)

		c.JSON(http.StatusOK, gin.H{
			"system_id":     systemID,
			"authenticated": authStatus,
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
