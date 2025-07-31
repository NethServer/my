package logto

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
)

func setupServicesTestEnvironment() {
	if !isServicesTestEnvironmentSetup {
		gin.SetMode(gin.TestMode)
		_ = logger.Init(&logger.Config{Level: logger.InfoLevel, Format: logger.JSONFormat, Output: logger.StdoutOutput, AppName: "[SERVICES-TEST]"})

		// Set test environment variables for configuration
		_ = os.Setenv("TENANT_ID", "test-tenant")
		_ = os.Setenv("TENANT_DOMAIN", "test-domain.com")
		_ = os.Setenv("JWT_SECRET", "test-secret-key")
		_ = os.Setenv("BACKEND_APP_ID", "test-client-id")
		_ = os.Setenv("BACKEND_APP_SECRET", "test-client-secret")
		_ = os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test_db")
		_ = os.Setenv("REDIS_URL", "redis://localhost:6379")

		configuration.Init()
		isServicesTestEnvironmentSetup = true
	}
}

var isServicesTestEnvironmentSetup bool

// Test Models Structures
func TestLogtoModels(t *testing.T) {
	t.Run("models.LogtoUserInfo", func(t *testing.T) {
		userInfo := models.LogtoUserInfo{
			Sub:              "user-123",
			Username:         "testuser",
			Email:            "test@example.com",
			Name:             "Test User",
			Roles:            []string{"Admin", "Support"},
			OrganizationId:   "org-123",
			OrganizationName: "Test Organization",
		}

		// Test JSON serialization
		jsonData, err := json.Marshal(userInfo)
		assert.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		// Test JSON deserialization
		var unmarshaled models.LogtoUserInfo
		err = json.Unmarshal(jsonData, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, userInfo, unmarshaled)
	})

	t.Run("models.LogtoRole", func(t *testing.T) {
		role := models.LogtoRole{
			ID:          "role-123",
			Name:        "Admin",
			Description: "Administrator role",
			Type:        "MachineToMachine",
		}

		jsonData, err := json.Marshal(role)
		assert.NoError(t, err)

		var unmarshaled models.LogtoRole
		err = json.Unmarshal(jsonData, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, role, unmarshaled)
	})

	t.Run("models.LogtoOrganization", func(t *testing.T) {
		org := models.LogtoOrganization{
			ID:          "org-123",
			Name:        "Test Organization",
			Description: "Test Description",
			CustomData:  map[string]interface{}{"type": "customer"},
			Branding: &models.LogtoOrganizationBranding{
				LogoUrl:     "https://example.com/logo.png",
				DarkLogoUrl: "https://example.com/dark-logo.png",
				Favicon:     "https://example.com/favicon.ico",
				DarkFavicon: "https://example.com/dark-favicon.ico",
			},
		}

		jsonData, err := json.Marshal(org)
		assert.NoError(t, err)

		var unmarshaled models.LogtoOrganization
		err = json.Unmarshal(jsonData, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, org, unmarshaled)
	})

	t.Run("models.LogtoUser", func(t *testing.T) {
		now := time.Now().Unix()
		lastSignIn := now - 3600

		user := models.LogtoUser{
			ID:            "user-123",
			Username:      "testuser",
			PrimaryEmail:  "test@example.com",
			PrimaryPhone:  "+1234567890",
			Name:          "Test User",
			Avatar:        "https://example.com/avatar.png",
			CustomData:    map[string]interface{}{"department": "IT"},
			Identities:    map[string]interface{}{"google": "google-id-123"},
			LastSignInAt:  &lastSignIn,
			IsSuspended:   false,
			HasPassword:   true,
			ApplicationId: "app-123",
			CreatedAt:     now - 86400,
			UpdatedAt:     now,
		}

		jsonData, err := json.Marshal(user)
		assert.NoError(t, err)

		var unmarshaled models.LogtoUser
		err = json.Unmarshal(jsonData, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, user, unmarshaled)
	})
}

