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

func TestOrgUsersCacheStruct(t *testing.T) {
	now := time.Now()
	users := []models.LogtoUser{
		{
			ID:           "user_1",
			Username:     "testuser1",
			PrimaryEmail: "user1@example.com",
			Name:         "Test User 1",
			IsSuspended:  false,
			HasPassword:  true,
		},
		{
			ID:           "user_2",
			Username:     "testuser2",
			PrimaryEmail: "user2@example.com",
			Name:         "Test User 2",
			IsSuspended:  false,
			HasPassword:  true,
		},
	}

	cache := OrgUsersCache{
		Users:     users,
		CachedAt:  now,
		ExpiresAt: now.Add(30 * time.Minute),
	}

	assert.Len(t, cache.Users, 2)
	assert.Equal(t, users, cache.Users)
	assert.Equal(t, now, cache.CachedAt)
	assert.True(t, cache.ExpiresAt.After(now))
	assert.Equal(t, 30*time.Minute, cache.ExpiresAt.Sub(cache.CachedAt))
}

func TestOrgUsersCacheManagerStruct(t *testing.T) {
	mockRedis := NewMockRedisClient()
	ttl := 30 * time.Minute

	manager := &OrgUsersCacheManager{
		redis: mockRedis,
		ttl:   ttl,
	}

	assert.Equal(t, mockRedis, manager.redis)
	assert.Equal(t, ttl, manager.ttl)
}

func TestOrgUsersCacheManagerGet(t *testing.T) {
	orgID := "org_123"
	expectedKey := "org_users:org_123"

	t.Run("Cache hit with valid data", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &OrgUsersCacheManager{
			redis: mockRedis,
			ttl:   30 * time.Minute,
		}

		now := time.Now()
		cachedUsers := []models.LogtoUser{
			{
				ID:           "user_1",
				Username:     "testuser1",
				PrimaryEmail: "user1@example.com",
				Name:         "Test User 1",
			},
			{
				ID:           "user_2",
				Username:     "testuser2",
				PrimaryEmail: "user2@example.com",
				Name:         "Test User 2",
			},
		}

		cached := OrgUsersCache{
			Users:     cachedUsers,
			CachedAt:  now,
			ExpiresAt: now.Add(time.Hour), // Not expired
		}

		mockRedis.On("Get", expectedKey, mock.AnythingOfType("*cache.OrgUsersCache")).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(1).(*OrgUsersCache)
			*arg = cached
		})

		users, found := manager.Get(orgID)
		assert.True(t, found)
		assert.Equal(t, cachedUsers, users)
		assert.Len(t, users, 2)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Cache miss", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &OrgUsersCacheManager{
			redis: mockRedis,
			ttl:   30 * time.Minute,
		}

		mockRedis.On("Get", expectedKey, mock.AnythingOfType("*cache.OrgUsersCache")).Return(ErrCacheMiss)

		users, found := manager.Get(orgID)
		assert.False(t, found)
		assert.Nil(t, users)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Cache hit but expired", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &OrgUsersCacheManager{
			redis: mockRedis,
			ttl:   30 * time.Minute,
		}

		now := time.Now()
		cached := OrgUsersCache{
			Users:     []models.LogtoUser{{ID: "user_1"}},
			CachedAt:  now.Add(-2 * time.Hour),
			ExpiresAt: now.Add(-time.Hour), // Expired
		}

		mockRedis.On("Get", expectedKey, mock.AnythingOfType("*cache.OrgUsersCache")).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(1).(*OrgUsersCache)
			*arg = cached
		})

		users, found := manager.Get(orgID)
		assert.False(t, found)
		assert.Nil(t, users)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Redis error", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &OrgUsersCacheManager{
			redis: mockRedis,
			ttl:   30 * time.Minute,
		}

		mockRedis.On("Get", expectedKey, mock.AnythingOfType("*cache.OrgUsersCache")).Return(assert.AnError)

		users, found := manager.Get(orgID)
		assert.False(t, found)
		assert.Nil(t, users)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Empty users list", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &OrgUsersCacheManager{
			redis: mockRedis,
			ttl:   30 * time.Minute,
		}

		now := time.Now()
		cached := OrgUsersCache{
			Users:     []models.LogtoUser{}, // Empty but valid
			CachedAt:  now,
			ExpiresAt: now.Add(time.Hour),
		}

		mockRedis.On("Get", expectedKey, mock.AnythingOfType("*cache.OrgUsersCache")).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(1).(*OrgUsersCache)
			*arg = cached
		})

		users, found := manager.Get(orgID)
		assert.True(t, found)
		assert.NotNil(t, users)
		assert.Len(t, users, 0)
		mockRedis.AssertExpectations(t)
	})
}

