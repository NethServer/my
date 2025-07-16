/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

import (
	"encoding/json"
	"time"
)

// InventoryRecord represents a stored inventory record in the database
type InventoryRecord struct {
	ID          int64           `json:"id" db:"id"`
	SystemID    string          `json:"system_id" db:"system_id"`
	Timestamp   time.Time       `json:"timestamp" db:"timestamp"`
	Data        json.RawMessage `json:"data" db:"data"`
	DataHash    string          `json:"data_hash" db:"data_hash"`
	DataSize    int64           `json:"data_size" db:"data_size"`
	ProcessedAt *time.Time      `json:"processed_at" db:"processed_at"`
	HasChanges  bool            `json:"has_changes" db:"has_changes"`
	ChangeCount int             `json:"change_count" db:"change_count"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// InventoryDiff represents a difference between two inventory snapshots
type InventoryDiff struct {
	ID               int64       `json:"id" db:"id"`
	SystemID         string      `json:"system_id" db:"system_id"`
	PreviousID       *int64      `json:"previous_id" db:"previous_id"`
	CurrentID        int64       `json:"current_id" db:"current_id"`
	DiffType         string      `json:"diff_type" db:"diff_type"` // create, update, delete
	FieldPath        string      `json:"field_path" db:"field_path"`
	PreviousValueRaw *string     `json:"-" db:"previous_value"`  // Raw value from DB (not exported)
	CurrentValueRaw  *string     `json:"-" db:"current_value"`   // Raw value from DB (not exported)
	PreviousValue    interface{} `json:"previous_value"`         // Parsed value for JSON response
	CurrentValue     interface{} `json:"current_value"`          // Parsed value for JSON response
	Severity         string      `json:"severity" db:"severity"` // low, medium, high, critical
	Category         string      `json:"category" db:"category"` // os, hardware, network, features
	NotificationSent bool        `json:"notification_sent" db:"notification_sent"`
	CreatedAt        time.Time   `json:"created_at" db:"created_at"`
}
