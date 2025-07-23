package logto

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogtoManagementClient_GetUserByID(t *testing.T) {
	setupServicesTestEnvironment()

	tests := []struct {
		name         string
		userID       string
		setupServer  func() *httptest.Server
		expectError  bool
		expectedUser *models.LogtoUser
	}{
		{
			name:   "successful user fetch",
			userID: "user-123",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/oidc/token":
						response := models.LogtoManagementTokenResponse{
							AccessToken: "test-token",
							TokenType:   "Bearer",
							ExpiresIn:   3600,
							Scope:       "all",
						}
						_ = json.NewEncoder(w).Encode(response)
					case "/api/users/user-123":
						user := models.LogtoUser{
							ID:           "user-123",
							Username:     "testuser",
							PrimaryEmail: "test@example.com",
							Name:         "Test User",
							CustomData:   map[string]interface{}{"department": "IT"},
							IsSuspended:  false,
							HasPassword:  true,
							CreatedAt:    1640995200,
							UpdatedAt:    1640995200,
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(user)
					}
				}))
			},
			expectError: false,
			expectedUser: &models.LogtoUser{
				ID:           "user-123",
				Username:     "testuser",
				PrimaryEmail: "test@example.com",
				Name:         "Test User",
				CustomData:   map[string]interface{}{"department": "IT"},
				IsSuspended:  false,
				HasPassword:  true,
				CreatedAt:    1640995200,
				UpdatedAt:    1640995200,
			},
		},
		{
			name:   "user not found",
			userID: "nonexistent-user",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/oidc/token":
						response := models.LogtoManagementTokenResponse{
							AccessToken: "test-token",
							TokenType:   "Bearer",
							ExpiresIn:   3600,
							Scope:       "all",
						}
						_ = json.NewEncoder(w).Encode(response)
					case "/api/users/nonexistent-user":
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(`{"error": "user not found"}`))
					}
				}))
			},
			expectError:  true,
			expectedUser: nil,
		},
		{
			name:   "internal server error",
			userID: "error-user",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/oidc/token":
						response := models.LogtoManagementTokenResponse{
							AccessToken: "test-token",
							TokenType:   "Bearer",
							ExpiresIn:   3600,
							Scope:       "all",
						}
						_ = json.NewEncoder(w).Encode(response)
					case "/api/users/error-user":
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = w.Write([]byte(`{"error": "internal server error"}`))
					}
				}))
			},
			expectError:  true,
			expectedUser: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			originalIssuer := configuration.Config.LogtoIssuer
			originalBaseURL := configuration.Config.LogtoManagementBaseURL
			configuration.Config.LogtoIssuer = server.URL
			configuration.Config.LogtoManagementBaseURL = server.URL + "/api"
			defer func() {
				configuration.Config.LogtoIssuer = originalIssuer
				configuration.Config.LogtoManagementBaseURL = originalBaseURL
			}()

			client := NewManagementClient()
			user, err := client.GetUserByID(tt.userID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, user)
				if tt.userID == "nonexistent-user" {
					assert.Contains(t, err.Error(), "user not found")
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUser, user)
			}
		})
	}
}

