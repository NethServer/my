/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package helpers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/models"
)

func TestGetUserFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logtoID := "logto123"

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		expectedUser   *models.User
		expectedOK     bool
		expectedStatus int
		checkResponse  bool
	}{
		{
			name: "valid user in context",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:             "user123",
					LogtoID:        &logtoID,
					Name:           "Test User",
					Email:          "test@example.com",
					OrganizationID: "org123",
					OrgRole:        "owner",
					UserRoles:      []string{"admin"},
				}
				c.Set("user", user)
			},
			expectedUser: &models.User{
				ID:             "user123",
				LogtoID:        &logtoID,
				Name:           "Test User",
				Email:          "test@example.com",
				OrganizationID: "org123",
				OrgRole:        "owner",
				UserRoles:      []string{"admin"},
			},
			expectedOK:     true,
			expectedStatus: 0, // No HTTP response
			checkResponse:  false,
		},
		{
			name: "user with empty ID falls back to LogtoID",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:             "", // Empty ID
					LogtoID:        &logtoID,
					Name:           "Test User",
					Email:          "test@example.com",
					OrganizationID: "org123",
					OrgRole:        "owner",
				}
				c.Set("user", user)
			},
			expectedUser: &models.User{
				ID:             "logto123", // Should be set to LogtoID
				LogtoID:        &logtoID,
				Name:           "Test User",
				Email:          "test@example.com",
				OrganizationID: "org123",
				OrgRole:        "owner",
			},
			expectedOK:     true,
			expectedStatus: 0, // No HTTP response
			checkResponse:  false,
		},
		{
			name: "no user in context",
			setupContext: func(c *gin.Context) {
				// Don't set any user
			},
			expectedUser:   nil,
			expectedOK:     false,
			expectedStatus: http.StatusUnauthorized,
			checkResponse:  true,
		},
		{
			name: "invalid user type in context",
			setupContext: func(c *gin.Context) {
				c.Set("user", "invalid_user_string") // Wrong type
			},
			expectedUser:   nil,
			expectedOK:     false,
			expectedStatus: http.StatusInternalServerError,
			checkResponse:  true,
		},
		{
			name: "nil user in context",
			setupContext: func(c *gin.Context) {
				var nilUser *models.User = nil
				c.Set("user", nilUser)
			},
			expectedUser:   nil,
			expectedOK:     false,
			expectedStatus: http.StatusInternalServerError,
			checkResponse:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Setup request
			req := httptest.NewRequest("GET", "/test", nil)
			c.Request = req

			// Setup context
			tt.setupContext(c)

			// Test the function
			user, ok := GetUserFromContext(c)

			// Check results
			assert.Equal(t, tt.expectedOK, ok, "OK status should match expected")

			if tt.expectedOK {
				assert.NotNil(t, user, "User should not be nil when OK is true")
				assert.Equal(t, tt.expectedUser.ID, user.ID)
				assert.Equal(t, tt.expectedUser.Name, user.Name)
				assert.Equal(t, tt.expectedUser.Email, user.Email)
				assert.Equal(t, tt.expectedUser.OrganizationID, user.OrganizationID)
				assert.Equal(t, tt.expectedUser.OrgRole, user.OrgRole)
			} else {
				assert.Nil(t, user, "User should be nil when OK is false")
			}

			if tt.checkResponse {
				assert.Equal(t, tt.expectedStatus, w.Code, "HTTP status should match expected")

				// Check that response is JSON
				var responseBody map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &responseBody)
				if err == nil && responseBody != nil {
					// Check that the response has the expected structure
					assert.Contains(t, responseBody, "message", "Response should contain message")

					// Safe check for success field
					if success, exists := responseBody["success"]; exists && success != nil {
						assert.False(t, success.(bool), "Response should indicate failure")
					}
				}
			}
		})
	}
}

