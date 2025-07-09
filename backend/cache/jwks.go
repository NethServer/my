/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package cache

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/nethesis/my/backend/configuration"
	"github.com/rs/zerolog/log"
)

// JWK represents a JSON Web Key
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWKSet represents a set of JSON Web Keys
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

// JWKSCache represents cached JWKS data
type JWKSCache struct {
	Keys      map[string]*rsa.PublicKey `json:"keys"`
	CachedAt  time.Time                 `json:"cached_at"`
	ExpiresAt time.Time                 `json:"expires_at"`
}

// JWKSCacheManager manages Redis cache for JWKS
type JWKSCacheManager struct {
	redis    *RedisClient
	ttl      time.Duration
	endpoint string
}

// Global cache instances
var jwksCache *JWKSCacheManager
var jwksCacheOnce sync.Once

// GetJWKSCacheManager returns singleton cache instance for JWKS
func GetJWKSCacheManager() *JWKSCacheManager {
	jwksCacheOnce.Do(func() {
		jwksCache = &JWKSCacheManager{
			redis:    GetRedisClient(),
			ttl:      configuration.Config.JWKSCacheTTL,
			endpoint: configuration.Config.JWKSEndpoint,
		}

		log.Info().
			Str("component", "jwks_cache").
			Dur("ttl", jwksCache.ttl).
			Str("endpoint", jwksCache.endpoint).
			Msg("JWKS cache manager initialized")
	})
	return jwksCache
}

// GetPublicKey retrieves a public key by kid
func (c *JWKSCacheManager) GetPublicKey(kid string) (*rsa.PublicKey, error) {
	// Try to get from cache first
	key := "jwks:keys"

	var cached JWKSCache
	err := c.redis.Get(key, &cached)
	if err == nil {
		// Check if expired (double check, Redis should handle TTL)
		if time.Now().Before(cached.ExpiresAt) {
			if publicKey, exists := cached.Keys[kid]; exists {
				log.Debug().
					Str("component", "jwks_cache").
					Str("operation", "cache_hit").
					Str("kid", kid).
					Time("cached_at", cached.CachedAt).
					Time("expires_at", cached.ExpiresAt).
					Msg("JWKS cache hit")
				return publicKey, nil
			}
		}
	}

	// Cache miss or expired, fetch from endpoint
	log.Debug().
		Str("component", "jwks_cache").
		Str("operation", "cache_miss").
		Str("kid", kid).
		Msg("JWKS cache miss, fetching from endpoint")

	keys, err := c.fetchAndCacheJWKS()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	if publicKey, exists := keys[kid]; exists {
		return publicKey, nil
	}

	return nil, fmt.Errorf("key not found for kid: %s", kid)
}

// fetchAndCacheJWKS fetches JWKS from endpoint and caches it
func (c *JWKSCacheManager) fetchAndCacheJWKS() (map[string]*rsa.PublicKey, error) {
	log.Info().
		Str("component", "jwks_cache").
		Str("operation", "fetch_start").
		Str("endpoint", c.endpoint).
		Msg("Fetching JWKS from endpoint")

	start := time.Now()
	client := &http.Client{Timeout: configuration.Config.JWKSHTTPTimeout}
	resp, err := client.Get(c.endpoint)
	if err != nil {
		log.Error().
			Str("component", "jwks_cache").
			Str("operation", "fetch_failed").
			Str("endpoint", c.endpoint).
			Dur("duration", time.Since(start)).
			Err(err).
			Msg("Failed to fetch JWKS")
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB max
		log.Error().
			Str("component", "jwks_cache").
			Str("operation", "fetch_bad_status").
			Int("status_code", resp.StatusCode).
			Str("endpoint", c.endpoint).
			Str("response_body", string(body)).
			Dur("duration", time.Since(start)).
			Msg("JWKS endpoint returned error status")
		return nil, fmt.Errorf("JWKS endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var jwks JWKSet
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1024*1024)).Decode(&jwks); err != nil {
		log.Error().
			Str("component", "jwks_cache").
			Str("operation", "decode_failed").
			Int("status_code", resp.StatusCode).
			Dur("duration", time.Since(start)).
			Err(err).
			Msg("Failed to decode JWKS response")
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Convert JWKs to RSA public keys
	keys := make(map[string]*rsa.PublicKey)
	keysProcessed := 0
	keysSuccessful := 0

	for _, jwk := range jwks.Keys {
		keysProcessed++
		if jwk.Kty == "RSA" && jwk.Use == "sig" {
			key, err := JwkToRSAPublicKey(jwk)
			if err != nil {
				log.Warn().
					Str("component", "jwks_cache").
					Str("operation", "convert_jwk_to_rsa").
					Str("kid", jwk.Kid).
					Err(err).
					Msg("Failed to convert JWK to RSA key")
				continue
			}
			keys[jwk.Kid] = key
			keysSuccessful++
		}
	}

	// Cache the keys
	now := time.Now()
	cached := JWKSCache{
		Keys:      keys,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}

	cacheKey := "jwks:keys"
	err = c.redis.Set(cacheKey, cached, c.ttl)
	if err != nil {
		log.Error().
			Str("component", "jwks_cache").
			Str("operation", "cache_set_failed").
			Err(err).
			Msg("Failed to cache JWKS keys, continuing with fetched keys")
	}

	log.Info().
		Str("component", "jwks_cache").
		Str("operation", "cache_updated").
		Int("keys_processed", keysProcessed).
		Int("keys_successful", keysSuccessful).
		Int("total_cached", len(keys)).
		Dur("fetch_duration", time.Since(start)).
		Time("cache_time", now).
		Msg("JWKS cache updated successfully")

	return keys, nil
}

// ClearCache removes all JWKS cache entries
func (c *JWKSCacheManager) ClearCache() error {
	pattern := "jwks:*"

	err := c.redis.DeletePattern(pattern)
	if err != nil {
		log.Error().
			Str("component", "jwks_cache").
			Str("operation", "clear_cache").
			Str("pattern", pattern).
			Err(err).
			Msg("Failed to clear JWKS cache")
		return err
	}

	log.Info().
		Str("component", "jwks_cache").
		Str("operation", "cache_cleared").
		Str("pattern", pattern).
		Msg("JWKS cache cleared")

	return nil
}

// GetStats returns cache statistics
func (c *JWKSCacheManager) GetStats() map[string]interface{} {
	// Get Redis client stats
	redisStats, err := c.redis.GetStats()
	if err != nil {
		log.Error().
			Str("component", "jwks_cache").
			Str("operation", "get_stats").
			Err(err).
			Msg("Failed to get Redis stats")
		return map[string]interface{}{
			"error":       "failed to get Redis stats",
			"ttl_minutes": c.ttl.Minutes(),
		}
	}

	stats := map[string]interface{}{
		"ttl_minutes":  c.ttl.Minutes(),
		"endpoint":     c.endpoint,
		"redis_stats":  redisStats,
		"cache_prefix": "jwks:",
	}

	return stats
}

// JwkToRSAPublicKey converts a JWK to RSA public key
func JwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decode base64url encoded modulus (n)
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	// Decode base64url encoded exponent (e)
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert bytes to big integers
	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())

	return &rsa.PublicKey{N: n, E: e}, nil
}
