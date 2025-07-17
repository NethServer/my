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
	"testing"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/logger"
)

// Test utility functions to increase coverage
func TestUtilityFunctions(t *testing.T) {
	logger.SetLevel("fatal")

	t.Run("test CreateRoleNameToIDMapping", func(t *testing.T) {
		roles := []client.LogtoRole{
			{Name: "Admin", ID: "role1"},
			{Name: "User", ID: "role2"},
			{Name: "SUPPORT", ID: "role3"},
		}

		mapping := CreateRoleNameToIDMapping(roles)

		expectedMappings := map[string]string{
			"admin":   "role1",
			"user":    "role2",
			"support": "role3",
		}

		if len(mapping) != len(expectedMappings) {
			t.Errorf("expected %d mappings, got %d", len(expectedMappings), len(mapping))
		}

		for expectedName, expectedID := range expectedMappings {
			if actualID, exists := mapping[expectedName]; !exists {
				t.Errorf("expected mapping for %s to exist", expectedName)
			} else if actualID != expectedID {
				t.Errorf("expected mapping %s -> %s, got %s -> %s", expectedName, expectedID, expectedName, actualID)
			}
		}
	})

	t.Run("test CreateResourceNameToIDMapping", func(t *testing.T) {
		resources := []client.LogtoResource{
			{Name: "systems", ID: "res1"},
			{Name: "users", ID: "res2"},
			{Name: "accounts", ID: "res3"},
		}

		mapping := CreateResourceNameToIDMapping(resources)

		expectedMappings := map[string]string{
			"systems":  "res1",
			"users":    "res2",
			"accounts": "res3",
		}

		if len(mapping) != len(expectedMappings) {
			t.Errorf("expected %d mappings, got %d", len(expectedMappings), len(mapping))
		}

		for expectedName, expectedID := range expectedMappings {
			if actualID, exists := mapping[expectedName]; !exists {
				t.Errorf("expected mapping for %s to exist", expectedName)
			} else if actualID != expectedID {
				t.Errorf("expected mapping %s -> %s, got %s -> %s", expectedName, expectedID, expectedName, actualID)
			}
		}
	})

	t.Run("test CreateScopeNameToIDMapping", func(t *testing.T) {
		scopes := []client.LogtoScope{
			{Name: "read:systems", ID: "scope1"},
			{Name: "write:systems", ID: "scope2"},
			{Name: "admin:systems", ID: "scope3"},
		}

		mapping := CreateScopeNameToIDMapping(scopes)

		expectedMappings := map[string]string{
			"read:systems":  "scope1",
			"write:systems": "scope2",
			"admin:systems": "scope3",
		}

		if len(mapping) != len(expectedMappings) {
			t.Errorf("expected %d mappings, got %d", len(expectedMappings), len(mapping))
		}

		for expectedName, expectedID := range expectedMappings {
			if actualID, exists := mapping[expectedName]; !exists {
				t.Errorf("expected mapping for %s to exist", expectedName)
			} else if actualID != expectedID {
				t.Errorf("expected mapping %s -> %s, got %s -> %s", expectedName, expectedID, expectedName, actualID)
			}
		}
	})

	t.Run("test NewSystemEntityDetector", func(t *testing.T) {
		detector := NewSystemEntityDetector()

		if detector == nil {
			t.Fatal("expected detector to be created")
		}

		// This check is safe because t.Fatal above would terminate if detector was nil
		if detector != nil && len(detector.systemPatterns) == 0 {
			t.Error("expected detector to have system patterns")
		}

		// Check that expected patterns are present
		expectedPatterns := []string{"logto:", "urn:logto:", "management api", "machine to machine"}
		for _, expectedPattern := range expectedPatterns {
			found := false
			for _, pattern := range detector.systemPatterns {
				if pattern == expectedPattern {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected pattern %s to be in system patterns", expectedPattern)
			}
		}
	})

	t.Run("test SystemEntityDetector.IsSystemEntity", func(t *testing.T) {
		detector := NewSystemEntityDetector()

		tests := []struct {
			name        string
			description string
			expected    bool
		}{
			{"logto:admin", "Admin role", true},
			{"urn:logto:resource", "Resource description", true},
			{"custom-role", "Management API role", true},
			{"m2m-role", "Machine to Machine role", true},
			{"custom-role", "Custom role description", false},
			{"user-role", "Regular user role", false},
			{"", "", false},
		}

		for _, test := range tests {
			result := detector.IsSystemEntity(test.name, test.description)
			if result != test.expected {
				t.Errorf("IsSystemEntity(%s, %s) = %v, expected %v", test.name, test.description, result, test.expected)
			}
		}
	})

	t.Run("test CalculatePermissionDiff", func(t *testing.T) {
		tests := []struct {
			name        string
			current     []string
			desired     []string
			expectedAdd []string
			expectedDel []string
		}{
			{
				name:        "add permissions",
				current:     []string{"read:systems"},
				desired:     []string{"read:systems", "write:systems"},
				expectedAdd: []string{"write:systems"},
				expectedDel: []string{},
			},
			{
				name:        "remove permissions",
				current:     []string{"read:systems", "write:systems"},
				desired:     []string{"read:systems"},
				expectedAdd: []string{},
				expectedDel: []string{"write:systems"},
			},
			{
				name:        "no changes",
				current:     []string{"read:systems", "write:systems"},
				desired:     []string{"read:systems", "write:systems"},
				expectedAdd: []string{},
				expectedDel: []string{},
			},
			{
				name:        "mixed changes",
				current:     []string{"read:systems", "admin:systems"},
				desired:     []string{"write:systems", "admin:systems"},
				expectedAdd: []string{"write:systems"},
				expectedDel: []string{"read:systems"},
			},
			{
				name:        "system permission preserved",
				current:     []string{"logto:admin", "read:systems"},
				desired:     []string{"write:systems"},
				expectedAdd: []string{"write:systems"},
				expectedDel: []string{"read:systems"}, // logto:admin should be preserved
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				diff := CalculatePermissionDiff(test.current, test.desired)

				if len(diff.ToAdd) != len(test.expectedAdd) {
					t.Errorf("expected %d permissions to add, got %d", len(test.expectedAdd), len(diff.ToAdd))
				}

				if len(diff.ToRemove) != len(test.expectedDel) {
					t.Errorf("expected %d permissions to remove, got %d", len(test.expectedDel), len(diff.ToRemove))
				}

				// Check ToAdd
				for _, expected := range test.expectedAdd {
					found := false
					for _, actual := range diff.ToAdd {
						if actual == expected {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected permission %s to be added", expected)
					}
				}

				// Check ToRemove
				for _, expected := range test.expectedDel {
					found := false
					for _, actual := range diff.ToRemove {
						if actual == expected {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected permission %s to be removed", expected)
					}
				}
			})
		}
	})
}

// Test edge cases and error conditions
func TestUtilityEdgeCases(t *testing.T) {
	logger.SetLevel("fatal")

	t.Run("test empty inputs", func(t *testing.T) {
		// Test with empty role list
		roleMapping := CreateRoleNameToIDMapping([]client.LogtoRole{})
		if len(roleMapping) != 0 {
			t.Error("expected empty mapping for empty role list")
		}

		// Test with empty resource list
		resourceMapping := CreateResourceNameToIDMapping([]client.LogtoResource{})
		if len(resourceMapping) != 0 {
			t.Error("expected empty mapping for empty resource list")
		}

		// Test with empty scope list
		scopeMapping := CreateScopeNameToIDMapping([]client.LogtoScope{})
		if len(scopeMapping) != 0 {
			t.Error("expected empty mapping for empty scope list")
		}

		// Test permission diff with empty lists
		diff := CalculatePermissionDiff([]string{}, []string{})
		if len(diff.ToAdd) != 0 || len(diff.ToRemove) != 0 {
			t.Error("expected empty diff for empty permission lists")
		}
	})

	t.Run("test case sensitivity", func(t *testing.T) {
		detector := NewSystemEntityDetector()

		// Test case insensitive detection
		if !detector.IsSystemEntity("LOGTO:ADMIN", "") {
			t.Error("expected case insensitive system entity detection")
		}

		if !detector.IsSystemEntity("", "MANAGEMENT API role") {
			t.Error("expected case insensitive system entity detection in description")
		}
	})

	t.Run("test duplicate names in mappings", func(t *testing.T) {
		// Test roles with duplicate names (case differences)
		roles := []client.LogtoRole{
			{Name: "Admin", ID: "role1"},
			{Name: "ADMIN", ID: "role2"}, // Should overwrite the first one
		}

		mapping := CreateRoleNameToIDMapping(roles)

		// Both should map to the same lowercase key, second should overwrite first
		if mapping["admin"] != "role2" {
			t.Errorf("expected duplicate name to be overwritten, got %s", mapping["admin"])
		}
	})
}
