package middleware

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/jwt"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequirePermission(t *testing.T) {
	// Set up test environment
	setupTestEnvironment()

	tests := []struct {
		name               string
		user               *models.User
		requiredPermission string
		expectedStatus     int
		expectMessage      string
	}{
		{
			name: "user with required user permission passes",
			user: &models.User{
				ID:              "user-1",
				Username:        "admin-user",
				UserPermissions: []string{"manage:systems", "view:logs"},
				OrgPermissions:  []string{"view:reports"},
			},
			requiredPermission: "manage:systems",
			expectedStatus:     http.StatusOK,
		},
		{
			name: "user with required org permission passes",
			user: &models.User{
				ID:              "user-2",
				Username:        "org-user",
				UserPermissions: []string{"view:logs"},
				OrgPermissions:  []string{"manage:systems", "view:reports"},
			},
			requiredPermission: "manage:systems",
			expectedStatus:     http.StatusOK,
		},
		{
			name: "user without required permission fails",
			user: &models.User{
				ID:              "user-3",
				Username:        "limited-user",
				UserPermissions: []string{"view:logs"},
				OrgPermissions:  []string{"view:reports"},
			},
			requiredPermission: "manage:systems",
			expectedStatus:     http.StatusForbidden,
			expectMessage:      "insufficient permissions",
		},
		{
			name: "user with empty permissions fails",
			user: &models.User{
				ID:              "user-4",
				Username:        "no-perms-user",
				UserPermissions: []string{},
				OrgPermissions:  []string{},
			},
			requiredPermission: "manage:systems",
			expectedStatus:     http.StatusForbidden,
			expectMessage:      "insufficient permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create router with middleware
			router := testutils.SetupTestGin()
			router.Use(func(c *gin.Context) {
				c.Set("user", tt.user)
				c.Next()
			})
			router.Use(RequirePermission(tt.requiredPermission))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "authorized"})
			})

			// Make request
			w := testutils.MakeRequest(t, router, "GET", "/test", nil, nil)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus != http.StatusOK {
				response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)
				assert.Contains(t, response["message"], tt.expectMessage)
			}
		})
	}
}

func TestRequireUserRole(t *testing.T) {
	setupTestEnvironment()

	tests := []struct {
		name           string
		user           *models.User
		requiredRole   string
		expectedStatus int
		expectMessage  string
	}{
		{
			name: "user with required role passes",
			user: &models.User{
				ID:        "user-1",
				Username:  "admin-user",
				UserRoles: []string{"Admin", "Support"},
			},
			requiredRole:   "Admin",
			expectedStatus: http.StatusOK,
		},
		{
			name: "user with multiple roles including required passes",
			user: &models.User{
				ID:        "user-2",
				Username:  "support-user",
				UserRoles: []string{"Support", "Viewer"},
			},
			requiredRole:   "Support",
			expectedStatus: http.StatusOK,
		},
		{
			name: "user without required role fails",
			user: &models.User{
				ID:        "user-3",
				Username:  "viewer-user",
				UserRoles: []string{"Viewer"},
			},
			requiredRole:   "Admin",
			expectedStatus: http.StatusForbidden,
			expectMessage:  "insufficient user role",
		},
		{
			name: "user with empty roles fails",
			user: &models.User{
				ID:        "user-4",
				Username:  "no-role-user",
				UserRoles: []string{},
			},
			requiredRole:   "Admin",
			expectedStatus: http.StatusForbidden,
			expectMessage:  "insufficient user role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create router with middleware
			router := testutils.SetupTestGin()
			router.Use(func(c *gin.Context) {
				c.Set("user", tt.user)
				c.Next()
			})
			router.Use(RequireUserRole(tt.requiredRole))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "authorized"})
			})

			// Make request
			w := testutils.MakeRequest(t, router, "GET", "/test", nil, nil)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus != http.StatusOK {
				response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)
				assert.Contains(t, response["message"], tt.expectMessage)
			}
		})
	}
}

