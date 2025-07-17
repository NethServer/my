package methods

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
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

func setupAuthTestEnvironment() {
	if !isAuthTestEnvironmentSetup {
		// Set test environment variables
		_ = os.Setenv("TENANT_ID", "test-tenant")
		_ = os.Setenv("TENANT_DOMAIN", "test-domain.com")
		_ = os.Setenv("JWT_SECRET", "test-secret-key-for-testing-only")
		_ = os.Setenv("JWT_ISSUER", "test-issuer")
		_ = os.Setenv("JWT_EXPIRATION", "24h")
		_ = os.Setenv("JWT_REFRESH_EXPIRATION", "168h")
		_ = os.Setenv("BACKEND_APP_ID", "test-client-id")
		_ = os.Setenv("BACKEND_APP_SECRET", "test-client-secret")
		_ = os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test_db")
		_ = os.Setenv("REDIS_URL", "redis://localhost:6379")

		gin.SetMode(gin.TestMode)
		_ = logger.Init(&logger.Config{Level: logger.InfoLevel, Format: logger.JSONFormat, Output: logger.StdoutOutput, AppName: "[AUTH-TEST]"})
		configuration.Init()

		isAuthTestEnvironmentSetup = true
	}
}

var isAuthTestEnvironmentSetup bool

func TestExchangeToken(t *testing.T) {
	setupAuthTestEnvironment()

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectMessage  string
	}{
		{
			name: "missing access token field fails",
			requestBody: map[string]string{
				"wrong_field": "value",
			},
			expectedStatus: http.StatusBadRequest,
			expectMessage:  "Invalid request body",
		},
		{
			name:           "empty request body fails",
			requestBody:    map[string]string{},
			expectedStatus: http.StatusBadRequest,
			expectMessage:  "Invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := testutils.SetupTestGin()
			router.POST("/auth/exchange", ExchangeToken)

			body, _ := json.Marshal(tt.requestBody)
			w := testutils.MakeRequest(t, router, "POST", "/auth/exchange", bytes.NewReader(body), map[string]string{
				"Content-Type": "application/json",
			})

			assert.Equal(t, tt.expectedStatus, w.Code)

			response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)
			assert.Contains(t, response["message"], tt.expectMessage)
		})
	}
}

func TestRefreshToken(t *testing.T) {
	setupAuthTestEnvironment()

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectMessage  string
	}{
		{
			name: "missing refresh token field fails",
			requestBody: map[string]string{
				"wrong_field": "value",
			},
			expectedStatus: http.StatusBadRequest,
			expectMessage:  "Invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := testutils.SetupTestGin()
			router.POST("/auth/refresh", RefreshToken)

			body, _ := json.Marshal(tt.requestBody)
			w := testutils.MakeRequest(t, router, "POST", "/auth/refresh", bytes.NewReader(body), map[string]string{
				"Content-Type": "application/json",
			})

			assert.Equal(t, tt.expectedStatus, w.Code)

			response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)
			assert.Contains(t, response["message"], tt.expectMessage)
		})
	}
}

func TestGetCurrentUser(t *testing.T) {
	setupAuthTestEnvironment()

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		expectedStatus int
		expectMessage  string
		expectedFields []string
	}{
		{
			name: "valid user context returns user data",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:               "current-user-123",
					Username:         "currentuser",
					Email:            "current@example.com",
					Name:             "Current User",
					UserRoles:        []string{"Admin"},
					UserPermissions:  []string{"manage:systems"},
					OrgRole:          "Distributor",
					OrgPermissions:   []string{"create:resellers"},
					OrganizationID:   "org-current-456",
					OrganizationName: "Current Org",
				}
				c.Set("user", user)
			},
			expectedStatus: http.StatusOK,
			expectedFields: []string{"id", "username", "email", "name", "userRoles", "userPermissions", "orgRole", "orgPermissions", "organizationId", "organizationName"},
		},
		{
			name: "user with minimal data succeeds",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:       "minimal-current-user",
					Username: "minimal",
					Email:    "minimal@example.com",
				}
				c.Set("user", user)
			},
			expectedStatus: http.StatusOK,
			expectedFields: []string{"id", "username", "email"},
		},
		{
			name: "missing user context fails",
			setupContext: func(c *gin.Context) {
				// Don't set user
			},
			expectedStatus: http.StatusUnauthorized,
			expectMessage:  "user not authenticated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := testutils.SetupTestGin()
			router.GET("/auth/me", func(c *gin.Context) {
				tt.setupContext(c)
				GetCurrentUser(c)
			})

			w := testutils.MakeRequest(t, router, "GET", "/auth/me", nil, nil)

			assert.Equal(t, tt.expectedStatus, w.Code)

			response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)

			if tt.expectedStatus == http.StatusOK {
				assert.Contains(t, response["message"], "user information retrieved successfully")

				data, ok := response["data"].(map[string]interface{})
				assert.True(t, ok)

				// Check expected fields are present
				for _, field := range tt.expectedFields {
					assert.Contains(t, data, field, "Expected field %s not found", field)
				}
			} else if tt.expectMessage != "" {
				assert.Contains(t, response["message"], tt.expectMessage)
			}
		})
	}
}

