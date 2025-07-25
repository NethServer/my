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
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/jwt"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
)

// JWTAuthMiddleware validates custom JWT tokens and sets user context
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.RequestLogger(c, "auth").Warn().
				Str("operation", "auth_header_missing").
				Str("client_ip", c.ClientIP()).
				Str("user_agent", c.GetHeader("User-Agent")).
				Msg("Authorization header missing")
			c.JSON(http.StatusUnauthorized, response.Unauthorized("authorization header required", nil))
			c.Abort()
			return
		}

		// Check Bearer prefix
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			logger.RequestLogger(c, "auth").Warn().
				Str("operation", "auth_header_invalid_format").
				Str("client_ip", c.ClientIP()).
				Str("auth_header_prefix", authHeader[:min(len(authHeader), 20)]).
				Msg("Invalid authorization header format")
			c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid authorization header format", nil))
			c.Abort()
			return
		}

		// Extract token
		tokenString := authHeader[len(bearerPrefix):]
		if tokenString == "" {
			logger.RequestLogger(c, "auth").Warn().
				Str("operation", "token_empty").
				Str("client_ip", c.ClientIP()).
				Msg("Empty token provided")
			c.JSON(http.StatusUnauthorized, response.Unauthorized("token not provided", nil))
			c.Abort()
			return
		}

		// Validate token
		claims, err := jwt.ValidateCustomToken(tokenString)
		if err != nil {
			logger.RequestLogger(c, "auth").Warn().
				Err(err).
				Str("operation", "token_validation_failed").
				Str("client_ip", c.ClientIP()).
				Str("error_type", "jwt_validation").
				Msg("Custom JWT token validation failed")
			c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid token: "+err.Error(), nil))
			c.Abort()
			return
		}

		// Log successful authentication
		logger.RequestLogger(c, "auth").Info().
			Str("operation", "token_validation_success").
			Str("user_id", claims.User.ID).
			Str("username", claims.User.Username).
			Str("organization_id", claims.User.OrganizationID).
			Str("org_role", claims.User.OrgRole).
			Strs("user_roles", claims.User.UserRoles).
			Msg("Custom JWT token validated successfully")

		// Set user context for subsequent handlers
		c.Set("user", &claims.User)
		c.Set("user_id", claims.User.ID)
		c.Set("username", claims.User.Username)
		c.Set("email", claims.User.Email)
		c.Set("name", claims.User.Name)
		c.Set("phone", claims.User.Phone)
		c.Set("user_roles", claims.User.UserRoles)
		c.Set("user_role_ids", claims.User.UserRoleIDs)
		c.Set("user_permissions", claims.User.UserPermissions)
		c.Set("org_role", claims.User.OrgRole)
		c.Set("org_role_id", claims.User.OrgRoleID)
		c.Set("org_permissions", claims.User.OrgPermissions)
		c.Set("organization_id", claims.User.OrganizationID)
		c.Set("organization_name", claims.User.OrganizationName)

		c.Next()
	}
}
