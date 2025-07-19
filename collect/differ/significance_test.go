/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package differ

import (
	"strings"
	"testing"
)

func TestConfigurableDiffer_IsSignificantChange(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name       string
		fieldPath  string
		changeType string
		category   string
		severity   string
		from       interface{}
		to         interface{}
		expected   bool
	}{
		// Always significant tests
		{
			name:       "always significant high severity",
			fieldPath:  "os.version",
			changeType: "update",
			category:   "os",
			severity:   "high",
			from:       "20.04",
			to:         "22.04",
			expected:   true,
		},
		{
			name:       "always significant critical severity",
			fieldPath:  "system.error",
			changeType: "create",
			category:   "system",
			severity:   "critical",
			from:       nil,
			to:         "error",
			expected:   true,
		},
		{
			name:       "always significant hardware category",
			fieldPath:  "memory.total",
			changeType: "update",
			category:   "hardware",
			severity:   "medium",
			from:       8192,
			to:         16384,
			expected:   true,
		},
		{
			name:       "always significant network category",
			fieldPath:  "networking.hostname",
			changeType: "update",
			category:   "network",
			severity:   "medium",
			from:       "old-host",
			to:         "new-host",
			expected:   true,
		},
		{
			name:       "always significant delete change",
			fieldPath:  "some.field",
			changeType: "delete",
			category:   "system",
			severity:   "medium",
			from:       "value",
			to:         nil,
			expected:   true,
		},

		// Never significant tests
		{
			name:       "never significant system uptime",
			fieldPath:  "system_uptime",
			changeType: "update",
			category:   "system",
			severity:   "low",
			from:       100,
			to:         200,
			expected:   false,
		},
		{
			name:       "never significant metrics timestamp",
			fieldPath:  "metrics.timestamp",
			changeType: "update",
			category:   "system",
			severity:   "low",
			from:       "2023-01-01",
			to:         "2023-01-02",
			expected:   false,
		},
		{
			name:       "never significant performance last update",
			fieldPath:  "performance.last_update",
			changeType: "update",
			category:   "system",
			severity:   "low",
			from:       1640995200,
			to:         1640995260,
			expected:   false,
		},
		{
			name:       "never significant monitoring heartbeat",
			fieldPath:  "monitoring.heartbeat",
			changeType: "update",
			category:   "system",
			severity:   "low",
			from:       1640995200,
			to:         1640995260,
			expected:   false,
		},

		// Time filtered tests
		{
			name:       "time filtered metrics field",
			fieldPath:  "metrics.cpu_usage",
			changeType: "update",
			category:   "system",
			severity:   "low",
			from:       50.0,
			to:         55.0,
			expected:   false,
		},
		{
			name:       "time filtered performance field",
			fieldPath:  "performance.memory_usage",
			changeType: "update",
			category:   "system",
			severity:   "low",
			from:       "60%",
			to:         "65%",
			expected:   false,
		},
		{
			name:       "time filtered monitoring field",
			fieldPath:  "monitoring.status",
			changeType: "update",
			category:   "system",
			severity:   "low",
			from:       "ok",
			to:         "warning",
			expected:   false,
		},

		// Default significance tests
		{
			name:       "default significant change",
			fieldPath:  "unknown.field",
			changeType: "update",
			category:   "system",
			severity:   "medium",
			from:       "old",
			to:         "new",
			expected:   true, // Default is significant=true
		},
		{
			name:       "normal configuration change",
			fieldPath:  "configuration.service",
			changeType: "update",
			category:   "features",
			severity:   "medium",
			from:       "disabled",
			to:         "enabled",
			expected:   true,
		},

		// Case insensitive tests
		{
			name:       "uppercase field path",
			fieldPath:  "SYSTEM_UPTIME",
			changeType: "UPDATE",
			category:   "SYSTEM",
			severity:   "LOW",
			from:       100,
			to:         200,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := differ.IsSignificantChange(tt.fieldPath, tt.changeType, tt.category, tt.severity, tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("Expected significance %v, got %v for change %s:%s (category:%s, severity:%s)",
					tt.expected, result, tt.fieldPath, tt.changeType, tt.category, tt.severity)
			}
		})
	}
}

