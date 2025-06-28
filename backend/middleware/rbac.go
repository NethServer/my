/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
)

// RequirePermission checks if user has a specific permission from either:
// 1. User roles (technical capabilities)
// 2. Organization role (business hierarchy)
// This is the main authorization middleware for the simplified RBAC system
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		// Check if user has permission via User Roles (technical capabilities)
		hasUserPermission := hasPermissionInList(user.UserPermissions, permission)

		// Check if user has permission via Organization Role (business hierarchy)
		hasOrgPermission := hasPermissionInList(user.OrgPermissions, permission)

		if hasUserPermission || hasOrgPermission {
			c.Next()
			return
		}

		// Permission denied
		c.JSON(http.StatusForbidden, response.Forbidden("insufficient permissions", gin.H{
			"required_permission": permission,
			"user_permissions":    user.UserPermissions,
			"org_permissions":     user.OrgPermissions,
			"user_roles":          user.UserRoles,
			"org_role":            user.OrgRole,
			"organization":        user.OrganizationName,
		}))
		c.Abort()
	}
}

// RequireUserRole checks if user has a specific technical capability role
// Use this when you need to ensure user has specific technical skills
func RequireUserRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		if !hasRoleInList(user.UserRoles, role) {
			c.JSON(http.StatusForbidden, response.Forbidden("insufficient user role", gin.H{
				"required_user_role": role,
				"user_roles":         user.UserRoles,
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireOrgRole checks if user's organization has a specific business hierarchy role
// Use this when you need to ensure user belongs to organization with specific business level
func RequireOrgRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		if user.OrgRole != role {
			c.JSON(http.StatusForbidden, response.Forbidden("insufficient organization role", gin.H{
				"required_org_role": role,
				"user_org_role":     user.OrgRole,
				"organization":      user.OrganizationName,
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyOrgRole checks if user's organization has any of the specified business hierarchy roles
// Useful for hierarchical access where multiple levels can access a resource
func RequireAnyOrgRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		for _, role := range roles {
			if user.OrgRole == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, response.Forbidden("insufficient organization role", gin.H{
			"required_org_roles": roles,
			"user_org_role":      user.OrgRole,
			"organization":       user.OrganizationName,
		}))
		c.Abort()
	}
}

// Helper functions

// getUserFromContext extracts the user from the Gin context and handles common error cases
func getUserFromContext(c *gin.Context) (*models.User, bool) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user not found in context", nil))
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

func hasPermissionInList(permissions []string, permission string) bool {
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

func hasRoleInList(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

func buildPermissionFromMethod(method, resource string) string {
	methodToAction := map[string]string{
		"GET":    "read",
		"POST":   "create",
		"PUT":    "manage",
		"PATCH":  "manage",
		"DELETE": "manage",
	}

	action, exists := methodToAction[method]
	if !exists {
		return ""
	}

	return fmt.Sprintf("%s:%s", action, resource)
}
