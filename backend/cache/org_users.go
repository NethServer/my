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

// OrgUsersCache represents cached organization users data
type OrgUsersCache struct {
	Users     []models.LogtoUser `json:"users"`
	CachedAt  time.Time          `json:"cached_at"`
	ExpiresAt time.Time          `json:"expires_at"`
}

// OrgUsersCacheManager manages Redis cache for organization users
type OrgUsersCacheManager struct {
	redis RedisInterface
	ttl   time.Duration
}

// Global cache instances
var orgUsersCache *OrgUsersCacheManager
var orgUsersCacheOnce sync.Once

// GetOrgUsersCacheManager returns singleton cache instance for organization users
func GetOrgUsersCacheManager() *OrgUsersCacheManager {
	orgUsersCacheOnce.Do(func() {
		orgUsersCache = &OrgUsersCacheManager{
			redis: GetRedisClient(),
			ttl:   configuration.Config.OrgUsersCacheTTL,
		}

		log.Info().
			Str("component", "org_users_cache").
			Dur("ttl", orgUsersCache.ttl).
			Msg("Organization users cache manager initialized")
	})
	return orgUsersCache
}

// Get retrieves organization users from cache
func (c *OrgUsersCacheManager) Get(orgID string) ([]models.LogtoUser, bool) {
	// Return cache miss if Redis is not available
	if c.redis == nil {
		return nil, false
	}

	key := fmt.Sprintf("org_users:%s", orgID)

	var cached OrgUsersCache
	err := c.redis.Get(key, &cached)
	if err != nil {
		if err == ErrCacheMiss {
			log.Debug().
				Str("component", "org_users_cache").
				Str("operation", "cache_miss").
				Str("org_id", orgID).
				Msg("Organization users cache miss")
			return nil, false
		}

		log.Error().
			Str("component", "org_users_cache").
			Str("operation", "get").
			Str("org_id", orgID).
			Err(err).
			Msg("Failed to get organization users from cache")
		return nil, false
	}

	// Check if expired (double check, Redis should handle TTL)
	if time.Now().After(cached.ExpiresAt) {
		log.Debug().
			Str("component", "org_users_cache").
			Str("operation", "cache_expired").
			Str("org_id", orgID).
			Time("expires_at", cached.ExpiresAt).
			Msg("Organization users cache entry expired")
		return nil, false
	}

	log.Debug().
		Str("component", "org_users_cache").
		Str("operation", "cache_hit").
		Str("org_id", orgID).
		Time("cached_at", cached.CachedAt).
		Time("expires_at", cached.ExpiresAt).
		Int("users_count", len(cached.Users)).
		Msg("Organization users cache hit")

	return cached.Users, true
}

// Set stores organization users in cache
func (c *OrgUsersCacheManager) Set(orgID string, users []models.LogtoUser) {
	// Skip if Redis is not available
	if c.redis == nil {
		return
	}

	key := fmt.Sprintf("org_users:%s", orgID)

	now := time.Now()
	cached := OrgUsersCache{
		Users:     users,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}

	err := c.redis.Set(key, cached, c.ttl)
	if err != nil {
		log.Error().
			Str("component", "org_users_cache").
			Str("operation", "set").
			Str("org_id", orgID).
			Err(err).
			Msg("Failed to set organization users in cache")
		return
	}

	log.Debug().
		Str("component", "org_users_cache").
		Str("operation", "cache_set").
		Str("org_id", orgID).
		Int("users_count", len(users)).
		Dur("ttl", c.ttl).
		Time("expires_at", now.Add(c.ttl)).
		Msg("Organization users cached")
}

// Clear removes entry from cache
func (c *OrgUsersCacheManager) Clear(orgID string) {
	// Skip if Redis is not available
	if c.redis == nil {
		return
	}

	key := fmt.Sprintf("org_users:%s", orgID)

	err := c.redis.Delete(key)
	if err != nil {
		log.Error().
			Str("component", "org_users_cache").
			Str("operation", "clear").
			Str("org_id", orgID).
			Err(err).
			Msg("Failed to clear organization users cache entry")
		return
	}

	log.Debug().
		Str("component", "org_users_cache").
		Str("operation", "cache_clear").
		Str("org_id", orgID).
		Msg("Organization users cache entry cleared")
}

// ClearAll removes all entries from cache
func (c *OrgUsersCacheManager) ClearAll() {
	// Skip if Redis is not available
	if c.redis == nil {
		return
	}

	pattern := "org_users:*"

	err := c.redis.DeletePattern(pattern)
	if err != nil {
		log.Error().
			Str("component", "org_users_cache").
			Str("operation", "clear_all").
			Str("pattern", pattern).
			Err(err).
			Msg("Failed to clear all organization users cache entries")
		return
	}

	log.Info().
		Str("component", "org_users_cache").
		Str("operation", "cache_clear_all").
		Str("pattern", pattern).
		Msg("All organization users cache entries cleared")
}

// GetStats returns cache statistics
func (c *OrgUsersCacheManager) GetStats() map[string]interface{} {
	// Return limited stats if Redis is not available
	if c.redis == nil {
		return map[string]interface{}{
			"redis_available": false,
			"ttl_minutes":     c.ttl.Minutes(),
			"cache_prefix":    "org_users:",
		}
	}

	// Get Redis client stats
	redisStats, err := c.redis.GetStats()
	if err != nil {
		log.Error().
			Str("component", "org_users_cache").
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
		"cache_prefix":    "org_users:",
	}

	return stats
}
