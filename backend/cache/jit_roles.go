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
	"fmt"
	"sync"
	"time"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/models"
	"github.com/rs/zerolog/log"
)

// JitRolesCache represents cached JIT roles data
type JitRolesCache struct {
	Roles     []models.LogtoOrganizationRole `json:"roles"`
	CachedAt  time.Time                      `json:"cached_at"`
	ExpiresAt time.Time                      `json:"expires_at"`
}

// JitRolesCacheManager manages Redis cache for JIT roles
type JitRolesCacheManager struct {
	redis RedisInterface
	ttl   time.Duration
}

// Global cache instances
var jitRolesCache *JitRolesCacheManager
var jitRolesCacheOnce sync.Once

// GetJitRolesCacheManager returns singleton cache instance
func GetJitRolesCacheManager() *JitRolesCacheManager {
	jitRolesCacheOnce.Do(func() {
		jitRolesCache = &JitRolesCacheManager{
			redis: GetRedisClient(),
			ttl:   configuration.Config.JitRolesCacheTTL,
		}

		if jitRolesCache.redis == nil {
			log.Warn().
				Str("component", "jit_roles_cache").
				Msg("JIT roles cache manager initialized without Redis - cache will be disabled")
		} else {
			log.Info().
				Str("component", "jit_roles_cache").
				Dur("ttl", jitRolesCache.ttl).
				Msg("JIT roles cache manager initialized")
		}
	})
	return jitRolesCache
}

// Get retrieves JIT roles from cache
func (c *JitRolesCacheManager) Get(orgID string) ([]models.LogtoOrganizationRole, bool) {
	// Return cache miss if Redis is not available
	if c.redis == nil {
		return nil, false
	}

	key := fmt.Sprintf("jit_roles:%s", orgID)

	var cached JitRolesCache
	err := c.redis.Get(key, &cached)
	if err != nil {
		if err == ErrCacheMiss {
			log.Debug().
				Str("component", "jit_roles_cache").
				Str("operation", "cache_miss").
				Str("org_id", orgID).
				Msg("JIT roles cache miss")
			return nil, false
		}

		log.Error().
			Str("component", "jit_roles_cache").
			Str("operation", "get").
			Str("org_id", orgID).
			Err(err).
			Msg("Failed to get JIT roles from cache")
		return nil, false
	}

	// Check if expired (double check, Redis should handle TTL)
	if time.Now().After(cached.ExpiresAt) {
		log.Debug().
			Str("component", "jit_roles_cache").
			Str("operation", "cache_expired").
			Str("org_id", orgID).
			Time("expires_at", cached.ExpiresAt).
			Msg("JIT roles cache entry expired")
		return nil, false
	}

	log.Debug().
		Str("component", "jit_roles_cache").
		Str("operation", "cache_hit").
		Str("org_id", orgID).
		Time("cached_at", cached.CachedAt).
		Time("expires_at", cached.ExpiresAt).
		Int("roles_count", len(cached.Roles)).
		Msg("JIT roles cache hit")

	return cached.Roles, true
}

// Set stores JIT roles in cache
func (c *JitRolesCacheManager) Set(orgID string, roles []models.LogtoOrganizationRole) {
	// Skip if Redis is not available
	if c.redis == nil {
		return
	}

	key := fmt.Sprintf("jit_roles:%s", orgID)

	now := time.Now()
	cached := JitRolesCache{
		Roles:     roles,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}

	err := c.redis.Set(key, cached, c.ttl)
	if err != nil {
		log.Error().
			Str("component", "jit_roles_cache").
			Str("operation", "set").
			Str("org_id", orgID).
			Err(err).
			Msg("Failed to set JIT roles in cache")
		return
	}

	log.Debug().
		Str("component", "jit_roles_cache").
		Str("operation", "cache_set").
		Str("org_id", orgID).
		Int("roles_count", len(roles)).
		Dur("ttl", c.ttl).
		Time("expires_at", now.Add(c.ttl)).
		Msg("JIT roles cached")
}

// Clear removes entry from cache
func (c *JitRolesCacheManager) Clear(orgID string) {
	// Skip if Redis is not available
	if c.redis == nil {
		return
	}

	key := fmt.Sprintf("jit_roles:%s", orgID)

	err := c.redis.Delete(key)
	if err != nil {
		log.Error().
			Str("component", "jit_roles_cache").
			Str("operation", "clear").
			Str("org_id", orgID).
			Err(err).
			Msg("Failed to clear JIT roles cache entry")
		return
	}

	log.Debug().
		Str("component", "jit_roles_cache").
		Str("operation", "cache_clear").
		Str("org_id", orgID).
		Msg("JIT roles cache entry cleared")
}

// ClearAll removes all entries from cache
func (c *JitRolesCacheManager) ClearAll() {
	// Skip if Redis is not available
	if c.redis == nil {
		return
	}

	pattern := "jit_roles:*"

	err := c.redis.DeletePattern(pattern)
	if err != nil {
		log.Error().
			Str("component", "jit_roles_cache").
			Str("operation", "clear_all").
			Str("pattern", pattern).
			Err(err).
			Msg("Failed to clear all JIT roles cache entries")
		return
	}

	log.Info().
		Str("component", "jit_roles_cache").
		Str("operation", "cache_clear_all").
		Str("pattern", pattern).
		Msg("All JIT roles cache entries cleared")
}

// GetStats returns cache statistics
func (c *JitRolesCacheManager) GetStats() map[string]interface{} {
	// Return limited stats if Redis is not available
	if c.redis == nil {
		return map[string]interface{}{
			"redis_available": false,
			"ttl_minutes":     c.ttl.Minutes(),
			"cache_prefix":    "jit_roles:",
		}
	}

	// Get Redis client stats
	redisStats, err := c.redis.GetStats()
	if err != nil {
		log.Error().
			Str("component", "jit_roles_cache").
			Str("operation", "get_stats").
			Err(err).
			Msg("Failed to get Redis stats")
		return map[string]interface{}{
			"error":       "failed to get Redis stats",
			"ttl_minutes": c.ttl.Minutes(),
		}
	}

	stats := map[string]interface{}{
		"redis_available": true,
		"ttl_minutes":     c.ttl.Minutes(),
		"redis_stats":     redisStats,
		"cache_prefix":    "jit_roles:",
	}

	return stats
}
