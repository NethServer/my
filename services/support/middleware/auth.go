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
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/nethesis/my/services/support/configuration"
	"github.com/nethesis/my/services/support/database"
	"github.com/nethesis/my/services/support/helpers"
	"github.com/nethesis/my/services/support/logger"
	"github.com/nethesis/my/services/support/queue"
	"github.com/nethesis/my/services/support/response"
)

// InternalSecretMiddleware validates the X-Internal-Secret header (#4).
// Provides defense-in-depth: even if a session token leaks, the caller
// must also know the shared internal secret to access tunnel endpoints.
// INTERNAL_SECRET is required at startup; this middleware always enforces it.
func InternalSecretMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := configuration.Config.InternalSecret
		provided := c.GetHeader("X-Internal-Secret")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(secret)) != 1 {
			logger.Warn().
				Str("client_ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("invalid or missing internal secret")
			c.JSON(http.StatusForbidden, response.Error(http.StatusForbidden, "forbidden", nil))
			c.Abort()
			return
		}
		c.Next()
	}
}

// SessionTokenMiddleware validates the X-Session-Token header for
// internal endpoints. Each request is tied to a specific active session
// via the session_id URL parameter, eliminating the single shared secret.
func SessionTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract session_id from URL (works for both /terminal/:session_id and /proxy/:session_id/...)
		sessionID := c.Param("session_id")
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, response.BadRequest("session id required", nil))
			c.Abort()
			return
		}

		provided := c.GetHeader("X-Session-Token")
		if provided == "" {
			logger.Warn().
				Str("client_ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("missing X-Session-Token header")
			c.JSON(http.StatusForbidden, response.Error(http.StatusForbidden, "session token required", nil))
			c.Abort()
			return
		}

		// Look up the session token from the database
		var storedToken string
		err := database.DB.QueryRow(
			`SELECT session_token FROM support_sessions WHERE id = $1 AND status IN ('pending', 'active')`,
			sessionID,
		).Scan(&storedToken)
		if err != nil {
			logger.Warn().
				Str("client_ip", c.ClientIP()).
				Str("session_id", sessionID).
				Msg("session not found or not active")
			c.JSON(http.StatusForbidden, response.Error(http.StatusForbidden, "invalid session", nil))
			c.Abort()
			return
		}

		// Constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(provided), []byte(storedToken)) != 1 {
			logger.Warn().
				Str("client_ip", c.ClientIP()).
				Str("session_id", sessionID).
				Msg("invalid session token")
			c.JSON(http.StatusForbidden, response.Error(http.StatusForbidden, "forbidden", nil))
			c.Abort()
			return
		}

		c.Next()
	}
}

// jitteredTTL returns a TTL with ±25% random variation to prevent thundering herd.
func jitteredTTL(base time.Duration) time.Duration {
	jitter := float64(base) * 0.25
	offset := rand.Float64()*2*jitter - jitter
	return base + time.Duration(offset)
}

var inProcessAuthCache sync.Map

func init() {
	// Sweep expired entries every 10 minutes to bound memory usage
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			inProcessAuthCache.Range(func(key, value any) bool {
				if entry, ok := value.(*authCacheEntry); ok && now.After(entry.expiresAt) {
					inProcessAuthCache.Delete(key)
				}
				return true
			})
		}
	}()
}

type authCacheEntry struct {
	systemID  string
	valid     bool
	expiresAt time.Time
}

// authCacheKey returns a consistent cache key for in-process and Redis caches
func authCacheKey(systemKey, systemSecret string) string {
	hash := sha256.Sum256([]byte(systemSecret))
	return fmt.Sprintf("%s:%x", systemKey, hash)
}

// checkInProcessCache checks the in-process auth cache.
// Returns (systemID, valid, found).
func checkInProcessCache(systemKey, systemSecret string) (string, bool, bool) {
	key := authCacheKey(systemKey, systemSecret)
	val, ok := inProcessAuthCache.Load(key)
	if !ok {
		return "", false, false
	}
	entry := val.(*authCacheEntry)
	if time.Now().After(entry.expiresAt) {
		inProcessAuthCache.Delete(key)
		return "", false, false
	}
	return entry.systemID, entry.valid, true
}

// setInProcessCache stores an auth result in the in-process cache
func setInProcessCache(systemKey, systemSecret, systemID string, valid bool) {
	key := authCacheKey(systemKey, systemSecret)
	var ttl time.Duration
	if valid {
		ttl = jitteredTTL(configuration.Config.SystemAuthCacheTTL)
	} else {
		ttl = 1 * time.Minute
	}
	inProcessAuthCache.Store(key, &authCacheEntry{
		systemID:  systemID,
		valid:     valid,
		expiresAt: time.Now().Add(ttl),
	})
}

