/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package differ

import (
	"math"
	"testing"
)

func TestConfigurableDiffer_DetermineSeverity(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name        string
		fieldPath   string
		changeType  string
		from        interface{}
		to          interface{}
		expectedSev string
	}{
		// Critical severity tests
		{
			name:        "critical processor delete",
			fieldPath:   "processors.count",
			changeType:  "delete",
			from:        4,
			to:          nil,
			expectedSev: "critical",
		},
		{
			name:        "critical memory delete",
			fieldPath:   "memory.total",
			changeType:  "delete",
			from:        8192,
			to:          nil,
			expectedSev: "critical",
		},
		{
			name:        "critical networking delete",
			fieldPath:   "networking.interfaces",
			changeType:  "delete",
			from:        "eth0",
			to:          nil,
			expectedSev: "critical",
		},
		{
			name:        "critical features delete",
			fieldPath:   "features.essential",
			changeType:  "delete",
			from:        true,
			to:          nil,
			expectedSev: "critical",
		},
		{
			name:        "critical error create",
			fieldPath:   "system.error",
			changeType:  "create",
			from:        nil,
			to:          "critical system failure",
			expectedSev: "critical",
		},

		// High severity tests
		{
			name:        "high OS version update",
			fieldPath:   "os.version",
			changeType:  "update",
			from:        "20.04",
			to:          "22.04",
			expectedSev: "high",
		},
		{
			name:        "high kernel update",
			fieldPath:   "kernel.version",
			changeType:  "update",
			from:        "5.4.0",
			to:          "5.15.0",
			expectedSev: "high",
		},
		{
			name:        "high public IP update",
			fieldPath:   "public_ip",
			changeType:  "update",
			from:        "1.2.3.4",
			to:          "5.6.7.8",
			expectedSev: "high",
		},
		{
			name:        "high certificates update",
			fieldPath:   "certificates.ssl",
			changeType:  "update",
			from:        "old-cert",
			to:          "new-cert",
			expectedSev: "high",
		},
		{
			name:        "high warning create",
			fieldPath:   "system.warning",
			changeType:  "create",
			from:        nil,
			to:          "system warning message",
			expectedSev: "high",
		},

		// Medium severity tests
		{
			name:        "medium configuration update",
			fieldPath:   "configuration.service",
			changeType:  "update",
			from:        "disabled",
			to:          "enabled",
			expectedSev: "medium",
		},
		{
			name:        "medium services update",
			fieldPath:   "services.nginx.status",
			changeType:  "update",
			from:        "stopped",
			to:          "running",
			expectedSev: "medium",
		},
		{
			name:        "medium features update",
			fieldPath:   "features.docker.enabled",
			changeType:  "update",
			from:        false,
			to:          true,
			expectedSev: "medium",
		},
		{
			name:        "medium info create",
			fieldPath:   "system.info",
			changeType:  "create",
			from:        nil,
			to:          "informational message",
			expectedSev: "medium",
		},

		// Low severity tests
		{
			name:        "low metrics update",
			fieldPath:   "metrics.cpu_usage",
			changeType:  "update",
			from:        50.0,
			to:          60.0,
			expectedSev: "low",
		},
		{
			name:        "low performance update",
			fieldPath:   "performance.memory_usage",
			changeType:  "update",
			from:        "70%",
			to:          "75%",
			expectedSev: "low",
		},
		{
			name:        "low monitoring update",
			fieldPath:   "monitoring.heartbeat",
			changeType:  "update",
			from:        1640995200,
			to:          1640995260,
			expectedSev: "low",
		},
		{
			name:        "low debug create",
			fieldPath:   "system.debug",
			changeType:  "create",
			from:        nil,
			to:          "debug information",
			expectedSev: "low",
		},

		// Default severity tests
		{
			name:        "unknown field update",
			fieldPath:   "unknown.field",
			changeType:  "update",
			from:        "old",
			to:          "new",
			expectedSev: "medium", // Default severity
		},
		{
			name:        "unknown change type",
			fieldPath:   "os.version",
			changeType:  "unknown",
			from:        "20.04",
			to:          "22.04",
			expectedSev: "medium", // Default severity
		},

		// Case insensitive tests
		{
			name:        "uppercase field path",
			fieldPath:   "OS.VERSION",
			changeType:  "UPDATE",
			from:        "20.04",
			to:          "22.04",
			expectedSev: "high",
		},
		{
			name:        "mixed case field path",
			fieldPath:   "ProCessors.Count",
			changeType:  "Delete",
			from:        4,
			to:          nil,
			expectedSev: "critical",
		},

		// Numeric significance tests
		{
			name:        "significant numeric change",
			fieldPath:   "custom.field",
			changeType:  "update",
			from:        100,
			to:          150,      // 50% increase
			expectedSev: "medium", // Should be detected as significant
		},
		{
			name:        "small numeric change",
			fieldPath:   "custom.field",
			changeType:  "update",
			from:        100,
			to:          105,      // 5% increase
			expectedSev: "medium", // Default severity
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity := differ.DetermineSeverity(tt.fieldPath, tt.changeType, tt.from, tt.to)
			if severity != tt.expectedSev {
				t.Errorf("Expected severity %s, got %s for field %s with change %s", tt.expectedSev, severity, tt.fieldPath, tt.changeType)
			}
		})
	}
}

