/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package differ

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nethesis/my/collect/models"
)

func TestCompareInventoryStructures(t *testing.T) {
	tests := []struct {
		name      string
		data1     string
		data2     string
		expected  bool
		expectErr bool
	}{
		{
			name: "identical structures",
			data1: `{
				"os": {"version": "20.04"},
				"networking": {"hostname": "test"},
				"processors": {"count": 4}
			}`,
			data2: `{
				"os": {"version": "22.04"},
				"networking": {"hostname": "prod"},
				"processors": {"count": 8}
			}`,
			expected:  true,
			expectErr: false,
		},
		{
			name: "compatible structures with different nesting",
			data1: `{
				"os": {"version": "20.04", "kernel": "5.4.0"},
				"networking": {"hostname": "test", "interfaces": {"eth0": "192.168.1.10"}}
			}`,
			data2: `{
				"os": {"version": "22.04", "kernel": "5.15.0"},
				"networking": {"hostname": "prod", "interfaces": {"eth0": "10.0.0.10"}}
			}`,
			expected:  true,
			expectErr: false,
		},
		{
			name: "incompatible structures - missing key",
			data1: `{
				"os": {"version": "20.04"},
				"networking": {"hostname": "test"},
				"processors": {"count": 4}
			}`,
			data2: `{
				"os": {"version": "22.04"},
				"networking": {"hostname": "prod"}
			}`,
			expected:  false,
			expectErr: false,
		},
		{
			name:      "invalid JSON in first structure",
			data1:     `{invalid json`,
			data2:     `{"valid": "json"}`,
			expected:  false,
			expectErr: true,
		},
		{
			name:      "invalid JSON in second structure",
			data1:     `{"valid": "json"}`,
			data2:     `{invalid json`,
			expected:  false,
			expectErr: true,
		},
		{
			name:      "empty structures",
			data1:     `{}`,
			data2:     `{}`,
			expected:  true,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareInventoryStructures(
				json.RawMessage(tt.data1),
				json.RawMessage(tt.data2),
			)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected compatibility %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSanitizeFieldPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal field path",
			input:    "os.version",
			expected: "os.version",
		},
		{
			name:     "field path with newlines",
			input:    "os\nversion",
			expected: "osversion",
		},
		{
			name:     "field path with carriage returns",
			input:    "os\rversion",
			expected: "osversion",
		},
		{
			name:     "field path with tabs",
			input:    "os\tversion",
			expected: "osversion",
		},
		{
			name:     "field path with multiple whitespace types",
			input:    "os\n\r\tversion",
			expected: "osversion",
		},
		{
			name:     "very long field path",
			input:    strings.Repeat("a", 600),
			expected: strings.Repeat("a", 500) + "...",
		},
		{
			name:     "empty field path",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFieldPath(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestValidateFieldPath(t *testing.T) {
	tests := []struct {
		name          string
		fieldPath     string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid field path",
			fieldPath:   "os.version",
			expectError: false,
		},
		{
			name:          "empty field path",
			fieldPath:     "",
			expectError:   true,
			errorContains: "cannot be empty",
		},
		{
			name:          "field path too long",
			fieldPath:     strings.Repeat("a", 501),
			expectError:   true,
			errorContains: "exceeds maximum length",
		},
		{
			name:          "field path with newline",
			fieldPath:     "os\nversion",
			expectError:   true,
			errorContains: "invalid character",
		},
		{
			name:          "field path with carriage return",
			fieldPath:     "os\rversion",
			expectError:   true,
			errorContains: "invalid character",
		},
		{
			name:          "field path with tab",
			fieldPath:     "os\tversion",
			expectError:   true,
			errorContains: "invalid character",
		},
		{
			name:          "field path with null character",
			fieldPath:     "os\x00version",
			expectError:   true,
			errorContains: "invalid character",
		},
		{
			name:        "field path with dots",
			fieldPath:   "os.version.detail",
			expectError: false,
		},
		{
			name:        "field path with underscores",
			fieldPath:   "uptime_seconds",
			expectError: false,
		},
		{
			name:        "field path at maximum length",
			fieldPath:   strings.Repeat("a", 500),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFieldPath(tt.fieldPath)

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

func TestFormatDiffSummary(t *testing.T) {
	tests := []struct {
		name     string
		diffs    []models.InventoryDiff
		expected string
	}{
		{
			name:     "empty diffs",
			diffs:    []models.InventoryDiff{},
			expected: "No differences found",
		},
		{
			name: "single diff",
			diffs: []models.InventoryDiff{
				{
					FieldPath: "os.version",
					Category:  "os",
					Severity:  "high",
					DiffType:  "update",
				},
			},
			expected: "Total: 1 changes | Categories: os: 1 | Severities: high: 1",
		},
		{
			name: "multiple diffs",
			diffs: []models.InventoryDiff{
				{FieldPath: "os.version", Category: "os", Severity: "high", DiffType: "update"},
				{FieldPath: "memory.total", Category: "hardware", Severity: "medium", DiffType: "update"},
				{FieldPath: "networking.hostname", Category: "network", Severity: "low", DiffType: "update"},
				{FieldPath: "features.docker", Category: "features", Severity: "medium", DiffType: "create"},
			},
			expected: "Total: 4 changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDiffSummary(tt.diffs)

			// For multiple diffs, just check it starts with the total
			if len(tt.diffs) > 1 {
				if !strings.HasPrefix(result, tt.expected) {
					t.Errorf("Expected result to start with '%s', got '%s'", tt.expected, result)
				}
			} else {
				if result != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, result)
				}
			}
		})
	}
}

func TestCalculateChangeVelocity(t *testing.T) {
	tests := []struct {
		name       string
		diffs      []models.InventoryDiff
		timeWindow time.Duration
		expected   float64
	}{
		{
			name:       "empty diffs",
			diffs:      []models.InventoryDiff{},
			timeWindow: time.Hour,
			expected:   0.0,
		},
		{
			name: "10 changes in 1 hour",
			diffs: []models.InventoryDiff{
				{FieldPath: "change1"}, {FieldPath: "change2"}, {FieldPath: "change3"},
				{FieldPath: "change4"}, {FieldPath: "change5"}, {FieldPath: "change6"},
				{FieldPath: "change7"}, {FieldPath: "change8"}, {FieldPath: "change9"},
				{FieldPath: "change10"},
			},
			timeWindow: time.Hour,
			expected:   10.0,
		},
		{
			name: "5 changes in 30 minutes",
			diffs: []models.InventoryDiff{
				{FieldPath: "change1"}, {FieldPath: "change2"}, {FieldPath: "change3"},
				{FieldPath: "change4"}, {FieldPath: "change5"},
			},
			timeWindow: 30 * time.Minute,
			expected:   10.0, // 5 changes / 0.5 hours = 10 changes/hour
		},
		{
			name:       "zero time window",
			diffs:      []models.InventoryDiff{{FieldPath: "change1"}},
			timeWindow: 0,
			expected:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateChangeVelocity(tt.diffs, tt.timeWindow)
			if result != tt.expected {
				t.Errorf("Expected velocity %.2f, got %.2f", tt.expected, result)
			}
		})
	}
}

func TestCalculateNoisiness(t *testing.T) {
	tests := []struct {
		name     string
		diffs    []models.InventoryDiff
		expected float64
	}{
		{
			name:     "empty diffs",
			diffs:    []models.InventoryDiff{},
			expected: 0.0,
		},
		{
			name: "all noisy changes",
			diffs: []models.InventoryDiff{
				{FieldPath: "system.timestamp", Severity: "medium"}, // noisy (pattern)
				{FieldPath: "metrics.uptime", Severity: "medium"},   // noisy (pattern)
				{FieldPath: "performance.cpu", Severity: "low"},     // noisy (pattern + severity = 2 counts)
			},
			expected: 133.3, // 4 noise points out of 3 changes = 133.3%
		},
		{
			name: "mixed noisy and clean changes",
			diffs: []models.InventoryDiff{
				{FieldPath: "system.timestamp", Severity: "medium"}, // noisy (pattern only)
				{FieldPath: "os.version", Severity: "high"},         // clean
				{FieldPath: "metrics.cpu", Severity: "medium"},      // noisy (pattern only)
				{FieldPath: "memory.total", Severity: "low"},        // noisy (severity only)
			},
			expected: 75.0, // 3 out of 4 are noisy
		},
		{
			name: "no noisy changes",
			diffs: []models.InventoryDiff{
				{FieldPath: "os.version", Severity: "high"},
				{FieldPath: "memory.total", Severity: "medium"},
				{FieldPath: "networking.hostname", Severity: "medium"},
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateNoisiness(tt.diffs)
			// Use a small tolerance for floating point comparison
			tolerance := 0.1
			if result < tt.expected-tolerance || result > tt.expected+tolerance {
				t.Errorf("Expected noisiness %.1f%%, got %.1f%%", tt.expected, result)
			}
		})
	}
}

func TestFilterDiffsByCategory(t *testing.T) {
	diffs := []models.InventoryDiff{
		{FieldPath: "os.version", Category: "os"},
		{FieldPath: "memory.total", Category: "hardware"},
		{FieldPath: "networking.hostname", Category: "network"},
		{FieldPath: "features.docker", Category: "features"},
	}

	tests := []struct {
		name       string
		categories []string
		expected   int
	}{
		{
			name:       "filter by single category",
			categories: []string{"os"},
			expected:   1,
		},
		{
			name:       "filter by multiple categories",
			categories: []string{"os", "hardware"},
			expected:   2,
		},
		{
			name:       "filter by non-existent category",
			categories: []string{"non-existent"},
			expected:   0,
		},
		{
			name:       "empty categories list",
			categories: []string{},
			expected:   4, // Should return all diffs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterDiffsByCategory(diffs, tt.categories)
			if len(result) != tt.expected {
				t.Errorf("Expected %d filtered diffs, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestFilterDiffsBySeverity(t *testing.T) {
	diffs := []models.InventoryDiff{
		{FieldPath: "critical.error", Severity: "critical"},
		{FieldPath: "os.version", Severity: "high"},
		{FieldPath: "config.service", Severity: "medium"},
		{FieldPath: "metrics.cpu", Severity: "low"},
	}

	tests := []struct {
		name       string
		severities []string
		expected   int
	}{
		{
			name:       "filter by single severity",
			severities: []string{"critical"},
			expected:   1,
		},
		{
			name:       "filter by multiple severities",
			severities: []string{"high", "critical"},
			expected:   2,
		},
		{
			name:       "filter by non-existent severity",
			severities: []string{"super-critical"},
			expected:   0,
		},
		{
			name:       "empty severities list",
			severities: []string{},
			expected:   4, // Should return all diffs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterDiffsBySeverity(diffs, tt.severities)
			if len(result) != tt.expected {
				t.Errorf("Expected %d filtered diffs, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestFindTopChangedPaths(t *testing.T) {
	diffs := []models.InventoryDiff{
		{FieldPath: "os.version"},
		{FieldPath: "os.kernel"},
		{FieldPath: "os.release"},
		{FieldPath: "memory.total"},
		{FieldPath: "memory.free"},
		{FieldPath: "networking.hostname"},
		{FieldPath: "features.docker"},
	}

	tests := []struct {
		name     string
		limit    int
		expected int
		topPath  string
		topCount int
	}{
		{
			name:     "find top 3 paths",
			limit:    3,
			expected: 3,
			topPath:  "os",
			topCount: 3,
		},
		{
			name:     "find all paths",
			limit:    0,
			expected: 4, // os, memory, networking, features
			topPath:  "os",
			topCount: 3,
		},
		{
			name:     "limit greater than available",
			limit:    10,
			expected: 4,
			topPath:  "os",
			topCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindTopChangedPaths(diffs, tt.limit)

			if len(result) != tt.expected {
				t.Errorf("Expected %d path frequencies, got %d", tt.expected, len(result))
			}

			if len(result) > 0 {
				if result[0].Path != tt.topPath {
					t.Errorf("Expected top path '%s', got '%s'", tt.topPath, result[0].Path)
				}
				if result[0].Count != tt.topCount {
					t.Errorf("Expected top count %d, got %d", tt.topCount, result[0].Count)
				}
			}
		})
	}
}

func TestDetectAnomalies(t *testing.T) {
	diffs := []models.InventoryDiff{
		{FieldPath: "system.error", Severity: "critical", DiffType: "create"},
		{FieldPath: "important.service", Severity: "high", DiffType: "delete"},
		{FieldPath: "os.version", Severity: "high", DiffType: "update"},
		{FieldPath: "config.service", Severity: "medium", DiffType: "update"},
		{FieldPath: "temporary.file", Severity: "low", DiffType: "delete"},
		{FieldPath: "metrics.cpu", Severity: "low", DiffType: "update"},
	}

	anomalies := DetectAnomalies(diffs)

	// Should detect:
	// 1. Critical severity change (system.error)
	// 2. Non-temporary delete (important.service)
	// 3. OS change (os.version)
	expectedCount := 3

	if len(anomalies) != expectedCount {
		t.Errorf("Expected %d anomalies, got %d", expectedCount, len(anomalies))
	}

	// Check that specific anomalies are detected
	anomalyPaths := make(map[string]bool)
	for _, anomaly := range anomalies {
		anomalyPaths[anomaly.FieldPath] = true
	}

	expectedAnomalies := []string{"system.error", "important.service", "os.version"}
	for _, expected := range expectedAnomalies {
		if !anomalyPaths[expected] {
			t.Errorf("Expected %s to be detected as an anomaly", expected)
		}
	}
}

func TestValidateDiffConsistency(t *testing.T) {
	tests := []struct {
		name           string
		diffs          []models.InventoryDiff
		expectedIssues int
		expectIssue    string
	}{
		{
			name: "valid diffs",
			diffs: []models.InventoryDiff{
				{FieldPath: "os.version", Category: "os", Severity: "high"},
				{FieldPath: "memory.total", Category: "hardware", Severity: "medium"},
			},
			expectedIssues: 0,
		},
		{
			name: "duplicate field paths",
			diffs: []models.InventoryDiff{
				{FieldPath: "os.version", Category: "os", Severity: "high"},
				{FieldPath: "os.version", Category: "os", Severity: "medium"},
			},
			expectedIssues: 1,
			expectIssue:    "Duplicate field path",
		},
		{
			name: "invalid field path",
			diffs: []models.InventoryDiff{
				{FieldPath: "invalid\nfield", Category: "os", Severity: "high"},
			},
			expectedIssues: 1,
			expectIssue:    "Invalid field path",
		},
		{
			name: "missing category",
			diffs: []models.InventoryDiff{
				{FieldPath: "os.version", Category: "", Severity: "high"},
			},
			expectedIssues: 1,
			expectIssue:    "Missing category",
		},
		{
			name: "missing severity",
			diffs: []models.InventoryDiff{
				{FieldPath: "os.version", Category: "os", Severity: ""},
			},
			expectedIssues: 1,
			expectIssue:    "Missing severity",
		},
		{
			name: "multiple issues",
			diffs: []models.InventoryDiff{
				{FieldPath: "os.version", Category: "", Severity: ""},
				{FieldPath: "invalid\nfield", Category: "os", Severity: "high"},
			},
			expectedIssues: 3, // Missing category, missing severity, invalid field path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := ValidateDiffConsistency(tt.diffs)

			if len(issues) != tt.expectedIssues {
				t.Errorf("Expected %d issues, got %d", tt.expectedIssues, len(issues))
			}

			if tt.expectIssue != "" && len(issues) > 0 {
				found := false
				for _, issue := range issues {
					if strings.Contains(issue, tt.expectIssue) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected issue containing '%s', got issues: %v", tt.expectIssue, issues)
				}
			}
		})
	}
}

func TestFormatDiffForDisplay(t *testing.T) {
	diff := models.InventoryDiff{
		FieldPath:     "os.version",
		DiffType:      "update",
		Category:      "os",
		Severity:      "high",
		PreviousValue: "old",
		CurrentValue:  "new",
	}

	result := FormatDiffForDisplay(diff)

	expectedParts := []string{
		"Field: os.version",
		"Type: update",
		"Category: os",
		"Severity: high",
		"Previous:",
		"Current:",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Expected result to contain '%s', got: %s", part, result)
		}
	}
}

func TestCalculateInventoryHealth(t *testing.T) {
	tests := []struct {
		name     string
		diffs    []models.InventoryDiff
		expected float64
	}{
		{
			name:     "no changes - perfect health",
			diffs:    []models.InventoryDiff{},
			expected: 100.0,
		},
		{
			name: "single critical change",
			diffs: []models.InventoryDiff{
				{Severity: "critical"},
			},
			expected: 90.0, // 100 - 10
		},
		{
			name: "mixed severity changes",
			diffs: []models.InventoryDiff{
				{Severity: "critical"}, // -10
				{Severity: "high"},     // -5
				{Severity: "medium"},   // -2
				{Severity: "low"},      // -1
			},
			expected: 82.0, // 100 - 18
		},
		{
			name: "many critical changes - minimum health",
			diffs: []models.InventoryDiff{
				{Severity: "critical"}, {Severity: "critical"}, {Severity: "critical"},
				{Severity: "critical"}, {Severity: "critical"}, {Severity: "critical"},
				{Severity: "critical"}, {Severity: "critical"}, {Severity: "critical"},
				{Severity: "critical"}, {Severity: "critical"}, // 11 critical = -110
			},
			expected: 0.0, // Can't go below 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateInventoryHealth(tt.diffs)
			if result != tt.expected {
				t.Errorf("Expected health score %.1f, got %.1f", tt.expected, result)
			}
		})
	}
}

func TestGetDiffMetrics(t *testing.T) {
	diffs := []models.InventoryDiff{
		{FieldPath: "os.version", Category: "os", Severity: "high", DiffType: "update"},
		{FieldPath: "memory.total", Category: "hardware", Severity: "medium", DiffType: "update"},
		{FieldPath: "critical.error", Category: "system", Severity: "critical", DiffType: "create"},
		{FieldPath: "metrics.cpu", Category: "system", Severity: "low", DiffType: "update"},
	}

	metrics := GetDiffMetrics(diffs)

	// Check required fields
	requiredFields := []string{
		"total_changes",
		"category_distribution",
		"severity_distribution",
		"type_distribution",
		"noisiness",
		"health_score",
		"anomaly_count",
		"top_changed_paths",
	}

	for _, field := range requiredFields {
		if _, exists := metrics[field]; !exists {
			t.Errorf("Expected metric field '%s' to exist", field)
		}
	}

	// Check total changes
	if totalChanges, ok := metrics["total_changes"].(int); !ok || totalChanges != 4 {
		t.Errorf("Expected total_changes to be 4, got %v", metrics["total_changes"])
	}

	// Check category distribution
	if categoryDist, ok := metrics["category_distribution"].(map[string]int); ok {
		if categoryDist["os"] != 1 {
			t.Errorf("Expected 1 OS change, got %d", categoryDist["os"])
		}
		if categoryDist["system"] != 2 {
			t.Errorf("Expected 2 system changes, got %d", categoryDist["system"])
		}
	} else {
		t.Error("Expected category_distribution to be map[string]int")
	}

	// Check severity distribution
	if severityDist, ok := metrics["severity_distribution"].(map[string]int); ok {
		if severityDist["critical"] != 1 {
			t.Errorf("Expected 1 critical change, got %d", severityDist["critical"])
		}
	} else {
		t.Error("Expected severity_distribution to be map[string]int")
	}

	// Check top changed paths
	if topPaths, ok := metrics["top_changed_paths"].([]PathFrequency); ok {
		if len(topPaths) == 0 {
			t.Error("Expected non-empty top_changed_paths")
		}
	} else {
		t.Error("Expected top_changed_paths to be []PathFrequency")
	}
}
