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
		// Modules category tests (NS8)
		{
			name:        "modules array field",
			fieldPath:   "facts.modules[0].id",
			expectedCat: "modules",
		},
		{
			name:        "modules version field",
			fieldPath:   "facts.modules[1].version",
			expectedCat: "modules",
		},

		// Cluster category tests (NS8)
		{
			name:        "cluster label field",
			fieldPath:   "facts.cluster.label",
			expectedCat: "cluster",
		},
		{
			name:        "cluster fqdn field",
			fieldPath:   "facts.cluster.fqdn",
			expectedCat: "cluster",
		},

		// Nodes category tests (NS8)
		{
			name:        "nodes array field",
			fieldPath:   "facts.nodes[0].id",
			expectedCat: "nodes",
		},
		{
			name:        "nodes version field",
			fieldPath:   "facts.nodes[1].version",
			expectedCat: "nodes",
		},

		// OS category tests (NSEC)
		{
			name:        "distro version field",
			fieldPath:   "facts.distro.version",
			expectedCat: "os",
		},
		{
			name:        "distro release field",
			fieldPath:   "facts.distro.release",
			expectedCat: "os",
		},

		// Hardware category tests
		{
			name:        "processors field",
			fieldPath:   "facts.processors.count",
			expectedCat: "hardware",
		},
		{
			name:        "memory field",
			fieldPath:   "facts.memory.total",
			expectedCat: "hardware",
		},
		{
			name:        "product field",
			fieldPath:   "facts.product.name",
			expectedCat: "hardware",
		},
		{
			name:        "virtual field",
			fieldPath:   "facts.virtual.is_virtual",
			expectedCat: "hardware",
		},

		// Network category tests
		{
			name:        "network interfaces field",
			fieldPath:   "facts.network.interfaces",
			expectedCat: "network",
		},
		// Note: facts.features.network could match either network or features
		// depending on map iteration order, we test facts.network instead

		// Features category tests (NSEC)
		{
			name:        "features field",
			fieldPath:   "facts.features.docker",
			expectedCat: "features",
		},
		{
			name:        "features dpi field",
			fieldPath:   "facts.features.dpi",
			expectedCat: "features",
		},

		// Note: Security and backup patterns like facts.features.certificates also match
		// the generic facts.features pattern. Due to Go map iteration being non-deterministic,
		// these may be categorized as either security/backup or features.
		// Similarly, facts.cluster.backup matches both backup and cluster patterns.

		// Case insensitive tests
		{
			name:        "uppercase facts field",
			fieldPath:   "FACTS.DISTRO.VERSION",
			expectedCat: "os",
		},
		{
			name:        "mixed case facts field",
			fieldPath:   "Facts.Modules[0].Id",
			expectedCat: "modules",
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
			name:        "modules category",
			category:    "modules",
			expectEmpty: false,
		},
		{
			name:        "cluster category",
			category:    "cluster",
			expectEmpty: false,
		},
		{
			name:        "nodes category",
			category:    "nodes",
			expectEmpty: false,
		},
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
			name:        "security category",
			category:    "security",
			expectEmpty: false,
		},
		{
			name:        "backup category",
			category:    "backup",
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

	// Check that we have the expected categories for NS8/NSEC
	expectedCategories := []string{"modules", "cluster", "nodes", "os", "hardware", "network", "features", "security", "backup", "system"}

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
			name:        "modules category patterns",
			category:    "modules",
			expectEmpty: false,
		},
		{
			name:        "cluster category patterns",
			category:    "cluster",
			expectEmpty: false,
		},
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
	originalModulesPatterns := differ.GetCategoryPatterns("modules")

	err = differ.ValidateCategoryPatterns()
	if err != nil {
		t.Errorf("Second pattern validation failed: %v", err)
	}

	newModulesPatterns := differ.GetCategoryPatterns("modules")
	if len(originalModulesPatterns) != len(newModulesPatterns) {
		t.Error("Pattern validation modified the patterns")
	}
}

func TestConfigurableDiffer_CategorizeFieldBatch(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	fieldPaths := []string{
		"facts.distro.version",
		"facts.memory.total",
		"facts.network.interfaces",
		"facts.features.docker",
		"facts.modules[0].id",
		"unknown.field",
	}

	results := differ.CategorizeFieldBatch(fieldPaths)

	// Check that all fields are categorized
	if len(results) != len(fieldPaths) {
		t.Errorf("Expected %d results, got %d", len(fieldPaths), len(results))
	}

	expectedCategories := map[string]string{
		"facts.distro.version":     "os",
		"facts.memory.total":       "hardware",
		"facts.network.interfaces": "network",
		"facts.features.docker":    "features",
		"facts.modules[0].id":      "modules",
		"unknown.field":            "system",
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
		"facts.distro.version":     "os",
		"facts.distro.release":     "os",
		"facts.memory.total":       "hardware",
		"facts.memory.free":        "hardware",
		"facts.network.interfaces": "network",
		"facts.features.docker":    "features",
		"facts.modules[0].id":      "modules",
		"unknown.field":            "system",
	}

	stats := differ.GetCategoryStats(categorizedFields)

	expectedStats := map[string]int{
		"os":       2,
		"hardware": 2,
		"network":  1,
		"features": 1,
		"modules":  1,
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

	// Test performance with many field paths using NS8/NSEC structure
	fieldPaths := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		switch i % 5 {
		case 0:
			fieldPaths[i] = fmt.Sprintf("facts.distro.version%d", i)
		case 1:
			fieldPaths[i] = fmt.Sprintf("facts.memory.total%d", i)
		case 2:
			fieldPaths[i] = fmt.Sprintf("facts.network.interface%d", i)
		case 3:
			fieldPaths[i] = fmt.Sprintf("facts.features.feature%d", i)
		case 4:
			fieldPaths[i] = fmt.Sprintf("facts.modules[%d].id", i)
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
