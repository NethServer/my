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
	"strings"
	"sync"
	"time"

	"github.com/nethesis/my/backend/configuration"
	"github.com/rs/zerolog/log"
)

// OrganizationNamesCache represents cached organization names data
type OrganizationNamesCache struct {
	Names     map[string]string `json:"names"` // map[lowercaseName]originalName
	CachedAt  time.Time         `json:"cached_at"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// OrganizationNamesCacheManager manages Redis cache for organization names
type OrganizationNamesCacheManager struct {
	redis RedisInterface
	ttl   time.Duration
}

// Global cache instances
var organizationNamesCache *OrganizationNamesCacheManager
var organizationNamesCacheOnce sync.Once

// GetOrganizationNamesCacheManager returns singleton cache instance
func GetOrganizationNamesCacheManager() *OrganizationNamesCacheManager {
	organizationNamesCacheOnce.Do(func() {
		organizationNamesCache = &OrganizationNamesCacheManager{
			redis: GetRedisClient(),
			ttl:   configuration.Config.OrganizationNamesCacheTTL,
		}

		if organizationNamesCache.redis == nil {
			log.Warn().
				Str("component", "organization_names_cache").
				Msg("Organization names cache manager initialized without Redis - cache will be disabled")
		} else {
			log.Info().
				Str("component", "organization_names_cache").
				Dur("ttl", organizationNamesCache.ttl).
				Msg("Organization names cache manager initialized")
		}
	})
	return organizationNamesCache
}

// Get retrieves organization names from cache
func (c *OrganizationNamesCacheManager) Get() (map[string]string, bool) {
	// Return cache miss if Redis is not available
	if c.redis == nil {
		return nil, false
	}

	key := "organization_names"

	var cached OrganizationNamesCache
	err := c.redis.Get(key, &cached)
	if err != nil {
		if err == ErrCacheMiss {
			log.Debug().
				Str("component", "organization_names_cache").
				Str("operation", "cache_miss").
				Msg("Organization names cache miss")
			return nil, false
		}

		log.Error().
			Str("component", "organization_names_cache").
			Str("operation", "get").
			Err(err).
			Msg("Failed to get organization names from cache")
		return nil, false
	}

	// Check if expired (double check, Redis should handle TTL)
	if time.Now().After(cached.ExpiresAt) {
		log.Debug().
			Str("component", "organization_names_cache").
			Str("operation", "cache_expired").
			Time("expires_at", cached.ExpiresAt).
			Msg("Organization names cache entry expired")
		return nil, false
	}

	log.Debug().
		Str("component", "organization_names_cache").
		Str("operation", "cache_hit").
		Time("cached_at", cached.CachedAt).
		Time("expires_at", cached.ExpiresAt).
		Int("names_count", len(cached.Names)).
		Msg("Organization names cache hit")

	return cached.Names, true
}

// Set stores organization names in cache
func (c *OrganizationNamesCacheManager) Set(names map[string]string) {
	// Skip if Redis is not available
	if c.redis == nil {
		return
	}

	key := "organization_names"

	now := time.Now()
	cached := OrganizationNamesCache{
		Names:     names,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}

	err := c.redis.Set(key, cached, c.ttl)
	if err != nil {
		log.Error().
			Str("component", "organization_names_cache").
			Str("operation", "set").
			Err(err).
			Msg("Failed to set organization names in cache")
		return
	}

	log.Debug().
		Str("component", "organization_names_cache").
		Str("operation", "cache_set").
		Int("names_count", len(names)).
		Dur("ttl", c.ttl).
		Time("expires_at", now.Add(c.ttl)).
		Msg("Organization names cached")
}

// AddName adds a single organization name to cache
func (c *OrganizationNamesCacheManager) AddName(name string) {
	// Skip if Redis is not available
	if c.redis == nil {
		return
	}

	// Get current cache
	names, found := c.Get()
	if !found {
		// If cache miss, create new map
		names = make(map[string]string)
	}

	// Add the new name
	names[strings.ToLower(name)] = name

	// Update cache
	c.Set(names)

	log.Debug().
		Str("component", "organization_names_cache").
		Str("operation", "add_name").
		Str("name", name).
		Msg("Organization name added to cache")
}

// RemoveName removes a single organization name from cache
func (c *OrganizationNamesCacheManager) RemoveName(name string) {
	// Skip if Redis is not available
	if c.redis == nil {
		return
	}

	// Get current cache
	names, found := c.Get()
	if !found {
		// If cache miss, nothing to remove
		return
	}

	// Remove the name
	delete(names, strings.ToLower(name))

	// Update cache
	c.Set(names)

	log.Debug().
		Str("component", "organization_names_cache").
		Str("operation", "remove_name").
		Str("name", name).
		Msg("Organization name removed from cache")
}

// IsNameTaken checks if a name is already taken (case-insensitive)
func (c *OrganizationNamesCacheManager) IsNameTaken(name string) (bool, string) {
	// Skip if Redis is not available
	if c.redis == nil {
		return false, ""
	}

	names, found := c.Get()
	if !found {
		// Cache miss, can't determine
		return false, ""
	}

	lowercaseName := strings.ToLower(name)
	originalName, exists := names[lowercaseName]
	return exists, originalName
}

// Clear removes all entries from cache
func (c *OrganizationNamesCacheManager) Clear() {
	// Skip if Redis is not available
	if c.redis == nil {
		return
	}

	key := "organization_names"

	err := c.redis.Delete(key)
	if err != nil {
		log.Error().
			Str("component", "organization_names_cache").
			Str("operation", "clear").
			Err(err).
			Msg("Failed to clear organization names cache entry")
		return
	}

	log.Debug().
		Str("component", "organization_names_cache").
		Str("operation", "cache_clear").
		Msg("Organization names cache entry cleared")
}

// GetStats returns cache statistics
func (c *OrganizationNamesCacheManager) GetStats() map[string]interface{} {
	// Return limited stats if Redis is not available
	if c.redis == nil {
		return map[string]interface{}{
			"redis_available": false,
			"ttl_minutes":     c.ttl.Minutes(),
			"cache_key":       "organization_names",
		}
	}

	// Get Redis client stats
	redisStats, err := c.redis.GetStats()
	if err != nil {
		log.Error().
			Str("component", "organization_names_cache").
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
		"cache_key":       "organization_names",
	}

	return stats
}
