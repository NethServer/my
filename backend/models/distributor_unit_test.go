package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDistributorStructure(t *testing.T) {
	now := time.Now()
	distributor := Distributor{
		ID:          "distributor-123",
		Name:        "Test Distributor",
		Email:       "distributor@example.com",
		CompanyName: "Test Distributor Inc.",
		Status:      "active",
		Region:      "Europe",
		Territory:   []string{"Italy", "Spain", "France"},
		Metadata:    map[string]string{"tier": "platinum", "support": "24/7"},
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   "admin-789",
	}

	assert.Equal(t, "distributor-123", distributor.ID)
	assert.Equal(t, "Test Distributor", distributor.Name)
	assert.Equal(t, "distributor@example.com", distributor.Email)
	assert.Equal(t, "Test Distributor Inc.", distributor.CompanyName)
	assert.Equal(t, "active", distributor.Status)
	assert.Equal(t, "Europe", distributor.Region)
	assert.Equal(t, []string{"Italy", "Spain", "France"}, distributor.Territory)
	assert.Equal(t, map[string]string{"tier": "platinum", "support": "24/7"}, distributor.Metadata)
	assert.Equal(t, now, distributor.CreatedAt)
	assert.Equal(t, now, distributor.UpdatedAt)
	assert.Equal(t, "admin-789", distributor.CreatedBy)
}

func TestDistributorJSONSerialization(t *testing.T) {
	now := time.Now()
	distributor := Distributor{
		ID:          "json-distributor-456",
		Name:        "JSON Test Distributor",
		Email:       "jsondistributor@example.com",
		CompanyName: "JSON Distributor Corp",
		Status:      "suspended",
		Region:      "Asia Pacific",
		Territory:   []string{"Japan", "South Korea", "Australia"},
		Metadata:    map[string]string{"contract": "enterprise", "revenue": "high"},
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   "json-admin-123",
	}

	jsonData, err := json.Marshal(distributor)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledDistributor Distributor
	err = json.Unmarshal(jsonData, &unmarshaledDistributor)
	assert.NoError(t, err)

	assert.Equal(t, distributor.ID, unmarshaledDistributor.ID)
	assert.Equal(t, distributor.Name, unmarshaledDistributor.Name)
	assert.Equal(t, distributor.Email, unmarshaledDistributor.Email)
	assert.Equal(t, distributor.CompanyName, unmarshaledDistributor.CompanyName)
	assert.Equal(t, distributor.Status, unmarshaledDistributor.Status)
	assert.Equal(t, distributor.Region, unmarshaledDistributor.Region)
	assert.Equal(t, distributor.Territory, unmarshaledDistributor.Territory)
	assert.Equal(t, distributor.Metadata, unmarshaledDistributor.Metadata)
	assert.Equal(t, distributor.CreatedAt.Unix(), unmarshaledDistributor.CreatedAt.Unix())
	assert.Equal(t, distributor.UpdatedAt.Unix(), unmarshaledDistributor.UpdatedAt.Unix())
	assert.Equal(t, distributor.CreatedBy, unmarshaledDistributor.CreatedBy)
}

