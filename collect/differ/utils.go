/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package differ

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/collect/models"
)

// CompareInventoryStructures compares two inventory structures for compatibility
func CompareInventoryStructures(data1, data2 json.RawMessage) (bool, error) {
	var struct1, struct2 map[string]interface{}

	if err := json.Unmarshal(data1, &struct1); err != nil {
		return false, fmt.Errorf("failed to parse first structure: %w", err)
	}

	if err := json.Unmarshal(data2, &struct2); err != nil {
		return false, fmt.Errorf("failed to parse second structure: %w", err)
	}

	return compareStructureRecursive(struct1, struct2, ""), nil
}

// compareStructureRecursive recursively compares two structures
func compareStructureRecursive(struct1, struct2 map[string]interface{}, path string) bool {
	// Check if all keys from struct1 exist in struct2
	for key := range struct1 {
		currentPath := path
		if currentPath != "" {
			currentPath += "."
		}
		currentPath += key

		if _, exists := struct2[key]; !exists {
			logger.ComponentLogger("differ-utils").Debug().
				Str("missing_key", key).
				Str("path", currentPath).
				Msg("Structure mismatch: key missing in second structure")
			return false
		}

		// If both values are maps, recurse
		if map1, ok1 := struct1[key].(map[string]interface{}); ok1 {
			if map2, ok2 := struct2[key].(map[string]interface{}); ok2 {
				if !compareStructureRecursive(map1, map2, currentPath) {
					return false
				}
			}
		}
	}

	return true
}

// SanitizeFieldPath sanitizes a field path for safe logging and storage
func SanitizeFieldPath(fieldPath string) string {
	// Remove potentially dangerous characters
	sanitized := strings.ReplaceAll(fieldPath, "\n", "")
	sanitized = strings.ReplaceAll(sanitized, "\r", "")
	sanitized = strings.ReplaceAll(sanitized, "\t", "")

	// Limit length to prevent abuse
	if len(sanitized) > 500 {
		sanitized = sanitized[:500] + "..."
	}

	return sanitized
}

// ValidateFieldPath validates that a field path is well-formed
func ValidateFieldPath(fieldPath string) error {
	if fieldPath == "" {
		return fmt.Errorf("field path cannot be empty")
	}

	if len(fieldPath) > 500 {
		return fmt.Errorf("field path exceeds maximum length of 500 characters")
	}

	// Check for invalid characters
	invalidChars := []string{"\n", "\r", "\t", "\x00"}
	for _, char := range invalidChars {
		if strings.Contains(fieldPath, char) {
			return fmt.Errorf("field path contains invalid character: %q", char)
		}
	}

	return nil
}

// FormatDiffSummary formats a summary of differences for logging
func FormatDiffSummary(diffs []models.InventoryDiff) string {
	if len(diffs) == 0 {
		return "No differences found"
	}

	categoryCount := make(map[string]int)
	severityCount := make(map[string]int)
	typeCount := make(map[string]int)

	for _, diff := range diffs {
		categoryCount[diff.Category]++
		severityCount[diff.Severity]++
		typeCount[diff.DiffType]++
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("Total: %d changes", len(diffs)))

	if len(categoryCount) > 0 {
		var categories []string
		for category, count := range categoryCount {
			categories = append(categories, fmt.Sprintf("%s: %d", category, count))
		}
		parts = append(parts, fmt.Sprintf("Categories: %s", strings.Join(categories, ", ")))
	}

	if len(severityCount) > 0 {
		var severities []string
		for severity, count := range severityCount {
			severities = append(severities, fmt.Sprintf("%s: %d", severity, count))
		}
		parts = append(parts, fmt.Sprintf("Severities: %s", strings.Join(severities, ", ")))
	}

	return strings.Join(parts, " | ")
}

// CalculateChangeVelocity calculates the velocity of changes over time
func CalculateChangeVelocity(diffs []models.InventoryDiff, timeWindow time.Duration) float64 {
	if len(diffs) == 0 || timeWindow <= 0 {
		return 0.0
	}

	// In a real implementation, you would use actual timestamps
	// For now, we'll simulate velocity calculation
	return float64(len(diffs)) / timeWindow.Hours()
}