func TestConfigurableDiffer_IsSignificantNumericChange(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name     string
		from     interface{}
		to       interface{}
		expected bool
	}{
		// Significant changes (>20% threshold)
		{
			name:     "50% increase",
			from:     100,
			to:       150,
			expected: true,
		},
		{
			name:     "30% decrease",
			from:     100,
			to:       70,
			expected: true,
		},
		{
			name:     "25% increase float",
			from:     100.0,
			to:       125.0,
			expected: true,
		},
		{
			name:     "from zero to non-zero",
			from:     0,
			to:       50,
			expected: true,
		},

		// Non-significant changes (<20% threshold)
		{
			name:     "10% increase",
			from:     100,
			to:       110,
			expected: false,
		},
		{
			name:     "5% decrease",
			from:     100,
			to:       95,
			expected: false,
		},
		{
			name:     "no change",
			from:     100,
			to:       100,
			expected: false,
		},
		{
			name:     "from zero to zero",
			from:     0,
			to:       0,
			expected: false,
		},

		// Different numeric types
		{
			name:     "int32 significant change",
			from:     int32(100),
			to:       int32(130),
			expected: true,
		},
		{
			name:     "int64 significant change",
			from:     int64(1000),
			to:       int64(1250),
			expected: true,
		},
		{
			name:     "float32 significant change",
			from:     float32(50.0),
			to:       float32(65.0),
			expected: true,
		},

		// String values (numeric strings are parsed)
		{
			name:     "string values",
			from:     "100",
			to:       "150",
			expected: true, // String parsing is implemented and 50% change is significant
		},
		{
			name:     "nil values",
			from:     nil,
			to:       100,
			expected: false,
		},
		{
			name:     "mixed types",
			from:     100,
			to:       "150",
			expected: true, // String parsing converts "150" to numeric and 50% change is significant
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := differ.isSignificantNumericChange(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("Expected significance %v, got %v for change from %v to %v", tt.expected, result, tt.from, tt.to)
			}
		})
	}
}

func TestConfigurableDiffer_ToFloat64(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name        string
		value       interface{}
		expectedVal float64
		expectedOk  bool
	}{
		// Integer types
		{
			name:        "int value",
			value:       42,
			expectedVal: 42.0,
			expectedOk:  true,
		},
		{
			name:        "int32 value",
			value:       int32(123),
			expectedVal: 123.0,
			expectedOk:  true,
		},
		{
			name:        "int64 value",
			value:       int64(456),
			expectedVal: 456.0,
			expectedOk:  true,
		},

		// Float types
		{
			name:        "float32 value",
			value:       float32(3.14),
			expectedVal: 3.140000104904175, // float32 precision
			expectedOk:  true,
		},
		{
			name:        "float64 value",
			value:       3.14159,
			expectedVal: 3.14159,
			expectedOk:  true,
		},

		// String types (simple numeric strings)
		{
			name:        "positive integer string",
			value:       "123",
			expectedVal: 123.0,
			expectedOk:  true,
		},
		{
			name:        "negative integer string",
			value:       "-456",
			expectedVal: -456.0,
			expectedOk:  true,
		},
		{
			name:        "positive sign string",
			value:       "+789",
			expectedVal: 789.0,
			expectedOk:  true,
		},
		{
			name:        "decimal string",
			value:       "123.45",
			expectedVal: 123.45,
			expectedOk:  true,
		},
		{
			name:        "zero string",
			value:       "0",
			expectedVal: 0.0,
			expectedOk:  true,
		},

		// Non-numeric values
		{
			name:        "non-numeric string",
			value:       "hello",
			expectedVal: 0.0,
			expectedOk:  false,
		},
		{
			name:        "empty string",
			value:       "",
			expectedVal: 0.0,
			expectedOk:  false,
		},
		{
			name:        "nil value",
			value:       nil,
			expectedVal: 0.0,
			expectedOk:  false,
		},
		{
			name:        "boolean value",
			value:       true,
			expectedVal: 0.0,
			expectedOk:  false,
		},
		{
			name:        "map value",
			value:       map[string]string{"key": "value"},
			expectedVal: 0.0,
			expectedOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := differ.toFloat64(tt.value)

			if ok != tt.expectedOk {
				t.Errorf("Expected ok=%v, got ok=%v for value %v", tt.expectedOk, ok, tt.value)
			}

			if tt.expectedOk && math.Abs(val-tt.expectedVal) > 1e-10 {
				t.Errorf("Expected value %f, got %f for input %v", tt.expectedVal, val, tt.value)
			}
		})
	}
}