func TestDistributorJSONTags(t *testing.T) {
	distributor := Distributor{
		ID:          "tag-distributor",
		Name:        "Tag Distributor",
		Email:       "tagdistributor@example.com",
		CompanyName: "Tag Distributor Ltd",
		Status:      "active",
		Region:      "North America",
		Territory:   []string{"USA", "Canada", "Mexico"},
		Metadata:    map[string]string{"test": "tags"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   "tag-admin",
	}

	jsonData, err := json.Marshal(distributor)
	assert.NoError(t, err)

	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	assert.NoError(t, err)

	// Verify JSON field names match struct tags
	assert.Contains(t, jsonMap, "id")
	assert.Contains(t, jsonMap, "name")
	assert.Contains(t, jsonMap, "email")
	assert.Contains(t, jsonMap, "company_name")
	assert.Contains(t, jsonMap, "status")
	assert.Contains(t, jsonMap, "region")
	assert.Contains(t, jsonMap, "territory")
	assert.Contains(t, jsonMap, "metadata")
	assert.Contains(t, jsonMap, "created_at")
	assert.Contains(t, jsonMap, "updated_at")
	assert.Contains(t, jsonMap, "created_by")

	// Verify values
	assert.Equal(t, "tag-distributor", jsonMap["id"])
	assert.Equal(t, "Tag Distributor", jsonMap["name"])
	assert.Equal(t, "tagdistributor@example.com", jsonMap["email"])
	assert.Equal(t, "Tag Distributor Ltd", jsonMap["company_name"])
	assert.Equal(t, "active", jsonMap["status"])
	assert.Equal(t, "North America", jsonMap["region"])
	assert.Equal(t, "tag-admin", jsonMap["created_by"])
}

func TestCreateDistributorRequestStructure(t *testing.T) {
	req := CreateDistributorRequest{
		Name:          "New Distributor",
		Description:   "A new distributor organization",
		CustomData:    map[string]interface{}{"email": "newdistributor@example.com", "region": "Europe"},
		IsMfaRequired: true,
	}

	assert.Equal(t, "New Distributor", req.Name)
	assert.Equal(t, "A new distributor organization", req.Description)
	assert.Equal(t, map[string]interface{}{"email": "newdistributor@example.com", "region": "Europe"}, req.CustomData)
	assert.True(t, req.IsMfaRequired)
}

func TestCreateDistributorRequestJSONSerialization(t *testing.T) {
	req := CreateDistributorRequest{
		Name:        "JSON Create Distributor",
		Description: "JSON distributor description",
		CustomData: map[string]interface{}{
			"region":      "Asia",
			"territory":   []string{"China", "India", "Singapore"},
			"tier":        "gold",
			"established": 2020,
		},
		IsMfaRequired: false,
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReq CreateDistributorRequest
	err = json.Unmarshal(jsonData, &unmarshaledReq)
	assert.NoError(t, err)

	assert.Equal(t, req.Name, unmarshaledReq.Name)
	assert.Equal(t, req.Description, unmarshaledReq.Description)
	// Compare individual CustomData fields due to JSON type conversion
	assert.Equal(t, req.CustomData["region"], unmarshaledReq.CustomData["region"])
	assert.Equal(t, req.CustomData["tier"], unmarshaledReq.CustomData["tier"])
	assert.Equal(t, float64(2020), unmarshaledReq.CustomData["established"]) // JSON converts int to float64
	// Territory slice becomes []interface{} after JSON unmarshal
	expectedTerritory := []interface{}{"China", "India", "Singapore"}
	assert.Equal(t, expectedTerritory, unmarshaledReq.CustomData["territory"])
	assert.Equal(t, req.IsMfaRequired, unmarshaledReq.IsMfaRequired)
}

func TestUpdateDistributorRequestStructure(t *testing.T) {
	mfaRequired := true
	req := UpdateDistributorRequest{
		Name:        "Updated Distributor",
		Description: "Updated distributor description",
		CustomData: map[string]interface{}{
			"status":   "updated",
			"version":  "2.0",
			"features": []string{"advanced_analytics", "multi_region"},
		},
		IsMfaRequired: &mfaRequired,
	}

	assert.Equal(t, "Updated Distributor", req.Name)
	assert.Equal(t, "Updated distributor description", req.Description)
	assert.NotNil(t, req.IsMfaRequired)
	assert.True(t, *req.IsMfaRequired)
}

func TestUpdateDistributorRequestJSONSerialization(t *testing.T) {
	mfaRequired := false
	req := UpdateDistributorRequest{
		Name:        "JSON Update Distributor",
		Description: "JSON update description",
		CustomData: map[string]interface{}{
			"api_version": "v2",
			"capabilities": map[string]bool{
				"reseller_management": true,
				"customer_management": true,
				"reporting":           true,
			},
		},
		IsMfaRequired: &mfaRequired,
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReq UpdateDistributorRequest
	err = json.Unmarshal(jsonData, &unmarshaledReq)
	assert.NoError(t, err)

	assert.Equal(t, req.Name, unmarshaledReq.Name)
	assert.Equal(t, req.Description, unmarshaledReq.Description)
	// Compare individual CustomData fields due to JSON type conversion
	assert.Equal(t, req.CustomData["api_version"], unmarshaledReq.CustomData["api_version"])
	// Capabilities map becomes map[string]interface{} after JSON unmarshal
	expectedCapabilities := map[string]interface{}{
		"reseller_management": true,
		"customer_management": true,
		"reporting":           true,
	}
	assert.Equal(t, expectedCapabilities, unmarshaledReq.CustomData["capabilities"])
	assert.NotNil(t, unmarshaledReq.IsMfaRequired)
	assert.False(t, *unmarshaledReq.IsMfaRequired)
}

func TestDistributorStatusTypes(t *testing.T) {
	statusTypes := []string{"active", "suspended", "inactive"}

	for _, status := range statusTypes {
		t.Run("status_"+status, func(t *testing.T) {
			distributor := Distributor{
				ID:     "status-test-" + status,
				Name:   "Status Test Distributor",
				Status: status,
			}

			assert.Equal(t, status, distributor.Status)

			jsonData, err := json.Marshal(distributor)
			assert.NoError(t, err)

			var unmarshaledDistributor Distributor
			err = json.Unmarshal(jsonData, &unmarshaledDistributor)
			assert.NoError(t, err)
			assert.Equal(t, status, unmarshaledDistributor.Status)
		})
	}
}

func TestDistributorRegionTypes(t *testing.T) {
	regionTypes := []string{"North America", "Europe", "Asia Pacific", "Latin America", "Middle East", "Africa"}

	for _, region := range regionTypes {
		t.Run("region_"+region, func(t *testing.T) {
			distributor := Distributor{
				ID:     "region-test-" + region,
				Name:   "Region Test Distributor",
				Region: region,
			}

			assert.Equal(t, region, distributor.Region)

			jsonData, err := json.Marshal(distributor)
			assert.NoError(t, err)

			var unmarshaledDistributor Distributor
			err = json.Unmarshal(jsonData, &unmarshaledDistributor)
			assert.NoError(t, err)
			assert.Equal(t, region, unmarshaledDistributor.Region)
		})
	}
}

func TestDistributorTerritoryTypes(t *testing.T) {
	territoryTypes := [][]string{
		{"USA", "Canada"},
		{"Germany", "France", "Italy", "Spain"},
		{"Japan", "South Korea", "Australia", "New Zealand"},
		{"Brazil", "Argentina", "Chile"},
		{},
		nil,
	}

	for i, territory := range territoryTypes {
		t.Run("territory_combination_"+string(rune(i+'A')), func(t *testing.T) {
			distributor := Distributor{
				ID:        "territory-test-" + string(rune(i+'A')),
				Name:      "Territory Test Distributor",
				Territory: territory,
			}

			assert.Equal(t, territory, distributor.Territory)

			jsonData, err := json.Marshal(distributor)
			assert.NoError(t, err)

			var unmarshaledDistributor Distributor
			err = json.Unmarshal(jsonData, &unmarshaledDistributor)
			assert.NoError(t, err)
			assert.Equal(t, territory, unmarshaledDistributor.Territory)
		})
	}
}

func TestDistributorWithEmptyFields(t *testing.T) {
	distributor := Distributor{
		ID:          "",
		Name:        "",
		Email:       "",
		CompanyName: "",
		Status:      "",
		Region:      "",
		Territory:   []string{},
		Metadata:    map[string]string{},
		CreatedAt:   time.Time{},
		UpdatedAt:   time.Time{},
		CreatedBy:   "",
	}

	jsonData, err := json.Marshal(distributor)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledDistributor Distributor
	err = json.Unmarshal(jsonData, &unmarshaledDistributor)
	assert.NoError(t, err)

	assert.Equal(t, "", unmarshaledDistributor.ID)
	assert.Equal(t, "", unmarshaledDistributor.Name)
	assert.Equal(t, "", unmarshaledDistributor.Email)
	assert.Equal(t, "", unmarshaledDistributor.CompanyName)
	assert.Equal(t, "", unmarshaledDistributor.Status)
	assert.Equal(t, "", unmarshaledDistributor.Region)
	assert.Equal(t, []string{}, unmarshaledDistributor.Territory)
	assert.Equal(t, map[string]string{}, unmarshaledDistributor.Metadata)
	assert.Equal(t, "", unmarshaledDistributor.CreatedBy)
}

func TestDistributorWithNilFields(t *testing.T) {
	distributor := Distributor{
		ID:          "nil-fields-distributor",
		Name:        "Nil Fields Distributor",
		Email:       "nilfields@example.com",
		CompanyName: "Nil Fields Corp",
		Status:      "active",
		Region:      "Global",
		Territory:   nil,
		Metadata:    nil,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   "nil-admin",
	}

	jsonData, err := json.Marshal(distributor)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledDistributor Distributor
	err = json.Unmarshal(jsonData, &unmarshaledDistributor)
	assert.NoError(t, err)

	assert.Equal(t, "nil-fields-distributor", unmarshaledDistributor.ID)
	assert.Equal(t, "Nil Fields Distributor", unmarshaledDistributor.Name)
	assert.Nil(t, unmarshaledDistributor.Territory)
	assert.Nil(t, unmarshaledDistributor.Metadata)
}

func TestDistributorRequestWithNilMfaRequired(t *testing.T) {
	req := UpdateDistributorRequest{
		Name:          "Nil MFA Distributor",
		Description:   "Distributor with nil MFA requirement",
		CustomData:    map[string]interface{}{"test": "nil_mfa"},
		IsMfaRequired: nil,
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReq UpdateDistributorRequest
	err = json.Unmarshal(jsonData, &unmarshaledReq)
	assert.NoError(t, err)

	assert.Equal(t, "Nil MFA Distributor", unmarshaledReq.Name)
	assert.Nil(t, unmarshaledReq.IsMfaRequired)
}

func TestDistributorRequestWithNilCustomData(t *testing.T) {
	createReq := CreateDistributorRequest{
		Name:          "Nil CustomData Create",
		Description:   "Create with nil custom data",
		CustomData:    nil,
		IsMfaRequired: false,
	}

	updateReq := UpdateDistributorRequest{
		Name:          "Nil CustomData Update",
		Description:   "Update with nil custom data",
		CustomData:    nil,
		IsMfaRequired: nil,
	}

	createJson, err := json.Marshal(createReq)
	assert.NoError(t, err)
	assert.NotEmpty(t, createJson)

	updateJson, err := json.Marshal(updateReq)
	assert.NoError(t, err)
	assert.NotEmpty(t, updateJson)

	assert.Nil(t, createReq.CustomData)
	assert.Nil(t, updateReq.CustomData)
}

func TestDistributorCompleteProfile(t *testing.T) {
	now := time.Now()
	distributor := Distributor{
		ID:          "complete-distributor-profile",
		Name:        "Complete Distributor",
		Email:       "complete@distributor.com",
		CompanyName: "Complete Distributor Solutions International Ltd",
		Status:      "active",
		Region:      "Multi-Regional",
		Territory: []string{
			"United States", "Canada", "Mexico",
			"Germany", "France", "Italy", "Spain", "United Kingdom",
			"Japan", "South Korea", "Australia", "Singapore",
		},
		Metadata: map[string]string{
			"tier":               "platinum",
			"support_level":      "enterprise",
			"contract_type":      "multi_year",
			"resellers_limit":    "unlimited",
			"features":           "all_premium",
			"revenue_tier":       "top_10_percent",
			"certification":      "gold_partner",
			"specialization":     "enterprise_solutions",
			"training_completed": "advanced",
		},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "complete-admin-456",
	}

	// Verify struct integrity
	assert.NotEmpty(t, distributor.ID)
	assert.NotEmpty(t, distributor.Name)
	assert.NotEmpty(t, distributor.Email)
	assert.NotEmpty(t, distributor.CompanyName)
	assert.NotEmpty(t, distributor.Status)
	assert.NotEmpty(t, distributor.Region)
	assert.NotEmpty(t, distributor.Territory)
	assert.NotEmpty(t, distributor.Metadata)
	assert.NotEmpty(t, distributor.CreatedBy)

	// Verify JSON round-trip
	jsonData, err := json.Marshal(distributor)
	assert.NoError(t, err)

	var unmarshaledDistributor Distributor
	err = json.Unmarshal(jsonData, &unmarshaledDistributor)
	assert.NoError(t, err)

	// Verify all fields match exactly (except time precision)
	assert.Equal(t, distributor.ID, unmarshaledDistributor.ID)
	assert.Equal(t, distributor.Name, unmarshaledDistributor.Name)
	assert.Equal(t, distributor.Email, unmarshaledDistributor.Email)
	assert.Equal(t, distributor.CompanyName, unmarshaledDistributor.CompanyName)
	assert.Equal(t, distributor.Status, unmarshaledDistributor.Status)
	assert.Equal(t, distributor.Region, unmarshaledDistributor.Region)
	assert.Equal(t, distributor.Territory, unmarshaledDistributor.Territory)
	assert.Equal(t, distributor.Metadata, unmarshaledDistributor.Metadata)
	assert.Equal(t, distributor.CreatedBy, unmarshaledDistributor.CreatedBy)
}