// Test LogtoManagementClient
func TestNewManagementClient(t *testing.T) {
	setupServicesTestEnvironment()

	client := NewManagementClient()

	assert.NotNil(t, client)
	assert.Equal(t, configuration.Config.LogtoManagementBaseURL, client.baseURL)
	assert.Equal(t, configuration.Config.LogtoManagementClientID, client.clientID)
	assert.Equal(t, configuration.Config.LogtoManagementClientSecret, client.clientSecret)

	// Check that global token cache is empty initially
	globalTokenCache.mu.RLock()
	assert.Empty(t, globalTokenCache.accessToken)
	assert.True(t, globalTokenCache.tokenExpiry.IsZero())
	globalTokenCache.mu.RUnlock()
}

func TestLogtoManagementClient_getAccessToken(t *testing.T) {
	setupServicesTestEnvironment()

	tests := []struct {
		name           string
		setupServer    func() *httptest.Server
		expectError    bool
		expectedToken  string
		validateClient func(*testing.T, *LogtoManagementClient)
	}{
		{
			name: "successful token request",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/oidc/token" && r.Method == "POST" {
						// Verify request format
						body, _ := io.ReadAll(r.Body)
						assert.Contains(t, string(body), "grant_type=client_credentials")
						assert.Contains(t, string(body), "client_id=test-client-id")
						assert.Contains(t, string(body), "client_secret=test-client-secret")

						// Return valid token response
						response := models.LogtoManagementTokenResponse{
							AccessToken: "test-access-token",
							TokenType:   "Bearer",
							ExpiresIn:   3600,
							Scope:       "all",
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(response)
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				}))
			},
			expectError:   false,
			expectedToken: "test-access-token",
			validateClient: func(t *testing.T, c *LogtoManagementClient) {
				globalTokenCache.mu.RLock()
				assert.Equal(t, "test-access-token", globalTokenCache.accessToken)
				assert.True(t, globalTokenCache.tokenExpiry.After(time.Now()))
				globalTokenCache.mu.RUnlock()
			},
		},
		{
			name: "token request with HTTP error",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error": "invalid_client"}`))
				}))
			},
			expectError:   true,
			expectedToken: "",
			validateClient: func(t *testing.T, c *LogtoManagementClient) {
				globalTokenCache.mu.RLock()
				assert.Empty(t, globalTokenCache.accessToken)
				globalTokenCache.mu.RUnlock()
			},
		},
		{
			name: "token request with invalid JSON response",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`invalid json`))
				}))
			},
			expectError:   true,
			expectedToken: "",
			validateClient: func(t *testing.T, c *LogtoManagementClient) {
				globalTokenCache.mu.RLock()
				assert.Empty(t, globalTokenCache.accessToken)
				globalTokenCache.mu.RUnlock()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			// Update configuration to use test server
			originalIssuer := configuration.Config.LogtoIssuer
			configuration.Config.LogtoIssuer = server.URL
			defer func() { configuration.Config.LogtoIssuer = originalIssuer }()

			// Reset cache before test
			invalidateToken()

			client := NewManagementClient()
			token, err := client.getAccessToken()

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}

			tt.validateClient(t, client)
		})
	}
}

