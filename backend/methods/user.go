/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"net/http"

	"github.com/fatih/structs"
	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/response"
)

func GetProfile(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "user profile retrieved successfully",
		Data:    user,
	}))
}

func GetProtectedResource(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, structs.Map(response.StatusUnauthorized{
			Code:    401,
			Message: "user not authenticated",
			Data:    nil,
		}))
		return
	}

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "protected resource accessed successfully",
		Data:    gin.H{"user_id": userID, "resource": "sensitive data"},
	}))
}

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

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "user permissions retrieved successfully",
		Data:    permissionsData,
	}))
}

// GetUserProfile returns complete user profile with business context
func GetUserProfile(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "user profile retrieved successfully",
		Data:    user,
	}))
}
