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
	"strings"
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
func (r *LocalInventoryRepository) GetInventoryDiffs(systemID string, page, pageSize int, severities, categories, diffTypes []string, fromDate, toDate *time.Time, inventoryIDs []int64) ([]models.InventoryDiff, int, error) {
	// Build WHERE clause with filters
	whereClause := "WHERE system_id = $1"
	args := []interface{}{systemID}
	argIndex := 2

	if len(inventoryIDs) > 0 {
		placeholders := make([]string, len(inventoryIDs))
		for i, id := range inventoryIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, id)
			argIndex++
		}
		whereClause += fmt.Sprintf(" AND current_id IN (%s)", strings.Join(placeholders, ", "))
	}

	if len(severities) > 0 {
		placeholders := make([]string, len(severities))
		for i, s := range severities {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, s)
			argIndex++
		}
		whereClause += fmt.Sprintf(" AND severity IN (%s)", strings.Join(placeholders, ", "))
	}

	if len(categories) > 0 {
		placeholders := make([]string, len(categories))
		for i, c := range categories {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, c)
			argIndex++
		}
		whereClause += fmt.Sprintf(" AND category IN (%s)", strings.Join(placeholders, ", "))
	}

	if len(diffTypes) > 0 {
		placeholders := make([]string, len(diffTypes))
		for i, dt := range diffTypes {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, dt)
			argIndex++
		}
		whereClause += fmt.Sprintf(" AND diff_type IN (%s)", strings.Join(placeholders, ", "))
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
			&diff.ID, &diff.SystemID, &previousID, &diff.InventoryID, &diff.DiffType,
			&diff.FieldPath, &previousValue, &currentValue, &diff.Severity,
			&diff.Category, &diff.NotificationSent, &diff.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan inventory diff: %w", err)
		}

		if previousID.Valid {
			diff.PreviousInventoryID = &previousID.Int64
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

// GetInventoryTimelineSummary returns filtered severity counts for the timeline view
func (r *LocalInventoryRepository) GetInventoryTimelineSummary(systemID string, severities, categories, diffTypes []string, fromDate, toDate *time.Time) (models.InventoryTimelineSummary, error) {
	whereClause := "WHERE system_id = $1"
	args := []interface{}{systemID}
	argIndex := 2

	if len(severities) > 0 {
		placeholders := make([]string, len(severities))
		for i, s := range severities {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, s)
			argIndex++
		}
		whereClause += fmt.Sprintf(" AND severity IN (%s)", strings.Join(placeholders, ", "))
	}
	if len(categories) > 0 {
		placeholders := make([]string, len(categories))
		for i, c := range categories {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, c)
			argIndex++
		}
		whereClause += fmt.Sprintf(" AND category IN (%s)", strings.Join(placeholders, ", "))
	}
	if len(diffTypes) > 0 {
		placeholders := make([]string, len(diffTypes))
		for i, dt := range diffTypes {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, dt)
			argIndex++
		}
		whereClause += fmt.Sprintf(" AND diff_type IN (%s)", strings.Join(placeholders, ", "))
	}
	if fromDate != nil {
		whereClause += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *fromDate)
		argIndex++
	}
	if toDate != nil {
		whereClause += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *toDate)
	}

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE severity = 'critical') as critical,
			COUNT(*) FILTER (WHERE severity = 'high') as high,
			COUNT(*) FILTER (WHERE severity = 'medium') as medium,
			COUNT(*) FILTER (WHERE severity = 'low') as low
		FROM inventory_diffs
		%s
	`, whereClause)

	var summary models.InventoryTimelineSummary
	err := r.db.QueryRow(query, args...).Scan(
		&summary.Total, &summary.Critical, &summary.High, &summary.Medium, &summary.Low,
	)
	if err != nil {
		return summary, fmt.Errorf("failed to query timeline summary: %w", err)
	}

	return summary, nil
}

// GetInventoryTimelineDateGroups returns paginated date groups with inventory and change counts
func (r *LocalInventoryRepository) GetInventoryTimelineDateGroups(systemID string, page, pageSize int, severities, categories, diffTypes []string, fromDate, toDate *time.Time) ([]models.InventoryTimelineGroup, int, error) {
	// Build WHERE clause for inventory_records (date range only)
	irWhereClause := "WHERE ir.system_id = $1"
	irArgs := []interface{}{systemID}
	irArgIndex := 2

	if fromDate != nil {
		irWhereClause += fmt.Sprintf(" AND ir.timestamp >= $%d", irArgIndex)
		irArgs = append(irArgs, *fromDate)
		irArgIndex++
	}
	if toDate != nil {
		irWhereClause += fmt.Sprintf(" AND ir.timestamp <= $%d", irArgIndex)
		irArgs = append(irArgs, *toDate)
		irArgIndex++
	}

	// Build JOIN condition for inventory_diffs (filter params)
	joinCondition := "d.current_id = ir.id AND d.system_id = ir.system_id"
	joinArgs := []interface{}{}
	joinArgIndex := irArgIndex

	if len(severities) > 0 {
		placeholders := make([]string, len(severities))
		for i, s := range severities {
			placeholders[i] = fmt.Sprintf("$%d", joinArgIndex)
			joinArgs = append(joinArgs, s)
			joinArgIndex++
		}
		joinCondition += fmt.Sprintf(" AND d.severity IN (%s)", strings.Join(placeholders, ", "))
	}
	if len(categories) > 0 {
		placeholders := make([]string, len(categories))
		for i, c := range categories {
			placeholders[i] = fmt.Sprintf("$%d", joinArgIndex)
			joinArgs = append(joinArgs, c)
			joinArgIndex++
		}
		joinCondition += fmt.Sprintf(" AND d.category IN (%s)", strings.Join(placeholders, ", "))
	}
	if len(diffTypes) > 0 {
		placeholders := make([]string, len(diffTypes))
		for i, dt := range diffTypes {
			placeholders[i] = fmt.Sprintf("$%d", joinArgIndex)
			joinArgs = append(joinArgs, dt)
			joinArgIndex++
		}
		joinCondition += fmt.Sprintf(" AND d.diff_type IN (%s)", strings.Join(placeholders, ", "))
	}

	// Count total date groups (only needs irArgs, no join filters)
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT DATE(ir.timestamp AT TIME ZONE 'UTC')::text)
		FROM inventory_records ir
		%s
	`, irWhereClause)

	var totalCount int
	if err := r.db.QueryRow(countQuery, irArgs...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count timeline date groups: %w", err)
	}

	// Build full args for main query (irArgs + joinArgs + pagination)
	allArgs := append(irArgs, joinArgs...)
	offset := (page - 1) * pageSize
	allArgs = append(allArgs, pageSize, offset)

	query := fmt.Sprintf(`
		SELECT
			DATE(ir.timestamp AT TIME ZONE 'UTC')::text as date,
			COUNT(DISTINCT ir.id) as inventory_count,
			COUNT(d.id) as change_count
		FROM inventory_records ir
		LEFT JOIN inventory_diffs d ON %s
		%s
		GROUP BY DATE(ir.timestamp AT TIME ZONE 'UTC')
		ORDER BY date DESC
		LIMIT $%d OFFSET $%d
	`, joinCondition, irWhereClause, joinArgIndex, joinArgIndex+1)

	rows, err := r.db.Query(query, allArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query timeline date groups: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var groups []models.InventoryTimelineGroup
	for rows.Next() {
		var group models.InventoryTimelineGroup
		if err := rows.Scan(&group.Date, &group.InventoryCount, &group.ChangeCount); err != nil {
			return nil, 0, fmt.Errorf("failed to scan timeline date group: %w", err)
		}
		group.InventoryIDs = []int64{}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating timeline date groups: %w", err)
	}

	return groups, totalCount, nil
}

