/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/models"
)

func TestHasOrgRolePermission(t *testing.T) {
	tests := []struct {
		name            string
		userOrgRole     string
		requiredOrgRole string
		expectedResult  bool
		description     string
	}{
		// Owner permissions
		{
			name:            "owner can access owner resources",
			userOrgRole:     "Owner",
			requiredOrgRole: "owner",
			expectedResult:  true,
			description:     "Owner should be able to access owner-restricted resources",
		},
		{
			name:            "owner can access distributor resources",
			userOrgRole:     "Owner",
			requiredOrgRole: "distributor",
			expectedResult:  true,
			description:     "Owner should be able to access distributor-restricted resources",
		},
		{
			name:            "owner can access reseller resources",
			userOrgRole:     "Owner",
			requiredOrgRole: "reseller",
			expectedResult:  true,
			description:     "Owner should be able to access reseller-restricted resources",
		},
		{
			name:            "owner can access customer resources",
			userOrgRole:     "Owner",
			requiredOrgRole: "customer",
			expectedResult:  true,
			description:     "Owner should be able to access customer-restricted resources",
		},

		// Distributor permissions
		{
			name:            "distributor cannot access owner resources",
			userOrgRole:     "Distributor",
			requiredOrgRole: "owner",
			expectedResult:  false,
			description:     "Distributor should not be able to access owner-restricted resources",
		},
		{
			name:            "distributor can access distributor resources",
			userOrgRole:     "Distributor",
			requiredOrgRole: "distributor",
			expectedResult:  true,
			description:     "Distributor should be able to access distributor-restricted resources",
		},
		{
			name:            "distributor can access reseller resources",
			userOrgRole:     "Distributor",
			requiredOrgRole: "reseller",
			expectedResult:  true,
			description:     "Distributor should be able to access reseller-restricted resources",
		},
		{
			name:            "distributor can access customer resources",
			userOrgRole:     "Distributor",
			requiredOrgRole: "customer",
			expectedResult:  true,
			description:     "Distributor should be able to access customer-restricted resources",
		},

		// Reseller permissions
		{
			name:            "reseller cannot access owner resources",
			userOrgRole:     "Reseller",
			requiredOrgRole: "owner",
			expectedResult:  false,
			description:     "Reseller should not be able to access owner-restricted resources",
		},
		{
			name:            "reseller cannot access distributor resources",
			userOrgRole:     "Reseller",
			requiredOrgRole: "distributor",
			expectedResult:  false,
			description:     "Reseller should not be able to access distributor-restricted resources",
		},
		{
			name:            "reseller can access reseller resources",
			userOrgRole:     "Reseller",
			requiredOrgRole: "reseller",
			expectedResult:  true,
			description:     "Reseller should be able to access reseller-restricted resources",
		},
		{
			name:            "reseller can access customer resources",
			userOrgRole:     "Reseller",
			requiredOrgRole: "customer",
			expectedResult:  true,
			description:     "Reseller should be able to access customer-restricted resources",
		},

		// Customer permissions
		{
			name:            "customer cannot access owner resources",
			userOrgRole:     "Customer",
			requiredOrgRole: "owner",
			expectedResult:  false,
			description:     "Customer should not be able to access owner-restricted resources",
		},
		{
			name:            "customer cannot access distributor resources",
			userOrgRole:     "Customer",
			requiredOrgRole: "distributor",
			expectedResult:  false,
			description:     "Customer should not be able to access distributor-restricted resources",
		},
		{
			name:            "customer cannot access reseller resources",
			userOrgRole:     "Customer",
			requiredOrgRole: "reseller",
			expectedResult:  false,
			description:     "Customer should not be able to access reseller-restricted resources",
		},
		{
			name:            "customer can access customer resources",
			userOrgRole:     "Customer",
			requiredOrgRole: "customer",
			expectedResult:  true,
			description:     "Customer should be able to access customer-restricted resources",
		},

		// Case sensitivity tests
		{
			name:            "case insensitive - owner uppercase can access owner lowercase",
			userOrgRole:     "OWNER",
			requiredOrgRole: "owner",
			expectedResult:  true,
			description:     "Case should not matter for permission checks",
		},
		{
			name:            "case insensitive - distributor mixed case",
			userOrgRole:     "DiStRiBuToR",
			requiredOrgRole: "DISTRIBUTOR",
			expectedResult:  true,
			description:     "Case should not matter for permission checks",
		},
		{
			name:            "case insensitive - customer access denied",
			userOrgRole:     "CUSTOMER",
			requiredOrgRole: "owner",
			expectedResult:  false,
			description:     "Case insensitive permission denial should work",
		},

		// Invalid role tests
		{
			name:            "invalid user role denied",
			userOrgRole:     "InvalidRole",
			requiredOrgRole: "owner",
			expectedResult:  false,
			description:     "Invalid user roles should be denied access",
		},
		{
			name:            "invalid required role denied",
			userOrgRole:     "Owner",
			requiredOrgRole: "InvalidRequiredRole",
			expectedResult:  false,
			description:     "Invalid required roles should deny access",
		},
		{
			name:            "both invalid roles denied",
			userOrgRole:     "InvalidUser",
			requiredOrgRole: "InvalidRequired",
			expectedResult:  false,
			description:     "Both invalid roles should deny access",
		},

		// Edge cases
		{
			name:            "empty user role denied",
			userOrgRole:     "",
			requiredOrgRole: "owner",
			expectedResult:  false,
			description:     "Empty user role should deny access",
		},
		{
			name:            "empty required role denied",
			userOrgRole:     "Owner",
			requiredOrgRole: "",
			expectedResult:  false,
			description:     "Empty required role should deny access",
		},
		{
			name:            "both empty roles denied",
			userOrgRole:     "",
			requiredOrgRole: "",
			expectedResult:  false,
			description:     "Both empty roles should deny access",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasOrgRolePermission(tt.userOrgRole, tt.requiredOrgRole)
			assert.Equal(t, tt.expectedResult, result, tt.description)
		})
	}
}

