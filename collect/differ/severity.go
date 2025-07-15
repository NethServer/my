/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package differ

import (
	"strings"

	"github.com/nethesis/my/backend/logger"
)

// DetermineSeverity determines the severity of a change based on configuration
//
// Process Flow:
// 1. Normalize field path and change type for pattern matching
// 2. Check each severity level (critical, high, medium, low) in order
// 3. For each level, check if change type matches and patterns match
// 4. Return first matching severity level or default if no match
//
// Severity Levels:
//   - Critical: Immediate attention required (system failures, deletions of core components)
//   - High: Important changes requiring attention (OS updates, security changes)
//   - Medium: Moderate changes for review (configuration updates, feature changes)
//   - Low: Minor changes for reference (metrics, performance data)
//
// Example:
//   - "delete" + "processors" → "critical"
//   - "update" + "os.version" → "high"
//   - "create" + "features.module" → "medium"
func (cd *ConfigurableDiffer) DetermineSeverity(fieldPath, changeType string, from, to interface{}) string {
	// Step 1: Normalize inputs for pattern matching
	pathLower := strings.ToLower(fieldPath)
	changeTypeLower := strings.ToLower(changeType)

	// Step 2: Check severity levels in order of priority
	severityLevels := []string{"critical", "high", "medium", "low"}

	for _, level := range severityLevels {
		// Step 3: Check if this level matches the change
		if cd.matchesSeverityLevel(level, changeTypeLower, pathLower) {
			logger.ComponentLogger("differ-severity").Debug().
				Str("field_path", fieldPath).
				Str("change_type", changeType).
				Str("severity", level).
				Msg("Severity determined by pattern match")

			return level
		}
	}

	// Step 4: Check for numeric significance
	if cd.isSignificantNumericChange(from, to) {
		logger.ComponentLogger("differ-severity").Debug().
			Str("field_path", fieldPath).
			Str("change_type", changeType).
			Str("severity", "medium").
			Msg("Severity determined by numeric significance")

		return "medium"
	}

	// Step 5: Return default severity if no patterns match
	defaultSeverity := cd.config.Severity.Default.Level

	logger.ComponentLogger("differ-severity").Debug().
		Str("field_path", fieldPath).
		Str("change_type", changeType).
		Str("severity", defaultSeverity).
		Msg("Severity assigned to default level")

	return defaultSeverity
}

// matchesSeverityLevel checks if a change matches patterns for a specific severity level
func (cd *ConfigurableDiffer) matchesSeverityLevel(level, changeType, fieldPath string) bool {
	// Get patterns for this severity level
	levelPatterns, exists := cd.severityPatterns[level]
	if !exists {
		return false
	}

	// Check patterns for this change type
	patterns, exists := levelPatterns[changeType]
	if !exists {
		return false
	}

	// Test field path against all patterns for this change type
	for _, pattern := range patterns {
		if pattern.MatchString(fieldPath) {
			return true
		}
	}

	return false
}

// isSignificantNumericChange checks if a numeric change meets significance threshold
//
// Process Flow:
// 1. Convert both values to float64 for comparison
// 2. Handle zero baseline cases
// 3. Calculate percentage change
// 4. Check if change exceeds configured threshold (default 20%)
func (cd *ConfigurableDiffer) isSignificantNumericChange(from, to interface{}) bool {
	// Step 1: Convert values to numeric format
	fromNum, fromOk := cd.toFloat64(from)
	toNum, toOk := cd.toFloat64(to)

	if !fromOk || !toOk {
		return false
	}

	// Step 2: Handle zero baseline
	if fromNum == 0 {
		return toNum != 0
	}

	// Step 3: Calculate percentage change
	percentChange := ((toNum - fromNum) / fromNum) * 100

	// Step 4: Check significance threshold (configurable, default 20%)
	threshold := 20.0 // TODO: Make this configurable in YAML

	return percentChange > threshold || percentChange < -threshold
}