func TestLogtoManagementClient_CreateUser(t *testing.T) {
	setupServicesTestEnvironment()

	tests := []struct {
		name         string
		request      models.CreateUserRequest
		setupServer  func() *httptest.Server
		expectError  bool
		expectedUser *models.LogtoUser
	}{
		{
			name: "successful user creation",
			request: models.CreateUserRequest{
				Username:     "newuser",
				PrimaryEmail: "new@example.com",
				Name:         "New User",
				Password:     "SecurePassword123!",
				CustomData:   map[string]interface{}{"department": "HR"},
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/oidc/token" {
						response := models.LogtoManagementTokenResponse{
							AccessToken: "test-token",
							TokenType:   "Bearer",
							ExpiresIn:   3600,
							Scope:       "all",
						}
						_ = json.NewEncoder(w).Encode(response)
					} else if r.URL.Path == "/api/users" && r.Method == "POST" {
						// Verify request body
						var requestBody models.CreateUserRequest
						err := json.NewDecoder(r.Body).Decode(&requestBody)
						require.NoError(t, err)

						assert.Equal(t, "newuser", requestBody.Username)
						assert.Equal(t, "new@example.com", requestBody.PrimaryEmail)
						assert.Equal(t, "New User", requestBody.Name)
						assert.Equal(t, "SecurePassword123!", requestBody.Password)
						assert.NotNil(t, requestBody.CustomData)

						// Return created user
						user := models.LogtoUser{
							ID:           "user-new-123",
							Username:     requestBody.Username,
							PrimaryEmail: requestBody.PrimaryEmail,
							Name:         requestBody.Name,
							CustomData:   requestBody.CustomData,
							HasPassword:  true,
							CreatedAt:    1640995200,
							UpdatedAt:    1640995200,
						}

						w.WriteHeader(http.StatusCreated)
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(user)
					}
				}))
			},
			expectError: false,
			expectedUser: &models.LogtoUser{
				ID:           "user-new-123",
				Username:     "newuser",
				PrimaryEmail: "new@example.com",
				Name:         "New User",
				CustomData:   map[string]interface{}{"department": "HR"},
				HasPassword:  true,
				CreatedAt:    1640995200,
				UpdatedAt:    1640995200,
			},
		},
		{
			name: "user creation with conflict (username exists)",
			request: models.CreateUserRequest{
				Username:     "existinguser",
				PrimaryEmail: "existing@example.com",
				Name:         "Existing User",
				Password:     "password123",
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/oidc/token" {
						response := models.LogtoManagementTokenResponse{
							AccessToken: "test-token",
							TokenType:   "Bearer",
							ExpiresIn:   3600,
							Scope:       "all",
						}
						_ = json.NewEncoder(w).Encode(response)
					} else if r.URL.Path == "/api/users" && r.Method == "POST" {
						w.WriteHeader(http.StatusConflict)
						_, _ = w.Write([]byte(`{"error": "username already exists"}`))
					}
				}))
			},
			expectError:  true,
			expectedUser: nil,
		},
		{
			name: "user creation with minimal data",
			request: models.CreateUserRequest{
				Username: "minimaluser",
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/oidc/token" {
						response := models.LogtoManagementTokenResponse{
							AccessToken: "test-token",
							TokenType:   "Bearer",
							ExpiresIn:   3600,
							Scope:       "all",
						}
						_ = json.NewEncoder(w).Encode(response)
					} else if r.URL.Path == "/api/users" && r.Method == "POST" {
						var requestBody models.CreateUserRequest
						_ = json.NewDecoder(r.Body).Decode(&requestBody)

						user := models.LogtoUser{
							ID:          "user-minimal-456",
							Username:    requestBody.Username,
							HasPassword: false,
							CreatedAt:   1640995200,
							UpdatedAt:   1640995200,
						}

						w.WriteHeader(http.StatusCreated)
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(user)
					}
				}))
			},
			expectError: false,
			expectedUser: &models.LogtoUser{
				ID:          "user-minimal-456",
				Username:    "minimaluser",
				HasPassword: false,
				CreatedAt:   1640995200,
				UpdatedAt:   1640995200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			originalIssuer := configuration.Config.LogtoIssuer
			originalBaseURL := configuration.Config.LogtoManagementBaseURL
			configuration.Config.LogtoIssuer = server.URL
			configuration.Config.LogtoManagementBaseURL = server.URL + "/api"
			defer func() {
				configuration.Config.LogtoIssuer = originalIssuer
				configuration.Config.LogtoManagementBaseURL = originalBaseURL
			}()

			client := NewManagementClient()
			user, err := client.CreateUser(tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUser, user)
			}
		})
	}
}

