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

	"github.com/nethesis/my/sync/internal/config"
	"github.com/nethesis/my/sync/internal/logger"
)

// Test role utility functions
func TestRoleUtilityFunctions(t *testing.T) {
	logger.SetLevel("fatal")

	t.Run("test convertIDsToNames", func(t *testing.T) {
		engine := &Engine{options: &Options{}}

		scopeMapping := &ScopeMapping{
			IDToName: map[string]string{
				"scope1": "read:systems",
				"scope2": "write:systems",
				"scope3": "admin:systems",
			},
		}

		currentIDs := []string{"scope1", "scope2", "unknown"}
		expectedIDs := []string{"scope2", "scope3", "missing"}

		result := engine.convertIDsToNames(currentIDs, expectedIDs, scopeMapping)

		// Check current names
		expectedCurrent := []string{"read:systems", "write:systems"}
		if len(result.Current) != len(expectedCurrent) {
			t.Errorf("expected %d current permissions, got %d", len(expectedCurrent), len(result.Current))
		}

		for i, expected := range expectedCurrent {
			if i >= len(result.Current) || result.Current[i] != expected {
				t.Errorf("expected current[%d] = %s, got %s", i, expected, result.Current[i])
			}
		}

		// Check expected names
		expectedExpected := []string{"write:systems", "admin:systems"}
		if len(result.Expected) != len(expectedExpected) {
			t.Errorf("expected %d expected permissions, got %d", len(expectedExpected), len(result.Expected))
		}

		for i, expected := range expectedExpected {
			if i >= len(result.Expected) || result.Expected[i] != expected {
				t.Errorf("expected expected[%d] = %s, got %s", i, expected, result.Expected[i])
			}
		}
	})

	t.Run("test buildExpectedPermissionIDs", func(t *testing.T) {
		engine := &Engine{options: &Options{}}

		scopeMapping := &ScopeMapping{
			NameToID: map[string]string{
				"read:systems":  "scope1",
				"write:systems": "scope2",
				"admin:systems": "scope3",
			},
		}

		configRole := config.Role{
			Name: "test-role",
			Permissions: []config.Permission{
				{ID: "read:systems"},
				{ID: "write:systems"},
				{ID: "unknown:permission"}, // Should be skipped
				{ID: ""},                   // Should be skipped
			},
		}

		expectedIDs := engine.buildExpectedPermissionIDs(configRole, scopeMapping)

		expectedResult := []string{"scope1", "scope2"}
		if len(expectedIDs) != len(expectedResult) {
			t.Errorf("expected %d permission IDs, got %d", len(expectedResult), len(expectedIDs))
		}

		for i, expected := range expectedResult {
			if i >= len(expectedIDs) || expectedIDs[i] != expected {
				t.Errorf("expected permission ID[%d] = %s, got %s", i, expected, expectedIDs[i])
			}
		}
	})

	t.Run("test applyUserRolePermissionChanges - dry run", func(t *testing.T) {
		engine := &Engine{
			options: &Options{DryRun: true},
		}

		result := &Result{
			Summary:    &Summary{},
			Operations: []Operation{},
		}

		scopeMapping := &ScopeMapping{
			NameToID: map[string]string{
				"read:systems":  "scope1",
				"write:systems": "scope2",
			},
		}

		diff := &PermissionDiff{
			ToAdd:    []string{"read:systems"},
			ToRemove: []string{"write:systems"},
		}

		err := engine.applyUserRolePermissionChanges("role1", "test-role", diff, scopeMapping, result)

		// Should not error in dry run
		if err != nil {
			t.Errorf("expected no error in dry run, got %v", err)
		}

		// Check summary counts
		if result.Summary.PermissionsCreated != 1 {
			t.Errorf("expected 1 permission created, got %d", result.Summary.PermissionsCreated)
		}

		if result.Summary.PermissionsDeleted != 1 {
			t.Errorf("expected 1 permission deleted, got %d", result.Summary.PermissionsDeleted)
		}

		// Check operations were recorded
		if len(result.Operations) != 2 {
			t.Errorf("expected 2 operations, got %d", len(result.Operations))
		}

		// Check assign operation
		assignOp := result.Operations[0]
		if assignOp.Type != "user-role-permission" || assignOp.Action != "assign" {
			t.Errorf("expected assign operation, got type=%s action=%s", assignOp.Type, assignOp.Action)
		}

		// Check remove operation
		removeOp := result.Operations[1]
		if removeOp.Type != "user-role-permission" || removeOp.Action != "remove" {
			t.Errorf("expected remove operation, got type=%s action=%s", removeOp.Type, removeOp.Action)
		}
	})

	t.Run("test applyUserRolePermissionChanges - no changes", func(t *testing.T) {
		engine := &Engine{
			options: &Options{DryRun: true},
		}

		result := &Result{
			Summary:    &Summary{},
			Operations: []Operation{},
		}

		scopeMapping := &ScopeMapping{
			NameToID: map[string]string{},
		}

		diff := &PermissionDiff{
			ToAdd:    []string{},
			ToRemove: []string{},
		}

		err := engine.applyUserRolePermissionChanges("role1", "test-role", diff, scopeMapping, result)

		// Should not error
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Check no operations were recorded
		if len(result.Operations) != 0 {
			t.Errorf("expected 0 operations, got %d", len(result.Operations))
		}

		// Check no summary changes
		if result.Summary.PermissionsCreated != 0 || result.Summary.PermissionsDeleted != 0 {
			t.Error("expected no permission changes in summary")
		}
	})

	t.Run("test applyUserRolePermissionChanges - unknown permissions", func(t *testing.T) {
		engine := &Engine{
			options: &Options{DryRun: true},
		}

		result := &Result{
			Summary:    &Summary{},
			Operations: []Operation{},
		}

		scopeMapping := &ScopeMapping{
			NameToID: map[string]string{}, // Empty mapping
		}

		diff := &PermissionDiff{
			ToAdd:    []string{"unknown:permission"},
			ToRemove: []string{"another:unknown"},
		}

		err := engine.applyUserRolePermissionChanges("role1", "test-role", diff, scopeMapping, result)

		// Should not error
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Check no operations were recorded (since no permissions were found)
		if len(result.Operations) != 0 {
			t.Errorf("expected 0 operations, got %d", len(result.Operations))
		}
	})
}

