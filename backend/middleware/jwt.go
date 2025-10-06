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
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/jwt"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
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

		// Try to validate as impersonation token first
		impersonationClaims, impErr := jwt.ValidateImpersonationToken(tokenString)
		if impErr == nil {
			// This is a valid impersonation token
			// Check if impersonation consent is still active (security: revoked consent should invalidate active tokens)
			// Skip consent validation if database is not available (e.g., in test environments)
			if database.DB != nil {
				impersonationService := local.NewImpersonationService()
				canBeImpersonated, consentErr := impersonationService.CanBeImpersonated(impersonationClaims.User.ID)
				if consentErr != nil {
					logger.RequestLogger(c, "auth").Warn().
						Err(consentErr).
						Str("operation", "impersonation_consent_check_failed").
						Str("impersonated_user_id", impersonationClaims.User.ID).
						Str("impersonator_user_id", impersonationClaims.ImpersonatedBy.ID).
						Str("client_ip", c.ClientIP()).
						Msg("Failed to check impersonation consent - allowing request (fail open)")
					// Continue with token validation if consent check fails (fail open for infrastructure issues)
				} else if !canBeImpersonated {
					logger.RequestLogger(c, "auth").Warn().
						Str("operation", "impersonation_consent_revoked").
						Str("impersonated_user_id", impersonationClaims.User.ID).
						Str("impersonator_user_id", impersonationClaims.ImpersonatedBy.ID).
						Str("session_id", impersonationClaims.SessionID).
						Str("client_ip", c.ClientIP()).
						Msg("Impersonation consent has been revoked - invalidating token")
					c.JSON(http.StatusUnauthorized, response.Unauthorized("impersonation consent has been revoked", gin.H{
						"message": "the user has revoked consent for impersonation. please exit impersonation mode.",
					}))
					c.Abort()
					return
				}
			}

			// Check user-level blacklist for the impersonated user
			isUserBlacklisted, userBlacklistReason, userBlacklistErr := blacklist.IsUserBlacklisted(impersonationClaims.User.ID)
			if userBlacklistErr != nil {
				logger.RequestLogger(c, "auth").Warn().
					Err(userBlacklistErr).
					Str("operation", "impersonated_user_blacklist_check_failed").
					Str("impersonated_user_id", impersonationClaims.User.ID).
					Str("impersonator_user_id", impersonationClaims.ImpersonatedBy.ID).
					Str("client_ip", c.ClientIP()).
					Msg("Failed to check impersonated user blacklist - allowing request")
				// Continue if user blacklist check fails (fail open)
			} else if isUserBlacklisted {
				logger.RequestLogger(c, "auth").Warn().
					Str("operation", "blacklisted_impersonated_user_rejected").
					Str("impersonated_user_id", impersonationClaims.User.ID).
					Str("impersonator_user_id", impersonationClaims.ImpersonatedBy.ID).
					Str("client_ip", c.ClientIP()).
					Str("blacklist_reason", userBlacklistReason).
					Msg("Request from blacklisted impersonated user rejected")
				c.JSON(http.StatusUnauthorized, response.Unauthorized("impersonated user account has been suspended", gin.H{
					"reason": userBlacklistReason,
				}))
				c.Abort()
				return
			}

			// Log successful impersonation authentication
			logger.RequestLogger(c, "auth").Info().
				Str("operation", "impersonation_token_validation_success").
				Str("impersonated_user_id", impersonationClaims.User.ID).
				Str("impersonated_username", impersonationClaims.User.Username).
				Str("impersonator_user_id", impersonationClaims.ImpersonatedBy.ID).
				Str("impersonator_username", impersonationClaims.ImpersonatedBy.Username).
				Str("organization_id", impersonationClaims.User.OrganizationID).
				Str("org_role", impersonationClaims.User.OrgRole).
				Strs("user_roles", impersonationClaims.User.UserRoles).
				Msg("Impersonation JWT token validated successfully")

			// Set impersonated user context
			c.Set("user", &impersonationClaims.User)
			c.Set("user_id", impersonationClaims.User.ID)
			c.Set("username", impersonationClaims.User.Username)
			c.Set("email", impersonationClaims.User.Email)
			c.Set("name", impersonationClaims.User.Name)
			c.Set("phone", impersonationClaims.User.Phone)
			c.Set("user_roles", impersonationClaims.User.UserRoles)
			c.Set("user_role_ids", impersonationClaims.User.UserRoleIDs)
			c.Set("user_permissions", impersonationClaims.User.UserPermissions)
			c.Set("org_role", impersonationClaims.User.OrgRole)
			c.Set("org_role_id", impersonationClaims.User.OrgRoleID)
			c.Set("org_permissions", impersonationClaims.User.OrgPermissions)
			c.Set("organization_id", impersonationClaims.User.OrganizationID)
			c.Set("organization_name", impersonationClaims.User.OrganizationName)

			// Set impersonation context
			c.Set("is_impersonated", true)
			c.Set("impersonated_by", &impersonationClaims.ImpersonatedBy)
			c.Set("impersonator_id", impersonationClaims.ImpersonatedBy.ID)
			c.Set("impersonator_username", impersonationClaims.ImpersonatedBy.Username)
			c.Set("session_id", impersonationClaims.SessionID)

			c.Next()
			return
		}

		// Try to validate as regular custom token
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

		// Set impersonation context (false for regular tokens)
		c.Set("is_impersonated", false)
		c.Set("impersonated_by", (*models.User)(nil))
		c.Set("impersonator_id", "")
		c.Set("impersonator_username", "")

		c.Next()
	}
}
