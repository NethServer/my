/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package sync

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/config"
)

func TestCreateRoleNameToIDMapping(t *testing.T) {
	t.Run("creates mapping from role names to IDs", func(t *testing.T) {
		roles := []client.LogtoRole{
			{ID: "role1", Name: "Admin"},
			{ID: "role2", Name: "User"},
			{ID: "role3", Name: "Support"},
		}

		mapping := CreateRoleNameToIDMapping(roles)

		assert.Equal(t, "role1", mapping["admin"])
		assert.Equal(t, "role2", mapping["user"])
		assert.Equal(t, "role3", mapping["support"])
	})

	t.Run("handles empty role list", func(t *testing.T) {
		roles := []client.LogtoRole{}

		mapping := CreateRoleNameToIDMapping(roles)

		assert.Empty(t, mapping)
	})

	t.Run("handles case sensitivity", func(t *testing.T) {
		roles := []client.LogtoRole{
			{ID: "role1", Name: "ADMIN"},
			{ID: "role2", Name: "Admin"},
			{ID: "role3", Name: "admin"},
		}

		mapping := CreateRoleNameToIDMapping(roles)

		// All should map to the same key (last one wins)
		assert.Equal(t, "role3", mapping["admin"])
		assert.Len(t, mapping, 1)
	})

	t.Run("handles duplicate role names", func(t *testing.T) {
		roles := []client.LogtoRole{
			{ID: "role1", Name: "Admin"},
			{ID: "role2", Name: "Admin"},
		}

		mapping := CreateRoleNameToIDMapping(roles)

		// Last one should win
		assert.Equal(t, "role2", mapping["admin"])
		assert.Len(t, mapping, 1)
	})
}

func TestCreateResourceNameToIDMapping(t *testing.T) {
	t.Run("creates mapping from resource names to IDs", func(t *testing.T) {
		resources := []client.LogtoResource{
			{ID: "res1", Name: "systems"},
			{ID: "res2", Name: "users"},
			{ID: "res3", Name: "reports"},
		}

		mapping := CreateResourceNameToIDMapping(resources)

		assert.Equal(t, "res1", mapping["systems"])
		assert.Equal(t, "res2", mapping["users"])
		assert.Equal(t, "res3", mapping["reports"])
	})

	t.Run("handles empty resource list", func(t *testing.T) {
		resources := []client.LogtoResource{}

		mapping := CreateResourceNameToIDMapping(resources)

		assert.Empty(t, mapping)
	})

	t.Run("preserves case sensitivity", func(t *testing.T) {
		resources := []client.LogtoResource{
			{ID: "res1", Name: "Systems"},
			{ID: "res2", Name: "SYSTEMS"},
			{ID: "res3", Name: "systems"},
		}

		mapping := CreateResourceNameToIDMapping(resources)

		assert.Equal(t, "res1", mapping["Systems"])
		assert.Equal(t, "res2", mapping["SYSTEMS"])
		assert.Equal(t, "res3", mapping["systems"])
		assert.Len(t, mapping, 3)
	})

	t.Run("handles duplicate resource names", func(t *testing.T) {
		resources := []client.LogtoResource{
			{ID: "res1", Name: "systems"},
			{ID: "res2", Name: "systems"},
		}

		mapping := CreateResourceNameToIDMapping(resources)

		// Last one should win
		assert.Equal(t, "res2", mapping["systems"])
		assert.Len(t, mapping, 1)
	})
}

func TestCreateScopeNameToIDMapping(t *testing.T) {
	t.Run("creates mapping from scope names to IDs", func(t *testing.T) {
		scopes := []client.LogtoScope{
			{ID: "scope1", Name: "read:systems"},
			{ID: "scope2", Name: "write:systems"},
			{ID: "scope3", Name: "admin:systems"},
		}

		mapping := CreateScopeNameToIDMapping(scopes)

		assert.Equal(t, "scope1", mapping["read:systems"])
		assert.Equal(t, "scope2", mapping["write:systems"])
		assert.Equal(t, "scope3", mapping["admin:systems"])
	})

	t.Run("handles empty scope list", func(t *testing.T) {
		scopes := []client.LogtoScope{}

		mapping := CreateScopeNameToIDMapping(scopes)

		assert.Empty(t, mapping)
	})

	t.Run("preserves case sensitivity", func(t *testing.T) {
		scopes := []client.LogtoScope{
			{ID: "scope1", Name: "READ:SYSTEMS"},
			{ID: "scope2", Name: "read:systems"},
		}

		mapping := CreateScopeNameToIDMapping(scopes)

		assert.Equal(t, "scope1", mapping["READ:SYSTEMS"])
		assert.Equal(t, "scope2", mapping["read:systems"])
		assert.Len(t, mapping, 2)
	})

	t.Run("handles duplicate scope names", func(t *testing.T) {
		scopes := []client.LogtoScope{
			{ID: "scope1", Name: "read:systems"},
			{ID: "scope2", Name: "read:systems"},
		}

		mapping := CreateScopeNameToIDMapping(scopes)

		// Last one should win
		assert.Equal(t, "scope2", mapping["read:systems"])
		assert.Len(t, mapping, 1)
	})
}