func TestLogtoManagementClient_UpdateUser(t *testing.T) {
	setupServicesTestEnvironment()

	tests := []struct {
		name         string
		userID       string
		request      models.UpdateUserRequest
		setupServer  func() *httptest.Server
		expectError  bool
		expectedUser *models.LogtoUser
	}{
		{
			name:   "successful user update",
			userID: "user-update-123",
			request: func() models.UpdateUserRequest {
				newEmail := "updated@example.com"
				newName := "Updated User"
				return models.UpdateUserRequest{
					PrimaryEmail: &newEmail,
					Name:         &newName,
					CustomData:   map[string]interface{}{"updated": true},
				}
			}(),
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/oidc/token" {
						response := models.LogtoManagementTokenResponse{
							AccessToken: "test-token",
							TokenType:   "Bearer",
							ExpiresIn:   3600,
							Scope:       "all",
						}
						_ = json.NewEncoder(w).Encode(response)
					} else if r.URL.Path == "/api/users/user-update-123" && r.Method == "PATCH" {
						// Verify request body
						var requestBody models.UpdateUserRequest
						err := json.NewDecoder(r.Body).Decode(&requestBody)
						require.NoError(t, err)

						assert.Equal(t, "updated@example.com", *requestBody.PrimaryEmail)
						assert.Equal(t, "Updated User", *requestBody.Name)
						assert.NotNil(t, requestBody.CustomData)

						// Return updated user
						user := models.LogtoUser{
							ID:           "user-update-123",
							Username:     "originaluser",
							PrimaryEmail: *requestBody.PrimaryEmail,
							Name:         *requestBody.Name,
							CustomData:   requestBody.CustomData,
							UpdatedAt:    1640995300,
						}

						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(user)
					}
				}))
			},
			expectError: false,
			expectedUser: &models.LogtoUser{
				ID:           "user-update-123",
				Username:     "originaluser",
				PrimaryEmail: "updated@example.com",
				Name:         "Updated User",
				CustomData:   map[string]interface{}{"updated": true},
				UpdatedAt:    1640995300,
			},
		},
		{
			name:   "user not found for update",
			userID: "nonexistent-user",
			request: func() models.UpdateUserRequest {
				newName := "New Name"
				return models.UpdateUserRequest{
					Name: &newName,
				}
			}(),
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/oidc/token" {
						response := models.LogtoManagementTokenResponse{
							AccessToken: "test-token",
							TokenType:   "Bearer",
							ExpiresIn:   3600,
							Scope:       "all",
						}
						_ = json.NewEncoder(w).Encode(response)
					} else if r.URL.Path == "/api/users/nonexistent-user" && r.Method == "PATCH" {
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(`{"error": "user not found"}`))
					}
				}))
			},
			expectError:  true,
			expectedUser: nil,
		},
		{
			name:   "empty update request",
			userID: "user-empty-update",
			request: models.UpdateUserRequest{
				CustomData: map[string]interface{}{"empty": "update"},
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/oidc/token" {
						response := models.LogtoManagementTokenResponse{
							AccessToken: "test-token",
							TokenType:   "Bearer",
							ExpiresIn:   3600,
							Scope:       "all",
						}
						_ = json.NewEncoder(w).Encode(response)
					} else if r.URL.Path == "/api/users/user-empty-update" && r.Method == "PATCH" {
						var requestBody models.UpdateUserRequest
						_ = json.NewDecoder(r.Body).Decode(&requestBody)

						// Verify nil fields are not sent
						assert.Nil(t, requestBody.Username)
						assert.Nil(t, requestBody.PrimaryEmail)
						assert.Nil(t, requestBody.Name)
						assert.NotNil(t, requestBody.CustomData)

						user := models.LogtoUser{
							ID:         "user-empty-update",
							Username:   "unchanged",
							CustomData: requestBody.CustomData,
						}

						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(user)
					}
				}))
			},
			expectError: false,
			expectedUser: &models.LogtoUser{
				ID:         "user-empty-update",
				Username:   "unchanged",
				CustomData: map[string]interface{}{"empty": "update"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			originalIssuer := configuration.Config.LogtoIssuer
			originalBaseURL := configuration.Config.LogtoManagementBaseURL
			configuration.Config.LogtoIssuer = server.URL
			configuration.Config.LogtoManagementBaseURL = server.URL + "/api"
			defer func() {
				configuration.Config.LogtoIssuer = originalIssuer
				configuration.Config.LogtoManagementBaseURL = originalBaseURL
			}()

			client := NewManagementClient()
			user, err := client.UpdateUser(tt.userID, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUser, user)
			}
		})
	}
}

