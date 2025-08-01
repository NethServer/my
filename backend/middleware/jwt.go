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
	"github.com/nethesis/my/backend/cache"
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

		// Check token blacklist before validation
		blacklist := cache.GetTokenBlacklist()
		isBlacklisted, blacklistReason, blacklistErr := blacklist.IsTokenBlacklisted(tokenString)
		if blacklistErr != nil {
			logger.RequestLogger(c, "auth").Warn().
				Err(blacklistErr).
				Str("operation", "blacklist_check_failed").
				Str("client_ip", c.ClientIP()).
				Msg("Failed to check token blacklist - allowing request")
			// Continue with token validation if blacklist check fails (fail open)
		} else if isBlacklisted {
			logger.RequestLogger(c, "auth").Warn().
				Str("operation", "blacklisted_token_rejected").
				Str("client_ip", c.ClientIP()).
				Str("blacklist_reason", blacklistReason).
				Msg("Blacklisted token rejected")
			c.JSON(http.StatusUnauthorized, response.Unauthorized("token has been revoked", gin.H{
				"reason": blacklistReason,
			}))
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
			c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid token", nil))
			c.Abort()
			return
		}

		// Check user-level blacklist after token validation
		isUserBlacklisted, userBlacklistReason, userBlacklistErr := blacklist.IsUserBlacklisted(claims.User.ID)
		if userBlacklistErr != nil {
			logger.RequestLogger(c, "auth").Warn().
				Err(userBlacklistErr).
				Str("operation", "user_blacklist_check_failed").
				Str("user_id", claims.User.ID).
				Str("client_ip", c.ClientIP()).
				Msg("Failed to check user blacklist - allowing request")
			// Continue if user blacklist check fails (fail open)
		} else if isUserBlacklisted {
			logger.RequestLogger(c, "auth").Warn().
				Str("operation", "blacklisted_user_rejected").
				Str("user_id", claims.User.ID).
				Str("client_ip", c.ClientIP()).
				Str("blacklist_reason", userBlacklistReason).
				Msg("Request from blacklisted user rejected")
			c.JSON(http.StatusUnauthorized, response.Unauthorized("user account has been suspended", gin.H{
				"reason": userBlacklistReason,
			}))
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