// GroupDiffsByTimeWindow groups diffs by time windows
func GroupDiffsByTimeWindow(diffs []models.InventoryDiff, windowSize time.Duration) map[string][]models.InventoryDiff {
	groups := make(map[string][]models.InventoryDiff)

	// In a real implementation, you would group by actual timestamps
	// For now, we'll group by categories as a simulation
	for _, diff := range diffs {
		// Simulate time window grouping using categories
		windowKey := fmt.Sprintf("window_%s", diff.Category)
		groups[windowKey] = append(groups[windowKey], diff)
	}

	return groups
}

// CalculateNoisiness calculates how noisy a set of changes is
func CalculateNoisiness(diffs []models.InventoryDiff) float64 {
	if len(diffs) == 0 {
		return 0.0
	}

	// Count low-severity and frequently changing fields
	noisyCount := 0
	noisyPatterns := []string{
		"timestamp",
		"uptime",
		"memory.used",
		"arp_macs",
		"performance",
		"metrics",
	}

	for _, diff := range diffs {
		fieldLower := strings.ToLower(diff.FieldPath)

		// Check for noisy patterns
		for _, pattern := range noisyPatterns {
			if strings.Contains(fieldLower, pattern) {
				noisyCount++
				break
			}
		}

		// Check for low severity
		if diff.Severity == "low" {
			noisyCount++
		}
	}

	return float64(noisyCount) / float64(len(diffs)) * 100
}

// FilterDiffsByCategory filters diffs by category
func FilterDiffsByCategory(diffs []models.InventoryDiff, categories []string) []models.InventoryDiff {
	if len(categories) == 0 {
		return diffs
	}

	categorySet := make(map[string]bool)
	for _, category := range categories {
		categorySet[category] = true
	}

	var filtered []models.InventoryDiff
	for _, diff := range diffs {
		if categorySet[diff.Category] {
			filtered = append(filtered, diff)
		}
	}

	return filtered
}

// FilterDiffsBySeverity filters diffs by severity level
func FilterDiffsBySeverity(diffs []models.InventoryDiff, severities []string) []models.InventoryDiff {
	if len(severities) == 0 {
		return diffs
	}

	severitySet := make(map[string]bool)
	for _, severity := range severities {
		severitySet[severity] = true
	}

	var filtered []models.InventoryDiff
	for _, diff := range diffs {
		if severitySet[diff.Severity] {
			filtered = append(filtered, diff)
		}
	}

	return filtered
}

// FindTopChangedPaths finds the most frequently changed paths
func FindTopChangedPaths(diffs []models.InventoryDiff, limit int) []PathFrequency {
	pathCount := make(map[string]int)

	for _, diff := range diffs {
		// Extract the root path (first part)
		parts := strings.Split(diff.FieldPath, ".")
		if len(parts) > 0 {
			rootPath := parts[0]
			pathCount[rootPath]++
		}
	}

	// Convert to slice and sort
	var frequencies []PathFrequency
	for path, count := range pathCount {
		frequencies = append(frequencies, PathFrequency{
			Path:  path,
			Count: count,
		})
	}

	// Simple sort (in production, you'd use sort.Slice)
	for i := 0; i < len(frequencies)-1; i++ {
		for j := i + 1; j < len(frequencies); j++ {
			if frequencies[j].Count > frequencies[i].Count {
				frequencies[i], frequencies[j] = frequencies[j], frequencies[i]
			}
		}
	}

	// Apply limit
	if limit > 0 && len(frequencies) > limit {
		frequencies = frequencies[:limit]
	}

	return frequencies
}

// PathFrequency represents a path and its change frequency
type PathFrequency struct {
	Path  string
	Count int
}

// DetectAnomalies detects anomalous changes in the diff set
func DetectAnomalies(diffs []models.InventoryDiff) []models.InventoryDiff {
	var anomalies []models.InventoryDiff

	// Simple anomaly detection based on severity and patterns
	for _, diff := range diffs {
		// Critical changes are potential anomalies
		if diff.Severity == "critical" {
			anomalies = append(anomalies, diff)
			continue
		}

		// Unusual delete operations
		if diff.DiffType == "delete" && !strings.Contains(diff.FieldPath, "temporary") {
			anomalies = append(anomalies, diff)
			continue
		}

		// Unexpected OS changes
		if strings.Contains(diff.FieldPath, "os.") && diff.DiffType == "update" {
			anomalies = append(anomalies, diff)
			continue
		}
	}

	return anomalies
}

