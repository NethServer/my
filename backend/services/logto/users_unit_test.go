/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package logto

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
)

// TestLogtoManagementClient_GetUserByID_Unit tests fetching a user by ID (unit test)
func TestLogtoManagementClient_GetUserByID_Unit(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
		expectedUser   *models.LogtoUser
	}{
		{
			name:   "successful user retrieval",
			userID: "user-123",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/users/user-123", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "user-123",
					"username": "testuser",
					"primaryEmail": "test@example.com",
					"name": "Test User"
				}`))
			},
			expectedError: "",
			expectedUser: &models.LogtoUser{
				ID:           "user-123",
				Username:     "testuser",
				PrimaryEmail: "test@example.com",
				Name:         "Test User",
			},
		},
		{
			name:   "user not found",
			userID: "nonexistent",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/users/nonexistent", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message": "User not found"}`))
			},
			expectedError: "user not found",
			expectedUser:  nil,
		},
		{
			name:   "server error",
			userID: "error-user",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"message": "Internal server error"}`))
			},
			expectedError: "failed to fetch user, status 500",
			expectedUser:  nil,
		},
		{
			name:   "invalid json response",
			userID: "invalid-json-user",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{invalid json`))
			},
			expectedError: "failed to decode user",
			expectedUser:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := &LogtoManagementClient{
				baseURL: server.URL,
			}

			user, err := client.GetUserByID(tt.userID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.expectedUser.ID, user.ID)
				assert.Equal(t, tt.expectedUser.Username, user.Username)
				assert.Equal(t, tt.expectedUser.PrimaryEmail, user.PrimaryEmail)
				assert.Equal(t, tt.expectedUser.Name, user.Name)
			}
		})
	}
}

