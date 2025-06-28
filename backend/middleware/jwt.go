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
	"github.com/nethesis/my/backend/response"
)

// JWTAuthMiddleware validates custom JWT tokens and sets user context
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, response.Unauthorized("authorization header required", nil))
			c.Abort()
			return
		}

		// Check Bearer prefix
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid authorization header format", nil))
			c.Abort()
			return
		}

		// Extract token
		tokenString := authHeader[len(bearerPrefix):]
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, response.Unauthorized("token not provided", nil))
			c.Abort()
			return
		}

		// Validate token
		claims, err := jwt.ValidateCustomToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid token: "+err.Error(), nil))
			c.Abort()
			return
		}

		// Set user context for subsequent handlers
		c.Set("user", &claims.User)
		c.Set("user_id", claims.User.ID)
		c.Set("username", claims.User.Username)
		c.Set("email", claims.User.Email)
		c.Set("name", claims.User.Name)
		c.Set("user_roles", claims.User.UserRoles)
		c.Set("user_permissions", claims.User.UserPermissions)
		c.Set("org_role", claims.User.OrgRole)
		c.Set("org_permissions", claims.User.OrgPermissions)
		c.Set("organization_id", claims.User.OrganizationID)
		c.Set("organization_name", claims.User.OrganizationName)

		c.Next()
	}
}
