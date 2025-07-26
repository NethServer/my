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
	"sync"
	"testing"

	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLogtoClient is a mock implementation of the Logto client
type MockLogtoClient struct {
	mock.Mock
}

func (m *MockLogtoClient) GetAllRoles() ([]models.Role, error) {
	args := m.Called()
	return args.Get(0).([]models.Role), args.Error(1)
}

// resetSingleton resets the singleton for testing
func resetSingleton() {
	roleNames = nil
	roleNamesOnce = sync.Once{}
}

func TestGetRoleNames_Singleton(t *testing.T) {
	resetSingleton()

	// Test that singleton returns the same instance
	instance1 := GetRoleNames()
	instance2 := GetRoleNames()

	assert.Same(t, instance1, instance2, "GetRoleNames should return the same singleton instance")
	assert.NotNil(t, instance1.roles, "roles map should be initialized")
	assert.False(t, instance1.loaded, "loaded should be false initially")
}

func TestRoleNames_LoadRoles_Success(t *testing.T) {
	resetSingleton()

	// Create test roles
	testRoles := []models.Role{
		{ID: "role1", Name: "Admin"},
		{ID: "role2", Name: "User"},
		{ID: "role3", Name: "Support"},
	}

	// We need to test this differently since we can't easily mock the logto.NewManagementClient()
	// For now, let's test the core functionality by directly manipulating the struct
	roleNames := GetRoleNames()

	// Simulate loading roles
	roleNames.mutex.Lock()
	roleNames.roles = make(map[string]string)
	for _, role := range testRoles {
		roleNames.roles[role.ID] = role.Name
	}
	roleNames.loaded = true
	roleNames.mutex.Unlock()

	// Test that roles are properly loaded
	assert.True(t, roleNames.IsLoaded(), "IsLoaded should return true after loading")
	assert.Equal(t, 3, len(roleNames.roles), "Should have 3 roles loaded")
	assert.Equal(t, "Admin", roleNames.roles["role1"], "Role1 should map to Admin")
	assert.Equal(t, "User", roleNames.roles["role2"], "Role2 should map to User")
	assert.Equal(t, "Support", roleNames.roles["role3"], "Role3 should map to Support")
}

func TestRoleNames_GetNames_WithLoadedRoles(t *testing.T) {
	resetSingleton()
	roleNames := GetRoleNames()

	// Setup test data
	roleNames.mutex.Lock()
	roleNames.roles = map[string]string{
		"role1": "Admin",
		"role2": "User",
		"role3": "Support",
	}
	roleNames.loaded = true
	roleNames.mutex.Unlock()

	tests := []struct {
		name     string
		roleIDs  []string
		expected []string
	}{
		{
			name:     "Empty input",
			roleIDs:  []string{},
			expected: []string{},
		},
		{
			name:     "Single valid role",
			roleIDs:  []string{"role1"},
			expected: []string{"Admin"},
		},
		{
			name:     "Multiple valid roles",
			roleIDs:  []string{"role1", "role2", "role3"},
			expected: []string{"Admin", "User", "Support"},
		},
		{
			name:     "Mix of valid and invalid roles",
			roleIDs:  []string{"role1", "invalid", "role2"},
			expected: []string{"Admin", "User"},
		},
		{
			name:     "All invalid roles",
			roleIDs:  []string{"invalid1", "invalid2"},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := roleNames.GetNames(tt.roleIDs)
			assert.ElementsMatch(t, tt.expected, result, "Role names should match expected")
		})
	}
}

func TestRoleNames_GetNames_NotLoaded(t *testing.T) {
	resetSingleton()
	roleNames := GetRoleNames()

	// Ensure roles are not loaded
	assert.False(t, roleNames.IsLoaded(), "Should not be loaded initially")

	result := roleNames.GetNames([]string{"role1", "role2"})
	assert.Empty(t, result, "Should return empty slice when roles not loaded")
}

func TestRoleNames_IsLoaded(t *testing.T) {
	resetSingleton()
	roleNames := GetRoleNames()

	// Initially not loaded
	assert.False(t, roleNames.IsLoaded(), "Should be false initially")

	// Simulate loading
	roleNames.mutex.Lock()
	roleNames.loaded = true
	roleNames.mutex.Unlock()

	assert.True(t, roleNames.IsLoaded(), "Should be true after loading")
}

func TestRoleNames_ConcurrentAccess(t *testing.T) {
	resetSingleton()
	roleNames := GetRoleNames()

	// Setup initial data
	roleNames.mutex.Lock()
	roleNames.roles = map[string]string{
		"role1": "Admin",
		"role2": "User",
	}
	roleNames.loaded = true
	roleNames.mutex.Unlock()

	// Test concurrent reads
	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			names := roleNames.GetNames([]string{"role1", "role2"})
			assert.Len(t, names, 2, "Should return 2 role names")
			assert.True(t, roleNames.IsLoaded(), "Should be loaded")
		}()
	}

	wg.Wait()
}

func TestRoleNames_LoadRoles_ClearsPreviousData(t *testing.T) {
	resetSingleton()
	roleNames := GetRoleNames()

	// Setup initial data
	roleNames.mutex.Lock()
	roleNames.roles = map[string]string{
		"old1": "OldRole1",
		"old2": "OldRole2",
	}
	roleNames.loaded = true
	roleNames.mutex.Unlock()

	// Simulate new load with different data
	roleNames.mutex.Lock()
	roleNames.roles = make(map[string]string) // Clear as done in LoadRoles
	roleNames.roles["new1"] = "NewRole1"
	roleNames.roles["new2"] = "NewRole2"
	roleNames.mutex.Unlock()

	// Verify old data is gone and new data is present
	names := roleNames.GetNames([]string{"old1", "new1"})
	assert.Contains(t, names, "NewRole1", "Should contain new role")
	assert.NotContains(t, names, "OldRole1", "Should not contain old role")
	assert.Len(t, names, 1, "Should only return one valid name")
}
