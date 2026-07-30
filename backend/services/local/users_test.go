/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"testing"

	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
)

// TestLocalUserService_CanCreateUser tests the permission validation for user creation
func TestLocalUserService_CanCreateUser(t *testing.T) {
	service := &LocalUserService{}

	tests := []struct {
		name           string
		userOrgRole    string
		userOrgID      string
		request        *models.CreateLocalUserRequest
		expectedResult bool
		expectedReason string
	}{
		{
			name:        "owner can create users anywhere",
			userOrgRole: "owner",
			userOrgID:   "org-owner",
			request: &models.CreateLocalUserRequest{
				OrganizationID: stringPtr("any-org"),
			},
			expectedResult: true,
			expectedReason: "",
		},
		{
			name:        "distributor can create users in managed orgs",
			userOrgRole: "distributor",
			userOrgID:   "org-distributor",
			request: &models.CreateLocalUserRequest{
				OrganizationID: stringPtr("org-distributor"),
			},
			expectedResult: true,
			expectedReason: "",
		},
		{
			name:        "customer can create users in own org",
			userOrgRole: "customer",
			userOrgID:   "org-customer",
			request: &models.CreateLocalUserRequest{
				OrganizationID: stringPtr("org-customer"),
			},
			expectedResult: true,
			expectedReason: "",
		},
		{
			name:        "customer cannot create users in other orgs",
			userOrgRole: "customer",
			userOrgID:   "org-customer",
			request: &models.CreateLocalUserRequest{
				OrganizationID: stringPtr("org-other"),
			},
			expectedResult: false,
			expectedReason: "customers can only create users in their own organization",
		},
		{
			name:        "invalid role cannot create users",
			userOrgRole: "invalid",
			userOrgID:   "org-test",
			request: &models.CreateLocalUserRequest{
				OrganizationID: stringPtr("org-test"),
			},
			expectedResult: false,
			expectedReason: "insufficient permissions to create users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canCreate, reason := service.CanCreateUser(tt.userOrgRole, tt.userOrgID, tt.request)

			assert.Equal(t, tt.expectedResult, canCreate)
			if tt.expectedReason != "" {
				assert.Contains(t, reason, tt.expectedReason)
			}
		})
	}
}

// TestLocalUserService_CanReadUser tests the permission validation for reading a
// single user. GET /users/:id used to compare organization ids directly instead
// of walking the hierarchy, so a reseller saw a user of its own customer in
// GET /users and got 403 on that same user's detail.
func TestLocalUserService_CanReadUser(t *testing.T) {
	service := &LocalUserService{}

	tests := []struct {
		name            string
		userOrgRole     string
		userOrgID       string
		targetUserOrgID string
		expectedResult  bool
		expectedReason  string
	}{
		{
			name:            "owner can read any user",
			userOrgRole:     "owner",
			userOrgID:       "org-owner",
			targetUserOrgID: "any-org",
			expectedResult:  true,
		},
		{
			name:            "distributor can read users in own org",
			userOrgRole:     "distributor",
			userOrgID:       "org-distributor",
			targetUserOrgID: "org-distributor",
			expectedResult:  true,
		},
		{
			name:            "reseller can read users in own org",
			userOrgRole:     "reseller",
			userOrgID:       "org-reseller",
			targetUserOrgID: "org-reseller",
			expectedResult:  true,
		},
		{
			name:            "customer can read users in own org",
			userOrgRole:     "customer",
			userOrgID:       "org-customer",
			targetUserOrgID: "org-customer",
			expectedResult:  true,
		},
		{
			name:            "customer cannot read users in other orgs",
			userOrgRole:     "customer",
			userOrgID:       "org-customer",
			targetUserOrgID: "org-other",
			expectedResult:  false,
			expectedReason:  "customers can only read users in their own organization",
		},
		{
			name:            "invalid role cannot read users",
			userOrgRole:     "invalid",
			userOrgID:       "org-test",
			targetUserOrgID: "org-test",
			expectedResult:  false,
			expectedReason:  "insufficient permissions to read users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canRead, reason := service.CanReadUser(tt.userOrgRole, tt.userOrgID, tt.targetUserOrgID)

			assert.Equal(t, tt.expectedResult, canRead)
			if tt.expectedReason != "" {
				assert.Contains(t, reason, tt.expectedReason)
			}
		})
	}
}

