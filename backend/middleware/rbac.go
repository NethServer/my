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
	"net/http"

	"github.com/fatih/structs"
	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
)

// Common helper functions used by both user and organization RBAC

// getUserFromContext extracts the user from the Gin context and handles common error cases
func getUserFromContext(c *gin.Context) (*models.User, bool) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, structs.Map(response.StatusUnauthorized{
			Code:    401,
			Message: "user not found in context",
			Data:    nil,
		}))
		c.Abort()
		return nil, false
	}

	user, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "invalid user context",
			Data:    nil,
		}))
		c.Abort()
		return nil, false
	}

	return user, true
}

// Convenience functions that combine multiple authorization checks

// RequireRoleOrScope checks if user has either a specific role OR a specific scope
// Useful for endpoints that can be accessed by role-based OR permission-based authorization
func RequireRoleOrScope(role, scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		hasRole := hasUserRole(user, role)
		hasScope := hasUserScope(user, scope)

		if !hasRole && !hasScope {
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient permissions",
				Data: gin.H{
					"required_role":  role,
					"required_scope": scope,
					"user_roles":     user.Roles,
					"user_scopes":    user.Scopes,
				},
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireOrganizationRoleOrScope checks if user has either an organization role OR an organization scope
func RequireOrganizationRoleOrScope(role, scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		hasRole := hasOrganizationRole(user, role)
		hasScope := hasOrganizationScope(user, scope)

		if !hasRole && !hasScope {
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient organization permissions",
				Data: gin.H{
					"required_organization_role":  role,
					"required_organization_scope": scope,
					"user_organization_roles":     user.OrganizationRoles,
					"user_organization_scopes":    user.OrganizationScopes,
				},
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission checks if user has ANY of the specified permissions (roles, scopes, org roles, org scopes)
// This is the most flexible authorization middleware for complex scenarios
func RequireAnyPermission(roles []string, scopes []string, orgRoles []string, orgScopes []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		// Check user roles
		for _, role := range roles {
			if hasUserRole(user, role) {
				c.Next()
				return
			}
		}

		// Check user scopes
		for _, scope := range scopes {
			if hasUserScope(user, scope) {
				c.Next()
				return
			}
		}

		// Check organization roles
		for _, orgRole := range orgRoles {
			if hasOrganizationRole(user, orgRole) {
				c.Next()
				return
			}
		}

		// Check organization scopes
		for _, orgScope := range orgScopes {
			if hasOrganizationScope(user, orgScope) {
				c.Next()
				return
			}
		}

		// No permissions found
		c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
			Code:    403,
			Message: "insufficient permissions",
			Data: gin.H{
				"required_roles":               roles,
				"required_scopes":              scopes,
				"required_organization_roles":  orgRoles,
				"required_organization_scopes": orgScopes,
				"user_roles":                   user.Roles,
				"user_scopes":                  user.Scopes,
				"user_organization_roles":      user.OrganizationRoles,
				"user_organization_scopes":     user.OrganizationScopes,
			},
		}))
		c.Abort()
	}
}
