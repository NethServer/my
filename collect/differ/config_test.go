/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package differ

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name          string
		configPath    string
		expectError   bool
		errorContains string
		setupFile     func(path string) error
		cleanup       func(path string)
	}{
		{
			name:        "default config when no file",
			configPath:  "",
			expectError: false,
		},
		{
			name:        "default config when file doesn't exist",
			configPath:  "/nonexistent/config.yml",
			expectError: false,
		},
		{
			name:        "valid YAML config",
			configPath:  "test_config.yml",
			expectError: false,
			setupFile: func(path string) error {
				content := `
categorization:
  os:
    patterns:
      - "os\\."
      - "kernel"
    description: "Operating system"
  default:
    name: "system"
    description: "General system"
severity:
  critical:
    conditions:
      - change_type: "delete"
        patterns: ["critical"]
    description: "Critical changes"
  default:
    level: "medium"
    description: "Default severity"
significance:
  always_significant:
    - "critical"
  never_significant:
    - "timestamp"
  default:
    significant: true
    description: "Default significance"
limits:
  max_diff_depth: 5
  max_diffs_per_run: 100
  max_field_path_length: 200
trends:
  enabled: true
  window_hours: 12
  min_occurrences: 2
notifications:
  grouping:
    enabled: true
    time_window_minutes: 15
    max_group_size: 5
  rate_limiting:
    enabled: true
    max_notifications_per_hour: 25
    max_critical_per_hour: 5
`
				return os.WriteFile(path, []byte(content), 0644)
			},
			cleanup: func(path string) {
				_ = os.Remove(path)
			},
		},
		{
			name:          "invalid YAML syntax",
			configPath:    "invalid_config.yml",
			expectError:   true,
			errorContains: "parse",
			setupFile: func(path string) error {
				content := `
invalid yaml: [
  missing closing bracket
`
				return os.WriteFile(path, []byte(content), 0644)
			},
			cleanup: func(path string) {
				_ = os.Remove(path)
			},
		},
		{
			name:          "invalid config values",
			configPath:    "invalid_values.yml",
			expectError:   true,
			errorContains: "invalid configuration",
			setupFile: func(path string) error {
				content := `
limits:
  max_diff_depth: -1
  max_diffs_per_run: 0
  max_field_path_length: -5
trends:
  enabled: true
  window_hours: -1
  min_occurrences: 0
`
				return os.WriteFile(path, []byte(content), 0644)
			},
			cleanup: func(path string) {
				_ = os.Remove(path)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test file if needed
			if tt.setupFile != nil {
				if err := tt.setupFile(tt.configPath); err != nil {
					t.Fatalf("Failed to setup test file: %v", err)
				}
			}

			// Cleanup after test
			if tt.cleanup != nil {
				defer tt.cleanup(tt.configPath)
			}

			config, err := LoadConfig(tt.configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if config == nil {
				t.Error("Expected non-nil config")
				return
			}

			// Validate config structure
			validateTestConfig(t, config)
		})
	}
}

func validateTestConfig(t *testing.T, config *DifferConfig) {
	// Check limits are positive
	if config.Limits.MaxDiffDepth <= 0 {
		t.Error("Expected positive MaxDiffDepth")
	}
	if config.Limits.MaxDiffsPerRun <= 0 {
		t.Error("Expected positive MaxDiffsPerRun")
	}
	if config.Limits.MaxFieldPathLength <= 0 {
		t.Error("Expected positive MaxFieldPathLength")
	}

	// Check trends configuration
	if config.Trends.WindowHours <= 0 {
		t.Error("Expected positive WindowHours")
	}
	if config.Trends.MinOccurrences <= 0 {
		t.Error("Expected positive MinOccurrences")
	}

	// Check default values are set
	if config.Categorization.Default.Name == "" {
		t.Error("Expected non-empty default category name")
	}
	if config.Severity.Default.Level == "" {
		t.Error("Expected non-empty default severity level")
	}
}

func TestNewConfigurableDiffer(t *testing.T) {
	tests := []struct {
		name          string
		configPath    string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid empty path",
			configPath:  "",
			expectError: false,
		},
		{
			name:        "valid nonexistent path",
			configPath:  "/nonexistent/config.yml",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			differ, err := NewConfigurableDiffer(tt.configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if differ == nil {
				t.Error("Expected non-nil differ")
				return
			}

			// Verify differ is properly initialized
			if differ.config == nil {
				t.Error("Expected non-nil config")
			}

			if differ.loadTime.IsZero() {
				t.Error("Expected non-zero load time")
			}

			// Test pattern compilation worked
			if len(differ.categoryPatterns) == 0 {
				t.Error("Expected compiled category patterns")
			}
		})
	}
}

func TestConfigurableDiffer_ReloadConfig(t *testing.T) {
	differ, err := NewConfigurableDiffer("")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	originalLoadTime := differ.GetLoadTime()

	// Wait to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	// Reload with empty path
	err = differ.ReloadConfig("")
	if err != nil {
		t.Errorf("Unexpected error reloading config: %v", err)
	}

	newLoadTime := differ.GetLoadTime()
	if !newLoadTime.After(originalLoadTime) {
		t.Error("Expected load time to be updated after reload")
	}

	// Verify config is still valid
	config := differ.GetConfig()
	if config == nil {
		t.Error("Expected non-nil config after reload")
	}
}

