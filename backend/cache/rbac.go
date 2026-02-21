/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package cache

import (
	"sync"
	"time"

	"github.com/nethesis/my/backend/logger"
)

const rbacCacheTTL = 5 * time.Minute

// RBACCacheEntry holds cached RBAC data for a role+org combination
type RBACCacheEntry struct {
	OrgIDs    []string  `json:"org_ids"`
	SystemIDs []string  `json:"system_ids"`
	CachedAt  time.Time `json:"cached_at"`
}

// RBACCache provides two-tier caching (in-memory + Redis) for RBAC org/system ID lookups
type RBACCache struct {
	memCache map[string]*RBACCacheEntry
	mutex    sync.RWMutex
}

var (
	rbacCache     *RBACCache
	rbacCacheOnce sync.Once
)

// GetRBACCache returns the singleton RBAC cache instance
func GetRBACCache() *RBACCache {
	rbacCacheOnce.Do(func() {
		rbacCache = &RBACCache{
			memCache: make(map[string]*RBACCacheEntry),
		}
	})
	return rbacCache
}

func rbacKey(role, orgID string) string {
	return "rbac:" + role + ":" + orgID
}

// GetOrgIDs retrieves cached organization IDs for a role+org combination
func (c *RBACCache) GetOrgIDs(role, orgID string) ([]string, bool) {
	key := rbacKey(role, orgID)

	// Check in-memory first
	c.mutex.RLock()
	entry, ok := c.memCache[key]
	c.mutex.RUnlock()

	if ok && time.Since(entry.CachedAt) < rbacCacheTTL {
		return entry.OrgIDs, true
	}

	// Check Redis
	rc := GetRedisClient()
	if rc != nil {
		var redisEntry RBACCacheEntry
		err := rc.Get(key, &redisEntry)
		if err == nil && time.Since(redisEntry.CachedAt) < rbacCacheTTL {
			// Populate in-memory cache from Redis
			c.mutex.Lock()
			c.memCache[key] = &redisEntry
			c.mutex.Unlock()
			return redisEntry.OrgIDs, true
		}
	}

	return nil, false
}

// GetSystemIDs retrieves cached system IDs for a role+org combination
func (c *RBACCache) GetSystemIDs(role, orgID string) ([]string, bool) {
	key := rbacKey(role, orgID)

	// Check in-memory first
	c.mutex.RLock()
	entry, ok := c.memCache[key]
	c.mutex.RUnlock()

	if ok && entry.SystemIDs != nil && time.Since(entry.CachedAt) < rbacCacheTTL {
		return entry.SystemIDs, true
	}

	// Check Redis
	rc := GetRedisClient()
	if rc != nil {
		var redisEntry RBACCacheEntry
		err := rc.Get(key, &redisEntry)
		if err == nil && redisEntry.SystemIDs != nil && time.Since(redisEntry.CachedAt) < rbacCacheTTL {
			c.mutex.Lock()
			c.memCache[key] = &redisEntry
			c.mutex.Unlock()
			return redisEntry.SystemIDs, true
		}
	}

	return nil, false
}

// SetOrgIDs stores organization IDs in both in-memory and Redis cache
func (c *RBACCache) SetOrgIDs(role, orgID string, orgIDs []string) {
	key := rbacKey(role, orgID)
	now := time.Now()

	c.mutex.Lock()
	entry, exists := c.memCache[key]
	if !exists {
		entry = &RBACCacheEntry{}
		c.memCache[key] = entry
	}
	entry.OrgIDs = orgIDs
	entry.CachedAt = now
	c.mutex.Unlock()

	// Store in Redis
	rc := GetRedisClient()
	if rc != nil {
		if err := rc.Set(key, entry, rbacCacheTTL); err != nil {
			logger.ComponentLogger("rbac_cache").Debug().Err(err).Msg("Failed to set RBAC cache in Redis")
		}
	}
}

// SetSystemIDs stores system IDs in both in-memory and Redis cache
func (c *RBACCache) SetSystemIDs(role, orgID string, systemIDs []string) {
	key := rbacKey(role, orgID)
	now := time.Now()

	c.mutex.Lock()
	entry, exists := c.memCache[key]
	if !exists {
		entry = &RBACCacheEntry{}
		c.memCache[key] = entry
	}
	entry.SystemIDs = systemIDs
	entry.CachedAt = now
	c.mutex.Unlock()

	// Store in Redis
	rc := GetRedisClient()
	if rc != nil {
		if err := rc.Set(key, entry, rbacCacheTTL); err != nil {
			logger.ComponentLogger("rbac_cache").Debug().Err(err).Msg("Failed to set RBAC cache in Redis")
		}
	}
}

// InvalidateAll clears all RBAC cache entries (both in-memory and Redis)
func (c *RBACCache) InvalidateAll() {
	c.mutex.Lock()
	c.memCache = make(map[string]*RBACCacheEntry)
	c.mutex.Unlock()

	// Clear Redis RBAC keys
	rc := GetRedisClient()
	if rc != nil {
		if err := rc.DeletePattern("rbac:*"); err != nil {
			logger.ComponentLogger("rbac_cache").Debug().Err(err).Msg("Failed to invalidate RBAC cache in Redis")
		}
	}

	logger.ComponentLogger("rbac_cache").Debug().Msg("RBAC cache invalidated")
}

// Invalidate clears RBAC cache entries for a specific role+org (both in-memory and Redis)
func (c *RBACCache) Invalidate(role, orgID string) {
	key := rbacKey(role, orgID)

	c.mutex.Lock()
	delete(c.memCache, key)
	c.mutex.Unlock()

	rc := GetRedisClient()
	if rc != nil {
		if err := rc.Delete(key); err != nil {
			logger.ComponentLogger("rbac_cache").Debug().Err(err).Msg("Failed to invalidate RBAC cache entry in Redis")
		}
	}
}
