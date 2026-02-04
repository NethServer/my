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
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/helpers"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/queue"
	"github.com/nethesis/my/collect/response"
)

// BasicAuthMiddleware implements HTTP Basic authentication for system credentials
func BasicAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			logger.Warn().
				Str("client_ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("Missing Authorization header")

			c.Header("WWW-Authenticate", `Basic realm="System Authentication"`)
			c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
			c.Abort()
			return
		}

		// Parse Basic auth header
		const prefix = "Basic "
		if !strings.HasPrefix(auth, prefix) {
			logger.Warn().
				Str("client_ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("Invalid Authorization header format")

			c.Header("WWW-Authenticate", `Basic realm="System Authentication"`)
			c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid authentication format", nil))
			c.Abort()
			return
		}

		// Decode base64 credentials
		encoded := auth[len(prefix):]
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("client_ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("Invalid base64 encoding in Authorization header")

			c.Header("WWW-Authenticate", `Basic realm="System Authentication"`)
			c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid authentication encoding", nil))
			c.Abort()
			return
		}

		// Parse username:password
		credentials := string(decoded)
		parts := strings.SplitN(credentials, ":", 2)
		if len(parts) != 2 {
			logger.Warn().
				Str("client_ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("invalid credentials format")

			c.Header("WWW-Authenticate", `Basic realm="System Authentication"`)
			c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid credentials format", nil))
			c.Abort()
			return
		}

		systemKey := parts[0]
		systemSecret := parts[1]

		// Validate system credentials
		systemID, valid := validateSystemCredentials(c, systemKey, systemSecret)
		if !valid {
			c.Header("WWW-Authenticate", `Basic realm="System Authentication"`)
			c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid system credentials", nil))
			c.Abort()
			return
		}

		// Set system context for subsequent handlers (both key and internal ID)
		c.Set("system_id", systemID)
		c.Set("system_key", systemKey)
		c.Set("authenticated_system", true)

		logger.Debug().
			Str("system_key", systemKey).
			Str("system_id", systemID).
			Str("client_ip", c.ClientIP()).
			Str("path", c.Request.URL.Path).
			Msg("System authenticated successfully")

		c.Next()
	}
}

// validateSystemCredentials validates system credentials against database and cache
// Returns the internal system_id and a boolean indicating success
func validateSystemCredentials(c *gin.Context, systemKey, systemSecret string) (string, bool) {
	// Validate token format: my_<public>.<secret>
	parts := strings.Split(systemSecret, ".")
	if len(parts) != 2 {
		logger.Warn().
			Str("system_key", systemKey).
			Msg("Invalid system secret format: missing dot separator")
		return "", false
	}

	// Extract public part (remove "my_" prefix)
	publicPart := strings.TrimPrefix(parts[0], "my_")
	if publicPart == parts[0] {
		logger.Warn().
			Str("system_key", systemKey).
			Msg("Invalid system secret format: missing 'my_' prefix")
		return "", false
	}
	secretPart := parts[1]

	// Check minimum secret part length
	if len(secretPart) < configuration.Config.SystemSecretMinLength {
		logger.Warn().
			Str("system_key", systemKey).
			Int("secret_length", len(secretPart)).
			Int("min_length", configuration.Config.SystemSecretMinLength).
			Msg("System secret part too short")
		return "", false
	}

	// Try cache first
	if cachedID := checkCredentialsCache(c, systemKey, systemSecret); cachedID != nil {
		if *cachedID != "" {
			return *cachedID, true
		}
		// If cached as invalid, still check database for updates
	}

	// Query database for system credentials including system_secret_public
	var creds models.SystemCredentials
	var systemSecretPublic string
	query := `
		SELECT id, system_secret, system_secret_public, true, null, created_at, updated_at
		FROM systems
		WHERE system_key = $1 AND deleted_at IS NULL
	`

	err := database.DB.QueryRow(query, systemKey).Scan(
		&creds.SystemID,
		&creds.SecretHash,
		&systemSecretPublic,
		&creds.IsActive,
		&creds.LastUsed,
		&creds.CreatedAt,
		&creds.UpdatedAt,
	)

	if err != nil {
		logger.Warn().
			Err(err).
			Str("system_key", systemKey).
			Msg("System credentials not found")

		// Cache negative result for short time to prevent brute force
		cacheCredentialsResult(c, systemKey, systemSecret, "", false)
		return "", false
	}

	// Verify that public part matches system_secret_public in database
	if systemSecretPublic != publicPart {
		logger.Warn().
			Str("system_key", systemKey).
			Str("system_id", creds.SystemID).
			Msg("Public part of system secret does not match")

		// Cache negative result
		cacheCredentialsResult(c, systemKey, systemSecret, "", false)
		return "", false
	}

	// Verify secret part hash using Argon2id
	valid, err := helpers.VerifySystemSecret(secretPart, creds.SecretHash)
	if err != nil {
		logger.Warn().
			Err(err).
			Str("system_key", systemKey).
			Str("system_id", creds.SystemID).
			Msg("Failed to verify system secret part")

		// Cache negative result
		cacheCredentialsResult(c, systemKey, systemSecret, "", false)
		return "", false
	}

	if !valid {
		logger.Warn().
			Str("system_key", systemKey).
			Str("system_id", creds.SystemID).
			Msg("Invalid system secret part")

		// Cache negative result
		cacheCredentialsResult(c, systemKey, systemSecret, "", false)
		return "", false
	}

	// Cache positive result
	cacheCredentialsResult(c, systemKey, systemSecret, creds.SystemID, true)

	return creds.SystemID, true
}

// checkCredentialsCache checks Redis cache for cached credentials
// Returns the cached system_id if valid, or nil if not cached
func checkCredentialsCache(c *gin.Context, systemKey, systemSecret string) *string {
	// Use SHA-256 hash of secret for cache key only (not for security verification)
	hash := sha256.Sum256([]byte(systemSecret))
	cacheKey := fmt.Sprintf("auth:system:%s:%x", systemKey, hash)

	rdb := queue.GetClient()
	result, err := rdb.Get(c.Request.Context(), cacheKey).Result()
	if err == redis.Nil {
		return nil // Not in cache
	}
	if err != nil {
		logger.Warn().Err(err).Msg("Redis cache error during auth check")
		return nil
	}

	if result == "invalid" {
		empty := ""
		return &empty
	}

	// Return the cached system_id
	return &result
}

// cacheCredentialsResult caches the authentication result
// If valid, caches the system_id for the given system_key
func cacheCredentialsResult(c *gin.Context, systemKey, systemSecret, systemID string, valid bool) {
	// Use SHA-256 hash of secret for cache key only (not for security verification)
	hash := sha256.Sum256([]byte(systemSecret))
	cacheKey := fmt.Sprintf("auth:system:%s:%x", systemKey, hash)

	var value string
	var ttl time.Duration

	if valid {
		value = systemID // Store the system_id
		ttl = configuration.Config.SystemAuthCacheTTL
	} else {
		value = "invalid"
		ttl = 1 * time.Minute // Short cache for failed attempts
	}

	rdb := queue.GetClient()
	err := rdb.Set(c.Request.Context(), cacheKey, value, ttl).Err()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to cache auth result")
	}
}