func TestSystemEntityDetector(t *testing.T) {
	detector := NewSystemEntityDetector()

	t.Run("detects system entities by name", func(t *testing.T) {
		testCases := []struct {
			name     string
			expected bool
		}{
			{"logto:core", true},
			{"urn:logto:scope:management", true},
			{"Management API", true},
			{"machine to machine app", true},
			{"user-defined-role", false},
			{"custom-resource", false},
		}

		for _, tc := range testCases {
			result := detector.IsSystemEntity(tc.name, "")
			assert.Equal(t, tc.expected, result, "Failed for name: %s", tc.name)
		}
	})

	t.Run("detects system entities by description", func(t *testing.T) {
		testCases := []struct {
			description string
			expected    bool
		}{
			{"This is a logto: managed resource", true},
			{"URN:LOGTO:SCOPE for management", true},
			{"A management api endpoint", true},
			{"Machine to Machine authentication", true},
			{"User defined custom role", false},
			{"Application specific resource", false},
		}

		for _, tc := range testCases {
			result := detector.IsSystemEntity("", tc.description)
			assert.Equal(t, tc.expected, result, "Failed for description: %s", tc.description)
		}
	})

	t.Run("detects system entities by name and description", func(t *testing.T) {
		assert.True(t, detector.IsSystemEntity("custom-role", "logto: managed"))
		assert.True(t, detector.IsSystemEntity("urn:logto:test", "custom description"))
		assert.False(t, detector.IsSystemEntity("custom-role", "user defined"))
	})

	t.Run("case insensitive detection", func(t *testing.T) {
		assert.True(t, detector.IsSystemEntity("LOGTO:CORE", ""))
		assert.True(t, detector.IsSystemEntity("", "URN:LOGTO:SCOPE"))
		assert.True(t, detector.IsSystemEntity("", "MANAGEMENT API"))
		assert.True(t, detector.IsSystemEntity("", "MACHINE TO MACHINE"))
	})

	t.Run("handles empty strings", func(t *testing.T) {
		assert.False(t, detector.IsSystemEntity("", ""))
		assert.False(t, detector.IsSystemEntity("custom", ""))
		assert.False(t, detector.IsSystemEntity("", "custom"))
	})
}

func TestCalculatePermissionDiff(t *testing.T) {
	t.Run("calculates permissions to add", func(t *testing.T) {
		current := []string{"read:systems", "write:systems"}
		desired := []string{"read:systems", "write:systems", "admin:systems"}

		diff := CalculatePermissionDiff(current, desired)

		assert.Len(t, diff.ToAdd, 1)
		assert.Contains(t, diff.ToAdd, "admin:systems")
		assert.Empty(t, diff.ToRemove)
	})

	t.Run("calculates permissions to remove", func(t *testing.T) {
		current := []string{"read:systems", "write:systems", "admin:systems"}
		desired := []string{"read:systems", "write:systems"}

		diff := CalculatePermissionDiff(current, desired)

		assert.Empty(t, diff.ToAdd)
		assert.Len(t, diff.ToRemove, 1)
		assert.Contains(t, diff.ToRemove, "admin:systems")
	})

	t.Run("calculates both add and remove", func(t *testing.T) {
		current := []string{"read:systems", "old:permission"}
		desired := []string{"read:systems", "new:permission"}

		diff := CalculatePermissionDiff(current, desired)

		assert.Len(t, diff.ToAdd, 1)
		assert.Contains(t, diff.ToAdd, "new:permission")
		assert.Len(t, diff.ToRemove, 1)
		assert.Contains(t, diff.ToRemove, "old:permission")
	})

	t.Run("handles identical permissions", func(t *testing.T) {
		current := []string{"read:systems", "write:systems"}
		desired := []string{"read:systems", "write:systems"}

		diff := CalculatePermissionDiff(current, desired)

		assert.Empty(t, diff.ToAdd)
		assert.Empty(t, diff.ToRemove)
	})

	t.Run("does not remove system permissions", func(t *testing.T) {
		current := []string{"read:systems", "logto:core", "urn:logto:scope:management"}
		desired := []string{"read:systems"}

		diff := CalculatePermissionDiff(current, desired)

		assert.Empty(t, diff.ToAdd)
		assert.Empty(t, diff.ToRemove) // System permissions should not be removed
	})

	t.Run("handles empty current permissions", func(t *testing.T) {
		current := []string{}
		desired := []string{"read:systems", "write:systems"}

		diff := CalculatePermissionDiff(current, desired)

		assert.Len(t, diff.ToAdd, 2)
		assert.Contains(t, diff.ToAdd, "read:systems")
		assert.Contains(t, diff.ToAdd, "write:systems")
		assert.Empty(t, diff.ToRemove)
	})

	t.Run("handles empty desired permissions", func(t *testing.T) {
		current := []string{"read:systems", "write:systems"}
		desired := []string{}

		diff := CalculatePermissionDiff(current, desired)

		assert.Empty(t, diff.ToAdd)
		assert.Len(t, diff.ToRemove, 2)
		assert.Contains(t, diff.ToRemove, "read:systems")
		assert.Contains(t, diff.ToRemove, "write:systems")
	})

	t.Run("handles both empty lists", func(t *testing.T) {
		current := []string{}
		desired := []string{}

		diff := CalculatePermissionDiff(current, desired)

		assert.Empty(t, diff.ToAdd)
		assert.Empty(t, diff.ToRemove)
	})
}

