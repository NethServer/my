package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Helper function to create a string pointer
func strPtr(s string) *string {
	return &s
}

func TestSystemStructure(t *testing.T) {
	now := time.Now()
	creator := SystemCreator{
		UserID:           "admin-456",
		Username:         "admin",
		Name:             "Admin User",
		Email:            "admin@test.com",
		OrganizationID:   "org-123",
		OrganizationName: "Test Organization",
	}
	system := System{
		ID:          "system-123",
		Name:        "Test System",
		Type:        nil,
		Status:      "unknown",
		FQDN:        "test-system.example.com",
		IPv4Address: "192.168.1.100",
		IPv6Address: "2001:db8::1",
		Version:     "1.2.3",
		SystemKey:   "ABC123DEF456",
		Organization: Organization{
			ID:      "db-uuid-123",
			LogtoID: "org-123",
			Name:    "Test Organization",
			Type:    "owner",
		},
		CustomData: map[string]string{"location": "datacenter1", "environment": "production"},
		CreatedAt:  now,
		UpdatedAt:  now,
		CreatedBy:  creator,
	}

	assert.Equal(t, "system-123", system.ID)
	assert.Equal(t, "Test System", system.Name)
	assert.Nil(t, system.Type)
	assert.Equal(t, "unknown", system.Status)
	assert.Equal(t, "test-system.example.com", system.FQDN)
	assert.Equal(t, "192.168.1.100", system.IPv4Address)
	assert.Equal(t, "2001:db8::1", system.IPv6Address)
	assert.Equal(t, "1.2.3", system.Version)
	assert.Equal(t, "ABC123DEF456", system.SystemKey)
	assert.Equal(t, "db-uuid-123", system.Organization.ID)
	assert.Equal(t, "org-123", system.Organization.LogtoID)
	assert.Equal(t, "Test Organization", system.Organization.Name)
	assert.Equal(t, "owner", system.Organization.Type)
	assert.Equal(t, map[string]string{"location": "datacenter1", "environment": "production"}, system.CustomData)
	assert.Equal(t, now, system.CreatedAt)
	assert.Equal(t, now, system.UpdatedAt)
	assert.Equal(t, creator, system.CreatedBy)
}

func TestSystemJSONSerialization(t *testing.T) {
	now := time.Now()
	creator := SystemCreator{
		UserID:           "json-admin-123",
		Username:         "json.admin",
		Name:             "JSON Admin",
		Email:            "json.admin@test.com",
		OrganizationID:   "org-456",
		OrganizationName: "JSON Organization",
	}
	system := System{
		ID:          "json-system-456",
		Name:        "JSON Test System",
		Type:        strPtr("nsec"),
		Status:      "offline",
		FQDN:        "json-test.example.com",
		IPv4Address: "10.0.0.50",
		IPv6Address: "2001:db8::2",
		Version:     "2.0.1",
		SystemKey:   "XYZ789GHI012",
		Organization: Organization{
			ID:      "db-uuid-456",
			LogtoID: "org-456",
			Name:    "JSON Organization",
			Type:    "distributor",
		},
		CustomData: map[string]string{"cluster": "web-servers", "role": "frontend"},
		CreatedAt:  now,
		UpdatedAt:  now,
		CreatedBy:  creator,
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
	assert.Equal(t, system.SystemKey, unmarshaledSystem.SystemKey)
	assert.Equal(t, system.Organization.ID, unmarshaledSystem.Organization.ID)
	assert.Equal(t, system.Organization.LogtoID, unmarshaledSystem.Organization.LogtoID)
	assert.Equal(t, system.Organization.Name, unmarshaledSystem.Organization.Name)
	assert.Equal(t, system.Organization.Type, unmarshaledSystem.Organization.Type)
	assert.Equal(t, system.CustomData, unmarshaledSystem.CustomData)
	assert.Equal(t, system.CreatedAt.Unix(), unmarshaledSystem.CreatedAt.Unix())
	assert.Equal(t, system.UpdatedAt.Unix(), unmarshaledSystem.UpdatedAt.Unix())
	assert.Equal(t, system.CreatedBy, unmarshaledSystem.CreatedBy)
}

func TestCreateSystemRequestStructure(t *testing.T) {
	req := CreateSystemRequest{
		Name:           "New System",
		OrganizationID: "org-123",
		CustomData:     map[string]string{"purpose": "testing", "owner": "dev-team"},
	}

	assert.Equal(t, "New System", req.Name)
	assert.Equal(t, "org-123", req.OrganizationID)
	assert.Equal(t, map[string]string{"purpose": "testing", "owner": "dev-team"}, req.CustomData)
}

func TestCreateSystemRequestJSONSerialization(t *testing.T) {
	req := CreateSystemRequest{
		Name:           "JSON Create System",
		OrganizationID: "org-789",
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
	assert.Equal(t, req.OrganizationID, unmarshaledReq.OrganizationID)
	assert.Equal(t, req.CustomData, unmarshaledReq.CustomData)
}

func TestUpdateSystemRequestStructure(t *testing.T) {
	req := UpdateSystemRequest{
		Name:           "Updated System",
		OrganizationID: "org-456",
		CustomData:     map[string]string{"status": "updated", "patch_level": "latest"},
	}

	assert.Equal(t, "Updated System", req.Name)
	assert.Equal(t, "org-456", req.OrganizationID)
	assert.Equal(t, map[string]string{"status": "updated", "patch_level": "latest"}, req.CustomData)
}

func TestSystemJSONTags(t *testing.T) {
	creator := SystemCreator{
		UserID:           "tag-admin",
		Username:         "tag.admin",
		Name:             "Tag Admin",
		Email:            "tag.admin@test.com",
		OrganizationID:   "org-tags",
		OrganizationName: "Tag Organization",
	}
	system := System{
		ID:          "tag-system",
		Name:        "Tag System",
		Type:        strPtr("nsec"),
		Status:      "online",
		FQDN:        "tag-system.example.com",
		IPv4Address: "172.16.0.10",
		IPv6Address: "2001:db8::3",
		Version:     "3.0.0",
		SystemKey:   "TAG789XYZ012",
		Organization: Organization{
			ID:      "db-uuid-tags",
			LogtoID: "org-tags",
			Name:    "Tag Organization",
			Type:    "customer",
		},
		CustomData: map[string]string{"test": "tags"},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		CreatedBy:  creator,
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
	assert.Contains(t, jsonMap, "system_key")
	assert.Contains(t, jsonMap, "organization")
	assert.Contains(t, jsonMap, "custom_data")
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
	assert.Equal(t, "TAG789XYZ012", jsonMap["system_key"])

	// Verify organization is an object
	orgMap, ok := jsonMap["organization"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "db-uuid-tags", orgMap["id"])
	assert.Equal(t, "org-tags", orgMap["logto_id"])
	assert.Equal(t, "Tag Organization", orgMap["name"])
	assert.Equal(t, "customer", orgMap["type"])

	// Verify created_by is an object
	createdByMap, ok := jsonMap["created_by"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "tag-admin", createdByMap["user_id"])
	assert.Equal(t, "tag.admin", createdByMap["username"])
	assert.Equal(t, "Tag Admin", createdByMap["name"])
	assert.Equal(t, "tag.admin@test.com", createdByMap["email"])
}
