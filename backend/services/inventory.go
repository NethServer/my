/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/logger"
	collectModels "github.com/nethesis/my/collect/models"
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

// InventoryService handles inventory-related operations
type InventoryService struct {
	db *sql.DB
}

// NewInventoryService creates a new inventory service
func NewInventoryService() *InventoryService {
	return &InventoryService{
		db: database.DB,
	}
}

// parseJSONValue attempts to parse a string as JSON object, returns the object or the original string
func parseJSONValue(value *string) interface{} {
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

// GetLatestInventory returns the most recent inventory record for a system
func (s *InventoryService) GetLatestInventory(systemID string) (*collectModels.InventoryRecord, error) {
	query := `
		SELECT id, system_id, timestamp, data, data_hash, data_size, 
		       processed_at, has_changes, change_count, created_at, updated_at
		FROM inventory_records 
		WHERE system_id = $1 
		ORDER BY timestamp DESC 
		LIMIT 1
	`

	var record collectModels.InventoryRecord
	var processedAt sql.NullTime

	err := s.db.QueryRow(query, systemID).Scan(
		&record.ID, &record.SystemID, &record.Timestamp, &record.Data, &record.DataHash,
		&record.DataSize, &processedAt, &record.HasChanges,
		&record.ChangeCount, &record.CreatedAt, &record.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no inventory found for system %s", systemID)
		}
		return nil, fmt.Errorf("failed to query latest inventory: %w", err)
	}

	if processedAt.Valid {
		record.ProcessedAt = &processedAt.Time
	}

	logger.Debug().
		Str("system_id", systemID).
		Int64("inventory_id", record.ID).
		Time("timestamp", record.Timestamp).
		Msg("Retrieved latest inventory")

	return &record, nil
}