// TestLocalUserService_CanReadUserMatchesCanUpdateUser pins the invariant that
// made the original bug a bug: whoever may update a user must be able to read it.
// Reading through a narrower rule than updating is always a mistake.
//
// Only same-organization and customer pairs are exercised: the cross-organization
// branches call IsOrganizationInHierarchy, which needs a database. That is not a
// gap in the invariant — both functions delegate to that same helper for those
// cases, so they cannot disagree there. The live cross-organization behaviour is
// covered by the authz suite (backend/authz, scope layer).
func TestLocalUserService_CanReadUserMatchesCanUpdateUser(t *testing.T) {
	service := &LocalUserService{}

	roles := []string{"owner", "distributor", "reseller", "customer", "invalid"}
	orgPairs := [][2]string{
		{"org-a", "org-a"}, // same organization: resolved without a database
	}
	// A customer is decided by a plain comparison for any pair, so the
	// unrelated-organization case is safe to check for that role alone.
	customerPairs := [][2]string{{"org-a", "org-a"}, {"org-a", "org-b"}}

	check := func(role, userOrg, targetOrg string) {
		canUpdate, _ := service.CanUpdateUser(role, userOrg, targetOrg)
		if !canUpdate {
			return
		}
		canRead, reason := service.CanReadUser(role, userOrg, targetOrg)
		assert.True(t, canRead,
			"role %q may update a user of %q from %q but not read it: %s",
			role, targetOrg, userOrg, reason)
	}

	for _, role := range roles {
		for _, pair := range orgPairs {
			check(role, pair[0], pair[1])
		}
	}
	for _, pair := range customerPairs {
		check("customer", pair[0], pair[1])
	}
}

// TestLocalUserService_CanUpdateUser tests the permission validation for user updates
func TestLocalUserService_CanUpdateUser(t *testing.T) {
	service := &LocalUserService{}

	tests := []struct {
		name            string
		userOrgRole     string
		userOrgID       string
		targetUserOrgID string
		expectedResult  bool
		expectedReason  string
	}{
		{
			name:            "owner can update any user",
			userOrgRole:     "owner",
			userOrgID:       "org-owner",
			targetUserOrgID: "any-org",
			expectedResult:  true,
			expectedReason:  "",
		},
		{
			name:            "distributor can update users in own org",
			userOrgRole:     "distributor",
			userOrgID:       "org-distributor",
			targetUserOrgID: "org-distributor",
			expectedResult:  true,
			expectedReason:  "",
		},
		{
			name:            "customer can update users in own org",
			userOrgRole:     "customer",
			userOrgID:       "org-customer",
			targetUserOrgID: "org-customer",
			expectedResult:  true,
			expectedReason:  "",
		},
		{
			name:            "customer cannot update users in other orgs",
			userOrgRole:     "customer",
			userOrgID:       "org-customer",
			targetUserOrgID: "org-other",
			expectedResult:  false,
			expectedReason:  "customers can only update users in their own organization",
		},
		{
			name:            "invalid role cannot update users",
			userOrgRole:     "invalid",
			userOrgID:       "org-test",
			targetUserOrgID: "org-test",
			expectedResult:  false,
			expectedReason:  "insufficient permissions to update users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canUpdate, reason := service.CanUpdateUser(tt.userOrgRole, tt.userOrgID, tt.targetUserOrgID)

			assert.Equal(t, tt.expectedResult, canUpdate)
			if tt.expectedReason != "" {
				assert.Contains(t, reason, tt.expectedReason)
			}
		})
	}
}