// Test edge cases for role utilities
func TestRoleUtilityEdgeCases(t *testing.T) {
	logger.SetLevel("fatal")

	t.Run("test convertIDsToNames with empty mapping", func(t *testing.T) {
		engine := &Engine{options: &Options{}}

		scopeMapping := &ScopeMapping{
			IDToName: map[string]string{}, // Empty mapping
		}

		currentIDs := []string{"scope1", "scope2"}
		expectedIDs := []string{"scope3"}

		result := engine.convertIDsToNames(currentIDs, expectedIDs, scopeMapping)

		// Should return empty slices
		if len(result.Current) != 0 {
			t.Errorf("expected 0 current permissions, got %d", len(result.Current))
		}

		if len(result.Expected) != 0 {
			t.Errorf("expected 0 expected permissions, got %d", len(result.Expected))
		}
	})

	t.Run("test buildExpectedPermissionIDs with empty role", func(t *testing.T) {
		engine := &Engine{options: &Options{}}

		scopeMapping := &ScopeMapping{
			NameToID: map[string]string{
				"read:systems": "scope1",
			},
		}

		configRole := config.Role{
			Name:        "empty-role",
			Permissions: []config.Permission{}, // No permissions
		}

		expectedIDs := engine.buildExpectedPermissionIDs(configRole, scopeMapping)

		if len(expectedIDs) != 0 {
			t.Errorf("expected 0 permission IDs for empty role, got %d", len(expectedIDs))
		}
	})

	t.Run("test buildExpectedPermissionIDs with only empty permissions", func(t *testing.T) {
		engine := &Engine{options: &Options{}}

		scopeMapping := &ScopeMapping{
			NameToID: map[string]string{
				"read:systems": "scope1",
			},
		}

		configRole := config.Role{
			Name: "role-with-empty-perms",
			Permissions: []config.Permission{
				{ID: ""},   // Empty ID
				{Name: ""}, // Only name, no ID
			},
		}

		expectedIDs := engine.buildExpectedPermissionIDs(configRole, scopeMapping)

		if len(expectedIDs) != 0 {
			t.Errorf("expected 0 permission IDs for role with empty permissions, got %d", len(expectedIDs))
		}
	})
}
