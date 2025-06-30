package methods

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/testutils"
	"github.com/stretchr/testify/assert"
)

func TestGetProfile(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		expectedStatus int
		expectMessage  string
		expectUser     bool
	}{
		{
			name: "valid user in context returns profile",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:               "test-user-123",
					Username:         "testuser",
					Email:            "test@example.com",
					Name:             "Test User",
					UserRoles:        []string{"Admin"},
					UserPermissions:  []string{"manage:systems"},
					OrgRole:          "Customer",
					OrgPermissions:   []string{"view:systems"},
					OrganizationID:   "org-123",
					OrganizationName: "Test Org",
				}
				c.Set("user", user)
			},
			expectedStatus: http.StatusOK,
			expectUser:     true,
		},
		{
			name: "missing user in context fails",
			setupContext: func(c *gin.Context) {
				// Don't set user in context
			},
			expectedStatus: http.StatusUnauthorized,
			expectMessage:  "user not authenticated",
			expectUser:     false,
		},
		{
			name: "invalid user type in context fails",
			setupContext: func(c *gin.Context) {
				c.Set("user", "not-a-user-object")
			},
			expectedStatus: http.StatusInternalServerError,
			expectMessage:  "invalid user context",
			expectUser:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := testutils.SetupTestGin()
			router.GET("/profile", func(c *gin.Context) {
				tt.setupContext(c)
				GetProfile(c)
			})

			w := testutils.MakeRequest(t, router, "GET", "/profile", nil, nil)

			assert.Equal(t, tt.expectedStatus, w.Code)

			response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)

			if tt.expectedStatus == http.StatusOK {
				assert.Contains(t, response["message"], "user profile retrieved successfully")
				assert.NotNil(t, response["data"])

				// Check user data is present
				userData, ok := response["data"].(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, "test-user-123", userData["id"])
				assert.Equal(t, "testuser", userData["username"])
				assert.Equal(t, "test@example.com", userData["email"])
			} else if tt.expectMessage != "" {
				assert.Contains(t, response["message"], tt.expectMessage)
			}
		})
	}
}

func TestGetProtectedResource(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		expectedStatus int
		expectMessage  string
	}{
		{
			name: "valid user_id in context succeeds",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "test-user-123")
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing user_id in context fails",
			setupContext: func(c *gin.Context) {
				// Don't set user_id in context
			},
			expectedStatus: http.StatusUnauthorized,
			expectMessage:  "user not authenticated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := testutils.SetupTestGin()
			router.GET("/protected", func(c *gin.Context) {
				tt.setupContext(c)
				GetProtectedResource(c)
			})

			w := testutils.MakeRequest(t, router, "GET", "/protected", nil, nil)

			assert.Equal(t, tt.expectedStatus, w.Code)

			response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)

			if tt.expectedStatus == http.StatusOK {
				assert.Contains(t, response["message"], "protected resource accessed successfully")

				data, ok := response["data"].(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, "test-user-123", data["user_id"])
				assert.Equal(t, "sensitive data", data["resource"])
			} else if tt.expectMessage != "" {
				assert.Contains(t, response["message"], tt.expectMessage)
			}
		})
	}
}

func TestGetUserPermissions(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		expectedStatus int
		expectMessage  string
		expectedData   map[string]interface{}
	}{
		{
			name: "valid user returns permissions data",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:               "test-user-123",
					Username:         "testuser",
					Email:            "test@example.com",
					Name:             "Test User",
					UserRoles:        []string{"Admin", "Support"},
					UserPermissions:  []string{"manage:systems", "view:logs"},
					OrgRole:          "Distributor",
					OrgPermissions:   []string{"create:resellers", "manage:customers"},
					OrganizationID:   "org-456",
					OrganizationName: "ACME Distribution",
				}
				c.Set("user", user)
			},
			expectedStatus: http.StatusOK,
			expectedData: map[string]interface{}{
				"user_roles":        []string{"Admin", "Support"},
				"user_permissions":  []string{"manage:systems", "view:logs"},
				"org_role":          "Distributor",
				"org_permissions":   []string{"create:resellers", "manage:customers"},
				"organization_id":   "org-456",
				"organization_name": "ACME Distribution",
			},
		},
		{
			name: "user with empty permissions returns empty arrays",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:               "test-user-456",
					Username:         "limiteduser",
					UserRoles:        []string{},
					UserPermissions:  []string{},
					OrgRole:          "",
					OrgPermissions:   []string{},
					OrganizationID:   "",
					OrganizationName: "",
				}
				c.Set("user", user)
			},
			expectedStatus: http.StatusOK,
			expectedData: map[string]interface{}{
				"user_roles":        []string{},
				"user_permissions":  []string{},
				"org_role":          "",
				"org_permissions":   []string{},
				"organization_id":   "",
				"organization_name": "",
			},
		},
		{
			name: "missing user in context fails",
			setupContext: func(c *gin.Context) {
				// Don't set user in context
			},
			expectedStatus: http.StatusUnauthorized,
			expectMessage:  "user not authenticated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := testutils.SetupTestGin()
			router.GET("/permissions", func(c *gin.Context) {
				tt.setupContext(c)
				GetUserPermissions(c)
			})

			w := testutils.MakeRequest(t, router, "GET", "/permissions", nil, nil)

			assert.Equal(t, tt.expectedStatus, w.Code)

			response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)

			if tt.expectedStatus == http.StatusOK {
				assert.Contains(t, response["message"], "user permissions retrieved successfully")

				data, ok := response["data"].(map[string]interface{})
				assert.True(t, ok)

				// Check all expected data fields
				for key, expectedValue := range tt.expectedData {
					actualValue := data[key]

					// Handle slice type conversion for JSON serialization
					if expectedSlice, ok := expectedValue.([]string); ok {
						if actualSlice, ok := actualValue.([]interface{}); ok {
							// Convert []interface{} to []string for comparison
							actualStrings := make([]string, len(actualSlice))
							for i, v := range actualSlice {
								if str, ok := v.(string); ok {
									actualStrings[i] = str
								}
							}
							assert.Equal(t, expectedSlice, actualStrings, "Mismatch for key %s", key)
						} else {
							assert.Equal(t, expectedValue, actualValue, "Mismatch for key %s", key)
						}
					} else {
						assert.Equal(t, expectedValue, actualValue, "Mismatch for key %s", key)
					}
				}
			} else if tt.expectMessage != "" {
				assert.Contains(t, response["message"], tt.expectMessage)
			}
		})
	}
}