func TestLogtoManagementClient_TokenCaching(t *testing.T) {
	setupServicesTestEnvironment()

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		response := models.LogtoManagementTokenResponse{
			AccessToken: "cached-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
			Scope:       "all",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalIssuer := configuration.Config.LogtoIssuer
	configuration.Config.LogtoIssuer = server.URL
	defer func() { configuration.Config.LogtoIssuer = originalIssuer }()

	// Reset cache before test
	invalidateToken()

	client := NewManagementClient()

	// First request should hit the server
	token, err := client.getAccessToken()
	assert.NoError(t, err)
	assert.Equal(t, 1, requestCount)
	assert.Equal(t, "cached-token", token)

	// Second request should use cached token
	token, err = client.getAccessToken()
	assert.NoError(t, err)
	assert.Equal(t, 1, requestCount) // Should not increase
	assert.Equal(t, "cached-token", token)

	// Expire the token manually
	globalTokenCache.mu.Lock()
	globalTokenCache.tokenExpiry = time.Now().Add(-1 * time.Hour)
	globalTokenCache.mu.Unlock()

	// Third request should hit the server again
	token, err = client.getAccessToken()
	assert.NoError(t, err)
	assert.Equal(t, 2, requestCount) // Should increase
	assert.Equal(t, "cached-token", token)
}

func TestLogtoManagementClient_makeRequest(t *testing.T) {
	setupServicesTestEnvironment()

	// Reset cache before test to avoid interference from previous tests
	invalidateToken()

	tests := []struct {
		name           string
		method         string
		endpoint       string
		body           io.Reader
		setupServer    func() *httptest.Server
		expectError    bool
		expectedStatus int
	}{
		{
			name:     "successful GET request",
			method:   "GET",
			endpoint: "/test",
			body:     nil,
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/oidc/token":
						// Token request
						response := models.LogtoManagementTokenResponse{
							AccessToken: "test-token",
							TokenType:   "Bearer",
							ExpiresIn:   3600,
							Scope:       "all",
						}
						_ = json.NewEncoder(w).Encode(response)
					case "/api/test":
						// API request
						assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
						assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`{"success": true}`))
					}
				}))
			},
			expectError:    false,
			expectedStatus: http.StatusOK,
		},
		{
			name:     "successful POST request with body",
			method:   "POST",
			endpoint: "/users",
			body:     bytes.NewReader([]byte(`{"username": "test"}`)),
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
					case "/api/users":
						body, _ := io.ReadAll(r.Body)
						assert.Equal(t, `{"username": "test"}`, string(body))
						w.WriteHeader(http.StatusCreated)
						_, _ = w.Write([]byte(`{"id": "user-123"}`))
					}
				}))
			},
			expectError:    false,
			expectedStatus: http.StatusCreated,
		},
		{
			name:     "API request returns error status",
			method:   "GET",
			endpoint: "/nonexistent",
			body:     nil,
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
					case "/api/nonexistent":
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(`{"error": "not found"}`))
					}
				}))
			},
			expectError:    false,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			// Update configuration to use test server
			originalIssuer := configuration.Config.LogtoIssuer
			originalBaseURL := configuration.Config.LogtoManagementBaseURL
			configuration.Config.LogtoIssuer = server.URL
			configuration.Config.LogtoManagementBaseURL = server.URL + "/api"
			defer func() {
				configuration.Config.LogtoIssuer = originalIssuer
				configuration.Config.LogtoManagementBaseURL = originalBaseURL
			}()

			// Reset cache for each sub-test to avoid interference
			invalidateToken()

			client := NewManagementClient()
			resp, err := client.makeRequest(tt.method, tt.endpoint, tt.body)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.expectedStatus, resp.StatusCode)
				_ = resp.Body.Close()
			}
		})
	}
}

func TestGetUserInfoFromLogto(t *testing.T) {
	setupServicesTestEnvironment()

	tests := []struct {
		name         string
		accessToken  string
		setupServer  func() *httptest.Server
		expectError  bool
		expectedUser *models.LogtoUserInfo
	}{
		{
			name:        "successful user info request",
			accessToken: "valid-access-token",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/oidc/me" {
						// Verify authorization header
						assert.Equal(t, "Bearer valid-access-token", r.Header.Get("Authorization"))

						userInfo := models.LogtoUserInfo{
							Sub:              "user-123",
							Username:         "testuser",
							Email:            "test@example.com",
							Name:             "Test User",
							Roles:            []string{"Admin"},
							OrganizationId:   "org-123",
							OrganizationName: "Test Organization",
						}

						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(userInfo)
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				}))
			},
			expectError: false,
			expectedUser: &models.LogtoUserInfo{
				Sub:              "user-123",
				Username:         "testuser",
				Email:            "test@example.com",
				Name:             "Test User",
				Roles:            []string{"Admin"},
				OrganizationId:   "org-123",
				OrganizationName: "Test Organization",
			},
		},
		{
			name:        "invalid access token",
			accessToken: "invalid-token",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error": "invalid_token"}`))
				}))
			},
			expectError:  true,
			expectedUser: nil,
		},
		{
			name:        "empty access token",
			accessToken: "",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error": "missing_token"}`))
				}))
			},
			expectError:  true,
			expectedUser: nil,
		},
		{
			name:        "invalid JSON response",
			accessToken: "valid-token",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`invalid json response`))
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

			// Update configuration to use test server
			originalIssuer := configuration.Config.LogtoIssuer
			configuration.Config.LogtoIssuer = server.URL
			defer func() { configuration.Config.LogtoIssuer = originalIssuer }()

			userInfo, err := GetUserInfoFromLogto(tt.accessToken)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, userInfo)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, userInfo)
				assert.Equal(t, tt.expectedUser, userInfo)
			}
		})
	}
}

