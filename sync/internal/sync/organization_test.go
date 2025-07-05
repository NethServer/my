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

func TestIsSystemRoleFunctions(t *testing.T) {
	// Test system role detection functions that were uncovered
	logger.SetLevel("fatal")

	t.Run("isSystemOrganizationRole", func(t *testing.T) {
		systemRole := client.LogtoOrganizationRole{
			Name:        "logto-admin",
			Description: "System admin role",
		}

		if !isSystemOrganizationRole(systemRole) {
			t.Error("expected logto-admin to be identified as system role")
		}

		userRole := client.LogtoOrganizationRole{
			Name:        "custom-role",
			Description: "User defined role",
		}

		if isSystemOrganizationRole(userRole) {
			t.Error("expected custom-role to not be identified as system role")
		}
	})

	t.Run("isSystemOrganizationScope", func(t *testing.T) {
		systemScope := client.LogtoOrganizationScope{
			Name:        "logto:management",
			Description: "Logto management scope",
		}

		if !isSystemOrganizationScope(systemScope) {
			t.Error("expected logto:management to be identified as system scope")
		}

		userScope := client.LogtoOrganizationScope{
			Name:        "custom:action",
			Description: "Organization scope: custom action",
		}

		if isSystemOrganizationScope(userScope) {
			t.Error("expected custom:action to not be identified as system scope")
		}
	})
}

func TestSystemOrganizationRolePatterns(t *testing.T) {
	// Test various system organization role patterns
	tests := []struct {
		name     string
		role     client.LogtoOrganizationRole
		expected bool
	}{
		{
			name:     "logto role",
			role:     client.LogtoOrganizationRole{Name: "logto-service", Description: "Service role"},
			expected: true,
		},
		{
			name:     "admin role",
			role:     client.LogtoOrganizationRole{Name: "admin", Description: "Admin role"},
			expected: true,
		},
		{
			name:     "system role",
			role:     client.LogtoOrganizationRole{Name: "system-role", Description: "System role"},
			expected: true,
		},
		{
			name:     "default role",
			role:     client.LogtoOrganizationRole{Name: "default", Description: "Default role"},
			expected: true,
		},
		{
			name:     "owner role",
			role:     client.LogtoOrganizationRole{Name: "owner", Description: "Owner role"},
			expected: true,
		},
		{
			name:     "member role",
			role:     client.LogtoOrganizationRole{Name: "member", Description: "Member role"},
			expected: true,
		},
		{
			name:     "custom role",
			role:     client.LogtoOrganizationRole{Name: "custom-role", Description: "Custom role"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSystemOrganizationRole(tt.role)
			if result != tt.expected {
				t.Errorf("isSystemOrganizationRole(%+v) = %v, expected %v", tt.role, result, tt.expected)
			}
		})
	}
}

func TestSystemOrganizationScopePatterns(t *testing.T) {
	// Test various system organization scope patterns
	tests := []struct {
		name     string
		scope    client.LogtoOrganizationScope
		expected bool
	}{
		{
			name:     "logto scope",
			scope:    client.LogtoOrganizationScope{Name: "logto:admin", Description: "Logto admin scope"},
			expected: true,
		},
		{
			name:     "system scope",
			scope:    client.LogtoOrganizationScope{Name: "system:read", Description: "System read scope"},
			expected: true,
		},
		{
			name:     "default scope",
			scope:    client.LogtoOrganizationScope{Name: "default:access", Description: "Default access scope"},
			expected: true,
		},
		{
			name:     "management scope",
			scope:    client.LogtoOrganizationScope{Name: "management:api", Description: "Management API scope"},
			expected: true,
		},
		{
			name:     "api scope",
			scope:    client.LogtoOrganizationScope{Name: "api:access", Description: "API access scope"},
			expected: true,
		},
		{
			name:     "logto description",
			scope:    client.LogtoOrganizationScope{Name: "custom", Description: "Logto internal scope"},
			expected: true,
		},
		{
			name:     "management description",
			scope:    client.LogtoOrganizationScope{Name: "custom", Description: "Management scope"},
			expected: true,
		},
		{
			name:     "system description but org scope",
			scope:    client.LogtoOrganizationScope{Name: "custom", Description: "Organization scope: system access"},
			expected: false,
		},
		{
			name:     "custom scope",
			scope:    client.LogtoOrganizationScope{Name: "read:users", Description: "Organization scope: Read users"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSystemOrganizationScope(tt.scope)
			if result != tt.expected {
				t.Errorf("isSystemOrganizationScope(%+v) = %v, expected %v", tt.scope, result, tt.expected)
			}
		})
	}
}
