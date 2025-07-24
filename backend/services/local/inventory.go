/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// InventoryChangesSummary represents a summary of inventory changes for a system
type InventoryChangesSummary struct {
	SystemID           string         `json:"system_id"`
	TotalChanges       int            `json:"total_changes"`
	RecentChanges      int            `json:"recent_changes"`
	LastInventoryTime  time.Time      `json:"last_inventory_time"`
	HasCriticalChanges bool           `json:"has_critical_changes"`
	HasAlerts          bool           `json:"has_alerts"`
	ChangesByCategory  map[string]int `json:"changes_by_category"`
	ChangesBySeverity  map[string]int `json:"changes_by_severity"`
}

// LocalInventoryService handles inventory-related operations
type LocalInventoryService struct {
	inventoryRepo *entities.LocalInventoryRepository
	db            *sql.DB
}

// NewInventoryService creates a new local inventory service
func NewInventoryService() *LocalInventoryService {
	return &LocalInventoryService{
		inventoryRepo: entities.NewLocalInventoryRepository(),
		db:            database.DB,
	}
}

// =============================================================================
// PUBLIC METHODS
// =============================================================================

// GetLatestInventory returns the most recent inventory record for a system
func (s *LocalInventoryService) GetLatestInventory(systemID string) (*models.InventoryRecord, error) {
	record, err := s.inventoryRepo.GetLatestInventory(systemID)
	if err != nil {
		return nil, err
	}

	logger.Debug().
		Str("system_id", systemID).
		Int64("inventory_id", record.ID).
		Time("timestamp", record.Timestamp).
		Msg("Retrieved latest inventory")

	return record, nil
}

// GetInventoryHistory returns paginated inventory history for a system
func (s *LocalInventoryService) GetInventoryHistory(systemID string, page, pageSize int, fromDate, toDate *time.Time) ([]models.InventoryRecord, int, error) {
	records, totalCount, err := s.inventoryRepo.GetInventoryHistory(systemID, page, pageSize, fromDate, toDate)
	if err != nil {
		return nil, 0, err
	}

	logger.Debug().
		Str("system_id", systemID).
		Int("count", len(records)).
		Int("total", totalCount).
		Int("page", page).
		Msg("Retrieved inventory history")

	return records, totalCount, nil
}

// GetInventoryDiffs returns paginated diffs for a system
func (s *LocalInventoryService) GetInventoryDiffs(systemID string, page, pageSize int, severity, category, diffType string, fromDate, toDate *time.Time) ([]models.InventoryDiff, int, error) {
	diffs, totalCount, err := s.inventoryRepo.GetInventoryDiffs(systemID, page, pageSize, severity, category, diffType, fromDate, toDate)
	if err != nil {
		return nil, 0, err
	}

	// Parse JSON values for each diff
	for i := range diffs {
		if diffs[i].PreviousValueRaw != nil {
			diffs[i].PreviousValue = s.parseJSONValue(diffs[i].PreviousValueRaw)
		}
		if diffs[i].CurrentValueRaw != nil {
			diffs[i].CurrentValue = s.parseJSONValue(diffs[i].CurrentValueRaw)
		}
	}

	logger.Debug().
		Str("system_id", systemID).
		Int("count", len(diffs)).
		Int("total", totalCount).
		Str("severity", severity).
		Str("category", category).
		Msg("Retrieved inventory diffs")

	return diffs, totalCount, nil
}

// GetLatestInventoryDiffs returns all diffs from the most recent inventory processing batch for a system
func (s *LocalInventoryService) GetLatestInventoryDiffs(systemID string) ([]models.InventoryDiff, error) {
	diffs, err := s.inventoryRepo.GetLatestInventoryDiffs(systemID)
	if err != nil {
		return nil, err
	}

	// Parse JSON values for each diff
	for i := range diffs {
		if diffs[i].PreviousValueRaw != nil {
			diffs[i].PreviousValue = s.parseJSONValue(diffs[i].PreviousValueRaw)
		}
		if diffs[i].CurrentValueRaw != nil {
			diffs[i].CurrentValue = s.parseJSONValue(diffs[i].CurrentValueRaw)
		}
	}

	logEvent := logger.Debug().
		Str("system_id", systemID).
		Int("count", len(diffs))

	if len(diffs) > 0 {
		logEvent.Int64("current_id", diffs[0].CurrentID)
	}

	logEvent.Msg("Retrieved latest inventory diffs batch")

	return diffs, nil
}

