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
	"sync"
	"time"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// OrgUsersCacheManager manages in-memory cache for organization users
type OrgUsersCacheManager struct {
	cache map[string]OrgUsersCache
	mutex sync.RWMutex
	ttl   time.Duration
}

// Global cache instances
var orgUsersCache *OrgUsersCacheManager
var orgUsersCacheOnce sync.Once

// GetOrgUsersCacheManager returns singleton cache instance for organization users
func GetOrgUsersCacheManager() *OrgUsersCacheManager {
	orgUsersCacheOnce.Do(func() {
		orgUsersCache = &OrgUsersCacheManager{
			cache: make(map[string]OrgUsersCache),
			ttl:   configuration.Config.OrgUsersCacheTTL,
		}

		// Start cleanup routine
		go orgUsersCache.startCleanup()

		logger.Info().
			Dur("ttl", orgUsersCache.ttl).
			Msg("Organization users cache manager initialized")
	})
	return orgUsersCache
}

// Get retrieves organization users from cache
func (c *OrgUsersCacheManager) Get(orgID string) ([]models.LogtoUser, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	cached, exists := c.cache[orgID]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(cached.ExpiresAt) {
		return nil, false
	}

	logger.ComponentLogger("cache").Debug().
		Str("operation", "orgusers_cache_hit").
		Str("org_id", orgID).
		Time("cached_at", cached.CachedAt).
		Time("expires_at", cached.ExpiresAt).
		Int("users_count", len(cached.Users)).
		Msg("Organization users cache hit")

	return cached.Users, true
}

// Set stores organization users in cache
func (c *OrgUsersCacheManager) Set(orgID string, users []models.LogtoUser) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	c.cache[orgID] = OrgUsersCache{
		Users:     users,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}

	logger.ComponentLogger("cache").Debug().
		Str("operation", "orgusers_cache_set").
		Str("org_id", orgID).
		Int("users_count", len(users)).
		Dur("ttl", c.ttl).
		Time("expires_at", now.Add(c.ttl)).
		Msg("Organization users cached")
}

// Clear removes entry from cache
func (c *OrgUsersCacheManager) Clear(orgID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.cache, orgID)

	logger.ComponentLogger("cache").Debug().
		Str("operation", "cache_clear").
		Str("org_id", orgID).
		Msg("Organization users cache entry cleared")
}

// ClearAll removes all entries from cache
func (c *OrgUsersCacheManager) ClearAll() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	count := len(c.cache)
	c.cache = make(map[string]OrgUsersCache)

	logger.ComponentLogger("cache").Info().
		Str("operation", "cache_clear_all").
		Int("cleared_entries", count).
		Msg("All organization users cache entries cleared")
}

// GetStats returns cache statistics
func (c *OrgUsersCacheManager) GetStats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	total := len(c.cache)
	expired := 0
	now := time.Now()

	for _, cached := range c.cache {
		if now.After(cached.ExpiresAt) {
			expired++
		}
	}

	return map[string]interface{}{
		"total_entries":   total,
		"expired_entries": expired,
		"active_entries":  total - expired,
		"ttl_minutes":     c.ttl.Minutes(),
	}
}

// startCleanup runs periodic cleanup of expired entries
func (c *OrgUsersCacheManager) startCleanup() {
	ticker := time.NewTicker(configuration.Config.OrgUsersCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired entries
func (c *OrgUsersCacheManager) cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	removed := 0

	for orgID, cached := range c.cache {
		if now.After(cached.ExpiresAt) {
			delete(c.cache, orgID)
			removed++
		}
	}

	if removed > 0 {
		logger.ComponentLogger("cache").Info().
			Str("operation", "orgusers_cache_cleanup").
			Int("removed_entries", removed).
			Int("remaining_entries", len(c.cache)).
			Msg("Cleaned up expired organization users cache entries")
	}
}