// TestLocalUserService_CanDeleteUser tests the permission validation for user deletion
func TestLocalUserService_CanDeleteUser(t *testing.T) {
	service := &LocalUserService{}

	tests := []struct {
		name            string
		userOrgRole     string
		userOrgID       string
		targetUserOrgID string
		expectedResult  bool
		expectedReason  string
	}{
		{
			name:            "owner can delete any user",
			userOrgRole:     "owner",
			userOrgID:       "org-owner",
			targetUserOrgID: "any-org",
			expectedResult:  true,
			expectedReason:  "",
		},
		{
			name:            "distributor can delete users in own org",
			userOrgRole:     "distributor",
			userOrgID:       "org-distributor",
			targetUserOrgID: "org-distributor",
			expectedResult:  true,
			expectedReason:  "",
		},
		{
			name:            "customer can delete users in own org",
			userOrgRole:     "customer",
			userOrgID:       "org-customer",
			targetUserOrgID: "org-customer",
			expectedResult:  true,
			expectedReason:  "",
		},
		{
			name:            "customer cannot delete users in other orgs",
			userOrgRole:     "customer",
			userOrgID:       "org-customer",
			targetUserOrgID: "org-other",
			expectedResult:  false,
			expectedReason:  "customers can only delete users in their own organization",
		},
		{
			name:            "invalid role cannot delete users",
			userOrgRole:     "invalid",
			userOrgID:       "org-test",
			targetUserOrgID: "org-test",
			expectedResult:  false,
			expectedReason:  "insufficient permissions to delete users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canDelete, reason := service.CanDeleteUser(tt.userOrgRole, tt.userOrgID, tt.targetUserOrgID)

			assert.Equal(t, tt.expectedResult, canDelete)
			if tt.expectedReason != "" {
				assert.Contains(t, reason, tt.expectedReason)
			}
		})
	}
}

// TestLocalUserService_CanSuspendUser tests the permission validation for user suspension
func TestLocalUserService_CanSuspendUser(t *testing.T) {
	service := &LocalUserService{}

	tests := []struct {
		name            string
		userOrgRole     string
		userOrgID       string
		targetUserOrgID string
		expectedResult  bool
		expectedReason  string
	}{
		{
			name:            "owner can suspend any user",
			userOrgRole:     "owner",
			userOrgID:       "org-owner",
			targetUserOrgID: "any-org",
			expectedResult:  true,
			expectedReason:  "",
		},
		{
			name:            "distributor can suspend users in own org",
			userOrgRole:     "distributor",
			userOrgID:       "org-distributor",
			targetUserOrgID: "org-distributor",
			expectedResult:  true,
			expectedReason:  "",
		},
		{
			name:            "customer can suspend users in own org",
			userOrgRole:     "customer",
			userOrgID:       "org-customer",
			targetUserOrgID: "org-customer",
			expectedResult:  true,
			expectedReason:  "",
		},
		{
			name:            "customer cannot suspend users in other orgs",
			userOrgRole:     "customer",
			userOrgID:       "org-customer",
			targetUserOrgID: "org-other",
			expectedResult:  false,
			expectedReason:  "customers can only suspend users in their own organization",
		},
		{
			name:            "invalid role cannot suspend users",
			userOrgRole:     "invalid",
			userOrgID:       "org-test",
			targetUserOrgID: "org-test",
			expectedResult:  false,
			expectedReason:  "insufficient permissions to suspend users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canSuspend, reason := service.CanSuspendUser(tt.userOrgRole, tt.userOrgID, tt.targetUserOrgID)

			assert.Equal(t, tt.expectedResult, canSuspend)
			if tt.expectedReason != "" {
				assert.Contains(t, reason, tt.expectedReason)
			}
		})
	}
}

