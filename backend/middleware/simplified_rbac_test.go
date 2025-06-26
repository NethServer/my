package middleware

import (
	"net/http"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logs"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/testutils"
	"github.com/stretchr/testify/assert"
)

func TestRequirePermission(t *testing.T) {
	// Set up test environment
	setupTestEnvironment()

	tests := []struct {
		name              string
		user              *models.User
		requiredPermission string
		expectedStatus    int
		expectMessage     string
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
				Username: "god-user",
				OrgRole:  "God",
			},
			requiredRole:   "God",
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
			requiredRoles:  []string{"God", "Distributor", "Reseller"},
			expectedStatus: http.StatusOK,
		},
		{
			name: "user with none of required org roles fails",
			user: &models.User{
				ID:       "user-2",
				Username: "customer-user",
				OrgRole:  "Customer",
			},
			requiredRoles:  []string{"God", "Distributor"},
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

func TestAutoPermissionRBAC(t *testing.T) {
	setupTestEnvironment()

	tests := []struct {
		name           string
		user           *models.User
		method         string
		resource       string
		expectedStatus int
		expectMessage  string
	}{
		{
			name: "GET request with read permission passes",
			user: &models.User{
				ID:              "user-1",
				Username:        "reader-user",
				UserPermissions: []string{"read:systems"},
			},
			method:         "GET",
			resource:       "systems",
			expectedStatus: http.StatusOK,
		},
		{
			name: "POST request with create permission passes",
			user: &models.User{
				ID:              "user-2",
				Username:        "creator-user",
				UserPermissions: []string{"create:accounts"},
			},
			method:         "POST",
			resource:       "accounts",
			expectedStatus: http.StatusOK,
		},
		{
			name: "PUT request with manage permission passes",
			user: &models.User{
				ID:              "user-3",
				Username:        "manager-user",
				OrgPermissions:  []string{"manage:systems"},
			},
			method:         "PUT",
			resource:       "systems",
			expectedStatus: http.StatusOK,
		},
		{
			name: "DELETE request without manage permission fails",
			user: &models.User{
				ID:              "user-4",
				Username:        "reader-user",
				UserPermissions: []string{"read:systems"},
			},
			method:         "DELETE",
			resource:       "systems",
			expectedStatus: http.StatusForbidden,
			expectMessage:  "insufficient permissions",
		},
		{
			name: "unsupported method fails",
			user: &models.User{
				ID:       "user-5",
				Username: "any-user",
			},
			method:         "OPTIONS",
			resource:       "systems",
			expectedStatus: http.StatusMethodNotAllowed,
			expectMessage:  "method not allowed",
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
			router.Use(AutoPermissionRBAC(tt.resource))
			
			// Add route for the specific method
			switch tt.method {
			case "GET":
				router.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"status": "authorized"})
				})
			case "POST":
				router.POST("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"status": "authorized"})
				})
			case "PUT":
				router.PUT("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"status": "authorized"})
				})
			case "DELETE":
				router.DELETE("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"status": "authorized"})
				})
			default:
				router.Handle(tt.method, "/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"status": "authorized"})
				})
			}

			// Make request
			w := testutils.MakeRequest(t, router, tt.method, "/test", nil, nil)
			
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
	t.Run("hasPermissionInList", func(t *testing.T) {
		permissions := []string{"read:systems", "manage:accounts", "view:logs"}
		
		assert.True(t, hasPermissionInList(permissions, "read:systems"))
		assert.True(t, hasPermissionInList(permissions, "manage:accounts"))
		assert.False(t, hasPermissionInList(permissions, "delete:systems"))
		assert.False(t, hasPermissionInList([]string{}, "any:permission"))
	})

	t.Run("hasRoleInList", func(t *testing.T) {
		roles := []string{"Admin", "Support", "Viewer"}
		
		assert.True(t, hasRoleInList(roles, "Admin"))
		assert.True(t, hasRoleInList(roles, "Support"))
		assert.False(t, hasRoleInList(roles, "SuperAdmin"))
		assert.False(t, hasRoleInList([]string{}, "Admin"))
	})

	t.Run("buildPermissionFromMethod", func(t *testing.T) {
		assert.Equal(t, "read:systems", buildPermissionFromMethod("GET", "systems"))
		assert.Equal(t, "create:accounts", buildPermissionFromMethod("POST", "accounts"))
		assert.Equal(t, "manage:users", buildPermissionFromMethod("PUT", "users"))
		assert.Equal(t, "manage:data", buildPermissionFromMethod("PATCH", "data"))
		assert.Equal(t, "manage:files", buildPermissionFromMethod("DELETE", "files"))
		assert.Equal(t, "", buildPermissionFromMethod("OPTIONS", "anything"))
		assert.Equal(t, "", buildPermissionFromMethod("INVALID", "resource"))
	})
}

func TestLegacyAliases(t *testing.T) {
	setupTestEnvironment()
	
	user := &models.User{
		ID:              "test-user",
		Username:        "testuser",
		UserRoles:       []string{"Admin"},
		UserPermissions: []string{"manage:systems"},
	}

	t.Run("AutoRoleRBAC alias works", func(t *testing.T) {
		router := testutils.SetupTestGin()
		router.Use(func(c *gin.Context) {
			c.Set("user", user)
			c.Next()
		})
		router.Use(AutoRoleRBAC("Admin"))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "authorized"})
		})

		w := testutils.MakeRequest(t, router, "GET", "/test", nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("RequireScope alias works", func(t *testing.T) {
		router := testutils.SetupTestGin()
		router.Use(func(c *gin.Context) {
			c.Set("user", user)
			c.Next()
		})
		router.Use(RequireScope("manage:systems"))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "authorized"})
		})

		w := testutils.MakeRequest(t, router, "GET", "/test", nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// Helper function to set up test environment
func setupTestEnvironment() {
	if !isTestEnvironmentSetup {
		// Set test environment variables
		os.Setenv("JWT_SECRET", "test-secret-key-for-testing-only")
		os.Setenv("JWT_ISSUER", "test-issuer")
		os.Setenv("JWT_EXPIRATION", "24h")
		os.Setenv("JWT_REFRESH_EXPIRATION", "168h")
		os.Setenv("LOGTO_ISSUER", "https://test-logto.example.com")
		os.Setenv("LOGTO_AUDIENCE", "test-api-resource")
		os.Setenv("LOGTO_MANAGEMENT_CLIENT_ID", "test-client-id")
		os.Setenv("LOGTO_MANAGEMENT_CLIENT_SECRET", "test-client-secret")

		gin.SetMode(gin.TestMode)
		logs.Init("[TEST]")
		configuration.Init()
		
		isTestEnvironmentSetup = true
	}
}

var isTestEnvironmentSetup bool