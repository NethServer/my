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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// SimpleRedisClient is a simple in-memory implementation for testing
type SimpleRedisClient struct {
	data map[string]interface{}
}

func NewSimpleRedisClient() *SimpleRedisClient {
	return &SimpleRedisClient{
		data: make(map[string]interface{}),
	}
}

func (s *SimpleRedisClient) Set(key string, value interface{}, ttl time.Duration) error {
	s.data[key] = value
	return nil
}

func (s *SimpleRedisClient) SetWithContext(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return s.Set(key, value, ttl)
}

func (s *SimpleRedisClient) Get(key string, dest interface{}) error {
	if value, exists := s.data[key]; exists {
		// Simple copy for testing
		switch v := value.(type) {
		case OrganizationNamesCache:
			*dest.(*OrganizationNamesCache) = v
		}
		return nil
	}
	return ErrCacheMiss
}

func (s *SimpleRedisClient) GetWithContext(ctx context.Context, key string, dest interface{}) error {
	return s.Get(key, dest)
}

func (s *SimpleRedisClient) Delete(key string) error {
	delete(s.data, key)
	return nil
}

func (s *SimpleRedisClient) DeleteWithContext(ctx context.Context, key string) error {
	return s.Delete(key)
}

// Implement other methods as no-ops for testing
func (s *SimpleRedisClient) DeletePattern(pattern string) error { return nil }
func (s *SimpleRedisClient) DeletePatternWithContext(ctx context.Context, pattern string) error {
	return nil
}
func (s *SimpleRedisClient) Exists(key string) (bool, error)           { return false, nil }
func (s *SimpleRedisClient) GetTTL(key string) (time.Duration, error)  { return 0, nil }
func (s *SimpleRedisClient) FlushDB() error                            { return nil }
func (s *SimpleRedisClient) GetStats() (map[string]interface{}, error) { return nil, nil }
func (s *SimpleRedisClient) Close() error                              { return nil }

func TestOrganizationNamesCacheManager_SetAndGet(t *testing.T) {
	// Create cache manager with simple Redis
	cache := &OrganizationNamesCacheManager{
		redis: NewSimpleRedisClient(),
		ttl:   5 * time.Minute,
	}

	// Test data
	testNames := map[string]string{
		"test org":     "Test Org",
		"example corp": "Example Corp",
		"edoardo srl":  "Edoardo SRL",
		"r":            "R",
		"erre":         "Erre",
	}

	// Set cache
	cache.Set(testNames)

	// Get cache
	retrieved, found := cache.Get()
	assert.True(t, found)
	assert.Equal(t, testNames, retrieved)
}

func TestOrganizationNamesCacheManager_IsNameTaken(t *testing.T) {
	// Create cache manager with simple Redis
	cache := &OrganizationNamesCacheManager{
		redis: NewSimpleRedisClient(),
		ttl:   5 * time.Minute,
	}

	// Test data
	testNames := map[string]string{
		"test org":     "Test Org",
		"example corp": "Example Corp",
		"edoardo srl":  "Edoardo SRL",
		"r":            "R",
		"erre":         "Erre",
	}

	// Set cache
	cache.Set(testNames)

	// Test case-insensitive name checking
	testCases := []struct {
		name        string
		shouldExist bool
		expected    string
	}{
		{"Test Org", true, "Test Org"},
		{"test org", true, "Test Org"},
		{"TEST ORG", true, "Test Org"},
		{"r", true, "R"},
		{"R", true, "R"},
		{"erre", true, "Erre"},
		{"ERRE", true, "Erre"},
		{"edoardo srl", true, "Edoardo SRL"},
		{"EDOARDO SRL", true, "Edoardo SRL"},
		{"nonexistent", false, ""},
		{"Random Name", false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isTaken, originalName := cache.IsNameTaken(tc.name)
			assert.Equal(t, tc.shouldExist, isTaken, "Name existence check failed for: %s", tc.name)
			assert.Equal(t, tc.expected, originalName, "Original name mismatch for: %s", tc.name)
		})
	}
}

func TestOrganizationNamesCacheManager_AddAndRemoveName(t *testing.T) {
	// Create cache manager with simple Redis
	cache := &OrganizationNamesCacheManager{
		redis: NewSimpleRedisClient(),
		ttl:   5 * time.Minute,
	}

	// Start with empty cache
	cache.Set(make(map[string]string))

	// Add name
	cache.AddName("New Organization")

	// Check it was added
	isTaken, originalName := cache.IsNameTaken("new organization")
	assert.True(t, isTaken)
	assert.Equal(t, "New Organization", originalName)

	// Remove name
	cache.RemoveName("New Organization")

	// Check it was removed
	isTaken, _ = cache.IsNameTaken("new organization")
	assert.False(t, isTaken)
}

func TestOrganizationNamesCacheManager_CacheMiss(t *testing.T) {
	// Create cache manager with empty Redis (cache miss)
	cache := &OrganizationNamesCacheManager{
		redis: NewSimpleRedisClient(),
		ttl:   5 * time.Minute,
	}

	// Try to get from cache (should be empty)
	_, found := cache.Get()
	assert.False(t, found)

	// Try to check name (should not be found)
	isTaken, _ := cache.IsNameTaken("test")
	assert.False(t, isTaken)
}

func TestOrganizationNamesCacheManager_WithoutRedis(t *testing.T) {
	// Create cache manager without Redis
	cache := &OrganizationNamesCacheManager{
		redis: nil,
		ttl:   5 * time.Minute,
	}

	// All operations should be no-ops
	cache.Set(map[string]string{"test": "Test"})
	cache.AddName("test")
	cache.RemoveName("test")
	cache.Clear()

	// Get should return false
	_, found := cache.Get()
	assert.False(t, found)

	// IsNameTaken should return false
	isTaken, _ := cache.IsNameTaken("test")
	assert.False(t, isTaken)
}

func TestOrganizationNamesCacheManager_ExactMatchBehavior(t *testing.T) {
	// Create cache manager with simple Redis
	cache := &OrganizationNamesCacheManager{
		redis: NewSimpleRedisClient(),
		ttl:   5 * time.Minute,
	}

	// Test the exact scenario from the original problem
	testNames := map[string]string{
		"erre":        "erre",
		"edoardo srl": "Edoardo SRL",
	}

	// Set cache
	cache.Set(testNames)

	// Test that "R" is NOT found even though it's a substring of "erre"
	isTaken, _ := cache.IsNameTaken("R")
	assert.False(t, isTaken, "R should not be found as it's not an exact match")

	// Test that "r" is NOT found even though it's a substring of "erre"
	isTaken, _ = cache.IsNameTaken("r")
	assert.False(t, isTaken, "r should not be found as it's not an exact match")

	// Test that exact matches still work
	isTaken, originalName := cache.IsNameTaken("erre")
	assert.True(t, isTaken)
	assert.Equal(t, "erre", originalName)

	isTaken, originalName = cache.IsNameTaken("ERRE")
	assert.True(t, isTaken)
	assert.Equal(t, "erre", originalName)

	isTaken, originalName = cache.IsNameTaken("edoardo srl")
	assert.True(t, isTaken)
	assert.Equal(t, "Edoardo SRL", originalName)

	isTaken, originalName = cache.IsNameTaken("EDOARDO SRL")
	assert.True(t, isTaken)
	assert.Equal(t, "Edoardo SRL", originalName)
}
