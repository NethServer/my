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

// TestDifferIntegration tests the complete diff workflow from start to finish
func TestDifferIntegration(t *testing.T) {
	// Step 1: Create a diff engine with default configuration
	engine, err := NewDiffEngine("")
	if err != nil {
		t.Fatalf("Failed to create diff engine: %v", err)
	}

	// Step 2: Create test inventory data representing a real system
	previousInventory := `{
		"os": {
			"version": "Ubuntu 20.04.3 LTS",
			"kernel": "5.4.0-80-generic",
			"architecture": "x86_64"
		},
		"networking": {
			"hostname": "web-server-01",
			"public_ip": "203.0.113.10",
			"interfaces": {
				"eth0": {
					"ip": "192.168.1.100",
					"mac": "00:1B:44:11:3A:B7"
				}
			}
		},
		"processors": {
			"count": 4,
			"model": "Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz",
			"cores": 6
		},
		"memory": {
			"total": 16777216,
			"available": 12884901
		},
		"features": {
			"docker": {
				"installed": true,
				"version": "20.10.8"
			},
			"nginx": {
				"installed": true,
				"version": "1.18.0"
			}
		},
		"esmithdb": {
			"configuration": {
				"httpd": "enabled",
				"ssh": "enabled"
			}
		},
		"system_uptime": 1640995200,
		"metrics": {
			"cpu_usage": 45.2,
			"memory_usage": 68.5,
			"disk_usage": 32.1,
			"timestamp": "2023-01-01T10:00:00Z"
		}
	}`

	currentInventory := `{
		"os": {
			"version": "Ubuntu 22.04.1 LTS",
			"kernel": "5.15.0-40-generic",
			"architecture": "x86_64"
		},
		"networking": {
			"hostname": "web-server-prod",
			"public_ip": "203.0.113.20",
			"interfaces": {
				"eth0": {
					"ip": "192.168.1.101",
					"mac": "00:1B:44:11:3A:B7"
				}
			}
		},
		"processors": {
			"count": 8,
			"model": "Intel(R) Core(TM) i9-9900K CPU @ 3.60GHz",
			"cores": 8
		},
		"memory": {
			"total": 33554432,
			"available": 25769803
		},
		"features": {
			"docker": {
				"installed": true,
				"version": "20.10.17"
			},
			"nginx": {
				"installed": true,
				"version": "1.22.0"
			},
			"redis": {
				"installed": true,
				"version": "6.2.7"
			}
		},
		"esmithdb": {
			"configuration": {
				"httpd": "enabled",
				"ssh": "disabled",
				"redis": "enabled"
			}
		},
		"system_uptime": 1640995500,
		"metrics": {
			"cpu_usage": 47.8,
			"memory_usage": 71.2,
			"disk_usage": 34.5,
			"timestamp": "2023-01-01T10:05:00Z"
		}
	}`

	// Step 3: Create inventory records
	previousRecord := &models.InventoryRecord{
		ID:        1,
		SystemID:  "integration-test-system",
		Data:      json.RawMessage(previousInventory),
		Timestamp: time.Now().Add(-time.Hour),
	}

	currentRecord := &models.InventoryRecord{
		ID:        2,
		SystemID:  "integration-test-system",
		Data:      json.RawMessage(currentInventory),
		Timestamp: time.Now(),
	}

	// Step 4: Compute differences
	diffs, err := engine.ComputeDiff("integration-test-system", previousRecord, currentRecord)
	if err != nil {
		t.Fatalf("Failed to compute diffs: %v", err)
	}

	// Step 5: Validate that differences were found
	if len(diffs) == 0 {
		t.Fatal("Expected to find differences between the inventories")
	}

	t.Logf("Found %d differences", len(diffs))

	// Step 6: Verify specific expected changes
	expectedChanges := map[string]struct {
		changeType string
		category   string
		severity   string
	}{
		"os.version":                    {"update", "os", "high"},
		"os.kernel":                     {"update", "os", "high"},
		"networking.hostname":           {"update", "network", "medium"},
		"networking.public_ip":          {"update", "network", "high"},
		"networking.interfaces.eth0.ip": {"update", "network", "medium"},
		"processors.count":              {"update", "hardware", "medium"},
		"processors.model":              {"update", "hardware", "medium"},
		"processors.cores":              {"update", "hardware", "medium"},
		"memory.total":                  {"update", "hardware", "medium"},
		"memory.available":              {"update", "hardware", "medium"},
		"features.docker.version":       {"update", "features", "medium"},
		"features.nginx.version":        {"update", "features", "medium"},
		"features.redis":                {"create", "features", "medium"},
	}

	// Track which expected changes we found
	foundChanges := make(map[string]bool)
	significantChangesCount := 0

	for _, diff := range diffs {
		t.Logf("Found diff: %s (%s, %s, %s)", diff.FieldPath, diff.DiffType, diff.Category, diff.Severity)

		// Validate diff structure
		if diff.SystemID != "integration-test-system" {
			t.Errorf("Expected SystemID 'integration-test-system', got '%s'", diff.SystemID)
		}

		if diff.FieldPath == "" {
			t.Error("FieldPath should not be empty")
		}

		if diff.DiffType == "" {
			t.Error("DiffType should not be empty")
		}

		if diff.Category == "" {
			t.Error("Category should not be empty")
		}

		if diff.Severity == "" {
			t.Error("Severity should not be empty")
		}

		// Check if this is an expected change
		if expected, exists := expectedChanges[diff.FieldPath]; exists {
			foundChanges[diff.FieldPath] = true

			if diff.DiffType != expected.changeType {
				t.Errorf("Expected change type %s for %s, got %s", expected.changeType, diff.FieldPath, diff.DiffType)
			}

			if diff.Category != expected.category {
				t.Errorf("Expected category %s for %s, got %s", expected.category, diff.FieldPath, diff.Category)
			}

			if diff.Severity != expected.severity {
				t.Errorf("Expected severity %s for %s, got %s", expected.severity, diff.FieldPath, diff.Severity)
			}

			significantChangesCount++
		}

		// Validate that non-significant changes are filtered out
		if diff.FieldPath == "system_uptime" {
			t.Error("system_uptime should be filtered out as non-significant")
		}

		if diff.FieldPath == "metrics.timestamp" {
			t.Error("metrics.timestamp should be filtered out as non-significant")
		}
	}

	// Step 7: Verify that key changes were detected
	criticalChanges := []string{"os.version", "os.kernel", "networking.public_ip"}
	for _, criticalChange := range criticalChanges {
		if !foundChanges[criticalChange] {
			t.Errorf("Expected to find critical change: %s", criticalChange)
		}
	}

	// Step 8: Test grouping functionality
	groups := engine.GroupRelatedChanges(diffs)

	// Check that we have groups (the exact names depend on the actual field paths found)
	if len(groups) == 0 {
		t.Error("Expected to find change groups")
	}

	// Verify that operating_system and network groups exist (these are consistent)
	expectedConsistentGroups := []string{"operating_system", "network", "hardware"}
	for _, expectedGroup := range expectedConsistentGroups {
		if _, exists := groups[expectedGroup]; !exists {
			t.Errorf("Expected group '%s' to exist", expectedGroup)
		}
	}

	// For features, check that at least one features group exists
	hasFeatureGroup := false
	for groupName := range groups {
		if strings.HasPrefix(groupName, "features") {
			hasFeatureGroup = true
			break
		}
	}
	if !hasFeatureGroup {
		t.Error("Expected at least one features group to exist")
	}

	// Step 9: Test trend analysis
	trends := engine.AnalyzeTrends("integration-test-system", diffs)

	requiredTrendFields := []string{
		"category_distribution",
		"severity_distribution",
		"type_distribution",
		"total_changes",
		"change_patterns",
	}

	for _, field := range requiredTrendFields {
		if _, exists := trends[field]; !exists {
			t.Errorf("Expected trend field '%s' to exist", field)
		}
	}

	// Step 10: Test utility functions
	summary := FormatDiffSummary(diffs)
	if summary == "" {
		t.Error("Expected non-empty diff summary")
	}
	t.Logf("Diff summary: %s", summary)

	// Test metrics calculation
	metrics := GetDiffMetrics(diffs)
	if totalChanges, ok := metrics["total_changes"].(int); !ok || totalChanges != len(diffs) {
		t.Errorf("Expected total_changes to be %d, got %v", len(diffs), metrics["total_changes"])
	}

	// Test health score calculation
	healthScore := CalculateInventoryHealth(diffs)
	if healthScore < 0 || healthScore > 100 {
		t.Errorf("Expected health score between 0-100, got %.2f", healthScore)
	}
	t.Logf("Inventory health score: %.2f", healthScore)

	// Test anomaly detection
	anomalies := DetectAnomalies(diffs)
	t.Logf("Detected %d anomalies", len(anomalies))

	// Step 11: Test configuration access
	config := engine.GetConfiguration()
	if config == nil {
		t.Error("Expected non-nil configuration")
	}

	loadTime := engine.GetConfigurationLoadTime()
	if loadTime.IsZero() {
		t.Error("Expected non-zero configuration load time")
	}

	// Step 12: Test engine statistics
	stats := engine.GetEngineStats()
	expectedStatKeys := []string{
		"max_depth",
		"max_diffs_per_run",
		"max_field_path_length",
		"config_load_time",
		"all_categories",
		"all_severity_levels",
		"significance_filters",
	}

	for _, key := range expectedStatKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected stat key '%s' to exist", key)
		}
	}

	t.Logf("Integration test completed successfully:")
	t.Logf("- Processed %d total differences", len(diffs))
	t.Logf("- Found %d significant changes", significantChangesCount)
	t.Logf("- Created %d change groups", len(groups))
	t.Logf("- Calculated health score: %.2f", healthScore)
	t.Logf("- Detected %d anomalies", len(anomalies))
}

