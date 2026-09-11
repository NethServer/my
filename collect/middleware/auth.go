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
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/helpers"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/queue"
	"github.com/nethesis/my/collect/response"
)

// jitteredTTL returns a TTL with ±25% random variation to prevent thundering herd.
// For a 24h base TTL, entries expire between 18h and 30h.
func jitteredTTL(base time.Duration) time.Duration {
	jitter := float64(base) * 0.25
	offset := rand.Float64()*2*jitter - jitter
	return base + time.Duration(offset)
}

// inProcessAuthCache provides an in-process cache for auth results.
var inProcessAuthCache sync.Map

type authCacheEntry struct {
	systemID   string
	valid      bool
	insertedAt time.Time
	expiresAt  time.Time
}

// authCacheKey returns a consistent cache key for in-process and Redis caches
func authCacheKey(systemKey, systemSecret string) string {
	hash := sha256.Sum256([]byte(systemSecret))
	return fmt.Sprintf("%s:%x", systemKey, hash)
}

// checkInProcessCache checks the in-process auth cache.
// Returns (systemID, valid, found).
//
// An entry is dropped when:
//   - it has reached its expiresAt, OR
//   - it was inserted before the most recent invalidation for this
//     system_key (closes the race where sync.Map.Range used by
//     purgeSystemAuthCache might miss an entry installed during the iteration).
func checkInProcessCache(systemKey, systemSecret string) (string, bool, bool) {
	key := authCacheKey(systemKey, systemSecret)
	val, ok := inProcessAuthCache.Load(key)
	if !ok {
		return "", false, false
	}
	entry := val.(*authCacheEntry)
	now := time.Now()
	if now.After(entry.expiresAt) {
		inProcessAuthCache.Delete(key)
		return "", false, false
	}
	if invalidatedAt := LastInvalidatedAt(systemKey); !invalidatedAt.IsZero() && entry.insertedAt.Before(invalidatedAt) {
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
	now := time.Now()
	inProcessAuthCache.Store(key, &authCacheEntry{
		systemID:   systemID,
		valid:      valid,
		insertedAt: now,
		expiresAt:  now.Add(ttl),
	})
}

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

		// Parse Basic auth header. RFC 7235 makes the scheme case-insensitive
		// and allows any run of whitespace before the credentials; every hop
		// in front of collect (nginx, the enterprise feeds' forwardAuth, PHP on
		// legacy my) must classify the same header the same way, so the parser
		// accepts exactly what they accept.
		fields := strings.Fields(auth)
		if len(fields) != 2 || !strings.EqualFold(fields[0], "Basic") {
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
		encoded := fields[1]
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
		systemID, valid, err := validateSystemCredentials(c, systemKey, systemSecret)
		if err != nil {
			// Infrastructure failure (database down or timing out): the
			// credentials were never actually checked, so answering 401
			// would be a lie the appliance acts on by dropping its payload
			// (heartbeat, inventory, ...). 503 tells it to retry instead.
			c.JSON(http.StatusServiceUnavailable, response.Error(http.StatusServiceUnavailable, "authentication temporarily unavailable", nil))
			c.Abort()
			return
		}
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

// systemCredentialsRow holds the DB row for system credentials lookup
type systemCredentialsRow struct {
	systemID     string
	secretPublic string
	secretSHA256 string
	registeredAt sql.NullTime
}

// systemCredentialsQuery is the single gate every appliance request passes
// through. Each lifecycle filter revokes access on its own, so dropping one
// silently brings a revoked credential back to life. The organization guard
// mirrors the LinkFailed monitor: the cascade writes suspended_at down to
// every system, but a system created under an already-suspended
// organization, or one the cascade missed, must not authenticate either.
const systemCredentialsQuery = `
	SELECT s.id, s.system_secret_public, s.system_secret_sha256, s.registered_at
	FROM systems s
	LEFT JOIN distributors d ON (s.organization_id = d.logto_id OR s.organization_id = d.id) AND d.deleted_at IS NULL
	LEFT JOIN resellers r ON (s.organization_id = r.logto_id OR s.organization_id = r.id) AND r.deleted_at IS NULL
	LEFT JOIN customers c ON (s.organization_id = c.logto_id OR s.organization_id = c.id) AND c.deleted_at IS NULL
	WHERE s.system_key = $1
	  AND s.deleted_at IS NULL
	  AND s.suspended_at IS NULL
	  AND s.unregistered_at IS NULL
	  AND COALESCE(d.suspended_at, r.suspended_at, c.suspended_at) IS NULL
`

// validateSystemCredentials validates system credentials against database and cache.
// Returns the internal system_id and a boolean indicating success. A non-nil
// error means the credentials could NOT be checked at all (database
// unreachable or timing out): the caller must answer 503, not 401 — during
// the 2026-09-08 incident a saturated Postgres made this path time out and
// the resulting mass 401s dropped a full fleet heartbeat cycle, firing ~370
// false LinkFailed alerts. Infrastructure failures are never cached as
// negative results for the same reason.
func validateSystemCredentials(c *gin.Context, systemKey, systemSecret string) (string, bool, error) {
	// A username that cannot be a system key never reaches the caches or the
	// database: each unknown pair below is a cache miss and a Postgres
	// round-trip, and the shared instance is the first thing a credential
	// flood saturates.
	if !systemKeyFormat.MatchString(systemKey) {
		logger.Warn().
			Str("system_key", systemKey).
			Msg("Invalid system key format")
		return "", false, nil
	}

	// Validate token format: my_<public>.<secret>
	parts := strings.Split(systemSecret, ".")
	if len(parts) != 2 {
		logger.Warn().
			Str("system_key", systemKey).
			Msg("Invalid system secret format: missing dot separator")
		return "", false, nil
	}

	// Extract public part (remove "my_" prefix)
	publicPart := strings.TrimPrefix(parts[0], "my_")
	if publicPart == parts[0] {
		logger.Warn().
			Str("system_key", systemKey).
			Msg("Invalid system secret format: missing 'my_' prefix")
		return "", false, nil
	}
	secretPart := parts[1]

	// Check minimum secret part length
	if len(secretPart) < configuration.Config.SystemSecretMinLength {
		logger.Warn().
			Str("system_key", systemKey).
			Int("secret_length", len(secretPart)).
			Int("min_length", configuration.Config.SystemSecretMinLength).
			Msg("System secret part too short")
		return "", false, nil
	}

	// Check in-process cache first (fastest, no network)
	if cachedID, valid, found := checkInProcessCache(systemKey, systemSecret); found {
		if valid {
			return cachedID, true, nil
		}
		return "", false, nil
	}

	// Check Redis cache
	if cachedID := checkCredentialsCache(c, systemKey, systemSecret); cachedID != nil {
		if *cachedID != "" {
			// Promote to in-process cache
			setInProcessCache(systemKey, systemSecret, *cachedID, true)
			return *cachedID, true, nil
		}
		// If cached as invalid, still check database for updates
	}

	// Query database for system credentials. `registered_at` is fetched so we
	// can reject systems that have a valid secret on file but have never
	// completed POST /systems/register on the backend — collect operations
	// (heartbeat, inventory, backup upload, alerts/silences proxy) all
	// require an actively-registered appliance. Without this check an
	// unregistered system could push backups into S3 that the admin UI then
	// cannot see (the backend hides `system_key` for non-registered systems
	// in GetSystem, breaking the S3 prefix used by the listing endpoint).
	// suspended_at is filtered too: a suspended system — directly, or via org
	// cascade (SuspendSystemsBy* writes suspended_at down to every affected
	// system) — must not authenticate, so no data is persisted on its behalf
	// (heartbeat, inventory, backups, alerts/silences). Suspending publishes an
	// auth-cache invalidation (cache.InvalidateSystemAuth) so this takes effect
	// immediately instead of after SystemAuthCacheTTL, the same way DeleteSystem
	// handles credential revocation.
	// unregistered_at is the appliance's own departure (POST /systems/unregister):
	// the pair is refused from that moment on, so a copy of the credentials kept
	// outside the machine stops working the instant the machine gives them up.
	var creds systemCredentialsRow

	queryCtx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	err := database.DB.QueryRowContext(queryCtx, systemCredentialsQuery, systemKey).Scan(
		&creds.systemID,
		&creds.secretPublic,
		&creds.secretSHA256,
		&creds.registeredAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		logger.Warn().
			Str("system_key", systemKey).
			Msg("System credentials not found")

		// Cache negative result for short time to prevent brute force
		cacheCredentialsResult(c, systemKey, systemSecret, "", false)
		return "", false, nil
	}
	if err != nil {
		// Database unreachable or query timed out: the credentials are
		// UNKNOWN, not invalid. Do not poison the negative caches and let
		// the caller answer 503 so the appliance knows to retry.
		logger.Error().
			Err(err).
			Str("system_key", systemKey).
			Msg("System credentials lookup failed (database unavailable)")
		return "", false, err
	}

	if !creds.registeredAt.Valid {
		logger.Warn().
			Str("system_key", systemKey).
			Str("system_id", creds.systemID).
			Msg("System has not completed registration; rejecting collect request")

		// Not cached: skipping the failure cache here avoids a stale-401
		// window after the appliance registers.
		return "", false, nil
	}

	// Verify that public part matches system_secret_public in database
	if creds.secretPublic != publicPart {
		logger.Warn().
			Str("system_key", systemKey).
			Str("system_id", creds.systemID).
			Msg("Public part of system secret does not match")

		// Cache negative result
		cacheCredentialsResult(c, systemKey, systemSecret, "", false)
		return "", false, nil
	}

	// Verify secret using SHA256
	valid, err := helpers.VerifySystemSecretSHA256(secretPart, creds.secretSHA256)
	if err != nil {
		logger.Warn().
			Err(err).
			Str("system_key", systemKey).
			Str("system_id", creds.systemID).
			Msg("Failed to verify SHA256 secret")

		cacheCredentialsResult(c, systemKey, systemSecret, "", false)
		setInProcessCache(systemKey, systemSecret, "", false)
		return "", false, nil
	}
	if !valid {
		logger.Warn().
			Str("system_key", systemKey).
			Str("system_id", creds.systemID).
			Msg("Invalid system secret part")

		cacheCredentialsResult(c, systemKey, systemSecret, "", false)
		setInProcessCache(systemKey, systemSecret, "", false)
		return "", false, nil
	}

	// Cache positive result in both caches
	cacheCredentialsResult(c, systemKey, systemSecret, creds.systemID, true)
	setInProcessCache(systemKey, systemSecret, creds.systemID, true)

	return creds.systemID, true, nil
}

// checkCredentialsCache checks Redis cache for cached credentials
// Returns the cached system_id if valid, or nil if not cached
func checkCredentialsCache(c *gin.Context, systemKey, systemSecret string) *string {
	// Use SHA-256 hash of secret for cache key only (not for security verification)
	hash := sha256.Sum256([]byte(systemSecret))
	cacheKey := fmt.Sprintf("auth:system:%s:%x", systemKey, hash)

	rdb := queue.GetClient()
	if rdb == nil {
		// Redis not initialized (unit tests): behave as a cache miss so auth
		// still resolves against the database.
		return nil
	}
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
		ttl = jitteredTTL(configuration.Config.SystemAuthCacheTTL)
	} else {
		value = "invalid"
		ttl = 1 * time.Minute // Short cache for failed attempts
	}

	rdb := queue.GetClient()
	if rdb == nil {
		return
	}
	err := rdb.Set(c.Request.Context(), cacheKey, value, ttl).Err()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to cache auth result")
	}
}
