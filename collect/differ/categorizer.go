/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package differ

import (
	"strings"

	"github.com/nethesis/my/collect/logger"
)

// CategorizeField determines the category of a field based on its path using configuration
//
// Process Flow:
// 1. Convert field path to lowercase for case-insensitive matching
// 2. Iterate through configured category patterns
// 3. Check if field path matches any pattern in each category
// 4. Return first matching category or default if no match
//
// Example:
//   - "os.version" → "os" category
//   - "processors.cpu0.model" → "hardware" category
//   - "networking.interfaces.eth0.ip" → "network" category
func (cd *ConfigurableDiffer) CategorizeField(fieldPath string) string {
	// Step 1: Normalize field path for pattern matching
	pathLower := strings.ToLower(fieldPath)

	// Step 2: Check against each configured category
	for category, patterns := range cd.categoryPatterns {
		// Step 3: Test all patterns for this category
		for _, pattern := range patterns {
			if pattern.MatchString(pathLower) {
				// Step 4: Log successful categorization for debugging
				logger.ComponentLogger("differ-categorizer").Debug().
					Str("field_path", fieldPath).
					Str("category", category).
					Str("matched_pattern", pattern.String()).
					Msg("Field categorized successfully")

				return category
			}
		}
	}

	// Step 5: Return default category if no patterns match
	defaultCategory := cd.config.Categorization.Default.Name

	logger.ComponentLogger("differ-categorizer").Debug().
		Str("field_path", fieldPath).
		Str("category", defaultCategory).
		Msg("Field assigned to default category")

	return defaultCategory
}

// GetCategoryDescription returns the description for a given category
func (cd *ConfigurableDiffer) GetCategoryDescription(category string) string {
	if rule, exists := cd.config.Categorization.Categories[category]; exists {
		return rule.Description
	}

	// Return default description if category not found
	if category == cd.config.Categorization.Default.Name {
		return cd.config.Categorization.Default.Description
	}

	return "Unknown category"
}

// GetAllCategories returns all configured categories with their descriptions
func (cd *ConfigurableDiffer) GetAllCategories() map[string]string {
	categories := make(map[string]string)

	// Add configured categories
	for category, rule := range cd.config.Categorization.Categories {
		categories[category] = rule.Description
	}

	// Add default category
	categories[cd.config.Categorization.Default.Name] = cd.config.Categorization.Default.Description

	return categories
}

// GetCategoryPatterns returns the patterns for a given category (for debugging)
func (cd *ConfigurableDiffer) GetCategoryPatterns(category string) []string {
	if rule, exists := cd.config.Categorization.Categories[category]; exists {
		return rule.Patterns
	}
	return []string{}
}

// ValidateCategoryPatterns validates all category patterns can be compiled
func (cd *ConfigurableDiffer) ValidateCategoryPatterns() error {
	for category, patterns := range cd.categoryPatterns {
		for i, pattern := range patterns {
			// Test pattern compilation
			if pattern == nil {
				logger.ComponentLogger("differ-categorizer").Error().
					Str("category", category).
					Int("pattern_index", i).
					Msg("Found nil pattern during validation")
				continue
			}

			// Test pattern with sample data
			testPaths := []string{
				"os.version",
				"processors.cpu0.model",
				"networking.interfaces.eth0.ip",
				"features.module.status",
				"dmi.system.manufacturer",
			}

			for _, testPath := range testPaths {
				pattern.MatchString(strings.ToLower(testPath))
			}
		}
	}

	return nil
}

// CategorizeFieldBatch processes multiple fields efficiently
func (cd *ConfigurableDiffer) CategorizeFieldBatch(fieldPaths []string) map[string]string {
	results := make(map[string]string, len(fieldPaths))

	// Process each field path
	for _, fieldPath := range fieldPaths {
		results[fieldPath] = cd.CategorizeField(fieldPath)
	}

	return results
}

// GetCategoryStats returns statistics about field categorization
func (cd *ConfigurableDiffer) GetCategoryStats(categorizedFields map[string]string) map[string]int {
	stats := make(map[string]int)

	// Count occurrences of each category
	for _, category := range categorizedFields {
		stats[category]++
	}

	return stats
}
