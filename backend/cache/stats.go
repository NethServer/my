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
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/models"
	"github.com/rs/zerolog/log"
)

// SystemStats represents aggregated statistics
type SystemStats struct {
	Distributors int       `json:"distributors"`
	Resellers    int       `json:"resellers"`
	Customers    int       `json:"customers"`
	Users        int       `json:"users"`
	Systems      int       `json:"systems"`
	LastUpdated  time.Time `json:"lastUpdated"`
	IsStale      bool      `json:"isStale"`
}

// LogtoClient interface for Logto operations
type LogtoClient interface {
	GetOrganizations() ([]models.LogtoOrganization, error)
	GetOrganizationUsers(ctx context.Context, orgID string) ([]models.LogtoUser, error)
	GetOrganizationUsersParallel(ctx context.Context, orgIDs []string) (map[string][]models.LogtoUser, error)
}

// StatsCacheManager manages system statistics caching with Redis
type StatsCacheManager struct {
	redis          RedisInterface
	cacheTTL       time.Duration
	updateTicker   *time.Ticker
	staleThreshold time.Duration
	stopChan       chan struct{}
	updateChan     chan struct{}
	isRunning      bool
	updateMutex    sync.Mutex
	logtoClient    LogtoClient
}

var (
	statsCacheManager     *StatsCacheManager
	statsCacheManagerOnce sync.Once
)

// GetStatsCacheManager returns singleton stats cache manager
func GetStatsCacheManager() *StatsCacheManager {
	statsCacheManagerOnce.Do(func() {
		statsCacheManager = &StatsCacheManager{
			redis:          GetRedisClient(),
			cacheTTL:       configuration.Config.StatsCacheTTL,
			staleThreshold: configuration.Config.StatsStaleThreshold,
			stopChan:       make(chan struct{}),
			updateChan:     make(chan struct{}, 1), // Buffered to prevent blocking
		}
	})
	return statsCacheManager
}

// InitAndStartStatsCacheManager initializes and starts stats cache with provided client
func InitAndStartStatsCacheManager(logtoClient LogtoClient) *StatsCacheManager {
	manager := GetStatsCacheManager()
	manager.SetLogtoClient(logtoClient)
	manager.StartBackgroundUpdater()
	return manager
}

// SetLogtoClient sets the Logto client for API operations
func (s *StatsCacheManager) SetLogtoClient(client LogtoClient) {
	s.logtoClient = client
}

// GetStats returns current cached statistics
func (s *StatsCacheManager) GetStats() *SystemStats {
	// Return empty stale stats if Redis is not available
	if s.redis == nil {
		return &SystemStats{
			LastUpdated: time.Time{},
			IsStale:     true,
		}
	}

	key := "stats:system"

	var stats SystemStats
	err := s.redis.Get(key, &stats)
	if err != nil {
		if err == ErrCacheMiss {
			log.Debug().
				Str("component", "stats_cache").
				Str("operation", "cache_miss").
				Msg("System stats cache miss")

			// Force immediate update if no cache exists
			s.triggerUpdate()

			// Return empty stats with stale flag
			return &SystemStats{
				LastUpdated: time.Time{},
				IsStale:     true,
			}
		}

		log.Error().
			Str("component", "stats_cache").
			Str("operation", "get_stats").
			Err(err).
			Msg("Failed to get system stats from cache")

		return &SystemStats{
			LastUpdated: time.Time{},
			IsStale:     true,
		}
	}

	// Check if stats are stale
	if time.Since(stats.LastUpdated) > s.staleThreshold {
		stats.IsStale = true
		log.Debug().
			Str("component", "stats_cache").
			Str("operation", "stale_detection").
			Time("last_updated", stats.LastUpdated).
			Dur("age", time.Since(stats.LastUpdated)).
			Dur("stale_threshold", s.staleThreshold).
			Msg("System stats are stale")
	}

	log.Debug().
		Str("component", "stats_cache").
		Str("operation", "cache_hit").
		Time("last_updated", stats.LastUpdated).
		Bool("is_stale", stats.IsStale).
		Msg("System stats cache hit")

	return &stats
}

