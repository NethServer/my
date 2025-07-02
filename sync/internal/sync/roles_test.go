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
)

func TestIsSystemUserRole(t *testing.T) {
	tests := []struct {
		name     string
		role     client.LogtoRole
		expected bool
	}{
		{
			name: "logto system role by name",
			role: client.LogtoRole{
				ID:          "role_123",
				Name:        "logto-admin",
				Description: "Logto administrator role",
			},
			expected: true,
		},
		{
			name: "admin role by name",
			role: client.LogtoRole{
				ID:          "role_456",
				Name:        "System Admin",
				Description: "System administrator",
			},
			expected: true,
		},
		{
			name: "machine-to-machine role by name",
			role: client.LogtoRole{
				ID:          "role_789",
				Name:        "Machine-to-Machine",
				Description: "M2M access role",
			},
			expected: true,
		},
		{
			name: "system role by name",
			role: client.LogtoRole{
				ID:          "role_101",
				Name:        "System Service",
				Description: "Internal system service role",
			},
			expected: true,
		},
		{
			name: "default role by name",
			role: client.LogtoRole{
				ID:          "role_102",
				Name:        "Default User",
				Description: "Default user role",
			},
			expected: true,
		},
		{
			name: "system role by description",
			role: client.LogtoRole{
				ID:          "role_103",
				Name:        "Internal Role",
				Description: "This is a system role for internal use",
			},
			expected: true,
		},
		{
			name: "default role by description",
			role: client.LogtoRole{
				ID:          "role_104",
				Name:        "User Role",
				Description: "Default role for all users",
			},
			expected: true,
		},
		{
			name: "logto role by description",
			role: client.LogtoRole{
				ID:          "role_105",
				Name:        "Service Role",
				Description: "Role managed by Logto service",
			},
			expected: true,
		},
		{
			name: "custom user role",
			role: client.LogtoRole{
				ID:          "role_201",
				Name:        "Customer Support",
				Description: "Customer support representative",
			},
			expected: false,
		},
		{
			name: "custom business role",
			role: client.LogtoRole{
				ID:          "role_202",
				Name:        "Sales Manager",
				Description: "Sales team manager",
			},
			expected: false,
		},
		{
			name: "organization role",
			role: client.LogtoRole{
				ID:          "role_203",
				Name:        "God",
				Description: "Highest level organization role",
			},
			expected: false,
		},
		{
			name: "regular user role",
			role: client.LogtoRole{
				ID:          "role_204",
				Name:        "Reseller",
				Description: "Reseller organization role",
			},
			expected: false,
		},
		{
			name: "case insensitive matching - uppercase",
			role: client.LogtoRole{
				ID:          "role_301",
				Name:        "ADMIN-USER",
				Description: "Administrative user",
			},
			expected: true,
		},
		{
			name: "case insensitive matching - mixed case",
			role: client.LogtoRole{
				ID:          "role_302",
				Name:        "LogTo-Service",
				Description: "LogTo service role",
			},
			expected: true,
		},
		{
			name: "partial match in name",
			role: client.LogtoRole{
				ID:          "role_303",
				Name:        "My-Admin-Role",
				Description: "Custom admin role",
			},
			expected: true,
		},
		{
			name: "partial match in description",
			role: client.LogtoRole{
				ID:          "role_304",
				Name:        "Service",
				Description: "Role for system services",
			},
			expected: true,
		},
		{
			name: "empty role name and description",
			role: client.LogtoRole{
				ID:          "role_401",
				Name:        "",
				Description: "",
			},
			expected: false,
		},
		{
			name: "role with only ID",
			role: client.LogtoRole{
				ID:          "role_402",
				Name:        "",
				Description: "",
			},
			expected: false,
		},
		{
			name: "similar but not system role",
			role: client.LogtoRole{
				ID:          "role_501",
				Name:        "Systematic User",
				Description: "User who follows systematic processes",
			},
			expected: true, // Contains "system" so would be preserved
		},
		{
			name: "administrator role",
			role: client.LogtoRole{
				ID:          "role_502",
				Name:        "Administrator",
				Description: "Full administrator access",
			},
			expected: true, // Contains "admin"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSystemUserRole(tt.role)
			if result != tt.expected {
				t.Errorf("isSystemUserRole(%+v) = %v, expected %v", tt.role, result, tt.expected)
			}
		})
	}
}

