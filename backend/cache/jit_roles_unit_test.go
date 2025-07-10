/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cache

import (
	"testing"
	"time"

	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestJitRolesCacheStruct(t *testing.T) {
	now := time.Now()
	roles := []models.LogtoOrganizationRole{
		{
			ID:          "role_1",
			Name:        "Owner",
			Description: "Organization owner",
			Type:        "User",
			IsDefault:   false,
		},
		{
			ID:          "role_2",
			Name:        "Admin",
			Description: "System administrator",
			Type:        "User",
			IsDefault:   true,
		},
	}

	cache := JitRolesCache{
		Roles:     roles,
		CachedAt:  now,
		ExpiresAt: now.Add(time.Hour),
	}

	assert.Len(t, cache.Roles, 2)
	assert.Equal(t, roles, cache.Roles)
	assert.Equal(t, now, cache.CachedAt)
	assert.True(t, cache.ExpiresAt.After(now))
	assert.Equal(t, time.Hour, cache.ExpiresAt.Sub(cache.CachedAt))
}

func TestJitRolesCacheManagerStruct(t *testing.T) {
	mockRedis := NewMockRedisClient()
	ttl := 15 * time.Minute

	manager := &JitRolesCacheManager{
		redis: mockRedis,
		ttl:   ttl,
	}

	assert.Equal(t, mockRedis, manager.redis)
	assert.Equal(t, ttl, manager.ttl)
}

func TestJitRolesCacheManagerGet(t *testing.T) {
	mockRedis := NewMockRedisClient()
	manager := &JitRolesCacheManager{
		redis: mockRedis,
		ttl:   15 * time.Minute,
	}

	orgID := "org_123"
	expectedKey := "jit_roles:org_123"

	t.Run("Cache hit with valid data", func(t *testing.T) {
		now := time.Now()
		cachedRoles := []models.LogtoOrganizationRole{
			{
				ID:          "role_1",
				Name:        "Owner",
				Description: "Organization owner",
			},
		}

		cached := JitRolesCache{
			Roles:     cachedRoles,
			CachedAt:  now,
			ExpiresAt: now.Add(time.Hour), // Not expired
		}

		mockRedis.On("Get", expectedKey, mock.AnythingOfType("*cache.JitRolesCache")).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(1).(*JitRolesCache)
			*arg = cached
		})

		roles, found := manager.Get(orgID)
		assert.True(t, found)
		assert.Equal(t, cachedRoles, roles)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Cache miss", func(t *testing.T) {
		mockRedis := NewMockRedisClient() // Fresh mock for this test
		manager := &JitRolesCacheManager{
			redis: mockRedis,
			ttl:   15 * time.Minute,
		}
		mockRedis.On("Get", expectedKey, mock.AnythingOfType("*cache.JitRolesCache")).Return(ErrCacheMiss)

		roles, found := manager.Get(orgID)
		assert.False(t, found)
		assert.Nil(t, roles)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Cache hit but expired", func(t *testing.T) {
		mockRedis := NewMockRedisClient() // Fresh mock for this test
		manager := &JitRolesCacheManager{
			redis: mockRedis,
			ttl:   15 * time.Minute,
		}
		now := time.Now()
		cached := JitRolesCache{
			Roles:     []models.LogtoOrganizationRole{{ID: "role_1"}},
			CachedAt:  now.Add(-2 * time.Hour),
			ExpiresAt: now.Add(-time.Hour), // Expired
		}

		mockRedis.On("Get", expectedKey, mock.AnythingOfType("*cache.JitRolesCache")).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(1).(*JitRolesCache)
			*arg = cached
		})

		roles, found := manager.Get(orgID)
		assert.False(t, found)
		assert.Nil(t, roles)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Redis error", func(t *testing.T) {
		mockRedis := NewMockRedisClient() // Fresh mock for this test
		manager := &JitRolesCacheManager{
			redis: mockRedis,
			ttl:   15 * time.Minute,
		}
		mockRedis.On("Get", expectedKey, mock.AnythingOfType("*cache.JitRolesCache")).Return(assert.AnError)

		roles, found := manager.Get(orgID)
		assert.False(t, found)
		assert.Nil(t, roles)
		mockRedis.AssertExpectations(t)
	})

	t.Run("No Redis client", func(t *testing.T) {
		managerWithoutRedis := &JitRolesCacheManager{
			redis: nil,
			ttl:   15 * time.Minute,
		}

		roles, found := managerWithoutRedis.Get(orgID)
		assert.False(t, found)
		assert.Nil(t, roles)
	})
}

