/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package entities

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// LocalInventoryRepository implements InventoryRepository for local database
type LocalInventoryRepository struct {
	db *sql.DB
}

// NewLocalInventoryRepository creates a new local inventory repository
func NewLocalInventoryRepository() *LocalInventoryRepository {
	return &LocalInventoryRepository{
		db: database.DB,
	}
}

// GetLatestInventory returns the most recent inventory record for a system
func (r *LocalInventoryRepository) GetLatestInventory(systemID string) (*models.InventoryRecord, error) {
	query := `
		SELECT id, system_id, timestamp, data, data_hash, data_size, 
		       processed_at, has_changes, change_count, created_at, updated_at
		FROM inventory_records 
		WHERE system_id = $1 
		ORDER BY timestamp DESC 
		LIMIT 1
	`

	var record models.InventoryRecord
	var processedAt sql.NullTime

	err := r.db.QueryRow(query, systemID).Scan(
		&record.ID, &record.SystemID, &record.Timestamp, &record.Data, &record.DataHash,
		&record.DataSize, &processedAt, &record.HasChanges,
		&record.ChangeCount, &record.CreatedAt, &record.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no inventory found for system %s", systemID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query latest inventory: %w", err)
	}

	if processedAt.Valid {
		record.ProcessedAt = &processedAt.Time
	}

	return &record, nil
}

// GetInventoryHistory returns paginated inventory history for a system
func (r *LocalInventoryRepository) GetInventoryHistory(systemID string, page, pageSize int, fromDate, toDate *time.Time) ([]models.InventoryRecord, int, error) {
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
	err := r.db.QueryRow(countQuery, args...).Scan(&totalCount)
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

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query inventory history: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []models.InventoryRecord
	for rows.Next() {
		var record models.InventoryRecord
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

	return records, totalCount, nil
}

// GetInventoryDiffs returns paginated diffs for a system
func (r *LocalInventoryRepository) GetInventoryDiffs(systemID string, page, pageSize int, severity, category, diffType string, fromDate, toDate *time.Time) ([]models.InventoryDiff, int, error) {
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
	err := r.db.QueryRow(countQuery, args...).Scan(&totalCount)
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

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query inventory diffs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var diffs []models.InventoryDiff
	for rows.Next() {
		var diff models.InventoryDiff
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
		}
		if currentValue.Valid {
			diff.CurrentValueRaw = &currentValue.String
		}

		diffs = append(diffs, diff)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating inventory diffs: %w", err)
	}

	return diffs, totalCount, nil
}

// GetLatestInventoryDiffs returns all diffs from the most recent inventory processing batch for a system
func (r *LocalInventoryRepository) GetLatestInventoryDiffs(systemID string) ([]models.InventoryDiff, error) {
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

	rows, err := r.db.Query(query, systemID)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest inventory diffs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var diffs []models.InventoryDiff
	for rows.Next() {
		var diff models.InventoryDiff
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
		}
		if currentValue.Valid {
			diff.CurrentValueRaw = &currentValue.String
		}

		diffs = append(diffs, diff)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory diffs: %w", err)
	}

	return diffs, nil
}