func TestRequireOrgRole(t *testing.T) {
	setupTestEnvironment()

	tests := []struct {
		name           string
		user           *models.User
		requiredRole   string
		expectedStatus int
		expectMessage  string
	}{
		{
			name: "user with required org role passes",
			user: &models.User{
				ID:       "user-1",
				Username: "owner-user",
				OrgRole:  "Owner",
			},
			requiredRole:   "Owner",
			expectedStatus: http.StatusOK,
		},
		{
			name: "user with different org role fails",
			user: &models.User{
				ID:       "user-2",
				Username: "customer-user",
				OrgRole:  "Customer",
			},
			requiredRole:   "Distributor",
			expectedStatus: http.StatusForbidden,
			expectMessage:  "insufficient organization role",
		},
		{
			name: "user with empty org role fails",
			user: &models.User{
				ID:       "user-3",
				Username: "no-org-user",
				OrgRole:  "",
			},
			requiredRole:   "Customer",
			expectedStatus: http.StatusForbidden,
			expectMessage:  "insufficient organization role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create router with middleware
			router := testutils.SetupTestGin()
			router.Use(func(c *gin.Context) {
				c.Set("user", tt.user)
				c.Next()
			})
			router.Use(RequireOrgRole(tt.requiredRole))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "authorized"})
			})

			// Make request
			w := testutils.MakeRequest(t, router, "GET", "/test", nil, nil)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus != http.StatusOK {
				response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)
				assert.Contains(t, response["message"], tt.expectMessage)
			}
		})
	}
}

func TestRequireAnyOrgRole(t *testing.T) {
	setupTestEnvironment()

	tests := []struct {
		name           string
		user           *models.User
		requiredRoles  []string
		expectedStatus int
		expectMessage  string
	}{
		{
			name: "user with one of required org roles passes",
			user: &models.User{
				ID:       "user-1",
				Username: "distributor-user",
				OrgRole:  "Distributor",
			},
			requiredRoles:  []string{"Owner", "Distributor", "Reseller"},
			expectedStatus: http.StatusOK,
		},
		{
			name: "user with none of required org roles fails",
			user: &models.User{
				ID:       "user-2",
				Username: "customer-user",
				OrgRole:  "Customer",
			},
			requiredRoles:  []string{"Owner", "Distributor"},
			expectedStatus: http.StatusForbidden,
			expectMessage:  "insufficient organization role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create router with middleware
			router := testutils.SetupTestGin()
			router.Use(func(c *gin.Context) {
				c.Set("user", tt.user)
				c.Next()
			})
			router.Use(RequireAnyOrgRole(tt.requiredRoles...))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "authorized"})
			})

			// Make request
			w := testutils.MakeRequest(t, router, "GET", "/test", nil, nil)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus != http.StatusOK {
				response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)
				assert.Contains(t, response["message"], tt.expectMessage)
			}
		})
	}
}

func TestGetUserFromContext(t *testing.T) {
	setupTestEnvironment()

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		expectedStatus int
		expectMessage  string
	}{
		{
			name: "valid user in context passes",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:              "test-user",
					Username:        "testuser",
					UserPermissions: []string{"any:permission"},
				}
				c.Set("user", user)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing user in context fails",
			setupContext: func(c *gin.Context) {
				// Don't set user in context
			},
			expectedStatus: http.StatusUnauthorized,
			expectMessage:  "user not found in context",
		},
		{
			name: "invalid user type in context fails",
			setupContext: func(c *gin.Context) {
				c.Set("user", "not-a-user-object")
			},
			expectedStatus: http.StatusInternalServerError,
			expectMessage:  "invalid user context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create router with middleware that uses getUserFromContext
			router := testutils.SetupTestGin()
			router.Use(func(c *gin.Context) {
				tt.setupContext(c)
				c.Next()
			})
			router.Use(RequirePermission("any:permission"))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "authorized"})
			})

			// Make request
			w := testutils.MakeRequest(t, router, "GET", "/test", nil, nil)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus != http.StatusOK {
				response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)
				assert.Contains(t, response["message"], tt.expectMessage)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("hasStringInList", func(t *testing.T) {
		permissions := []string{"read:systems", "manage:accounts", "view:logs"}

		assert.True(t, hasStringInList(permissions, "read:systems"))
		assert.True(t, hasStringInList(permissions, "manage:accounts"))
		assert.False(t, hasStringInList(permissions, "delete:systems"))
		assert.False(t, hasStringInList([]string{}, "any:permission"))
	})

	t.Run("hasStringInList", func(t *testing.T) {
		roles := []string{"Admin", "Support", "Viewer"}

		assert.True(t, hasStringInList(roles, "Admin"))
		assert.True(t, hasStringInList(roles, "Support"))
		assert.False(t, hasStringInList(roles, "SuperAdmin"))
		assert.False(t, hasStringInList([]string{}, "Admin"))
	})
}

