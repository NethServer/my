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
				Hierarchy: Hierarchy{
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
				Hierarchy: Hierarchy{
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
				Hierarchy: Hierarchy{
					UserRoles: []Role{
						{
							ID:   "admin",
							Name: "Admin",
							Permissions: []Permission{
								{ID: "manage:systems"},
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
				Hierarchy: Hierarchy{
					Resources: []Resource{
						{
							Name:    "systems",
							Actions: []string{"read"},
						},
						{
							Name:    "systems",
							Actions: []string{"manage"},
						},
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
				Hierarchy: Hierarchy{
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

func TestIsSystemPermission(t *testing.T) {
	config := &Config{}

	tests := []struct {
		name         string
		permissionID string
		expected     bool
	}{
		{"admin permission", "admin:users", true},
		{"manage permission", "manage:systems", true},
		{"view permission", "view:reports", true},
		{"create permission", "create:accounts", true},
		{"read permission", "read:data", true},
		{"update permission", "update:profile", true},
		{"delete permission", "delete:files", true},
		{"destroy permission", "destroy:system", true},
		{"audit permission", "audit:logs", true},
		{"backup permission", "backup:data", true},
		{"invalid permission", "invalid:permission", false},
		{"custom permission", "custom:action", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.isSystemPermission(tt.permissionID)
			if result != tt.expected {
				t.Errorf("expected %v for permission %q, got %v", tt.expected, tt.permissionID, result)
			}
		})
	}
}

func TestGetUserTypeRoles(t *testing.T) {
	config := &Config{}

	roles := []Role{
		{ID: "admin", Name: "Admin", Type: "user"},
		{ID: "owner", Name: "Owner", Type: "organization"},
		{ID: "support", Name: "Support", Type: ""},
		{ID: "system", Name: "System", Type: "system"},
	}

	userRoles := config.GetUserTypeRoles(roles)

	expectedCount := 2 // admin (type: user) and support (type: empty)
	if len(userRoles) != expectedCount {
		t.Errorf("expected %d user roles, got %d", expectedCount, len(userRoles))
	}

	// Check that the correct roles are returned
	found := make(map[string]bool)
	for _, role := range userRoles {
		found[role.ID] = true
	}

	if !found["admin"] {
		t.Error("expected admin role to be included")
	}
	if !found["support"] {
		t.Error("expected support role to be included")
	}
	if found["owner"] {
		t.Error("expected owner role to be excluded")
	}
	if found["system"] {
		t.Error("expected system role to be excluded")
	}
}

func TestGetAllPermissions(t *testing.T) {
	config := &Config{
		Hierarchy: Hierarchy{
			OrganizationRoles: []Role{
				{
					ID:   "owner",
					Name: "Owner",
					Type: "user",
					Permissions: []Permission{
						{ID: "manage:systems"},
						{ID: "admin:users"},
					},
				},
				{
					ID:   "distributor",
					Name: "Distributor",
					Type: "organization", // Should be excluded
					Permissions: []Permission{
						{ID: "excluded:permission"},
					},
				},
			},
			UserRoles: []Role{
				{
					ID:   "admin",
					Name: "Admin",
					Type: "user",
					Permissions: []Permission{
						{ID: "read:systems"},
						{ID: "manage:systems"}, // Duplicate, should be deduplicated
					},
				},
				{
					ID:   "support",
					Name: "Support",
					Type: "", // Empty type should be included
					Permissions: []Permission{
						{ID: "view:reports"},
					},
				},
			},
		},
	}

	allPermissions := config.GetAllPermissions()

	expectedPermissions := []string{"manage:systems", "admin:users", "read:systems", "view:reports"}
	if len(allPermissions) != len(expectedPermissions) {
		t.Errorf("expected %d permissions, got %d", len(expectedPermissions), len(allPermissions))
	}

	for _, expected := range expectedPermissions {
		if _, found := allPermissions[expected]; !found {
			t.Errorf("expected permission %q to be included", expected)
		}
	}

	// Check that excluded permission is not included
	if _, found := allPermissions["excluded:permission"]; found {
		t.Error("expected excluded:permission to not be included")
	}
}
