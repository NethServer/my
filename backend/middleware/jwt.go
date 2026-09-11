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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/jwt"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
)

// setUserContext sets user-related context values on the gin context
func setUserContext(c *gin.Context, user *models.User, isImpersonated bool, impersonator *models.User, sessionID string) {
	c.Set("user", user)
	c.Set("user_id", user.ID)
	c.Set("user_logto_id", user.LogtoID)
	c.Set("username", user.Username)
	c.Set("email", user.Email)
	c.Set("name", user.Name)
	c.Set("phone", user.Phone)
	c.Set("user_roles", user.UserRoles)
	c.Set("user_role_ids", user.UserRoleIDs)
	c.Set("user_permissions", user.UserPermissions)
	c.Set("org_role", user.OrgRole)
	c.Set("org_role_id", user.OrgRoleID)
	c.Set("org_permissions", user.OrgPermissions)
	c.Set("organization_id", user.OrganizationID)
	c.Set("organization_name", user.OrganizationName)

	c.Set("is_impersonated", isImpersonated)
	if isImpersonated && impersonator != nil {
		c.Set("impersonated_by", impersonator)
		c.Set("impersonator_id", impersonator.ID)
		c.Set("impersonator_username", impersonator.Username)
		c.Set("session_id", sessionID)
	} else {
		c.Set("impersonated_by", (*models.User)(nil))
		c.Set("impersonator_id", "")
		c.Set("impersonator_username", "")
	}
}

// revocationSubject is the identifier user-level revocation is keyed on: the
// Logto ID, the one stable identifier every principal has (the bootstrap owner
// has no local row). Every writer — suspend, logout, refresh-reuse burn, the
// organization cascades — keys on the same value.
func revocationSubject(u *models.User) string {
	if u.LogtoID != nil && *u.LogtoID != "" {
		return *u.LogtoID
	}
	return u.ID
}

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
			logger.RequestLogger(c, "auth").Error().
				Err(blacklistErr).
				Str("operation", "blacklist_check_failed").
				Str("client_ip", c.ClientIP()).
				Msg("Failed to check token blacklist - denying request")
			c.JSON(http.StatusServiceUnavailable, response.ServiceUnavailable("security service temporarily unavailable", nil))
			c.Abort()
			return
		}
		if isBlacklisted {
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
					logger.RequestLogger(c, "auth").Error().
						Err(consentErr).
						Str("operation", "impersonation_consent_check_failed").
						Str("impersonated_user_id", impersonationClaims.User.ID).
						Str("impersonator_user_id", impersonationClaims.ImpersonatedBy.ID).
						Str("client_ip", c.ClientIP()).
						Msg("Failed to check impersonation consent - denying request")
					c.JSON(http.StatusServiceUnavailable, response.ServiceUnavailable("security service temporarily unavailable", nil))
					c.Abort()
					return
				}
				if !canBeImpersonated {
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

			// User-level revocation of BOTH principals: a suspended (or logged
			// out) impersonator must not keep acting through the target, and a
			// suspended target must not be acted for. Keyed on the Logto ID
			// like every writer, bound to the token's iat.
			impIssuedAt := time.Time{}
			if impersonationClaims.IssuedAt != nil {
				impIssuedAt = impersonationClaims.IssuedAt.Time
			}
			for _, principal := range []struct {
				user *models.User
				role string
			}{
				{&impersonationClaims.User, "impersonated user"},
				{&impersonationClaims.ImpersonatedBy, "impersonator"},
			} {
				revoked, revokeReason, revokeErr := blacklist.IsUserTokenInvalidatedSince(revocationSubject(principal.user), impIssuedAt)
				if revokeErr != nil {
					logger.RequestLogger(c, "auth").Error().
						Err(revokeErr).
						Str("operation", "impersonation_revocation_check_failed").
						Str("principal", principal.role).
						Str("impersonated_user_id", impersonationClaims.User.ID).
						Str("impersonator_user_id", impersonationClaims.ImpersonatedBy.ID).
						Str("client_ip", c.ClientIP()).
						Msg("Failed to check user revocation - denying request")
					c.JSON(http.StatusServiceUnavailable, response.ServiceUnavailable("security service temporarily unavailable", nil))
					c.Abort()
					return
				}
				if revoked {
					logger.RequestLogger(c, "auth").Warn().
						Str("operation", "revoked_impersonation_principal_rejected").
						Str("principal", principal.role).
						Str("impersonated_user_id", impersonationClaims.User.ID).
						Str("impersonator_user_id", impersonationClaims.ImpersonatedBy.ID).
						Str("client_ip", c.ClientIP()).
						Str("blacklist_reason", revokeReason).
						Msg("Impersonation token rejected: principal revoked")
					c.JSON(http.StatusUnauthorized, response.Unauthorized(principal.role+" account has been suspended", gin.H{
						"reason": revokeReason,
					}))
					c.Abort()
					return
				}
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
			setUserContext(c, &impersonationClaims.User, true, &impersonationClaims.ImpersonatedBy, impersonationClaims.SessionID)

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

		// User-level revocation (suspension, logout, refresh-reuse burn,
		// organization cascade): keyed on the Logto ID like every writer, and
		// bound to the token's iat so a login that follows a logout is not
		// caught by it.
		issuedAt := time.Time{}
		if claims.IssuedAt != nil {
			issuedAt = claims.IssuedAt.Time
		}
		revoked, revokeReason, revokeErr := blacklist.IsUserTokenInvalidatedSince(revocationSubject(&claims.User), issuedAt)
		if revokeErr != nil {
			logger.RequestLogger(c, "auth").Error().
				Err(revokeErr).
				Str("operation", "user_revocation_check_failed").
				Str("user_id", claims.User.ID).
				Str("client_ip", c.ClientIP()).
				Msg("Failed to check user revocation - denying request")
			c.JSON(http.StatusServiceUnavailable, response.ServiceUnavailable("security service temporarily unavailable", nil))
			c.Abort()
			return
		}
		if revoked {
			logger.RequestLogger(c, "auth").Warn().
				Str("operation", "revoked_user_rejected").
				Str("user_id", claims.User.ID).
				Str("client_ip", c.ClientIP()).
				Str("blacklist_reason", revokeReason).
				Msg("Request from revoked user rejected")
			c.JSON(http.StatusUnauthorized, response.Unauthorized("user account has been suspended", gin.H{
				"reason": revokeReason,
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
		setUserContext(c, &claims.User, false, nil, "")

		c.Next()
	}
}