func TestConfigurableDiffer_GetMethods(t *testing.T) {
	differ, err := NewConfigurableDiffer("")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	// Test GetConfig
	config := differ.GetConfig()
	if config == nil {
		t.Error("Expected non-nil config")
	}

	// Test GetLoadTime
	loadTime := differ.GetLoadTime()
	if loadTime.IsZero() {
		t.Error("Expected non-zero load time")
	}

	// Test GetAllCategories
	categories := differ.GetAllCategories()
	if len(categories) == 0 {
		t.Error("Expected at least one category")
	}

	expectedCategories := []string{"os", "hardware", "network", "features"}
	for _, expected := range expectedCategories {
		found := false
		for cat := range categories {
			if cat == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected category '%s' to be in list", expected)
		}
	}

	// Test GetAllSeverityLevels
	severities := differ.GetAllSeverityLevels()
	if len(severities) == 0 {
		t.Error("Expected at least one severity level")
	}

	expectedSeverities := []string{"critical", "high", "medium", "low"}
	for _, expected := range expectedSeverities {
		found := false
		for sev := range severities {
			if sev == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected severity '%s' to be in list", expected)
		}
	}

	// Test GetSignificanceFilters
	filters := differ.GetSignificanceFilters()
	if len(filters) == 0 {
		t.Error("Expected at least one significance filter")
	}
}

func TestValidateConfigFunc(t *testing.T) {
	tests := []struct {
		name          string
		config        *DifferConfig
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid config",
			config:      getDefaultConfig(),
			expectError: false,
		},
		{
			name: "invalid max_diff_depth",
			config: &DifferConfig{
				Limits: LimitsConfig{
					MaxDiffDepth:       -1,
					MaxDiffsPerRun:     1000,
					MaxFieldPathLength: 500,
				},
				Trends: TrendsConfig{
					WindowHours:    24,
					MinOccurrences: 3,
				},
			},
			expectError:   true,
			errorContains: "max_diff_depth must be positive",
		},
		{
			name: "invalid max_diffs_per_run",
			config: &DifferConfig{
				Limits: LimitsConfig{
					MaxDiffDepth:       10,
					MaxDiffsPerRun:     0,
					MaxFieldPathLength: 500,
				},
				Trends: TrendsConfig{
					WindowHours:    24,
					MinOccurrences: 3,
				},
			},
			expectError:   true,
			errorContains: "max_diffs_per_run must be positive",
		},
		{
			name: "invalid trends window_hours",
			config: &DifferConfig{
				Limits: LimitsConfig{
					MaxDiffDepth:       10,
					MaxDiffsPerRun:     1000,
					MaxFieldPathLength: 500,
				},
				Trends: TrendsConfig{
					WindowHours:    -1,
					MinOccurrences: 3,
				},
			},
			expectError:   true,
			errorContains: "window_hours must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errorContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGetDefaultConfig(t *testing.T) {
	config := getDefaultConfig()

	if config == nil {
		t.Fatal("Expected non-nil default config")
	}

	// Validate the default config
	validateTestConfig(t, config)

	// Check specific default values
	if config.Categorization.Default.Name != "system" {
		t.Errorf("Expected default category 'system', got '%s'", config.Categorization.Default.Name)
	}

	if config.Severity.Default.Level != "medium" {
		t.Errorf("Expected default severity 'medium', got '%s'", config.Severity.Default.Level)
	}

	if !config.Significance.Default.Significant {
		t.Error("Expected default significance to be true")
	}

	if !config.Trends.Enabled {
		t.Error("Expected trends to be enabled by default")
	}
}

func TestConfigurableDifferPatternCompilation(t *testing.T) {
	// Create a custom config file with known patterns
	tmpFile := filepath.Join(os.TempDir(), "test_pattern_config.yml")
	content := `
categorization:
  test_category:
    patterns:
      - "test\\..*"
      - "example\\."
    description: "Test category"
  default:
    name: "default"
    description: "Default category"
severity:
  critical:
    conditions:
      - change_type: "delete"
        patterns: ["critical\\..*"]
    description: "Critical changes"
  default:
    level: "medium"
    description: "Default severity"
significance:
  always_significant:
    - "always\\..*"
  never_significant:
    - "never\\..*"
  default:
    significant: true
    description: "Default significance"
limits:
  max_diff_depth: 10
  max_diffs_per_run: 1000
  max_field_path_length: 500
trends:
  enabled: true
  window_hours: 24
  min_occurrences: 3
notifications:
  grouping:
    enabled: true
    time_window_minutes: 30
    max_group_size: 10
  rate_limiting:
    enabled: true
    max_notifications_per_hour: 50
    max_critical_per_hour: 10
`

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	differ, err := NewConfigurableDiffer(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create differ with test config: %v", err)
	}

	// Test pattern matching
	tests := []struct {
		fieldPath        string
		expectedCategory string
	}{
		{"test.field", "test_category"},
		{"example.value", "test_category"},
		{"other.field", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldPath, func(t *testing.T) {
			category := differ.CategorizeField(tt.fieldPath)
			if category != tt.expectedCategory {
				t.Errorf("Expected category %s, got %s for field %s", tt.expectedCategory, category, tt.fieldPath)
			}
		})
	}
}