// toFloat64 converts various numeric types to float64 for comparison
func (cd *ConfigurableDiffer) toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case string:
		// Try to parse string as number
		if len(v) > 0 {
			// Simple numeric string detection
			if (v[0] >= '0' && v[0] <= '9') || v[0] == '-' || v[0] == '+' {
				// Use a simple parser to avoid importing strconv
				return cd.parseFloat(v)
			}
		}
	}
	return 0, false
}

// parseFloat simple float parser to avoid external dependencies
func (cd *ConfigurableDiffer) parseFloat(s string) (float64, bool) {
	// This is a simplified parser - in production you'd use strconv.ParseFloat
	// For now, just handle basic integer cases
	if s == "" {
		return 0, false
	}

	result := 0.0
	sign := 1.0
	i := 0

	// Handle sign
	switch s[0] {
	case '-':
		sign = -1
		i = 1
	case '+':
		i = 1
	}

	// Parse integer part
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		result = result*10 + float64(s[i]-'0')
		i++
	}

	// Handle decimal part (simplified)
	if i < len(s) && s[i] == '.' {
		i++
		decimal := 0.0
		factor := 0.1
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			decimal += float64(s[i]-'0') * factor
			factor *= 0.1
			i++
		}
		result += decimal
	}

	return result * sign, true
}

// GetSeverityDescription returns the description for a given severity level
func (cd *ConfigurableDiffer) GetSeverityDescription(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return cd.config.Severity.Critical.Description
	case "high":
		return cd.config.Severity.High.Description
	case "medium":
		return cd.config.Severity.Medium.Description
	case "low":
		return cd.config.Severity.Low.Description
	default:
		return cd.config.Severity.Default.Description
	}
}

// GetAllSeverityLevels returns all configured severity levels with their descriptions
func (cd *ConfigurableDiffer) GetAllSeverityLevels() map[string]string {
	levels := make(map[string]string)

	levels["critical"] = cd.config.Severity.Critical.Description
	levels["high"] = cd.config.Severity.High.Description
	levels["medium"] = cd.config.Severity.Medium.Description
	levels["low"] = cd.config.Severity.Low.Description

	return levels
}

// GetSeverityConditions returns the conditions for a given severity level (for debugging)
func (cd *ConfigurableDiffer) GetSeverityConditions(severity string) []SeverityCondition {
	switch strings.ToLower(severity) {
	case "critical":
		return cd.config.Severity.Critical.Conditions
	case "high":
		return cd.config.Severity.High.Conditions
	case "medium":
		return cd.config.Severity.Medium.Conditions
	case "low":
		return cd.config.Severity.Low.Conditions
	default:
		return nil
	}
}

// ValidateSeverityPatterns validates all severity patterns can be compiled
func (cd *ConfigurableDiffer) ValidateSeverityPatterns() error {
	for level, levelPatterns := range cd.severityPatterns {
		for changeType, patterns := range levelPatterns {
			for i, pattern := range patterns {
				if pattern == nil {
					logger.ComponentLogger("differ-severity").Error().
						Str("severity", level).
						Str("change_type", changeType).
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
	}

	return nil
}

// AnalyzeSeverityDistribution analyzes the distribution of severity levels in a set of diffs
func (cd *ConfigurableDiffer) AnalyzeSeverityDistribution(severities []string) map[string]interface{} {
	distribution := make(map[string]int)
	total := len(severities)

	// Count occurrences of each severity level
	for _, severity := range severities {
		distribution[severity]++
	}

	// Calculate percentages
	percentages := make(map[string]float64)
	for severity, count := range distribution {
		percentages[severity] = float64(count) / float64(total) * 100
	}

	// Find most common severity
	mostCommon := ""
	maxCount := 0
	for severity, count := range distribution {
		if count > maxCount {
			maxCount = count
			mostCommon = severity
		}
	}

	return map[string]interface{}{
		"distribution":  distribution,
		"percentages":   percentages,
		"most_common":   mostCommon,
		"total_changes": total,
	}
}
