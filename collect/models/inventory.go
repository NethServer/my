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

// InventoryData represents the complete inventory payload from a system (with auto-populated fields)
type InventoryData struct {
	SystemID  string          `json:"system_id" validate:"required"`
	Timestamp time.Time       `json:"timestamp" validate:"required"`
	Data      json.RawMessage `json:"data" validate:"required"`
	ID        int64           `json:"id"`
}

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

// SystemCredentials represents the authentication credentials for a system
type SystemCredentials struct {
	SystemID   string     `json:"system_id" db:"system_id"`
	SecretHash string     `json:"secret_hash" db:"secret_hash"`
	IsActive   bool       `json:"is_active" db:"is_active"`
	LastUsed   *time.Time `json:"last_used" db:"last_used"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// QueueMessage represents a message in the processing queue
type QueueMessage struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"` // inventory, processing, notification
	SystemID    string          `json:"system_id"`
	Data        json.RawMessage `json:"data"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	CreatedAt   time.Time       `json:"created_at"`
	ProcessedAt *time.Time      `json:"processed_at"`
	Error       *string         `json:"error"`
}

// InventoryProcessingJob represents a job for processing inventory data
type InventoryProcessingJob struct {
	InventoryRecord *InventoryRecord `json:"inventory_record"`
	SystemID        string           `json:"system_id"`
	ForceProcess    bool             `json:"force_process"`
}

// InventoryAlert represents an alert triggered by inventory changes
type InventoryAlert struct {
	ID         int64      `json:"id" db:"id"`
	SystemID   string     `json:"system_id" db:"system_id"`
	DiffID     *int64     `json:"diff_id" db:"diff_id"`
	AlertType  string     `json:"alert_type" db:"alert_type"` // change, pattern
	Message    string     `json:"message" db:"message"`
	Severity   string     `json:"severity" db:"severity"`
	IsResolved bool       `json:"is_resolved" db:"is_resolved"`
	ResolvedAt *time.Time `json:"resolved_at" db:"resolved_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// NotificationJob represents a job for sending notifications
type NotificationJob struct {
	Type       string          `json:"type"` // diff, alert, system_status
	SystemID   string          `json:"system_id"`
	Diffs      []InventoryDiff `json:"diffs,omitempty"`
	Alert      *InventoryAlert `json:"alert,omitempty"`
	Message    string          `json:"message"`
	Severity   string          `json:"severity"`
	Recipients []string        `json:"recipients"`
}

// InventoryStats represents statistics about inventory processing
type InventoryStats struct {
	PendingJobs    int64  `json:"pending_jobs"`
	ProcessingJobs int64  `json:"processing_jobs"`
	FailedJobs     int64  `json:"failed_jobs"`
	QueueHealth    string `json:"queue_health"` // healthy, warning, critical
}

// ValidateInventoryData validates the inventory data payload
func (i *InventoryData) ValidateInventoryData() error {
	if i.SystemID == "" {
		return &ValidationError{Field: "system_id", Message: "system_id is required"}
	}
	if i.Timestamp.IsZero() {
		return &ValidationError{Field: "timestamp", Message: "timestamp is required"}
	}
	if len(i.Data) == 0 {
		return &ValidationError{Field: "data", Message: "data is required"}
	}

	// Validate that data is valid JSON
	var testData interface{}
	if err := json.Unmarshal(i.Data, &testData); err != nil {
		return &ValidationError{Field: "data", Message: "data must be valid JSON"}
	}

	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
}
