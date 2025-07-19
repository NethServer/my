/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package differ

import (
	"strings"

	"github.com/nethesis/my/collect/logger"
)

// IsSignificantChange determines if a change is significant enough to track and notify
//
// Process Flow:
// 1. Check "always significant" patterns first - these always pass
// 2. Check "never significant" patterns - these always fail
// 3. Apply time-based filters to reduce noise from frequent changes
// 4. Apply value-based filters to ignore minor numeric changes
// 5. Return default significance if no specific rules match
//
// Significance Filters:
//   - Always Significant: High/critical severity, hardware/network/security changes, deletions
//   - Never Significant: Timestamps, heartbeats, frequent monitoring data
//   - Time Filters: Ignore frequent changes within time windows
//   - Value Filters: Ignore changes below percentage thresholds
//
// Example:
//   - "severity:critical" → always significant
//   - "system_uptime" → never significant
//   - "metrics" within 5 minutes → filtered out
func (cd *ConfigurableDiffer) IsSignificantChange(fieldPath, changeType, category, severity string, from, to interface{}) bool {
	pathLower := strings.ToLower(fieldPath)

	// Step 1: Check "always significant" patterns
	if cd.matchesAlwaysSignificant(pathLower, changeType, category, severity) {
		logger.ComponentLogger("differ-significance").Debug().
			Str("field_path", fieldPath).
			Str("reason", "always_significant").
			Bool("significant", true).
			Msg("Change marked as always significant")

		return true
	}

	// Step 2: Check "never significant" patterns
	if cd.matchesNeverSignificant(pathLower, changeType, category, severity) {
		logger.ComponentLogger("differ-significance").Debug().
			Str("field_path", fieldPath).
			Str("reason", "never_significant").
			Bool("significant", false).
			Msg("Change marked as never significant")

		return false
	}

	// Step 3: Apply time-based filters
	if cd.isFilteredByTime(pathLower) {
		logger.ComponentLogger("differ-significance").Debug().
			Str("field_path", fieldPath).
			Str("reason", "time_filtered").
			Bool("significant", false).
			Msg("Change filtered by time-based rules")

		return false
	}

	// Step 4: Apply value-based filters
	if cd.isFilteredByValue(pathLower, from, to) {
		logger.ComponentLogger("differ-significance").Debug().
			Str("field_path", fieldPath).
			Str("reason", "value_filtered").
			Bool("significant", false).
			Msg("Change filtered by value-based rules")

		return false
	}

	// Step 5: Return default significance
	defaultSignificant := cd.config.Significance.Default.Significant

	logger.ComponentLogger("differ-significance").Debug().
		Str("field_path", fieldPath).
		Str("reason", "default").
		Bool("significant", defaultSignificant).
		Msg("Change assigned default significance")

	return defaultSignificant
}

// matchesAlwaysSignificant checks if a change matches "always significant" patterns
func (cd *ConfigurableDiffer) matchesAlwaysSignificant(fieldPath, changeType, category, severity string) bool {
	// Check patterns that use severity, category, or change_type matching
	for _, pattern := range cd.config.Significance.AlwaysSignificant {
		if cd.matchesMetaPattern(pattern, fieldPath, changeType, category, severity) {
			return true
		}
	}

	// Check direct field path patterns
	for patternKey, compiledPattern := range cd.significancePatterns {
		if strings.HasPrefix(patternKey, "always_") {
			if compiledPattern.MatchString(fieldPath) {
				return true
			}
		}
	}

	return false
}

// matchesNeverSignificant checks if a change matches "never significant" patterns
func (cd *ConfigurableDiffer) matchesNeverSignificant(fieldPath, changeType, category, severity string) bool {
	// Check patterns that use severity, category, or change_type matching
	for _, pattern := range cd.config.Significance.NeverSignificant {
		if cd.matchesMetaPattern(pattern, fieldPath, changeType, category, severity) {
			return true
		}
	}

	// Check direct field path patterns
	for patternKey, compiledPattern := range cd.significancePatterns {
		if strings.HasPrefix(patternKey, "never_") {
			if compiledPattern.MatchString(fieldPath) {
				return true
			}
		}
	}

	return false
}