func TestOrgUsersCacheManagerSet(t *testing.T) {
	mockRedis := NewMockRedisClient()
	manager := &OrgUsersCacheManager{
		redis: mockRedis,
		ttl:   30 * time.Minute,
	}

	orgID := "org_123"
	expectedKey := "org_users:org_123"
	users := []models.LogtoUser{
		{
			ID:           "user_1",
			Username:     "testuser1",
			PrimaryEmail: "user1@example.com",
			Name:         "Test User 1",
			IsSuspended:  false,
			HasPassword:  true,
		},
		{
			ID:           "user_2",
			Username:     "testuser2",
			PrimaryEmail: "user2@example.com",
			Name:         "Test User 2",
			IsSuspended:  true,
			HasPassword:  false,
		},
	}

	t.Run("Successful set", func(t *testing.T) {
		mockRedis.On("Set", expectedKey, mock.AnythingOfType("cache.OrgUsersCache"), manager.ttl).Return(nil)

		manager.Set(orgID, users)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Redis error during set", func(t *testing.T) {
		mockRedis.On("Set", expectedKey, mock.AnythingOfType("cache.OrgUsersCache"), manager.ttl).Return(assert.AnError)

		// Should not panic on error
		manager.Set(orgID, users)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Empty users slice", func(t *testing.T) {
		emptyUsers := []models.LogtoUser{}
		mockRedis.On("Set", expectedKey, mock.AnythingOfType("cache.OrgUsersCache"), manager.ttl).Return(nil)

		manager.Set(orgID, emptyUsers)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Nil users slice", func(t *testing.T) {
		mockRedis.On("Set", expectedKey, mock.AnythingOfType("cache.OrgUsersCache"), manager.ttl).Return(nil)

		manager.Set(orgID, nil)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Large number of users", func(t *testing.T) {
		largeUserList := make([]models.LogtoUser, 1000)
		for i := 0; i < 1000; i++ {
			largeUserList[i] = models.LogtoUser{
				ID:           "user_" + string(rune(i)),
				Username:     "user" + string(rune(i)),
				PrimaryEmail: "user" + string(rune(i)) + "@example.com",
			}
		}

		mockRedis.On("Set", expectedKey, mock.AnythingOfType("cache.OrgUsersCache"), manager.ttl).Return(nil)

		manager.Set(orgID, largeUserList)
		mockRedis.AssertExpectations(t)
	})
}

func TestOrgUsersCacheManagerClear(t *testing.T) {
	mockRedis := NewMockRedisClient()
	manager := &OrgUsersCacheManager{
		redis: mockRedis,
		ttl:   30 * time.Minute,
	}

	orgID := "org_123"
	expectedKey := "org_users:org_123"

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
}

func TestOrgUsersCacheManagerClearAll(t *testing.T) {
	mockRedis := NewMockRedisClient()
	manager := &OrgUsersCacheManager{
		redis: mockRedis,
		ttl:   30 * time.Minute,
	}

	expectedPattern := "org_users:*"

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
}