func TestConfigurableDiffer_ParseFloat(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name        string
		input       string
		expectedVal float64
		expectedOk  bool
	}{
		// Valid integer strings
		{
			name:        "positive integer",
			input:       "123",
			expectedVal: 123.0,
			expectedOk:  true,
		},
		{
			name:        "negative integer",
			input:       "-456",
			expectedVal: -456.0,
			expectedOk:  true,
		},
		{
			name:        "positive sign integer",
			input:       "+789",
			expectedVal: 789.0,
			expectedOk:  true,
		},
		{
			name:        "zero",
			input:       "0",
			expectedVal: 0.0,
			expectedOk:  true,
		},

		// Valid decimal strings
		{
			name:        "positive decimal",
			input:       "123.45",
			expectedVal: 123.45,
			expectedOk:  true,
		},
		{
			name:        "negative decimal",
			input:       "-67.89",
			expectedVal: -67.89,
			expectedOk:  true,
		},
		{
			name:        "decimal with leading zero",
			input:       "0.123",
			expectedVal: 0.123,
			expectedOk:  true,
		},

		// Edge cases
		{
			name:        "empty string",
			input:       "",
			expectedVal: 0.0,
			expectedOk:  false,
		},
		{
			name:        "only sign",
			input:       "-",
			expectedVal: 0.0,
			expectedOk:  true,
		},
		{
			name:        "only decimal point",
			input:       ".",
			expectedVal: 0.0,
			expectedOk:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := differ.parseFloat(tt.input)

			if ok != tt.expectedOk {
				t.Errorf("Expected ok=%v, got ok=%v for input '%s'", tt.expectedOk, ok, tt.input)
			}

			if tt.expectedOk && math.Abs(val-tt.expectedVal) > 1e-10 {
				t.Errorf("Expected value %f, got %f for input '%s'", tt.expectedVal, val, tt.input)
			}
		})
	}
}

func TestConfigurableDiffer_GetSeverityDescription(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name        string
		severity    string
		expectEmpty bool
	}{
		{
			name:        "critical severity",
			severity:    "critical",
			expectEmpty: false,
		},
		{
			name:        "high severity",
			severity:    "high",
			expectEmpty: false,
		},
		{
			name:        "medium severity",
			severity:    "medium",
			expectEmpty: false,
		},
		{
			name:        "low severity",
			severity:    "low",
			expectEmpty: false,
		},
		{
			name:        "uppercase severity",
			severity:    "CRITICAL",
			expectEmpty: false,
		},
		{
			name:        "mixed case severity",
			severity:    "High",
			expectEmpty: false,
		},
		{
			name:        "unknown severity",
			severity:    "unknown",
			expectEmpty: false, // Should return default description
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description := differ.GetSeverityDescription(tt.severity)

			if tt.expectEmpty && description != "" {
				t.Errorf("Expected empty description for severity %s, got: %s", tt.severity, description)
			}

			if !tt.expectEmpty && description == "" {
				t.Errorf("Expected non-empty description for severity %s", tt.severity)
			}
		})
	}
}