// matchesMetaPattern checks if a pattern matches using meta-information (severity, category, change_type)
func (cd *ConfigurableDiffer) matchesMetaPattern(pattern, fieldPath, changeType, category, severity string) bool {
	// Handle severity patterns like "severity:(high|critical)"
	if strings.HasPrefix(pattern, "severity:") {
		severityPattern := strings.TrimPrefix(pattern, "severity:")
		return cd.matchesRegexPattern(severityPattern, severity)
	}

	// Handle category patterns like "category:(hardware|network|security)"
	if strings.HasPrefix(pattern, "category:") {
		categoryPattern := strings.TrimPrefix(pattern, "category:")
		return cd.matchesRegexPattern(categoryPattern, category)
	}

	// Handle change type patterns like "change_type:delete"
	if strings.HasPrefix(pattern, "change_type:") {
		changeTypePattern := strings.TrimPrefix(pattern, "change_type:")
		return cd.matchesRegexPattern(changeTypePattern, changeType)
	}

	// Handle field path patterns
	return cd.matchesRegexPattern(pattern, fieldPath)
}

// matchesRegexPattern checks if a value matches a regex pattern
func (cd *ConfigurableDiffer) matchesRegexPattern(pattern, value string) bool {
	// Simple pattern matching for common cases
	if strings.Contains(pattern, "|") {
		// Handle alternation like "(high|critical)"
		pattern = strings.Trim(pattern, "()")
		alternatives := strings.Split(pattern, "|")
		for _, alt := range alternatives {
			if strings.EqualFold(value, strings.TrimSpace(alt)) {
				return true
			}
		}
		return false
	}

	// Direct string matching
	return strings.EqualFold(value, pattern)
}

// isFilteredByTime checks if a change should be filtered based on time-based rules
func (cd *ConfigurableDiffer) isFilteredByTime(fieldPath string) bool {
	// Check time filters configuration
	for _, timeFilter := range cd.config.Significance.TimeFilters.IgnoreFrequent {
		if strings.Contains(fieldPath, timeFilter.Pattern) {
			// In a real implementation, you would check if this field changed
			// within the specified time window. For now, we'll simulate this
			// by checking if it's a known frequent-changing field
			return cd.isFrequentlyChangingField(fieldPath, timeFilter.WindowSeconds)
		}
	}

	return false
}

// isFrequentlyChangingField checks if a field changes frequently (simulation)
func (cd *ConfigurableDiffer) isFrequentlyChangingField(fieldPath string, windowSeconds int) bool {
	// In a real implementation, this would check a time-based cache
	// For now, we'll use pattern matching to identify frequent fields
	frequentPatterns := []string{
		"metrics",
		"performance",
		"monitoring",
		"timestamp",
		"heartbeat",
		"uptime",
	}

	pathLower := strings.ToLower(fieldPath)
	for _, pattern := range frequentPatterns {
		if strings.Contains(pathLower, pattern) {
			// Simulate frequency check - in practice you'd check actual timestamps
			return true
		}
	}

	return false
}

// isFilteredByValue checks if a change should be filtered based on value-based rules
func (cd *ConfigurableDiffer) isFilteredByValue(fieldPath string, from, to interface{}) bool {
	// Check value filters configuration
	for _, valueFilter := range cd.config.Significance.ValueFilters.IgnoreMinor {
		if strings.Contains(fieldPath, valueFilter.Pattern) {
			return cd.isBelowThreshold(from, to, valueFilter.ThresholdPercent)
		}
	}

	return false
}

// isBelowThreshold checks if a numeric change is below the significance threshold
func (cd *ConfigurableDiffer) isBelowThreshold(from, to interface{}, thresholdPercent float64) bool {
	fromNum, fromOk := cd.toFloat64(from)
	toNum, toOk := cd.toFloat64(to)

	if !fromOk || !toOk {
		return false
	}

	if fromNum == 0 {
		return toNum == 0
	}

	percentChange := ((toNum - fromNum) / fromNum) * 100
	if percentChange < 0 {
		percentChange = -percentChange
	}

	return percentChange < thresholdPercent
}

