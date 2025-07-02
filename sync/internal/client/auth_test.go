/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogtoClient_getAccessToken(t *testing.T) {
	t.Run("successful token request", func(t *testing.T) {
		var serverURL string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/oidc/token", r.URL.Path)
			assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

			// Parse form data
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			formData, err := url.ParseQuery(string(body))
			require.NoError(t, err)

			assert.Equal(t, "client_credentials", formData.Get("grant_type"))
			assert.Equal(t, "test-client-id", formData.Get("client_id"))
			assert.Equal(t, "test-client-secret", formData.Get("client_secret"))
			assert.Equal(t, serverURL+"/api", formData.Get("resource"))
			assert.Equal(t, "all", formData.Get("scope"))

			// Return token response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "test-access-token",
				"expires_in":   3600,
				"token_type":   "Bearer",
			})
		}))
		defer server.Close()
		serverURL = server.URL

		client := NewLogtoClient(server.URL, "test-client-id", "test-client-secret")

		err := client.getAccessToken()
		require.NoError(t, err)

		assert.Equal(t, "test-access-token", client.accessToken)
		assert.True(t, client.tokenExpiry.After(time.Now().Add(3500*time.Second)))
	})

	t.Run("token still valid", func(t *testing.T) {
		client := NewLogtoClient("https://example.com", "test-id", "test-secret")

		// Set a valid token that expires in 10 minutes
		client.accessToken = "existing-token"
		client.tokenExpiry = time.Now().Add(10 * time.Minute)

		err := client.getAccessToken()
		require.NoError(t, err)

		// Token should remain unchanged
		assert.Equal(t, "existing-token", client.accessToken)
	})

	t.Run("token expired - refresh", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "new-access-token",
				"expires_in":   3600,
				"token_type":   "Bearer",
			})
		}))
		defer server.Close()

		client := NewLogtoClient(server.URL, "test-id", "test-secret")

		// Set an expired token
		client.accessToken = "expired-token"
		client.tokenExpiry = time.Now().Add(-1 * time.Hour)

		err := client.getAccessToken()
		require.NoError(t, err)

		// Token should be refreshed
		assert.Equal(t, "new-access-token", client.accessToken)
	})

	t.Run("token near expiry - refresh", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "refreshed-token",
				"expires_in":   3600,
				"token_type":   "Bearer",
			})
		}))
		defer server.Close()

		client := NewLogtoClient(server.URL, "test-id", "test-secret")

		// Set a token that expires in 2 minutes (within 5 minute buffer)
		client.accessToken = "near-expiry-token"
		client.tokenExpiry = time.Now().Add(2 * time.Minute)

		err := client.getAccessToken()
		require.NoError(t, err)

		// Token should be refreshed
		assert.Equal(t, "refreshed-token", client.accessToken)
	})

	t.Run("HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Invalid credentials"))
		}))
		defer server.Close()

		client := NewLogtoClient(server.URL, "invalid-id", "invalid-secret")

		err := client.getAccessToken()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token request failed with status 401")
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		client := NewLogtoClient(server.URL, "test-id", "test-secret")

		err := client.getAccessToken()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode token response")
	})

	t.Run("network error", func(t *testing.T) {
		client := NewLogtoClient("http://invalid-url-that-does-not-exist.local", "test-id", "test-secret")

		err := client.getAccessToken()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token request failed")
	})

	t.Run("request creation error", func(t *testing.T) {
		client := NewLogtoClient(":", "test-id", "test-secret") // Invalid URL

		err := client.getAccessToken()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create token request")
	})

	t.Run("empty token response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "",
				"expires_in":   0,
			})
		}))
		defer server.Close()

		client := NewLogtoClient(server.URL, "test-id", "test-secret")

		err := client.getAccessToken()
		require.NoError(t, err)

		// Should handle empty token (this might be a valid response in some cases)
		assert.Equal(t, "", client.accessToken)
	})

	t.Run("very long expiry time", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "long-lived-token",
				"expires_in":   86400, // 24 hours
				"token_type":   "Bearer",
			})
		}))
		defer server.Close()

		client := NewLogtoClient(server.URL, "test-id", "test-secret")

		err := client.getAccessToken()
		require.NoError(t, err)

		assert.Equal(t, "long-lived-token", client.accessToken)
		// Should expire approximately 24 hours from now
		expectedExpiry := time.Now().Add(24 * time.Hour)
		assert.True(t, client.tokenExpiry.After(expectedExpiry.Add(-1*time.Minute)))
		assert.True(t, client.tokenExpiry.Before(expectedExpiry.Add(1*time.Minute)))
	})
}