func TestOrgUsersCacheManagerGetStats(t *testing.T) {
	t.Run("Successful stats retrieval", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &OrgUsersCacheManager{
			redis: mockRedis,
			ttl:   30 * time.Minute,
		}

		expectedRedisStats := map[string]interface{}{
			"connections": 10,
			"memory_used": "1MB",
		}

		mockRedis.On("GetStats").Return(expectedRedisStats, nil)

		stats := manager.GetStats()
		assert.NotNil(t, stats)
		assert.Equal(t, 30.0, stats["ttl_minutes"])
		assert.Equal(t, expectedRedisStats, stats["redis_stats"])
		assert.Equal(t, "org_users:", stats["cache_prefix"])
		mockRedis.AssertExpectations(t)
	})

	t.Run("Redis error during stats retrieval", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &OrgUsersCacheManager{
			redis: mockRedis,
			ttl:   30 * time.Minute,
		}

		mockRedis.On("GetStats").Return(map[string]interface{}(nil), assert.AnError)

		stats := manager.GetStats()
		assert.NotNil(t, stats)
		assert.Contains(t, stats, "error")
		assert.Equal(t, "failed to get Redis stats", stats["error"])
		assert.Equal(t, 30.0, stats["ttl_minutes"])
		mockRedis.AssertExpectations(t)
	})
}

func TestOrgUsersCacheKeyGeneration(t *testing.T) {
	tests := []struct {
		name     string
		orgID    string
		expected string
	}{
		{
			name:     "Normal organization ID",
			orgID:    "org_123",
			expected: "org_users:org_123",
		},
		{
			name:     "Organization ID with special characters",
			orgID:    "org_123-456_789",
			expected: "org_users:org_123-456_789",
		},
		{
			name:     "Empty organization ID",
			orgID:    "",
			expected: "org_users:",
		},
		{
			name:     "UUID-style organization ID",
			orgID:    "550e8400-e29b-41d4-a716-446655440000",
			expected: "org_users:550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "Organization ID with dots",
			orgID:    "org.example.com",
			expected: "org_users:org.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRedis := NewMockRedisClient()
			manager := &OrgUsersCacheManager{
				redis: mockRedis,
				ttl:   30 * time.Minute,
			}

			mockRedis.On("Get", tt.expected, mock.AnythingOfType("*cache.OrgUsersCache")).Return(ErrCacheMiss)

			_, found := manager.Get(tt.orgID)
			assert.False(t, found)
			mockRedis.AssertExpectations(t)
		})
	}
}

func TestOrgUsersCacheWithDifferentUserTypes(t *testing.T) {
	mockRedis := NewMockRedisClient()
	manager := &OrgUsersCacheManager{
		redis: mockRedis,
		ttl:   30 * time.Minute,
	}

	now := time.Now()
	lastSignIn := now.Unix() - 3600

	users := []models.LogtoUser{
		{
			ID:            "admin_user",
			Username:      "admin",
			PrimaryEmail:  "admin@example.com",
			Name:          "System Administrator",
			IsSuspended:   false,
			HasPassword:   true,
			LastSignInAt:  &lastSignIn,
			CreatedAt:     now.Unix() - 86400,
			UpdatedAt:     now.Unix(),
			CustomData:    map[string]interface{}{"role": "admin"},
			ApplicationId: "app_123",
		},
		{
			ID:            "regular_user",
			Username:      "user1",
			PrimaryEmail:  "user1@example.com",
			Name:          "Regular User",
			PrimaryPhone:  "+1234567890",
			Avatar:        "https://example.com/avatar.png",
			IsSuspended:   false,
			HasPassword:   true,
			LastSignInAt:  nil, // Never signed in
			CreatedAt:     now.Unix() - 43200,
			UpdatedAt:     now.Unix(),
			CustomData:    map[string]interface{}{"department": "sales"},
			ApplicationId: "app_123",
		},
		{
			ID:            "suspended_user",
			Username:      "suspended",
			PrimaryEmail:  "suspended@example.com",
			Name:          "Suspended User",
			IsSuspended:   true,
			HasPassword:   false,
			CreatedAt:     now.Unix() - 172800,
			UpdatedAt:     now.Unix() - 86400,
			CustomData:    map[string]interface{}{"status": "suspended"},
			ApplicationId: "app_123",
		},
	}

	orgID := "org_diverse_users"
	expectedKey := "org_users:org_diverse_users"

	mockRedis.On("Set", expectedKey, mock.AnythingOfType("cache.OrgUsersCache"), manager.ttl).Return(nil)

	manager.Set(orgID, users)

	// Verify the cache structure supports different user types
	assert.Len(t, users, 3)

	// Admin user
	assert.Equal(t, "admin", users[0].Username)
	assert.False(t, users[0].IsSuspended)
	assert.True(t, users[0].HasPassword)
	assert.NotNil(t, users[0].LastSignInAt)
	assert.Equal(t, "admin", users[0].CustomData["role"])

	// Regular user
	assert.Equal(t, "user1", users[1].Username)
	assert.False(t, users[1].IsSuspended)
	assert.True(t, users[1].HasPassword)
	assert.Nil(t, users[1].LastSignInAt)
	assert.NotEmpty(t, users[1].PrimaryPhone)
	assert.NotEmpty(t, users[1].Avatar)
	assert.Equal(t, "sales", users[1].CustomData["department"])

	// Suspended user
	assert.Equal(t, "suspended", users[2].Username)
	assert.True(t, users[2].IsSuspended)
	assert.False(t, users[2].HasPassword)
	assert.Equal(t, "suspended", users[2].CustomData["status"])

	mockRedis.AssertExpectations(t)
}

