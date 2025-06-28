/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
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

// GetUserContextData extracts individual context fields (legacy compatibility)
type UserContextData struct {
	UserID           string
	UserOrgRole      string
	UserOrgID        string
	UserRole         []string
	UserPermissions  []string
	OrgPermissions   []string
	OrganizationName string
}

// GetUserContextData extracts user context as individual fields
func GetUserContextData(c *gin.Context) (*UserContextData, bool) {
	user, ok := GetUserFromContext(c)
	if !ok {
		return nil, false
	}

	return &UserContextData{
		UserID:           user.ID,
		UserOrgRole:      user.OrgRole,
		UserOrgID:        user.OrganizationID,
		UserRole:         user.UserRoles,
		UserPermissions:  user.UserPermissions,
		OrgPermissions:   user.OrgPermissions,
		OrganizationName: user.OrganizationName,
	}, true
}
