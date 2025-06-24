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
		c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
			Code:    403,
			Message: "insufficient permissions",
			Data: gin.H{
				"required_permission":    permission,
				"user_permissions":       user.UserPermissions,
				"org_permissions":        user.OrgPermissions,
				"user_roles":            user.UserRoles,
				"org_role":              user.OrgRole,
				"organization":          user.OrganizationName,
			},
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
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient user role",
				Data: gin.H{
					"required_user_role": role,
					"user_roles":        user.UserRoles,
				},
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
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient organization role",
				Data: gin.H{
					"required_org_role": role,
					"user_org_role":     user.OrgRole,
					"organization":      user.OrganizationName,
				},
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

		c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
			Code:    403,
			Message: "insufficient organization role",
			Data: gin.H{
				"required_org_roles": roles,
				"user_org_role":      user.OrgRole,
				"organization":       user.OrganizationName,
			},
		}))
		c.Abort()
	}
}

// AutoPermissionRBAC automatically determines required permission based on HTTP method and resource
// GET -> read:resource, POST -> create:resource, PUT/PATCH -> manage:resource, DELETE -> manage:resource
func AutoPermissionRBAC(resource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		requiredPermission := buildPermissionFromMethod(c.Request.Method, resource)
		if requiredPermission == "" {
			c.JSON(http.StatusMethodNotAllowed, structs.Map(response.StatusNotFound{
				Code:    405,
				Message: "method not allowed",
				Data:    nil,
			}))
			c.Abort()
			return
		}

		// Check if user has permission via User Roles or Organization Role
		hasUserPermission := hasPermissionInList(user.UserPermissions, requiredPermission)
		hasOrgPermission := hasPermissionInList(user.OrgPermissions, requiredPermission)

		if !hasUserPermission && !hasOrgPermission {
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient permissions",
				Data: gin.H{
					"required_permission": requiredPermission,
					"user_permissions":    user.UserPermissions,
					"org_permissions":     user.OrgPermissions,
					"user_roles":         user.UserRoles,
					"org_role":           user.OrgRole,
				},
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

// Helper functions

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

// Convenience aliases for backward compatibility and common patterns

// AutoRoleRBAC is an alias for RequireUserRole for backward compatibility
func AutoRoleRBAC(requiredRole string) gin.HandlerFunc {
	return RequireUserRole(requiredRole)
}

// RequireRole is an alias for RequireUserRole for backward compatibility
func RequireRole(role string) gin.HandlerFunc {
	return RequireUserRole(role)
}

// RequireScope is an alias for RequirePermission for backward compatibility
func RequireScope(permission string) gin.HandlerFunc {
	return RequirePermission(permission)
}

// AutoRBAC is an alias for AutoPermissionRBAC for backward compatibility
func AutoRBAC(resource string) gin.HandlerFunc {
	return AutoPermissionRBAC(resource)
}

// Legacy organization middleware aliases - now use simplified org role checks
func AutoOrganizationRoleRBAC(requiredRole string) gin.HandlerFunc {
	return RequireOrgRole(requiredRole)
}

func RequireOrganizationRole(role string) gin.HandlerFunc {
	return RequireOrgRole(role)
}

func RequireAnyOrganizationRole(roles ...string) gin.HandlerFunc {
	return RequireAnyOrgRole(roles...)
}

func RequireOrganizationScope(permission string) gin.HandlerFunc {
	return RequirePermission(permission)
}

// Legacy combination functions
func RequireRoleOrScope(role, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		hasRole := hasRoleInList(user.UserRoles, role)
		hasPermission := hasPermissionInList(user.UserPermissions, permission) || 
		                hasPermissionInList(user.OrgPermissions, permission)

		if !hasRole && !hasPermission {
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient permissions",
				Data: gin.H{
					"required_role":       role,
					"required_permission": permission,
					"user_roles":         user.UserRoles,
					"user_permissions":   user.UserPermissions,
					"org_permissions":    user.OrgPermissions,
				},
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

func RequireOrganizationRoleOrScope(role, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		hasRole := user.OrgRole == role
		hasPermission := hasPermissionInList(user.UserPermissions, permission) || 
		                hasPermissionInList(user.OrgPermissions, permission)

		if !hasRole && !hasPermission {
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient organization permissions",
				Data: gin.H{
					"required_org_role":   role,
					"required_permission": permission,
					"user_org_role":      user.OrgRole,
					"user_permissions":   user.UserPermissions,
					"org_permissions":    user.OrgPermissions,
				},
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}