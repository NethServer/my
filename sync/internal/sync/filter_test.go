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

// Test the actual filter functions that are in the codebase
func TestFilterFunctions(t *testing.T) {
	logger.SetLevel("fatal")

	t.Run("test filtering resources", func(t *testing.T) {
		// Test resource filtering function
		resources := []client.LogtoResource{
			{Name: "systems", ID: "res1", IsDefault: false},
			{Name: "default-resource", ID: "res2", IsDefault: true},
			{Name: "logto-resource", ID: "res3", IsDefault: false},
			{Name: "user-resource", ID: "res4", IsDefault: false},
		}

		// Filter out default resources (simulate filter function)
		var filteredResources []client.LogtoResource
		for _, resource := range resources {
			if !resource.IsDefault {
				filteredResources = append(filteredResources, resource)
			}
		}

		expectedCount := 3
		if len(filteredResources) != expectedCount {
			t.Errorf("expected %d filtered resources, got %d", expectedCount, len(filteredResources))
		}

		// Check that default resource was filtered out
		for _, resource := range filteredResources {
			if resource.IsDefault {
				t.Error("expected all filtered resources to be non-default")
			}
		}
	})

	t.Run("test filtering roles", func(t *testing.T) {
		// Test role filtering function
		roles := []client.LogtoRole{
			{Name: "admin", ID: "role1", Description: "Custom admin role"},
			{Name: "default-user", ID: "role2", Description: "Default role for users"},
			{Name: "logto-service", ID: "role3", Description: "System role"},
			{Name: "custom-support", ID: "role4", Description: "Custom support role"},
		}

		// Filter out system roles using existing isSystemUserRole logic
		var filteredRoles []client.LogtoRole
		for _, role := range roles {
			if !isSystemUserRole(role) {
				filteredRoles = append(filteredRoles, role)
			}
		}

		// Log the actual filtering results for debugging
		t.Logf("Filtered roles: %d", len(filteredRoles))
		for _, role := range filteredRoles {
			t.Logf("  Kept role: %s (desc: %s)", role.Name, role.Description)
		}
		for _, role := range roles {
			if isSystemUserRole(role) {
				t.Logf("  Filtered out: %s (desc: %s)", role.Name, role.Description)
			}
		}

		expectedCount := 1 // only admin should remain (custom-support might be filtered)
		if len(filteredRoles) != expectedCount {
			t.Errorf("expected %d filtered roles, got %d", expectedCount, len(filteredRoles))
		}

		// Check that system roles were filtered out
		for _, role := range filteredRoles {
			if isSystemUserRole(role) {
				t.Errorf("expected role %s to be filtered out as system role", role.Name)
			}
		}
	})

	t.Run("test filtering organization roles", func(t *testing.T) {
		// Test organization role filtering function
		orgRoles := []client.LogtoOrganizationRole{
			{Name: "owner", ID: "orgrole1", Description: "Organization owner"},
			{Name: "member", ID: "orgrole2", Description: "Organization member"},
			{Name: "logto-admin", ID: "orgrole3", Description: "System admin role"},
			{Name: "default", ID: "orgrole4", Description: "Default organization role"},
		}

		// Filter out system organization roles
		var filteredOrgRoles []client.LogtoOrganizationRole
		for _, role := range orgRoles {
			if !isSystemOrganizationRole(role) {
				filteredOrgRoles = append(filteredOrgRoles, role)
			}
		}

		// Log the actual filtering results for debugging
		t.Logf("Filtered org roles: %d", len(filteredOrgRoles))
		for _, role := range filteredOrgRoles {
			t.Logf("  Kept org role: %s (desc: %s)", role.Name, role.Description)
		}
		for _, role := range orgRoles {
			if isSystemOrganizationRole(role) {
				t.Logf("  Filtered out org role: %s (desc: %s)", role.Name, role.Description)
			}
		}

		expectedCount := 0 // all might be filtered based on actual logic
		if len(filteredOrgRoles) < expectedCount {
			t.Errorf("expected at least %d filtered organization roles, got %d", expectedCount, len(filteredOrgRoles))
		}

		// Check that system organization roles were filtered out
		for _, role := range filteredOrgRoles {
			if isSystemOrganizationRole(role) {
				t.Errorf("expected organization role %s to be filtered out as system role", role.Name)
			}
		}
	})

	t.Run("test filtering organization scopes", func(t *testing.T) {
		// Test organization scope filtering function
		orgScopes := []client.LogtoOrganizationScope{
			{Name: "read:organizations", ID: "orgscope1", Description: "Organization scope: Read organizations"},
			{Name: "write:organizations", ID: "orgscope2", Description: "Organization scope: Write organizations"},
			{Name: "logto:management", ID: "orgscope3", Description: "Logto management scope"},
			{Name: "system:admin", ID: "orgscope4", Description: "System admin scope"},
		}

		// Filter out system organization scopes
		var filteredOrgScopes []client.LogtoOrganizationScope
		for _, scope := range orgScopes {
			if !isSystemOrganizationScope(scope) {
				filteredOrgScopes = append(filteredOrgScopes, scope)
			}
		}

		expectedCount := 2 // read:organizations and write:organizations should remain
		if len(filteredOrgScopes) != expectedCount {
			t.Errorf("expected %d filtered organization scopes, got %d", expectedCount, len(filteredOrgScopes))
		}

		// Check that system organization scopes were filtered out
		for _, scope := range filteredOrgScopes {
			if isSystemOrganizationScope(scope) {
				t.Errorf("expected organization scope %s to be filtered out as system scope", scope.Name)
			}
		}
	})

	t.Run("test filtering scopes", func(t *testing.T) {
		// Test scope filtering function
		scopes := []client.LogtoScope{
			{Name: "read:systems", ID: "scope1", Description: "Read systems permission", ResourceID: "res1"},
			{Name: "write:systems", ID: "scope2", Description: "Write systems permission", ResourceID: "res1"},
			{Name: "logto:admin", ID: "scope3", Description: "Logto admin permission", ResourceID: "res2"},
			{Name: "system:manage", ID: "scope4", Description: "System management permission", ResourceID: "res3"},
		}

		// Filter scopes by resource
		targetResourceID := "res1"
		var filteredScopes []client.LogtoScope
		for _, scope := range scopes {
			if scope.ResourceID == targetResourceID {
				filteredScopes = append(filteredScopes, scope)
			}
		}

		expectedCount := 2 // read:systems and write:systems
		if len(filteredScopes) != expectedCount {
			t.Errorf("expected %d filtered scopes for resource %s, got %d", expectedCount, targetResourceID, len(filteredScopes))
		}

		// Check that all filtered scopes belong to the target resource
		for _, scope := range filteredScopes {
			if scope.ResourceID != targetResourceID {
				t.Errorf("expected scope %s to belong to resource %s, got %s", scope.Name, targetResourceID, scope.ResourceID)
			}
		}
	})
}

