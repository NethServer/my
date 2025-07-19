/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package differ

import (
	"fmt"
	"strings"
	"testing"
)

func TestConfigurableDiffer_CategorizeField(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name        string
		fieldPath   string
		expectedCat string
	}{
		// OS category tests
		{
			name:        "OS version field",
			fieldPath:   "os.version",
			expectedCat: "os",
		},
		{
			name:        "OS release field",
			fieldPath:   "os.release",
			expectedCat: "os",
		},
		{
			name:        "kernel field",
			fieldPath:   "kernel.version",
			expectedCat: "os",
		},
		{
			name:        "kernel release field",
			fieldPath:   "kernelrelease",
			expectedCat: "os",
		},
		{
			name:        "system uptime field",
			fieldPath:   "system_uptime",
			expectedCat: "os",
		},

		// Hardware category tests
		{
			name:        "DMI field",
			fieldPath:   "dmi.system.manufacturer",
			expectedCat: "hardware",
		},
		{
			name:        "processors field",
			fieldPath:   "processors.cpu0.model",
			expectedCat: "hardware",
		},
		{
			name:        "memory field",
			fieldPath:   "memory.total",
			expectedCat: "hardware",
		},
		{
			name:        "mountpoints field",
			fieldPath:   "mountpoints.root.size",
			expectedCat: "hardware",
		},

		// Network category tests
		{
			name:        "networking field",
			fieldPath:   "networking.hostname",
			expectedCat: "network",
		},
		{
			name:        "public IP field",
			fieldPath:   "public_ip",
			expectedCat: "network",
		},
		{
			name:        "ARP MACs field",
			fieldPath:   "arp_macs.gateway",
			expectedCat: "network",
		},
		{
			name:        "esmithdb networks field",
			fieldPath:   "esmithdb.networks.eth0",
			expectedCat: "network",
		},

		// Features category tests
		{
			name:        "features field",
			fieldPath:   "features.docker",
			expectedCat: "features",
		},
		{
			name:        "services field",
			fieldPath:   "services.nginx.status",
			expectedCat: "features",
		},
		{
			name:        "esmithdb configuration field",
			fieldPath:   "esmithdb.configuration.httpd",
			expectedCat: "features",
		},

		// Case insensitive tests
		{
			name:        "uppercase OS field",
			fieldPath:   "OS.VERSION",
			expectedCat: "os",
		},
		{
			name:        "mixed case networking field",
			fieldPath:   "NetWorking.HostName",
			expectedCat: "network",
		},

		// Default category tests
		{
			name:        "unknown field",
			fieldPath:   "unknown.field",
			expectedCat: "system", // Should use default
		},
		{
			name:        "random field",
			fieldPath:   "random.data.value",
			expectedCat: "system", // Should use default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category := differ.CategorizeField(tt.fieldPath)
			if category != tt.expectedCat {
				t.Errorf("Expected category %s, got %s for field %s", tt.expectedCat, category, tt.fieldPath)
			}
		})
	}
}

func TestConfigurableDiffer_GetCategoryDescription(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name        string
		category    string
		expectEmpty bool
	}{
		{
			name:        "OS category",
			category:    "os",
			expectEmpty: false,
		},
		{
			name:        "hardware category",
			category:    "hardware",
			expectEmpty: false,
		},
		{
			name:        "network category",
			category:    "network",
			expectEmpty: false,
		},
		{
			name:        "features category",
			category:    "features",
			expectEmpty: false,
		},
		{
			name:        "default category",
			category:    "system",
			expectEmpty: false,
		},
		{
			name:        "unknown category",
			category:    "nonexistent",
			expectEmpty: false, // Should return "Unknown category"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description := differ.GetCategoryDescription(tt.category)

			if tt.expectEmpty && description != "" {
				t.Errorf("Expected empty description for category %s, got: %s", tt.category, description)
			}

			if !tt.expectEmpty && description == "" {
				t.Errorf("Expected non-empty description for category %s", tt.category)
			}

			// For unknown category, should return specific message
			if tt.category == "nonexistent" && description != "Unknown category" {
				t.Errorf("Expected 'Unknown category' for nonexistent category, got: %s", description)
			}
		})
	}
}

func TestConfigurableDiffer_GetAllCategories(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	categories := differ.GetAllCategories()

	// Check that we have at least the expected categories
	expectedCategories := []string{"os", "hardware", "network", "features", "system"}

	for _, expected := range expectedCategories {
		if description, exists := categories[expected]; !exists {
			t.Errorf("Expected category '%s' to be in results", expected)
		} else if description == "" {
			t.Errorf("Expected non-empty description for category '%s'", expected)
		}
	}

	// Check that all returned categories have descriptions
	for category, description := range categories {
		if description == "" {
			t.Errorf("Category '%s' has empty description", category)
		}
	}
}

func TestConfigurableDiffer_GetCategoryPatterns(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name        string
		category    string
		expectEmpty bool
		expectCount int
	}{
		{
			name:        "OS category patterns",
			category:    "os",
			expectEmpty: false,
		},
		{
			name:        "hardware category patterns",
			category:    "hardware",
			expectEmpty: false,
		},
		{
			name:        "network category patterns",
			category:    "network",
			expectEmpty: false,
		},
		{
			name:        "features category patterns",
			category:    "features",
			expectEmpty: false,
		},
		{
			name:        "nonexistent category patterns",
			category:    "nonexistent",
			expectEmpty: true,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns := differ.GetCategoryPatterns(tt.category)

			if tt.expectEmpty {
				if len(patterns) != tt.expectCount {
					t.Errorf("Expected %d patterns for category %s, got %d", tt.expectCount, tt.category, len(patterns))
				}
			} else {
				if len(patterns) == 0 {
					t.Errorf("Expected non-empty patterns for category %s", tt.category)
				}

				// Check that all patterns are valid strings
				for i, pattern := range patterns {
					if pattern == "" {
						t.Errorf("Pattern %d for category %s is empty", i, tt.category)
					}
				}
			}
		})
	}
}