// GetChangesSummary returns a summary of changes for a system
func (s *LocalInventoryService) GetChangesSummary(systemID string) (*InventoryChangesSummary, error) {
	// Get latest inventory timestamp
	var lastInventoryTime time.Time
	err := s.db.QueryRow(`
		SELECT timestamp FROM inventory_records 
		WHERE system_id = $1 
		ORDER BY timestamp DESC 
		LIMIT 1
	`, systemID).Scan(&lastInventoryTime)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no inventory found for system %s", systemID)
		}
		return nil, fmt.Errorf("failed to get latest inventory time: %w", err)
	}

	// Count total changes
	var totalChanges int
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM inventory_diffs WHERE system_id = $1
	`, systemID).Scan(&totalChanges)
	if err != nil {
		return nil, fmt.Errorf("failed to count total changes: %w", err)
	}

	// Count recent changes (last 24h)
	var recentChanges int
	yesterday := time.Now().Add(-24 * time.Hour)
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM inventory_diffs 
		WHERE system_id = $1 AND created_at >= $2
	`, systemID, yesterday).Scan(&recentChanges)
	if err != nil {
		return nil, fmt.Errorf("failed to count recent changes: %w", err)
	}

	// Check for critical changes
	var criticalChanges int
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM inventory_diffs 
		WHERE system_id = $1 AND severity = 'critical'
	`, systemID).Scan(&criticalChanges)
	if err != nil {
		return nil, fmt.Errorf("failed to count critical changes: %w", err)
	}

	// Check for alerts
	var alerts int
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM inventory_alerts 
		WHERE system_id = $1 AND is_resolved = false
	`, systemID).Scan(&alerts)
	if err != nil {
		return nil, fmt.Errorf("failed to count alerts: %w", err)
	}

	// Get changes by category
	changesByCategory, err := s.getChangesByCategory(systemID)
	if err != nil {
		return nil, err
	}

	// Get changes by severity
	changesBySeverity, err := s.getChangesBySeverity(systemID)
	if err != nil {
		return nil, err
	}

	summary := &InventoryChangesSummary{
		SystemID:           systemID,
		TotalChanges:       totalChanges,
		RecentChanges:      recentChanges,
		LastInventoryTime:  lastInventoryTime,
		HasCriticalChanges: criticalChanges > 0,
		HasAlerts:          alerts > 0,
		ChangesByCategory:  changesByCategory,
		ChangesBySeverity:  changesBySeverity,
	}

	logger.Debug().
		Str("system_id", systemID).
		Int("total_changes", totalChanges).
		Int("recent_changes", recentChanges).
		Bool("has_critical", summary.HasCriticalChanges).
		Bool("has_alerts", summary.HasAlerts).
		Msg("Generated changes summary")

	return summary, nil
}

