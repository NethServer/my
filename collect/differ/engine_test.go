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

func TestNewDiffEngine(t *testing.T) {
	tests := []struct {
		name          string
		configPath    string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid config path",
			configPath:  "", // Will use default config
			expectError: false,
		},
		{
			name:        "invalid config path",
			configPath:  "/nonexistent/config.yml",
			expectError: false, // Should fall back to default config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewDiffEngine(tt.configPath)

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

			if engine == nil {
				t.Error("Expected non-nil engine")
				return
			}

			// Verify engine is configured with defaults
			if engine.maxDepth <= 0 {
				t.Error("Expected positive maxDepth")
			}
			if engine.maxDiffsPerRun <= 0 {
				t.Error("Expected positive maxDiffsPerRun")
			}
			if engine.maxFieldPathLength <= 0 {
				t.Error("Expected positive maxFieldPathLength")
			}
		})
	}
}

func TestDiffEngine_ComputeDiff(t *testing.T) {
	engine, err := NewDiffEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name          string
		systemID      string
		previousData  string
		currentData   string
		expectError   bool
		errorContains string
		expectedDiffs int
		minDiffs      int
	}{
		{
			name:     "simple OS version change",
			systemID: "test-system-1",
			previousData: `{
				"os": {"version": "20.04"},
				"networking": {"hostname": "test"},
				"processors": {"count": 4},
				"memory": {"total": 8192}
			}`,
			currentData: `{
				"os": {"version": "22.04"},
				"networking": {"hostname": "test"},
				"processors": {"count": 4},
				"memory": {"total": 8192}
			}`,
			expectError: false,
			minDiffs:    1,
		},
		{
			name:     "multiple changes",
			systemID: "test-system-2",
			previousData: `{
				"os": {"version": "20.04", "kernel": "5.4.0"},
				"networking": {"hostname": "old-host", "public_ip": "1.2.3.4"},
				"processors": {"count": 4},
				"memory": {"total": 8192}
			}`,
			currentData: `{
				"os": {"version": "22.04", "kernel": "5.15.0"},
				"networking": {"hostname": "new-host", "public_ip": "5.6.7.8"},
				"processors": {"count": 8},
				"memory": {"total": 16384}
			}`,
			expectError: false,
			minDiffs:    4,
		},
		{
			name:     "new field added",
			systemID: "test-system-3",
			previousData: `{
				"os": {"version": "20.04"},
				"networking": {"hostname": "test"}
			}`,
			currentData: `{
				"os": {"version": "20.04"},
				"networking": {"hostname": "test"},
				"features": {"docker": true}
			}`,
			expectError: false,
			minDiffs:    1,
		},
		{
			name:     "field deleted",
			systemID: "test-system-4",
			previousData: `{
				"os": {"version": "20.04"},
				"networking": {"hostname": "test"},
				"old_feature": {"enabled": true}
			}`,
			currentData: `{
				"os": {"version": "20.04"},
				"networking": {"hostname": "test"}
			}`,
			expectError: false,
			minDiffs:    1,
		},
		{
			name:     "no changes",
			systemID: "test-system-5",
			previousData: `{
				"os": {"version": "20.04"},
				"networking": {"hostname": "test"}
			}`,
			currentData: `{
				"os": {"version": "20.04"},
				"networking": {"hostname": "test"}
			}`,
			expectError:   false,
			expectedDiffs: 0,
		},
		{
			name:          "invalid previous JSON",
			systemID:      "test-system-6",
			previousData:  `{invalid json`,
			currentData:   `{"valid": "json"}`,
			expectError:   true,
			errorContains: "inventory validation failed",
		},
		{
			name:          "invalid current JSON",
			systemID:      "test-system-7",
			previousData:  `{"valid": "json"}`,
			currentData:   `{invalid json`,
			expectError:   true,
			errorContains: "inventory validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test inventory records
			previous := &models.InventoryRecord{
				ID:        1,
				SystemID:  tt.systemID,
				Data:      json.RawMessage(tt.previousData),
				Timestamp: time.Now().Add(-time.Hour),
			}

			current := &models.InventoryRecord{
				ID:        2,
				SystemID:  tt.systemID,
				Data:      json.RawMessage(tt.currentData),
				Timestamp: time.Now(),
			}

			diffs, err := engine.ComputeDiff(tt.systemID, previous, current)

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

			// Check expected number of diffs
			if tt.expectedDiffs > 0 && len(diffs) != tt.expectedDiffs {
				t.Errorf("Expected exactly %d diffs, got %d", tt.expectedDiffs, len(diffs))
			}

			if tt.minDiffs > 0 && len(diffs) < tt.minDiffs {
				t.Errorf("Expected at least %d diffs, got %d", tt.minDiffs, len(diffs))
			}

			// Validate diff structure
			for i, diff := range diffs {
				if diff.SystemID != tt.systemID {
					t.Errorf("Diff %d: expected SystemID %s, got %s", i, tt.systemID, diff.SystemID)
				}

				if diff.PreviousID == nil || *diff.PreviousID != previous.ID {
					t.Errorf("Diff %d: expected PreviousID %d, got %v", i, previous.ID, diff.PreviousID)
				}

				if diff.CurrentID != current.ID {
					t.Errorf("Diff %d: expected CurrentID %d, got %d", i, current.ID, diff.CurrentID)
				}

				if diff.FieldPath == "" {
					t.Errorf("Diff %d: expected non-empty FieldPath", i)
				}

				if diff.DiffType == "" {
					t.Errorf("Diff %d: expected non-empty DiffType", i)
				}

				if diff.Category == "" {
					t.Errorf("Diff %d: expected non-empty Category", i)
				}

				if diff.Severity == "" {
					t.Errorf("Diff %d: expected non-empty Severity", i)
				}
			}
		})
	}
}

