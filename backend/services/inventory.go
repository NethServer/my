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
	"fmt"
	"time"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/logger"
	collectModels "github.com/nethesis/my/collect/models"
)

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

// GetLatestInventory returns the most recent inventory record for a system
func (s *InventoryService) GetLatestInventory(systemID string) (*collectModels.InventoryRecord, error) {
	query := `
		SELECT id, system_id, timestamp, data, data_hash, data_size, compressed, 
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
		&record.DataSize, &record.Compressed, &processedAt, &record.HasChanges,
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
		SELECT id, system_id, timestamp, data, data_hash, data_size, compressed,
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
	defer rows.Close()
	
	var records []collectModels.InventoryRecord
	for rows.Next() {
		var record collectModels.InventoryRecord
		var processedAt sql.NullTime
		
		err := rows.Scan(
			&record.ID, &record.SystemID, &record.Timestamp, &record.Data, &record.DataHash,
			&record.DataSize, &record.Compressed, &processedAt, &record.HasChanges,
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
	defer rows.Close()
	
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
			diff.PreviousValue = &previousValue.String
		}
		if currentValue.Valid {
			diff.CurrentValue = &currentValue.String
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
func (s *InventoryService) GetChangesSummary(systemID string) (*collectModels.InventoryChangesSummary, error) {
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
	defer categoryRows.Close()
	
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
	defer severityRows.Close()
	
	changesBySeverity := make(map[string]int)
	for severityRows.Next() {
		var severity string
		var count int
		if err := severityRows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("failed to scan severity changes: %w", err)
		}
		changesBySeverity[severity] = count
	}
	
	summary := &collectModels.InventoryChangesSummary{
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