// TestLogtoManagementClient_CreateUser_Unit tests user creation (unit test)
func TestLogtoManagementClient_CreateUser_Unit(t *testing.T) {
	tests := []struct {
		name           string
		request        models.CreateUserRequest
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
		expectedUser   *models.LogtoUser
	}{
		{
			name: "successful user creation",
			request: models.CreateUserRequest{
				Username:     "newuser",
				Name:         "New User",
				PrimaryEmail: "newuser@example.com",
				Password:     "MySecurePassword123!",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/users", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Verify request body
				body, _ := io.ReadAll(r.Body)
				assert.Contains(t, string(body), "newuser")
				assert.Contains(t, string(body), "newuser@example.com")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{
					"id": "user-456",
					"username": "newuser",
					"primaryEmail": "newuser@example.com",
					"name": "New User"
				}`))
			},
			expectedError: "",
			expectedUser: &models.LogtoUser{
				ID:           "user-456",
				Username:     "newuser",
				PrimaryEmail: "newuser@example.com",
				Name:         "New User",
			},
		},
		{
			name: "validation error from logto",
			request: models.CreateUserRequest{
				Username:     "invalid@user",
				Name:         "Invalid User",
				PrimaryEmail: "invalid-email",
				Password:     "weak",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{
					"message": "Validation failed",
					"details": {
						"username": "Invalid username format",
						"primaryEmail": "Invalid email format",
						"password": "Password too weak"
					}
				}`))
			},
			expectedError: "failed to create user, status 400",
			expectedUser:  nil,
		},
		{
			name: "duplicate user error",
			request: models.CreateUserRequest{
				Username:     "existing",
				Name:         "Existing User",
				PrimaryEmail: "existing@example.com",
				Password:     "MySecurePassword123!",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"message": "User already exists"}`))
			},
			expectedError: "failed to create user, status 409",
			expectedUser:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := &LogtoManagementClient{
				baseURL: server.URL,
			}

			user, err := client.CreateUser(tt.request)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.expectedUser.ID, user.ID)
				assert.Equal(t, tt.expectedUser.Username, user.Username)
				assert.Equal(t, tt.expectedUser.PrimaryEmail, user.PrimaryEmail)
			}
		})
	}
}

// TestLogtoManagementClient_UpdateUser_Unit tests user updates (unit test)
func TestLogtoManagementClient_UpdateUser_Unit(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		request        models.UpdateUserRequest
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
		expectedUser   *models.LogtoUser
	}{
		{
			name:   "successful user update",
			userID: "user-123",
			request: models.UpdateUserRequest{
				Name:         stringPtr("Updated Name"),
				PrimaryEmail: stringPtr("updated@example.com"),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PATCH", r.Method)
				assert.Equal(t, "/users/user-123", r.URL.Path)

				// Verify request body
				body, _ := io.ReadAll(r.Body)
				assert.Contains(t, string(body), "Updated Name")
				assert.Contains(t, string(body), "updated@example.com")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "user-123",
					"username": "testuser",
					"primaryEmail": "updated@example.com",
					"name": "Updated Name"
				}`))
			},
			expectedError: "",
			expectedUser: &models.LogtoUser{
				ID:           "user-123",
				Username:     "testuser",
				PrimaryEmail: "updated@example.com",
				Name:         "Updated Name",
			},
		},
		{
			name:   "user not found for update",
			userID: "nonexistent",
			request: models.UpdateUserRequest{
				Name: stringPtr("New Name"),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message": "User not found"}`))
			},
			expectedError: "failed to update user, status 404",
			expectedUser:  nil,
		},
		{
			name:   "validation error on update",
			userID: "user-456",
			request: models.UpdateUserRequest{
				PrimaryEmail: stringPtr("invalid-email"),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{
					"message": "Validation failed",
					"details": {
						"primaryEmail": "Invalid email format"
					}
				}`))
			},
			expectedError: "failed to update user, status 400",
			expectedUser:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := &LogtoManagementClient{
				baseURL: server.URL,
			}

			user, err := client.UpdateUser(tt.userID, tt.request)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.expectedUser.ID, user.ID)
				assert.Equal(t, tt.expectedUser.Name, user.Name)
				assert.Equal(t, tt.expectedUser.PrimaryEmail, user.PrimaryEmail)
			}
		})
	}
}

// TestLogtoManagementClient_DeleteUser_Unit tests user deletion (unit test)
func TestLogtoManagementClient_DeleteUser_Unit(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
	}{
		{
			name:   "successful user deletion",
			userID: "user-123",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				assert.Equal(t, "/users/user-123", r.URL.Path)
				w.WriteHeader(http.StatusNoContent)
			},
			expectedError: "",
		},
		{
			name:   "user not found for deletion",
			userID: "nonexistent",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message": "User not found"}`))
			},
			expectedError: "failed to delete user, status 404",
		},
		{
			name:   "protected user deletion",
			userID: "protected-user",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"message": "Cannot delete protected user"}`))
			},
			expectedError: "failed to delete user, status 403",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := &LogtoManagementClient{
				baseURL: server.URL,
			}

			err := client.DeleteUser(tt.userID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLogtoManagementClient_ResetUserPassword tests password reset functionality
func TestLogtoManagementClient_ResetUserPassword(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		password       string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
	}{
		{
			name:     "successful password reset",
			userID:   "user-123",
			password: "MyNewSecurePassword123!",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PATCH", r.Method)
				assert.Equal(t, "/users/user-123/password", r.URL.Path)

				// Verify request body contains password
				body, _ := io.ReadAll(r.Body)
				assert.Contains(t, string(body), "MyNewSecurePassword123!")

				w.WriteHeader(http.StatusOK)
			},
			expectedError: "",
		},
		{
			name:     "user not found for password reset",
			userID:   "nonexistent",
			password: "MyNewSecurePassword123!",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message": "User not found"}`))
			},
			expectedError: "failed to update user password, status 404",
		},
		{
			name:     "weak password validation",
			userID:   "user-456",
			password: "weak",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{
					"message": "Password validation failed",
					"details": {
						"password": "Password does not meet requirements"
					}
				}`))
			},
			expectedError: "failed to update user password, status 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := &LogtoManagementClient{
				baseURL: server.URL,
			}

			err := client.ResetUserPassword(tt.userID, tt.password)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLogtoManagementClient_SuspendUser tests user suspension
func TestLogtoManagementClient_SuspendUser(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
	}{
		{
			name:   "successful user suspension",
			userID: "user-123",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PATCH", r.Method)
				assert.Equal(t, "/users/user-123/is-suspended", r.URL.Path)

				// Verify request body contains isSuspended: true
				body, _ := io.ReadAll(r.Body)
				assert.Contains(t, string(body), "isSuspended")
				assert.Contains(t, string(body), "true")

				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "user-123",
					"isSuspended": true
				}`))
			},
			expectedError: "",
		},
		{
			name:   "user not found for suspension",
			userID: "nonexistent",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message": "User not found"}`))
			},
			expectedError: "failed to suspend user, status 404",
		},
		{
			name:   "user already suspended",
			userID: "already-suspended",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"message": "User is already suspended"}`))
			},
			expectedError: "failed to suspend user, status 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := &LogtoManagementClient{
				baseURL: server.URL,
			}

			err := client.SuspendUser(tt.userID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLogtoManagementClient_ReactivateUser tests user reactivation
func TestLogtoManagementClient_ReactivateUser(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
	}{
		{
			name:   "successful user reactivation",
			userID: "suspended-user",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PATCH", r.Method)
				assert.Equal(t, "/users/suspended-user/is-suspended", r.URL.Path)

				// Verify request body contains isSuspended: false
				body, _ := io.ReadAll(r.Body)
				assert.Contains(t, string(body), "isSuspended")
				assert.Contains(t, string(body), "false")

				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": "suspended-user",
					"isSuspended": false
				}`))
			},
			expectedError: "",
		},
		{
			name:   "user not found for reactivation",
			userID: "nonexistent",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message": "User not found"}`))
			},
			expectedError: "failed to reactivate user, status 404",
		},
		{
			name:   "user not suspended",
			userID: "active-user",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"message": "User is not suspended"}`))
			},
			expectedError: "failed to reactivate user, status 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := &LogtoManagementClient{
				baseURL: server.URL,
			}

			err := client.ReactivateUser(tt.userID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLogtoManagementClient_AssignUserToOrganization tests organization assignment
func TestLogtoManagementClient_AssignUserToOrganization(t *testing.T) {
	tests := []struct {
		name           string
		orgID          string
		userID         string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
	}{
		{
			name:   "successful organization assignment",
			orgID:  "org-123",
			userID: "user-456",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/organizations/org-123/users", r.URL.Path)

				// Verify request body contains userIds
				body, _ := io.ReadAll(r.Body)
				assert.Contains(t, string(body), "user-456")
				assert.Contains(t, string(body), "userIds")

				w.WriteHeader(http.StatusCreated)
			},
			expectedError: "",
		},
		{
			name:   "organization not found",
			orgID:  "nonexistent-org",
			userID: "user-456",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message": "Organization not found"}`))
			},
			expectedError: "failed to assign user to organization, status 404",
		},
		{
			name:   "user already in organization",
			orgID:  "org-123",
			userID: "existing-user",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"message": "User already in organization"}`))
			},
			expectedError: "failed to assign user to organization, status 409",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := &LogtoManagementClient{
				baseURL: server.URL,
			}

			err := client.AssignUserToOrganization(tt.orgID, tt.userID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLogtoManagementClient_RemoveUserFromOrganization tests organization removal
func TestLogtoManagementClient_RemoveUserFromOrganization(t *testing.T) {
	tests := []struct {
		name           string
		orgID          string
		userID         string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
	}{
		{
			name:   "successful organization removal",
			orgID:  "org-123",
			userID: "user-456",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				assert.Equal(t, "/organizations/org-123/users/user-456", r.URL.Path)
				w.WriteHeader(http.StatusNoContent)
			},
			expectedError: "",
		},
		{
			name:   "user not in organization",
			orgID:  "org-123",
			userID: "nonmember-user",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message": "User not in organization"}`))
			},
			expectedError: "failed to remove user from organization, status 404",
		},
		{
			name:   "organization not found",
			orgID:  "nonexistent-org",
			userID: "user-456",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message": "Organization not found"}`))
			},
			expectedError: "failed to remove user from organization, status 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := &LogtoManagementClient{
				baseURL: server.URL,
			}

			err := client.RemoveUserFromOrganization(tt.orgID, tt.userID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLogtoManagementClient_makeRequest tests the makeRequest helper method indirectly
func TestLogtoManagementClient_makeRequest_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
	}{
		{
			name: "network timeout simulation",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Simulate a slow server response that might timeout
				w.WriteHeader(http.StatusRequestTimeout)
				_, _ = w.Write([]byte(`{"message": "Request timeout"}`))
			},
			expectedError: "failed to fetch user, status 408",
		},
		{
			name: "server maintenance mode",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"message": "Service temporarily unavailable"}`))
			},
			expectedError: "failed to fetch user, status 503",
		},
		{
			name: "rate limiting",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"message": "Too many requests"}`))
			},
			expectedError: "failed to fetch user, status 429",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := &LogtoManagementClient{
				baseURL: server.URL,
			}

			// Test error handling through GetUserByID as a representative method
			_, err := client.GetUserByID("test-user")

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// TestLogtoManagementClient_RequestHeaders tests that proper headers are sent
func TestLogtoManagementClient_RequestHeaders(t *testing.T) {
	headerChecked := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Content-Type header for POST/PATCH requests
		if r.Method == "POST" || r.Method == "PATCH" {
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		}

		// Verify that request has proper structure
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/users") {
			body, _ := io.ReadAll(r.Body)
			assert.True(t, len(body) > 0, "Request body should not be empty for POST")
		}

		headerChecked = true
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": "test-user"}`))
	}))
	defer server.Close()

	client := &LogtoManagementClient{
		baseURL: server.URL,
	}

	// Test with a POST request that should include Content-Type header
	_, err := client.CreateUser(models.CreateUserRequest{
		Username:     "test",
		Name:         "Test User",
		PrimaryEmail: "test@example.com",
		Password:     "MySecurePassword123!",
	})

	assert.NoError(t, err)
	assert.True(t, headerChecked, "Headers should have been checked")
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
