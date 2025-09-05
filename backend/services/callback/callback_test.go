/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package callback

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
)

func TestCallbackService_ExecuteSystemCreationCallback(t *testing.T) {
	// Create a test system
	testSystem := models.System{
		ID:        "sys_test123",
		Name:      "Test System",
		Type:      "ns8",
		CreatedAt: time.Now(),
	}

	t.Run("successful callback", func(t *testing.T) {
		// Create test server that accepts the callback
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "My-Nethesis/1.0", r.Header.Get("User-Agent"))

			// Check query parameters (OAuth-style callback)
			query := r.URL.Query()
			assert.Equal(t, "test_state_123", query.Get("state"))
			assert.Equal(t, "sys_test123", query.Get("system_id"))
			assert.Equal(t, "Test System", query.Get("system_name"))
			assert.Equal(t, "ns8", query.Get("system_type"))
			assert.NotEmpty(t, query.Get("timestamp"))

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		service := NewCallbackService()
		success := service.ExecuteSystemCreationCallback(server.URL, "test_state_123", testSystem)
		assert.True(t, success)
	})

	t.Run("callback returns error status", func(t *testing.T) {
		// Create test server that returns error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		service := NewCallbackService()
		success := service.ExecuteSystemCreationCallback(server.URL, "test_state_123", testSystem)
		assert.False(t, success)
	})

	t.Run("unreachable callback URL", func(t *testing.T) {
		service := NewCallbackService()
		success := service.ExecuteSystemCreationCallback("http://localhost:99999/callback", "test_state_123", testSystem)
		assert.False(t, success)
	})
}