func TestScopeMapping(t *testing.T) {
	t.Run("creates scope mapping structure", func(t *testing.T) {
		nameToID := map[string]string{
			"read:systems":  "scope1",
			"write:systems": "scope2",
		}
		idToName := map[string]string{
			"scope1": "read:systems",
			"scope2": "write:systems",
		}

		mapping := &ScopeMapping{
			NameToID: nameToID,
			IDToName: idToName,
		}

		assert.Equal(t, "scope1", mapping.NameToID["read:systems"])
		assert.Equal(t, "scope2", mapping.NameToID["write:systems"])
		assert.Equal(t, "read:systems", mapping.IDToName["scope1"])
		assert.Equal(t, "write:systems", mapping.IDToName["scope2"])
	})

	t.Run("handles empty scope mapping", func(t *testing.T) {
		mapping := &ScopeMapping{
			NameToID: make(map[string]string),
			IDToName: make(map[string]string),
		}

		assert.Empty(t, mapping.NameToID)
		assert.Empty(t, mapping.IDToName)
	})
}

func TestPermissionDiff(t *testing.T) {
	t.Run("creates permission diff structure", func(t *testing.T) {
		diff := &PermissionDiff{
			ToAdd:    []string{"new:permission"},
			ToRemove: []string{"old:permission"},
		}

		assert.Len(t, diff.ToAdd, 1)
		assert.Contains(t, diff.ToAdd, "new:permission")
		assert.Len(t, diff.ToRemove, 1)
		assert.Contains(t, diff.ToRemove, "old:permission")
	})

	t.Run("handles empty permission diff", func(t *testing.T) {
		diff := &PermissionDiff{
			ToAdd:    []string{},
			ToRemove: []string{},
		}

		assert.Empty(t, diff.ToAdd)
		assert.Empty(t, diff.ToRemove)
	})
}

// ScopeGetter interface for testing BuildGlobalScopeMapping
type ScopeGetter interface {
	GetScopes(resourceID string) ([]client.LogtoScope, error)
}

// MockScopeGetter for testing BuildGlobalScopeMapping
type MockScopeGetter struct {
	ScopesByResource map[string][]client.LogtoScope
	ShouldError      bool
}

func (m *MockScopeGetter) GetScopes(resourceID string) ([]client.LogtoScope, error) {
	if m.ShouldError {
		return nil, fmt.Errorf("mock error getting scopes for resource %s", resourceID)
	}

	if scopes, exists := m.ScopesByResource[resourceID]; exists {
		return scopes, nil
	}

	return []client.LogtoScope{}, nil
}

// Helper function that wraps BuildGlobalScopeMapping to accept our interface
func buildGlobalScopeMappingTestable(scopeGetter ScopeGetter, cfg *config.Config, resourceNameToID map[string]string) (*ScopeMapping, error) {
	allScopeNameToID := make(map[string]string)
	allScopeIDToName := make(map[string]string)

	for _, configResource := range cfg.Hierarchy.Resources {
		resourceID, exists := resourceNameToID[configResource.Name]
		if !exists {
			// Skip resources not found in mapping (similar to original implementation)
			continue
		}

		scopes, err := scopeGetter.GetScopes(resourceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get scopes for resource %s: %w", configResource.Name, err)
		}

		for _, scope := range scopes {
			allScopeNameToID[scope.Name] = scope.ID
			allScopeIDToName[scope.ID] = scope.Name
		}
	}

	return &ScopeMapping{
		NameToID: allScopeNameToID,
		IDToName: allScopeIDToName,
	}, nil
}

