/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package queue

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/collect/models"
)

func TestQueueManager(t *testing.T) {
	manager := NewQueueManager()
	assert.NotNil(t, manager)
}

func TestInventoryDataSerialization(t *testing.T) {
	// Test JSON serialization/deserialization of InventoryData
	now := time.Now()
	originalData := &models.InventoryData{
		SystemID:  "test-system-001",
		Timestamp: now,
		Data:      json.RawMessage(`{"cpu": "Intel i7", "memory": "16GB"}`),
		ID:        1,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(originalData)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var deserializedData models.InventoryData
	err = json.Unmarshal(jsonData, &deserializedData)
	require.NoError(t, err)

	assert.Equal(t, originalData.SystemID, deserializedData.SystemID)
	assert.Equal(t, originalData.ID, deserializedData.ID)

	// Compare JSON content by unmarshaling both
	var originalJSON, deserializedJSON interface{}
	err = json.Unmarshal(originalData.Data, &originalJSON)
	require.NoError(t, err)
	err = json.Unmarshal(deserializedData.Data, &deserializedJSON)
	require.NoError(t, err)
	assert.Equal(t, originalJSON, deserializedJSON)
}

func TestQueueMessageSerialization(t *testing.T) {
	now := time.Now()
	processedTime := now.Add(time.Minute)
	errorMsg := "processing failed"

	originalMessage := &models.QueueMessage{
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

	// Test JSON marshaling
	jsonData, err := json.Marshal(originalMessage)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var deserializedMessage models.QueueMessage
	err = json.Unmarshal(jsonData, &deserializedMessage)
	require.NoError(t, err)

	assert.Equal(t, originalMessage.ID, deserializedMessage.ID)
	assert.Equal(t, originalMessage.Type, deserializedMessage.Type)
	assert.Equal(t, originalMessage.SystemID, deserializedMessage.SystemID)

	// Compare JSON content by unmarshaling both
	var originalJSON, deserializedJSON interface{}
	err = json.Unmarshal(originalMessage.Data, &originalJSON)
	require.NoError(t, err)
	err = json.Unmarshal(deserializedMessage.Data, &deserializedJSON)
	require.NoError(t, err)
	assert.Equal(t, originalJSON, deserializedJSON)

	assert.Equal(t, originalMessage.Attempts, deserializedMessage.Attempts)
	assert.Equal(t, originalMessage.MaxAttempts, deserializedMessage.MaxAttempts)
	assert.Equal(t, originalMessage.Error, deserializedMessage.Error)
}

func TestInventoryProcessingJobSerialization(t *testing.T) {
	now := time.Now()
	record := &models.InventoryRecord{
		ID:        1,
		SystemID:  "test-system-001",
		Timestamp: now,
		Data:      json.RawMessage(`{"cpu": "Intel i7"}`),
	}

	originalJob := &models.InventoryProcessingJob{
		InventoryRecord: record,
		SystemID:        "test-system-001",
		ForceProcess:    true,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(originalJob)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var deserializedJob models.InventoryProcessingJob
	err = json.Unmarshal(jsonData, &deserializedJob)
	require.NoError(t, err)

	assert.Equal(t, originalJob.SystemID, deserializedJob.SystemID)
	assert.Equal(t, originalJob.ForceProcess, deserializedJob.ForceProcess)
	assert.Equal(t, originalJob.InventoryRecord.ID, deserializedJob.InventoryRecord.ID)
	assert.Equal(t, originalJob.InventoryRecord.SystemID, deserializedJob.InventoryRecord.SystemID)

	// Compare JSON content by unmarshaling both
	var originalJSON, deserializedJSON interface{}
	err = json.Unmarshal(originalJob.InventoryRecord.Data, &originalJSON)
	require.NoError(t, err)
	err = json.Unmarshal(deserializedJob.InventoryRecord.Data, &deserializedJSON)
	require.NoError(t, err)
	assert.Equal(t, originalJSON, deserializedJSON)
}

func TestNotificationJobSerialization(t *testing.T) {
	diffs := []models.InventoryDiff{
		{
			ID:       1,
			SystemID: "test-system-001",
			DiffType: "update",
		},
	}

	alert := &models.InventoryAlert{
		ID:        1,
		SystemID:  "test-system-001",
		AlertType: "change",
		Message:   "CPU model changed",
		Severity:  "medium",
	}

	originalJob := &models.NotificationJob{
		Type:       "diff",
		SystemID:   "test-system-001",
		Diffs:      diffs,
		Alert:      alert,
		Message:    "System inventory changed",
		Severity:   "medium",
		Recipients: []string{"admin@example.com", "support@example.com"},
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(originalJob)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var deserializedJob models.NotificationJob
	err = json.Unmarshal(jsonData, &deserializedJob)
	require.NoError(t, err)

	assert.Equal(t, originalJob.Type, deserializedJob.Type)
	assert.Equal(t, originalJob.SystemID, deserializedJob.SystemID)
	assert.Equal(t, originalJob.Message, deserializedJob.Message)
	assert.Equal(t, originalJob.Severity, deserializedJob.Severity)
	assert.Equal(t, originalJob.Recipients, deserializedJob.Recipients)
	assert.Equal(t, len(originalJob.Diffs), len(deserializedJob.Diffs))
	assert.Equal(t, originalJob.Alert.ID, deserializedJob.Alert.ID)
	assert.Equal(t, originalJob.Alert.SystemID, deserializedJob.Alert.SystemID)
}

func TestInventoryStatsStructure(t *testing.T) {
	stats := &models.InventoryStats{
		PendingJobs:    10,
		ProcessingJobs: 5,
		FailedJobs:     2,
		QueueHealth:    "healthy",
	}

	assert.Equal(t, int64(10), stats.PendingJobs)
	assert.Equal(t, int64(5), stats.ProcessingJobs)
	assert.Equal(t, int64(2), stats.FailedJobs)
	assert.Equal(t, "healthy", stats.QueueHealth)

	// Test JSON serialization
	jsonData, err := json.Marshal(stats)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON deserialization
	var deserializedStats models.InventoryStats
	err = json.Unmarshal(jsonData, &deserializedStats)
	require.NoError(t, err)

	assert.Equal(t, stats.PendingJobs, deserializedStats.PendingJobs)
	assert.Equal(t, stats.ProcessingJobs, deserializedStats.ProcessingJobs)
	assert.Equal(t, stats.FailedJobs, deserializedStats.FailedJobs)
	assert.Equal(t, stats.QueueHealth, deserializedStats.QueueHealth)
}

func TestContextTimeout(t *testing.T) {
	// Test context timeout handling
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Verify timeout is set
	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	assert.True(t, time.Until(deadline) <= 1*time.Second)

	// Test context cancellation
	select {
	case <-ctx.Done():
		assert.Error(t, ctx.Err())
	case <-time.After(2 * time.Second):
		t.Error("Context should have timed out")
	}
}

func TestQueueManagerCreation(t *testing.T) {
	tests := []struct {
		name     string
		creation func() *QueueManager
	}{
		{
			name:     "NewQueueManager",
			creation: NewQueueManager,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := tt.creation()
			assert.NotNil(t, manager)
		})
	}
}

func TestInventoryDataValidation(t *testing.T) {
	tests := []struct {
		name        string
		data        *models.InventoryData
		expectError bool
	}{
		{
			name: "valid data",
			data: &models.InventoryData{
				SystemID:  "test-system-001",
				Timestamp: time.Now(),
				Data:      json.RawMessage(`{"cpu": "Intel i7"}`),
			},
			expectError: false,
		},
		{
			name: "empty system id",
			data: &models.InventoryData{
				SystemID:  "",
				Timestamp: time.Now(),
				Data:      json.RawMessage(`{"cpu": "Intel i7"}`),
			},
			expectError: true,
		},
		{
			name: "zero timestamp",
			data: &models.InventoryData{
				SystemID:  "test-system-001",
				Timestamp: time.Time{},
				Data:      json.RawMessage(`{"cpu": "Intel i7"}`),
			},
			expectError: true,
		},
		{
			name: "empty data",
			data: &models.InventoryData{
				SystemID:  "test-system-001",
				Timestamp: time.Now(),
				Data:      json.RawMessage(``),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.data.ValidateInventoryData()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestQueueDataTypes(t *testing.T) {
	// Test that queue-related data types can be properly handled
	queueTypes := []string{"inventory", "processing", "notification"}

	for _, queueType := range queueTypes {
		t.Run(queueType, func(t *testing.T) {
			message := &models.QueueMessage{
				ID:          "msg-001",
				Type:        queueType,
				SystemID:    "test-system-001",
				Data:        json.RawMessage(`{"test": "data"}`),
				Attempts:    0,
				MaxAttempts: 3,
				CreatedAt:   time.Now(),
			}

			assert.Equal(t, queueType, message.Type)
			assert.True(t, message.Attempts >= 0)
			assert.True(t, message.MaxAttempts > 0)
			assert.False(t, message.CreatedAt.IsZero())
		})
	}
}

func TestQueueMessageAttempts(t *testing.T) {
	message := &models.QueueMessage{
		ID:          "msg-001",
		Type:        "inventory",
		SystemID:    "test-system-001",
		Data:        json.RawMessage(`{"test": "data"}`),
		Attempts:    0,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
	}

	// Test attempt increment
	assert.Equal(t, 0, message.Attempts)
	message.Attempts++
	assert.Equal(t, 1, message.Attempts)

	// Test max attempts check
	assert.True(t, message.Attempts < message.MaxAttempts)

	// Simulate reaching max attempts
	message.Attempts = message.MaxAttempts
	assert.Equal(t, message.Attempts, message.MaxAttempts)
}

func TestQueueMessageError(t *testing.T) {
	message := &models.QueueMessage{
		ID:          "msg-001",
		Type:        "inventory",
		SystemID:    "test-system-001",
		Data:        json.RawMessage(`{"test": "data"}`),
		Attempts:    0,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
	}

	// Initially no error
	assert.Nil(t, message.Error)

	// Set error
	errorMsg := "processing failed"
	message.Error = &errorMsg
	assert.NotNil(t, message.Error)
	assert.Equal(t, errorMsg, *message.Error)

	// Clear error
	message.Error = nil
	assert.Nil(t, message.Error)
}
