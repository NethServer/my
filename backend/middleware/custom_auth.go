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
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/jwt"
	"github.com/nethesis/my/backend/logs"
	"github.com/nethesis/my/backend/response"
)

// CustomAuthMiddleware validates our custom JWT tokens and sets user context
func CustomAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logs.Logs.Println("[ERROR][AUTH] Missing Authorization header")
			c.JSON(http.StatusUnauthorized, response.StatusUnauthorized{
				Code:    401,
				Message: "Missing Authorization header",
				Data:    nil,
			})
			c.Abort()
			return
		}

		// Check Bearer prefix
		if !strings.HasPrefix(authHeader, "Bearer ") {
			logs.Logs.Println("[ERROR][AUTH] Invalid Authorization header format")
			c.JSON(http.StatusUnauthorized, response.StatusUnauthorized{
				Code:    401,
				Message: "Invalid Authorization header format",
				Data:    nil,
			})
			c.Abort()
			return
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate custom JWT token
		claims, err := jwt.ValidateCustomToken(tokenString)
		if err != nil {
			logs.Logs.Printf("[ERROR][AUTH] Token validation failed: %v", err)
			c.JSON(http.StatusUnauthorized, response.StatusUnauthorized{
				Code:    401,
				Message: "Invalid token",
				Data:    nil,
			})
			c.Abort()
			return
		}

		// Set user in context for use by other handlers (must be pointer)
		c.Set("user", &claims.User)
		c.Set("user_id", claims.User.ID)
		c.Set("username", claims.User.Username)
		c.Set("email", claims.User.Email)
		c.Set("name", claims.User.Name)
		c.Set("user_roles", claims.User.UserRoles)

		// Set primary user role (first role in array) for backward compatibility
		var userRole string
		if len(claims.User.UserRoles) > 0 {
			userRole = claims.User.UserRoles[0]
		}
		c.Set("user_role", userRole)

		c.Set("user_permissions", claims.User.UserPermissions)
		c.Set("org_role", claims.User.OrgRole)
		c.Set("org_permissions", claims.User.OrgPermissions)
		c.Set("organization_id", claims.User.OrganizationID)
		c.Set("organization_name", claims.User.OrganizationName)

		logs.Logs.Printf("[INFO][AUTH] Custom token validated for user: %s", claims.User.ID)

		// Continue to next handler
		c.Next()
	}
}