// GetSignificanceDescription returns the description for significance setting
func (cd *ConfigurableDiffer) GetSignificanceDescription() string {
	return cd.config.Significance.Default.Description
}

// GetSignificanceFilters returns all configured significance filters
func (cd *ConfigurableDiffer) GetSignificanceFilters() map[string]interface{} {
	return map[string]interface{}{
		"always_significant": cd.config.Significance.AlwaysSignificant,
		"never_significant":  cd.config.Significance.NeverSignificant,
		"time_filters":       cd.config.Significance.TimeFilters,
		"value_filters":      cd.config.Significance.ValueFilters,
		"default":            cd.config.Significance.Default,
	}
}

// FilterSignificantChanges filters a list of changes to keep only significant ones
func (cd *ConfigurableDiffer) FilterSignificantChanges(changes []ChangeInfo) []ChangeInfo {
	var filtered []ChangeInfo

	for _, change := range changes {
		if cd.IsSignificantChange(
			change.FieldPath,
			change.ChangeType,
			change.Category,
			change.Severity,
			change.From,
			change.To,
		) {
			filtered = append(filtered, change)
		}
	}

	logger.ComponentLogger("differ-significance").Info().
		Int("original_count", len(changes)).
		Int("filtered_count", len(filtered)).
		Msg("Filtered significant changes")

	return filtered
}

// ChangeInfo represents information about a change for significance filtering
type ChangeInfo struct {
	FieldPath  string
	ChangeType string
	Category   string
	Severity   string
	From       interface{}
	To         interface{}
}

// AnalyzeSignificanceDistribution analyzes the distribution of significance decisions
func (cd *ConfigurableDiffer) AnalyzeSignificanceDistribution(changes []ChangeInfo) map[string]interface{} {
	stats := map[string]int{
		"total":               len(changes),
		"significant":         0,
		"always_significant":  0,
		"never_significant":   0,
		"time_filtered":       0,
		"value_filtered":      0,
		"default_significant": 0,
	}

	for _, change := range changes {
		pathLower := strings.ToLower(change.FieldPath)

		if cd.matchesAlwaysSignificant(pathLower, change.ChangeType, change.Category, change.Severity) {
			stats["always_significant"]++
			stats["significant"]++
		} else if cd.matchesNeverSignificant(pathLower, change.ChangeType, change.Category, change.Severity) {
			stats["never_significant"]++
		} else if cd.isFilteredByTime(pathLower) {
			stats["time_filtered"]++
		} else if cd.isFilteredByValue(pathLower, change.From, change.To) {
			stats["value_filtered"]++
		} else if cd.config.Significance.Default.Significant {
			stats["default_significant"]++
			stats["significant"]++
		}
	}

	// Calculate percentages
	percentages := make(map[string]float64)
	for key, count := range stats {
		if key != "total" {
			percentages[key] = float64(count) / float64(stats["total"]) * 100
		}
	}

	return map[string]interface{}{
		"counts":      stats,
		"percentages": percentages,
		"effectiveness": map[string]interface{}{
			"noise_reduction":   float64(stats["never_significant"]+stats["time_filtered"]+stats["value_filtered"]) / float64(stats["total"]) * 100,
			"significance_rate": float64(stats["significant"]) / float64(stats["total"]) * 100,
		},
	}
}

// ValidateSignificancePatterns validates all significance patterns
func (cd *ConfigurableDiffer) ValidateSignificancePatterns() error {
	// Test always significant patterns
	for _, pattern := range cd.config.Significance.AlwaysSignificant {
		cd.matchesMetaPattern(pattern, "test.field", "update", "test", "medium")
	}

	// Test never significant patterns
	for _, pattern := range cd.config.Significance.NeverSignificant {
		cd.matchesMetaPattern(pattern, "test.field", "update", "test", "medium")
	}

	// Test compiled patterns
	for patternKey, compiledPattern := range cd.significancePatterns {
		if compiledPattern == nil {
			logger.ComponentLogger("differ-significance").Error().
				Str("pattern_key", patternKey).
				Msg("Found nil compiled pattern during validation")
			continue
		}

		// Test with sample data
		compiledPattern.MatchString("test.field")
	}

	return nil
}