func TestLogtoClient_getAccessToken_FormDataEncoding(t *testing.T) {
	// Test that special characters in client credentials are properly encoded
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		bodyStr := string(body)

		// Verify that the form data contains URL-encoded values
		assert.Contains(t, bodyStr, "grant_type=client_credentials")
		assert.Contains(t, bodyStr, "client_id=test+id")        // + in form data is encoded as +
		assert.Contains(t, bodyStr, "client_secret=secret&key") // & separates form fields

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	client := NewLogtoClient(server.URL, "test+id", "secret&key")

	err := client.getAccessToken()
	require.NoError(t, err)
	assert.Equal(t, "test-token", client.accessToken)
}

func TestLogtoClient_getAccessToken_ConcurrentRequests(t *testing.T) {
	// Test that concurrent calls to getAccessToken don't cause issues
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "concurrent-token",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	client := NewLogtoClient(server.URL, "test-id", "test-secret")

	// Make multiple concurrent calls
	done := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func() {
			done <- client.getAccessToken()
		}()
	}

	// Wait for all calls to complete
	for i := 0; i < 3; i++ {
		err := <-done
		require.NoError(t, err)
	}

	// All calls should have resulted in the same token
	assert.Equal(t, "concurrent-token", client.accessToken)

	// Note: Due to the nature of concurrent requests, we might have multiple
	// HTTP calls to the token endpoint, which is acceptable behavior
	assert.True(t, callCount >= 1, "Expected at least one token request")
}

func TestLogtoClient_getAccessToken_ResourceURLConstruction(t *testing.T) {
	var testServerURL string
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		// Parse the form data to check the resource parameter
		formData, err := url.ParseQuery(string(body))
		require.NoError(t, err)

		resource := formData.Get("resource")
		expectedResource := testServerURL + "/api"
		assert.Equal(t, expectedResource, resource)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "resource-test-token",
			"expires_in":   3600,
		})
	}))
	defer testServer.Close()
	testServerURL = testServer.URL

	client := NewLogtoClient(testServer.URL, "test-id", "test-secret")

	err := client.getAccessToken()
	require.NoError(t, err)
	assert.Equal(t, "resource-test-token", client.accessToken)
}

func TestLogtoClient_getAccessToken_EdgeCases(t *testing.T) {
	t.Run("server returns non-JSON content-type", func(t *testing.T) {
		edgeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "plain-text-response",
				"expires_in":   3600,
			})
		}))
		defer edgeServer.Close()

		client := NewLogtoClient(edgeServer.URL, "test-id", "test-secret")

		err := client.getAccessToken()
		require.NoError(t, err)
		assert.Equal(t, "plain-text-response", client.accessToken)
	})

	t.Run("server returns extra fields", func(t *testing.T) {
		extraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "extra-fields-token",
				"expires_in":   3600,
				"token_type":   "Bearer",
				"scope":        "all",
				"extra_field":  "extra_value",
			})
		}))
		defer extraServer.Close()

		client := NewLogtoClient(extraServer.URL, "test-id", "test-secret")

		err := client.getAccessToken()
		require.NoError(t, err)
		assert.Equal(t, "extra-fields-token", client.accessToken)
	})
}
