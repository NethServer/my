/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/response"
)

// GetUserPermissions returns user permissions and role information
func GetUserPermissions(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	permissionsData := gin.H{
		"user_roles":        user.UserRoles,
		"user_permissions":  user.UserPermissions,
		"org_role":          user.OrgRole,
		"org_permissions":   user.OrgPermissions,
		"organization_id":   user.OrganizationID,
		"organization_name": user.OrganizationName,
	}

	c.JSON(http.StatusOK, response.OK("user permissions retrieved successfully", permissionsData))
}

// GetUserProfile returns complete user profile with business context
func GetUserProfile(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, response.OK("user profile retrieved successfully", user))
}
