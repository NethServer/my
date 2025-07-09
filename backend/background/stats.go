/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package background

import (
	"context"
	"sync"
	"time"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
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

// StatsCacheManager manages system statistics caching
type StatsCacheManager struct {
	stats       *SystemStats
	mutex       sync.RWMutex
	stopChan    chan struct{}
	updateChan  chan struct{}
	isRunning   bool
	updateMutex sync.Mutex
	logtoClient LogtoClient
}

var (
	statsCacheManager     *StatsCacheManager
	statsCacheManagerOnce sync.Once
)

// Cache TTL constants are now configured via environment variables
// See configuration.go for defaults

// GetStatsCacheManager returns singleton stats cache manager
func GetStatsCacheManager() *StatsCacheManager {
	statsCacheManagerOnce.Do(func() {
		statsCacheManager = &StatsCacheManager{
			stats: &SystemStats{
				LastUpdated: time.Time{}, // Zero time indicates no data
				IsStale:     true,
			},
			stopChan:   make(chan struct{}),
			updateChan: make(chan struct{}, 1), // Buffered to prevent blocking
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

// GetStats returns current cached statistics
func (s *StatsCacheManager) GetStats() *SystemStats {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Check if stats are stale
	if time.Since(s.stats.LastUpdated) > configuration.Config.StatsStaleThreshold {
		s.stats.IsStale = true
	}

	// Return a copy to avoid race conditions
	statsCopy := *s.stats
	return &statsCopy
}

// SetStats updates cached statistics
func (s *StatsCacheManager) SetStats(newStats *SystemStats) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	newStats.LastUpdated = time.Now()
	newStats.IsStale = false
	s.stats = newStats
}

// TriggerUpdate requests an immediate stats update
func (s *StatsCacheManager) TriggerUpdate() {
	select {
	case s.updateChan <- struct{}{}:
		// Update triggered successfully
	default:
		// Update already pending, skip
	}
}

// StartBackgroundUpdater starts the background statistics updater
func (s *StatsCacheManager) StartBackgroundUpdater() {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	if s.isRunning {
		return // Already running
	}

	s.isRunning = true
	go s.backgroundUpdater()

	logger.Info().
		Dur("update_interval", configuration.Config.StatsUpdateInterval).
		Dur("cache_ttl", configuration.Config.StatsCacheTTL).
		Msg("Statistics cache background updater started")
}

// StopBackgroundUpdater stops the background statistics updater
func (s *StatsCacheManager) StopBackgroundUpdater() {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	if !s.isRunning {
		return // Not running
	}

	close(s.stopChan)
	s.isRunning = false

	logger.Info().Msg("Statistics cache background updater stopped")
}

// backgroundUpdater runs the background update loop
func (s *StatsCacheManager) backgroundUpdater() {
	ticker := time.NewTicker(configuration.Config.StatsUpdateInterval)
	defer ticker.Stop()

	// Initial update
	s.updateStatsAsync()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.updateStatsAsync()
		case <-s.updateChan:
			s.updateStatsAsync()
		}
	}
}

// SetLogtoClient sets the Logto client for the stats manager
func (s *StatsCacheManager) SetLogtoClient(client LogtoClient) {
	s.logtoClient = client
}

// updateStatsAsync performs async statistics update with error handling
func (s *StatsCacheManager) updateStatsAsync() {
	if s.logtoClient == nil {
		logger.Error().Msg("Logto client not set for stats manager")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()
	logger.Debug().Msg("Starting background statistics update")

	newStats, err := s.calculateStatsWithContext(ctx, s.logtoClient)
	if err != nil {
		logger.Error().
			Err(err).
			Dur("duration", time.Since(startTime)).
			Msg("Failed to update system statistics")
		return
	}

	s.SetStats(newStats)

	logger.Info().
		Int("distributors", newStats.Distributors).
		Int("resellers", newStats.Resellers).
		Int("customers", newStats.Customers).
		Int("users", newStats.Users).
		Int("systems", newStats.Systems).
		Dur("duration", time.Since(startTime)).
		Msg("System statistics updated successfully")
}

// LogtoClient interface to avoid import cycle
type LogtoClient interface {
	GetOrganizationsPaginated(page, pageSize int, filters models.OrganizationFilters) (*models.PaginatedOrganizations, error)
	GetOrganizationUsers(orgID string) ([]models.LogtoUser, error)
}

// calculateStatsWithContext calculates statistics with context cancellation
func (s *StatsCacheManager) calculateStatsWithContext(ctx context.Context, client LogtoClient) (*SystemStats, error) {
	// Get organizations with pagination and streaming
	orgStats, totalUsers, err := s.calculateOrganizationStats(ctx, client)
	if err != nil {
		return nil, err
	}

	// Systems count (using demo systems storage)
	systemsCount := 0 // Systems count not implemented yet

	return &SystemStats{
		Distributors: orgStats.Distributors,
		Resellers:    orgStats.Resellers,
		Customers:    orgStats.Customers,
		Users:        totalUsers,
		Systems:      systemsCount,
	}, nil
}

// OrganizationStats represents organization type counts
type OrganizationStats struct {
	Distributors int
	Resellers    int
	Customers    int
}

// calculateOrganizationStats efficiently calculates organization statistics
func (s *StatsCacheManager) calculateOrganizationStats(ctx context.Context, client LogtoClient) (*OrganizationStats, int, error) {
	stats := &OrganizationStats{}
	totalUsers := 0
	page := 1
	pageSize := 100 // Larger page size for efficiency

	for {
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		default:
		}

		// Get organizations with pagination
		result, err := client.GetOrganizationsPaginated(page, pageSize, models.OrganizationFilters{})
		if err != nil {
			return nil, 0, err
		}

		// Process this batch
		for _, org := range result.Data {
			if org.CustomData != nil {
				if orgType, ok := org.CustomData["type"].(string); ok {
					switch orgType {
					case "distributor":
						stats.Distributors++
					case "reseller":
						stats.Resellers++
					case "customer":
						stats.Customers++
					}
				}
			}

			// Count users in organization (if needed for user stats)
			// This could be expensive - consider making it optional
			if ctx.Err() == nil {
				if users, err := client.GetOrganizationUsers(org.ID); err == nil {
					totalUsers += len(users)
				}
			}
		}

		// Check if we have more pages
		if !result.Pagination.HasNext {
			break
		}

		page++

		// Yield to prevent blocking
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		default:
		}
	}

	return stats, totalUsers, nil
}

// CalculateStatsRealTime calculates statistics in real-time (fallback)
// NOTE: This function should be called from services package with a client
func CalculateStatsRealTime(client LogtoClient) (*SystemStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	manager := GetStatsCacheManager()
	return manager.calculateStatsWithContext(ctx, client)
}