func TestConfigurableDiffer_MatchesAlwaysSignificant(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name       string
		fieldPath  string
		changeType string
		category   string
		severity   string
		expected   bool
	}{
		// Severity patterns
		{
			name:       "high severity matches",
			fieldPath:  "any.field",
			changeType: "update",
			category:   "system",
			severity:   "high",
			expected:   true,
		},
		{
			name:       "critical severity matches",
			fieldPath:  "any.field",
			changeType: "create",
			category:   "system",
			severity:   "critical",
			expected:   true,
		},
		{
			name:       "medium severity doesn't match",
			fieldPath:  "any.field",
			changeType: "update",
			category:   "system",
			severity:   "medium",
			expected:   false,
		},

		// Category patterns
		{
			name:       "hardware category matches",
			fieldPath:  "memory.total",
			changeType: "update",
			category:   "hardware",
			severity:   "medium",
			expected:   true,
		},
		{
			name:       "network category matches",
			fieldPath:  "networking.ip",
			changeType: "update",
			category:   "network",
			severity:   "medium",
			expected:   true,
		},
		{
			name:       "system category doesn't match",
			fieldPath:  "system.field",
			changeType: "update",
			category:   "system",
			severity:   "medium",
			expected:   false,
		},

		// Change type patterns
		{
			name:       "delete change type matches",
			fieldPath:  "any.field",
			changeType: "delete",
			category:   "system",
			severity:   "low",
			expected:   true,
		},
		{
			name:       "update change type doesn't match delete pattern",
			fieldPath:  "any.field",
			changeType: "update",
			category:   "system",
			severity:   "low",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := differ.matchesAlwaysSignificant(
				strings.ToLower(tt.fieldPath),
				tt.changeType,
				tt.category,
				tt.severity,
			)
			if result != tt.expected {
				t.Errorf("Expected always significant %v, got %v for %s:%s (category:%s, severity:%s)",
					tt.expected, result, tt.fieldPath, tt.changeType, tt.category, tt.severity)
			}
		})
	}
}

func TestConfigurableDiffer_MatchesNeverSignificant(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name       string
		fieldPath  string
		changeType string
		category   string
		severity   string
		expected   bool
	}{
		// Never significant patterns
		{
			name:       "system uptime matches",
			fieldPath:  "system_uptime",
			changeType: "update",
			category:   "system",
			severity:   "low",
			expected:   true,
		},
		{
			name:       "metrics timestamp matches",
			fieldPath:  "metrics.timestamp",
			changeType: "update",
			category:   "system",
			severity:   "low",
			expected:   true,
		},
		{
			name:       "performance last update matches",
			fieldPath:  "performance.last_update",
			changeType: "update",
			category:   "system",
			severity:   "low",
			expected:   true,
		},
		{
			name:       "monitoring heartbeat matches",
			fieldPath:  "monitoring.heartbeat",
			changeType: "update",
			category:   "system",
			severity:   "low",
			expected:   true,
		},
		{
			name:       "normal field doesn't match",
			fieldPath:  "normal.field",
			changeType: "update",
			category:   "system",
			severity:   "medium",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := differ.matchesNeverSignificant(
				strings.ToLower(tt.fieldPath),
				tt.changeType,
				tt.category,
				tt.severity,
			)
			if result != tt.expected {
				t.Errorf("Expected never significant %v, got %v for %s:%s (category:%s, severity:%s)",
					tt.expected, result, tt.fieldPath, tt.changeType, tt.category, tt.severity)
			}
		})
	}
}

