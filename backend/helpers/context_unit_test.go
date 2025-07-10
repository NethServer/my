package helpers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
)

func setupHelpersTestEnvironment() {
	gin.SetMode(gin.TestMode)
}

func TestGetUserFromContext(t *testing.T) {
	setupHelpersTestEnvironment()

	tests := []struct {
		name         string
		setupContext func(*gin.Context)
		expectUser   bool
		expectAbort  bool
		expectedCode int
		expectedUser *models.User
	}{
		{
			name: "valid user in context returns user",
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
					OrganizationName: "Test Organization",
				}
				c.Set("user", user)
			},
			expectUser:   true,
			expectAbort:  false,
			expectedCode: 0, // No HTTP response when successful
			expectedUser: &models.User{
				ID:               "test-user-123",
				Username:         "testuser",
				Email:            "test@example.com",
				Name:             "Test User",
				UserRoles:        []string{"Admin"},
				UserPermissions:  []string{"manage:systems"},
				OrgRole:          "Customer",
				OrgPermissions:   []string{"view:systems"},
				OrganizationID:   "org-123",
				OrganizationName: "Test Organization",
			},
		},
		{
			name: "missing user in context returns error",
			setupContext: func(c *gin.Context) {
				// Don't set user
			},
			expectUser:   false,
			expectAbort:  true,
			expectedCode: http.StatusUnauthorized,
			expectedUser: nil,
		},
		{
			name: "invalid user type in context returns error",
			setupContext: func(c *gin.Context) {
				c.Set("user", "not-a-user-object")
			},
			expectUser:   false,
			expectAbort:  true,
			expectedCode: http.StatusInternalServerError,
			expectedUser: nil,
		},
		{
			name: "nil user in context returns error",
			setupContext: func(c *gin.Context) {
				var nilUser *models.User
				c.Set("user", nilUser)
			},
			expectUser:   true, // Nil pointer is still a valid *models.User type
			expectAbort:  false,
			expectedCode: 0,
			expectedUser: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			tt.setupContext(c)

			user, ok := GetUserFromContext(c)

			assert.Equal(t, tt.expectUser, ok)

			if tt.expectUser {
				if tt.expectedUser != nil {
					assert.NotNil(t, user)
					assert.Equal(t, tt.expectedUser.ID, user.ID)
					assert.Equal(t, tt.expectedUser.Username, user.Username)
					assert.Equal(t, tt.expectedUser.Email, user.Email)
					assert.Equal(t, tt.expectedUser.Name, user.Name)
					assert.Equal(t, tt.expectedUser.UserRoles, user.UserRoles)
					assert.Equal(t, tt.expectedUser.UserPermissions, user.UserPermissions)
					assert.Equal(t, tt.expectedUser.OrgRole, user.OrgRole)
					assert.Equal(t, tt.expectedUser.OrgPermissions, user.OrgPermissions)
					assert.Equal(t, tt.expectedUser.OrganizationID, user.OrganizationID)
					assert.Equal(t, tt.expectedUser.OrganizationName, user.OrganizationName)
				} else {
					// expectedUser is nil, so user should also be nil
					assert.Nil(t, user)
				}
				assert.False(t, c.IsAborted())
			} else {
				assert.Nil(t, user)
				if tt.expectAbort {
					assert.True(t, c.IsAborted())
					assert.Equal(t, tt.expectedCode, w.Code)
				}
			}
		})
	}
}

func TestGetUserFromContextEdgeCases(t *testing.T) {
	setupHelpersTestEnvironment()

	tests := []struct {
		name         string
		setupContext func(*gin.Context)
		expectUser   bool
		description  string
	}{
		{
			name: "user with empty fields succeeds",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:               "",
					Username:         "",
					Email:            "",
					Name:             "",
					UserRoles:        []string{},
					UserPermissions:  []string{},
					OrgRole:          "",
					OrgPermissions:   []string{},
					OrganizationID:   "",
					OrganizationName: "",
				}
				c.Set("user", user)
			},
			expectUser:  true,
			description: "User with all empty fields should still be valid",
		},
		{
			name: "user with nil slices succeeds",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:              "nil-slices-user",
					UserRoles:       nil,
					UserPermissions: nil,
					OrgPermissions:  nil,
				}
				c.Set("user", user)
			},
			expectUser:  true,
			description: "User with nil slices should be valid",
		},
		{
			name: "interface{} pointing to valid user succeeds",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:       "interface-user",
					Username: "interface",
				}
				var userInterface interface{} = user
				c.Set("user", userInterface)
			},
			expectUser:  true,
			description: "interface{} containing *models.User should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			tt.setupContext(c)

			user, ok := GetUserFromContext(c)

			assert.Equal(t, tt.expectUser, ok, tt.description)
			if tt.expectUser {
				assert.NotNil(t, user)
				assert.IsType(t, &models.User{}, user)
			}
		})
	}
}

func TestGetUserFromContextIntegration(t *testing.T) {
	setupHelpersTestEnvironment()

	// Test GetUserFromContext with a complete user
	user := &models.User{
		ID:               "integration-test-user",
		Username:         "integrationuser",
		Email:            "integration@test.com",
		Name:             "Integration Test User",
		UserRoles:        []string{"Admin", "Support", "Manager"},
		UserPermissions:  []string{"manage:systems", "view:logs", "admin:accounts"},
		OrgRole:          "Owner",
		OrgPermissions:   []string{"create:distributors", "manage:all", "admin:system"},
		OrganizationID:   "org-integration-test",
		OrganizationName: "Integration Test Organization",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set("user", user)

	// Test GetUserFromContext
	retrievedUser, ok := GetUserFromContext(c)
	assert.True(t, ok)
	assert.Equal(t, user, retrievedUser)
}
