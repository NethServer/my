package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInventoryRecordStructure(t *testing.T) {
	now := time.Now()
	processedAt := now.Add(5 * time.Minute)
	rawData := json.RawMessage(`{"cpu": "Intel i7", "memory": "16GB"}`)

	record := InventoryRecord{
		ID:          123,
		SystemID:    "system-inv-123",
		Timestamp:   now,
		Data:        rawData,
		DataHash:    "abc123def456",
		DataSize:    1024,
		ProcessedAt: &processedAt,
		HasChanges:  true,
		ChangeCount: 5,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, int64(123), record.ID)
	assert.Equal(t, "system-inv-123", record.SystemID)
	assert.Equal(t, now, record.Timestamp)
	assert.Equal(t, rawData, record.Data)
	assert.Equal(t, "abc123def456", record.DataHash)
	assert.Equal(t, int64(1024), record.DataSize)
	assert.Equal(t, &processedAt, record.ProcessedAt)
	assert.True(t, record.HasChanges)
	assert.Equal(t, 5, record.ChangeCount)
	assert.Equal(t, now, record.CreatedAt)
	assert.Equal(t, now, record.UpdatedAt)
}

func TestInventoryRecordJSONSerialization(t *testing.T) {
	now := time.Now()
	processedAt := now.Add(10 * time.Minute)
	rawData := json.RawMessage(`{"os": "Linux", "version": "5.4.0"}`)

	record := InventoryRecord{
		ID:          456,
		SystemID:    "json-system-456",
		Timestamp:   now,
		Data:        rawData,
		DataHash:    "hash456def789",
		DataSize:    2048,
		ProcessedAt: &processedAt,
		HasChanges:  false,
		ChangeCount: 0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(record)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var unmarshaledRecord InventoryRecord
	err = json.Unmarshal(jsonData, &unmarshaledRecord)
	assert.NoError(t, err)

	assert.Equal(t, record.ID, unmarshaledRecord.ID)
	assert.Equal(t, record.SystemID, unmarshaledRecord.SystemID)
	assert.WithinDuration(t, record.Timestamp, unmarshaledRecord.Timestamp, time.Second)
	// JSON data might be reformatted during marshaling/unmarshaling, so compare the actual content
	var originalData, unmarshaledData interface{}
	err = json.Unmarshal(record.Data, &originalData)
	assert.NoError(t, err)
	err = json.Unmarshal(unmarshaledRecord.Data, &unmarshaledData)
	assert.NoError(t, err)
	assert.Equal(t, originalData, unmarshaledData)
	assert.Equal(t, record.DataHash, unmarshaledRecord.DataHash)
	assert.Equal(t, record.DataSize, unmarshaledRecord.DataSize)
	assert.WithinDuration(t, *record.ProcessedAt, *unmarshaledRecord.ProcessedAt, time.Second)
	assert.Equal(t, record.HasChanges, unmarshaledRecord.HasChanges)
	assert.Equal(t, record.ChangeCount, unmarshaledRecord.ChangeCount)
	assert.WithinDuration(t, record.CreatedAt, unmarshaledRecord.CreatedAt, time.Second)
	assert.WithinDuration(t, record.UpdatedAt, unmarshaledRecord.UpdatedAt, time.Second)
}

func TestInventoryRecordWithNilProcessedAt(t *testing.T) {
	now := time.Now()
	rawData := json.RawMessage(`{"network": "192.168.1.0/24"}`)

	record := InventoryRecord{
		ID:          789,
		SystemID:    "nil-processed-system-789",
		Timestamp:   now,
		Data:        rawData,
		DataHash:    "nil789abc123",
		DataSize:    512,
		ProcessedAt: nil,
		HasChanges:  true,
		ChangeCount: 3,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Test JSON marshaling with nil ProcessedAt
	jsonData, err := json.Marshal(record)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var unmarshaledRecord InventoryRecord
	err = json.Unmarshal(jsonData, &unmarshaledRecord)
	assert.NoError(t, err)

	assert.Equal(t, record.ID, unmarshaledRecord.ID)
	assert.Equal(t, record.SystemID, unmarshaledRecord.SystemID)
	assert.Nil(t, unmarshaledRecord.ProcessedAt)
	assert.Equal(t, record.HasChanges, unmarshaledRecord.HasChanges)
	assert.Equal(t, record.ChangeCount, unmarshaledRecord.ChangeCount)
}

func TestInventoryRecordJSONTags(t *testing.T) {
	now := time.Now()
	rawData := json.RawMessage(`{"storage": "1TB SSD"}`)

	record := InventoryRecord{
		ID:          999,
		SystemID:    "tag-system-999",
		Timestamp:   now,
		Data:        rawData,
		DataHash:    "tag999hash",
		DataSize:    4096,
		ProcessedAt: nil,
		HasChanges:  false,
		ChangeCount: 0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	jsonData, err := json.Marshal(record)
	assert.NoError(t, err)

	// Parse JSON to verify field names match tags
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	assert.NoError(t, err)

	// Verify JSON field names match struct tags
	expectedFields := []string{
		"id", "system_id", "timestamp", "data", "data_hash",
		"data_size", "processed_at", "has_changes", "change_count",
		"created_at", "updated_at",
	}

	for _, field := range expectedFields {
		assert.Contains(t, jsonMap, field)
	}

	// Verify values
	assert.Equal(t, float64(999), jsonMap["id"])
	assert.Equal(t, "tag-system-999", jsonMap["system_id"])
	assert.Equal(t, "tag999hash", jsonMap["data_hash"])
	assert.Equal(t, float64(4096), jsonMap["data_size"])
	assert.Equal(t, false, jsonMap["has_changes"])
	assert.Equal(t, float64(0), jsonMap["change_count"])
}

func TestInventoryDiffStructure(t *testing.T) {
	now := time.Now()
	previousID := int64(100)
	previousValue := "old_value"
	currentValue := "new_value"

	diff := InventoryDiff{
		ID:               1,
		SystemID:         "diff-system-123",
		PreviousID:       &previousID,
		CurrentID:        200,
		DiffType:         "update",
		FieldPath:        "hardware.memory",
		PreviousValueRaw: &previousValue,
		CurrentValueRaw:  &currentValue,
		PreviousValue:    "8GB",
		CurrentValue:     "16GB",
		Severity:         "medium",
		Category:         "hardware",
		NotificationSent: false,
		CreatedAt:        now,
	}

	assert.Equal(t, int64(1), diff.ID)
	assert.Equal(t, "diff-system-123", diff.SystemID)
	assert.Equal(t, &previousID, diff.PreviousID)
	assert.Equal(t, int64(200), diff.CurrentID)
	assert.Equal(t, "update", diff.DiffType)
	assert.Equal(t, "hardware.memory", diff.FieldPath)
	assert.Equal(t, &previousValue, diff.PreviousValueRaw)
	assert.Equal(t, &currentValue, diff.CurrentValueRaw)
	assert.Equal(t, "8GB", diff.PreviousValue)
	assert.Equal(t, "16GB", diff.CurrentValue)
	assert.Equal(t, "medium", diff.Severity)
	assert.Equal(t, "hardware", diff.Category)
	assert.False(t, diff.NotificationSent)
	assert.Equal(t, now, diff.CreatedAt)
}

func TestInventoryDiffJSONSerialization(t *testing.T) {
	now := time.Now()
	previousID := int64(300)
	previousValue := "Linux 5.4"
	currentValue := "Linux 5.8"

	diff := InventoryDiff{
		ID:               2,
		SystemID:         "json-diff-system-456",
		PreviousID:       &previousID,
		CurrentID:        400,
		DiffType:         "update",
		FieldPath:        "os.version",
		PreviousValueRaw: &previousValue,
		CurrentValueRaw:  &currentValue,
		PreviousValue:    "5.4.0",
		CurrentValue:     "5.8.0",
		Severity:         "low",
		Category:         "os",
		NotificationSent: true,
		CreatedAt:        now,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(diff)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var unmarshaledDiff InventoryDiff
	err = json.Unmarshal(jsonData, &unmarshaledDiff)
	assert.NoError(t, err)

	assert.Equal(t, diff.ID, unmarshaledDiff.ID)
	assert.Equal(t, diff.SystemID, unmarshaledDiff.SystemID)
	assert.Equal(t, *diff.PreviousID, *unmarshaledDiff.PreviousID)
	assert.Equal(t, diff.CurrentID, unmarshaledDiff.CurrentID)
	assert.Equal(t, diff.DiffType, unmarshaledDiff.DiffType)
	assert.Equal(t, diff.FieldPath, unmarshaledDiff.FieldPath)
	assert.Equal(t, diff.PreviousValue, unmarshaledDiff.PreviousValue)
	assert.Equal(t, diff.CurrentValue, unmarshaledDiff.CurrentValue)
	assert.Equal(t, diff.Severity, unmarshaledDiff.Severity)
	assert.Equal(t, diff.Category, unmarshaledDiff.Category)
	assert.Equal(t, diff.NotificationSent, unmarshaledDiff.NotificationSent)
	assert.WithinDuration(t, diff.CreatedAt, unmarshaledDiff.CreatedAt, time.Second)
}

func TestInventoryDiffWithNilFields(t *testing.T) {
	now := time.Now()

	diff := InventoryDiff{
		ID:               3,
		SystemID:         "nil-diff-system-789",
		PreviousID:       nil,
		CurrentID:        500,
		DiffType:         "create",
		FieldPath:        "new.field",
		PreviousValueRaw: nil,
		CurrentValueRaw:  nil,
		PreviousValue:    nil,
		CurrentValue:     "new_value",
		Severity:         "high",
		Category:         "features",
		NotificationSent: false,
		CreatedAt:        now,
	}

	// Test JSON marshaling with nil fields
	jsonData, err := json.Marshal(diff)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var unmarshaledDiff InventoryDiff
	err = json.Unmarshal(jsonData, &unmarshaledDiff)
	assert.NoError(t, err)

	assert.Equal(t, diff.ID, unmarshaledDiff.ID)
	assert.Equal(t, diff.SystemID, unmarshaledDiff.SystemID)
	assert.Nil(t, unmarshaledDiff.PreviousID)
	assert.Equal(t, diff.CurrentID, unmarshaledDiff.CurrentID)
	assert.Equal(t, diff.DiffType, unmarshaledDiff.DiffType)
	assert.Nil(t, unmarshaledDiff.PreviousValue)
	assert.Equal(t, diff.CurrentValue, unmarshaledDiff.CurrentValue)
	assert.Equal(t, diff.Severity, unmarshaledDiff.Severity)
	assert.Equal(t, diff.Category, unmarshaledDiff.Category)
}

func TestInventoryDiffTypes(t *testing.T) {
	diffTypes := []string{"create", "update", "delete"}

	for _, diffType := range diffTypes {
		t.Run("diff_type_"+diffType, func(t *testing.T) {
			diff := InventoryDiff{
				ID:       int64(1),
				SystemID: "type-test-system",
				DiffType: diffType,
			}

			assert.Equal(t, diffType, diff.DiffType)

			// Verify JSON serialization preserves diff type
			jsonData, err := json.Marshal(diff)
			assert.NoError(t, err)

			var unmarshaledDiff InventoryDiff
			err = json.Unmarshal(jsonData, &unmarshaledDiff)
			assert.NoError(t, err)
			assert.Equal(t, diffType, unmarshaledDiff.DiffType)
		})
	}
}

func TestInventoryDiffSeverityTypes(t *testing.T) {
	severityTypes := []string{"low", "medium", "high", "critical"}

	for _, severity := range severityTypes {
		t.Run("severity_"+severity, func(t *testing.T) {
			diff := InventoryDiff{
				ID:       int64(1),
				SystemID: "severity-test-system",
				Severity: severity,
			}

			assert.Equal(t, severity, diff.Severity)

			// Verify JSON serialization preserves severity
			jsonData, err := json.Marshal(diff)
			assert.NoError(t, err)

			var unmarshaledDiff InventoryDiff
			err = json.Unmarshal(jsonData, &unmarshaledDiff)
			assert.NoError(t, err)
			assert.Equal(t, severity, unmarshaledDiff.Severity)
		})
	}
}

func TestInventoryDiffCategoryTypes(t *testing.T) {
	categoryTypes := []string{"os", "hardware", "network", "features"}

	for _, category := range categoryTypes {
		t.Run("category_"+category, func(t *testing.T) {
			diff := InventoryDiff{
				ID:       int64(1),
				SystemID: "category-test-system",
				Category: category,
			}

			assert.Equal(t, category, diff.Category)

			// Verify JSON serialization preserves category
			jsonData, err := json.Marshal(diff)
			assert.NoError(t, err)

			var unmarshaledDiff InventoryDiff
			err = json.Unmarshal(jsonData, &unmarshaledDiff)
			assert.NoError(t, err)
			assert.Equal(t, category, unmarshaledDiff.Category)
		})
	}
}

func TestInventoryDiffJSONTags(t *testing.T) {
	now := time.Now()
	diff := InventoryDiff{
		ID:               10,
		SystemID:         "tag-diff-system",
		CurrentID:        20,
		DiffType:         "update",
		FieldPath:        "test.path",
		PreviousValue:    "old",
		CurrentValue:     "new",
		Severity:         "medium",
		Category:         "network",
		NotificationSent: true,
		CreatedAt:        now,
	}

	jsonData, err := json.Marshal(diff)
	assert.NoError(t, err)

	// Parse JSON to verify field names match tags
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	assert.NoError(t, err)

	// Verify JSON field names match struct tags
	expectedFields := []string{
		"id", "system_id", "previous_id", "current_id", "diff_type",
		"field_path", "previous_value", "current_value", "severity",
		"category", "notification_sent", "created_at",
	}

	for _, field := range expectedFields {
		assert.Contains(t, jsonMap, field)
	}

	// Verify values
	assert.Equal(t, float64(10), jsonMap["id"])
	assert.Equal(t, "tag-diff-system", jsonMap["system_id"])
	assert.Equal(t, float64(20), jsonMap["current_id"])
	assert.Equal(t, "update", jsonMap["diff_type"])
	assert.Equal(t, "test.path", jsonMap["field_path"])
	assert.Equal(t, "old", jsonMap["previous_value"])
	assert.Equal(t, "new", jsonMap["current_value"])
	assert.Equal(t, "medium", jsonMap["severity"])
	assert.Equal(t, "network", jsonMap["category"])
	assert.Equal(t, true, jsonMap["notification_sent"])
}

func TestInventoryPointerOperations(t *testing.T) {
	now := time.Now()
	rawData := json.RawMessage(`{"test": "data"}`)

	record := &InventoryRecord{
		ID:        1,
		SystemID:  "pointer-system",
		Timestamp: now,
		Data:      rawData,
	}

	assert.NotNil(t, record)
	assert.Equal(t, int64(1), record.ID)
	assert.Equal(t, "pointer-system", record.SystemID)

	// Test JSON serialization with pointer
	jsonData, err := json.Marshal(record)
	assert.NoError(t, err)

	var unmarshaledRecord *InventoryRecord
	err = json.Unmarshal(jsonData, &unmarshaledRecord)
	assert.NoError(t, err)
	assert.NotNil(t, unmarshaledRecord)
	assert.Equal(t, record.ID, unmarshaledRecord.ID)
	assert.Equal(t, record.SystemID, unmarshaledRecord.SystemID)
}