// StartBackgroundUpdater starts the background update process
func (s *StatsCacheManager) StartBackgroundUpdater() {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	if s.isRunning {
		log.Warn().
			Str("component", "stats_cache").
			Str("operation", "start_background_updater").
			Msg("Background updater already running")
		return
	}

	s.isRunning = true
	s.updateTicker = time.NewTicker(configuration.Config.StatsUpdateInterval)

	go s.backgroundUpdateLoop()

	log.Info().
		Str("component", "stats_cache").
		Str("operation", "background_updater_started").
		Dur("update_interval", configuration.Config.StatsUpdateInterval).
		Dur("cache_ttl", s.cacheTTL).
		Dur("stale_threshold", s.staleThreshold).
		Msg("System stats background updater started")
}

// StopBackgroundUpdater stops the background update process
func (s *StatsCacheManager) StopBackgroundUpdater() {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	if !s.isRunning {
		return
	}

	s.isRunning = false
	close(s.stopChan)

	if s.updateTicker != nil {
		s.updateTicker.Stop()
		s.updateTicker = nil
	}

	// Recreate channels for potential restart
	s.stopChan = make(chan struct{})
	s.updateChan = make(chan struct{}, 1)

	log.Info().
		Str("component", "stats_cache").
		Str("operation", "background_updater_stopped").
		Msg("System stats background updater stopped")
}

// triggerUpdate triggers an immediate stats update
func (s *StatsCacheManager) triggerUpdate() {
	select {
	case s.updateChan <- struct{}{}:
		log.Debug().
			Str("component", "stats_cache").
			Str("operation", "trigger_update").
			Msg("Stats update triggered")
	default:
		log.Debug().
			Str("component", "stats_cache").
			Str("operation", "trigger_update_skipped").
			Msg("Stats update already pending")
	}
}

// backgroundUpdateLoop runs the background update process
func (s *StatsCacheManager) backgroundUpdateLoop() {
	// Initial update
	s.updateStats()

	for {
		select {
		case <-s.stopChan:
			log.Info().
				Str("component", "stats_cache").
				Str("operation", "background_loop_stopped").
				Msg("Background update loop stopped")
			return
		case <-s.updateTicker.C:
			s.updateStats()
		case <-s.updateChan:
			s.updateStats()
		}
	}
}

// updateStats fetches and caches fresh statistics
func (s *StatsCacheManager) updateStats() {
	if s.logtoClient == nil {
		log.Error().
			Str("component", "stats_cache").
			Str("operation", "update_stats").
			Msg("Logto client not set, cannot update stats")
		s.setStatsError("Logto client not configured")
		return
	}

	log.Debug().
		Str("component", "stats_cache").
		Str("operation", "update_stats_start").
		Msg("Starting stats update")

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Fetch organizations
	orgs, err := s.logtoClient.GetOrganizations()
	if err != nil {
		log.Error().
			Str("component", "stats_cache").
			Str("operation", "fetch_organizations").
			Err(err).
			Msg("Failed to fetch organizations")
		s.setStatsError(fmt.Sprintf("Failed to fetch organizations: %v", err))
		return
	}

	// Count organizations by type
	distributors := 0
	resellers := 0
	customers := 0
	totalUsers := 0
	totalSystems := 0

	orgIDs := make([]string, 0, len(orgs))
	for _, org := range orgs {
		orgIDs = append(orgIDs, org.ID)

		// Count by organization type using metadata or configurable logic
		orgType := s.getOrganizationType(org)
		switch orgType {
		case "distributor":
			distributors++
		case "reseller":
			resellers++
		case "customer":
			customers++
		}
	}

	// Fetch users for all organizations in parallel
	usersMap, err := s.logtoClient.GetOrganizationUsersParallel(ctx, orgIDs)
	if err != nil {
		log.Error().
			Str("component", "stats_cache").
			Str("operation", "fetch_users").
			Err(err).
			Msg("Failed to fetch users")
		s.setStatsError(fmt.Sprintf("Failed to fetch users: %v", err))
		return
	}

	// Count total users
	for _, users := range usersMap {
		totalUsers += len(users)
	}

	// Count systems - placeholder implementation
	// This should be replaced with actual system counting logic based on your business requirements
	totalSystems = s.countSystems(orgs, usersMap)

	// Create stats object
	stats := SystemStats{
		Distributors: distributors,
		Resellers:    resellers,
		Customers:    customers,
		Users:        totalUsers,
		Systems:      totalSystems,
		LastUpdated:  time.Now(),
		IsStale:      false,
	}

	// Cache the stats
	key := "stats:system"
	err = s.redis.Set(key, stats, s.cacheTTL)
	if err != nil {
		log.Error().
			Str("component", "stats_cache").
			Str("operation", "cache_stats").
			Err(err).
			Msg("Failed to cache system stats")
		s.setStatsError(fmt.Sprintf("Failed to cache stats: %v", err))
		return
	}

	log.Info().
		Str("component", "stats_cache").
		Str("operation", "stats_updated").
		Int("distributors", distributors).
		Int("resellers", resellers).
		Int("customers", customers).
		Int("users", totalUsers).
		Int("systems", totalSystems).
		Int("organizations_processed", len(orgs)).
		Dur("update_duration", time.Since(start)).
		Msg("System stats updated successfully")
}