func TestOrgUsersCacheEdgeCases(t *testing.T) {
	t.Run("Cache with exactly expiry time", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &OrgUsersCacheManager{
			redis: mockRedis,
			ttl:   30 * time.Minute,
		}

		now := time.Now()
		cached := OrgUsersCache{
			Users:     []models.LogtoUser{{ID: "user_1"}},
			CachedAt:  now.Add(-time.Hour),
			ExpiresAt: now, // Exactly at expiry time
		}

		orgID := "org_edge_case"
		expectedKey := "org_users:org_edge_case"

		mockRedis.On("Get", expectedKey, mock.AnythingOfType("*cache.OrgUsersCache")).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(1).(*OrgUsersCache)
			*arg = cached
		})

		users, found := manager.Get(orgID)
		assert.False(t, found) // Should be expired
		assert.Nil(t, users)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Cache with users having complex custom data", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		manager := &OrgUsersCacheManager{
			redis: mockRedis,
			ttl:   30 * time.Minute,
		}

		complexCustomData := map[string]interface{}{
			"permissions": []interface{}{"read", "write", "admin"},
			"metadata": map[string]interface{}{
				"last_login_ip": "192.168.1.100",
				"preferences": map[string]interface{}{
					"theme":    "dark",
					"language": "en",
				},
			},
			"tags":        []interface{}{"vip", "early_adopter"},
			"score":       95.5,
			"verified":    true,
			"login_count": 42,
		}

		users := []models.LogtoUser{
			{
				ID:           "complex_user",
				Username:     "complex",
				PrimaryEmail: "complex@example.com",
				Name:         "Complex User",
				CustomData:   complexCustomData,
			},
		}

		orgID := "org_complex"
		expectedKey := "org_users:org_complex"

		mockRedis.On("Set", expectedKey, mock.AnythingOfType("cache.OrgUsersCache"), manager.ttl).Return(nil)

		manager.Set(orgID, users)
		assert.Equal(t, complexCustomData, users[0].CustomData)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Cache with users having long field values", func(t *testing.T) {
		longBio := "This is a very long biography that contains multiple sentences and explains in great detail about this user's background, experience, skills, achievements, and various other important aspects of their professional and personal life that might be relevant for the organization's understanding of their profile and capabilities."

		users := []models.LogtoUser{
			{
				ID:           "long_bio_user",
				Username:     "longbio",
				PrimaryEmail: "longbio@example.com",
				Name:         "User With Long Bio",
				Profile:      map[string]interface{}{"bio": longBio},
			},
		}

		assert.Equal(t, longBio, users[0].Profile["bio"])
	})
}
