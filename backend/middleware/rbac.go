/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/logger"
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
			logger.RequestLogger(c, "rbac").Info().
				Str("operation", "permission_granted").
				Str("required_permission", permission).
				Str("user_id", user.ID).
				Str("username", user.Username).
				Str("organization_id", user.OrganizationID).
				Str("org_role", user.OrgRole).
				Strs("user_roles", user.UserRoles).
				Bool("via_user_permission", hasUserPermission).
				Bool("via_org_permission", hasOrgPermission).
				Msg("Permission granted")
			c.Next()
			return
		}

		// Permission denied
		logger.RequestLogger(c, "rbac").Warn().
			Str("operation", "permission_denied").
			Str("required_permission", permission).
			Str("user_id", user.ID).
			Str("username", user.Username).
			Str("organization_id", user.OrganizationID).
			Str("org_role", user.OrgRole).
			Strs("user_roles", user.UserRoles).
			Strs("user_permissions", user.UserPermissions).
			Strs("org_permissions", user.OrgPermissions).
			Str("client_ip", c.ClientIP()).
			Str("path", c.Request.URL.Path).
			Str("method", c.Request.Method).
			Msg("Permission denied - insufficient permissions")

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
			logger.RequestLogger(c, "rbac").Warn().
				Str("operation", "user_role_denied").
				Str("required_user_role", role).
				Str("user_id", user.ID).
				Str("username", user.Username).
				Str("organization_id", user.OrganizationID).
				Strs("user_roles", user.UserRoles).
				Str("client_ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Msg("User role denied - insufficient user role")

			c.JSON(http.StatusForbidden, response.Forbidden("insufficient user role", gin.H{
				"required_user_role": role,
				"user_roles":         user.UserRoles,
			}))
			c.Abort()
			return
		}

		logger.RequestLogger(c, "rbac").Info().
			Str("operation", "user_role_granted").
			Str("required_user_role", role).
			Str("user_id", user.ID).
			Str("username", user.Username).
			Strs("user_roles", user.UserRoles).
			Msg("User role granted")

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
			logger.RequestLogger(c, "rbac").Warn().
				Str("operation", "org_role_denied").
				Str("required_org_role", role).
				Str("user_org_role", user.OrgRole).
				Str("user_id", user.ID).
				Str("username", user.Username).
				Str("organization_id", user.OrganizationID).
				Str("organization", user.OrganizationName).
				Str("client_ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Msg("Organization role denied - insufficient organization role")

			c.JSON(http.StatusForbidden, response.Forbidden("insufficient organization role", gin.H{
				"required_org_role": role,
				"user_org_role":     user.OrgRole,
				"organization":      user.OrganizationName,
			}))
			c.Abort()
			return
		}

		logger.RequestLogger(c, "rbac").Info().
			Str("operation", "org_role_granted").
			Str("required_org_role", role).
			Str("user_org_role", user.OrgRole).
			Str("user_id", user.ID).
			Str("username", user.Username).
			Str("organization_id", user.OrganizationID).
			Str("organization", user.OrganizationName).
			Msg("Organization role granted")

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
				logger.RequestLogger(c, "rbac").Info().
					Str("operation", "any_org_role_granted").
					Strs("required_org_roles", roles).
					Str("matched_org_role", role).
					Str("user_org_role", user.OrgRole).
					Str("user_id", user.ID).
					Str("username", user.Username).
					Str("organization_id", user.OrganizationID).
					Str("organization", user.OrganizationName).
					Msg("Organization role granted (any)")
				c.Next()
				return
			}
		}

		logger.RequestLogger(c, "rbac").Warn().
			Str("operation", "any_org_role_denied").
			Strs("required_org_roles", roles).
			Str("user_org_role", user.OrgRole).
			Str("user_id", user.ID).
			Str("username", user.Username).
			Str("organization_id", user.OrganizationID).
			Str("organization", user.OrganizationName).
			Str("client_ip", c.ClientIP()).
			Str("path", c.Request.URL.Path).
			Str("method", c.Request.Method).
			Msg("Organization role denied - insufficient organization role (any)")

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
