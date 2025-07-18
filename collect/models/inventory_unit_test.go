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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInventoryDataValidation(t *testing.T) {
	tests := []struct {
		name        string
		data        InventoryData
		expectError bool
		errorField  string
	}{
		{
			name: "valid inventory data",
			data: InventoryData{
				SystemID:  "test-system-001",
				Timestamp: time.Now(),
				Data:      json.RawMessage(`{"cpu": "Intel i7", "memory": "16GB"}`),
			},
			expectError: false,
		},
		{
			name: "empty system id",
			data: InventoryData{
				SystemID:  "",
				Timestamp: time.Now(),
				Data:      json.RawMessage(`{"cpu": "Intel i7"}`),
			},
			expectError: true,
			errorField:  "system_id",
		},
		{
			name: "zero timestamp",
			data: InventoryData{
				SystemID:  "test-system-001",
				Timestamp: time.Time{},
				Data:      json.RawMessage(`{"cpu": "Intel i7"}`),
			},
			expectError: true,
			errorField:  "timestamp",
		},
		{
			name: "empty data",
			data: InventoryData{
				SystemID:  "test-system-001",
				Timestamp: time.Now(),
				Data:      json.RawMessage(``),
			},
			expectError: true,
			errorField:  "data",
		},
		{
			name: "invalid json data",
			data: InventoryData{
				SystemID:  "test-system-001",
				Timestamp: time.Now(),
				Data:      json.RawMessage(`{"cpu": "Intel i7", "memory":}`),
			},
			expectError: true,
			errorField:  "data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.data.ValidateInventoryData()
			if tt.expectError {
				require.Error(t, err)
				if validationErr, ok := err.(*ValidationError); ok {
					assert.Equal(t, tt.errorField, validationErr.Field)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInventorySubmissionRequest(t *testing.T) {
	validJSON := json.RawMessage(`{"cpu": "Intel i7", "memory": "16GB", "disk": "1TB SSD"}`)

	request := InventorySubmissionRequest{
		Data: validJSON,
	}

	assert.NotEmpty(t, request.Data)

	// Test JSON marshaling/unmarshaling
	jsonData, err := json.Marshal(request)
	require.NoError(t, err)

	var unmarshaledRequest InventorySubmissionRequest
	err = json.Unmarshal(jsonData, &unmarshaledRequest)
	require.NoError(t, err)

	// Validate that both are valid JSON with same content
	var originalJSON, unmarshaledJSON interface{}
	err = json.Unmarshal(request.Data, &originalJSON)
	require.NoError(t, err)
	err = json.Unmarshal(unmarshaledRequest.Data, &unmarshaledJSON)
	require.NoError(t, err)
	assert.Equal(t, originalJSON, unmarshaledJSON)
}

func TestInventoryRecord(t *testing.T) {
	now := time.Now()
	processedTime := now.Add(time.Minute)

	record := InventoryRecord{
		ID:          1,
		SystemID:    "test-system-001",
		Timestamp:   now,
		Data:        json.RawMessage(`{"cpu": "Intel i7"}`),
		DataHash:    "abc123",
		DataSize:    100,
		ProcessedAt: &processedTime,
		HasChanges:  true,
		ChangeCount: 5,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, int64(1), record.ID)
	assert.Equal(t, "test-system-001", record.SystemID)
	assert.Equal(t, now, record.Timestamp)
	assert.Equal(t, json.RawMessage(`{"cpu": "Intel i7"}`), record.Data)
	assert.Equal(t, "abc123", record.DataHash)
	assert.Equal(t, int64(100), record.DataSize)
	assert.Equal(t, &processedTime, record.ProcessedAt)
	assert.True(t, record.HasChanges)
	assert.Equal(t, 5, record.ChangeCount)
	assert.Equal(t, now, record.CreatedAt)
	assert.Equal(t, now, record.UpdatedAt)
}

func TestInventoryDiff(t *testing.T) {
	now := time.Now()
	previousID := int64(1)

	diff := InventoryDiff{
		ID:               2,
		SystemID:         "test-system-001",
		PreviousID:       &previousID,
		CurrentID:        2,
		DiffType:         "update",
		FieldPath:        "cpu.model",
		PreviousValueRaw: stringPtr("Intel i5"),
		CurrentValueRaw:  stringPtr("Intel i7"),
		PreviousValue:    "Intel i5",
		CurrentValue:     "Intel i7",
		Severity:         "medium",
		Category:         "hardware",
		NotificationSent: false,
		CreatedAt:        now,
	}

	assert.Equal(t, int64(2), diff.ID)
	assert.Equal(t, "test-system-001", diff.SystemID)
	assert.Equal(t, &previousID, diff.PreviousID)
	assert.Equal(t, int64(2), diff.CurrentID)
	assert.Equal(t, "update", diff.DiffType)
	assert.Equal(t, "cpu.model", diff.FieldPath)
	assert.Equal(t, "Intel i5", *diff.PreviousValueRaw)
	assert.Equal(t, "Intel i7", *diff.CurrentValueRaw)
	assert.Equal(t, "Intel i5", diff.PreviousValue)
	assert.Equal(t, "Intel i7", diff.CurrentValue)
	assert.Equal(t, "medium", diff.Severity)
	assert.Equal(t, "hardware", diff.Category)
	assert.False(t, diff.NotificationSent)
	assert.Equal(t, now, diff.CreatedAt)
}

func TestSystemCredentials(t *testing.T) {
	now := time.Now()
	lastUsed := now.Add(-time.Hour)

	creds := SystemCredentials{
		SystemID:   "test-system-001",
		SecretHash: "hashed-secret-123",
		IsActive:   true,
		LastUsed:   &lastUsed,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	assert.Equal(t, "test-system-001", creds.SystemID)
	assert.Equal(t, "hashed-secret-123", creds.SecretHash)
	assert.True(t, creds.IsActive)
	assert.Equal(t, &lastUsed, creds.LastUsed)
	assert.Equal(t, now, creds.CreatedAt)
	assert.Equal(t, now, creds.UpdatedAt)
}

func TestQueueMessage(t *testing.T) {
	now := time.Now()
	processedTime := now.Add(time.Minute)
	errorMsg := "processing failed"

	message := QueueMessage{
		ID:          "msg-001",
		Type:        "inventory",
		SystemID:    "test-system-001",
		Data:        json.RawMessage(`{"cpu": "Intel i7"}`),
		Attempts:    2,
		MaxAttempts: 3,
		CreatedAt:   now,
		ProcessedAt: &processedTime,
		Error:       &errorMsg,
	}

	assert.Equal(t, "msg-001", message.ID)
	assert.Equal(t, "inventory", message.Type)
	assert.Equal(t, "test-system-001", message.SystemID)
	assert.Equal(t, json.RawMessage(`{"cpu": "Intel i7"}`), message.Data)
	assert.Equal(t, 2, message.Attempts)
	assert.Equal(t, 3, message.MaxAttempts)
	assert.Equal(t, now, message.CreatedAt)
	assert.Equal(t, &processedTime, message.ProcessedAt)
	assert.Equal(t, &errorMsg, message.Error)
}

func TestInventoryProcessingJob(t *testing.T) {
	now := time.Now()
	record := &InventoryRecord{
		ID:        1,
		SystemID:  "test-system-001",
		Timestamp: now,
		Data:      json.RawMessage(`{"cpu": "Intel i7"}`),
	}

	job := InventoryProcessingJob{
		InventoryRecord: record,
		SystemID:        "test-system-001",
		ForceProcess:    true,
	}

	assert.Equal(t, record, job.InventoryRecord)
	assert.Equal(t, "test-system-001", job.SystemID)
	assert.True(t, job.ForceProcess)
}

func TestInventoryAlert(t *testing.T) {
	now := time.Now()
	resolvedTime := now.Add(time.Hour)
	diffID := int64(1)

	alert := InventoryAlert{
		ID:         1,
		SystemID:   "test-system-001",
		DiffID:     &diffID,
		AlertType:  "change",
		Message:    "CPU model changed",
		Severity:   "medium",
		IsResolved: true,
		ResolvedAt: &resolvedTime,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	assert.Equal(t, int64(1), alert.ID)
	assert.Equal(t, "test-system-001", alert.SystemID)
	assert.Equal(t, &diffID, alert.DiffID)
	assert.Equal(t, "change", alert.AlertType)
	assert.Equal(t, "CPU model changed", alert.Message)
	assert.Equal(t, "medium", alert.Severity)
	assert.True(t, alert.IsResolved)
	assert.Equal(t, &resolvedTime, alert.ResolvedAt)
	assert.Equal(t, now, alert.CreatedAt)
	assert.Equal(t, now, alert.UpdatedAt)
}

func TestNotificationJob(t *testing.T) {
	diffs := []InventoryDiff{
		{
			ID:       1,
			SystemID: "test-system-001",
			DiffType: "update",
		},
	}

	alert := &InventoryAlert{
		ID:        1,
		SystemID:  "test-system-001",
		AlertType: "change",
		Message:   "CPU model changed",
		Severity:  "medium",
	}

	job := NotificationJob{
		Type:       "diff",
		SystemID:   "test-system-001",
		Diffs:      diffs,
		Alert:      alert,
		Message:    "System inventory changed",
		Severity:   "medium",
		Recipients: []string{"admin@example.com", "support@example.com"},
	}

	assert.Equal(t, "diff", job.Type)
	assert.Equal(t, "test-system-001", job.SystemID)
	assert.Equal(t, diffs, job.Diffs)
	assert.Equal(t, alert, job.Alert)
	assert.Equal(t, "System inventory changed", job.Message)
	assert.Equal(t, "medium", job.Severity)
	assert.Equal(t, []string{"admin@example.com", "support@example.com"}, job.Recipients)
}

func TestInventoryStats(t *testing.T) {
	stats := InventoryStats{
		PendingJobs:    10,
		ProcessingJobs: 5,
		FailedJobs:     2,
		QueueHealth:    "healthy",
	}

	assert.Equal(t, int64(10), stats.PendingJobs)
	assert.Equal(t, int64(5), stats.ProcessingJobs)
	assert.Equal(t, int64(2), stats.FailedJobs)
	assert.Equal(t, "healthy", stats.QueueHealth)
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "test_field",
		Message: "test error message",
	}

	assert.Equal(t, "test_field", err.Field)
	assert.Equal(t, "test error message", err.Message)
	assert.Equal(t, "test error message", err.Error())
}

func TestInventoryDataJSONSerialization(t *testing.T) {
	now := time.Now()
	data := InventoryData{
		SystemID:  "test-system-001",
		Timestamp: now,
		Data:      json.RawMessage(`{"cpu": "Intel i7", "memory": "16GB"}`),
		ID:        1,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(data)
	require.NoError(t, err)

	// Test JSON unmarshaling
	var unmarshaledData InventoryData
	err = json.Unmarshal(jsonData, &unmarshaledData)
	require.NoError(t, err)

	assert.Equal(t, data.SystemID, unmarshaledData.SystemID)
	assert.Equal(t, data.ID, unmarshaledData.ID)

	// Compare JSON content by unmarshaling both
	var originalJSON, unmarshaledJSON interface{}
	err = json.Unmarshal(data.Data, &originalJSON)
	require.NoError(t, err)
	err = json.Unmarshal(unmarshaledData.Data, &unmarshaledJSON)
	require.NoError(t, err)
	assert.Equal(t, originalJSON, unmarshaledJSON)
}

// Helper function for tests
func stringPtr(s string) *string {
	return &s
}
