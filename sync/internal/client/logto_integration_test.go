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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nethesis/my/sync/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogtoClient(t *testing.T) {
	client := NewLogtoClient("https://example.com", "test-id", "test-secret")

	assert.Equal(t, "https://example.com", client.BaseURL)
	assert.Equal(t, "test-id", client.ClientID)
	assert.Equal(t, "test-secret", client.ClientSecret)
	assert.NotNil(t, client.HTTPClient)
	assert.Equal(t, constants.DefaultHTTPTimeout*time.Second, client.HTTPClient.Timeout)
}

func TestLogtoClient_makeRequest(t *testing.T) {
	// Combined server that handles both auth and API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oidc/token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "test-token",
				"expires_in":   3600,
			})
			return
		}

		// Check authorization header for API calls
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Check content type for POST requests
		if r.Method == "POST" && r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch r.URL.Path {
		case "/api/test":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "success"})
		case "/api/test-post":
			// Read and validate request body
			body, _ := io.ReadAll(r.Body)
			var data map[string]interface{}
			if err := json.Unmarshal(body, &data); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(data)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewLogtoClient(server.URL, "test-id", "test-secret")

	t.Run("GET request success", func(t *testing.T) {
		resp, err := client.makeRequest("GET", "/api/test", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		_ = resp.Body.Close()
	})

	t.Run("POST request with body", func(t *testing.T) {
		body := map[string]interface{}{
			"name":  "test",
			"value": "data",
		}

		resp, err := client.makeRequest("POST", "/api/test-post", body)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		_ = resp.Body.Close()
	})

	t.Run("404 error", func(t *testing.T) {
		resp, err := client.makeRequest("GET", "/api/nonexistent", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		_ = resp.Body.Close()
	})
}

func TestLogtoClient_handleResponse(t *testing.T) {
	client := NewLogtoClient("https://example.com", "test-id", "test-secret")

	t.Run("success with JSON response", func(t *testing.T) {
		jsonData := `{"name": "test", "id": "123"}`
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(jsonData)),
		}

		var result map[string]interface{}
		err := client.handleResponse(resp, http.StatusOK, &result)
		require.NoError(t, err)
		assert.Equal(t, "test", result["name"])
		assert.Equal(t, "123", result["id"])
	})

	t.Run("success with no body", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
		}

		err := client.handleResponse(resp, http.StatusOK, nil)
		require.NoError(t, err)
	})

	t.Run("unexpected status code", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader("Bad request")),
		}

		err := client.handleResponse(resp, http.StatusOK, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status 400")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("invalid json")),
		}

		var result map[string]interface{}
		err := client.handleResponse(resp, http.StatusOK, &result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal response")
	})
}

func TestLogtoClient_handleCreationResponse(t *testing.T) {
	client := NewLogtoClient("https://example.com", "test-id", "test-secret")

	t.Run("201 Created", func(t *testing.T) {
		jsonData := `{"name": "test", "id": "123"}`
		resp := &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader(jsonData)),
		}

		var result map[string]interface{}
		err := client.handleCreationResponse(resp, &result)
		require.NoError(t, err)
		assert.Equal(t, "test", result["name"])
	})

	t.Run("200 OK", func(t *testing.T) {
		jsonData := `{"name": "test", "id": "123"}`
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(jsonData)),
		}

		var result map[string]interface{}
		err := client.handleCreationResponse(resp, &result)
		require.NoError(t, err)
		assert.Equal(t, "test", result["name"])
	})

	t.Run("unexpected status", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader("Bad request")),
		}

		err := client.handleCreationResponse(resp, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status 400")
	})
}

func TestLogtoClient_handlePaginatedResponse(t *testing.T) {
	client := NewLogtoClient("https://example.com", "test-id", "test-secret")

	t.Run("paginated response", func(t *testing.T) {
		jsonData := `{"data": [{"name": "test1"}, {"name": "test2"}]}`
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(jsonData)),
		}

		var result []map[string]interface{}
		err := client.handlePaginatedResponse(resp, &result)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "test1", result[0]["name"])
		assert.Equal(t, "test2", result[1]["name"])
	})

	t.Run("direct array response", func(t *testing.T) {
		jsonData := `[{"name": "test1"}, {"name": "test2"}]`
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(jsonData)),
		}

		var result []map[string]interface{}
		err := client.handlePaginatedResponse(resp, &result)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "test1", result[0]["name"])
	})

	t.Run("unexpected status", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader("Bad request")),
		}

		var result []map[string]interface{}
		err := client.handlePaginatedResponse(resp, &result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status 400")
	})
}

func TestFindEntityByField(t *testing.T) {
	entities := []map[string]interface{}{
		{"id": "1", "name": "entity1"},
		{"id": "2", "name": "entity2"},
		{"id": "3", "name": "entity3"},
	}

	t.Run("found entity", func(t *testing.T) {
		entity, found := FindEntityByField(entities, "name", "entity2")
		assert.True(t, found)
		assert.Equal(t, "2", entity["id"])
		assert.Equal(t, "entity2", entity["name"])
	})

	t.Run("not found", func(t *testing.T) {
		entity, found := FindEntityByField(entities, "name", "nonexistent")
		assert.False(t, found)
		assert.Nil(t, entity)
	})

	t.Run("field not exists", func(t *testing.T) {
		entity, found := FindEntityByField(entities, "nonexistent", "value")
		assert.False(t, found)
		assert.Nil(t, entity)
	})
}