// InvalidateAuthCache removes cached credentials for a system key from both caches.
// Called when system secrets are regenerated via Redis pub/sub.
func InvalidateAuthCache(ctx context.Context, systemKey string) {
	// Clear all in-process cache entries for this system key
	inProcessAuthCache.Range(func(key, _ any) bool {
		if k, ok := key.(string); ok && strings.HasPrefix(k, systemKey+":") {
			inProcessAuthCache.Delete(key)
		}
		return true
	})

	// Clear Redis cache entries for this system key
	rdb := queue.GetClient()
	pattern := fmt.Sprintf("auth:system:%s:*", systemKey)
	iter := rdb.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		_ = rdb.Del(ctx, iter.Val()).Err()
	}
	if err := iter.Err(); err != nil {
		logger.Warn().Err(err).Str("system_key", systemKey).Msg("redis scan error during cache invalidation")
	}
}

// StartAuthCacheInvalidator listens for cache invalidation events via Redis pub/sub.
// When a system secret is regenerated, the backend publishes the system_key to this channel.
func StartAuthCacheInvalidator(ctx context.Context) {
	log := logger.ComponentLogger("auth_cache")
	pubsub := queue.Subscribe(ctx, "support:auth:invalidate")
	defer func() { _ = pubsub.Close() }()

	ch := pubsub.Channel()
	log.Info().Msg("auth cache invalidator started on support:auth:invalidate channel")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("auth cache invalidator stopped")
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			systemKey := msg.Payload
			if systemKey != "" {
				InvalidateAuthCache(ctx, systemKey)
				log.Info().Str("system_key", systemKey).Msg("auth cache invalidated")
			}
		}
	}
}

// BasicAuthMiddleware validates system credentials using HTTP Basic Auth.
// Also verifies the system has support_enabled = true (#2).
func BasicAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			logger.Warn().
				Str("client_ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("missing Authorization header")

			c.Header("WWW-Authenticate", `Basic realm="System Authentication"`)
			c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
			c.Abort()
			return
		}

		const prefix = "Basic "
		if !strings.HasPrefix(auth, prefix) {
			logger.Warn().
				Str("client_ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("invalid Authorization header format")

			c.Header("WWW-Authenticate", `Basic realm="System Authentication"`)
			c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid authentication format", nil))
			c.Abort()
			return
		}

		encoded := auth[len(prefix):]
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("client_ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("invalid base64 encoding in Authorization header")

			c.Header("WWW-Authenticate", `Basic realm="System Authentication"`)
			c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid authentication encoding", nil))
			c.Abort()
			return
		}

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

		systemID, valid := validateSystemCredentials(c, systemKey, systemSecret)
		if !valid {
			c.Header("WWW-Authenticate", `Basic realm="System Authentication"`)
			c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid system credentials", nil))
			c.Abort()
			return
		}

		// #2: Check support_enabled flag — system must opt-in explicitly
		var supportEnabled bool
		err = database.DB.QueryRow(
			`SELECT support_enabled FROM systems WHERE id = $1 AND deleted_at IS NULL`,
			systemID,
		).Scan(&supportEnabled)
		if err != nil || !supportEnabled {
			logger.Warn().
				Str("system_key", systemKey).
				Str("system_id", systemID).
				Bool("support_enabled", supportEnabled).
				Msg("support not enabled for this system")
			c.JSON(http.StatusForbidden, response.Error(http.StatusForbidden, "support is not enabled for this system", nil))
			c.Abort()
			return
		}

		c.Set("system_id", systemID)
		c.Set("system_key", systemKey)
		c.Set("authenticated_system", true)

		logger.Debug().
			Str("system_key", systemKey).
			Str("system_id", systemID).
			Str("client_ip", c.ClientIP()).
			Str("path", c.Request.URL.Path).
			Msg("system authenticated successfully")

		c.Next()
	}
}

// systemCredentialsRow holds the DB row for system credentials lookup
type systemCredentialsRow struct {
	systemID     string
	secretPublic string
	secretSHA256 string
}

