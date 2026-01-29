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

	"github.com/r3labs/diff/v3"

	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
)

// DiffEngine handles JSON diff operations for inventory data with configurable behavior
//
// The engine processes inventory data through several stages:
// 1. Data Parsing - Convert JSON to comparable structures
// 2. Diff Computation - Calculate differences using r3labs/diff
// 3. Diff Processing - Convert to internal format with categorization
// 4. Significance Filtering - Remove noise and irrelevant changes
// 5. Trend Analysis - Analyze patterns and group related changes
//
// All behavior is configurable through YAML configuration including:
// - Field categorization rules
// - Severity determination logic
// - Significance filtering patterns
// - Processing limits and thresholds
type DiffEngine struct {
	configurableDiffer *ConfigurableDiffer
	maxDepth           int
	maxDiffsPerRun     int
	maxFieldPathLength int
}

// NewDiffEngine creates a new diff engine with configurable behavior
func NewDiffEngine(configPath string) (*DiffEngine, error) {
	// Load configuration and create configurable differ
	configurableDiffer, err := NewConfigurableDiffer(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create configurable differ: %w", err)
	}

	config := configurableDiffer.GetConfig()

	return &DiffEngine{
		configurableDiffer: configurableDiffer,
		maxDepth:           config.Limits.MaxDiffDepth,
		maxDiffsPerRun:     config.Limits.MaxDiffsPerRun,
		maxFieldPathLength: config.Limits.MaxFieldPathLength,
	}, nil
}