// ValidateDiffConsistency validates that a set of diffs is consistent
func ValidateDiffConsistency(diffs []models.InventoryDiff) []string {
	var issues []string

	// Check for duplicate field paths
	pathSeen := make(map[string]bool)
	for _, diff := range diffs {
		if pathSeen[diff.FieldPath] {
			issues = append(issues, fmt.Sprintf("Duplicate field path: %s", diff.FieldPath))
		}
		pathSeen[diff.FieldPath] = true
	}

	// Check for invalid field paths
	for _, diff := range diffs {
		if err := ValidateFieldPath(diff.FieldPath); err != nil {
			issues = append(issues, fmt.Sprintf("Invalid field path %s: %v", diff.FieldPath, err))
		}
	}

	// Check for inconsistent categories
	for _, diff := range diffs {
		if diff.Category == "" {
			issues = append(issues, fmt.Sprintf("Missing category for field path: %s", diff.FieldPath))
		}
	}

	// Check for inconsistent severities
	for _, diff := range diffs {
		if diff.Severity == "" {
			issues = append(issues, fmt.Sprintf("Missing severity for field path: %s", diff.FieldPath))
		}
	}

	return issues
}

// FormatDiffForDisplay formats a diff for human-readable display
func FormatDiffForDisplay(diff models.InventoryDiff) string {
	var parts []string

	// Add field path
	parts = append(parts, fmt.Sprintf("Field: %s", diff.FieldPath))

	// Add change type
	parts = append(parts, fmt.Sprintf("Type: %s", diff.DiffType))

	// Add category and severity
	parts = append(parts, fmt.Sprintf("Category: %s", diff.Category))
	parts = append(parts, fmt.Sprintf("Severity: %s", diff.Severity))

	// Add values if available
	if diff.PreviousValue != nil {
		parts = append(parts, fmt.Sprintf("Previous: %v", diff.PreviousValue))
	}
	if diff.CurrentValue != nil {
		parts = append(parts, fmt.Sprintf("Current: %v", diff.CurrentValue))
	}

	return strings.Join(parts, " | ")
}

// CalculateInventoryHealth calculates an overall health score based on diffs
func CalculateInventoryHealth(diffs []models.InventoryDiff) float64 {
	if len(diffs) == 0 {
		return 100.0 // Perfect health with no changes
	}

	score := 100.0

	// Deduct points based on severity
	for _, diff := range diffs {
		switch diff.Severity {
		case "critical":
			score -= 10.0
		case "high":
			score -= 5.0
		case "medium":
			score -= 2.0
		case "low":
			score -= 1.0
		}
	}

	// Ensure score doesn't go below 0
	if score < 0 {
		score = 0
	}

	return score
}

// GetDiffMetrics calculates comprehensive metrics for a set of diffs
func GetDiffMetrics(diffs []models.InventoryDiff) map[string]interface{} {
	metrics := make(map[string]interface{})

	// Basic counts
	metrics["total_changes"] = len(diffs)

	// Category distribution
	categoryCount := make(map[string]int)
	for _, diff := range diffs {
		categoryCount[diff.Category]++
	}
	metrics["category_distribution"] = categoryCount

	// Severity distribution
	severityCount := make(map[string]int)
	for _, diff := range diffs {
		severityCount[diff.Severity]++
	}
	metrics["severity_distribution"] = severityCount

	// Type distribution
	typeCount := make(map[string]int)
	for _, diff := range diffs {
		typeCount[diff.DiffType]++
	}
	metrics["type_distribution"] = typeCount

	// Quality metrics
	metrics["noisiness"] = CalculateNoisiness(diffs)
	metrics["health_score"] = CalculateInventoryHealth(diffs)
	metrics["anomaly_count"] = len(DetectAnomalies(diffs))

	// Top changed paths
	metrics["top_changed_paths"] = FindTopChangedPaths(diffs, 10)

	return metrics
}