func TestCanUserAccessRoleCached(t *testing.T) {

	// Test users with different org roles
	ownerUser := &models.User{
		ID:       "owner-123",
		Username: "owner",
		OrgRole:  "Owner",
	}

	distributorUser := &models.User{
		ID:       "dist-456",
		Username: "distributor",
		OrgRole:  "Distributor",
	}

	customerUser := &models.User{
		ID:       "customer-789",
		Username: "customer",
		OrgRole:  "Customer",
	}

	tests := []struct {
		name              string
		roleID            string
		user              *models.User
		mockAccessControl *cache.RoleAccessControl
		mockExists        bool
		expectedResult    bool
		description       string
	}{
		{
			name:              "role not in cache denied (fail closed)",
			roleID:            "missing-role-123",
			user:              ownerUser,
			mockAccessControl: nil,
			mockExists:        false,
			expectedResult:    false,
			description:       "Roles not found in cache should be denied access for security",
		},
		{
			name:   "role with no access control allows everyone",
			roleID: "public-role-456",
			user:   customerUser,
			mockAccessControl: &cache.RoleAccessControl{
				HasAccessControl: false,
				RequiredOrgRole:  "",
			},
			mockExists:     true,
			expectedResult: true,
			description:    "Roles without access control should be accessible to everyone",
		},
		{
			name:   "owner can access owner-restricted role",
			roleID: "super-admin-role-789",
			user:   ownerUser,
			mockAccessControl: &cache.RoleAccessControl{
				HasAccessControl: true,
				RequiredOrgRole:  "owner",
			},
			mockExists:     true,
			expectedResult: true,
			description:    "Owner should be able to access owner-restricted roles",
		},
		{
			name:   "distributor cannot access owner-restricted role",
			roleID: "super-admin-role-789",
			user:   distributorUser,
			mockAccessControl: &cache.RoleAccessControl{
				HasAccessControl: true,
				RequiredOrgRole:  "owner",
			},
			mockExists:     true,
			expectedResult: false,
			description:    "Distributor should not be able to access owner-restricted roles",
		},
		{
			name:   "customer cannot access owner-restricted role",
			roleID: "super-admin-role-789",
			user:   customerUser,
			mockAccessControl: &cache.RoleAccessControl{
				HasAccessControl: true,
				RequiredOrgRole:  "owner",
			},
			mockExists:     true,
			expectedResult: false,
			description:    "Customer should not be able to access owner-restricted roles",
		},
		{
			name:   "distributor can access distributor-restricted role",
			roleID: "admin-role-101",
			user:   distributorUser,
			mockAccessControl: &cache.RoleAccessControl{
				HasAccessControl: true,
				RequiredOrgRole:  "distributor",
			},
			mockExists:     true,
			expectedResult: true,
			description:    "Distributor should be able to access distributor-restricted roles",
		},
		{
			name:   "owner can access distributor-restricted role",
			roleID: "admin-role-101",
			user:   ownerUser,
			mockAccessControl: &cache.RoleAccessControl{
				HasAccessControl: true,
				RequiredOrgRole:  "distributor",
			},
			mockExists:     true,
			expectedResult: true,
			description:    "Owner should be able to access distributor-restricted roles (hierarchy)",
		},
		{
			name:   "customer cannot access distributor-restricted role",
			roleID: "admin-role-101",
			user:   customerUser,
			mockAccessControl: &cache.RoleAccessControl{
				HasAccessControl: true,
				RequiredOrgRole:  "distributor",
			},
			mockExists:     true,
			expectedResult: false,
			description:    "Customer should not be able to access distributor-restricted roles",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock role cache that returns our test data
			mockRoleCache := &mockRoleCache{
				accessControl: tt.mockAccessControl,
				exists:        tt.mockExists,
			}

			result := canUserAccessRoleCachedWithMock(mockRoleCache, tt.roleID, tt.user)
			assert.Equal(t, tt.expectedResult, result, tt.description)
		})
	}
}

