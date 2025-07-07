package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSystemStructure(t *testing.T) {
	now := time.Now()
	lastSeen := now.Add(-1 * time.Hour)
	system := System{
		ID:        "system-123",
		Name:      "Test System",
		Type:      "linux",
		Status:    "online",
		IPAddress: "192.168.1.100",
		Version:   "1.2.3",
		LastSeen:  lastSeen,
		Metadata:  map[string]string{"location": "datacenter1", "environment": "production"},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "admin-456",
	}

	assert.Equal(t, "system-123", system.ID)
	assert.Equal(t, "Test System", system.Name)
	assert.Equal(t, "linux", system.Type)
	assert.Equal(t, "online", system.Status)
	assert.Equal(t, "192.168.1.100", system.IPAddress)
	assert.Equal(t, "1.2.3", system.Version)
	assert.Equal(t, lastSeen, system.LastSeen)
	assert.Equal(t, map[string]string{"location": "datacenter1", "environment": "production"}, system.Metadata)
	assert.Equal(t, now, system.CreatedAt)
	assert.Equal(t, now, system.UpdatedAt)
	assert.Equal(t, "admin-456", system.CreatedBy)
}

func TestSystemJSONSerialization(t *testing.T) {
	now := time.Now()
	lastSeen := now.Add(-2 * time.Hour)
	system := System{
		ID:        "json-system-456",
		Name:      "JSON Test System",
		Type:      "windows",
		Status:    "maintenance",
		IPAddress: "10.0.0.50",
		Version:   "2.0.1",
		LastSeen:  lastSeen,
		Metadata:  map[string]string{"cluster": "web-servers", "role": "frontend"},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "json-admin-123",
	}

	jsonData, err := json.Marshal(system)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledSystem System
	err = json.Unmarshal(jsonData, &unmarshaledSystem)
	assert.NoError(t, err)

	assert.Equal(t, system.ID, unmarshaledSystem.ID)
	assert.Equal(t, system.Name, unmarshaledSystem.Name)
	assert.Equal(t, system.Type, unmarshaledSystem.Type)
	assert.Equal(t, system.Status, unmarshaledSystem.Status)
	assert.Equal(t, system.IPAddress, unmarshaledSystem.IPAddress)
	assert.Equal(t, system.Version, unmarshaledSystem.Version)
	assert.Equal(t, system.LastSeen.Unix(), unmarshaledSystem.LastSeen.Unix())
	assert.Equal(t, system.Metadata, unmarshaledSystem.Metadata)
	assert.Equal(t, system.CreatedAt.Unix(), unmarshaledSystem.CreatedAt.Unix())
	assert.Equal(t, system.UpdatedAt.Unix(), unmarshaledSystem.UpdatedAt.Unix())
	assert.Equal(t, system.CreatedBy, unmarshaledSystem.CreatedBy)
}

