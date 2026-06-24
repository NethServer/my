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
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
)

// Per-key request budget for API-key-authenticated traffic: sustained rate plus
// a burst allowance, independent of the per-IP limit.
const (
	apiKeyRateLimitPerSecond = 10
	apiKeyRateLimitBurst     = 20
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
		// Audit attributable failures (the key row was found): revoked/expired
		// key, suspended owner, wrong secret. Unknown/unparseable keys carry no
		// attribution and are not audited.
		if result != nil {
			rec := models.APIKeyAuditRecord{
				APIKeyID:       result.KeyID,
				UserID:         result.UserID,
				OrganizationID: result.OrganizationID,
				Event:          models.APIKeyEventAuthFailed,
				Reason:         apiKeyAuditReason(err),
				KeyName:        result.Name,
				KeyMode:        result.Mode,
				IP:             c.ClientIP(),
				Method:         c.Request.Method,
				Path:           c.Request.URL.Path,
			}
			go local.NewAPIKeysService().RecordAPIKeyEvent(rec)
		}
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
		Str("method", c.Request.Method).
		Str("path", c.Request.URL.Path).
		Msg("API key authenticated successfully")

	c.Next()
}

// apiKeyAuditReason maps an authentication error to an audit reason code. It is
// only called for attributable failures (a key row was found).
func apiKeyAuditReason(err error) string {
	switch {
	case errors.Is(err, local.ErrAPIKeyRevoked):
		return models.APIKeyReasonRevoked
	case errors.Is(err, local.ErrAPIKeyExpired):
		return models.APIKeyReasonExpired
	case errors.Is(err, local.ErrAPIKeyUserInactive):
		return models.APIKeyReasonUserInactive
	default:
		return models.APIKeyReasonInvalidSecret
	}
}

// APIKeyRateLimit throttles requests authenticated via an API key, keyed on the
// key id (not the IP) with a token bucket. Requests on any other credential pass
// through untouched — they keep the per-IP limit applied elsewhere.
func APIKeyRateLimit() gin.HandlerFunc {
	var mu sync.Mutex
	keys := make(map[string]*rateLimiterEntry)

	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			now := time.Now()
			for id, entry := range keys {
				if now.Sub(entry.lastCheck) > 10*time.Minute {
					delete(keys, id)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		if !c.GetBool("is_api_key") {
			c.Next()
			return
		}
		id := c.GetString("api_key_id")
		if id == "" {
			c.Next()
			return
		}

		mu.Lock()
		entry, exists := keys[id]
		now := time.Now()
		if !exists {
			entry = &rateLimiterEntry{tokens: float64(apiKeyRateLimitBurst), lastCheck: now}
			keys[id] = entry
		}
		entry.tokens += now.Sub(entry.lastCheck).Seconds() * apiKeyRateLimitPerSecond
		if entry.tokens > float64(apiKeyRateLimitBurst) {
			entry.tokens = float64(apiKeyRateLimitBurst)
		}
		entry.lastCheck = now

		if entry.tokens < 1 {
			mu.Unlock()
			rec := models.APIKeyAuditRecord{
				APIKeyID:       id,
				UserID:         c.GetString("user_id"),
				OrganizationID: c.GetString("organization_id"),
				Event:          models.APIKeyEventRateLimited,
				IP:             c.ClientIP(),
				Method:         c.Request.Method,
				Path:           c.Request.URL.Path,
			}
			go local.NewAPIKeysService().RecordAPIKeyEvent(rec)
			logger.RequestLogger(c, "auth").Warn().
				Str("operation", "api_key_rate_limited").
				Str("api_key_id", id).
				Str("client_ip", c.ClientIP()).
				Msg("API key rate limit exceeded")
			c.JSON(http.StatusTooManyRequests, response.TooManyRequests("rate limit exceeded", nil))
			c.Abort()
			return
		}

		entry.tokens--
		mu.Unlock()
		c.Next()
	}
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