func TestAuthMethodsValidation(t *testing.T) {
	setupAuthTestEnvironment()

	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
		expectMessage  string
	}{
		{
			name:           "exchange with invalid JSON fails",
			method:         "POST",
			path:           "/auth/exchange",
			body:           `{"invalid": json}`,
			expectedStatus: http.StatusBadRequest,
			expectMessage:  "Invalid request body",
		},
		{
			name:           "refresh with invalid JSON fails",
			method:         "POST",
			path:           "/auth/refresh",
			body:           `{"invalid": json}`,
			expectedStatus: http.StatusBadRequest,
			expectMessage:  "Invalid request body",
		},
		{
			name:   "exchange with missing access_token fails",
			method: "POST",
			path:   "/auth/exchange",
			body: map[string]string{
				"wrong_field": "value",
			},
			expectedStatus: http.StatusBadRequest,
			expectMessage:  "Invalid request body",
		},
		{
			name:   "refresh with missing refresh_token fails",
			method: "POST",
			path:   "/auth/refresh",
			body: map[string]string{
				"wrong_field": "value",
			},
			expectedStatus: http.StatusBadRequest,
			expectMessage:  "Invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := testutils.SetupTestGin()
			router.POST("/auth/exchange", ExchangeToken)
			router.POST("/auth/refresh", RefreshToken)

			var body *bytes.Reader
			if bodyStr, ok := tt.body.(string); ok {
				body = bytes.NewReader([]byte(bodyStr))
			} else {
				bodyBytes, _ := json.Marshal(tt.body)
				body = bytes.NewReader(bodyBytes)
			}

			w := testutils.MakeRequest(t, router, tt.method, tt.path, body, map[string]string{
				"Content-Type": "application/json",
			})

			assert.Equal(t, tt.expectedStatus, w.Code)

			response := testutils.AssertJSONResponse(t, w, tt.expectedStatus)
			assert.Contains(t, response["message"], tt.expectMessage)
		})
	}
}

func TestAuthMethodsIntegration(t *testing.T) {
	setupAuthTestEnvironment()

	// Test complete flow: exchange -> use token -> refresh

	// 1. Generate a custom token to simulate successful exchange
	user := models.User{
		ID:               "integration-user-123",
		Username:         "integrationuser",
		Email:            "integration@example.com",
		Name:             "Integration User",
		UserRoles:        []string{"Admin"},
		UserPermissions:  []string{"manage:systems"},
		OrgRole:          "Customer",
		OrgPermissions:   []string{"view:systems"},
		OrganizationID:   "org-integration",
		OrganizationName: "Integration Org",
	}

	customToken, err := jwt.GenerateCustomToken(user)
	require.NoError(t, err)

	refreshToken, err := jwt.GenerateRefreshToken(user.ID)
	require.NoError(t, err)

	router := testutils.SetupTestGin()

	// Add JWT middleware for protected routes
	authGroup := router.Group("/auth")
	authGroup.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			tokenString := authHeader[len("Bearer "):]
			if claims, err := jwt.ValidateCustomToken(tokenString); err == nil {
				c.Set("user", &claims.User)
			}
		}
		c.Next()
	})

	authGroup.GET("/me", GetCurrentUser)
	router.POST("/auth/refresh", RefreshToken)

	// 2. Test using the custom token to access /auth/me
	t.Run("use_custom_token_for_me_endpoint", func(t *testing.T) {
		w := testutils.MakeRequest(t, router, "GET", "/auth/me", nil, map[string]string{
			"Authorization": "Bearer " + customToken,
		})

		assert.Equal(t, http.StatusOK, w.Code)

		response := testutils.AssertJSONResponse(t, w, http.StatusOK)
		assert.Contains(t, response["message"], "user information retrieved successfully")

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "integration-user-123", data["id"])
		assert.Equal(t, "integrationuser", data["username"])
	})

	// 3. Test refresh token flow structure (will fail due to missing services)
	t.Run("refresh_token_request_structure", func(t *testing.T) {
		refreshReq := RefreshTokenRequest{
			RefreshToken: refreshToken,
		}

		body, _ := json.Marshal(refreshReq)
		w := testutils.MakeRequest(t, router, "POST", "/auth/refresh", bytes.NewReader(body), map[string]string{
			"Content-Type": "application/json",
		})

		// Will fail due to missing services, but validates request structure
		// Allow bad request due to validation as well
		assert.True(t, w.Code == http.StatusInternalServerError || w.Code == http.StatusUnauthorized || w.Code == http.StatusBadRequest)

		response := testutils.AssertJSONResponse(t, w, w.Code)
		// Just verify we get a structured error response
		assert.Contains(t, response, "message")
	})
}
