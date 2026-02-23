/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cache

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/nethesis/my/backend/logger"
)

const appsCacheTTL = 30 * time.Second

// AppsCacheEntry holds cached data for a specific endpoint+role+org combination
type AppsCacheEntry struct {
	Data     json.RawMessage `json:"data"`
	CachedAt time.Time       `json:"cached_at"`
}

// AppsCache provides two-tier caching (in-memory + Redis) for applications aggregated data
type AppsCache struct {
	memCache map[string]*AppsCacheEntry
	mutex    sync.RWMutex
}

var (
	appsCache     *AppsCache
	appsCacheOnce sync.Once
)

// GetAppsCache returns the singleton applications cache instance
func GetAppsCache() *AppsCache {
	appsCacheOnce.Do(func() {
		appsCache = &AppsCache{
			memCache: make(map[string]*AppsCacheEntry),
		}
	})
	return appsCache
}

func appsCacheKey(prefix, role, orgID string) string {
	return "apps:" + prefix + ":" + role + ":" + orgID
}

// Get retrieves cached data for a given key, unmarshaling into dest
func (c *AppsCache) Get(prefix, role, orgID string, dest interface{}) bool {
	key := appsCacheKey(prefix, role, orgID)

	// Check in-memory first
	c.mutex.RLock()
	entry, ok := c.memCache[key]
	c.mutex.RUnlock()

	if ok && time.Since(entry.CachedAt) < appsCacheTTL {
		if err := json.Unmarshal(entry.Data, dest); err == nil {
			return true
		}
	}

	// Check Redis
	rc := GetRedisClient()
	if rc != nil {
		var redisEntry AppsCacheEntry
		err := rc.Get(key, &redisEntry)
		if err == nil && time.Since(redisEntry.CachedAt) < appsCacheTTL {
			// Populate in-memory cache from Redis
			c.mutex.Lock()
			c.memCache[key] = &redisEntry
			c.mutex.Unlock()
			if err := json.Unmarshal(redisEntry.Data, dest); err == nil {
				return true
			}
		}
	}

	return false
}

// Set stores data in both in-memory and Redis cache
func (c *AppsCache) Set(prefix, role, orgID string, value interface{}) {
	key := appsCacheKey(prefix, role, orgID)

	data, err := json.Marshal(value)
	if err != nil {
		logger.ComponentLogger("apps_cache").Debug().Err(err).Msg("Failed to marshal cache value")
		return
	}

	entry := &AppsCacheEntry{
		Data:     data,
		CachedAt: time.Now(),
	}

	c.mutex.Lock()
	c.memCache[key] = entry
	c.mutex.Unlock()

	// Store in Redis
	rc := GetRedisClient()
	if rc != nil {
		if err := rc.Set(key, entry, appsCacheTTL); err != nil {
			logger.ComponentLogger("apps_cache").Debug().Err(err).Msg("Failed to set apps cache in Redis")
		}
	}
}

// InvalidateAll clears all applications cache entries (both in-memory and Redis)
func (c *AppsCache) InvalidateAll() {
	c.mutex.Lock()
	c.memCache = make(map[string]*AppsCacheEntry)
	c.mutex.Unlock()

	// Clear Redis apps keys
	rc := GetRedisClient()
	if rc != nil {
		if err := rc.DeletePattern("apps:*"); err != nil {
			logger.ComponentLogger("apps_cache").Debug().Err(err).Msg("Failed to invalidate apps cache in Redis")
		}
	}

	logger.ComponentLogger("apps_cache").Debug().Msg("Applications cache invalidated")
}