func TestConfigurableDiffer_MatchesMetaPattern(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name       string
		pattern    string
		fieldPath  string
		changeType string
		category   string
		severity   string
		expected   bool
	}{
		// Severity patterns
		{
			name:       "severity high matches",
			pattern:    "severity:(high|critical)",
			fieldPath:  "any.field",
			changeType: "update",
			category:   "system",
			severity:   "high",
			expected:   true,
		},
		{
			name:       "severity critical matches",
			pattern:    "severity:(high|critical)",
			fieldPath:  "any.field",
			changeType: "update",
			category:   "system",
			severity:   "critical",
			expected:   true,
		},
		{
			name:       "severity medium doesn't match",
			pattern:    "severity:(high|critical)",
			fieldPath:  "any.field",
			changeType: "update",
			category:   "system",
			severity:   "medium",
			expected:   false,
		},

		// Category patterns
		{
			name:       "category hardware matches",
			pattern:    "category:(hardware|network|security)",
			fieldPath:  "memory.total",
			changeType: "update",
			category:   "hardware",
			severity:   "medium",
			expected:   true,
		},
		{
			name:       "category network matches",
			pattern:    "category:(hardware|network|security)",
			fieldPath:  "networking.ip",
			changeType: "update",
			category:   "network",
			severity:   "medium",
			expected:   true,
		},
		{
			name:       "category system doesn't match",
			pattern:    "category:(hardware|network|security)",
			fieldPath:  "system.field",
			changeType: "update",
			category:   "system",
			severity:   "medium",
			expected:   false,
		},

		// Change type patterns
		{
			name:       "change type delete matches",
			pattern:    "change_type:delete",
			fieldPath:  "any.field",
			changeType: "delete",
			category:   "system",
			severity:   "medium",
			expected:   true,
		},
		{
			name:       "change type update doesn't match delete",
			pattern:    "change_type:delete",
			fieldPath:  "any.field",
			changeType: "update",
			category:   "system",
			severity:   "medium",
			expected:   false,
		},

		// Field path patterns
		{
			name:       "direct field path match",
			pattern:    "system_uptime",
			fieldPath:  "system_uptime",
			changeType: "update",
			category:   "system",
			severity:   "low",
			expected:   true,
		},
		{
			name:       "direct field path no match",
			pattern:    "system_uptime",
			fieldPath:  "other.field",
			changeType: "update",
			category:   "system",
			severity:   "low",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := differ.matchesMetaPattern(tt.pattern, tt.fieldPath, tt.changeType, tt.category, tt.severity)
			if result != tt.expected {
				t.Errorf("Expected meta pattern match %v, got %v for pattern '%s' with %s:%s (category:%s, severity:%s)",
					tt.expected, result, tt.pattern, tt.fieldPath, tt.changeType, tt.category, tt.severity)
			}
		})
	}
}

func TestConfigurableDiffer_MatchesRegexPattern(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name     string
		pattern  string
		value    string
		expected bool
	}{
		// Alternation patterns
		{
			name:     "alternation first option matches",
			pattern:  "(high|critical)",
			value:    "high",
			expected: true,
		},
		{
			name:     "alternation second option matches",
			pattern:  "(high|critical)",
			value:    "critical",
			expected: true,
		},
		{
			name:     "alternation no match",
			pattern:  "(high|critical)",
			value:    "medium",
			expected: false,
		},
		{
			name:     "alternation case insensitive",
			pattern:  "(high|critical)",
			value:    "HIGH",
			expected: true,
		},

		// Direct string matching
		{
			name:     "direct match",
			pattern:  "delete",
			value:    "delete",
			expected: true,
		},
		{
			name:     "direct case insensitive match",
			pattern:  "delete",
			value:    "DELETE",
			expected: true,
		},
		{
			name:     "direct no match",
			pattern:  "delete",
			value:    "update",
			expected: false,
		},

		// Edge cases
		{
			name:     "empty pattern",
			pattern:  "",
			value:    "any",
			expected: false,
		},
		{
			name:     "empty value",
			pattern:  "pattern",
			value:    "",
			expected: false,
		},
		{
			name:     "both empty",
			pattern:  "",
			value:    "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := differ.matchesRegexPattern(tt.pattern, tt.value)
			if result != tt.expected {
				t.Errorf("Expected regex pattern match %v, got %v for pattern '%s' with value '%s'",
					tt.expected, result, tt.pattern, tt.value)
			}
		})
	}
}

func TestConfigurableDiffer_IsFilteredByTime(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name      string
		fieldPath string
		expected  bool
	}{
		// Frequent changing fields (should be filtered)
		{
			name:      "metrics field filtered",
			fieldPath: "metrics.cpu_usage",
			expected:  true,
		},
		{
			name:      "performance field filtered",
			fieldPath: "performance.memory_usage",
			expected:  true,
		},
		{
			name:      "monitoring field filtered",
			fieldPath: "monitoring.status",
			expected:  true,
		},
		{
			name:      "timestamp field filtered",
			fieldPath: "system.timestamp",
			expected:  true,
		},
		{
			name:      "heartbeat field filtered",
			fieldPath: "monitoring.heartbeat",
			expected:  true,
		},
		{
			name:      "uptime field filtered",
			fieldPath: "system_uptime",
			expected:  true,
		},

		// Normal fields (should not be filtered)
		{
			name:      "os field not filtered",
			fieldPath: "os.version",
			expected:  false,
		},
		{
			name:      "configuration field not filtered",
			fieldPath: "configuration.service",
			expected:  false,
		},
		{
			name:      "networking field not filtered",
			fieldPath: "networking.hostname",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := differ.isFilteredByTime(strings.ToLower(tt.fieldPath))
			if result != tt.expected {
				t.Errorf("Expected time filter %v, got %v for field path '%s'",
					tt.expected, result, tt.fieldPath)
			}
		})
	}
}