// ComputeDiff compares two inventory records and returns categorized, filtered differences
//
// Complete Processing Flow:
// 1. Data Validation - Ensure inventory records are valid
// 2. JSON Parsing - Parse raw JSON data into comparable structures
// 3. Diff Calculation - Use r3labs/diff to compute raw differences
// 4. Diff Conversion - Convert to internal format with metadata
// 5. Field Categorization - Classify fields by functional area (OS, hardware, network, etc.)
// 6. Severity Determination - Assign severity levels based on impact
// 7. Significance Filtering - Remove noise and irrelevant changes
// 8. Result Compilation - Return processed and filtered differences
//
// Example Flow:
//
//	Input: Previous inventory {os: {version: "20.04"}} → Current inventory {os: {version: "22.04"}}
//	→ Raw Diff: [{type: "update", path: ["os", "version"], from: "20.04", to: "22.04"}]
//	→ Categorized: {category: "os", severity: "high", significant: true}
//	→ Filtered: Keep (OS version changes are always significant)
//	→ Output: Single InventoryDiff with all metadata
func (de *DiffEngine) ComputeDiff(systemID string, previous, current *models.InventoryRecord) ([]models.InventoryDiff, error) {
	engineLogger := logger.ComponentLogger("differ-engine")

	// Step 1: Data Validation
	engineLogger.Debug().
		Str("system_id", systemID).
		Int64("previous_id", previous.ID).
		Int64("current_id", current.ID).
		Msg("Starting inventory diff computation")

	if err := de.validateInventoryData(previous.Data, current.Data); err != nil {
		return nil, fmt.Errorf("inventory validation failed: %w", err)
	}

	// Step 2: JSON Parsing
	var prevData, currData interface{}

	if err := json.Unmarshal(previous.Data, &prevData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal previous data: %w", err)
	}

	if err := json.Unmarshal(current.Data, &currData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal current data: %w", err)
	}

	engineLogger.Debug().Msg("JSON parsing completed successfully")

	// Step 3: Diff Calculation
	changelog, err := diff.Diff(prevData, currData)
	if err != nil {
		return nil, fmt.Errorf("failed to compute diff: %w", err)
	}

	engineLogger.Debug().
		Int("raw_changes", len(changelog)).
		Msg("Raw diff computation completed")

	// Step 4: Diff Conversion and Processing
	var diffs []models.InventoryDiff
	changeInfos := make([]ChangeInfo, 0, len(changelog))

	for i, change := range changelog {
		// Apply processing limits
		if len(diffs) >= de.maxDiffsPerRun {
			engineLogger.Warn().
				Int("max_diffs", de.maxDiffsPerRun).
				Int("total_changes", len(changelog)).
				Msg("Reached maximum diffs per run limit")
			break
		}

		// Step 4a: Convert to internal format
		inventoryDiff := de.convertToInventoryDiff(systemID, previous.ID, current.ID, change)
		if inventoryDiff == nil {
			continue // Skip if conversion failed or filtered by depth
		}

		// Step 4b: Apply field length limits
		if len(inventoryDiff.FieldPath) > de.maxFieldPathLength {
			engineLogger.Debug().
				Str("field_path", inventoryDiff.FieldPath).
				Int("length", len(inventoryDiff.FieldPath)).
				Int("max_length", de.maxFieldPathLength).
				Msg("Skipping field path exceeding length limit")
			continue
		}

		// Step 5: Field Categorization
		inventoryDiff.Category = de.configurableDiffer.CategorizeField(inventoryDiff.FieldPath)

		// Step 6: Severity Determination
		inventoryDiff.Severity = de.configurableDiffer.DetermineSeverity(
			inventoryDiff.FieldPath,
			inventoryDiff.DiffType,
			change.From,
			change.To,
		)

		// Create change info for significance filtering
		changeInfo := ChangeInfo{
			FieldPath:  inventoryDiff.FieldPath,
			ChangeType: inventoryDiff.DiffType,
			Category:   inventoryDiff.Category,
			Severity:   inventoryDiff.Severity,
			From:       change.From,
			To:         change.To,
		}

		changeInfos = append(changeInfos, changeInfo)
		diffs = append(diffs, *inventoryDiff)

		// Log processing progress periodically
		if (i+1)%100 == 0 {
			engineLogger.Debug().
				Int("processed", i+1).
				Int("total", len(changelog)).
				Msg("Diff processing progress")
		}
	}

	engineLogger.Debug().
		Int("converted_diffs", len(diffs)).
		Msg("Diff conversion completed")

	// Step 7: Significance Filtering
	significantChanges := de.configurableDiffer.FilterSignificantChanges(changeInfos)

	// Filter the diffs to keep only significant ones
	var filteredDiffs []models.InventoryDiff
	significantPaths := make(map[string]bool)

	for _, change := range significantChanges {
		significantPaths[change.FieldPath] = true
	}

	for _, diff := range diffs {
		if significantPaths[diff.FieldPath] {
			filteredDiffs = append(filteredDiffs, diff)
		}
	}

	engineLogger.Info().
		Str("system_id", systemID).
		Int64("previous_id", previous.ID).
		Int64("current_id", current.ID).
		Int("raw_changes", len(changelog)).
		Int("processed_diffs", len(diffs)).
		Int("significant_diffs", len(filteredDiffs)).
		Msg("Inventory diff computation completed")

	// Step 8: Result Compilation
	return filteredDiffs, nil
}

// convertToInventoryDiff converts a diff.Change to our InventoryDiff model
//
// Conversion Process:
// 1. Depth Filtering - Skip changes that are too deep in the structure
// 2. Path Formatting - Convert diff path to readable field path
// 3. Value Conversion - Convert values to string representation
// 4. Model Creation - Create InventoryDiff with all metadata
func (de *DiffEngine) convertToInventoryDiff(systemID string, previousID, currentID int64, change diff.Change) *models.InventoryDiff {
	// Step 1: Depth Filtering
	if de.getPathDepth(change.Path) > de.maxDepth {
		logger.ComponentLogger("differ-engine").Debug().
			Strs("path", change.Path).
			Int("depth", de.getPathDepth(change.Path)).
			Int("max_depth", de.maxDepth).
			Msg("Skipping change exceeding max depth")
		return nil
	}

	// Step 2: Path Formatting
	fieldPath := de.formatPath(change.Path)

	// Step 3: Model Creation
	inventoryDiff := &models.InventoryDiff{
		SystemID:   systemID,
		PreviousID: &previousID,
		CurrentID:  currentID,
		FieldPath:  fieldPath,
		DiffType:   string(change.Type),
		// Category and Severity will be set by the caller
	}

	// Step 4: Value Conversion
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

// valueToString converts any value to a string representation
func (de *DiffEngine) valueToString(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case string:
		// JSON-encode string so it's valid JSONB when stored in PostgreSQL
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return v
		}
		return string(jsonBytes)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", v)
	case int32:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float32:
		return fmt.Sprintf("%.2f", v)
	case float64:
		return fmt.Sprintf("%.2f", v)
	default:
		// For complex types, marshal to JSON
		if data, err := json.Marshal(v); err == nil {
			return string(data)
		}
		return fmt.Sprintf("%v", v)
	}
}