// GetInventoryIDsForDates returns inventory record IDs grouped by date (YYYY-MM-DD) for the given dates
func (r *LocalInventoryRepository) GetInventoryIDsForDates(systemID string, dates []string) (map[string][]int64, error) {
	if len(dates) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(dates))
	args := []interface{}{systemID}
	for i, d := range dates {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, d)
	}

	query := fmt.Sprintf(`
		SELECT id, DATE(timestamp AT TIME ZONE 'UTC')::text as date
		FROM inventory_records
		WHERE system_id = $1
		AND DATE(timestamp AT TIME ZONE 'UTC')::text IN (%s)
		ORDER BY id
	`, strings.Join(placeholders, ", "))

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory IDs for dates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string][]int64)
	for rows.Next() {
		var id int64
		var date string
		if err := rows.Scan(&id, &date); err != nil {
			return nil, fmt.Errorf("failed to scan inventory ID for date: %w", err)
		}
		result[date] = append(result[date], id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory IDs for dates: %w", err)
	}

	return result, nil
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
			&diff.ID, &diff.SystemID, &previousID, &diff.InventoryID, &diff.DiffType,
			&diff.FieldPath, &previousValue, &currentValue, &diff.Severity,
			&diff.Category, &diff.NotificationSent, &diff.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory diff: %w", err)
		}

		if previousID.Valid {
			diff.PreviousInventoryID = &previousID.Int64
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
