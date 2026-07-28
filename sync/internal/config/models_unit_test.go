/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package config

import (
	"testing"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config passes validation",
			config: &Config{
				Metadata: Metadata{
					Name:        "Test Config",
					Version:     "1.0.0",
					Description: "Test description",
				},
				OrganizationRoles: []Role{
					{
						ID:   "owner",
						Name: "Owner",
						Permissions: []Permission{
							{ID: "manage:systems"},
						},
					},
				},
				UserRoles: []Role{
					{
						ID:   "admin",
						Name: "Admin",
						Permissions: []Permission{
							{ID: "read:systems"},
						},
					},
				},
				Resources: []Resource{
					{
						Name:    "systems",
						Actions: []string{"read", "manage"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing metadata name fails validation",
			config: &Config{
				Metadata: Metadata{
					Version: "1.0.0",
				},
			},
			expectError: true,
			errorMsg:    "metadata.name is required",
		},
		{
			name: "missing metadata version fails validation",
			config: &Config{
				Metadata: Metadata{
					Name: "Test Config",
				},
			},
			expectError: true,
			errorMsg:    "metadata.version is required",
		},
		{
			name: "duplicate organization role ID fails validation",
			config: &Config{
				Metadata: Metadata{
					Name:    "Test Config",
					Version: "1.0.0",
				},
				OrganizationRoles: []Role{
					{
						ID:   "owner",
						Name: "Owner",
						Permissions: []Permission{
							{ID: "manage:systems"},
						},
					},
					{
						ID:   "owner",
						Name: "Owner Duplicate",
						Permissions: []Permission{
							{ID: "read:systems"},
						},
					},
				},
				Resources: []Resource{
					{
						Name:    "systems",
						Actions: []string{"read", "manage"},
					},
				},
			},
			expectError: true,
			errorMsg:    "duplicate organization role ID",
		},
		{
			name: "duplicate user role ID fails validation",
			config: &Config{
				Metadata: Metadata{
					Name:    "Test Config",
					Version: "1.0.0",
				},
				UserRoles: []Role{
					{
						ID:   "admin",
						Name: "Admin",
						Permissions: []Permission{
							{ID: "read:systems"},
						},
					},
					{
						ID:   "admin",
						Name: "Admin Duplicate",
						Permissions: []Permission{
							{ID: "read:systems"},
						},
					},
				},
				Resources: []Resource{
					{
						Name:    "systems",
						Actions: []string{"read", "manage"},
					},
				},
			},
			expectError: true,
			errorMsg:    "duplicate user role ID",
		},
		{
			name: "duplicate resource name fails validation",
			config: &Config{
				Metadata: Metadata{
					Name:    "Test Config",
					Version: "1.0.0",
				},
				Resources: []Resource{
					{
						Name:    "systems",
						Actions: []string{"read"},
					},
					{
						Name:    "systems",
						Actions: []string{"write"},
					},
				},
			},
			expectError: true,
			errorMsg:    "duplicate resource name",
		},
		{
			name: "invalid permission reference fails validation",
			config: &Config{
				Metadata: Metadata{
					Name:    "Test Config",
					Version: "1.0.0",
				},
				OrganizationRoles: []Role{
					{
						ID:   "owner",
						Name: "Owner",
						Permissions: []Permission{
							{ID: "invalid:permission"},
						},
					},
				},
				Resources: []Resource{
					{
						Name:    "systems",
						Actions: []string{"read"},
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid permission reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !containsSubstring(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateRole(t *testing.T) {
	config := &Config{}

	tests := []struct {
		name        string
		role        Role
		roleType    string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid role passes validation",
			role: Role{
				ID:   "admin",
				Name: "Admin",
				Type: "user",
				Permissions: []Permission{
					{ID: "read:systems"},
				},
			},
			roleType:    "user",
			expectError: false,
		},
		{
			name: "missing role ID fails validation",
			role: Role{
				Name: "Admin",
			},
			roleType:    "user",
			expectError: true,
			errorMsg:    "role ID is required",
		},
		{
			name: "missing role name fails validation",
			role: Role{
				ID: "admin",
			},
			roleType:    "user",
			expectError: true,
			errorMsg:    "role name is required",
		},
		{
			name: "invalid role type fails validation",
			role: Role{
				ID:   "admin",
				Name: "Admin",
				Type: "invalid",
			},
			roleType:    "user",
			expectError: true,
			errorMsg:    "invalid role type",
		},
		{
			name: "negative priority fails validation",
			role: Role{
				ID:       "admin",
				Name:     "Admin",
				Priority: -1,
			},
			roleType:    "user",
			expectError: true,
			errorMsg:    "role priority must be non-negative",
		},
		{
			name: "duplicate permission ID fails validation",
			role: Role{
				ID:   "admin",
				Name: "Admin",
				Permissions: []Permission{
					{ID: "read:systems"},
					{ID: "read:systems"},
				},
			},
			roleType:    "user",
			expectError: true,
			errorMsg:    "duplicate permission ID",
		},
		{
			name: "missing permission ID fails validation",
			role: Role{
				ID:   "admin",
				Name: "Admin",
				Permissions: []Permission{
					{ID: ""},
				},
			},
			roleType:    "user",
			expectError: true,
			errorMsg:    "permission ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.validateRole(tt.role, tt.roleType)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !containsSubstring(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateResource(t *testing.T) {
	config := &Config{}

	tests := []struct {
		name        string
		resource    Resource
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid resource passes validation",
			resource: Resource{
				Name:    "systems",
				Actions: []string{"read", "write"},
			},
			expectError: false,
		},
		{
			name: "missing resource name fails validation",
			resource: Resource{
				Actions: []string{"read"},
			},
			expectError: true,
			errorMsg:    "resource name is required",
		},
		{
			name: "empty actions fails validation",
			resource: Resource{
				Name:    "systems",
				Actions: []string{},
			},
			expectError: true,
			errorMsg:    "must have at least one action",
		},
		{
			name: "duplicate action fails validation",
			resource: Resource{
				Name:    "systems",
				Actions: []string{"read", "read"},
			},
			expectError: true,
			errorMsg:    "duplicate action",
		},
		{
			name: "empty action fails validation",
			resource: Resource{
				Name:    "systems",
				Actions: []string{"read", ""},
			},
			expectError: true,
			errorMsg:    "empty action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.validateResource(tt.resource)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !containsSubstring(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateApplication(t *testing.T) {
	config := &Config{
		OrganizationRoles: []Role{
			{ID: "owner", Name: "Owner"},
		},
		UserRoles: []Role{
			{ID: "admin", Name: "Admin"},
		},
	}

	tests := []struct {
		name        string
		app         Application
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid application passes validation",
			app: Application{
				Name:        "test.example.com",
				Description: "Test application",
				DisplayName: "Test App",
				AccessControl: &AccessControl{
					OrganizationRoles: []string{"owner"},
					UserRoles:         []string{"admin"},
				},
			},
			expectError: false,
		},
		{
			name: "missing application name fails validation",
			app: Application{
				Description: "Test application",
				DisplayName: "Test App",
			},
			expectError: true,
			errorMsg:    "application name is required",
		},
		{
			name: "missing description fails validation",
			app: Application{
				Name:        "test.example.com",
				DisplayName: "Test App",
			},
			expectError: true,
			errorMsg:    "application description is required",
		},
		{
			name: "missing display name fails validation",
			app: Application{
				Name:        "test.example.com",
				Description: "Test application",
			},
			expectError: true,
			errorMsg:    "application display_name is required",
		},
		{
			name: "invalid organization role in access control fails validation",
			app: Application{
				Name:        "test.example.com",
				Description: "Test application",
				DisplayName: "Test App",
				AccessControl: &AccessControl{
					OrganizationRoles: []string{"invalid"},
				},
			},
			expectError: true,
			errorMsg:    "invalid organization role",
		},
		{
			name: "invalid user role in access control fails validation",
			app: Application{
				Name:        "test.example.com",
				Description: "Test application",
				DisplayName: "Test App",
				AccessControl: &AccessControl{
					UserRoles: []string{"invalid"},
				},
			},
			expectError: true,
			errorMsg:    "invalid user role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.validateApplication(tt.app)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !containsSubstring(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestContainerName(t *testing.T) {
	tests := []struct {
		name     string
		resource Resource
		expected string
	}{
		{
			name:     "falls back to the resource name",
			resource: Resource{Name: "systems", Actions: []string{"read"}},
			expected: "systems",
		},
		{
			name:     "honours an explicit container",
			resource: Resource{Name: "systems", Actions: []string{"read"}, Container: "permissions"},
			expected: "permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.resource.ContainerName(); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestGetResourceContainers(t *testing.T) {
	scopeNames := func(container ResourceContainer) []string {
		names := make([]string, 0, len(container.Scopes))
		for _, scope := range container.Scopes {
			names = append(names, scope.Name)
		}
		return names
	}

	t.Run("grouping does not rename scopes", func(t *testing.T) {
		config := &Config{
			Resources: []Resource{
				{Name: "distributors", Actions: []string{"read", "manage"}, Container: "permissions"},
				{Name: "systems", Actions: []string{"read", "connect"}, Container: "permissions"},
				{Name: "role-access-control", Actions: []string{"owner"}, Container: "permissions"},
			},
		}

		containers := config.GetResourceContainers()

		if len(containers) != 1 {
			t.Fatalf("expected 1 container, got %d", len(containers))
		}
		if containers[0].Name != "permissions" {
			t.Errorf("expected container %q, got %q", "permissions", containers[0].Name)
		}

		// The point of the grouping: scope names stay "{action}:{resource}",
		// because roles and the backend match on these exact strings.
		expected := []string{
			"read:distributors", "manage:distributors",
			"read:systems", "connect:systems",
			"owner:role-access-control",
		}
		got := scopeNames(containers[0])

		if len(got) != len(expected) {
			t.Fatalf("expected %d scopes, got %d: %v", len(expected), len(got), got)
		}
		for i, name := range expected {
			if got[i] != name {
				t.Errorf("scope %d: expected %q, got %q", i, name, got[i])
			}
		}
	})

	t.Run("no container keeps one resource per entry", func(t *testing.T) {
		config := &Config{
			Resources: []Resource{
				{Name: "distributors", Actions: []string{"read"}},
				{Name: "systems", Actions: []string{"read", "manage"}},
			},
		}

		containers := config.GetResourceContainers()

		if len(containers) != 2 {
			t.Fatalf("expected 2 containers for backward compatibility, got %d", len(containers))
		}
		if containers[0].Name != "distributors" || containers[1].Name != "systems" {
			t.Errorf("expected config order [distributors systems], got [%s %s]",
				containers[0].Name, containers[1].Name)
		}
		if names := scopeNames(containers[1]); len(names) != 2 || names[0] != "read:systems" {
			t.Errorf("unexpected scopes on systems: %v", names)
		}
	})

	t.Run("mixed grouping keeps first-seen order", func(t *testing.T) {
		config := &Config{
			Resources: []Resource{
				{Name: "distributors", Actions: []string{"read"}, Container: "permissions"},
				{Name: "systems", Actions: []string{"read"}},
				{Name: "users", Actions: []string{"read"}, Container: "permissions"},
			},
		}

		containers := config.GetResourceContainers()

		if len(containers) != 2 {
			t.Fatalf("expected 2 containers, got %d", len(containers))
		}
		if containers[0].Name != "permissions" || containers[1].Name != "systems" {
			t.Fatalf("expected [permissions systems], got [%s %s]", containers[0].Name, containers[1].Name)
		}
		if names := scopeNames(containers[0]); len(names) != 2 ||
			names[0] != "read:distributors" || names[1] != "read:users" {
			t.Errorf("expected permissions to hold both scopes in config order, got %v", names)
		}
	})

	// The production shape: 9 resources, 27 scopes, a single billable container.
	t.Run("production shape collapses to one container", func(t *testing.T) {
		config := &Config{
			Resources: []Resource{
				{Name: "distributors", Actions: []string{"read", "manage", "destroy"}, Container: "permissions"},
				{Name: "resellers", Actions: []string{"read", "manage", "destroy"}, Container: "permissions"},
				{Name: "customers", Actions: []string{"read", "manage", "destroy"}, Container: "permissions"},
				{Name: "systems", Actions: []string{"read", "manage", "destroy", "connect"}, Container: "permissions"},
				{Name: "entitlements", Actions: []string{"read", "manage"}, Container: "permissions"},
				{Name: "users", Actions: []string{"read", "manage", "destroy", "impersonate"}, Container: "permissions"},
				{Name: "applications", Actions: []string{"read", "manage"}, Container: "permissions"},
				{Name: "alerts", Actions: []string{"read", "manage", "config"}, Container: "permissions"},
				{Name: "role-access-control", Actions: []string{"owner", "distributor", "reseller"}, Container: "permissions"},
			},
		}

		containers := config.GetResourceContainers()

		if len(containers) != 1 {
			t.Fatalf("expected the 9 resources to collapse into 1 container, got %d", len(containers))
		}
		if len(containers[0].Scopes) != 27 {
			t.Errorf("expected 27 scopes, got %d", len(containers[0].Scopes))
		}

		// Logto rejects duplicate scope names within a resource.
		present := make(map[string]bool)
		for _, scope := range containers[0].Scopes {
			if present[scope.Name] {
				t.Errorf("duplicate scope name %q would be rejected by Logto", scope.Name)
			}
			present[scope.Name] = true
		}

		// Every permission the backend hardcodes must survive the grouping.
		for _, name := range []string{
			"destroy:distributors", "destroy:resellers", "destroy:customers",
			"destroy:systems", "connect:systems", "impersonate:users",
			"config:alerts", "manage:entitlements",
			"owner:role-access-control", "distributor:role-access-control", "reseller:role-access-control",
		} {
			if !present[name] {
				t.Errorf("permission %q is missing after grouping", name)
			}
		}
	})
}

// Grouping must not change what validation accepts, since role definitions
// reference permissions by scope name.
func TestGroupedConfigStillValidates(t *testing.T) {
	config := &Config{
		Metadata: Metadata{Name: "Test Config", Version: "1.0.0"},
		OrganizationRoles: []Role{
			{ID: "owner", Name: "Owner", Permissions: []Permission{{ID: "read:distributors"}}},
		},
		UserRoles: []Role{
			{ID: "admin", Name: "Admin", Permissions: []Permission{{ID: "connect:systems"}}},
		},
		Resources: []Resource{
			{Name: "distributors", Actions: []string{"read"}, Container: "permissions"},
			{Name: "systems", Actions: []string{"connect"}, Container: "permissions"},
		},
	}

	if err := config.Validate(); err != nil {
		t.Errorf("expected grouped config to validate, got %v", err)
	}
}

// Helper function for substring matching
func containsSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