func TestIsSystemRole(t *testing.T) {
	tests := []struct {
		name        string
		roleName    string
		description string
		expected    bool
		reason      string
	}{
		{
			name:        "logto role detected",
			roleName:    "Logto Management API access",
			description: "This default role grants access to the Logto management API.",
			expected:    true,
			reason:      "Should detect Logto system role by name",
		},
		{
			name:        "logto role detected by description",
			roleName:    "API Access",
			description: "This role provides Logto management access",
			expected:    true,
			reason:      "Should detect Logto system role by description",
		},
		{
			name:        "machine-to-machine role detected",
			roleName:    "M2M Service Role",
			description: "Machine-to-Machine service role",
			expected:    true,
			reason:      "Should detect M2M system role",
		},
		{
			name:        "default role detected",
			roleName:    "Default User Role",
			description: "Default role for users",
			expected:    true,
			reason:      "Should detect default system role",
		},
		{
			name:        "normal business role not detected",
			roleName:    "Admin",
			description: "Administrator role for the application",
			expected:    false,
			reason:      "Should not detect normal business roles as system roles",
		},
		{
			name:        "support role not detected",
			roleName:    "Support",
			description: "Support team role",
			expected:    false,
			reason:      "Should not detect business roles as system roles",
		},
		{
			name:        "super admin not detected as system role",
			roleName:    "Super Admin",
			description: "Super administrator with highest privileges",
			expected:    false,
			reason:      "Should not detect business admin roles as system roles",
		},
		{
			name:        "case insensitive detection",
			roleName:    "LOGTO ROLE",
			description: "LOGTO ACCESS DESCRIPTION",
			expected:    true,
			reason:      "Should detect system roles case-insensitively",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSystemRole(tt.roleName, tt.description)
			assert.Equal(t, tt.expected, result, tt.reason)
		})
	}
}

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name            string
		userPermissions []string
		orgPermissions  []string
		permission      string
		expected        bool
		description     string
	}{
		{
			name:            "permission found in user permissions",
			userPermissions: []string{"read:systems", "manage:users", "impersonate:users"},
			orgPermissions:  []string{"read:customers"},
			permission:      "impersonate:users",
			expected:        true,
			description:     "Should find permission in user permissions array",
		},
		{
			name:            "permission found in org permissions",
			userPermissions: []string{"read:systems"},
			orgPermissions:  []string{"read:customers", "manage:resellers", "create:distributors"},
			permission:      "manage:resellers",
			expected:        true,
			description:     "Should find permission in organization permissions array",
		},
		{
			name:            "permission not found anywhere",
			userPermissions: []string{"read:systems"},
			orgPermissions:  []string{"read:customers"},
			permission:      "destroy:systems",
			expected:        false,
			description:     "Should return false when permission is not found",
		},
		{
			name:            "empty permissions arrays",
			userPermissions: []string{},
			orgPermissions:  []string{},
			permission:      "read:systems",
			expected:        false,
			description:     "Should return false with empty permission arrays",
		},
		{
			name:            "nil user permissions",
			userPermissions: nil,
			orgPermissions:  []string{"read:customers"},
			permission:      "read:customers",
			expected:        true,
			description:     "Should work with nil user permissions",
		},
		{
			name:            "nil org permissions",
			userPermissions: []string{"read:systems"},
			orgPermissions:  nil,
			permission:      "read:systems",
			expected:        true,
			description:     "Should work with nil org permissions",
		},
		{
			name:            "both nil permissions",
			userPermissions: nil,
			orgPermissions:  nil,
			permission:      "read:systems",
			expected:        false,
			description:     "Should return false with both nil permission arrays",
		},
		{
			name:            "exact string matching",
			userPermissions: []string{"read:system", "read:systems"},
			orgPermissions:  []string{},
			permission:      "read:systems",
			expected:        true,
			description:     "Should do exact string matching (not partial)",
		},
		{
			name:            "case sensitive matching",
			userPermissions: []string{"READ:SYSTEMS"},
			orgPermissions:  []string{},
			permission:      "read:systems",
			expected:        false,
			description:     "Should be case sensitive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPermission(tt.userPermissions, tt.orgPermissions, tt.permission)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// Mock role cache for testing
type mockRoleCache struct {
	accessControl *cache.RoleAccessControl
	exists        bool
}

func (m *mockRoleCache) GetAccessControl(roleID string) (cache.RoleAccessControl, bool) {
	if !m.exists {
		return cache.RoleAccessControl{}, false
	}
	return *m.accessControl, true
}

// canUserAccessRoleCachedWithMock is a test version of canUserAccessRoleCached that accepts a mock
func canUserAccessRoleCachedWithMock(roleCache mockRoleCacheInterface, roleID string, user *models.User) bool {
	// Get access control information from cache
	accessControl, exists := roleCache.GetAccessControl(roleID)
	if !exists {
		// If role not found in cache, deny access for security (fail closed)
		return false
	}

	// If role has no access control restrictions, everyone can access it
	if !accessControl.HasAccessControl {
		return true
	}

	// Check if user's organization role has sufficient privileges
	return HasOrgRolePermission(user.OrgRole, accessControl.RequiredOrgRole)
}

// Interface for mocking role cache
type mockRoleCacheInterface interface {
	GetAccessControl(roleID string) (cache.RoleAccessControl, bool)
}