func TestJWTAuthMiddleware(t *testing.T) {
	setupTestEnvironment()

	// Generate a valid custom token for testing
	user := models.User{
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

	validToken, err := jwt.GenerateCustomToken(user)
	require.NoError(t, err)

	tests := []struct {
		name           string
		setupAuth      func() string
		expectedStatus int
		expectMessage  string
		expectUser     bool
		expectContext  map[string]interface{}
	}{
		{
			name: "valid custom token passes",
			setupAuth: func() string {
				return "Bearer " + validToken
			},
			expectedStatus: http.StatusOK,
			expectUser:     true,
			expectContext: map[string]interface{}{
				"user_id":           "test-user-123",
				"username":          "testuser",
				"email":             "test@example.com",
				"name":              "Test User",
				"organization_id":   "org-123",
				"organization_name": "Test Org",
				"org_role":          "Customer",
			},
		},
		{
			name: "missing authorization header fails",
			setupAuth: func() string {
				return ""
			},
			expectedStatus: http.StatusUnauthorized,
			expectMessage:  "authorization header required",
			expectUser:     false,
		},
		{
			name: "invalid authorization format fails",
			setupAuth: func() string {
				return "InvalidToken"
			},
			expectedStatus: http.StatusUnauthorized,
			expectMessage:  "invalid authorization header format",
			expectUser:     false,
		},
		{
			name: "bearer with empty token fails",
			setupAuth: func() string {
				return "Bearer "
			},
			expectedStatus: http.StatusUnauthorized,
			expectMessage:  "token not provided",
			expectUser:     false,
		},
		{
			name: "invalid token fails",
			setupAuth: func() string {
				return "Bearer invalid.jwt.token"
			},
			expectedStatus: http.StatusUnauthorized,
			expectMessage:  "invalid token",
			expectUser:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := testutils.SetupTestGin()
			router.Use(JWTAuthMiddleware())
			router.GET("/test", func(c *gin.Context) {
				if tt.expectUser {
					user, exists := c.Get("user")
					assert.True(t, exists)
					assert.IsType(t, &models.User{}, user)

					// Check context values
					for key, expectedValue := range tt.expectContext {
						value, exists := c.Get(key)
						assert.True(t, exists, "Expected context key %s to exist", key)
						assert.Equal(t, expectedValue, value, "Context value mismatch for key %s", key)
					}

					// Check array context values
					userRoles, exists := c.Get("user_roles")
					assert.True(t, exists)
					assert.Equal(t, []string{"Admin"}, userRoles)

					userPermissions, exists := c.Get("user_permissions")
					assert.True(t, exists)
					assert.Equal(t, []string{"manage:systems"}, userPermissions)

					orgPermissions, exists := c.Get("org_permissions")
					assert.True(t, exists)
					assert.Equal(t, []string{"view:systems"}, orgPermissions)
				}
				c.JSON(http.StatusOK, gin.H{"status": "authorized"})
			})

			headers := map[string]string{}
			if authHeader := tt.setupAuth(); authHeader != "" {
				headers["Authorization"] = authHeader
			}

			w := testutils.MakeRequest(t, router, "GET", "/test", nil, headers)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus != http.StatusOK {
				response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)
				assert.Contains(t, response["message"], tt.expectMessage)
			}
		})
	}
}

func TestJWTAuthMiddlewareEdgeCases(t *testing.T) {
	setupTestEnvironment()

	tests := []struct {
		name           string
		setupToken     func() string
		expectedStatus int
		expectMessage  string
	}{
		{
			name: "bearer prefix case sensitivity",
			setupToken: func() string {
				user := models.User{ID: "test-user", Username: "testuser"}
				token, _ := jwt.GenerateCustomToken(user)
				return "bearer " + token // lowercase bearer
			},
			expectedStatus: http.StatusUnauthorized,
			expectMessage:  "invalid authorization header format",
		},
		{
			name: "extra spaces in bearer token",
			setupToken: func() string {
				user := models.User{ID: "test-user", Username: "testuser"}
				token, _ := jwt.GenerateCustomToken(user)
				return "Bearer  " + token // extra space
			},
			expectedStatus: http.StatusUnauthorized,
			expectMessage:  "invalid token",
		},
		{
			name: "very long invalid token",
			setupToken: func() string {
				return "Bearer " + strings.Repeat("invalid", 1000)
			},
			expectedStatus: http.StatusUnauthorized,
			expectMessage:  "invalid token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := testutils.SetupTestGin()
			router.Use(JWTAuthMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "authorized"})
			})

			headers := map[string]string{
				"Authorization": tt.setupToken(),
			}

			w := testutils.MakeRequest(t, router, "GET", "/test", nil, headers)

			assert.Equal(t, tt.expectedStatus, w.Code)
			response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)
			assert.Contains(t, response["message"], tt.expectMessage)
		})
	}
}