func TestDiffEngine_FormatPath(t *testing.T) {
	engine, err := NewDiffEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name       string
		path       []string
		expectPath string
	}{
		{
			name:       "simple field path",
			path:       []string{"os", "version"},
			expectPath: "os.version",
		},
		{
			name:       "root level path",
			path:       []string{"hostname"},
			expectPath: "hostname",
		},
		{
			name:       "empty path",
			path:       []string{},
			expectPath: "root",
		},
		{
			name:       "deep path",
			path:       []string{"config", "services", "nginx", "status"},
			expectPath: "config.services.nginx.status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.formatPath(tt.path)
			if result != tt.expectPath {
				t.Errorf("Expected path %s, got %s", tt.expectPath, result)
			}
		})
	}
}

func TestDiffEngine_ValueToString(t *testing.T) {
	engine, err := NewDiffEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "string value",
			value:    "test",
			expected: "test",
		},
		{
			name:     "bool true",
			value:    true,
			expected: "true",
		},
		{
			name:     "bool false",
			value:    false,
			expected: "false",
		},
		{
			name:     "int value",
			value:    42,
			expected: "42",
		},
		{
			name:     "int64 value",
			value:    int64(123456789),
			expected: "123456789",
		},
		{
			name:     "float32 value",
			value:    float32(3.14),
			expected: "3.14",
		},
		{
			name:     "float64 value",
			value:    3.14159,
			expected: "3.14",
		},
		{
			name:     "nil value",
			value:    nil,
			expected: "null",
		},
		{
			name:     "map value",
			value:    map[string]string{"key": "value"},
			expected: `{"key":"value"}`,
		},
		{
			name:     "slice value",
			value:    []string{"a", "b", "c"},
			expected: `["a","b","c"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.valueToString(tt.value)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestDiffEngine_GroupRelatedChanges(t *testing.T) {
	engine, err := NewDiffEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Use NS8/NSEC field paths
	diffs := []models.InventoryDiff{
		{FieldPath: "facts.distro.version", Category: "os"},
		{FieldPath: "facts.distro.release", Category: "os"},
		{FieldPath: "facts.network.hostname", Category: "network"},
		{FieldPath: "facts.cluster.public_ip", Category: "cluster"},
		{FieldPath: "facts.processors.count", Category: "hardware"},
		{FieldPath: "facts.memory.total", Category: "hardware"},
		{FieldPath: "facts.features.docker", Category: "features"},
	}

	groups := engine.GroupRelatedChanges(diffs)

	// With NS8 structure, we expect: operating_system, network, cluster, hardware, features_docker
	if len(groups) == 0 {
		t.Error("Expected at least one group")
	}

	// Check that distro changes are grouped together as operating_system
	if osGroup, exists := groups["operating_system"]; exists {
		if len(osGroup) != 2 {
			t.Errorf("Expected 2 OS changes, got %d", len(osGroup))
		}
	} else {
		t.Error("Expected operating_system group to exist")
	}

	// Check that hardware changes are grouped together
	if hwGroup, exists := groups["hardware"]; exists {
		if len(hwGroup) != 2 {
			t.Errorf("Expected 2 hardware changes, got %d", len(hwGroup))
		}
	} else {
		t.Error("Expected hardware group to exist")
	}
}

func TestDiffEngine_AnalyzeTrends(t *testing.T) {
	engine, err := NewDiffEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	diffs := []models.InventoryDiff{
		{Category: "os", Severity: "high", DiffType: "update"},
		{Category: "os", Severity: "medium", DiffType: "update"},
		{Category: "network", Severity: "critical", DiffType: "delete"},
		{Category: "hardware", Severity: "low", DiffType: "create"},
		{Category: "features", Severity: "medium", DiffType: "update"},
	}

	trends := engine.AnalyzeTrends("test-system", diffs)

	// Check required trend fields
	requiredFields := []string{
		"category_distribution",
		"severity_distribution",
		"type_distribution",
		"total_changes",
		"most_changed_category",
		"dominant_severity",
		"dominant_type",
		"change_patterns",
	}

	for _, field := range requiredFields {
		if _, exists := trends[field]; !exists {
			t.Errorf("Expected trend field '%s' to exist", field)
		}
	}

	// Check total changes
	if totalChanges, ok := trends["total_changes"].(int); !ok || totalChanges != 5 {
		t.Errorf("Expected total_changes to be 5, got %v", trends["total_changes"])
	}

	// Check category distribution
	if categoryDist, ok := trends["category_distribution"].(map[string]int); ok {
		if categoryDist["os"] != 2 {
			t.Errorf("Expected 2 OS changes, got %d", categoryDist["os"])
		}
	} else {
		t.Error("Expected category_distribution to be map[string]int")
	}

	// Check patterns
	if patterns, ok := trends["change_patterns"].(map[string]interface{}); ok {
		if criticalChanges, exists := patterns["critical_changes"]; !exists {
			t.Error("Expected critical_changes in patterns")
		} else if criticalChanges != 1 {
			t.Errorf("Expected 1 critical change, got %v", criticalChanges)
		}
	} else {
		t.Error("Expected change_patterns to be map[string]interface{}")
	}
}

func TestDiffEngine_GetEngineStats(t *testing.T) {
	engine, err := NewDiffEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	stats := engine.GetEngineStats()

	expectedKeys := []string{
		"max_depth",
		"max_diffs_per_run",
		"max_field_path_length",
		"config_load_time",
		"all_categories",
		"all_severity_levels",
		"significance_filters",
	}

	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected stat key '%s' to exist", key)
		}
	}

	// Verify numeric values are positive
	if maxDepth, ok := stats["max_depth"].(int); !ok || maxDepth <= 0 {
		t.Errorf("Expected positive max_depth, got %v", stats["max_depth"])
	}

	if maxDiffs, ok := stats["max_diffs_per_run"].(int); !ok || maxDiffs <= 0 {
		t.Errorf("Expected positive max_diffs_per_run, got %v", stats["max_diffs_per_run"])
	}
}

func TestDiffEngine_ConfigurationReload(t *testing.T) {
	engine, err := NewDiffEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	originalLoadTime := engine.GetConfigurationLoadTime()

	// Wait a bit to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	// Reload with empty path (should use default)
	err = engine.ReloadConfiguration("")
	if err != nil {
		t.Errorf("Unexpected error reloading config: %v", err)
	}

	newLoadTime := engine.GetConfigurationLoadTime()
	if !newLoadTime.After(originalLoadTime) {
		t.Error("Expected config load time to be updated after reload")
	}
}

func TestDiffEngine_ProcessingLimits(t *testing.T) {
	engine, err := NewDiffEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Test with data that would exceed maxDiffsPerRun
	// Create a large JSON structure with many differences
	previousData := `{"items": {}}`

	// Build current data with many items to exceed diff limit
	currentItems := make(map[string]interface{})
	limit := engine.maxDiffsPerRun + 100 // Exceed the limit

	for i := 0; i < limit; i++ {
		key := string(rune('a'+i%26)) + string(rune('0'+i/26))
		currentItems[key] = i
	}

	currentDataMap := map[string]interface{}{"items": currentItems}
	currentDataBytes, _ := json.Marshal(currentDataMap)
	currentData := string(currentDataBytes)

	previous := &models.InventoryRecord{
		ID:       1,
		SystemID: "limit-test",
		Data:     json.RawMessage(previousData),
	}

	current := &models.InventoryRecord{
		ID:       2,
		SystemID: "limit-test",
		Data:     json.RawMessage(currentData),
	}

	diffs, err := engine.ComputeDiff("limit-test", previous, current)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should not exceed the maximum limit
	if len(diffs) > engine.maxDiffsPerRun {
		t.Errorf("Expected at most %d diffs, got %d", engine.maxDiffsPerRun, len(diffs))
	}
}