// validateInventoryData validates that inventory data has expected structure
func (de *DiffEngine) validateInventoryData(previous, current json.RawMessage) error {
	// Validate previous data structure
	var prevParsed map[string]interface{}
	if err := json.Unmarshal(previous, &prevParsed); err != nil {
		return fmt.Errorf("invalid previous inventory JSON structure: %w", err)
	}

	// Validate current data structure
	var currParsed map[string]interface{}
	if err := json.Unmarshal(current, &currParsed); err != nil {
		return fmt.Errorf("invalid current inventory JSON structure: %w", err)
	}

	// Both NS8 (nethserver) and NSEC (nethsecurity) use the same top-level structure:
	// $schema, uuid, installation, facts
	expectedFields := []string{"facts", "uuid", "installation"}

	for _, field := range expectedFields {
		if _, exists := prevParsed[field]; !exists {
			logger.ComponentLogger("differ-engine").Warn().
				Str("missing_field", field).
				Msg("Previous inventory missing expected field")
		}
		if _, exists := currParsed[field]; !exists {
			logger.ComponentLogger("differ-engine").Warn().
				Str("missing_field", field).
				Msg("Current inventory missing expected field")
		}
	}

	return nil
}

// GroupRelatedChanges groups related changes together for better organization
//
// Grouping Strategy:
// 1. Top-level categorization by functional area
// 2. Sub-categorization by specific components
// 3. Logical grouping for related configuration changes
func (de *DiffEngine) GroupRelatedChanges(diffs []models.InventoryDiff) map[string][]models.InventoryDiff {
	groups := make(map[string][]models.InventoryDiff)

	for _, diff := range diffs {
		groupKey := de.getGroupKey(diff.FieldPath)
		groups[groupKey] = append(groups[groupKey], diff)
	}

	logger.ComponentLogger("differ-engine").Debug().
		Int("total_diffs", len(diffs)).
		Int("groups_created", len(groups)).
		Msg("Grouped related changes")

	return groups
}

// getGroupKey determines the grouping key for a field path
// Supports NS8/NSEC inventory structure (facts.*)
func (de *DiffEngine) getGroupKey(fieldPath string) string {
	parts := strings.Split(fieldPath, ".")
	if len(parts) == 0 {
		return "general"
	}

	// Handle NS8/NSEC structure (facts.*)
	if parts[0] == "facts" && len(parts) > 1 {
		secondLevel := parts[1]
		switch {
		case strings.HasPrefix(secondLevel, "modules"):
			return "modules"
		case secondLevel == "cluster":
			return "cluster"
		case strings.HasPrefix(secondLevel, "nodes"):
			return "nodes"
		case secondLevel == "distro":
			return "operating_system"
		case secondLevel == "processors" || secondLevel == "memory" || secondLevel == "product" || secondLevel == "virtual" || secondLevel == "pci":
			return "hardware"
		case secondLevel == "network":
			return "network"
		case secondLevel == "features":
			if len(parts) > 2 {
				return fmt.Sprintf("features_%s", parts[2])
			}
			return "features"
		default:
			return secondLevel
		}
	}

	// Fallback for non-facts paths
	return parts[0]
}