func TestConfigurableDiffer_IsBelowThreshold(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name             string
		from             interface{}
		to               interface{}
		thresholdPercent float64
		expected         bool
	}{
		// Below threshold (should be filtered)
		{
			name:             "5% change below 20% threshold",
			from:             100,
			to:               105,
			thresholdPercent: 20.0,
			expected:         true,
		},
		{
			name:             "10% change below 20% threshold",
			from:             100,
			to:               110,
			thresholdPercent: 20.0,
			expected:         true,
		},
		{
			name:             "no change",
			from:             100,
			to:               100,
			thresholdPercent: 20.0,
			expected:         true,
		},

		// Above threshold (should not be filtered)
		{
			name:             "25% change above 20% threshold",
			from:             100,
			to:               125,
			thresholdPercent: 20.0,
			expected:         false,
		},
		{
			name:             "50% change above 20% threshold",
			from:             100,
			to:               150,
			thresholdPercent: 20.0,
			expected:         false,
		},
		{
			name:             "negative 30% change above 20% threshold",
			from:             100,
			to:               70,
			thresholdPercent: 20.0,
			expected:         false,
		},

		// Zero baseline cases
		{
			name:             "from zero to zero",
			from:             0,
			to:               0,
			thresholdPercent: 20.0,
			expected:         true,
		},
		{
			name:             "from zero to non-zero",
			from:             0,
			to:               50,
			thresholdPercent: 20.0,
			expected:         false,
		},

		// Non-numeric values
		{
			name:             "string values",
			from:             "old",
			to:               "new",
			thresholdPercent: 20.0,
			expected:         false,
		},
		{
			name:             "mixed types",
			from:             100,
			to:               "new",
			thresholdPercent: 20.0,
			expected:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := differ.isBelowThreshold(tt.from, tt.to, tt.thresholdPercent)
			if result != tt.expected {
				t.Errorf("Expected below threshold %v, got %v for change from %v to %v (threshold: %.1f%%)",
					tt.expected, result, tt.from, tt.to, tt.thresholdPercent)
			}
		})
	}
}

func TestConfigurableDiffer_FilterSignificantChanges(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	changes := []ChangeInfo{
		// Should be significant
		{
			FieldPath:  "os.version",
			ChangeType: "update",
			Category:   "os",
			Severity:   "high",
			From:       "20.04",
			To:         "22.04",
		},
		{
			FieldPath:  "memory.total",
			ChangeType: "update",
			Category:   "hardware",
			Severity:   "medium",
			From:       8192,
			To:         16384,
		},
		{
			FieldPath:  "critical.service",
			ChangeType: "delete",
			Category:   "features",
			Severity:   "critical",
			From:       "running",
			To:         nil,
		},

		// Should be filtered out
		{
			FieldPath:  "system_uptime",
			ChangeType: "update",
			Category:   "system",
			Severity:   "low",
			From:       100,
			To:         200,
		},
		{
			FieldPath:  "metrics.timestamp",
			ChangeType: "update",
			Category:   "system",
			Severity:   "low",
			From:       "2023-01-01",
			To:         "2023-01-02",
		},
		{
			FieldPath:  "monitoring.heartbeat",
			ChangeType: "update",
			Category:   "system",
			Severity:   "low",
			From:       1640995200,
			To:         1640995260,
		},
	}

	filtered := differ.FilterSignificantChanges(changes)

	// Should keep 3 significant changes and filter out 3 non-significant ones
	expectedCount := 3
	if len(filtered) != expectedCount {
		t.Errorf("Expected %d significant changes, got %d", expectedCount, len(filtered))
	}

	// Check that the right changes are kept
	significantPaths := make(map[string]bool)
	for _, change := range filtered {
		significantPaths[change.FieldPath] = true
	}

	expectedSignificant := []string{"os.version", "memory.total", "critical.service"}
	for _, path := range expectedSignificant {
		if !significantPaths[path] {
			t.Errorf("Expected %s to be significant", path)
		}
	}

	expectedFiltered := []string{"system_uptime", "metrics.timestamp", "monitoring.heartbeat"}
	for _, path := range expectedFiltered {
		if significantPaths[path] {
			t.Errorf("Expected %s to be filtered out", path)
		}
	}
}