func TestCreateUserRequest(t *testing.T) {
	// Test models.CreateUserRequest struct
	customData := map[string]interface{}{
		"department": "IT",
		"location":   "Milan",
	}

	request := models.CreateUserRequest{
		Username:     "newuser",
		PrimaryEmail: "new@example.com",
		Name:         "New User",
		CustomData:   customData,
		Password:     "SecurePassword123!",
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(request)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Verify JSON contains expected fields
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	assert.NoError(t, err)
	assert.Equal(t, "newuser", jsonMap["username"])
	assert.Equal(t, "new@example.com", jsonMap["primaryEmail"])
	assert.Equal(t, "New User", jsonMap["name"])
	assert.Equal(t, "SecurePassword123!", jsonMap["password"])
	assert.NotNil(t, jsonMap["customData"])
}

func TestUpdateUserRequest(t *testing.T) {
	// Test models.UpdateUserRequest with pointer fields
	newUsername := "updateduser"
	newEmail := "updated@example.com"
	newName := "Updated User"

	request := models.UpdateUserRequest{
		Username:     &newUsername,
		PrimaryEmail: &newEmail,
		Name:         &newName,
		CustomData:   map[string]interface{}{"updated": true},
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(request)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test with nil fields (should be omitted)
	emptyRequest := models.UpdateUserRequest{
		CustomData: map[string]interface{}{"key": "value"},
	}

	jsonData, err = json.Marshal(emptyRequest)
	assert.NoError(t, err)

	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	assert.NoError(t, err)
	assert.NotContains(t, jsonMap, "username")
	assert.NotContains(t, jsonMap, "primaryEmail")
	assert.Contains(t, jsonMap, "customData")
}

func TestCreateOrganizationRequest(t *testing.T) {
	// Test models.CreateOrganizationRequest
	customData := map[string]interface{}{
		"type":   "enterprise",
		"tier":   "premium",
		"region": "eu-west",
	}

	branding := &models.LogtoOrganizationBranding{
		LogoUrl:     "https://example.com/logo.png",
		DarkLogoUrl: "https://example.com/dark-logo.png",
		Favicon:     "https://example.com/favicon.ico",
		DarkFavicon: "https://example.com/dark-favicon.ico",
	}

	request := models.CreateOrganizationRequest{
		Name:        "New Organization",
		Description: "A new test organization",
		CustomData:  customData,
		Branding:    branding,
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(request)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test deserialization
	var unmarshaled models.CreateOrganizationRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, request.Name, unmarshaled.Name)
	assert.Equal(t, request.Description, unmarshaled.Description)
	assert.Equal(t, request.CustomData, unmarshaled.CustomData)
	assert.NotNil(t, unmarshaled.Branding)
	assert.Equal(t, branding.LogoUrl, unmarshaled.Branding.LogoUrl)
}

// Test that the services properly handle network errors
func TestServicesNetworkErrorHandling(t *testing.T) {
	setupServicesTestEnvironment()

	// Test with non-existent server
	originalIssuer := configuration.Config.LogtoIssuer
	configuration.Config.LogtoIssuer = "http://non-existent-server.invalid"
	defer func() { configuration.Config.LogtoIssuer = originalIssuer }()

	t.Run("GetUserInfoFromLogto with network error", func(t *testing.T) {
		userInfo, err := GetUserInfoFromLogto("any-token")
		assert.Error(t, err)
		assert.Nil(t, userInfo)
		assert.Contains(t, err.Error(), "failed to fetch user info")
	})

	t.Run("LogtoManagementClient getAccessToken with network error", func(t *testing.T) {
		// Reset cache before test
		invalidateToken()

		client := NewManagementClient()
		token, err := client.getAccessToken()
		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "failed to request token")
	})
}
