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
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
)

// GetUserFromContext extracts user context data from Gin context
// Returns user object and boolean indicating success
func GetUserFromContext(c *gin.Context) (*models.User, bool) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user not authenticated", nil))
		c.Abort()
		return nil, false
	}

	user, ok := userInterface.(*models.User)
	if !ok || user == nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("invalid user context", nil))
		c.Abort()
		return nil, false
	}

	// Apply the same fallback logic as GetUserContextExtended
	// Use LogtoID if local ID is empty (for users not yet synced to local database)
	if user != nil && user.ID == "" && user.LogtoID != nil {
		user.ID = *user.LogtoID
	}

	return user, true
}

// GetUserContextExtended extracts user ID, organization ID, and role information from Gin context
// Returns userID, userOrgID, userOrgRole, userRole strings
func GetUserContextExtended(c *gin.Context) (string, string, string, string) {
	userInterface, exists := c.Get("user")
	if !exists {
		return "", "", "", ""
	}

	user, ok := userInterface.(*models.User)
	if !ok {
		return "", "", "", ""
	}

	// Use local ID if available, otherwise fall back to LogtoID for user identification
	userID := user.ID
	if userID == "" && user.LogtoID != nil {
		userID = *user.LogtoID
	}

	userOrgID := user.OrganizationID
	userOrgRole := user.OrgRole
	userRole := ""

	// Extract user role (highest privilege role)
	if len(user.UserRoles) > 0 {
		userRole = user.UserRoles[0]
	}

	return userID, userOrgID, userOrgRole, userRole
}
