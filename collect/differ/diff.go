/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package differ

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/r3labs/diff/v3"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/models"
)

// DiffEngine handles JSON diff operations for inventory data
type DiffEngine struct {
	maxDepth int
}

// NewDiffEngine creates a new diff engine with configuration
func NewDiffEngine() *DiffEngine {
	return &DiffEngine{
		maxDepth: configuration.Config.InventoryDiffDepth,
	}
}

// ComputeDiff compares two inventory records and returns the differences
func (de *DiffEngine) ComputeDiff(systemID string, previous, current *models.InventoryRecord) ([]models.InventoryDiff, error) {
	// Parse JSON data
	var prevData, currData interface{}

	if err := json.Unmarshal(previous.Data, &prevData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal previous data: %w", err)
	}

	if err := json.Unmarshal(current.Data, &currData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal current data: %w", err)
	}

	// Compute differences using r3labs/diff
	changelog, err := diff.Diff(prevData, currData)
	if err != nil {
		return nil, fmt.Errorf("failed to compute diff: %w", err)
	}

	// Convert to our diff format
	var diffs []models.InventoryDiff
	for _, change := range changelog {
		inventoryDiff := de.convertToInventoryDiff(systemID, previous.ID, current.ID, change)
		if inventoryDiff != nil {
			diffs = append(diffs, *inventoryDiff)
		}
	}

	logger.Debug().
		Str("system_id", systemID).
		Int64("previous_id", previous.ID).
		Int64("current_id", current.ID).
		Int("diff_count", len(diffs)).
		Msg("Computed inventory diff")

	return diffs, nil
}

// convertToInventoryDiff converts a diff.Change to our InventoryDiff model
func (de *DiffEngine) convertToInventoryDiff(systemID string, previousID, currentID int64, change diff.Change) *models.InventoryDiff {
	// Skip changes that are too deep to avoid noise
	if de.getPathDepth(change.Path) > de.maxDepth {
		return nil
	}

	fieldPath := de.formatPath(change.Path)

	inventoryDiff := &models.InventoryDiff{
		SystemID:   systemID,
		PreviousID: &previousID,
		CurrentID:  currentID,
		FieldPath:  fieldPath,
		DiffType:   string(change.Type),
		Category:   de.categorizeField(fieldPath),
		Severity:   de.determineSeverity(fieldPath, string(change.Type), change.From, change.To),
	}

	// Set previous and current values
	if change.From != nil {
		prevStr := de.valueToString(change.From)
		inventoryDiff.PreviousValue = &prevStr
	}

	if change.To != nil {
		currStr := de.valueToString(change.To)
		inventoryDiff.CurrentValue = &currStr
	}

	return inventoryDiff
}

// formatPath converts a diff path to a readable field path
func (de *DiffEngine) formatPath(path []string) string {
	if len(path) == 0 {
		return "root"
	}
	return strings.Join(path, ".")
}

// getPathDepth returns the depth of a path
func (de *DiffEngine) getPathDepth(path []string) int {
	return len(path)
}

// categorizeField determines the category of a field based on its path
func (de *DiffEngine) categorizeField(fieldPath string) string {
	pathLower := strings.ToLower(fieldPath)

	// Operating system related
	if strings.Contains(pathLower, "os.") ||
		strings.Contains(pathLower, "kernel") ||
		strings.Contains(pathLower, "system_uptime") {
		return "os"
	}

	// Hardware related
	if strings.Contains(pathLower, "dmi.") ||
		strings.Contains(pathLower, "processors") ||
		strings.Contains(pathLower, "memory") ||
		strings.Contains(pathLower, "mountpoints") {
		return "hardware"
	}

	// Network related
	if strings.Contains(pathLower, "networking") ||
		strings.Contains(pathLower, "esmithdb.networks") ||
		strings.Contains(pathLower, "public_ip") ||
		strings.Contains(pathLower, "arp_macs") {
		return "network"
	}

	// Features and services
	if strings.Contains(pathLower, "features.") ||
		strings.Contains(pathLower, "rpms") {
		return "features"
	}

	// Configuration
	if strings.Contains(pathLower, "esmithdb.configuration") {
		return "configuration"
	}

	return "general"
}

// determineSeverity determines the severity of a change based on field and change type
func (de *DiffEngine) determineSeverity(fieldPath string, changeType string, from, to interface{}) string {
	pathLower := strings.ToLower(fieldPath)

	// Critical changes
	if changeType == "delete" {
		// Deleting important components is critical
		if strings.Contains(pathLower, "os.") ||
			strings.Contains(pathLower, "kernel") ||
			strings.Contains(pathLower, "networking.fqdn") {
			return "critical"
		}
	}

	// High severity changes
	if strings.Contains(pathLower, "os.release") ||
		strings.Contains(pathLower, "kernel") ||
		strings.Contains(pathLower, "subscription_status") ||
		strings.Contains(pathLower, "networking.fqdn") {
		return "high"
	}

	// Medium severity changes
	if strings.Contains(pathLower, "features.") ||
		strings.Contains(pathLower, "rpms") ||
		strings.Contains(pathLower, "memory") ||
		strings.Contains(pathLower, "public_ip") {
		return "medium"
	}

	// Special case: numeric threshold changes
	if de.isSignificantNumericChange(from, to) {
		return "medium"
	}

	return "low"
}

// isSignificantNumericChange checks if a numeric change is significant
func (de *DiffEngine) isSignificantNumericChange(from, to interface{}) bool {
	fromNum, fromOk := de.toFloat64(from)
	toNum, toOk := de.toFloat64(to)

	if !fromOk || !toOk {
		return false
	}

	// Consider changes > 20% as significant
	if fromNum == 0 {
		return toNum != 0
	}

	percentChange := ((toNum - fromNum) / fromNum) * 100
	return percentChange > 20 || percentChange < -20
}