func TestJitRolesCacheManagerSet(t *testing.T) {
	mockRedis := NewMockRedisClient()
	manager := &JitRolesCacheManager{
		redis: mockRedis,
		ttl:   15 * time.Minute,
	}

	orgID := "org_123"
	expectedKey := "jit_roles:org_123"
	roles := []models.LogtoOrganizationRole{
		{
			ID:          "role_1",
			Name:        "Owner",
			Description: "Organization owner",
		},
		{
			ID:          "role_2",
			Name:        "Admin",
			Description: "System administrator",
		},
	}

	t.Run("Successful set", func(t *testing.T) {
		mockRedis.On("Set", expectedKey, mock.AnythingOfType("cache.JitRolesCache"), manager.ttl).Return(nil)

		manager.Set(orgID, roles)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Redis error during set", func(t *testing.T) {
		mockRedis.On("Set", expectedKey, mock.AnythingOfType("cache.JitRolesCache"), manager.ttl).Return(assert.AnError)

		// Should not panic on error
		manager.Set(orgID, roles)
		mockRedis.AssertExpectations(t)
	})

	t.Run("No Redis client", func(t *testing.T) {
		managerWithoutRedis := &JitRolesCacheManager{
			redis: nil,
			ttl:   15 * time.Minute,
		}

		// Should not panic without Redis
		managerWithoutRedis.Set(orgID, roles)
	})

	t.Run("Empty roles slice", func(t *testing.T) {
		emptyRoles := []models.LogtoOrganizationRole{}
		mockRedis.On("Set", expectedKey, mock.AnythingOfType("cache.JitRolesCache"), manager.ttl).Return(nil)

		manager.Set(orgID, emptyRoles)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Nil roles slice", func(t *testing.T) {
		mockRedis.On("Set", expectedKey, mock.AnythingOfType("cache.JitRolesCache"), manager.ttl).Return(nil)

		manager.Set(orgID, nil)
		mockRedis.AssertExpectations(t)
	})
}

func TestJitRolesCacheManagerClear(t *testing.T) {
	mockRedis := NewMockRedisClient()
	manager := &JitRolesCacheManager{
		redis: mockRedis,
		ttl:   15 * time.Minute,
	}

	orgID := "org_123"
	expectedKey := "jit_roles:org_123"

	t.Run("Successful clear", func(t *testing.T) {
		mockRedis.On("Delete", expectedKey).Return(nil)

		manager.Clear(orgID)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Redis error during clear", func(t *testing.T) {
		mockRedis.On("Delete", expectedKey).Return(assert.AnError)

		// Should not panic on error
		manager.Clear(orgID)
		mockRedis.AssertExpectations(t)
	})

	t.Run("No Redis client", func(t *testing.T) {
		managerWithoutRedis := &JitRolesCacheManager{
			redis: nil,
			ttl:   15 * time.Minute,
		}

		// Should not panic without Redis
		managerWithoutRedis.Clear(orgID)
	})
}

func TestJitRolesCacheManagerClearAll(t *testing.T) {
	mockRedis := NewMockRedisClient()
	manager := &JitRolesCacheManager{
		redis: mockRedis,
		ttl:   15 * time.Minute,
	}

	expectedPattern := "jit_roles:*"

	t.Run("Successful clear all", func(t *testing.T) {
		mockRedis.On("DeletePattern", expectedPattern).Return(nil)

		manager.ClearAll()
		mockRedis.AssertExpectations(t)
	})

	t.Run("Redis error during clear all", func(t *testing.T) {
		mockRedis.On("DeletePattern", expectedPattern).Return(assert.AnError)

		// Should not panic on error
		manager.ClearAll()
		mockRedis.AssertExpectations(t)
	})

	t.Run("No Redis client", func(t *testing.T) {
		managerWithoutRedis := &JitRolesCacheManager{
			redis: nil,
			ttl:   15 * time.Minute,
		}

		// Should not panic without Redis
		managerWithoutRedis.ClearAll()
	})
}

func TestJitRolesCacheManagerGetStats(t *testing.T) {
	t.Run("With Redis client available", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &JitRolesCacheManager{
			redis: mockRedis,
			ttl:   15 * time.Minute,
		}

		expectedRedisStats := map[string]interface{}{
			"connections": 10,
			"memory_used": "1MB",
		}

		mockRedis.On("GetStats").Return(expectedRedisStats, nil)

		stats := manager.GetStats()
		assert.NotNil(t, stats)
		assert.Equal(t, true, stats["redis_available"])
		assert.Equal(t, 15.0, stats["ttl_minutes"])
		assert.Equal(t, expectedRedisStats, stats["redis_stats"])
		assert.Equal(t, "jit_roles:", stats["cache_prefix"])
		mockRedis.AssertExpectations(t)
	})

	t.Run("With Redis client error", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &JitRolesCacheManager{
			redis: mockRedis,
			ttl:   15 * time.Minute,
		}

		mockRedis.On("GetStats").Return(map[string]interface{}(nil), assert.AnError)

		stats := manager.GetStats()
		assert.NotNil(t, stats)
		assert.Contains(t, stats, "error")
		assert.Equal(t, "failed to get Redis stats", stats["error"])
		assert.Equal(t, 15.0, stats["ttl_minutes"])
		mockRedis.AssertExpectations(t)
	})

	t.Run("Without Redis client", func(t *testing.T) {
		manager := &JitRolesCacheManager{
			redis: nil,
			ttl:   15 * time.Minute,
		}

		stats := manager.GetStats()
		assert.NotNil(t, stats)
		assert.Equal(t, false, stats["redis_available"])
		assert.Equal(t, 15.0, stats["ttl_minutes"])
		assert.Equal(t, "jit_roles:", stats["cache_prefix"])
	})
}

