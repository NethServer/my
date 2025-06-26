package middleware

import (
	"net/http"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/jwt"
	"github.com/nethesis/my/backend/logs"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Set test environment variables manually
	os.Setenv("JWT_SECRET", "test-secret-key-for-testing-only")
	os.Setenv("JWT_ISSUER", "test-issuer")
	os.Setenv("JWT_EXPIRATION", "24h")
	os.Setenv("JWT_REFRESH_EXPIRATION", "168h")
	os.Setenv("LOGTO_ISSUER", "https://test-logto.example.com")
	os.Setenv("LOGTO_AUDIENCE", "test-api-resource")
	os.Setenv("LOGTO_MANAGEMENT_CLIENT_ID", "test-client-id")
	os.Setenv("LOGTO_MANAGEMENT_CLIENT_SECRET", "test-client-secret")

	gin.SetMode(gin.TestMode)

	// Initialize logs
	logs.Init("[TEST]")

	// Initialize configuration
	configuration.Init()

	code := m.Run()
	os.Exit(code)
}

func TestCustomAuthMiddleware(t *testing.T) {
	// Create a test user and valid JWT
	testUser := models.User{
		ID:               "test-user-123",
		Username:         "testuser",
		Email:            "test@example.com",
		Name:             "Test User",
		OrganizationID:   "test-org-456",
		OrganizationName: "Test Organization",
		UserRoles:        []string{"Admin"},
		UserPermissions:  []string{"manage:systems"},
		OrgRole:          "Customer",
		OrgPermissions:   []string{"view:systems"},
	}

	validToken, err := jwt.GenerateCustomToken(testUser)
	require.NoError(t, err)

	tests := []struct {
		name             string
		authHeader       string
		expectedStatus   int
		expectUser       bool
		validateResponse func(t *testing.T, response map[string]interface{})
	}{
		{
			name:           "valid JWT token authenticates successfully",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectUser:     true,
			validateResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "authenticated", response["status"])
				assert.Equal(t, "test-user-123", response["user_id"])
				assert.Equal(t, "testuser", response["username"])
			},
		},
		{
			name:           "missing authorization header returns 401",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectUser:     false,
			validateResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response["message"], "Authorization header")
			},
		},
		{
			name:           "malformed authorization header returns 401",
			authHeader:     "NotBearer token",
			expectedStatus: http.StatusUnauthorized,
			expectUser:     false,
			validateResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response["message"], "Authorization header format")
			},
		},
		{
			name:           "invalid JWT token returns 401",
			authHeader:     "Bearer invalid.jwt.token",
			expectedStatus: http.StatusUnauthorized,
			expectUser:     false,
			validateResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response["message"], "Invalid token")
			},
		},
		{
			name:           "empty bearer token returns 401",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			expectUser:     false,
			validateResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response["message"], "Invalid token")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup router with middleware
			router := testutils.SetupTestGin()
			router.Use(CustomAuthMiddleware())

			// Add a protected endpoint that returns user info if authenticated
			router.GET("/protected", func(c *gin.Context) {
				user, exists := c.Get("user")
				if exists {
					userObj := user.(*models.User)
					c.JSON(http.StatusOK, gin.H{
						"status":   "authenticated",
						"user_id":  userObj.ID,
						"username": userObj.Username,
					})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "User not found in context",
					})
				}
			})

			// Prepare headers
			headers := make(map[string]string)
			if tt.authHeader != "" {
				headers["Authorization"] = tt.authHeader
			}

			// Make request
			w := testutils.MakeRequest(t, router, "GET", "/protected", nil, headers)

			// Validate response
			response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)
			tt.validateResponse(t, response)
		})
	}
}

func TestCustomAuthMiddleware_UserInContext(t *testing.T) {
	// Create test user and token
	testUser := models.User{
		ID:               "test-user-123",
		Username:         "testuser",
		Email:            "test@example.com",
		OrganizationID:   "test-org-456",
		OrganizationName: "Test Organization",
		UserRoles:        []string{"Admin"},
		UserPermissions:  []string{"manage:systems"},
		OrgRole:          "Customer",
		OrgPermissions:   []string{"view:systems"},
	}

	validToken, err := jwt.GenerateCustomToken(testUser)
	require.NoError(t, err)

	// Setup router
	router := testutils.SetupTestGin()
	router.Use(CustomAuthMiddleware())

	// Add endpoint that checks user context
	router.GET("/check-user", func(c *gin.Context) {
		user, exists := c.Get("user")
		assert.True(t, exists, "User should exist in context")

		userObj, ok := user.(*models.User)
		assert.True(t, ok, "User should be of correct type")
		assert.Equal(t, testUser.ID, userObj.ID)
		assert.Equal(t, testUser.Username, userObj.Username)
		assert.Equal(t, testUser.Email, userObj.Email)
		assert.Equal(t, testUser.OrganizationID, userObj.OrganizationID)
		assert.Equal(t, testUser.UserRoles, userObj.UserRoles)
		assert.Equal(t, testUser.UserPermissions, userObj.UserPermissions)
		assert.Equal(t, testUser.OrgRole, userObj.OrgRole)
		assert.Equal(t, testUser.OrgPermissions, userObj.OrgPermissions)

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Make authenticated request
	headers := map[string]string{
		"Authorization": "Bearer " + validToken,
	}

	w := testutils.MakeRequest(t, router, "GET", "/check-user", nil, headers)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCustomAuthMiddleware_HandlesJWTErrors(t *testing.T) {
	// Test with various JWT error conditions
	tests := []struct {
		name           string
		setupEnv       func()
		token          string
		expectedStatus int
		expectError    string
	}{
		{
			name: "missing JWT secret in environment",
			setupEnv: func() {
				os.Unsetenv("JWT_SECRET")
			},
			token:          "any.jwt.token",
			expectedStatus: http.StatusUnauthorized,
			expectError:    "Invalid token",
		},
		{
			name: "token with wrong signature",
			setupEnv: func() {
				os.Setenv("JWT_SECRET", "different-secret")
			},
			token:          "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0IiwiZXhwIjo5OTk5OTk5OTk5fQ.wrong-signature",
			expectedStatus: http.StatusUnauthorized,
			expectError:    "Invalid token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			originalSecret := os.Getenv("JWT_SECRET")
			tt.setupEnv()
			defer os.Setenv("JWT_SECRET", originalSecret)

			// Setup router
			router := testutils.SetupTestGin()
			router.Use(CustomAuthMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			// Make request
			headers := map[string]string{
				"Authorization": "Bearer " + tt.token,
			}

			w := testutils.MakeRequest(t, router, "GET", "/test", nil, headers)
			response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)

			assert.Contains(t, response["message"], tt.expectError)
		})
	}
}