func TestGetUserProfile(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		expectedStatus int
		expectMessage  string
		expectUser     bool
	}{
		{
			name: "complete user profile returns all fields",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:               "profile-user-789",
					Username:         "profileuser",
					Email:            "profile@example.com",
					Name:             "Profile User",
					UserRoles:        []string{"Admin"},
					UserPermissions:  []string{"manage:systems", "admin:accounts"},
					OrgRole:          "God",
					OrgPermissions:   []string{"create:distributors", "manage:all"},
					OrganizationID:   "org-nethesis",
					OrganizationName: "Nethesis",
				}
				c.Set("user", user)
			},
			expectedStatus: http.StatusOK,
			expectUser:     true,
		},
		{
			name: "user with minimal data succeeds",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:       "minimal-user",
					Username: "minimal",
				}
				c.Set("user", user)
			},
			expectedStatus: http.StatusOK,
			expectUser:     true,
		},
		{
			name: "missing user context fails",
			setupContext: func(c *gin.Context) {
				// Don't set user
			},
			expectedStatus: http.StatusUnauthorized,
			expectMessage:  "user not authenticated",
			expectUser:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := testutils.SetupTestGin()
			router.GET("/user/profile", func(c *gin.Context) {
				tt.setupContext(c)
				GetUserProfile(c)
			})

			w := testutils.MakeRequest(t, router, "GET", "/user/profile", nil, nil)

			assert.Equal(t, tt.expectedStatus, w.Code)

			response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)

			if tt.expectedStatus == http.StatusOK {
				assert.Contains(t, response["message"], "user profile retrieved successfully")
				assert.NotNil(t, response["data"])

				// Verify it's a user object with expected structure
				userData, ok := response["data"].(map[string]interface{})
				assert.True(t, ok)
				assert.Contains(t, userData, "id")
				assert.Contains(t, userData, "username")
			} else if tt.expectMessage != "" {
				assert.Contains(t, response["message"], tt.expectMessage)
			}
		})
	}
}

func TestUserMethodsIntegration(t *testing.T) {
	// Test that all user methods work together with the same user context
	user := &models.User{
		ID:               "integration-user-123",
		Username:         "integrationuser",
		Email:            "integration@example.com",
		Name:             "Integration Test User",
		UserRoles:        []string{"Admin", "Support"},
		UserPermissions:  []string{"manage:systems", "view:logs", "admin:accounts"},
		OrgRole:          "Distributor",
		OrgPermissions:   []string{"create:resellers", "manage:customers"},
		OrganizationID:   "org-integration-456",
		OrganizationName: "Integration Test Org",
	}

	router := testutils.SetupTestGin()

	// Add middleware to set user context
	router.Use(func(c *gin.Context) {
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Next()
	})

	// Register all user endpoints
	router.GET("/profile", GetProfile)
	router.GET("/protected", GetProtectedResource)
	router.GET("/permissions", GetUserPermissions)
	router.GET("/user/profile", GetUserProfile)

	endpoints := []struct {
		path           string
		expectedFields []string
	}{
		{
			path:           "/profile",
			expectedFields: []string{"id", "username", "email", "name"},
		},
		{
			path:           "/protected",
			expectedFields: []string{"user_id", "resource"},
		},
		{
			path:           "/permissions",
			expectedFields: []string{"user_roles", "user_permissions", "org_role", "org_permissions"},
		},
		{
			path:           "/user/profile",
			expectedFields: []string{"id", "username", "email", "name", "user_roles"},
		},
	}

	for _, endpoint := range endpoints {
		t.Run("integration_test_"+endpoint.path, func(t *testing.T) {
			w := testutils.MakeRequest(t, router, "GET", endpoint.path, nil, nil)

			assert.Equal(t, http.StatusOK, w.Code)

			response := testutils.AssertJSONResponse(t, w, http.StatusOK)
			assert.NotNil(t, response["data"])

			data, ok := response["data"].(map[string]interface{})
			assert.True(t, ok)

			// Check that expected fields are present
			for _, field := range endpoint.expectedFields {
				assert.Contains(t, data, field, "Expected field %s not found in response", field)
			}
		})
	}
}
