/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package services

import (
	"sync"
	"time"

	"github.com/nethesis/my/backend/logger"
)

// JitRolesCacheManager manages in-memory cache for JIT roles
type JitRolesCacheManager struct {
	cache map[string]JitRolesCache
	mutex sync.RWMutex
	ttl   time.Duration
}

// OrgUsersCacheManager manages in-memory cache for organization users
type OrgUsersCacheManager struct {
	cache map[string]OrgUsersCache
	mutex sync.RWMutex
	ttl   time.Duration
}

// Global cache instances
var jitRolesCache *JitRolesCacheManager
var orgUsersCache *OrgUsersCacheManager
var cacheOnce sync.Once
var orgUsersCacheOnce sync.Once

// GetJitRolesCacheManager returns singleton cache instance
func GetJitRolesCacheManager() *JitRolesCacheManager {
	cacheOnce.Do(func() {
		jitRolesCache = &JitRolesCacheManager{
			cache: make(map[string]JitRolesCache),
			ttl:   5 * time.Minute, // 5 minute TTL
		}

		// Start cleanup routine
		go jitRolesCache.startCleanup()
	})
	return jitRolesCache
}

// Get retrieves JIT roles from cache
func (c *JitRolesCacheManager) Get(orgID string) ([]LogtoOrganizationRole, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	cached, exists := c.cache[orgID]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(cached.ExpiresAt) {
		logger.ComponentLogger("cache").Debug().
			Str("operation", "cache_expired").
			Str("org_id", orgID).
			Msg("JIT roles cache entry expired")
		return nil, false
	}

	logger.ComponentLogger("cache").Debug().
		Str("operation", "cache_hit").
		Str("org_id", orgID).
		Int("roles_count", len(cached.Roles)).
		Msg("JIT roles cache hit")

	return cached.Roles, true
}

// Set stores JIT roles in cache
func (c *JitRolesCacheManager) Set(orgID string, roles []LogtoOrganizationRole) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	c.cache[orgID] = JitRolesCache{
		Roles:     roles,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}

	logger.ComponentLogger("cache").Debug().
		Str("operation", "cache_set").
		Str("org_id", orgID).
		Int("roles_count", len(roles)).
		Time("expires_at", now.Add(c.ttl)).
		Msg("JIT roles cached")
}

// Clear removes entry from cache
func (c *JitRolesCacheManager) Clear(orgID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.cache, orgID)

	logger.ComponentLogger("cache").Debug().
		Str("operation", "cache_clear").
		Str("org_id", orgID).
		Msg("JIT roles cache entry cleared")
}

// ClearAll removes all entries from cache
func (c *JitRolesCacheManager) ClearAll() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	count := len(c.cache)
	c.cache = make(map[string]JitRolesCache)

	logger.ComponentLogger("cache").Info().
		Str("operation", "cache_clear_all").
		Int("cleared_entries", count).
		Msg("All JIT roles cache entries cleared")
}

// GetStats returns cache statistics
func (c *JitRolesCacheManager) GetStats() map[string]interface{} {
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
func (c *JitRolesCacheManager) startCleanup() {
	ticker := time.NewTicker(2 * time.Minute) // Clean every 2 minutes
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired entries
func (c *JitRolesCacheManager) cleanup() {
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
			Str("operation", "cache_cleanup").
			Int("removed_entries", removed).
			Int("remaining_entries", len(c.cache)).
			Msg("Cleaned up expired JIT roles cache entries")
	}
}

// GetOrgUsersCacheManager returns singleton cache instance for organization users
func GetOrgUsersCacheManager() *OrgUsersCacheManager {
	orgUsersCacheOnce.Do(func() {
		orgUsersCache = &OrgUsersCacheManager{
			cache: make(map[string]OrgUsersCache),
			ttl:   3 * time.Minute, // 3 minute TTL (shorter than JIT roles)
		}

		// Start cleanup routine
		go orgUsersCache.startCleanup()
	})
	return orgUsersCache
}

// Get retrieves organization users from cache
func (c *OrgUsersCacheManager) Get(orgID string) ([]LogtoUser, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	cached, exists := c.cache[orgID]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(cached.ExpiresAt) {
		logger.ComponentLogger("cache").Debug().
			Str("operation", "orgusers_cache_expired").
			Str("org_id", orgID).
			Msg("Organization users cache entry expired")
		return nil, false
	}

	logger.ComponentLogger("cache").Debug().
		Str("operation", "orgusers_cache_hit").
		Str("org_id", orgID).
		Int("users_count", len(cached.Users)).
		Msg("Organization users cache hit")

	return cached.Users, true
}

// Set stores organization users in cache
func (c *OrgUsersCacheManager) Set(orgID string, users []LogtoUser) {
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
		Time("expires_at", now.Add(c.ttl)).
		Msg("Organization users cached")
}

// Clear removes entry from cache
func (c *OrgUsersCacheManager) Clear(orgID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.cache, orgID)

	logger.ComponentLogger("cache").Debug().
		Str("operation", "orgusers_cache_clear").
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
		Str("operation", "orgusers_cache_clear_all").
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
	ticker := time.NewTicker(1 * time.Minute) // Clean every minute
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
