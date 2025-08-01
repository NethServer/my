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

func TestLogtoManagementClient_GetUserRoles(t *testing.T) {
	setupServicesTestEnvironment()

	tests := []struct {
		name          string
		userID        string
		setupServer   func() *httptest.Server
		expectError   bool
		expectedRoles []models.LogtoRole
	}{
		{
			name:   "successful user roles fetch",
			userID: "user-123",
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
					case "/api/users/user-123/roles":
						// User roles request
						roles := []models.LogtoRole{
							{ID: "role-1", Name: "Admin", Description: "Administrator", Type: "User"},
							{ID: "role-2", Name: "Support", Description: "Support user", Type: "User"},
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(roles)
					}
				}))
			},
			expectError: false,
			expectedRoles: []models.LogtoRole{
				{ID: "role-1", Name: "Admin", Description: "Administrator", Type: "User"},
				{ID: "role-2", Name: "Support", Description: "Support user", Type: "User"},
			},
		},
		{
			name:   "user not found returns error",
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
					case "/api/users/nonexistent-user/roles":
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(`{"error": "user not found"}`))
					}
				}))
			},
			expectError:   true,
			expectedRoles: nil,
		},
		{
			name:   "empty roles list returns empty array",
			userID: "user-no-roles",
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
					case "/api/users/user-no-roles/roles":
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode([]models.LogtoRole{})
					}
				}))
			},
			expectError:   false,
			expectedRoles: []models.LogtoRole{},
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

			client := NewManagementClient()
			roles, err := client.GetUserRoles(tt.userID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, roles)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedRoles, roles)
			}
		})
	}
}

func TestLogtoManagementClient_GetRoleScopes(t *testing.T) {
	setupServicesTestEnvironment()

	tests := []struct {
		name           string
		roleID         string
		setupServer    func() *httptest.Server
		expectError    bool
		expectedScopes []models.LogtoScope
	}{
		{
			name:   "successful role scopes fetch",
			roleID: "role-123",
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
					case "/api/roles/role-123/scopes":
						scopes := []models.LogtoScope{
							{ID: "scope-1", Name: "manage:systems", Description: "Manage systems", ResourceID: "api-resource"},
							{ID: "scope-2", Name: "view:logs", Description: "View logs", ResourceID: "api-resource"},
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(scopes)
					}
				}))
			},
			expectError: false,
			expectedScopes: []models.LogtoScope{
				{ID: "scope-1", Name: "manage:systems", Description: "Manage systems", ResourceID: "api-resource"},
				{ID: "scope-2", Name: "view:logs", Description: "View logs", ResourceID: "api-resource"},
			},
		},
		{
			name:   "role not found returns error",
			roleID: "nonexistent-role",
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
					case "/api/roles/nonexistent-role/scopes":
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(`{"error": "role not found"}`))
					}
				}))
			},
			expectError:    true,
			expectedScopes: nil,
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
			scopes, err := client.GetRoleScopes(tt.roleID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, scopes)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedScopes, scopes)
			}
		})
	}
}

func TestLogtoManagementClient_GetUserOrganizationRoles(t *testing.T) {
	setupServicesTestEnvironment()

	tests := []struct {
		name           string
		userID         string
		organizationID string
		setupServer    func() *httptest.Server
		expectError    bool
		expectedRoles  []models.LogtoOrganizationRole
	}{
		{
			name:           "successful user organization roles fetch",
			userID:         "user-123",
			organizationID: "org-456",
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
					case "/api/organizations/org-456/users/user-123/roles":
						roles := []models.LogtoOrganizationRole{
							{ID: "org-role-1", Name: "Owner"},
							{ID: "org-role-2", Name: "Distributor"},
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(roles)
					}
				}))
			},
			expectError: false,
			expectedRoles: []models.LogtoOrganizationRole{
				{ID: "org-role-1", Name: "Owner"},
				{ID: "org-role-2", Name: "Distributor"},
			},
		},
		{
			name:           "empty organization roles returns empty array",
			userID:         "user-no-org-roles",
			organizationID: "org-empty",
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
					case "/api/organizations/org-empty/users/user-no-org-roles/roles":
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode([]models.LogtoOrganizationRole{})
					}
				}))
			},
			expectError:   false,
			expectedRoles: []models.LogtoOrganizationRole{},
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
			roles, err := client.GetUserOrganizationRoles(tt.organizationID, tt.userID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, roles)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedRoles, roles)
			}
		})
	}
}

func TestLogtoManagementClient_GetRoleByName(t *testing.T) {
	setupServicesTestEnvironment()

	tests := []struct {
		name         string
		roleName     string
		setupServer  func() *httptest.Server
		expectError  bool
		expectedRole *models.LogtoRole
	}{
		{
			name:     "role found by name",
			roleName: "Admin",
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
					case "/api/roles":
						roles := []models.LogtoRole{
							{ID: "role-1", Name: "Admin", Description: "Administrator", Type: "User"},
							{ID: "role-2", Name: "Support", Description: "Support user", Type: "User"},
							{ID: "role-3", Name: "Manager", Description: "Manager", Type: "User"},
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(roles)
					}
				}))
			},
			expectError: false,
			expectedRole: &models.LogtoRole{
				ID:          "role-1",
				Name:        "Admin",
				Description: "Administrator",
				Type:        "User",
			},
		},
		{
			name:     "role not found by name",
			roleName: "NonExistent",
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
					case "/api/roles":
						roles := []models.LogtoRole{
							{ID: "role-1", Name: "Admin", Description: "Administrator", Type: "User"},
							{ID: "role-2", Name: "Support", Description: "Support user", Type: "User"},
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(roles)
					}
				}))
			},
			expectError:  true,
			expectedRole: nil,
		},
		{
			name:     "empty roles list",
			roleName: "AnyRole",
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
					case "/api/roles":
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode([]models.LogtoRole{})
					}
				}))
			},
			expectError:  true,
			expectedRole: nil,
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
			role, err := client.GetRoleByName(tt.roleName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, role)
				if tt.roleName == "NonExistent" {
					assert.Contains(t, err.Error(), "role 'NonExistent' not found")
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedRole, role)
			}
		})
	}
}