// toFloat64 converts various numeric types to float64
func (de *DiffEngine) toFloat64(value interface{}) (float64, bool) {
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
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// valueToString converts any value to a string representation
func (de *DiffEngine) valueToString(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%.2f", v)
	default:
		// For complex types, marshal to JSON
		if data, err := json.Marshal(v); err == nil {
			return string(data)
		}
		return fmt.Sprintf("%v", v)
	}
}

// FilterSignificantChanges filters out noise and keeps only significant changes
func (de *DiffEngine) FilterSignificantChanges(diffs []models.InventoryDiff) []models.InventoryDiff {
	var filtered []models.InventoryDiff

	for _, diff := range diffs {
		if de.isSignificantChange(diff) {
			filtered = append(filtered, diff)
		}
	}

	logger.Debug().
		Int("original_count", len(diffs)).
		Int("filtered_count", len(filtered)).
		Msg("Filtered significant changes")

	return filtered
}

// isSignificantChange determines if a change is significant enough to track
func (de *DiffEngine) isSignificantChange(diff models.InventoryDiff) bool {
	pathLower := strings.ToLower(diff.FieldPath)

	// Always track high and critical severity changes
	if diff.Severity == "high" || diff.Severity == "critical" {
		return true
	}

	// Filter out noise from frequent changing fields
	noiseFields := []string{
		"timestamp",
		"system_uptime.seconds",
		"arp_macs",                 // This changes frequently
		"memory.system.used_bytes", // Memory usage fluctuates
		"memory.system.available_bytes",
	}

	for _, noiseField := range noiseFields {
		if strings.Contains(pathLower, noiseField) {
			// Only track if it's a significant numeric change
			if diff.Severity == "medium" {
				return true
			}
			return false
		}
	}

	// Track configuration and feature changes
	if strings.Contains(pathLower, "features.") ||
		strings.Contains(pathLower, "esmithdb.configuration") ||
		strings.Contains(pathLower, "rpms") {
		return true
	}

	// Track network changes
	if strings.Contains(pathLower, "networking") ||
		strings.Contains(pathLower, "esmithdb.networks") ||
		strings.Contains(pathLower, "public_ip") {
		return true
	}

	// Track OS and hardware changes
	if strings.Contains(pathLower, "os.") ||
		strings.Contains(pathLower, "kernel") ||
		strings.Contains(pathLower, "dmi.") ||
		strings.Contains(pathLower, "processors") {
		return true
	}

	return false
}

// GroupRelatedChanges groups related changes together for better organization
func (de *DiffEngine) GroupRelatedChanges(diffs []models.InventoryDiff) map[string][]models.InventoryDiff {
	groups := make(map[string][]models.InventoryDiff)

	for _, diff := range diffs {
		groupKey := de.getGroupKey(diff.FieldPath)
		groups[groupKey] = append(groups[groupKey], diff)
	}

	return groups
}

// getGroupKey determines the grouping key for a field path
func (de *DiffEngine) getGroupKey(fieldPath string) string {
	parts := strings.Split(fieldPath, ".")
	if len(parts) == 0 {
		return "general"
	}

	// Group by top-level categories
	topLevel := parts[0]
	switch topLevel {
	case "os", "kernel", "kernelrelease":
		return "operating_system"
	case "dmi", "processors", "memory", "mountpoints":
		return "hardware"
	case "networking", "public_ip", "arp_macs":
		return "network"
	case "features":
		if len(parts) > 1 {
			return fmt.Sprintf("features_%s", parts[1])
		}
		return "features"
	case "esmithdb":
		if len(parts) > 1 {
			return fmt.Sprintf("configuration_%s", parts[1])
		}
		return "configuration"
	case "rpms":
		return "packages"
	default:
		return topLevel
	}
}

// AnalyzeTrends analyzes trends in inventory changes over time
func (de *DiffEngine) AnalyzeTrends(systemID string, diffs []models.InventoryDiff) map[string]interface{} {
	trends := make(map[string]interface{})

	// Group by category
	categoryCount := make(map[string]int)
	severityCount := make(map[string]int)
	typeCount := make(map[string]int)

	for _, diff := range diffs {
		categoryCount[diff.Category]++
		severityCount[diff.Severity]++
		typeCount[diff.DiffType]++
	}

	trends["category_distribution"] = categoryCount
	trends["severity_distribution"] = severityCount
	trends["type_distribution"] = typeCount
	trends["total_changes"] = len(diffs)

	// Calculate change frequency
	if len(diffs) > 0 {
		trends["most_changed_category"] = de.findMaxKey(categoryCount)
		trends["dominant_severity"] = de.findMaxKey(severityCount)
		trends["dominant_type"] = de.findMaxKey(typeCount)
	}

	return trends
}

// findMaxKey finds the key with maximum value in a map
func (de *DiffEngine) findMaxKey(m map[string]int) string {
	var maxKey string
	var maxVal int

	for key, val := range m {
		if val > maxVal {
			maxVal = val
			maxKey = key
		}
	}

	return maxKey
}

// ValidateInventoryStructure validates that inventory data has expected structure
func (de *DiffEngine) ValidateInventoryStructure(data json.RawMessage) error {
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("invalid JSON structure: %w", err)
	}

	// Check for required top-level fields
	requiredFields := []string{"os", "networking", "processors", "memory"}
	for _, field := range requiredFields {
		if _, exists := parsed[field]; !exists {
			logger.Warn().
				Str("missing_field", field).
				Msg("Inventory missing expected field")
		}
	}

	return nil
}