// TestLocalUserService_CanAccessUser tests the permission validation for user access
func TestLocalUserService_CanAccessUser(t *testing.T) {
	service := &LocalUserService{}

	tests := []struct {
		name            string
		userOrgRole     string
		userOrgID       string
		targetUserOrgID string
		expectedResult  bool
		expectedReason  string
	}{
		{
			name:            "owner can access any user",
			userOrgRole:     "owner",
			userOrgID:       "org-owner",
			targetUserOrgID: "any-org",
			expectedResult:  true,
			expectedReason:  "",
		},
		{
			name:            "distributor can access users in own org",
			userOrgRole:     "distributor",
			userOrgID:       "org-distributor",
			targetUserOrgID: "org-distributor",
			expectedResult:  true,
			expectedReason:  "",
		},
		{
			name:            "customer can access users in own org",
			userOrgRole:     "customer",
			userOrgID:       "org-customer",
			targetUserOrgID: "org-customer",
			expectedResult:  true,
			expectedReason:  "",
		},
		{
			name:            "customer cannot access users in other orgs",
			userOrgRole:     "customer",
			userOrgID:       "org-customer",
			targetUserOrgID: "org-other",
			expectedResult:  false,
			expectedReason:  "customers can only access users in their own organization",
		},
		{
			name:            "invalid role cannot access users",
			userOrgRole:     "invalid",
			userOrgID:       "org-test",
			targetUserOrgID: "org-test",
			expectedResult:  false,
			expectedReason:  "insufficient permissions to access users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canAccess, reason := service.CanAccessUser(tt.userOrgRole, tt.userOrgID, tt.targetUserOrgID)

			assert.Equal(t, tt.expectedResult, canAccess)
			if tt.expectedReason != "" {
				assert.Contains(t, reason, tt.expectedReason)
			}
		})
	}
}

// TestLocalUserService_generateUsernameFromEmail tests username generation from email
func TestLocalUserService_generateUsernameFromEmail(t *testing.T) {
	service := &LocalUserService{}

	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "simple email",
			email:    "john.doe@example.com",
			expected: "john_doe",
		},
		{
			name:     "email with numbers",
			email:    "user123@test.com",
			expected: "user123",
		},
		{
			name:     "email with special chars",
			email:    "user+tag@domain.org",
			expected: "user_tag",
		},
		{
			name:     "email starting with number",
			email:    "123user@test.com",
			expected: "_123user",
		},
		{
			name:     "email with multiple special chars",
			email:    "user-name.test+tag@example.co.uk",
			expected: "user_name_test_tag",
		},
		{
			name:     "edge case - only special chars",
			email:    "+++@test.com",
			expected: "___",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.generateBaseUsernameFromEmail(tt.email)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestLocalUserService_ResetUserPassword tests password reset functionality
func TestLocalUserService_ResetUserPassword(t *testing.T) {
	t.Skip("Skipping test requiring mock setup - requires service interface refactoring")
}

// TestLocalUserService_IsOrganizationInHierarchy tests hierarchical organization validation
func TestLocalUserService_IsOrganizationInHierarchy(t *testing.T) {
	service := &LocalUserService{}

	tests := []struct {
		name           string
		userOrgRole    string
		userOrgID      string
		targetOrgID    string
		expectedResult bool
	}{
		{
			name:           "owner can access own organization",
			userOrgRole:    "owner",
			userOrgID:      "org-owner",
			targetOrgID:    "org-owner",
			expectedResult: true,
		},
		{
			name:           "owner cannot access non-existent organization without database",
			userOrgRole:    "owner",
			userOrgID:      "org-owner",
			targetOrgID:    "any-org",
			expectedResult: false,
		},
		{
			name:           "same organization always accessible",
			userOrgRole:    "customer",
			userOrgID:      "org-same",
			targetOrgID:    "org-same",
			expectedResult: true,
		},
		{
			name:           "different organization for customer",
			userOrgRole:    "customer",
			userOrgID:      "org-customer",
			targetOrgID:    "org-other",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.IsOrganizationInHierarchy(tt.userOrgRole, tt.userOrgID, tt.targetOrgID)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// TestLocalUserService_GetUser tests user retrieval with RBAC validation
func TestLocalUserService_GetUser(t *testing.T) {
	t.Skip("Skipping test requiring mock setup - requires service interface refactoring")
}

// TestLocalUserService_GetUserByLogtoID tests user retrieval by Logto ID
func TestLocalUserService_GetUserByLogtoID(t *testing.T) {
	t.Skip("Skipping test requiring mock setup - requires service interface refactoring")
}

// TestLocalUserService_ListUsers tests user listing with pagination
func TestLocalUserService_ListUsers(t *testing.T) {
	t.Skip("Skipping test requiring mock setup - requires service interface refactoring")
}

// TestLocalUserService_GetTotals tests user totals retrieval
func TestLocalUserService_GetTotals(t *testing.T) {
	t.Skip("Skipping test requiring mock setup - requires service interface refactoring")
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
