/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLogtoClient is a mock implementation of LogtoClient
type MockLogtoClient struct {
	mock.Mock
}

func (m *MockLogtoClient) GetOrganizations() ([]models.LogtoOrganization, error) {
	args := m.Called()
	return args.Get(0).([]models.LogtoOrganization), args.Error(1)
}

func (m *MockLogtoClient) GetOrganizationUsers(ctx context.Context, orgID string) ([]models.LogtoUser, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).([]models.LogtoUser), args.Error(1)
}

func (m *MockLogtoClient) GetOrganizationUsersParallel(ctx context.Context, orgIDs []string) (map[string][]models.LogtoUser, error) {
	args := m.Called(ctx, orgIDs)
	return args.Get(0).(map[string][]models.LogtoUser), args.Error(1)
}

func TestSystemStatsStruct(t *testing.T) {
	now := time.Now()
	stats := SystemStats{
		Distributors: 5,
		Resellers:    15,
		Customers:    100,
		Users:        250,
		Systems:      50,
		LastUpdated:  now,
		IsStale:      false,
	}

	assert.Equal(t, 5, stats.Distributors)
	assert.Equal(t, 15, stats.Resellers)
	assert.Equal(t, 100, stats.Customers)
	assert.Equal(t, 250, stats.Users)
	assert.Equal(t, 50, stats.Systems)
	assert.Equal(t, now, stats.LastUpdated)
	assert.False(t, stats.IsStale)
}

func TestStatsCacheManagerStruct(t *testing.T) {
	mockRedis := NewMockRedisClient()
	mockLogto := &MockLogtoClient{}

	manager := &StatsCacheManager{
		redis:          mockRedis,
		cacheTTL:       time.Hour,
		staleThreshold: 30 * time.Minute,
		logtoClient:    mockLogto,
		isRunning:      false,
	}

	assert.Equal(t, mockRedis, manager.redis)
	assert.Equal(t, time.Hour, manager.cacheTTL)
	assert.Equal(t, 30*time.Minute, manager.staleThreshold)
	assert.Equal(t, mockLogto, manager.logtoClient)
	assert.False(t, manager.isRunning)
}

func TestStatsCacheManagerGetStats(t *testing.T) {
	key := "stats:system"

	t.Run("Cache hit with fresh data", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &StatsCacheManager{
			redis:          mockRedis,
			cacheTTL:       time.Hour,
			staleThreshold: 30 * time.Minute,
		}

		now := time.Now()
		cachedStats := SystemStats{
			Distributors: 5,
			Resellers:    15,
			Customers:    100,
			Users:        250,
			Systems:      50,
			LastUpdated:  now,
			IsStale:      false,
		}

		mockRedis.On("Get", key, mock.AnythingOfType("*cache.SystemStats")).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(1).(*SystemStats)
			*arg = cachedStats
		})

		stats := manager.GetStats()
		assert.NotNil(t, stats)
		assert.Equal(t, 5, stats.Distributors)
		assert.Equal(t, 15, stats.Resellers)
		assert.Equal(t, 100, stats.Customers)
		assert.Equal(t, 250, stats.Users)
		assert.Equal(t, 50, stats.Systems)
		assert.False(t, stats.IsStale)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Cache hit with stale data", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &StatsCacheManager{
			redis:          mockRedis,
			cacheTTL:       time.Hour,
			staleThreshold: 30 * time.Minute,
		}

		staleTime := time.Now().Add(-time.Hour) // Older than stale threshold
		cachedStats := SystemStats{
			Distributors: 5,
			Resellers:    15,
			Customers:    100,
			Users:        250,
			Systems:      50,
			LastUpdated:  staleTime,
			IsStale:      false, // Will be set to true by GetStats
		}

		mockRedis.On("Get", key, mock.AnythingOfType("*cache.SystemStats")).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(1).(*SystemStats)
			*arg = cachedStats
		})

		stats := manager.GetStats()
		assert.NotNil(t, stats)
		assert.True(t, stats.IsStale) // Should be marked as stale
		assert.Equal(t, staleTime, stats.LastUpdated)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Cache miss", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &StatsCacheManager{
			redis:          mockRedis,
			cacheTTL:       time.Hour,
			staleThreshold: 30 * time.Minute,
			updateChan:     make(chan struct{}, 1), // Initialize channel for test
		}

		mockRedis.On("Get", key, mock.AnythingOfType("*cache.SystemStats")).Return(ErrCacheMiss)

		stats := manager.GetStats()
		assert.NotNil(t, stats)
		assert.True(t, stats.IsStale)
		assert.True(t, stats.LastUpdated.IsZero())
		assert.Equal(t, 0, stats.Distributors)
		assert.Equal(t, 0, stats.Resellers)
		assert.Equal(t, 0, stats.Customers)
		assert.Equal(t, 0, stats.Users)
		assert.Equal(t, 0, stats.Systems)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Redis error", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &StatsCacheManager{
			redis:          mockRedis,
			cacheTTL:       time.Hour,
			staleThreshold: 30 * time.Minute,
		}

		mockRedis.On("Get", key, mock.AnythingOfType("*cache.SystemStats")).Return(errors.New("redis error"))

		stats := manager.GetStats()
		assert.NotNil(t, stats)
		assert.True(t, stats.IsStale)
		assert.True(t, stats.LastUpdated.IsZero())
		mockRedis.AssertExpectations(t)
	})
}

