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
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   func(t *testing.T) string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid YAML file loads successfully",
			setupFile: func(t *testing.T) string {
				content := `
metadata:
  name: "Test Config"
  version: "1.0.0"
  description: "Test configuration"
hierarchy:
  organization_roles:
    - id: "owner"
      name: "Owner"
      permissions:
        - id: "manage:systems"
  user_roles:
    - id: "admin"
      name: "Admin"
      permissions:
        - id: "read:systems"
  resources:
    - name: "systems"
      actions: ["read", "manage"]
`
				tmpFile := createTempFile(t, content)
				return tmpFile
			},
			expectError: false,
		},
		{
			name: "empty filename returns error",
			setupFile: func(t *testing.T) string {
				return ""
			},
			expectError: true,
			errorMsg:    "configuration file path is required",
		},
		{
			name: "non-existent file returns error",
			setupFile: func(t *testing.T) string {
				return "/path/that/does/not/exist.yml"
			},
			expectError: true,
			errorMsg:    "configuration file does not exist",
		},
		{
			name: "invalid YAML returns error",
			setupFile: func(t *testing.T) string {
				content := `
metadata:
  name: "Test Config"
  version: 1.0.0
hierarchy:
  invalid_yaml: [
    - missing_closing_bracket
`
				return createTempFile(t, content)
			},
			expectError: true,
			errorMsg:    "failed to parse YAML configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.setupFile(t)
			if filename != "" && filename != "/path/that/does/not/exist.yml" {
				defer func() { _ = os.Remove(filename) }()
			}

			config, err := LoadFromFile(filename)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if config == nil {
				t.Error("expected config to be non-nil")
				return
			}

			// Validate basic structure
			if config.Metadata.Name == "" {
				t.Error("expected metadata name to be set")
			}
			if config.Metadata.Version == "" {
				t.Error("expected metadata version to be set")
			}
		})
	}
}

func TestLoadFromData(t *testing.T) {
	tests := []struct {
		name        string
		yamlData    string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid YAML data loads successfully",
			yamlData: `
metadata:
  name: "Test Config"
  version: "1.0.0"
hierarchy:
  organization_roles:
    - id: "owner"
      name: "Owner"
      permissions:
        - id: "manage:systems"
  user_roles:
    - id: "admin"
      name: "Admin"
      permissions:
        - id: "read:systems"
  resources:
    - name: "systems"
      actions: ["read", "manage"]
`,
			expectError: false,
		},
		{
			name: "invalid YAML data returns error",
			yamlData: `
metadata:
  name: "Test"
hierarchy:
  invalid: [
    - unclosed
`,
			expectError: true,
			errorMsg:    "failed to parse YAML configuration",
		},
		{
			name:        "empty data returns minimal config",
			yamlData:    "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadFromData([]byte(tt.yamlData))

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if config == nil {
				t.Error("expected config to be non-nil")
			}
		})
	}
}

func TestConfigSaveToFile(t *testing.T) {
	config := &Config{
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
	}

	tmpFile := filepath.Join(t.TempDir(), "test-config.yml")

	err := config.SaveToFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file was created and can be loaded back
	loadedConfig, err := LoadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if loadedConfig.Metadata.Name != config.Metadata.Name {
		t.Errorf("expected name %q, got %q", config.Metadata.Name, loadedConfig.Metadata.Name)
	}
	if loadedConfig.Metadata.Version != config.Metadata.Version {
		t.Errorf("expected version %q, got %q", config.Metadata.Version, loadedConfig.Metadata.Version)
	}
}

func TestConfigToYAML(t *testing.T) {
	config := &Config{
		Metadata: Metadata{
			Name:    "Test Config",
			Version: "1.0.0",
		},
		Hierarchy: Hierarchy{
			OrganizationRoles: []Role{
				{
					ID:   "owner",
					Name: "Owner",
				},
			},
		},
	}

	yamlStr, err := config.ToYAML()
	if err != nil {
		t.Fatalf("failed to convert to YAML: %v", err)
	}

	if yamlStr == "" {
		t.Error("expected non-empty YAML string")
	}

	// Verify YAML contains expected fields
	if !contains(yamlStr, "name: Test Config") {
		t.Error("expected YAML to contain metadata name")
	}
	if !contains(yamlStr, "version: 1.0.0") {
		t.Error("expected YAML to contain metadata version")
	}
}

// Helper functions
func createTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "config-test-*.yml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	if err := tmpFile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	return tmpFile.Name()
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) && str[:len(substr)] == substr ||
		len(str) > len(substr) && containsSubstring(str, substr)
}

func containsSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
