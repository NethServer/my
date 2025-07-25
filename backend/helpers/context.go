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
	if !ok {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("invalid user context", nil))
		c.Abort()
		return nil, false
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

	userID := user.ID
	userOrgID := user.OrganizationID
	userOrgRole := user.OrgRole
	userRole := ""

	// Extract user role (highest privilege role)
	if len(user.UserRoles) > 0 {
		userRole = user.UserRoles[0]
	}

	return userID, userOrgID, userOrgRole, userRole
}
