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
func (s *LocalInventoryService) GetInventoryDiffs(systemID string, page, pageSize int, severities, categories, diffTypes []string, fromDate, toDate *time.Time, inventoryIDs []int64) ([]models.InventoryDiff, int, error) {
	diffs, totalCount, err := s.inventoryRepo.GetInventoryDiffs(systemID, page, pageSize, severities, categories, diffTypes, fromDate, toDate, inventoryIDs)
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
		Strs("severity", severities).
		Strs("category", categories).
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
		logEvent.Int64("inventory_id", diffs[0].InventoryID)
	}

	logEvent.Msg("Retrieved latest inventory diffs batch")

	return diffs, nil
}

// GetInventoryTimeline returns a date-grouped timeline with summary counts and inventory IDs per group
func (s *LocalInventoryService) GetInventoryTimeline(systemID string, page, pageSize int, severities, categories, diffTypes []string, fromDate, toDate *time.Time) (models.InventoryTimelineSummary, []models.InventoryTimelineGroup, int, error) {
	// Get filtered summary counts
	summary, err := s.inventoryRepo.GetInventoryTimelineSummary(systemID, severities, categories, diffTypes, fromDate, toDate)
	if err != nil {
		return summary, nil, 0, err
	}

	// Get paginated date groups
	groups, totalCount, err := s.inventoryRepo.GetInventoryTimelineDateGroups(systemID, page, pageSize, severities, categories, diffTypes, fromDate, toDate)
	if err != nil {
		return summary, nil, 0, err
	}

	// Collect dates for this page and fetch their inventory IDs
	if len(groups) > 0 {
		dates := make([]string, len(groups))
		for i, g := range groups {
			dates[i] = g.Date
		}

		idsByDate, err := s.inventoryRepo.GetInventoryIDsForDates(systemID, dates)
		if err != nil {
			return summary, nil, 0, err
		}

		for i := range groups {
			if ids, ok := idsByDate[groups[i].Date]; ok {
				groups[i].InventoryIDs = ids
			} else {
				groups[i].InventoryIDs = []int64{}
			}
		}
	}

	logger.Debug().
		Str("system_id", systemID).
		Int("groups", len(groups)).
		Int("total_groups", totalCount).
		Int("summary_total", summary.Total).
		Msg("Retrieved inventory timeline")

	return summary, groups, totalCount, nil
}

// GetChangesSummary returns a summary of all changes for a system
func (s *LocalInventoryService) GetChangesSummary(systemID string) (*InventoryChangesSummary, error) {
	var lastInventoryTime time.Time
	err := s.db.QueryRow(`
		SELECT timestamp FROM inventory_records WHERE system_id = $1 ORDER BY timestamp DESC LIMIT 1
	`, systemID).Scan(&lastInventoryTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no inventory found for system %s", systemID)
		}
		return nil, fmt.Errorf("failed to get latest inventory time: %w", err)
	}

	var totalChanges, recentChanges, criticalChanges, alerts int
	yesterday := time.Now().Add(-24 * time.Hour)
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM inventory_diffs WHERE system_id = $1`, systemID).Scan(&totalChanges); err != nil {
		return nil, fmt.Errorf("failed to count total changes: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM inventory_diffs WHERE system_id = $1 AND created_at >= $2`, systemID, yesterday).Scan(&recentChanges); err != nil {
		return nil, fmt.Errorf("failed to count recent changes: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM inventory_diffs WHERE system_id = $1 AND severity = 'critical'`, systemID).Scan(&criticalChanges); err != nil {
		return nil, fmt.Errorf("failed to count critical changes: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM inventory_alerts WHERE system_id = $1 AND is_resolved = false`, systemID).Scan(&alerts); err != nil {
		return nil, fmt.Errorf("failed to count alerts: %w", err)
	}

	changesByCategory, err := s.queryGroupedCounts(`SELECT category, COUNT(*) FROM inventory_diffs WHERE system_id = $1 GROUP BY category`, systemID)
	if err != nil {
		return nil, err
	}
	changesBySeverity, err := s.queryGroupedCounts(`SELECT severity, COUNT(*) FROM inventory_diffs WHERE system_id = $1 GROUP BY severity`, systemID)
	if err != nil {
		return nil, err
	}

	return &InventoryChangesSummary{
		SystemID:           systemID,
		TotalChanges:       totalChanges,
		RecentChanges:      recentChanges,
		LastInventoryTime:  lastInventoryTime,
		HasCriticalChanges: criticalChanges > 0,
		HasAlerts:          alerts > 0,
		ChangesByCategory:  changesByCategory,
		ChangesBySeverity:  changesBySeverity,
	}, nil
}

// GetLatestInventoryChangesSummary returns a changes summary scoped to the most recent inventory batch
func (s *LocalInventoryService) GetLatestInventoryChangesSummary(systemID string) (*InventoryChangesSummary, error) {
	var lastInventoryTime time.Time
	var lastInventoryID int64
	err := s.db.QueryRow(`
		SELECT id, timestamp FROM inventory_records WHERE system_id = $1 ORDER BY timestamp DESC LIMIT 1
	`, systemID).Scan(&lastInventoryID, &lastInventoryTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no inventory found for system %s", systemID)
		}
		return nil, fmt.Errorf("failed to get latest inventory: %w", err)
	}

	var totalChanges, criticalChanges, alerts int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM inventory_diffs WHERE system_id = $1 AND current_id = $2`, systemID, lastInventoryID).Scan(&totalChanges); err != nil {
		return nil, fmt.Errorf("failed to count latest batch changes: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM inventory_diffs WHERE system_id = $1 AND current_id = $2 AND severity = 'critical'`, systemID, lastInventoryID).Scan(&criticalChanges); err != nil {
		return nil, fmt.Errorf("failed to count critical changes: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM inventory_alerts WHERE system_id = $1 AND is_resolved = false AND created_at >= $2`, systemID, lastInventoryTime.Add(-1*time.Hour)).Scan(&alerts); err != nil {
		return nil, fmt.Errorf("failed to count alerts: %w", err)
	}

	changesByCategory, err := s.queryGroupedCounts(`SELECT category, COUNT(*) FROM inventory_diffs WHERE system_id = $1 AND current_id = $2 GROUP BY category`, systemID, lastInventoryID)
	if err != nil {
		return nil, err
	}
	changesBySeverity, err := s.queryGroupedCounts(`SELECT severity, COUNT(*) FROM inventory_diffs WHERE system_id = $1 AND current_id = $2 GROUP BY severity`, systemID, lastInventoryID)
	if err != nil {
		return nil, err
	}

	return &InventoryChangesSummary{
		SystemID:           systemID,
		TotalChanges:       totalChanges,
		RecentChanges:      totalChanges,
		LastInventoryTime:  lastInventoryTime,
		HasCriticalChanges: criticalChanges > 0,
		HasAlerts:          alerts > 0,
		ChangesByCategory:  changesByCategory,
		ChangesBySeverity:  changesBySeverity,
	}, nil
}

// =============================================================================
// PRIVATE METHODS
// =============================================================================

func (s *LocalInventoryService) queryGroupedCounts(query string, args ...interface{}) (map[string]int, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query grouped counts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]int)
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return nil, fmt.Errorf("failed to scan grouped count: %w", err)
		}
		result[key] = count
	}
	return result, rows.Err()
}

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
		return *value
	}

	return jsonObj
}