// TestDifferPerformance tests the performance of the differ with large datasets
func TestDifferPerformance(t *testing.T) {
	engine, err := NewDiffEngine("")
	if err != nil {
		t.Fatalf("Failed to create diff engine: %v", err)
	}

	// Create a large inventory with many fields
	largeInventory := map[string]interface{}{
		"os": map[string]interface{}{
			"version":      "Ubuntu 20.04.3 LTS",
			"kernel":       "5.4.0-80-generic",
			"architecture": "x86_64",
		},
		"networking": map[string]interface{}{
			"hostname":  "test-server",
			"public_ip": "203.0.113.10",
		},
	}

	// Add many dynamic fields to test performance
	for i := 0; i < 100; i++ {
		key := string(rune('a'+i%26)) + string(rune('0'+i/26))
		largeInventory[key] = map[string]interface{}{
			"value1": i,
			"value2": i * 2,
			"value3": string(rune('A' + i%26)),
		}
	}

	previousData, _ := json.Marshal(largeInventory)

	// Modify some values for the current inventory
	for _, value := range largeInventory {
		if valueMap, ok := value.(map[string]interface{}); ok {
			for subKey, subValue := range valueMap {
				if intVal, ok := subValue.(int); ok {
					valueMap[subKey] = intVal + 1
				}
			}
		}
	}

	currentData, _ := json.Marshal(largeInventory)

	previousRecord := &models.InventoryRecord{
		ID:       1,
		SystemID: "performance-test",
		Data:     previousData,
	}

	currentRecord := &models.InventoryRecord{
		ID:       2,
		SystemID: "performance-test",
		Data:     currentData,
	}

	// Measure performance
	start := time.Now()
	diffs, err := engine.ComputeDiff("performance-test", previousRecord, currentRecord)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to compute diffs: %v", err)
	}

	t.Logf("Performance test completed in %v", duration)
	t.Logf("Processed large inventory with %d differences", len(diffs))

	// Verify we didn't hit the limit (should be way under)
	if len(diffs) >= engine.maxDiffsPerRun {
		t.Errorf("Hit maximum diffs limit, may indicate performance issue")
	}

	// Performance benchmark - should complete within reasonable time
	if duration > 5*time.Second {
		t.Errorf("Diff computation took too long: %v", duration)
	}
}

