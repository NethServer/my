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
	creator := SystemCreator{
		UserID:           "admin-456",
		UserName:         "Admin User",
		OrganizationID:   "org-123",
		OrganizationName: "Test Organization",
	}
	system := System{
		ID:          "system-123",
		Name:        "Test System",
		Type:        "ns8",
		Status:      "online",
		FQDN:        "test-system.example.com",
		IPv4Address: "192.168.1.100",
		IPv6Address: "2001:db8::1",
		Version:     "1.2.3",
		LastSeen:    lastSeen,
		CustomData:  map[string]string{"location": "datacenter1", "environment": "production"},
		CustomerID:  "customer-123",
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   creator,
	}

	assert.Equal(t, "system-123", system.ID)
	assert.Equal(t, "Test System", system.Name)
	assert.Equal(t, "ns8", system.Type)
	assert.Equal(t, "online", system.Status)
	assert.Equal(t, "test-system.example.com", system.FQDN)
	assert.Equal(t, "192.168.1.100", system.IPv4Address)
	assert.Equal(t, "2001:db8::1", system.IPv6Address)
	assert.Equal(t, "1.2.3", system.Version)
	assert.Equal(t, lastSeen, system.LastSeen)
	assert.Equal(t, map[string]string{"location": "datacenter1", "environment": "production"}, system.CustomData)
	assert.Equal(t, "customer-123", system.CustomerID)
	assert.Equal(t, now, system.CreatedAt)
	assert.Equal(t, now, system.UpdatedAt)
	assert.Equal(t, creator, system.CreatedBy)
}

func TestSystemJSONSerialization(t *testing.T) {
	now := time.Now()
	lastSeen := now.Add(-2 * time.Hour)
	creator := SystemCreator{
		UserID:           "json-admin-123",
		UserName:         "JSON Admin",
		OrganizationID:   "org-456",
		OrganizationName: "JSON Organization",
	}
	system := System{
		ID:          "json-system-456",
		Name:        "JSON Test System",
		Type:        "nsec",
		Status:      "maintenance",
		FQDN:        "json-test.example.com",
		IPv4Address: "10.0.0.50",
		IPv6Address: "2001:db8::2",
		Version:     "2.0.1",
		LastSeen:    lastSeen,
		CustomData:  map[string]string{"cluster": "web-servers", "role": "frontend"},
		CustomerID:  "customer-456",
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   creator,
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
	assert.Equal(t, system.FQDN, unmarshaledSystem.FQDN)
	assert.Equal(t, system.IPv4Address, unmarshaledSystem.IPv4Address)
	assert.Equal(t, system.IPv6Address, unmarshaledSystem.IPv6Address)
	assert.Equal(t, system.Version, unmarshaledSystem.Version)
	assert.Equal(t, system.LastSeen.Unix(), unmarshaledSystem.LastSeen.Unix())
	assert.Equal(t, system.CustomData, unmarshaledSystem.CustomData)
	assert.Equal(t, system.CustomerID, unmarshaledSystem.CustomerID)
	assert.Equal(t, system.CreatedAt.Unix(), unmarshaledSystem.CreatedAt.Unix())
	assert.Equal(t, system.UpdatedAt.Unix(), unmarshaledSystem.UpdatedAt.Unix())
	assert.Equal(t, system.CreatedBy, unmarshaledSystem.CreatedBy)
}

func TestCreateSystemRequestStructure(t *testing.T) {
	req := CreateSystemRequest{
		Name:       "New System",
		Type:       "ns8",
		CustomerID: "customer-789",
		CustomData: map[string]string{"purpose": "testing", "owner": "dev-team"},
	}

	assert.Equal(t, "New System", req.Name)
	assert.Equal(t, "ns8", req.Type)
	assert.Equal(t, "customer-789", req.CustomerID)
	assert.Equal(t, map[string]string{"purpose": "testing", "owner": "dev-team"}, req.CustomData)
}

func TestCreateSystemRequestJSONSerialization(t *testing.T) {
	req := CreateSystemRequest{
		Name:       "JSON Create System",
		Type:       "nsec",
		CustomerID: "customer-json-123",
		CustomData: map[string]string{
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
	assert.Equal(t, req.CustomerID, unmarshaledReq.CustomerID)
	assert.Equal(t, req.CustomData, unmarshaledReq.CustomData)
}

func TestUpdateSystemRequestStructure(t *testing.T) {
	req := UpdateSystemRequest{
		Name:       "Updated System",
		Type:       "ns8",
		CustomerID: "customer-updated-123",
		CustomData: map[string]string{"status": "updated", "patch_level": "latest"},
	}

	assert.Equal(t, "Updated System", req.Name)
	assert.Equal(t, "ns8", req.Type)
	assert.Equal(t, "customer-updated-123", req.CustomerID)
	assert.Equal(t, map[string]string{"status": "updated", "patch_level": "latest"}, req.CustomData)
}

func TestSystemJSONTags(t *testing.T) {
	creator := SystemCreator{
		UserID:           "tag-admin",
		UserName:         "Tag Admin",
		OrganizationID:   "org-tags",
		OrganizationName: "Tag Organization",
	}
	system := System{
		ID:          "tag-system",
		Name:        "Tag System",
		Type:        "nsec",
		Status:      "online",
		FQDN:        "tag-system.example.com",
		IPv4Address: "172.16.0.10",
		IPv6Address: "2001:db8::3",
		Version:     "3.0.0",
		LastSeen:    time.Now(),
		CustomData:  map[string]string{"test": "tags"},
		CustomerID:  "customer-tags",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   creator,
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
	assert.Contains(t, jsonMap, "fqdn")
	assert.Contains(t, jsonMap, "ipv4_address")
	assert.Contains(t, jsonMap, "ipv6_address")
	assert.Contains(t, jsonMap, "version")
	assert.Contains(t, jsonMap, "last_seen")
	assert.Contains(t, jsonMap, "custom_data")
	assert.Contains(t, jsonMap, "customer_id")
	assert.Contains(t, jsonMap, "created_at")
	assert.Contains(t, jsonMap, "updated_at")
	assert.Contains(t, jsonMap, "created_by")

	// Verify values
	assert.Equal(t, "tag-system", jsonMap["id"])
	assert.Equal(t, "Tag System", jsonMap["name"])
	assert.Equal(t, "nsec", jsonMap["type"])
	assert.Equal(t, "online", jsonMap["status"])
	assert.Equal(t, "tag-system.example.com", jsonMap["fqdn"])
	assert.Equal(t, "172.16.0.10", jsonMap["ipv4_address"])
	assert.Equal(t, "2001:db8::3", jsonMap["ipv6_address"])
	assert.Equal(t, "3.0.0", jsonMap["version"])
	assert.Equal(t, "customer-tags", jsonMap["customer_id"])

	// Verify created_by is an object
	createdByMap, ok := jsonMap["created_by"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "tag-admin", createdByMap["user_id"])
	assert.Equal(t, "Tag Admin", createdByMap["user_name"])
}