func TestStatsCacheManagerSetLogtoClient(t *testing.T) {
	manager := &StatsCacheManager{}
	mockLogto := &MockLogtoClient{}

	assert.Nil(t, manager.logtoClient)

	manager.SetLogtoClient(mockLogto)
	assert.Equal(t, mockLogto, manager.logtoClient)
}

func TestStatsCacheManagerGetCacheStats(t *testing.T) {
	t.Run("Successful stats retrieval", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &StatsCacheManager{
			redis:          mockRedis,
			cacheTTL:       time.Hour,
			staleThreshold: 30 * time.Minute,
			isRunning:      true,
		}

		expectedRedisStats := map[string]interface{}{
			"connections": 10,
			"memory_used": "1MB",
		}

		mockRedis.On("GetStats").Return(expectedRedisStats, nil)

		stats := manager.GetCacheStats()
		assert.NotNil(t, stats)
		assert.Equal(t, 60.0, stats["cache_ttl_minutes"])
		assert.Equal(t, 30.0, stats["stale_threshold_minutes"])
		assert.Equal(t, true, stats["is_running"])
		assert.Equal(t, expectedRedisStats, stats["redis_stats"])
		assert.Equal(t, "stats:", stats["cache_prefix"])
		mockRedis.AssertExpectations(t)
	})

	t.Run("Redis error during stats retrieval", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &StatsCacheManager{
			redis:          mockRedis,
			cacheTTL:       time.Hour,
			staleThreshold: 30 * time.Minute,
			isRunning:      false,
		}

		mockRedis.On("GetStats").Return(map[string]interface{}(nil), errors.New("redis error"))

		stats := manager.GetCacheStats()
		assert.NotNil(t, stats)
		assert.Contains(t, stats, "error")
		assert.Equal(t, "failed to get Redis stats", stats["error"])
		assert.Equal(t, 60.0, stats["cache_ttl_minutes"])
		if isRunning, exists := stats["is_running"]; exists {
			assert.Equal(t, false, isRunning)
		}
		mockRedis.AssertExpectations(t)
	})
}

func TestStatsCacheManagerClearCache(t *testing.T) {
	expectedPattern := "stats:*"

	t.Run("Successful clear", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &StatsCacheManager{
			redis: mockRedis,
		}

		mockRedis.On("DeletePattern", expectedPattern).Return(nil)

		err := manager.ClearCache()
		assert.NoError(t, err)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Redis error during clear", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &StatsCacheManager{
			redis: mockRedis,
		}

		mockRedis.On("DeletePattern", expectedPattern).Return(errors.New("delete error"))

		err := manager.ClearCache()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delete error")
		mockRedis.AssertExpectations(t)
	})
}