func TestJitRolesCacheKeyGeneration(t *testing.T) {
	tests := []struct {
		name     string
		orgID    string
		expected string
	}{
		{
			name:     "Normal organization ID",
			orgID:    "org_123",
			expected: "jit_roles:org_123",
		},
		{
			name:     "Organization ID with special characters",
			orgID:    "org_123-456_789",
			expected: "jit_roles:org_123-456_789",
		},
		{
			name:     "Empty organization ID",
			orgID:    "",
			expected: "jit_roles:",
		},
		{
			name:     "UUID-style organization ID",
			orgID:    "550e8400-e29b-41d4-a716-446655440000",
			expected: "jit_roles:550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRedis := NewMockRedisClient()
			manager := &JitRolesCacheManager{
				redis: mockRedis,
				ttl:   15 * time.Minute,
			}

			mockRedis.On("Get", tt.expected, mock.AnythingOfType("*cache.JitRolesCache")).Return(ErrCacheMiss)

			_, found := manager.Get(tt.orgID)
			assert.False(t, found)
			mockRedis.AssertExpectations(t)
		})
	}
}

func TestJitRolesCacheWithDifferentRoleTypes(t *testing.T) {
	mockRedis := NewMockRedisClient()
	manager := &JitRolesCacheManager{
		redis: mockRedis,
		ttl:   15 * time.Minute,
	}

	roles := []models.LogtoOrganizationRole{
		{
			ID:          "org_rol_owner",
			Name:        "Owner",
			Description: "Organization owner",
			Type:        "User",
			IsDefault:   false,
		},
		{
			ID:          "org_rol_admin",
			Name:        "Admin",
			Description: "System administrator",
			Type:        "Machine",
			IsDefault:   true,
		},
		{
			ID:          "org_rol_viewer",
			Name:        "Viewer",
			Description: "Read-only access",
			Type:        "User",
			IsDefault:   false,
		},
	}

	orgID := "org_test"
	expectedKey := "jit_roles:org_test"

	mockRedis.On("Set", expectedKey, mock.AnythingOfType("cache.JitRolesCache"), manager.ttl).Return(nil)

	manager.Set(orgID, roles)

	// Verify the cache structure supports different role types
	assert.Len(t, roles, 3)
	assert.Equal(t, "Owner", roles[0].Name)
	assert.Equal(t, "User", roles[0].Type)
	assert.Equal(t, "Admin", roles[1].Name)
	assert.Equal(t, "Machine", roles[1].Type)
	assert.True(t, roles[1].IsDefault)
	assert.False(t, roles[2].IsDefault)

	mockRedis.AssertExpectations(t)
}

func TestJitRolesCacheEdgeCases(t *testing.T) {
	t.Run("Cache with exactly expiry time", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &JitRolesCacheManager{
			redis: mockRedis,
			ttl:   15 * time.Minute,
		}

		now := time.Now()
		cached := JitRolesCache{
			Roles:     []models.LogtoOrganizationRole{{ID: "role_1"}},
			CachedAt:  now.Add(-time.Hour),
			ExpiresAt: now, // Exactly at expiry time
		}

		orgID := "org_edge_case"
		expectedKey := "jit_roles:org_edge_case"

		mockRedis.On("Get", expectedKey, mock.AnythingOfType("*cache.JitRolesCache")).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(1).(*JitRolesCache)
			*arg = cached
		})

		roles, found := manager.Get(orgID)
		assert.False(t, found) // Should be expired
		assert.Nil(t, roles)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Cache with very long role descriptions", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &JitRolesCacheManager{
			redis: mockRedis,
			ttl:   15 * time.Minute,
		}

		longDescription := "This is a very long description that contains multiple sentences and explains in great detail what this organization role does, its responsibilities, permissions, and various other important aspects that might be relevant for understanding the complete scope of this role within the organization hierarchy and business operations."

		roles := []models.LogtoOrganizationRole{
			{
				ID:          "role_long_desc",
				Name:        "Complex Role",
				Description: longDescription,
				Type:        "User",
				IsDefault:   false,
			},
		}

		orgID := "org_long_desc"
		expectedKey := "jit_roles:org_long_desc"

		mockRedis.On("Set", expectedKey, mock.AnythingOfType("cache.JitRolesCache"), manager.ttl).Return(nil)

		manager.Set(orgID, roles)
		assert.Equal(t, longDescription, roles[0].Description)
		mockRedis.AssertExpectations(t)
	})
}