// GetLatestInventoryChangesSummary returns a summary of changes from the most recent inventory processing batch
func (s *LocalInventoryService) GetLatestInventoryChangesSummary(systemID string) (*InventoryChangesSummary, error) {
	// Get latest inventory timestamp and ID
	var lastInventoryTime time.Time
	var lastInventoryID int64
	err := s.db.QueryRow(`
		SELECT id, timestamp FROM inventory_records 
		WHERE system_id = $1 
		ORDER BY timestamp DESC 
		LIMIT 1
	`, systemID).Scan(&lastInventoryID, &lastInventoryTime)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no inventory found for system %s", systemID)
		}
		return nil, fmt.Errorf("failed to get latest inventory: %w", err)
	}

	// Count changes for this specific inventory batch
	var totalChanges int
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM inventory_diffs 
		WHERE system_id = $1 AND current_id = $2
	`, systemID, lastInventoryID).Scan(&totalChanges)
	if err != nil {
		return nil, fmt.Errorf("failed to count latest batch changes: %w", err)
	}

	// Count critical changes for this batch
	var criticalChanges int
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM inventory_diffs 
		WHERE system_id = $1 AND current_id = $2 AND severity = 'critical'
	`, systemID, lastInventoryID).Scan(&criticalChanges)
	if err != nil {
		return nil, fmt.Errorf("failed to count critical changes: %w", err)
	}

	// Check for alerts related to this inventory batch
	var alerts int
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM inventory_alerts 
		WHERE system_id = $1 AND is_resolved = false
		AND created_at >= $2
	`, systemID, lastInventoryTime.Add(-1*time.Hour)).Scan(&alerts) // Check alerts from 1 hour before inventory
	if err != nil {
		return nil, fmt.Errorf("failed to count alerts: %w", err)
	}

	// Get changes by category for this batch
	changesByCategory, err := s.getChangesByCategoryForBatch(systemID, lastInventoryID)
	if err != nil {
		return nil, err
	}

	// Get changes by severity for this batch
	changesBySeverity, err := s.getChangesBySeverityForBatch(systemID, lastInventoryID)
	if err != nil {
		return nil, err
	}

	summary := &InventoryChangesSummary{
		SystemID:           systemID,
		TotalChanges:       totalChanges,
		RecentChanges:      totalChanges, // For latest batch, all changes are "recent"
		LastInventoryTime:  lastInventoryTime,
		HasCriticalChanges: criticalChanges > 0,
		HasAlerts:          alerts > 0,
		ChangesByCategory:  changesByCategory,
		ChangesBySeverity:  changesBySeverity,
	}

	logger.Debug().
		Str("system_id", systemID).
		Int64("inventory_id", lastInventoryID).
		Int("total_changes", totalChanges).
		Bool("has_critical", summary.HasCriticalChanges).
		Bool("has_alerts", summary.HasAlerts).
		Msg("Generated latest inventory changes summary")

	return summary, nil
}

// =============================================================================
// PRIVATE METHODS
// =============================================================================

// parseJSONValue attempts to parse a string as JSON object, returns the object or the original string
func (s *LocalInventoryService) parseJSONValue(value *string) interface{} {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if len(trimmed) == 0 {
		return *value
	}

	// Check if it looks like JSON (starts with { or [)
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return *value
	}

	// Try to parse the JSON
	var jsonObj interface{}
	if err := json.Unmarshal([]byte(trimmed), &jsonObj); err != nil {
		// Not valid JSON, return original string
		return *value
	}

	// Return the parsed JSON object
	return jsonObj
}

// getChangesByCategory returns changes grouped by category for a system
func (s *LocalInventoryService) getChangesByCategory(systemID string) (map[string]int, error) {
	categoryRows, err := s.db.Query(`
		SELECT category, COUNT(*) FROM inventory_diffs 
		WHERE system_id = $1 
		GROUP BY category
	`, systemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get changes by category: %w", err)
	}
	defer func() { _ = categoryRows.Close() }()

	changesByCategory := make(map[string]int)
	for categoryRows.Next() {
		var category string
		var count int
		if err := categoryRows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("failed to scan category changes: %w", err)
		}
		changesByCategory[category] = count
	}

	return changesByCategory, nil
}

// getChangesBySeverity returns changes grouped by severity for a system
func (s *LocalInventoryService) getChangesBySeverity(systemID string) (map[string]int, error) {
	severityRows, err := s.db.Query(`
		SELECT severity, COUNT(*) FROM inventory_diffs 
		WHERE system_id = $1 
		GROUP BY severity
	`, systemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get changes by severity: %w", err)
	}
	defer func() { _ = severityRows.Close() }()

	changesBySeverity := make(map[string]int)
	for severityRows.Next() {
		var severity string
		var count int
		if err := severityRows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("failed to scan severity changes: %w", err)
		}
		changesBySeverity[severity] = count
	}

	return changesBySeverity, nil
}

// getChangesByCategoryForBatch returns changes grouped by category for a specific inventory batch
func (s *LocalInventoryService) getChangesByCategoryForBatch(systemID string, inventoryID int64) (map[string]int, error) {
	categoryRows, err := s.db.Query(`
		SELECT category, COUNT(*) FROM inventory_diffs 
		WHERE system_id = $1 AND current_id = $2
		GROUP BY category
	`, systemID, inventoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get changes by category: %w", err)
	}
	defer func() { _ = categoryRows.Close() }()

	changesByCategory := make(map[string]int)
	for categoryRows.Next() {
		var category string
		var count int
		if err := categoryRows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("failed to scan category changes: %w", err)
		}
		changesByCategory[category] = count
	}

	return changesByCategory, nil
}

// getChangesBySeverityForBatch returns changes grouped by severity for a specific inventory batch
func (s *LocalInventoryService) getChangesBySeverityForBatch(systemID string, inventoryID int64) (map[string]int, error) {
	severityRows, err := s.db.Query(`
		SELECT severity, COUNT(*) FROM inventory_diffs 
		WHERE system_id = $1 AND current_id = $2
		GROUP BY severity
	`, systemID, inventoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get changes by severity: %w", err)
	}
	defer func() { _ = severityRows.Close() }()

	changesBySeverity := make(map[string]int)
	for severityRows.Next() {
		var severity string
		var count int
		if err := severityRows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("failed to scan severity changes: %w", err)
		}
		changesBySeverity[severity] = count
	}

	return changesBySeverity, nil
}