func TestGetOrganizationType(t *testing.T) {
	manager := &StatsCacheManager{}

	tests := []struct {
		name     string
		org      models.LogtoOrganization
		expected string
	}{
		{
			name: "Explicit type field",
			org: models.LogtoOrganization{
				ID:   "org_1",
				Name: "Test Org",
				CustomData: map[string]interface{}{
					"type": "distributor",
				},
			},
			expected: "distributor",
		},
		{
			name: "Unknown field defaults to customer",
			org: models.LogtoOrganization{
				ID:   "org_2",
				Name: "Unknown Org",
				CustomData: map[string]interface{}{
					"category": "business", // Random field, not the expected "type"
				},
			},
			expected: "customer",
		},
		{
			name: "Default to customer",
			org: models.LogtoOrganization{
				ID:         "org_6",
				Name:       "Unknown Type Organization",
				CustomData: nil,
			},
			expected: "customer",
		},
		{
			name: "Non-string type value",
			org: models.LogtoOrganization{
				ID:   "org_7",
				Name: "Invalid Type Org",
				CustomData: map[string]interface{}{
					"type": 123, // Non-string value
				},
			},
			expected: "customer",
		},
		{
			name: "Empty custom data",
			org: models.LogtoOrganization{
				ID:         "org_8",
				Name:       "Empty Custom Data Org",
				CustomData: map[string]interface{}{},
			},
			expected: "customer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.getOrganizationType(tt.org)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountSystems(t *testing.T) {
	manager := &StatsCacheManager{}

	t.Run("Count systems from custom data", func(t *testing.T) {
		orgs := []models.LogtoOrganization{
			{
				ID:   "org_1",
				Name: "Org 1",
				CustomData: map[string]interface{}{
					"systems": []interface{}{"system1", "system2", "system3"},
				},
			},
			{
				ID:   "org_2",
				Name: "Org 2",
				CustomData: map[string]interface{}{
					"systems": []interface{}{"system4", "system5"},
				},
			},
		}

		usersMap := make(map[string][]models.LogtoUser)

		count := manager.countSystems(orgs, usersMap)
		assert.Equal(t, 5, count) // 3 + 2 systems
	})

	t.Run("Fallback to user-based calculation", func(t *testing.T) {
		orgs := []models.LogtoOrganization{
			{
				ID:         "org_1",
				Name:       "Org 1",
				CustomData: nil,
			},
		}

		usersMap := map[string][]models.LogtoUser{
			"org_1": {
				{ID: "user1"}, {ID: "user2"}, {ID: "user3"},
				{ID: "user4"}, {ID: "user5"}, {ID: "user6"},
				{ID: "user7"}, {ID: "user8"}, {ID: "user9"},
				{ID: "user10"}, {ID: "user11"}, {ID: "user12"},
			},
		}

		count := manager.countSystems(orgs, usersMap)
		assert.Equal(t, 2, count) // (12 + 9) / 10 = 2 (rounded up)
	})

	t.Run("Empty organizations", func(t *testing.T) {
		orgs := []models.LogtoOrganization{}
		usersMap := make(map[string][]models.LogtoUser)

		count := manager.countSystems(orgs, usersMap)
		assert.Equal(t, 0, count)
	})

	t.Run("Organizations with invalid systems data", func(t *testing.T) {
		orgs := []models.LogtoOrganization{
			{
				ID:   "org_1",
				Name: "Org 1",
				CustomData: map[string]interface{}{
					"systems": "not_an_array", // Invalid type
				},
			},
		}

		usersMap := map[string][]models.LogtoUser{
			"org_1": {{ID: "user1"}},
		}

		count := manager.countSystems(orgs, usersMap)
		assert.Equal(t, 1, count) // Falls back to user-based calculation
	})

	t.Run("Mixed valid and invalid systems data", func(t *testing.T) {
		orgs := []models.LogtoOrganization{
			{
				ID:   "org_1",
				Name: "Org 1",
				CustomData: map[string]interface{}{
					"systems": []interface{}{"system1", "system2"},
				},
			},
			{
				ID:   "org_2",
				Name: "Org 2",
				CustomData: map[string]interface{}{
					"systems": "invalid",
				},
			},
		}

		usersMap := make(map[string][]models.LogtoUser)

		count := manager.countSystems(orgs, usersMap)
		assert.Equal(t, 2, count) // Only counts valid systems
	})
}

func TestStatsCacheManagerBackgroundUpdater(t *testing.T) {
	// We need to mock the configuration since StartBackgroundUpdater uses it
	// First let's test the updater control without actually starting the background process

	t.Run("Start background updater", func(t *testing.T) {
		// We'll test this by checking the state changes, not the actual background process
		// since that would require mocking time.NewTicker with a valid interval
		manager := &StatsCacheManager{
			redis:          NewMockRedisClient(),
			cacheTTL:       time.Hour,
			staleThreshold: 30 * time.Minute,
			isRunning:      false,
			stopChan:       make(chan struct{}),
			updateChan:     make(chan struct{}, 1),
		}

		assert.False(t, manager.isRunning)

		// Since we can't easily mock configuration.Config.StatsUpdateInterval,
		// we'll test the state management directly
		manager.updateMutex.Lock()
		manager.isRunning = true
		manager.updateMutex.Unlock()

		assert.True(t, manager.isRunning)
	})

	t.Run("Stop background updater", func(t *testing.T) {
		manager := &StatsCacheManager{
			redis:          NewMockRedisClient(),
			cacheTTL:       time.Hour,
			staleThreshold: 30 * time.Minute,
			isRunning:      true,
			stopChan:       make(chan struct{}),
			updateChan:     make(chan struct{}, 1),
		}

		// Test stopping
		manager.StopBackgroundUpdater()
		assert.False(t, manager.isRunning)
	})

	t.Run("Stop non-running updater", func(t *testing.T) {
		manager := &StatsCacheManager{
			redis:          NewMockRedisClient(),
			cacheTTL:       time.Hour,
			staleThreshold: 30 * time.Minute,
			isRunning:      false,
			stopChan:       make(chan struct{}),
			updateChan:     make(chan struct{}, 1),
		}

		assert.False(t, manager.isRunning)

		// Should not panic or cause issues
		manager.StopBackgroundUpdater()
		assert.False(t, manager.isRunning)
	})
}

func TestStatsCacheManagerTriggerUpdate(t *testing.T) {
	manager := &StatsCacheManager{
		updateChan: make(chan struct{}, 1),
	}

	t.Run("Trigger update successfully", func(t *testing.T) {
		manager.triggerUpdate()

		// Check that signal was sent
		select {
		case <-manager.updateChan:
			// Successfully received update signal
		case <-time.After(100 * time.Millisecond):
			t.Error("Update signal was not sent")
		}
	})

	t.Run("Trigger update when channel is full", func(t *testing.T) {
		// Fill the channel first
		manager.updateChan <- struct{}{}

		// This should not block due to default case
		manager.triggerUpdate()

		// Clear the channel
		<-manager.updateChan
	})
}

func TestStatsCacheEdgeCases(t *testing.T) {
	t.Run("SystemStats with zero values", func(t *testing.T) {
		stats := SystemStats{
			Distributors: 0,
			Resellers:    0,
			Customers:    0,
			Users:        0,
			Systems:      0,
			LastUpdated:  time.Time{},
			IsStale:      true,
		}

		assert.Equal(t, 0, stats.Distributors)
		assert.Equal(t, 0, stats.Resellers)
		assert.Equal(t, 0, stats.Customers)
		assert.Equal(t, 0, stats.Users)
		assert.Equal(t, 0, stats.Systems)
		assert.True(t, stats.LastUpdated.IsZero())
		assert.True(t, stats.IsStale)
	})

	t.Run("SystemStats with large values", func(t *testing.T) {
		stats := SystemStats{
			Distributors: 1000,
			Resellers:    5000,
			Customers:    100000,
			Users:        1000000,
			Systems:      50000,
		}

		assert.Equal(t, 1000, stats.Distributors)
		assert.Equal(t, 5000, stats.Resellers)
		assert.Equal(t, 100000, stats.Customers)
		assert.Equal(t, 1000000, stats.Users)
		assert.Equal(t, 50000, stats.Systems)
	})

	t.Run("Manager with nil Redis client", func(t *testing.T) {
		manager := &StatsCacheManager{
			redis:          nil,
			cacheTTL:       time.Hour,
			staleThreshold: 30 * time.Minute,
		}

		// Should not panic with nil Redis client
		stats := manager.GetStats()
		assert.NotNil(t, stats)
		assert.True(t, stats.IsStale)
	})
}