func TestConfigurableDiffer_GetSignificanceFilters(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	filters := differ.GetSignificanceFilters()

	expectedKeys := []string{
		"always_significant",
		"never_significant",
		"time_filters",
		"value_filters",
		"default",
	}

	for _, key := range expectedKeys {
		if _, exists := filters[key]; !exists {
			t.Errorf("Expected filter key '%s' to exist", key)
		}
	}

	// Check always_significant has expected patterns
	if alwaysSignificant, ok := filters["always_significant"].([]string); ok {
		if len(alwaysSignificant) == 0 {
			t.Error("Expected non-empty always_significant patterns")
		}
	} else {
		t.Error("Expected always_significant to be []string")
	}

	// Check never_significant has expected patterns
	if neverSignificant, ok := filters["never_significant"].([]string); ok {
		if len(neverSignificant) == 0 {
			t.Error("Expected non-empty never_significant patterns")
		}
	} else {
		t.Error("Expected never_significant to be []string")
	}
}

func TestConfigurableDiffer_AnalyzeSignificanceDistribution(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	changes := []ChangeInfo{
		// Always significant (high severity)
		{
			FieldPath:  "os.version",
			ChangeType: "update",
			Category:   "os",
			Severity:   "high",
			From:       "20.04",
			To:         "22.04",
		},
		// Always significant (hardware category)
		{
			FieldPath:  "memory.total",
			ChangeType: "update",
			Category:   "hardware",
			Severity:   "medium",
			From:       8192,
			To:         16384,
		},
		// Never significant (system_uptime)
		{
			FieldPath:  "system_uptime",
			ChangeType: "update",
			Category:   "system",
			Severity:   "low",
			From:       100,
			To:         200,
		},
		// Time filtered (metrics)
		{
			FieldPath:  "metrics.cpu_usage",
			ChangeType: "update",
			Category:   "system",
			Severity:   "low",
			From:       50.0,
			To:         55.0,
		},
		// Default significant
		{
			FieldPath:  "configuration.service",
			ChangeType: "update",
			Category:   "features",
			Severity:   "medium",
			From:       "disabled",
			To:         "enabled",
		},
	}

	analysis := differ.AnalyzeSignificanceDistribution(changes)

	// Check required fields
	requiredFields := []string{"counts", "percentages", "effectiveness"}
	for _, field := range requiredFields {
		if _, exists := analysis[field]; !exists {
			t.Errorf("Expected analysis field '%s' to exist", field)
		}
	}

	// Check counts
	if counts, ok := analysis["counts"].(map[string]int); ok {
		if counts["total"] != 5 {
			t.Errorf("Expected total count 5, got %d", counts["total"])
		}
		if counts["always_significant"] != 2 {
			t.Errorf("Expected 2 always significant, got %d", counts["always_significant"])
		}
		if counts["never_significant"] != 1 {
			t.Errorf("Expected 1 never significant, got %d", counts["never_significant"])
		}
		if counts["time_filtered"] != 1 {
			t.Errorf("Expected 1 time filtered, got %d", counts["time_filtered"])
		}
		if counts["default_significant"] != 1 {
			t.Errorf("Expected 1 default significant, got %d", counts["default_significant"])
		}
		if counts["significant"] != 3 {
			t.Errorf("Expected 3 total significant, got %d", counts["significant"])
		}
	} else {
		t.Error("Expected counts to be map[string]int")
	}

	// Check percentages
	if percentages, ok := analysis["percentages"].(map[string]float64); ok {
		if percentages["significant"] != 60.0 {
			t.Errorf("Expected 60%% significant, got %.1f%%", percentages["significant"])
		}
	} else {
		t.Error("Expected percentages to be map[string]float64")
	}

	// Check effectiveness
	if effectiveness, ok := analysis["effectiveness"].(map[string]interface{}); ok {
		if noiseReduction, exists := effectiveness["noise_reduction"]; !exists {
			t.Error("Expected noise_reduction in effectiveness")
		} else if noiseReduction != 40.0 {
			t.Errorf("Expected 40%% noise reduction, got %v", noiseReduction)
		}

		if significanceRate, exists := effectiveness["significance_rate"]; !exists {
			t.Error("Expected significance_rate in effectiveness")
		} else if significanceRate != 60.0 {
			t.Errorf("Expected 60%% significance rate, got %v", significanceRate)
		}
	} else {
		t.Error("Expected effectiveness to be map[string]interface{}")
	}
}

func TestConfigurableDiffer_ValidateSignificancePatterns(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	err = differ.ValidateSignificancePatterns()
	if err != nil {
		t.Errorf("Significance pattern validation failed: %v", err)
	}
}
