/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package workers

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/models"
)

// GetPreviousInventoryRecord gets the most recent previous inventory record for a system.
// Returns nil, nil when no previous record exists.
func GetPreviousInventoryRecord(ctx context.Context, systemID string, currentID int64) (*models.InventoryRecord, error) {
	query := `
		SELECT id, system_id, timestamp, data, data_hash, data_size,
		       processed_at, has_changes, change_count, created_at, updated_at
		FROM inventory_records
		WHERE system_id = $1 AND id < $2
		ORDER BY timestamp DESC, id DESC
		LIMIT 1
	`

	record := &models.InventoryRecord{}
	err := database.DB.QueryRowContext(ctx, query, systemID, currentID).Scan(
		&record.ID,
		&record.SystemID,
		&record.Timestamp,
		&record.Data,
		&record.DataHash,
		&record.DataSize,
		&record.ProcessedAt,
		&record.HasChanges,
		&record.ChangeCount,
		&record.CreatedAt,
		&record.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get previous inventory record: %w", err)
	}

	return record, nil
}

// GetInventoryRecordByID loads a full inventory record by its ID.
func GetInventoryRecordByID(ctx context.Context, id int64) (*models.InventoryRecord, error) {
	query := `
		SELECT id, system_id, timestamp, data, data_hash, data_size,
		       processed_at, has_changes, change_count, created_at, updated_at
		FROM inventory_records
		WHERE id = $1
	`

	record := &models.InventoryRecord{}
	err := database.DB.QueryRowContext(ctx, query, id).Scan(
		&record.ID,
		&record.SystemID,
		&record.Timestamp,
		&record.Data,
		&record.DataHash,
		&record.DataSize,
		&record.ProcessedAt,
		&record.HasChanges,
		&record.ChangeCount,
		&record.CreatedAt,
		&record.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("inventory record %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory record by id: %w", err)
	}

	return record, nil
}