// GetInventoryHistory returns paginated inventory history for a system
func (s *InventoryService) GetInventoryHistory(systemID string, page, pageSize int, fromDate, toDate *time.Time) ([]collectModels.InventoryRecord, int, error) {
	// Build WHERE clause with date filters
	whereClause := "WHERE system_id = $1"
	args := []interface{}{systemID}
	argIndex := 2

	if fromDate != nil {
		whereClause += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, *fromDate)
		argIndex++
	}

	if toDate != nil {
		whereClause += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, *toDate)
		argIndex++
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM inventory_records %s", whereClause)
	var totalCount int
	err := s.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count inventory records: %w", err)
	}

	// Get paginated records
	offset := (page - 1) * pageSize
	query := fmt.Sprintf(`
		SELECT id, system_id, timestamp, data, data_hash, data_size,
		       processed_at, has_changes, change_count, created_at, updated_at
		FROM inventory_records 
		%s
		ORDER BY timestamp DESC 
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, pageSize, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query inventory history: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var records []collectModels.InventoryRecord
	for rows.Next() {
		var record collectModels.InventoryRecord
		var processedAt sql.NullTime

		err := rows.Scan(
			&record.ID, &record.SystemID, &record.Timestamp, &record.Data, &record.DataHash,
			&record.DataSize, &processedAt, &record.HasChanges,
			&record.ChangeCount, &record.CreatedAt, &record.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan inventory record: %w", err)
		}

		if processedAt.Valid {
			record.ProcessedAt = &processedAt.Time
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating inventory records: %w", err)
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
func (s *InventoryService) GetInventoryDiffs(systemID string, page, pageSize int, severity, category, diffType string, fromDate, toDate *time.Time) ([]collectModels.InventoryDiff, int, error) {
	// Build WHERE clause with filters
	whereClause := "WHERE system_id = $1"
	args := []interface{}{systemID}
	argIndex := 2

	if severity != "" {
		whereClause += fmt.Sprintf(" AND severity = $%d", argIndex)
		args = append(args, severity)
		argIndex++
	}

	if category != "" {
		whereClause += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, category)
		argIndex++
	}

	if diffType != "" {
		whereClause += fmt.Sprintf(" AND diff_type = $%d", argIndex)
		args = append(args, diffType)
		argIndex++
	}

	if fromDate != nil {
		whereClause += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *fromDate)
		argIndex++
	}

	if toDate != nil {
		whereClause += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *toDate)
		argIndex++
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM inventory_diffs %s", whereClause)
	var totalCount int
	err := s.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count inventory diffs: %w", err)
	}

	// Get paginated records
	offset := (page - 1) * pageSize
	query := fmt.Sprintf(`
		SELECT id, system_id, previous_id, current_id, diff_type, field_path,
		       previous_value, current_value, severity, category, notification_sent, created_at
		FROM inventory_diffs 
		%s
		ORDER BY created_at DESC 
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, pageSize, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query inventory diffs: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var diffs []collectModels.InventoryDiff
	for rows.Next() {
		var diff collectModels.InventoryDiff
		var previousID sql.NullInt64
		var previousValue, currentValue sql.NullString

		err := rows.Scan(
			&diff.ID, &diff.SystemID, &previousID, &diff.CurrentID, &diff.DiffType,
			&diff.FieldPath, &previousValue, &currentValue, &diff.Severity,
			&diff.Category, &diff.NotificationSent, &diff.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan inventory diff: %w", err)
		}

		if previousID.Valid {
			diff.PreviousID = &previousID.Int64
		}
		if previousValue.Valid {
			diff.PreviousValueRaw = &previousValue.String
			diff.PreviousValue = parseJSONValue(&previousValue.String)
		}
		if currentValue.Valid {
			diff.CurrentValueRaw = &currentValue.String
			diff.CurrentValue = parseJSONValue(&currentValue.String)
		}

		diffs = append(diffs, diff)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating inventory diffs: %w", err)
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

// GetChangesSummary returns a summary of changes for a system
func (s *InventoryService) GetChangesSummary(systemID string) (*InventoryChangesSummary, error) {
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
	categoryRows, err := s.db.Query(`
		SELECT category, COUNT(*) FROM inventory_diffs 
		WHERE system_id = $1 
		GROUP BY category
	`, systemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get changes by category: %w", err)
	}
	defer func() {
		_ = categoryRows.Close()
	}()

	changesByCategory := make(map[string]int)
	for categoryRows.Next() {
		var category string
		var count int
		if err := categoryRows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("failed to scan category changes: %w", err)
		}
		changesByCategory[category] = count
	}

	// Get changes by severity
	severityRows, err := s.db.Query(`
		SELECT severity, COUNT(*) FROM inventory_diffs 
		WHERE system_id = $1 
		GROUP BY severity
	`, systemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get changes by severity: %w", err)
	}
	defer func() {
		_ = severityRows.Close()
	}()

	changesBySeverity := make(map[string]int)
	for severityRows.Next() {
		var severity string
		var count int
		if err := severityRows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("failed to scan severity changes: %w", err)
		}
		changesBySeverity[severity] = count
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
func (s *InventoryService) GetLatestInventoryChangesSummary(systemID string) (*InventoryChangesSummary, error) {
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
	categoryRows, err := s.db.Query(`
		SELECT category, COUNT(*) FROM inventory_diffs 
		WHERE system_id = $1 AND current_id = $2
		GROUP BY category
	`, systemID, lastInventoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get changes by category: %w", err)
	}
	defer func() {
		_ = categoryRows.Close()
	}()

	changesByCategory := make(map[string]int)
	for categoryRows.Next() {
		var category string
		var count int
		if err := categoryRows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("failed to scan category changes: %w", err)
		}
		changesByCategory[category] = count
	}

	// Get changes by severity for this batch
	severityRows, err := s.db.Query(`
		SELECT severity, COUNT(*) FROM inventory_diffs 
		WHERE system_id = $1 AND current_id = $2
		GROUP BY severity
	`, systemID, lastInventoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get changes by severity: %w", err)
	}
	defer func() {
		_ = severityRows.Close()
	}()

	changesBySeverity := make(map[string]int)
	for severityRows.Next() {
		var severity string
		var count int
		if err := severityRows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("failed to scan severity changes: %w", err)
		}
		changesBySeverity[severity] = count
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

// GetLatestInventoryDiffs returns all diffs from the most recent inventory processing batch for a system
func (s *InventoryService) GetLatestInventoryDiffs(systemID string) ([]collectModels.InventoryDiff, error) {
	query := `
		SELECT d.id, d.system_id, d.previous_id, d.current_id, d.diff_type, d.field_path,
		       d.previous_value, d.current_value, d.severity, d.category, d.notification_sent, d.created_at
		FROM inventory_diffs d
		WHERE d.system_id = $1 
		AND d.current_id = (
			SELECT id FROM inventory_records 
			WHERE system_id = $1 
			ORDER BY timestamp DESC 
			LIMIT 1
		)
		ORDER BY 
			CASE d.severity 
				WHEN 'critical' THEN 1 
				WHEN 'high' THEN 2 
				WHEN 'medium' THEN 3 
				WHEN 'low' THEN 4 
			END,
			d.created_at DESC
	`

	rows, err := s.db.Query(query, systemID)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest inventory diffs: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var diffs []collectModels.InventoryDiff
	for rows.Next() {
		var diff collectModels.InventoryDiff
		var previousID sql.NullInt64
		var previousValue, currentValue sql.NullString

		err := rows.Scan(
			&diff.ID, &diff.SystemID, &previousID, &diff.CurrentID, &diff.DiffType,
			&diff.FieldPath, &previousValue, &currentValue, &diff.Severity,
			&diff.Category, &diff.NotificationSent, &diff.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory diff: %w", err)
		}

		if previousID.Valid {
			diff.PreviousID = &previousID.Int64
		}
		if previousValue.Valid {
			diff.PreviousValueRaw = &previousValue.String
			diff.PreviousValue = parseJSONValue(&previousValue.String)
		}
		if currentValue.Valid {
			diff.CurrentValueRaw = &currentValue.String
			diff.CurrentValue = parseJSONValue(&currentValue.String)
		}

		diffs = append(diffs, diff)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory diffs: %w", err)
	}

	// Return empty array instead of error for consistency with other APIs
	// This allows clients to handle empty results uniformly

	logEvent := logger.Debug().
		Str("system_id", systemID).
		Int("count", len(diffs))

	if len(diffs) > 0 {
		logEvent.Int64("current_id", diffs[0].CurrentID)
	}

	logEvent.Msg("Retrieved latest inventory diffs batch")

	return diffs, nil
}
