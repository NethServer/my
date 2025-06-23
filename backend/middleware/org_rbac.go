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

	"github.com/fatih/structs"
	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
)

// RequireOrganizationRole checks if user has a specific organization role (God, Distributor, Reseller, Customer)
func RequireOrganizationRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		if !hasOrganizationRole(user, role) {
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient organization role",
				Data: gin.H{
					"required_organization_role": role,
					"user_organization_roles":    user.OrganizationRoles,
				},
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyOrganizationRole checks if user has at least one of the specified organization roles
// Useful for hierarchical access where multiple levels can access a resource
func RequireAnyOrganizationRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		if !hasAnyOrganizationRole(user, roles) {
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient organization role",
				Data: gin.H{
					"required_organization_roles": roles,
					"user_organization_roles":     user.OrganizationRoles,
				},
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireOrganizationScope checks if user has a specific organization scope/permission
func RequireOrganizationScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		if !hasOrganizationScope(user, scope) {
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient organization scope",
				Data: gin.H{
					"required_organization_scope": scope,
					"user_organization_scopes":    user.OrganizationScopes,
				},
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

// AutoOrganizationRoleRBAC checks if user has the required organization role for accessing a resource
// This is a convenience middleware for simple organization role-based access
func AutoOrganizationRoleRBAC(requiredRole string) gin.HandlerFunc {
	return RequireOrganizationRole(requiredRole)
}

// AutoOrganizationRBAC automatically determines required organization scope based on HTTP method and resource
// Similar to AutoRBAC but uses organization scopes instead of user scopes
func AutoOrganizationRBAC(resource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		requiredScope := buildScopeFromMethod(c.Request.Method, resource)
		if requiredScope == "" {
			c.JSON(http.StatusMethodNotAllowed, structs.Map(response.StatusNotFound{
				Code:    405,
				Message: "method not allowed",
				Data:    nil,
			}))
			c.Abort()
			return
		}

		if !hasOrganizationScope(user, requiredScope) {
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient organization scope",
				Data: gin.H{
					"required_organization_scope": requiredScope,
					"user_organization_scopes":    user.OrganizationScopes,
					"user_organization_roles":     user.OrganizationRoles,
				},
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

// AutoOrganizationRBACWithManage uses "manage" scope for update/delete operations instead of separate update/delete
// This follows the pattern where some resources use "manage:resource" for all modification operations
func AutoOrganizationRBACWithManage(resource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		var requiredScope string
		switch c.Request.Method {
		case "GET":
			requiredScope = fmt.Sprintf("read:%s", resource)
		case "POST":
			requiredScope = fmt.Sprintf("create:%s", resource)
		case "PUT", "PATCH", "DELETE":
			requiredScope = fmt.Sprintf("manage:%s", resource)
		default:
			c.JSON(http.StatusMethodNotAllowed, structs.Map(response.StatusNotFound{
				Code:    405,
				Message: "method not allowed",
				Data:    nil,
			}))
			c.Abort()
			return
		}

		if !hasOrganizationScope(user, requiredScope) {
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient organization scope",
				Data: gin.H{
					"required_organization_scope": requiredScope,
					"user_organization_scopes":    user.OrganizationScopes,
					"user_organization_roles":     user.OrganizationRoles,
				},
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

// Helper functions for organization RBAC

func hasOrganizationRole(user *models.User, role string) bool {
	for _, orgRole := range user.OrganizationRoles {
		if orgRole == role {
			return true
		}
	}
	return false
}

func hasAnyOrganizationRole(user *models.User, roles []string) bool {
	for _, userRole := range user.OrganizationRoles {
		for _, requiredRole := range roles {
			if userRole == requiredRole {
				return true
			}
		}
	}
	return false
}

func hasOrganizationScope(user *models.User, scope string) bool {
	for _, orgScope := range user.OrganizationScopes {
		if orgScope == scope {
			return true
		}
	}
	return false
}