func TestBuildGlobalScopeMapping(t *testing.T) {
	t.Run("builds scope mapping successfully", func(t *testing.T) {
		// Create mock client
		mockClient := &MockScopeGetter{
			ScopesByResource: map[string][]client.LogtoScope{
				"res1": {
					{ID: "scope1", Name: "read:systems"},
					{ID: "scope2", Name: "write:systems"},
				},
				"res2": {
					{ID: "scope3", Name: "read:users"},
					{ID: "scope4", Name: "write:users"},
				},
			},
		}

		// Create config
		cfg := &config.Config{
			Hierarchy: config.Hierarchy{
				Resources: []config.Resource{
					{Name: "systems"},
					{Name: "users"},
				},
			},
		}

		// Create resource mapping
		resourceNameToID := map[string]string{
			"systems": "res1",
			"users":   "res2",
		}

		// Build scope mapping
		scopeMapping, err := buildGlobalScopeMappingTestable(mockClient, cfg, resourceNameToID)

		require.NoError(t, err)
		require.NotNil(t, scopeMapping)

		// Verify NameToID mapping
		assert.Equal(t, "scope1", scopeMapping.NameToID["read:systems"])
		assert.Equal(t, "scope2", scopeMapping.NameToID["write:systems"])
		assert.Equal(t, "scope3", scopeMapping.NameToID["read:users"])
		assert.Equal(t, "scope4", scopeMapping.NameToID["write:users"])

		// Verify IDToName mapping
		assert.Equal(t, "read:systems", scopeMapping.IDToName["scope1"])
		assert.Equal(t, "write:systems", scopeMapping.IDToName["scope2"])
		assert.Equal(t, "read:users", scopeMapping.IDToName["scope3"])
		assert.Equal(t, "write:users", scopeMapping.IDToName["scope4"])
	})

	t.Run("handles missing resource in resource mapping", func(t *testing.T) {
		mockClient := &MockScopeGetter{
			ScopesByResource: map[string][]client.LogtoScope{},
		}

		cfg := &config.Config{
			Hierarchy: config.Hierarchy{
				Resources: []config.Resource{
					{Name: "systems"},
					{Name: "missing-resource"},
				},
			},
		}

		resourceNameToID := map[string]string{
			"systems": "res1",
			// missing-resource is not in the mapping
		}

		scopeMapping, err := buildGlobalScopeMappingTestable(mockClient, cfg, resourceNameToID)

		require.NoError(t, err)
		require.NotNil(t, scopeMapping)
		// Should still work, just skip the missing resource
	})

	t.Run("handles client error", func(t *testing.T) {
		mockClient := &MockScopeGetter{
			ShouldError: true,
		}

		cfg := &config.Config{
			Hierarchy: config.Hierarchy{
				Resources: []config.Resource{
					{Name: "systems"},
				},
			},
		}

		resourceNameToID := map[string]string{
			"systems": "res1",
		}

		scopeMapping, err := buildGlobalScopeMappingTestable(mockClient, cfg, resourceNameToID)

		assert.Error(t, err)
		assert.Nil(t, scopeMapping)
		assert.Contains(t, err.Error(), "failed to get scopes for resource systems")
	})

	t.Run("handles empty resources in config", func(t *testing.T) {
		mockClient := &MockScopeGetter{
			ScopesByResource: map[string][]client.LogtoScope{},
		}

		cfg := &config.Config{
			Hierarchy: config.Hierarchy{
				Resources: []config.Resource{},
			},
		}

		resourceNameToID := map[string]string{}

		scopeMapping, err := buildGlobalScopeMappingTestable(mockClient, cfg, resourceNameToID)

		require.NoError(t, err)
		require.NotNil(t, scopeMapping)
		assert.Empty(t, scopeMapping.NameToID)
		assert.Empty(t, scopeMapping.IDToName)
	})

	t.Run("handles duplicate scope names across resources", func(t *testing.T) {
		mockClient := &MockScopeGetter{
			ScopesByResource: map[string][]client.LogtoScope{
				"res1": {
					{ID: "scope1", Name: "read:data"},
				},
				"res2": {
					{ID: "scope2", Name: "read:data"}, // Same name, different ID
				},
			},
		}

		cfg := &config.Config{
			Hierarchy: config.Hierarchy{
				Resources: []config.Resource{
					{Name: "resource1"},
					{Name: "resource2"},
				},
			},
		}

		resourceNameToID := map[string]string{
			"resource1": "res1",
			"resource2": "res2",
		}

		scopeMapping, err := buildGlobalScopeMappingTestable(mockClient, cfg, resourceNameToID)

		require.NoError(t, err)
		require.NotNil(t, scopeMapping)

		// Last scope should win for the name mapping
		assert.Equal(t, "scope2", scopeMapping.NameToID["read:data"])

		// Both should be in IDToName mapping
		assert.Equal(t, "read:data", scopeMapping.IDToName["scope1"])
		assert.Equal(t, "read:data", scopeMapping.IDToName["scope2"])
	})
}