// validateSystemCredentials validates system credentials against database and cache
func validateSystemCredentials(c *gin.Context, systemKey, systemSecret string) (string, bool) {
	parts := strings.Split(systemSecret, ".")
	if len(parts) != 2 {
		logger.Warn().
			Str("system_key", systemKey).
			Msg("invalid system secret format: missing dot separator")
		return "", false
	}

	publicPart := strings.TrimPrefix(parts[0], "my_")
	if publicPart == parts[0] {
		logger.Warn().
			Str("system_key", systemKey).
			Msg("invalid system secret format: missing 'my_' prefix")
		return "", false
	}
	secretPart := parts[1]

	if len(secretPart) < configuration.Config.SystemSecretMinLength {
		logger.Warn().
			Str("system_key", systemKey).
			Int("secret_length", len(secretPart)).
			Int("min_length", configuration.Config.SystemSecretMinLength).
			Msg("system secret part too short")
		return "", false
	}

	// Check in-process cache first (fastest, no network)
	if cachedID, valid, found := checkInProcessCache(systemKey, systemSecret); found {
		if valid {
			return cachedID, true
		}
		return "", false
	}

	// Check Redis cache
	if cachedID := checkCredentialsCache(c, systemKey, systemSecret); cachedID != nil {
		if *cachedID != "" {
			setInProcessCache(systemKey, systemSecret, *cachedID, true)
			return *cachedID, true
		}
	}

	// Query database for system credentials
	var creds systemCredentialsRow
	query := `
		SELECT id, system_secret_public, system_secret_sha256
		FROM systems
		WHERE system_key = $1 AND deleted_at IS NULL
	`

	err := database.DB.QueryRow(query, systemKey).Scan(
		&creds.systemID,
		&creds.secretPublic,
		&creds.secretSHA256,
	)

	if err != nil {
		logger.Warn().
			Err(err).
			Str("system_key", systemKey).
			Msg("system credentials not found")

		cacheCredentialsResult(c, systemKey, systemSecret, "", false)
		return "", false
	}

	if creds.secretPublic != publicPart {
		logger.Warn().
			Str("system_key", systemKey).
			Str("system_id", creds.systemID).
			Msg("public part of system secret does not match")

		cacheCredentialsResult(c, systemKey, systemSecret, "", false)
		return "", false
	}

	// Verify secret using SHA256
	valid, err := helpers.VerifySystemSecretSHA256(secretPart, creds.secretSHA256)
	if err != nil {
		logger.Warn().
			Err(err).
			Str("system_key", systemKey).
			Str("system_id", creds.systemID).
			Msg("failed to verify SHA256 secret")

		cacheCredentialsResult(c, systemKey, systemSecret, "", false)
		setInProcessCache(systemKey, systemSecret, "", false)
		return "", false
	}
	if !valid {
		logger.Warn().
			Str("system_key", systemKey).
			Str("system_id", creds.systemID).
			Msg("invalid system secret part")

		cacheCredentialsResult(c, systemKey, systemSecret, "", false)
		setInProcessCache(systemKey, systemSecret, "", false)
		return "", false
	}

	// Cache positive result in both caches
	cacheCredentialsResult(c, systemKey, systemSecret, creds.systemID, true)
	setInProcessCache(systemKey, systemSecret, creds.systemID, true)

	return creds.systemID, true
}

// checkCredentialsCache checks Redis cache for cached credentials
func checkCredentialsCache(c *gin.Context, systemKey, systemSecret string) *string {
	hash := sha256.Sum256([]byte(systemSecret))
	cacheKey := fmt.Sprintf("auth:system:%s:%x", systemKey, hash)

	rdb := queue.GetClient()
	result, err := rdb.Get(c.Request.Context(), cacheKey).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		logger.Warn().Err(err).Msg("redis cache error during auth check")
		return nil
	}

	if result == "invalid" {
		empty := ""
		return &empty
	}

	return &result
}

// cacheCredentialsResult caches the authentication result
func cacheCredentialsResult(c *gin.Context, systemKey, systemSecret, systemID string, valid bool) {
	hash := sha256.Sum256([]byte(systemSecret))
	cacheKey := fmt.Sprintf("auth:system:%s:%x", systemKey, hash)

	var value string
	var ttl time.Duration

	if valid {
		value = systemID
		ttl = jitteredTTL(configuration.Config.SystemAuthCacheTTL)
	} else {
		value = "invalid"
		ttl = 1 * time.Minute
	}

	rdb := queue.GetClient()
	err := rdb.Set(c.Request.Context(), cacheKey, value, ttl).Err()
	if err != nil {
		logger.Warn().Err(err).Msg("failed to cache auth result")
	}
}