// TestDifferErrorHandling tests error handling in various scenarios
func TestDifferErrorHandling(t *testing.T) {
	engine, err := NewDiffEngine("")
	if err != nil {
		t.Fatalf("Failed to create diff engine: %v", err)
	}

	tests := []struct {
		name         string
		previousData string
		currentData  string
		expectError  bool
	}{
		{
			name:         "invalid JSON in previous",
			previousData: `{invalid json`,
			currentData:  `{"valid": "json"}`,
			expectError:  true,
		},
		{
			name:         "invalid JSON in current",
			previousData: `{"valid": "json"}`,
			currentData:  `{invalid json`,
			expectError:  true,
		},
		{
			name:         "empty objects",
			previousData: `{}`,
			currentData:  `{}`,
			expectError:  false,
		},
		{
			name:         "null values",
			previousData: `{"field": null}`,
			currentData:  `{"field": "value"}`,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previousRecord := &models.InventoryRecord{
				ID:       1,
				SystemID: "error-test",
				Data:     json.RawMessage(tt.previousData),
			}

			currentRecord := &models.InventoryRecord{
				ID:       2,
				SystemID: "error-test",
				Data:     json.RawMessage(tt.currentData),
			}

			_, err := engine.ComputeDiff("error-test", previousRecord, currentRecord)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
