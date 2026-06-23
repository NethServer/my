/*
 * Copyright (C) 2026 Nethesis S.r.l.
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

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
)

// AuthMiddleware authenticates a request via a personal API key when the bearer
// token carries the API-key prefix, otherwise it falls back to the custom JWT.
// Data endpoints accept either credential; interactive-only groups additionally
// apply RejectAPIKey.
func AuthMiddleware() gin.HandlerFunc {
	jwtAuth := JWTAuthMiddleware()
	return func(c *gin.Context) {
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if strings.HasPrefix(token, helpers.APIKeyPrefix) {
			authenticateAPIKey(c, token)
			return
		}
		jwtAuth(c)
	}
}

func authenticateAPIKey(c *gin.Context, token string) {
	result, err := local.NewAPIKeysService().AuthenticateAPIKey(token)
	if err != nil {
		logger.RequestLogger(c, "auth").Warn().
			Err(err).
			Str("operation", "api_key_auth_failed").
			Str("client_ip", c.ClientIP()).
			Msg("API key authentication failed")
		// Opaque error: never tell the caller why a key was rejected.
		c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid api key", nil))
		c.Abort()
		return
	}

	setUserContext(c, result.User, false, nil, "")
	c.Set("is_api_key", true)
	c.Set("api_key_id", result.KeyID)

	// Record usage out of band so it never adds latency to the request.
	ip := c.ClientIP()
	keyID := result.KeyID
	go local.NewAPIKeysService().TouchLastUsed(keyID, ip)

	logger.RequestLogger(c, "auth").Info().
		Str("operation", "api_key_auth_success").
		Str("user_id", result.User.ID).
		Str("organization_id", result.User.OrganizationID).
		Str("api_key_id", keyID).
		Msg("API key authenticated successfully")

	c.Next()
}

// RejectAPIKey blocks requests authenticated via an API key. Apply it to groups
// that must stay tied to an interactive session: self-service profile, auth,
// impersonation, and API-key management itself (no key may mint or revoke keys).
func RejectAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("is_api_key") {
			c.JSON(http.StatusForbidden, response.Forbidden("this endpoint cannot be accessed with an api key", nil))
			c.Abort()
			return
		}
		c.Next()
	}
}