func TestSystemJSONTags(t *testing.T) {
	system := System{
		ID:        "tag-system",
		Name:      "Tag System",
		Type:      "linux",
		Status:    "online",
		IPAddress: "172.16.0.10",
		Version:   "3.0.0",
		LastSeen:  time.Now(),
		Metadata:  map[string]string{"test": "tags"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: "tag-admin",
	}

	jsonData, err := json.Marshal(system)
	assert.NoError(t, err)

	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	assert.NoError(t, err)

	// Verify JSON field names match struct tags
	assert.Contains(t, jsonMap, "id")
	assert.Contains(t, jsonMap, "name")
	assert.Contains(t, jsonMap, "type")
	assert.Contains(t, jsonMap, "status")
	assert.Contains(t, jsonMap, "ip_address")
	assert.Contains(t, jsonMap, "version")
	assert.Contains(t, jsonMap, "last_seen")
	assert.Contains(t, jsonMap, "metadata")
	assert.Contains(t, jsonMap, "created_at")
	assert.Contains(t, jsonMap, "updated_at")
	assert.Contains(t, jsonMap, "created_by")

	// Verify values
	assert.Equal(t, "tag-system", jsonMap["id"])
	assert.Equal(t, "Tag System", jsonMap["name"])
	assert.Equal(t, "linux", jsonMap["type"])
	assert.Equal(t, "online", jsonMap["status"])
	assert.Equal(t, "172.16.0.10", jsonMap["ip_address"])
	assert.Equal(t, "3.0.0", jsonMap["version"])
	assert.Equal(t, "tag-admin", jsonMap["created_by"])
}

func TestCreateSystemRequestStructure(t *testing.T) {
	req := CreateSystemRequest{
		Name:      "New System",
		Type:      "linux",
		IPAddress: "192.168.1.200",
		Version:   "1.0.0",
		Metadata:  map[string]string{"purpose": "testing", "owner": "dev-team"},
	}

	assert.Equal(t, "New System", req.Name)
	assert.Equal(t, "linux", req.Type)
	assert.Equal(t, "192.168.1.200", req.IPAddress)
	assert.Equal(t, "1.0.0", req.Version)
	assert.Equal(t, map[string]string{"purpose": "testing", "owner": "dev-team"}, req.Metadata)
}

func TestCreateSystemRequestJSONSerialization(t *testing.T) {
	req := CreateSystemRequest{
		Name:      "JSON Create System",
		Type:      "windows",
		IPAddress: "10.1.1.100",
		Version:   "2.1.0",
		Metadata: map[string]string{
			"environment": "staging",
			"team":        "qa",
			"backup":      "enabled",
			"monitoring":  "prometheus",
		},
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReq CreateSystemRequest
	err = json.Unmarshal(jsonData, &unmarshaledReq)
	assert.NoError(t, err)

	assert.Equal(t, req.Name, unmarshaledReq.Name)
	assert.Equal(t, req.Type, unmarshaledReq.Type)
	assert.Equal(t, req.IPAddress, unmarshaledReq.IPAddress)
	assert.Equal(t, req.Version, unmarshaledReq.Version)
	assert.Equal(t, req.Metadata, unmarshaledReq.Metadata)
}

func TestUpdateSystemRequestStructure(t *testing.T) {
	req := UpdateSystemRequest{
		Name:      "Updated System",
		Type:      "linux",
		Status:    "maintenance",
		IPAddress: "192.168.1.201",
		Version:   "1.1.0",
		Metadata:  map[string]string{"status": "updated", "patch_level": "latest"},
	}

	assert.Equal(t, "Updated System", req.Name)
	assert.Equal(t, "linux", req.Type)
	assert.Equal(t, "maintenance", req.Status)
	assert.Equal(t, "192.168.1.201", req.IPAddress)
	assert.Equal(t, "1.1.0", req.Version)
	assert.Equal(t, map[string]string{"status": "updated", "patch_level": "latest"}, req.Metadata)
}

func TestUpdateSystemRequestJSONSerialization(t *testing.T) {
	req := UpdateSystemRequest{
		Name:      "JSON Update System",
		Type:      "container",
		Status:    "online",
		IPAddress: "172.20.0.5",
		Version:   "3.2.1",
		Metadata: map[string]string{
			"orchestrator": "kubernetes",
			"namespace":    "production",
			"replicas":     "3",
			"resources":    "high",
		},
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReq UpdateSystemRequest
	err = json.Unmarshal(jsonData, &unmarshaledReq)
	assert.NoError(t, err)

	assert.Equal(t, req.Name, unmarshaledReq.Name)
	assert.Equal(t, req.Type, unmarshaledReq.Type)
	assert.Equal(t, req.Status, unmarshaledReq.Status)
	assert.Equal(t, req.IPAddress, unmarshaledReq.IPAddress)
	assert.Equal(t, req.Version, unmarshaledReq.Version)
	assert.Equal(t, req.Metadata, unmarshaledReq.Metadata)
}

func TestSystemSubscriptionStructure(t *testing.T) {
	startDate := time.Now()
	endDate := startDate.Add(365 * 24 * time.Hour) // 1 year

	subscription := SystemSubscription{
		SystemID:   "system-sub-123",
		Plan:       "enterprise",
		Status:     "active",
		StartDate:  startDate,
		EndDate:    endDate,
		Features:   []string{"backup", "monitoring", "support"},
		MaxUsers:   100,
		MaxStorage: 1099511627776, // 1TB in bytes
	}

	assert.Equal(t, "system-sub-123", subscription.SystemID)
	assert.Equal(t, "enterprise", subscription.Plan)
	assert.Equal(t, "active", subscription.Status)
	assert.Equal(t, startDate, subscription.StartDate)
	assert.Equal(t, endDate, subscription.EndDate)
	assert.Equal(t, []string{"backup", "monitoring", "support"}, subscription.Features)
	assert.Equal(t, 100, subscription.MaxUsers)
	assert.Equal(t, int64(1099511627776), subscription.MaxStorage)
}

func TestSystemSubscriptionJSONSerialization(t *testing.T) {
	startDate := time.Now()
	endDate := startDate.Add(180 * 24 * time.Hour) // 6 months

	subscription := SystemSubscription{
		SystemID:   "json-sub-456",
		Plan:       "professional",
		Status:     "expired",
		StartDate:  startDate,
		EndDate:    endDate,
		Features:   []string{"basic_monitoring", "email_support"},
		MaxUsers:   50,
		MaxStorage: 107374182400, // 100GB in bytes
	}

	jsonData, err := json.Marshal(subscription)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledSubscription SystemSubscription
	err = json.Unmarshal(jsonData, &unmarshaledSubscription)
	assert.NoError(t, err)

	assert.Equal(t, subscription.SystemID, unmarshaledSubscription.SystemID)
	assert.Equal(t, subscription.Plan, unmarshaledSubscription.Plan)
	assert.Equal(t, subscription.Status, unmarshaledSubscription.Status)
	assert.Equal(t, subscription.StartDate.Unix(), unmarshaledSubscription.StartDate.Unix())
	assert.Equal(t, subscription.EndDate.Unix(), unmarshaledSubscription.EndDate.Unix())
	assert.Equal(t, subscription.Features, unmarshaledSubscription.Features)
	assert.Equal(t, subscription.MaxUsers, unmarshaledSubscription.MaxUsers)
	assert.Equal(t, subscription.MaxStorage, unmarshaledSubscription.MaxStorage)
}

func TestSystemActionRequestStructure(t *testing.T) {
	req := SystemActionRequest{
		Force:   true,
		Options: map[string]string{"timeout": "30s", "backup": "true"},
	}

	assert.True(t, req.Force)
	assert.Equal(t, map[string]string{"timeout": "30s", "backup": "true"}, req.Options)
}

func TestSystemActionRequestJSONSerialization(t *testing.T) {
	req := SystemActionRequest{
		Force: false,
		Options: map[string]string{
			"graceful_shutdown": "true",
			"wait_time":         "10s",
			"notify_users":      "false",
			"maintenance_mode":  "true",
		},
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReq SystemActionRequest
	err = json.Unmarshal(jsonData, &unmarshaledReq)
	assert.NoError(t, err)

	assert.Equal(t, req.Force, unmarshaledReq.Force)
	assert.Equal(t, req.Options, unmarshaledReq.Options)
}

func TestSystemTypeTypes(t *testing.T) {
	systemTypes := []string{"linux", "windows", "macos", "container", "vm", "bare_metal"}

	for _, systemType := range systemTypes {
		t.Run("type_"+systemType, func(t *testing.T) {
			system := System{
				ID:   "type-test-" + systemType,
				Name: "Type Test System",
				Type: systemType,
			}

			assert.Equal(t, systemType, system.Type)

			jsonData, err := json.Marshal(system)
			assert.NoError(t, err)

			var unmarshaledSystem System
			err = json.Unmarshal(jsonData, &unmarshaledSystem)
			assert.NoError(t, err)
			assert.Equal(t, systemType, unmarshaledSystem.Type)
		})
	}
}

func TestSystemStatusTypes(t *testing.T) {
	statusTypes := []string{"online", "offline", "maintenance", "error", "unknown"}

	for _, status := range statusTypes {
		t.Run("status_"+status, func(t *testing.T) {
			system := System{
				ID:     "status-test-" + status,
				Name:   "Status Test System",
				Status: status,
			}

			assert.Equal(t, status, system.Status)

			jsonData, err := json.Marshal(system)
			assert.NoError(t, err)

			var unmarshaledSystem System
			err = json.Unmarshal(jsonData, &unmarshaledSystem)
			assert.NoError(t, err)
			assert.Equal(t, status, unmarshaledSystem.Status)
		})
	}
}

func TestSystemSubscriptionPlanTypes(t *testing.T) {
	planTypes := []string{"basic", "professional", "enterprise", "custom"}

	for _, plan := range planTypes {
		t.Run("plan_"+plan, func(t *testing.T) {
			subscription := SystemSubscription{
				SystemID: "plan-test-" + plan,
				Plan:     plan,
				Status:   "active",
			}

			assert.Equal(t, plan, subscription.Plan)

			jsonData, err := json.Marshal(subscription)
			assert.NoError(t, err)

			var unmarshaledSubscription SystemSubscription
			err = json.Unmarshal(jsonData, &unmarshaledSubscription)
			assert.NoError(t, err)
			assert.Equal(t, plan, unmarshaledSubscription.Plan)
		})
	}
}

func TestSystemSubscriptionStatusTypes(t *testing.T) {
	statusTypes := []string{"active", "expired", "suspended", "pending", "cancelled"}

	for _, status := range statusTypes {
		t.Run("subscription_status_"+status, func(t *testing.T) {
			subscription := SystemSubscription{
				SystemID: "sub-status-test-" + status,
				Plan:     "enterprise",
				Status:   status,
			}

			assert.Equal(t, status, subscription.Status)

			jsonData, err := json.Marshal(subscription)
			assert.NoError(t, err)

			var unmarshaledSubscription SystemSubscription
			err = json.Unmarshal(jsonData, &unmarshaledSubscription)
			assert.NoError(t, err)
			assert.Equal(t, status, unmarshaledSubscription.Status)
		})
	}
}

func TestSystemWithEmptyFields(t *testing.T) {
	system := System{
		ID:        "",
		Name:      "",
		Type:      "",
		Status:    "",
		IPAddress: "",
		Version:   "",
		LastSeen:  time.Time{},
		Metadata:  map[string]string{},
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
		CreatedBy: "",
	}

	jsonData, err := json.Marshal(system)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledSystem System
	err = json.Unmarshal(jsonData, &unmarshaledSystem)
	assert.NoError(t, err)

	assert.Equal(t, "", unmarshaledSystem.ID)
	assert.Equal(t, "", unmarshaledSystem.Name)
	assert.Equal(t, "", unmarshaledSystem.Type)
	assert.Equal(t, "", unmarshaledSystem.Status)
	assert.Equal(t, "", unmarshaledSystem.IPAddress)
	assert.Equal(t, "", unmarshaledSystem.Version)
	assert.Equal(t, map[string]string{}, unmarshaledSystem.Metadata)
	assert.Equal(t, "", unmarshaledSystem.CreatedBy)
}

func TestSystemWithNilMetadata(t *testing.T) {
	system := System{
		ID:        "nil-metadata-system",
		Name:      "Nil Metadata System",
		Type:      "linux",
		Status:    "online",
		IPAddress: "192.168.1.50",
		Version:   "1.0.0",
		LastSeen:  time.Now(),
		Metadata:  nil,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: "nil-admin",
	}

	jsonData, err := json.Marshal(system)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledSystem System
	err = json.Unmarshal(jsonData, &unmarshaledSystem)
	assert.NoError(t, err)

	assert.Equal(t, "nil-metadata-system", unmarshaledSystem.ID)
	assert.Equal(t, "Nil Metadata System", unmarshaledSystem.Name)
	assert.Nil(t, unmarshaledSystem.Metadata)
}

func TestSystemSubscriptionWithEmptyFeatures(t *testing.T) {
	subscription := SystemSubscription{
		SystemID:   "empty-features-sub",
		Plan:       "basic",
		Status:     "active",
		StartDate:  time.Now(),
		EndDate:    time.Now().Add(30 * 24 * time.Hour),
		Features:   []string{},
		MaxUsers:   10,
		MaxStorage: 1073741824, // 1GB
	}

	jsonData, err := json.Marshal(subscription)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledSubscription SystemSubscription
	err = json.Unmarshal(jsonData, &unmarshaledSubscription)
	assert.NoError(t, err)

	assert.Equal(t, []string{}, unmarshaledSubscription.Features)
	assert.Equal(t, "empty-features-sub", unmarshaledSubscription.SystemID)
}

func TestSystemSubscriptionWithNilFeatures(t *testing.T) {
	subscription := SystemSubscription{
		SystemID:   "nil-features-sub",
		Plan:       "trial",
		Status:     "pending",
		StartDate:  time.Now(),
		EndDate:    time.Now().Add(7 * 24 * time.Hour),
		Features:   nil,
		MaxUsers:   5,
		MaxStorage: 536870912, // 512MB
	}

	jsonData, err := json.Marshal(subscription)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledSubscription SystemSubscription
	err = json.Unmarshal(jsonData, &unmarshaledSubscription)
	assert.NoError(t, err)

	assert.Nil(t, unmarshaledSubscription.Features)
	assert.Equal(t, "nil-features-sub", unmarshaledSubscription.SystemID)
}

func TestSystemActionRequestWithEmptyOptions(t *testing.T) {
	req := SystemActionRequest{
		Force:   false,
		Options: map[string]string{},
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReq SystemActionRequest
	err = json.Unmarshal(jsonData, &unmarshaledReq)
	assert.NoError(t, err)

	assert.False(t, unmarshaledReq.Force)
	assert.Equal(t, map[string]string{}, unmarshaledReq.Options)
}

func TestSystemActionRequestWithNilOptions(t *testing.T) {
	req := SystemActionRequest{
		Force:   true,
		Options: nil,
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReq SystemActionRequest
	err = json.Unmarshal(jsonData, &unmarshaledReq)
	assert.NoError(t, err)

	assert.True(t, unmarshaledReq.Force)
	assert.Nil(t, unmarshaledReq.Options)
}

func TestSystemCompleteProfile(t *testing.T) {
	now := time.Now()
	lastSeen := now.Add(-15 * time.Minute)

	system := System{
		ID:        "complete-system-profile",
		Name:      "Complete Production System",
		Type:      "linux",
		Status:    "online",
		IPAddress: "10.0.1.50",
		Version:   "3.2.1-stable",
		LastSeen:  lastSeen,
		Metadata: map[string]string{
			"environment":       "production",
			"datacenter":        "primary",
			"rack":              "A-15",
			"cpu_cores":         "16",
			"memory_gb":         "64",
			"storage_gb":        "2048",
			"network_zone":      "dmz",
			"backup_enabled":    "true",
			"monitoring":        "prometheus",
			"logging":           "elk",
			"security_group":    "web-servers",
			"load_balancer":     "nginx",
			"ssl_enabled":       "true",
			"auto_scaling":      "enabled",
			"disaster_recovery": "configured",
		},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "complete-admin-456",
	}

	// Verify struct integrity
	assert.NotEmpty(t, system.ID)
	assert.NotEmpty(t, system.Name)
	assert.NotEmpty(t, system.Type)
	assert.NotEmpty(t, system.Status)
	assert.NotEmpty(t, system.IPAddress)
	assert.NotEmpty(t, system.Version)
	assert.NotEmpty(t, system.Metadata)
	assert.NotEmpty(t, system.CreatedBy)

	// Verify JSON round-trip
	jsonData, err := json.Marshal(system)
	assert.NoError(t, err)

	var unmarshaledSystem System
	err = json.Unmarshal(jsonData, &unmarshaledSystem)
	assert.NoError(t, err)

	// Verify all fields match exactly (except time precision)
	assert.Equal(t, system.ID, unmarshaledSystem.ID)
	assert.Equal(t, system.Name, unmarshaledSystem.Name)
	assert.Equal(t, system.Type, unmarshaledSystem.Type)
	assert.Equal(t, system.Status, unmarshaledSystem.Status)
	assert.Equal(t, system.IPAddress, unmarshaledSystem.IPAddress)
	assert.Equal(t, system.Version, unmarshaledSystem.Version)
	assert.Equal(t, system.Metadata, unmarshaledSystem.Metadata)
	assert.Equal(t, system.CreatedBy, unmarshaledSystem.CreatedBy)
}