func TestGetUserContextExtended(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logtoID := "logto456"

	tests := []struct {
		name             string
		setupContext     func(*gin.Context)
		expectedUserID   string
		expectedOrgID    string
		expectedOrgRole  string
		expectedUserRole string
	}{
		{
			name: "complete user context",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:             "user123",
					LogtoID:        &logtoID,
					Name:           "Test User",
					Email:          "test@example.com",
					OrganizationID: "org123",
					OrgRole:        "owner",
					UserRoles:      []string{"admin", "support"},
				}
				c.Set("user", user)
			},
			expectedUserID:   "user123",
			expectedOrgID:    "org123",
			expectedOrgRole:  "owner",
			expectedUserRole: "admin", // First role in the slice
		},
		{
			name: "user with empty ID falls back to LogtoID",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:             "", // Empty ID
					LogtoID:        &logtoID,
					Name:           "Test User",
					OrganizationID: "org456",
					OrgRole:        "distributor",
					UserRoles:      []string{"support"},
				}
				c.Set("user", user)
			},
			expectedUserID:   "logto456", // Should use LogtoID
			expectedOrgID:    "org456",
			expectedOrgRole:  "distributor",
			expectedUserRole: "support",
		},
		{
			name: "user with no user roles",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:             "user789",
					Name:           "Test User",
					OrganizationID: "org789",
					OrgRole:        "customer",
					UserRoles:      []string{}, // Empty roles
				}
				c.Set("user", user)
			},
			expectedUserID:   "user789",
			expectedOrgID:    "org789",
			expectedOrgRole:  "customer",
			expectedUserRole: "", // Empty when no roles
		},
		{
			name: "user with nil user roles",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:             "user999",
					Name:           "Test User",
					OrganizationID: "org999",
					OrgRole:        "reseller",
					UserRoles:      nil, // Nil roles
				}
				c.Set("user", user)
			},
			expectedUserID:   "user999",
			expectedOrgID:    "org999",
			expectedOrgRole:  "reseller",
			expectedUserRole: "", // Empty when roles is nil
		},
		{
			name: "no user in context",
			setupContext: func(c *gin.Context) {
				// Don't set any user
			},
			expectedUserID:   "",
			expectedOrgID:    "",
			expectedOrgRole:  "",
			expectedUserRole: "",
		},
		{
			name: "invalid user type in context",
			setupContext: func(c *gin.Context) {
				c.Set("user", "invalid_user_string") // Wrong type
			},
			expectedUserID:   "",
			expectedOrgID:    "",
			expectedOrgRole:  "",
			expectedUserRole: "",
		},
		{
			name: "user with only LogtoID (no local ID)",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					// ID is empty/default
					LogtoID:        &logtoID,
					Name:           "Logto User",
					OrganizationID: "logto-org",
					OrgRole:        "customer",
					UserRoles:      []string{"viewer"},
				}
				c.Set("user", user)
			},
			expectedUserID:   "logto456", // Should use LogtoID
			expectedOrgID:    "logto-org",
			expectedOrgRole:  "customer",
			expectedUserRole: "viewer",
		},
		{
			name: "user with empty strings",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:             "",
					LogtoID:        nil, // No LogtoID either
					Name:           "",
					OrganizationID: "",
					OrgRole:        "",
					UserRoles:      []string{},
				}
				c.Set("user", user)
			},
			expectedUserID:   "", // Both ID and LogtoID are empty
			expectedOrgID:    "",
			expectedOrgRole:  "",
			expectedUserRole: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Setup request
			req := httptest.NewRequest("GET", "/test", nil)
			c.Request = req

			// Setup context
			tt.setupContext(c)

			// Test the function
			userID, orgID, orgRole, userRole := GetUserContextExtended(c)

			// Check results
			assert.Equal(t, tt.expectedUserID, userID, "User ID should match expected")
			assert.Equal(t, tt.expectedOrgID, orgID, "Organization ID should match expected")
			assert.Equal(t, tt.expectedOrgRole, orgRole, "Organization role should match expected")
			assert.Equal(t, tt.expectedUserRole, userRole, "User role should match expected")
		})
	}
}

func TestContextHelpers_AbortBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		setupContext func(*gin.Context)
		expectAbort  bool
	}{
		{
			name: "valid context should not abort",
			setupContext: func(c *gin.Context) {
				user := &models.User{
					ID:             "user123",
					Name:           "Test User",
					OrganizationID: "org123",
					OrgRole:        "owner",
				}
				c.Set("user", user)
			},
			expectAbort: false,
		},
		{
			name: "missing user should abort",
			setupContext: func(c *gin.Context) {
				// Don't set user
			},
			expectAbort: true,
		},
		{
			name: "invalid user type should abort",
			setupContext: func(c *gin.Context) {
				c.Set("user", "invalid")
			},
			expectAbort: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req := httptest.NewRequest("GET", "/test", nil)
			c.Request = req

			tt.setupContext(c)

			// Test GetUserFromContext (which can abort)
			_, ok := GetUserFromContext(c)

			if tt.expectAbort {
				assert.False(t, ok, "Should return false when aborting")
				assert.True(t, c.IsAborted(), "Context should be aborted")
			} else {
				assert.True(t, ok, "Should return true when not aborting")
				assert.False(t, c.IsAborted(), "Context should not be aborted")
			}
		})
	}
}

func TestContextHelpers_EdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("user with multiple roles returns first", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req

		user := &models.User{
			ID:        "user123",
			UserRoles: []string{"admin", "support", "viewer"}, // Multiple roles
		}
		c.Set("user", user)

		_, _, _, userRole := GetUserContextExtended(c)
		assert.Equal(t, "admin", userRole, "Should return first role")
	})

	t.Run("empty LogtoID pointer", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req

		emptyLogtoID := ""
		user := &models.User{
			ID:      "",            // Empty ID
			LogtoID: &emptyLogtoID, // Empty LogtoID
		}
		c.Set("user", user)

		userID, _, _, _ := GetUserContextExtended(c)
		assert.Equal(t, "", userID, "Should be empty when both ID and LogtoID are empty")
	})
}
