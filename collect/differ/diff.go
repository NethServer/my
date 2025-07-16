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
	"fmt"

	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
)

// NewDefaultDiffEngine creates a new configurable diff engine with default configuration
func NewDefaultDiffEngine() (*DiffEngine, error) {
	// Use default config path - config.yml in the differ directory
	return NewDiffEngine("")
}

// NewDiffEngineWithConfig creates a diff engine with custom configuration
func NewDiffEngineWithConfig(configPath string) (*DiffEngine, error) {
	if configPath == "" {
		// Use default configuration path
		configPath = "differ/config.yml"
	}

	engine, err := NewDiffEngine(configPath)
	if err != nil {
		logger.ComponentLogger("differ").Error().
			Str("config_path", configPath).
			Err(err).
			Msg("Failed to create configurable diff engine")
		return nil, err
	}

	logger.ComponentLogger("differ").Info().
		Str("config_path", configPath).
		Msg("Created configurable diff engine successfully")

	return engine, nil
}

// ComputeDiff compares two inventory records using configurable rules
// This is the main entry point that delegates to the new configurable engine
func ComputeDiff(systemID string, previous, current *models.InventoryRecord) ([]models.InventoryDiff, error) {
	engine, err := NewDefaultDiffEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create diff engine: %w", err)
	}

	return engine.ComputeDiff(systemID, previous, current)
}

// FilterSignificantChanges filters out noise and keeps only significant changes
// This function provides backward compatibility for existing code
func FilterSignificantChanges(diffs []models.InventoryDiff) ([]models.InventoryDiff, error) {
	engine, err := NewDefaultDiffEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create diff engine: %w", err)
	}

	// Convert to ChangeInfo format
	changes := make([]ChangeInfo, len(diffs))
	for i, diff := range diffs {
		changes[i] = ChangeInfo{
			FieldPath:  diff.FieldPath,
			ChangeType: diff.DiffType,
			Category:   diff.Category,
			Severity:   diff.Severity,
			From:       nil, // Values not available in this context
			To:         nil,
		}
	}

	// Filter using configurable engine
	significantChanges := engine.configurableDiffer.FilterSignificantChanges(changes)

	// Convert back to InventoryDiff format
	var filtered []models.InventoryDiff
	significantPaths := make(map[string]bool)
	for _, change := range significantChanges {
		significantPaths[change.FieldPath] = true
	}

	for _, diff := range diffs {
		if significantPaths[diff.FieldPath] {
			filtered = append(filtered, diff)
		}
	}

	return filtered, nil
}