func TestLogtoManagementClient_AssignUserRoles(t *testing.T) {
	setupServicesTestEnvironment()

	tests := []struct {
		name        string
		userID      string
		roleIDs     []string
		setupServer func() *httptest.Server
		expectError bool
	}{
		{
			name:    "successful role assignment",
			userID:  "user-123",
			roleIDs: []string{"role-1", "role-2"},
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
					} else if r.URL.Path == "/api/users/user-123/roles" && r.Method == "POST" {
						// Verify request body
						var requestBody map[string]interface{}
						_ = json.NewDecoder(r.Body).Decode(&requestBody)

						roleIds, ok := requestBody["roleIds"].([]interface{})
						require.True(t, ok)
						assert.Len(t, roleIds, 2)
						assert.Equal(t, "role-1", roleIds[0])
						assert.Equal(t, "role-2", roleIds[1])

						w.WriteHeader(http.StatusCreated)
						_, _ = w.Write([]byte(`{"success": true}`))
					}
				}))
			},
			expectError: false,
		},
		{
			name:    "user not found for role assignment",
			userID:  "nonexistent-user",
			roleIDs: []string{"role-1"},
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
					case "/api/users/nonexistent-user/roles":
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(`{"error": "user not found"}`))
					}
				}))
			},
			expectError: true,
		},
		{
			name:    "empty role IDs succeeds",
			userID:  "user-empty-roles",
			roleIDs: []string{},
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
					case "/api/users/user-empty-roles/roles":
						var requestBody map[string]interface{}
						_ = json.NewDecoder(r.Body).Decode(&requestBody)

						roleIds, ok := requestBody["roleIds"].([]interface{})
						require.True(t, ok)
						assert.Len(t, roleIds, 0)

						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`{"success": true}`))
					}
				}))
			},
			expectError: false,
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
			err := client.AssignUserRoles(tt.userID, tt.roleIDs)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEnrichUserWithRolesAndPermissions(t *testing.T) {
	setupServicesTestEnvironment()

	t.Run("user with no roles returns empty permissions", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/oidc/token":
				response := models.LogtoManagementTokenResponse{
					AccessToken: "test-token",
					TokenType:   "Bearer",
					ExpiresIn:   3600,
					Scope:       "all",
				}
				_ = json.NewEncoder(w).Encode(response)
			case "/api/users/user-no-roles/roles":
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode([]models.LogtoRole{})
			case "/api/users/user-no-roles/organizations":
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode([]models.LogtoOrganization{})
			}
		}))
		defer server.Close()

		originalIssuer := configuration.Config.LogtoIssuer
		originalBaseURL := configuration.Config.LogtoManagementBaseURL
		configuration.Config.LogtoIssuer = server.URL
		configuration.Config.LogtoManagementBaseURL = server.URL + "/api"
		defer func() {
			configuration.Config.LogtoIssuer = originalIssuer
			configuration.Config.LogtoManagementBaseURL = originalBaseURL
		}()

		user, err := EnrichUserWithRolesAndPermissions("user-no-roles")

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "user-no-roles", user.ID)
		assert.Empty(t, user.UserRoles)
		assert.Empty(t, user.UserPermissions)
		assert.Empty(t, user.OrgRole)
		assert.Empty(t, user.OrgPermissions)
		assert.Empty(t, user.OrganizationID)
		assert.Empty(t, user.OrganizationName)
	})

	// Skip the complex test for now - too many API calls to mock properly
	t.Run("basic user enrichment succeeds", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/oidc/token":
				response := models.LogtoManagementTokenResponse{
					AccessToken: "test-token",
					TokenType:   "Bearer",
					ExpiresIn:   3600,
					Scope:       "all",
				}
				_ = json.NewEncoder(w).Encode(response)
			case "/api/users/user-basic/roles":
				roles := []models.LogtoRole{
					{ID: "role-1", Name: "Admin", Description: "Administrator", Type: "User"},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(roles)
			case "/api/roles/role-1/scopes":
				scopes := []models.LogtoScope{
					{ID: "scope-1", Name: "manage:systems", Description: "Manage systems", ResourceID: "api"},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(scopes)
			case "/api/users/user-basic/organizations":
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode([]models.LogtoOrganization{})
			}
		}))
		defer server.Close()

		originalIssuer := configuration.Config.LogtoIssuer
		originalBaseURL := configuration.Config.LogtoManagementBaseURL
		configuration.Config.LogtoIssuer = server.URL
		configuration.Config.LogtoManagementBaseURL = server.URL + "/api"
		defer func() {
			configuration.Config.LogtoIssuer = originalIssuer
			configuration.Config.LogtoManagementBaseURL = originalBaseURL
		}()

		user, err := EnrichUserWithRolesAndPermissions("user-basic")

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "user-basic", user.ID)
		assert.Contains(t, user.UserRoles, "Admin")
		assert.Contains(t, user.UserPermissions, "manage:systems")
	})
}