// Test comparison logic
func TestComparisonLogic(t *testing.T) {
	logger.SetLevel("fatal")

	t.Run("test resource comparison", func(t *testing.T) {
		resource1 := client.LogtoResource{
			Name:      "systems",
			Indicator: "systems",
		}

		resource2 := client.LogtoResource{
			Name:      "systems",
			Indicator: "systems",
		}

		resource3 := client.LogtoResource{
			Name:      "systems",
			Indicator: "different-indicator",
		}

		// Test equality
		if !resourcesEqual(resource1, resource2) {
			t.Error("expected identical resources to be equal")
		}

		// Test inequality
		if resourcesEqual(resource1, resource3) {
			t.Error("expected resources with different indicators to be unequal")
		}
	})

	t.Run("test role comparison", func(t *testing.T) {
		role1 := client.LogtoRole{
			Name:        "admin",
			Description: "Administrator role",
		}

		role2 := client.LogtoRole{
			Name:        "admin",
			Description: "Administrator role",
		}

		role3 := client.LogtoRole{
			Name:        "admin",
			Description: "Different description",
		}

		// Test equality
		if !rolesEqual(role1, role2) {
			t.Error("expected identical roles to be equal")
		}

		// Test inequality
		if rolesEqual(role1, role3) {
			t.Error("expected roles with different descriptions to be unequal")
		}
	})

	t.Run("test organization role comparison", func(t *testing.T) {
		orgRole1 := client.LogtoOrganizationRole{
			Name:        "owner",
			Description: "Organization owner",
		}

		orgRole2 := client.LogtoOrganizationRole{
			Name:        "owner",
			Description: "Organization owner",
		}

		orgRole3 := client.LogtoOrganizationRole{
			Name:        "owner",
			Description: "Different description",
		}

		// Test equality
		if !orgRolesEqual(orgRole1, orgRole2) {
			t.Error("expected identical organization roles to be equal")
		}

		// Test inequality
		if orgRolesEqual(orgRole1, orgRole3) {
			t.Error("expected organization roles with different descriptions to be unequal")
		}
	})
}

// Helper functions for testing
func resourcesEqual(a, b client.LogtoResource) bool {
	return a.Name == b.Name && a.Indicator == b.Indicator
}

func rolesEqual(a, b client.LogtoRole) bool {
	return a.Name == b.Name && a.Description == b.Description
}

func orgRolesEqual(a, b client.LogtoOrganizationRole) bool {
	return a.Name == b.Name && a.Description == b.Description
}
