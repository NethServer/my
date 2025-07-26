package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSystemHeartbeatStructure(t *testing.T) {
	now := time.Now()
	heartbeat := SystemHeartbeat{
		SystemID:      "system-123",
		LastHeartbeat: now,
	}

	assert.Equal(t, "system-123", heartbeat.SystemID)
	assert.Equal(t, now, heartbeat.LastHeartbeat)
}

func TestSystemHeartbeatJSONSerialization(t *testing.T) {
	now := time.Now()
	heartbeat := SystemHeartbeat{
		SystemID:      "json-system-456",
		LastHeartbeat: now,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(heartbeat)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var unmarshaledHeartbeat SystemHeartbeat
	err = json.Unmarshal(jsonData, &unmarshaledHeartbeat)
	assert.NoError(t, err)

	assert.Equal(t, heartbeat.SystemID, unmarshaledHeartbeat.SystemID)
	assert.WithinDuration(t, heartbeat.LastHeartbeat, unmarshaledHeartbeat.LastHeartbeat, time.Second)
}

func TestSystemHeartbeatJSONTags(t *testing.T) {
	now := time.Now()
	heartbeat := SystemHeartbeat{
		SystemID:      "tag-system-789",
		LastHeartbeat: now,
	}

	jsonData, err := json.Marshal(heartbeat)
	assert.NoError(t, err)

	// Parse JSON to verify field names match tags
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	assert.NoError(t, err)

	// Verify JSON field names match struct tags
	assert.Contains(t, jsonMap, "system_id")
	assert.Contains(t, jsonMap, "last_heartbeat")

	// Verify values
	assert.Equal(t, "tag-system-789", jsonMap["system_id"])
	assert.NotNil(t, jsonMap["last_heartbeat"])
}

func TestSystemStatusStructure(t *testing.T) {
	now := time.Now()
	minutesAgo := 5

	status := SystemStatus{
		SystemID:      "status-system-123",
		LastHeartbeat: &now,
		Status:        "alive",
		MinutesAgo:    &minutesAgo,
	}

	assert.Equal(t, "status-system-123", status.SystemID)
	assert.Equal(t, &now, status.LastHeartbeat)
	assert.Equal(t, "alive", status.Status)
	assert.Equal(t, &minutesAgo, status.MinutesAgo)
}

func TestSystemStatusJSONSerialization(t *testing.T) {
	now := time.Now()
	minutesAgo := 10

	status := SystemStatus{
		SystemID:      "json-status-system-456",
		LastHeartbeat: &now,
		Status:        "dead",
		MinutesAgo:    &minutesAgo,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(status)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var unmarshaledStatus SystemStatus
	err = json.Unmarshal(jsonData, &unmarshaledStatus)
	assert.NoError(t, err)

	assert.Equal(t, status.SystemID, unmarshaledStatus.SystemID)
	assert.WithinDuration(t, *status.LastHeartbeat, *unmarshaledStatus.LastHeartbeat, time.Second)
	assert.Equal(t, status.Status, unmarshaledStatus.Status)
	assert.Equal(t, *status.MinutesAgo, *unmarshaledStatus.MinutesAgo)
}

func TestSystemStatusWithNilFields(t *testing.T) {
	status := SystemStatus{
		SystemID:      "nil-status-system-789",
		LastHeartbeat: nil,
		Status:        "zombie",
		MinutesAgo:    nil,
	}

	// Test JSON marshaling with nil fields
	jsonData, err := json.Marshal(status)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var unmarshaledStatus SystemStatus
	err = json.Unmarshal(jsonData, &unmarshaledStatus)
	assert.NoError(t, err)

	assert.Equal(t, status.SystemID, unmarshaledStatus.SystemID)
	assert.Nil(t, unmarshaledStatus.LastHeartbeat)
	assert.Equal(t, status.Status, unmarshaledStatus.Status)
	assert.Nil(t, unmarshaledStatus.MinutesAgo)
}

func TestSystemStatusTypes(t *testing.T) {
	statusTypes := []string{"alive", "dead", "zombie"}

	for _, statusType := range statusTypes {
		t.Run("status_"+statusType, func(t *testing.T) {
			status := SystemStatus{
				SystemID: "status-test-" + statusType,
				Status:   statusType,
			}

			assert.Equal(t, statusType, status.Status)

			// Verify JSON serialization preserves status
			jsonData, err := json.Marshal(status)
			assert.NoError(t, err)

			var unmarshaledStatus SystemStatus
			err = json.Unmarshal(jsonData, &unmarshaledStatus)
			assert.NoError(t, err)
			assert.Equal(t, statusType, unmarshaledStatus.Status)
		})
	}
}

func TestSystemsStatusSummaryStructure(t *testing.T) {
	now := time.Now()
	systems := []SystemStatus{
		{
			SystemID:      "summary-system-1",
			LastHeartbeat: &now,
			Status:        "alive",
			MinutesAgo:    &[]int{2}[0],
		},
		{
			SystemID:      "summary-system-2",
			LastHeartbeat: nil,
			Status:        "dead",
			MinutesAgo:    &[]int{30}[0],
		},
	}

	summary := SystemsStatusSummary{
		TotalSystems:   10,
		AliveSystems:   7,
		DeadSystems:    2,
		ZombieSystems:  1,
		TimeoutMinutes: 15,
		Systems:        systems,
	}

	assert.Equal(t, 10, summary.TotalSystems)
	assert.Equal(t, 7, summary.AliveSystems)
	assert.Equal(t, 2, summary.DeadSystems)
	assert.Equal(t, 1, summary.ZombieSystems)
	assert.Equal(t, 15, summary.TimeoutMinutes)
	assert.Len(t, summary.Systems, 2)
	assert.Equal(t, "summary-system-1", summary.Systems[0].SystemID)
	assert.Equal(t, "summary-system-2", summary.Systems[1].SystemID)
}

func TestSystemsStatusSummaryJSONSerialization(t *testing.T) {
	now := time.Now()
	systems := []SystemStatus{
		{
			SystemID:      "json-summary-system-1",
			LastHeartbeat: &now,
			Status:        "alive",
			MinutesAgo:    &[]int{5}[0],
		},
	}

	summary := SystemsStatusSummary{
		TotalSystems:   5,
		AliveSystems:   4,
		DeadSystems:    1,
		ZombieSystems:  0,
		TimeoutMinutes: 20,
		Systems:        systems,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(summary)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var unmarshaledSummary SystemsStatusSummary
	err = json.Unmarshal(jsonData, &unmarshaledSummary)
	assert.NoError(t, err)

	assert.Equal(t, summary.TotalSystems, unmarshaledSummary.TotalSystems)
	assert.Equal(t, summary.AliveSystems, unmarshaledSummary.AliveSystems)
	assert.Equal(t, summary.DeadSystems, unmarshaledSummary.DeadSystems)
	assert.Equal(t, summary.ZombieSystems, unmarshaledSummary.ZombieSystems)
	assert.Equal(t, summary.TimeoutMinutes, unmarshaledSummary.TimeoutMinutes)
	assert.Len(t, unmarshaledSummary.Systems, 1)
	assert.Equal(t, "json-summary-system-1", unmarshaledSummary.Systems[0].SystemID)
}

func TestSystemsStatusSummaryWithEmptySystems(t *testing.T) {
	summary := SystemsStatusSummary{
		TotalSystems:   0,
		AliveSystems:   0,
		DeadSystems:    0,
		ZombieSystems:  0,
		TimeoutMinutes: 15,
		Systems:        []SystemStatus{},
	}

	// Test JSON marshaling with empty systems
	jsonData, err := json.Marshal(summary)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var unmarshaledSummary SystemsStatusSummary
	err = json.Unmarshal(jsonData, &unmarshaledSummary)
	assert.NoError(t, err)

	assert.Equal(t, 0, unmarshaledSummary.TotalSystems)
	assert.Equal(t, 0, unmarshaledSummary.AliveSystems)
	assert.Equal(t, 0, unmarshaledSummary.DeadSystems)
	assert.Equal(t, 0, unmarshaledSummary.ZombieSystems)
	assert.Equal(t, 15, unmarshaledSummary.TimeoutMinutes)
	assert.Empty(t, unmarshaledSummary.Systems)
}

func TestSystemsStatusSummaryJSONTags(t *testing.T) {
	summary := SystemsStatusSummary{
		TotalSystems:   8,
		AliveSystems:   6,
		DeadSystems:    1,
		ZombieSystems:  1,
		TimeoutMinutes: 25,
		Systems:        []SystemStatus{},
	}

	jsonData, err := json.Marshal(summary)
	assert.NoError(t, err)

	// Parse JSON to verify field names match tags
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	assert.NoError(t, err)

	// Verify JSON field names match struct tags
	assert.Contains(t, jsonMap, "total_systems")
	assert.Contains(t, jsonMap, "alive_systems")
	assert.Contains(t, jsonMap, "dead_systems")
	assert.Contains(t, jsonMap, "zombie_systems")
	assert.Contains(t, jsonMap, "timeout_minutes")
	assert.Contains(t, jsonMap, "systems")

	// Verify values
	assert.Equal(t, float64(8), jsonMap["total_systems"])
	assert.Equal(t, float64(6), jsonMap["alive_systems"])
	assert.Equal(t, float64(1), jsonMap["dead_systems"])
	assert.Equal(t, float64(1), jsonMap["zombie_systems"])
	assert.Equal(t, float64(25), jsonMap["timeout_minutes"])
}

func TestHeartbeatPointerOperations(t *testing.T) {
	now := time.Now()
	heartbeat := &SystemHeartbeat{
		SystemID:      "pointer-system",
		LastHeartbeat: now,
	}

	assert.NotNil(t, heartbeat)
	assert.Equal(t, "pointer-system", heartbeat.SystemID)
	assert.Equal(t, now, heartbeat.LastHeartbeat)

	// Test JSON serialization with pointer
	jsonData, err := json.Marshal(heartbeat)
	assert.NoError(t, err)

	var unmarshaledHeartbeat *SystemHeartbeat
	err = json.Unmarshal(jsonData, &unmarshaledHeartbeat)
	assert.NoError(t, err)
	assert.NotNil(t, unmarshaledHeartbeat)
	assert.Equal(t, heartbeat.SystemID, unmarshaledHeartbeat.SystemID)
}