func TestFindEntityID(t *testing.T) {
	entities := []map[string]interface{}{
		{"id": "1", "name": "entity1"},
		{"id": "2", "name": "entity2"},
		{"id": 3, "name": "entity3"}, // Non-string ID
	}

	t.Run("found ID", func(t *testing.T) {
		id, found := FindEntityID(entities, "name", "entity2")
		assert.True(t, found)
		assert.Equal(t, "2", id)
	})

	t.Run("not found", func(t *testing.T) {
		id, found := FindEntityID(entities, "name", "nonexistent")
		assert.False(t, found)
		assert.Empty(t, id)
	})

	t.Run("non-string ID", func(t *testing.T) {
		id, found := FindEntityID(entities, "name", "entity3")
		assert.False(t, found)
		assert.Empty(t, id)
	})
}

func TestLogtoClient_createEntitySimple(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oidc/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "test-token",
				"expires_in":   3600,
			})
			return
		}

		if r.URL.Path == "/api/entities" && r.Method == "POST" {
			var data map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&data)

			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(data)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewLogtoClient(server.URL, "test-id", "test-secret")
	client.BaseURL = server.URL

	entityData := map[string]interface{}{
		"name": "test-entity",
		"type": "test",
	}

	err := client.createEntitySimple("/api/entities", entityData, "entity")
	require.NoError(t, err)
}

// Mock server for integration tests
func createMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Auth endpoint
		if r.URL.Path == "/oidc/token" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "mock-token",
				"expires_in":   3600,
			})
			return
		}

		// Check authorization
		if r.Header.Get("Authorization") != "Bearer mock-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case "/api/resources":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "1", "name": "resource1"},
			})
		case "/api/applications":
			switch r.Method {
			case "GET":
				_ = json.NewEncoder(w).Encode([]map[string]interface{}{
					{"id": "app1", "name": "test-app"},
				})
			case "POST":
				var app map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&app)
				app["id"] = "new-app-id"
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(app)
			}
		case "/api/users":
			switch r.Method {
			case "GET":
				_ = json.NewEncoder(w).Encode([]map[string]interface{}{
					{"id": "user1", "username": "testuser"},
				})
			case "POST":
				var user map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&user)
				user["id"] = "new-user-id"
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(user)
			}
		case "/api/domains":
			switch r.Method {
			case "GET":
				_ = json.NewEncoder(w).Encode([]map[string]interface{}{
					{"id": "domain1", "domain": "example.com"},
				})
			case "POST":
				var domain map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&domain)
				domain["id"] = "new-domain-id"
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(domain)
			}
		default:
			if strings.HasPrefix(r.URL.Path, "/api/users/") && strings.HasSuffix(r.URL.Path, "/password") {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestLogtoClient_APIEndpoints(t *testing.T) {
	server := createMockServer()
	defer server.Close()

	client := NewLogtoClient(server.URL, "test-id", "test-secret")
	client.BaseURL = server.URL

	t.Run("TestConnection", func(t *testing.T) {
		err := client.TestConnection()
		require.NoError(t, err)
	})

	t.Run("GetApplications", func(t *testing.T) {
		apps, err := client.GetApplications()
		require.NoError(t, err)
		assert.Len(t, apps, 1)
		assert.Equal(t, "test-app", apps[0]["name"])
	})

	t.Run("CreateApplication", func(t *testing.T) {
		app := map[string]interface{}{
			"name": "new-app",
			"type": "web",
		}

		result, err := client.CreateApplication(app)
		require.NoError(t, err)
		assert.Equal(t, "new-app-id", result["id"])
		assert.Equal(t, "new-app", result["name"])
	})

	t.Run("GetAllUsers", func(t *testing.T) {
		users, err := client.GetAllUsers()
		require.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, "testuser", users[0]["username"])
	})

	t.Run("CreateUser", func(t *testing.T) {
		user := map[string]interface{}{
			"username": "newuser",
			"name":     "New User",
		}

		result, err := client.CreateUser(user)
		require.NoError(t, err)
		assert.Equal(t, "new-user-id", result["id"])
		assert.Equal(t, "newuser", result["username"])
	})

	t.Run("SetUserPassword", func(t *testing.T) {
		err := client.SetUserPassword("user1", "newpassword")
		require.NoError(t, err)
	})

	t.Run("GetDomains", func(t *testing.T) {
		domains, err := client.GetDomains()
		require.NoError(t, err)
		assert.Len(t, domains, 1)
		assert.Equal(t, "example.com", domains[0]["domain"])
	})

	t.Run("CreateDomain", func(t *testing.T) {
		domain := map[string]interface{}{
			"domain": "newdomain.com",
		}

		result, err := client.CreateDomain(domain)
		require.NoError(t, err)
		assert.Equal(t, "new-domain-id", result["id"])
		assert.Equal(t, "newdomain.com", result["domain"])
	})
}

func TestLogtoClient_ErrorHandling(t *testing.T) {
	// Server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oidc/token" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = fmt.Fprint(w, "Invalid credentials")
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "Internal server error")
	}))
	defer server.Close()

	client := NewLogtoClient(server.URL, "invalid-id", "invalid-secret")
	client.BaseURL = server.URL

	t.Run("TestConnection fails", func(t *testing.T) {
		err := client.TestConnection()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to authenticate")
	})
}