func TestConfigurableDiffer_GetAllSeverityLevels(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	levels := differ.GetAllSeverityLevels()

	expectedLevels := []string{"critical", "high", "medium", "low"}

	for _, expected := range expectedLevels {
		if description, exists := levels[expected]; !exists {
			t.Errorf("Expected severity level '%s' to be in results", expected)
		} else if description == "" {
			t.Errorf("Expected non-empty description for severity level '%s'", expected)
		}
	}

	// Check that all returned levels have descriptions
	for level, description := range levels {
		if description == "" {
			t.Errorf("Severity level '%s' has empty description", level)
		}
	}
}

func TestConfigurableDiffer_GetSeverityConditions(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	tests := []struct {
		name        string
		severity    string
		expectEmpty bool
	}{
		{
			name:        "critical severity conditions",
			severity:    "critical",
			expectEmpty: false,
		},
		{
			name:        "high severity conditions",
			severity:    "high",
			expectEmpty: false,
		},
		{
			name:        "medium severity conditions",
			severity:    "medium",
			expectEmpty: false,
		},
		{
			name:        "low severity conditions",
			severity:    "low",
			expectEmpty: false,
		},
		{
			name:        "unknown severity conditions",
			severity:    "unknown",
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditions := differ.GetSeverityConditions(tt.severity)

			if tt.expectEmpty {
				if conditions != nil {
					t.Errorf("Expected nil conditions for severity %s, got %d conditions", tt.severity, len(conditions))
				}
			} else {
				if len(conditions) == 0 {
					t.Errorf("Expected non-empty conditions for severity %s", tt.severity)
				}

				// Validate condition structure
				for i, condition := range conditions {
					if condition.ChangeType == "" {
						t.Errorf("Condition %d for severity %s has empty ChangeType", i, tt.severity)
					}
					if len(condition.Patterns) == 0 {
						t.Errorf("Condition %d for severity %s has no patterns", i, tt.severity)
					}
				}
			}
		})
	}
}

func TestConfigurableDiffer_ValidateSeverityPatterns(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	err = differ.ValidateSeverityPatterns()
	if err != nil {
		t.Errorf("Severity pattern validation failed: %v", err)
	}
}

func TestConfigurableDiffer_AnalyzeSeverityDistribution(t *testing.T) {
	differ, err := NewConfigurableDiffer("config.yml")
	if err != nil {
		t.Fatalf("Failed to create differ: %v", err)
	}

	severities := []string{
		"critical", "critical",
		"high", "high", "high",
		"medium", "medium",
		"low",
	}

	analysis := differ.AnalyzeSeverityDistribution(severities)

	// Check required fields
	requiredFields := []string{"distribution", "percentages", "most_common", "total_changes"}
	for _, field := range requiredFields {
		if _, exists := analysis[field]; !exists {
			t.Errorf("Expected analysis field '%s' to exist", field)
		}
	}

	// Check total changes
	if totalChanges, ok := analysis["total_changes"].(int); !ok || totalChanges != 8 {
		t.Errorf("Expected total_changes to be 8, got %v", analysis["total_changes"])
	}

	// Check distribution
	if distribution, ok := analysis["distribution"].(map[string]int); ok {
		if distribution["critical"] != 2 {
			t.Errorf("Expected 2 critical changes, got %d", distribution["critical"])
		}
		if distribution["high"] != 3 {
			t.Errorf("Expected 3 high changes, got %d", distribution["high"])
		}
		if distribution["medium"] != 2 {
			t.Errorf("Expected 2 medium changes, got %d", distribution["medium"])
		}
		if distribution["low"] != 1 {
			t.Errorf("Expected 1 low change, got %d", distribution["low"])
		}
	} else {
		t.Error("Expected distribution to be map[string]int")
	}

	// Check most common
	if mostCommon, ok := analysis["most_common"].(string); !ok || mostCommon != "high" {
		t.Errorf("Expected most_common to be 'high', got %v", analysis["most_common"])
	}

	// Test empty input
	emptyAnalysis := differ.AnalyzeSeverityDistribution([]string{})
	if totalChanges, ok := emptyAnalysis["total_changes"].(int); !ok || totalChanges != 0 {
		t.Errorf("Expected total_changes to be 0 for empty input, got %v", emptyAnalysis["total_changes"])
	}
}