// ClearCache removes all stats cache entries
func (s *StatsCacheManager) ClearCache() error {
	pattern := "stats:*"

	err := s.redis.DeletePattern(pattern)
	if err != nil {
		log.Error().
			Str("component", "stats_cache").
			Str("operation", "clear_cache").
			Str("pattern", pattern).
			Err(err).
			Msg("Failed to clear stats cache")
		return err
	}

	log.Info().
		Str("component", "stats_cache").
		Str("operation", "cache_cleared").
		Str("pattern", pattern).
		Msg("Stats cache cleared")

	return nil
}

// GetCacheStats returns cache statistics
func (s *StatsCacheManager) GetCacheStats() map[string]interface{} {
	// Get Redis client stats
	redisStats, err := s.redis.GetStats()
	if err != nil {
		log.Error().
			Str("component", "stats_cache").
			Str("operation", "get_cache_stats").
			Err(err).
			Msg("Failed to get Redis stats")
		return map[string]interface{}{
			"error":                   "failed to get Redis stats",
			"cache_ttl_minutes":       s.cacheTTL.Minutes(),
			"stale_threshold_minutes": s.staleThreshold.Minutes(),
		}
	}

	stats := map[string]interface{}{
		"cache_ttl_minutes":       s.cacheTTL.Minutes(),
		"stale_threshold_minutes": s.staleThreshold.Minutes(),
		"update_interval_minutes": configuration.Config.StatsUpdateInterval.Minutes(),
		"is_running":              s.isRunning,
		"redis_stats":             redisStats,
		"cache_prefix":            "stats:",
	}

	return stats
}

// setStatsError sets an error state in the stats cache
func (s *StatsCacheManager) setStatsError(errorMsg string) {
	if s.redis == nil {
		return
	}

	key := "stats:system:error"
	errorInfo := map[string]interface{}{
		"error":     errorMsg,
		"timestamp": time.Now(),
	}

	// Cache error for 5 minutes
	err := s.redis.Set(key, errorInfo, 5*time.Minute)
	if err != nil {
		log.Error().
			Str("component", "stats_cache").
			Str("operation", "set_error").
			Err(err).
			Msg("Failed to cache error state")
	}
}

// getOrganizationType determines the type of organization based on custom data
func (s *StatsCacheManager) getOrganizationType(org models.LogtoOrganization) string {
	// Check for explicit type field in custom data
	if org.CustomData != nil {
		if orgType, exists := org.CustomData["type"]; exists {
			if typeStr, ok := orgType.(string); ok {
				return typeStr
			}
		}
	}

	// Default to customer type if no type can be determined
	return "customer"
}

// countSystems counts systems based on organizations and users
// This is a placeholder implementation that should be replaced with actual business logic
func (s *StatsCacheManager) countSystems(orgs []models.LogtoOrganization, usersMap map[string][]models.LogtoUser) int {
	// Placeholder implementation - replace with actual system counting logic
	// For example:
	// - Count based on custom data in organizations
	// - Count based on user roles or permissions
	// - Count based on external system registry

	systemCount := 0

	// Example: Count systems based on organization custom data
	for _, org := range orgs {
		if org.CustomData != nil {
			if systems, exists := org.CustomData["systems"]; exists {
				if systemsSlice, ok := systems.([]interface{}); ok {
					systemCount += len(systemsSlice)
				}
			}
		}
	}

	// If no systems found in metadata, use a basic formula as fallback
	if systemCount == 0 {
		// Example: Assume 1 system per 10 users as a rough estimate
		totalUsers := 0
		for _, users := range usersMap {
			totalUsers += len(users)
		}
		systemCount = (totalUsers + 9) / 10 // Round up division
	}

	return systemCount
}