func TestConfigurableDiffer_ValidateCategoryPatterns(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	err = differ.ValidateCategoryPatterns()
	if err != nil {
		t.Errorf("Pattern validation failed: %v", err)
	}

	// Test that validation doesn't modify the patterns
	originalOSPatterns := differ.GetCategoryPatterns("os")

	err = differ.ValidateCategoryPatterns()
	if err != nil {
		t.Errorf("Second pattern validation failed: %v", err)
	}

	newOSPatterns := differ.GetCategoryPatterns("os")
	if len(originalOSPatterns) != len(newOSPatterns) {
		t.Error("Pattern validation modified the patterns")
	}
}

func TestConfigurableDiffer_CategorizeFieldBatch(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	fieldPaths := []string{
		"os.version",
		"memory.total",
		"networking.hostname",
		"features.docker",
		"unknown.field",
	}

	results := differ.CategorizeFieldBatch(fieldPaths)

	// Check that all fields are categorized
	if len(results) != len(fieldPaths) {
		t.Errorf("Expected %d results, got %d", len(fieldPaths), len(results))
	}

	expectedCategories := map[string]string{
		"os.version":          "os",
		"memory.total":        "hardware",
		"networking.hostname": "network",
		"features.docker":     "features",
		"unknown.field":       "system",
	}

	for fieldPath, expectedCategory := range expectedCategories {
		if category, exists := results[fieldPath]; !exists {
			t.Errorf("Expected result for field %s", fieldPath)
		} else if category != expectedCategory {
			t.Errorf("Expected category %s for field %s, got %s", expectedCategory, fieldPath, category)
		}
	}

	// Test empty input
	emptyResults := differ.CategorizeFieldBatch([]string{})
	if len(emptyResults) != 0 {
		t.Errorf("Expected empty results for empty input, got %d results", len(emptyResults))
	}
}

func TestConfigurableDiffer_GetCategoryStats(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	categorizedFields := map[string]string{
		"os.version":          "os",
		"os.kernel":           "os",
		"memory.total":        "hardware",
		"memory.free":         "hardware",
		"networking.hostname": "network",
		"features.docker":     "features",
		"unknown.field":       "system",
	}

	stats := differ.GetCategoryStats(categorizedFields)

	expectedStats := map[string]int{
		"os":       2,
		"hardware": 2,
		"network":  1,
		"features": 1,
		"system":   1,
	}

	for category, expectedCount := range expectedStats {
		if count, exists := stats[category]; !exists {
			t.Errorf("Expected stat for category %s", category)
		} else if count != expectedCount {
			t.Errorf("Expected %d occurrences for category %s, got %d", expectedCount, category, count)
		}
	}

	// Test empty input
	emptyStats := differ.GetCategoryStats(map[string]string{})
	if len(emptyStats) != 0 {
		t.Errorf("Expected empty stats for empty input, got %d stats", len(emptyStats))
	}
}

func TestConfigurableDiffer_CategorizeField_EdgeCases(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name        string
		fieldPath   string
		expectValid bool
	}{
		{
			name:        "empty field path",
			fieldPath:   "",
			expectValid: true, // Should return default category
		},
		{
			name:        "single character field",
			fieldPath:   "a",
			expectValid: true,
		},
		{
			name:        "very long field path",
			fieldPath:   strings.Repeat("very.long.field.path.", 20),
			expectValid: true,
		},
		{
			name:        "field with special characters",
			fieldPath:   "field-with_special.chars@123",
			expectValid: true,
		},
		{
			name:        "field with unicode characters",
			fieldPath:   "field.with.ūníćödė",
			expectValid: true,
		},
		{
			name:        "field with numbers only",
			fieldPath:   "123.456.789",
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category := differ.CategorizeField(tt.fieldPath)

			if tt.expectValid {
				if category == "" {
					t.Errorf("Expected non-empty category for field %s", tt.fieldPath)
				}
			} else {
				if category != "" {
					t.Errorf("Expected empty category for invalid field %s, got %s", tt.fieldPath, category)
				}
			}
		})
	}
}

func TestConfigurableDiffer_CategorizeField_Performance(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	// Test performance with many field paths (with unique variations)
	fieldPaths := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		switch i % 4 {
		case 0:
			fieldPaths[i] = fmt.Sprintf("os.version%d", i)
		case 1:
			fieldPaths[i] = fmt.Sprintf("memory.total%d", i)
		case 2:
			fieldPaths[i] = fmt.Sprintf("networking.hostname%d", i)
		case 3:
			fieldPaths[i] = fmt.Sprintf("features.docker%d", i)
		}
	}

	// Measure categorization performance
	for _, fieldPath := range fieldPaths {
		category := differ.CategorizeField(fieldPath)
		if category == "" {
			t.Errorf("Got empty category for field %s", fieldPath)
		}
	}

	// Test batch performance
	results := differ.CategorizeFieldBatch(fieldPaths)
	if len(results) != len(fieldPaths) {
		t.Errorf("Expected %d batch results, got %d", len(fieldPaths), len(results))
	}
}
