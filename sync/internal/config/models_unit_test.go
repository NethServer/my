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

// Helper function for substring matching
func containsSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
