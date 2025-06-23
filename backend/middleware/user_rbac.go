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

// RequireRole checks if user has a specific user role (Support, Admin, Sales, etc.)
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		if !hasUserRole(user, role) {
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient role permissions",
				Data: gin.H{
					"required_role": role,
					"user_roles":    user.Roles,
				},
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireScope checks if user has a specific scope/permission
func RequireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := getUserFromContext(c)
		if !ok {
			return
		}

		if !hasUserScope(user, scope) {
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient scope permissions",
				Data: gin.H{
					"required_scope": scope,
					"user_scopes":    user.Scopes,
				},
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

// AutoRoleRBAC checks if user has the required role for accessing a resource
// This is a convenience middleware for simple role-based access
func AutoRoleRBAC(requiredRole string) gin.HandlerFunc {
	return RequireRole(requiredRole)
}

// AutoRBAC automatically determines required scope based on HTTP method and resource
// GET -> read:resource, POST -> create:resource, PUT/PATCH -> update:resource, DELETE -> delete:resource
func AutoRBAC(resource string) gin.HandlerFunc {
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

		if !hasUserScope(user, requiredScope) {
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient scope permissions",
				Data: gin.H{
					"required_scope": requiredScope,
					"user_scopes":    user.Scopes,
				},
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

// Helper functions for user RBAC

func hasUserRole(user *models.User, role string) bool {
	for _, userRole := range user.Roles {
		if userRole == role {
			return true
		}
	}
	return false
}

func hasUserScope(user *models.User, scope string) bool {
	for _, userScope := range user.Scopes {
		if userScope == scope {
			return true
		}
	}
	return false
}

func buildScopeFromMethod(method, resource string) string {
	methodToAction := map[string]string{
		"GET":    "read",
		"POST":   "create",
		"PUT":    "update",
		"PATCH":  "update",
		"DELETE": "delete",
	}

	action, exists := methodToAction[method]
	if !exists {
		return ""
	}

	return fmt.Sprintf("%s:%s", action, resource)
}