func TestIsSystemUserRoleEdgeCases(t *testing.T) {
	t.Run("role with special characters", func(t *testing.T) {
		role := client.LogtoRole{
			ID:          "role_special",
			Name:        "admin@system.local",
			Description: "Admin role with special chars",
		}
		result := isSystemUserRole(role)
		if !result {
			t.Error("expected role with 'admin' in name to be considered system role")
		}
	})

	t.Run("role with numbers", func(t *testing.T) {
		role := client.LogtoRole{
			ID:          "role_numeric",
			Name:        "admin123",
			Description: "Admin role version 123",
		}
		result := isSystemUserRole(role)
		if !result {
			t.Error("expected role with 'admin' in name to be considered system role")
		}
	})

	t.Run("role with hyphen and underscore", func(t *testing.T) {
		role := client.LogtoRole{
			ID:          "role_hyphen",
			Name:        "system-admin_role",
			Description: "System admin role with separators",
		}
		result := isSystemUserRole(role)
		if !result {
			t.Error("expected role with 'system' and 'admin' in name to be considered system role")
		}
	})

	t.Run("very long role name", func(t *testing.T) {
		role := client.LogtoRole{
			ID:          "role_long",
			Name:        "this_is_a_very_long_role_name_that_contains_the_word_system_somewhere_in_the_middle",
			Description: "Long role name test",
		}
		result := isSystemUserRole(role)
		if !result {
			t.Error("expected role with 'system' in long name to be considered system role")
		}
	})

	t.Run("role with system in unusual context", func(t *testing.T) {
		role := client.LogtoRole{
			ID:          "role_context",
			Name:        "ecosystem-manager",
			Description: "Manages the ecosystem",
		}
		result := isSystemUserRole(role)
		if !result {
			t.Error("expected role with 'system' substring to be considered system role")
		}
	})
}

func TestSystemRolePatterns(t *testing.T) {
	systemPatterns := []string{
		"logto",
		"admin",
		"machine-to-machine",
		"system",
		"default",
	}

	for _, pattern := range systemPatterns {
		t.Run("pattern_"+pattern, func(t *testing.T) {
			// Test pattern in name
			role := client.LogtoRole{
				ID:          "test_role",
				Name:        "test-" + pattern + "-role",
				Description: "Test role",
			}
			if !isSystemUserRole(role) {
				t.Errorf("expected role with '%s' in name to be system role", pattern)
			}

			// Test pattern in description - only test patterns that work in description
			if pattern == "system" || pattern == "default" || pattern == "logto" {
				role2 := client.LogtoRole{
					ID:          "test_role2",
					Name:        "custom-role",
					Description: "This is a " + pattern + " managed role",
				}
				if !isSystemUserRole(role2) {
					t.Errorf("expected role with '%s' in description to be system role", pattern)
				}
			}
		})
	}
}

func TestNonSystemRoleExamples(t *testing.T) {
	nonSystemRoles := []client.LogtoRole{
		{
			ID:          "role_god",
			Name:        "God",
			Description: "Highest organization role",
		},
		{
			ID:          "role_distributor",
			Name:        "Distributor",
			Description: "Distribution organization role",
		},
		{
			ID:          "role_reseller",
			Name:        "Reseller",
			Description: "Reseller organization role",
		},
		{
			ID:          "role_customer",
			Name:        "Customer",
			Description: "Customer organization role",
		},
		{
			ID:          "role_support",
			Name:        "Support",
			Description: "Customer support role",
		},
		{
			ID:          "role_viewer",
			Name:        "Viewer",
			Description: "Read-only access role",
		},
		{
			ID:          "role_editor",
			Name:        "Editor",
			Description: "Content editor role",
		},
	}

	for _, role := range nonSystemRoles {
		t.Run("non_system_"+role.Name, func(t *testing.T) {
			if isSystemUserRole(role) {
				t.Errorf("expected role %s to NOT be considered system role", role.Name)
			}
		})
	}
}