func TestMiddlewareChaining(t *testing.T) {
	setupTestEnvironment()

	// Generate a valid token for a user with specific permissions
	user := models.User{
		ID:              "chain-test-user",
		Username:        "chainuser",
		UserRoles:       []string{"Admin"},
		UserPermissions: []string{"manage:systems", "view:logs"},
		OrgRole:         "Distributor",
		OrgPermissions:  []string{"create:resellers"},
	}

	validToken, err := jwt.GenerateCustomToken(user)
	require.NoError(t, err)

	tests := []struct {
		name            string
		setupMiddleware func(*gin.RouterGroup)
		authHeader      string
		expectedStatus  int
		expectMessage   string
	}{
		{
			name: "JWT + RequirePermission chain succeeds",
			setupMiddleware: func(rg *gin.RouterGroup) {
				rg.Use(JWTAuthMiddleware())
				rg.Use(RequirePermission("manage:systems"))
			},
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
		},
		{
			name: "JWT + RequirePermission chain fails for missing permission",
			setupMiddleware: func(rg *gin.RouterGroup) {
				rg.Use(JWTAuthMiddleware())
				rg.Use(RequirePermission("delete:systems"))
			},
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusForbidden,
			expectMessage:  "insufficient permissions",
		},
		{
			name: "JWT + RequireUserRole chain succeeds",
			setupMiddleware: func(rg *gin.RouterGroup) {
				rg.Use(JWTAuthMiddleware())
				rg.Use(RequireUserRole("Admin"))
			},
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
		},
		{
			name: "JWT + RequireOrgRole chain succeeds",
			setupMiddleware: func(rg *gin.RouterGroup) {
				rg.Use(JWTAuthMiddleware())
				rg.Use(RequireOrgRole("Distributor"))
			},
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
		},
		{
			name: "JWT + multiple middleware chain succeeds",
			setupMiddleware: func(rg *gin.RouterGroup) {
				rg.Use(JWTAuthMiddleware())
				rg.Use(RequireUserRole("Admin"))
				rg.Use(RequirePermission("manage:systems"))
				rg.Use(RequireOrgRole("Distributor"))
			},
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid token prevents access to protected endpoint",
			setupMiddleware: func(rg *gin.RouterGroup) {
				rg.Use(JWTAuthMiddleware())
				rg.Use(RequirePermission("manage:systems"))
			},
			authHeader:     "Bearer invalid.token",
			expectedStatus: http.StatusUnauthorized,
			expectMessage:  "invalid token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := testutils.SetupTestGin()
			group := router.Group("/api")
			tt.setupMiddleware(group)
			group.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			headers := map[string]string{
				"Authorization": tt.authHeader,
			}

			w := testutils.MakeRequest(t, router, "GET", "/api/test", nil, headers)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus != http.StatusOK {
				response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)
				assert.Contains(t, response["message"], tt.expectMessage)
			}
		})
	}
}

// Helper function to set up test environment
func setupTestEnvironment() {
	if !isTestEnvironmentSetup {
		// Set test environment variables
		_ = os.Setenv("TENANT_ID", "test-tenant")
		_ = os.Setenv("TENANT_DOMAIN", "test-domain.com")
		_ = os.Setenv("APP_URL", "https://test-app.com")
		_ = os.Setenv("JWT_SECRET", "test-secret-key-for-testing-only")
		_ = os.Setenv("JWT_ISSUER", "test-issuer")
		_ = os.Setenv("JWT_EXPIRATION", "24h")
		_ = os.Setenv("JWT_REFRESH_EXPIRATION", "168h")
		_ = os.Setenv("BACKEND_APP_ID", "test-client-id")
		_ = os.Setenv("BACKEND_APP_SECRET", "test-client-secret")
		_ = os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test_db")
		_ = os.Setenv("REDIS_URL", "redis://localhost:6379")

		gin.SetMode(gin.TestMode)
		_ = logger.Init(&logger.Config{Level: logger.InfoLevel, Format: logger.JSONFormat, Output: logger.StdoutOutput, AppName: "[TEST]"})
		configuration.Init()

		isTestEnvironmentSetup = true
	}
}

var isTestEnvironmentSetup bool
