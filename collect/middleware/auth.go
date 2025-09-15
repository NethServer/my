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
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
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

		systemID := parts[0]
		systemSecret := parts[1]

		// Validate system credentials
		if !validateSystemCredentials(c, systemID, systemSecret) {
			c.Header("WWW-Authenticate", `Basic realm="System Authentication"`)
			c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid system credentials", nil))
			c.Abort()
			return
		}

		// Set system context for subsequent handlers
		c.Set("system_id", systemID)
		c.Set("authenticated_system", true)

		logger.Debug().
			Str("system_id", systemID).
			Str("client_ip", c.ClientIP()).
			Str("path", c.Request.URL.Path).
			Msg("System authenticated successfully")

		c.Next()
	}
}

// validateSystemCredentials validates system credentials against database and cache
func validateSystemCredentials(c *gin.Context, systemID, systemSecret string) bool {
	// Check minimum secret length
	if len(systemSecret) < configuration.Config.SystemSecretMinLength {
		logger.Warn().
			Str("system_id", systemID).
			Int("secret_length", len(systemSecret)).
			Int("min_length", configuration.Config.SystemSecretMinLength).
			Msg("System secret too short")
		return false
	}

	// Try cache first
	if cached := checkCredentialsCache(c, systemID, systemSecret); cached != nil {
		if *cached {
			return true
		}
		// If cached as invalid, still check database for updates
	}

	// Query database for system credentials from systems table
	var creds models.SystemCredentials
	query := `
		SELECT id, secret_hash, true, null, created_at, updated_at
		FROM systems 
		WHERE id = $1
	`

	err := database.DB.QueryRow(query, systemID).Scan(
		&creds.SystemID,
		&creds.SecretHash,
		&creds.IsActive,
		&creds.LastUsed,
		&creds.CreatedAt,
		&creds.UpdatedAt,
	)

	if err != nil {
		logger.Warn().
			Err(err).
			Str("system_id", systemID).
			Msg("System credentials not found")

		// Cache negative result for short time to prevent brute force
		cacheCredentialsResult(c, systemID, systemSecret, false)
		return false
	}

	// Verify secret hash
	secretHash := hashSystemSecret(systemSecret)
	if secretHash != creds.SecretHash {
		logger.Warn().
			Str("system_id", systemID).
			Msg("Invalid system secret")

		// Cache negative result
		cacheCredentialsResult(c, systemID, systemSecret, false)
		return false
	}

	// Cache positive result
	cacheCredentialsResult(c, systemID, systemSecret, true)

	return true
}

// checkCredentialsCache checks Redis cache for cached credentials
func checkCredentialsCache(c *gin.Context, systemID, systemSecret string) *bool {
	cacheKey := fmt.Sprintf("auth:system:%s:%s", systemID, hashSystemSecret(systemSecret))

	rdb := database.GetRedisClient()
	result, err := rdb.Get(c.Request.Context(), cacheKey).Result()
	if err == redis.Nil {
		return nil // Not in cache
	}
	if err != nil {
		logger.Warn().Err(err).Msg("Redis cache error during auth check")
		return nil
	}

	valid := result == "valid"
	return &valid
}

// cacheCredentialsResult caches the authentication result
func cacheCredentialsResult(c *gin.Context, systemID, systemSecret string, valid bool) {
	cacheKey := fmt.Sprintf("auth:system:%s:%s", systemID, hashSystemSecret(systemSecret))

	var value string
	var ttl time.Duration

	if valid {
		value = "valid"
		ttl = configuration.Config.SystemAuthCacheTTL
	} else {
		value = "invalid"
		ttl = 1 * time.Minute // Short cache for failed attempts
	}

	rdb := database.GetRedisClient()
	err := rdb.Set(c.Request.Context(), cacheKey, value, ttl).Err()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to cache auth result")
	}
}

// hashSystemSecret creates a SHA-256 hash of the system secret
func hashSystemSecret(secret string) string {
	hash := sha256.Sum256([]byte(secret))
	return fmt.Sprintf("%x", hash)
}