// AnalyzeTrends analyzes trends in inventory changes over time
//
// Trend Analysis Flow:
// 1. Category Distribution - Count changes by functional area
// 2. Severity Distribution - Analyze severity patterns
// 3. Change Type Distribution - Track create/update/delete patterns
// 4. Frequency Analysis - Identify most active areas
// 5. Pattern Recognition - Detect recurring change patterns
func (de *DiffEngine) AnalyzeTrends(systemID string, diffs []models.InventoryDiff) map[string]interface{} {
	trends := make(map[string]interface{})

	// Step 1: Category Distribution
	categoryCount := make(map[string]int)
	severityCount := make(map[string]int)
	typeCount := make(map[string]int)

	for _, diff := range diffs {
		categoryCount[diff.Category]++
		severityCount[diff.Severity]++
		typeCount[diff.DiffType]++
	}

	// Step 2: Compile Distribution Data
	trends["category_distribution"] = categoryCount
	trends["severity_distribution"] = severityCount
	trends["type_distribution"] = typeCount
	trends["total_changes"] = len(diffs)

	// Step 3: Frequency Analysis
	if len(diffs) > 0 {
		trends["most_changed_category"] = de.findMaxKey(categoryCount)
		trends["dominant_severity"] = de.findMaxKey(severityCount)
		trends["dominant_type"] = de.findMaxKey(typeCount)
	}

	// Step 4: Pattern Recognition
	trends["change_patterns"] = de.analyzeChangePatterns(diffs)

	logger.ComponentLogger("differ-engine").Debug().
		Str("system_id", systemID).
		Int("total_changes", len(diffs)).
		Interface("trends", trends).
		Msg("Trend analysis completed")

	return trends
}

// analyzeChangePatterns identifies patterns in changes
func (de *DiffEngine) analyzeChangePatterns(diffs []models.InventoryDiff) map[string]interface{} {
	patterns := make(map[string]interface{})

	// Analyze critical changes
	criticalChanges := 0
	for _, diff := range diffs {
		if diff.Severity == "critical" {
			criticalChanges++
		}
	}

	// Analyze configuration changes
	configChanges := 0
	for _, diff := range diffs {
		if strings.Contains(diff.FieldPath, "configuration") ||
			strings.Contains(diff.FieldPath, "features") {
			configChanges++
		}
	}

	patterns["critical_changes"] = criticalChanges
	patterns["configuration_changes"] = configChanges
	patterns["critical_ratio"] = float64(criticalChanges) / float64(len(diffs)) * 100
	patterns["configuration_ratio"] = float64(configChanges) / float64(len(diffs)) * 100

	return patterns
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

// GetConfiguration returns the current configuration
func (de *DiffEngine) GetConfiguration() *DifferConfig {
	return de.configurableDiffer.GetConfig()
}

// GetConfigurationLoadTime returns when the configuration was loaded
func (de *DiffEngine) GetConfigurationLoadTime() time.Time {
	return de.configurableDiffer.GetLoadTime()
}

// ReloadConfiguration reloads the configuration from file
func (de *DiffEngine) ReloadConfiguration(configPath string) error {
	if err := de.configurableDiffer.ReloadConfig(configPath); err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	// Update limits from new configuration
	config := de.configurableDiffer.GetConfig()
	de.maxDepth = config.Limits.MaxDiffDepth
	de.maxDiffsPerRun = config.Limits.MaxDiffsPerRun
	de.maxFieldPathLength = config.Limits.MaxFieldPathLength

	logger.ComponentLogger("differ-engine").Info().
		Str("config_path", configPath).
		Time("load_time", de.configurableDiffer.GetLoadTime()).
		Msg("Configuration reloaded successfully")

	return nil
}

// GetEngineStats returns engine statistics
func (de *DiffEngine) GetEngineStats() map[string]interface{} {
	return map[string]interface{}{
		"max_depth":             de.maxDepth,
		"max_diffs_per_run":     de.maxDiffsPerRun,
		"max_field_path_length": de.maxFieldPathLength,
		"config_load_time":      de.configurableDiffer.GetLoadTime(),
		"all_categories":        de.configurableDiffer.GetAllCategories(),
		"all_severity_levels":   de.configurableDiffer.GetAllSeverityLevels(),
		"significance_filters":  de.configurableDiffer.GetSignificanceFilters(),
	}
}
