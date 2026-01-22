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
// Uses NS8/NSEC inventory structure
func TestDifferIntegration(t *testing.T) {
	// Step 1: Create a diff engine with default configuration
	engine, err := NewDiffEngine("")
	if err != nil {
		t.Fatalf("Failed to create diff engine: %v", err)
	}

	// Step 2: Create test inventory data representing a real NS8 system
	previousInventory := `{
		"installation": "nethserver",
		"facts": {
			"cluster": {
				"label": "production-cluster",
				"fqdn": "cluster.example.com",
				"public_ip": "203.0.113.10",
				"subscription": "active"
			},
			"nodes": [
				{
					"id": 1,
					"version": "8.2.0",
					"label": "node1"
				}
			],
			"modules": [
				{
					"id": "dokuwiki1",
					"name": "dokuwiki",
					"version": "1.0.0",
					"node": 1,
					"label": "Wiki"
				},
				{
					"id": "nextcloud1",
					"name": "nextcloud",
					"version": "25.0.0",
					"node": 1,
					"label": "Cloud"
				}
			],
			"distro": {
				"version": "8.2.0",
				"release": "ns8"
			},
			"processors": {
				"count": 4,
				"model": "Intel Core i7"
			},
			"memory": {
				"total": 16777216,
				"available": 12884901,
				"used_bytes": 3892315
			},
			"features": {
				"docker": true,
				"traefik": true,
				"certificates": {
					"count": 5
				}
			}
		},
		"system_uptime": 1640995200,
		"metrics": {
			"timestamp": "2023-01-01T10:00:00Z"
		}
	}`

	currentInventory := `{
		"installation": "nethserver",
		"facts": {
			"cluster": {
				"label": "production-cluster-v2",
				"fqdn": "cluster.example.com",
				"public_ip": "203.0.113.20",
				"subscription": "active"
			},
			"nodes": [
				{
					"id": 1,
					"version": "8.3.0",
					"label": "node1"
				},
				{
					"id": 2,
					"version": "8.3.0",
					"label": "node2"
				}
			],
			"modules": [
				{
					"id": "dokuwiki1",
					"name": "dokuwiki",
					"version": "1.1.0",
					"node": 1,
					"label": "Wiki"
				},
				{
					"id": "nextcloud1",
					"name": "nextcloud",
					"version": "26.0.0",
					"node": 1,
					"label": "Cloud Storage"
				},
				{
					"id": "mattermost1",
					"name": "mattermost",
					"version": "7.0.0",
					"node": 2,
					"label": "Chat"
				}
			],
			"distro": {
				"version": "8.3.0",
				"release": "ns8"
			},
			"processors": {
				"count": 8,
				"model": "Intel Core i9"
			},
			"memory": {
				"total": 33554432,
				"available": 25769803,
				"used_bytes": 7784629
			},
			"features": {
				"docker": true,
				"traefik": true,
				"certificates": {
					"count": 10
				}
			}
		},
		"system_uptime": 1640995500,
		"metrics": {
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

	// Step 6: Track changes by category
	categoryCount := make(map[string]int)
	severityCount := make(map[string]int)
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

		categoryCount[diff.Category]++
		severityCount[diff.Severity]++

		// Check that significant changes are not filtered
		if strings.Contains(diff.FieldPath, "facts.") {
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

	// Step 7: Verify that we have facts-related changes
	if significantChangesCount == 0 {
		t.Error("Expected to find significant facts.* changes")
	}

	// Step 8: Test grouping functionality
	groups := engine.GroupRelatedChanges(diffs)

	// Check that we have groups
	if len(groups) == 0 {
		t.Error("Expected to find change groups")
	}

	t.Logf("Found %d groups: %v", len(groups), getGroupKeys(groups))

	// Check for expected NS8 groups
	hasExpectedGroup := false
	for groupName := range groups {
		if strings.HasPrefix(groupName, "facts") ||
			groupName == "modules" ||
			groupName == "cluster" ||
			groupName == "nodes" ||
			groupName == "hardware" ||
			groupName == "operating_system" ||
			strings.HasPrefix(groupName, "features") {
			hasExpectedGroup = true
			break
		}
	}
	if !hasExpectedGroup {
		t.Error("Expected at least one NS8-related group (facts, modules, cluster, nodes, hardware, or features)")
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

	// Step 11: Test health score calculation
	healthScore := CalculateInventoryHealth(diffs)
	t.Logf("Inventory health score: %.2f", healthScore)

	// Score should be between 0 and 100
	if healthScore < 0 || healthScore > 100 {
		t.Errorf("Expected health score between 0-100, got %.2f", healthScore)
	}

	// Step 12: Test anomaly detection
	anomalies := DetectAnomalies(diffs)
	t.Logf("Detected %d anomalies", len(anomalies))

	// Verify anomaly structure if any were found
	for _, anomaly := range anomalies {
		// Anomalies are inventory diffs with high severity
		if anomaly.FieldPath == "" {
			t.Error("Anomaly field path should not be empty")
		}
		if anomaly.Severity == "" {
			t.Error("Anomaly severity should not be empty")
		}
	}

	// Step 13: Final summary
	t.Logf("Integration test completed successfully:")
	t.Logf("- Processed %d total differences", len(diffs))
	t.Logf("- Found %d significant changes", significantChangesCount)
	t.Logf("- Created %d change groups", len(groups))
	t.Logf("- Calculated health score: %.2f", healthScore)
	t.Logf("- Detected %d anomalies", len(anomalies))
}

// getGroupKeys returns the keys from a map as a slice
func getGroupKeys(groups map[string][]models.InventoryDiff) []string {
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	return keys
}

// TestDifferIntegrationNSEC tests the complete diff workflow for NSEC systems
func TestDifferIntegrationNSEC(t *testing.T) {
	// Create a diff engine with default configuration
	engine, err := NewDiffEngine("")
	if err != nil {
		t.Fatalf("Failed to create diff engine: %v", err)
	}

	// Create test inventory data representing a real NSEC system
	previousInventory := `{
		"installation": "nethsecurity",
		"facts": {
			"distro": {
				"version": "23.05.4",
				"release": "nsec",
				"architecture": "x86_64"
			},
			"features": {
				"firewall": {
					"enabled": true,
					"rules": 50
				},
				"openvpn": {
					"enabled": true,
					"tunnels": 3
				},
				"certificates": {
					"count": 5
				},
				"docker": false
			},
			"memory": {
				"total": 8589934592,
				"available": 6442450944
			}
		}
	}`

	currentInventory := `{
		"installation": "nethsecurity",
		"facts": {
			"distro": {
				"version": "24.10",
				"release": "nsec",
				"architecture": "x86_64"
			},
			"features": {
				"firewall": {
					"enabled": true,
					"rules": 65
				},
				"openvpn": {
					"enabled": true,
					"tunnels": 5
				},
				"wireguard": {
					"enabled": true,
					"peers": 10
				},
				"certificates": {
					"count": 8
				},
				"docker": true
			},
			"memory": {
				"total": 8589934592,
				"available": 5368709120
			}
		}
	}`

	// Create inventory records
	previousRecord := &models.InventoryRecord{
		ID:        1,
		SystemID:  "nsec-test-system",
		Data:      json.RawMessage(previousInventory),
		Timestamp: time.Now().Add(-time.Hour),
	}

	currentRecord := &models.InventoryRecord{
		ID:        2,
		SystemID:  "nsec-test-system",
		Data:      json.RawMessage(currentInventory),
		Timestamp: time.Now(),
	}

	// Compute differences
	diffs, err := engine.ComputeDiff("nsec-test-system", previousRecord, currentRecord)
	if err != nil {
		t.Fatalf("Failed to compute diffs: %v", err)
	}

	// Validate that differences were found
	if len(diffs) == 0 {
		t.Fatal("Expected to find differences between the NSEC inventories")
	}

	t.Logf("Found %d NSEC differences", len(diffs))

	// Track changes
	for _, diff := range diffs {
		t.Logf("NSEC diff: %s (%s, %s, %s)", diff.FieldPath, diff.DiffType, diff.Category, diff.Severity)

		// Validate structure
		if diff.SystemID != "nsec-test-system" {
			t.Errorf("Expected SystemID 'nsec-test-system', got '%s'", diff.SystemID)
		}
	}

	t.Logf("NSEC integration test completed with %d differences", len(diffs))
}