func TestLogtoManagementClient_DeleteUser(t *testing.T) {
	setupServicesTestEnvironment()

	tests := []struct {
		name        string
		userID      string
		setupServer func() *httptest.Server
		expectError bool
	}{
		{
			name:   "successful user deletion",
			userID: "user-delete-123",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/oidc/token" {
						response := models.LogtoManagementTokenResponse{
							AccessToken: "test-token",
							TokenType:   "Bearer",
							ExpiresIn:   3600,
							Scope:       "all",
						}
						_ = json.NewEncoder(w).Encode(response)
					} else if r.URL.Path == "/api/users/user-delete-123" && r.Method == "DELETE" {
						w.WriteHeader(http.StatusNoContent)
					}
				}))
			},
			expectError: false,
		},
		{
			name:   "user not found for deletion",
			userID: "nonexistent-user",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/oidc/token" {
						response := models.LogtoManagementTokenResponse{
							AccessToken: "test-token",
							TokenType:   "Bearer",
							ExpiresIn:   3600,
							Scope:       "all",
						}
						_ = json.NewEncoder(w).Encode(response)
					} else if r.URL.Path == "/api/users/nonexistent-user" && r.Method == "DELETE" {
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(`{"error": "user not found"}`))
					}
				}))
			},
			expectError: true,
		},
		{
			name:   "forbidden deletion",
			userID: "protected-user",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/oidc/token" {
						response := models.LogtoManagementTokenResponse{
							AccessToken: "test-token",
							TokenType:   "Bearer",
							ExpiresIn:   3600,
							Scope:       "all",
						}
						_ = json.NewEncoder(w).Encode(response)
					} else if r.URL.Path == "/api/users/protected-user" && r.Method == "DELETE" {
						w.WriteHeader(http.StatusForbidden)
						_, _ = w.Write([]byte(`{"error": "cannot delete protected user"}`))
					}
				}))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			originalIssuer := configuration.Config.LogtoIssuer
			originalBaseURL := configuration.Config.LogtoManagementBaseURL
			configuration.Config.LogtoIssuer = server.URL
			configuration.Config.LogtoManagementBaseURL = server.URL + "/api"
			defer func() {
				configuration.Config.LogtoIssuer = originalIssuer
				configuration.Config.LogtoManagementBaseURL = originalBaseURL
			}()

			client := NewManagementClient()
			err := client.DeleteUser(tt.userID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetUserProfileFromLogto(t *testing.T) {
	setupServicesTestEnvironment()

	tests := []struct {
		name            string
		userID          string
		setupServer     func() *httptest.Server
		expectError     bool
		expectedProfile *models.LogtoUser
	}{
		{
			name:   "successful user profile fetch",
			userID: "profile-user-123",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/oidc/token":
						response := models.LogtoManagementTokenResponse{
							AccessToken: "test-token",
							TokenType:   "Bearer",
							ExpiresIn:   3600,
							Scope:       "all",
						}
						_ = json.NewEncoder(w).Encode(response)
					case "/api/users/profile-user-123":
						profile := models.LogtoUser{
							ID:           "profile-user-123",
							Username:     "profileuser",
							PrimaryEmail: "profile@example.com",
							Name:         "Profile User",
							Avatar:       "https://example.com/avatar.png",
							CustomData:   map[string]interface{}{"role": "manager"},
							CreatedAt:    1640995200,
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(profile)
					}
				}))
			},
			expectError: false,
			expectedProfile: &models.LogtoUser{
				ID:           "profile-user-123",
				Username:     "profileuser",
				PrimaryEmail: "profile@example.com",
				Name:         "Profile User",
				Avatar:       "https://example.com/avatar.png",
				CustomData:   map[string]interface{}{"role": "manager"},
				CreatedAt:    1640995200,
			},
		},
		{
			name:   "user profile not found",
			userID: "nonexistent-profile-user",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/oidc/token":
						response := models.LogtoManagementTokenResponse{
							AccessToken: "test-token",
							TokenType:   "Bearer",
							ExpiresIn:   3600,
							Scope:       "all",
						}
						_ = json.NewEncoder(w).Encode(response)
					case "/api/users/nonexistent-profile-user":
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(`{"error": "user not found"}`))
					}
				}))
			},
			expectError:     true,
			expectedProfile: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			originalIssuer := configuration.Config.LogtoIssuer
			originalBaseURL := configuration.Config.LogtoManagementBaseURL
			configuration.Config.LogtoIssuer = server.URL
			configuration.Config.LogtoManagementBaseURL = server.URL + "/api"
			defer func() {
				configuration.Config.LogtoIssuer = originalIssuer
				configuration.Config.LogtoManagementBaseURL = originalBaseURL
			}()

			profile, err := GetUserProfileFromLogto(tt.userID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, profile)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedProfile, profile)
			}
		})
	}
}
