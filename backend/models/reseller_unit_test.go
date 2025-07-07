package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResellerStructure(t *testing.T) {
	now := time.Now()
	reseller := Reseller{
		ID:          "reseller-123",
		Name:        "Test Reseller",
		Email:       "reseller@example.com",
		CompanyName: "Test Reseller Inc.",
		Status:      "active",
		Region:      "North America",
		Metadata:    map[string]string{"tier": "gold", "specialization": "SMB"},
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   "distributor-456",
	}

	assert.Equal(t, "reseller-123", reseller.ID)
	assert.Equal(t, "Test Reseller", reseller.Name)
	assert.Equal(t, "reseller@example.com", reseller.Email)
	assert.Equal(t, "Test Reseller Inc.", reseller.CompanyName)
	assert.Equal(t, "active", reseller.Status)
	assert.Equal(t, "North America", reseller.Region)
	assert.Equal(t, map[string]string{"tier": "gold", "specialization": "SMB"}, reseller.Metadata)
	assert.Equal(t, now, reseller.CreatedAt)
	assert.Equal(t, now, reseller.UpdatedAt)
	assert.Equal(t, "distributor-456", reseller.CreatedBy)
}

func TestResellerJSONSerialization(t *testing.T) {
	now := time.Now()
	reseller := Reseller{
		ID:          "json-reseller-456",
		Name:        "JSON Test Reseller",
		Email:       "jsonreseller@example.com",
		CompanyName: "JSON Reseller Corp",
		Status:      "suspended",
		Region:      "Europe",
		Metadata:    map[string]string{"contract": "annual", "performance": "excellent"},
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   "json-distributor-123",
	}

	jsonData, err := json.Marshal(reseller)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReseller Reseller
	err = json.Unmarshal(jsonData, &unmarshaledReseller)
	assert.NoError(t, err)

	assert.Equal(t, reseller.ID, unmarshaledReseller.ID)
	assert.Equal(t, reseller.Name, unmarshaledReseller.Name)
	assert.Equal(t, reseller.Email, unmarshaledReseller.Email)
	assert.Equal(t, reseller.CompanyName, unmarshaledReseller.CompanyName)
	assert.Equal(t, reseller.Status, unmarshaledReseller.Status)
	assert.Equal(t, reseller.Region, unmarshaledReseller.Region)
	assert.Equal(t, reseller.Metadata, unmarshaledReseller.Metadata)
	assert.Equal(t, reseller.CreatedAt.Unix(), unmarshaledReseller.CreatedAt.Unix())
	assert.Equal(t, reseller.UpdatedAt.Unix(), unmarshaledReseller.UpdatedAt.Unix())
	assert.Equal(t, reseller.CreatedBy, unmarshaledReseller.CreatedBy)
}

func TestResellerJSONTags(t *testing.T) {
	reseller := Reseller{
		ID:          "tag-reseller",
		Name:        "Tag Reseller",
		Email:       "tagreseller@example.com",
		CompanyName: "Tag Reseller Ltd",
		Status:      "active",
		Region:      "Asia Pacific",
		Metadata:    map[string]string{"test": "tags"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   "tag-distributor",
	}

	jsonData, err := json.Marshal(reseller)
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
	assert.Contains(t, jsonMap, "metadata")
	assert.Contains(t, jsonMap, "created_at")
	assert.Contains(t, jsonMap, "updated_at")
	assert.Contains(t, jsonMap, "created_by")

	// Verify values
	assert.Equal(t, "tag-reseller", jsonMap["id"])
	assert.Equal(t, "Tag Reseller", jsonMap["name"])
	assert.Equal(t, "tagreseller@example.com", jsonMap["email"])
	assert.Equal(t, "Tag Reseller Ltd", jsonMap["company_name"])
	assert.Equal(t, "active", jsonMap["status"])
	assert.Equal(t, "Asia Pacific", jsonMap["region"])
	assert.Equal(t, "tag-distributor", jsonMap["created_by"])
}

func TestCreateResellerRequestStructure(t *testing.T) {
	req := CreateResellerRequest{
		Name:          "New Reseller",
		Description:   "A new reseller organization",
		CustomData:    map[string]interface{}{"email": "newreseller@example.com", "region": "Latin America"},
		IsMfaRequired: true,
	}

	assert.Equal(t, "New Reseller", req.Name)
	assert.Equal(t, "A new reseller organization", req.Description)
	assert.Equal(t, map[string]interface{}{"email": "newreseller@example.com", "region": "Latin America"}, req.CustomData)
	assert.True(t, req.IsMfaRequired)
}

func TestCreateResellerRequestJSONSerialization(t *testing.T) {
	req := CreateResellerRequest{
		Name:        "JSON Create Reseller",
		Description: "JSON reseller description",
		CustomData: map[string]interface{}{
			"region":        "Middle East",
			"specialties":   []string{"retail", "hospitality", "healthcare"},
			"tier":          "silver",
			"established":   2018,
			"team_size":     25,
			"certification": true,
		},
		IsMfaRequired: false,
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReq CreateResellerRequest
	err = json.Unmarshal(jsonData, &unmarshaledReq)
	assert.NoError(t, err)

	assert.Equal(t, req.Name, unmarshaledReq.Name)
	assert.Equal(t, req.Description, unmarshaledReq.Description)
	// Compare individual CustomData fields due to JSON type conversion
	assert.Equal(t, req.CustomData["region"], unmarshaledReq.CustomData["region"])
	assert.Equal(t, req.CustomData["tier"], unmarshaledReq.CustomData["tier"])
	assert.Equal(t, req.CustomData["certification"], unmarshaledReq.CustomData["certification"])
	assert.Equal(t, float64(2018), unmarshaledReq.CustomData["established"]) // JSON converts int to float64
	assert.Equal(t, float64(25), unmarshaledReq.CustomData["team_size"])     // JSON converts int to float64
	// Specialties slice becomes []interface{} after JSON unmarshal
	expectedSpecialties := []interface{}{"retail", "hospitality", "healthcare"}
	assert.Equal(t, expectedSpecialties, unmarshaledReq.CustomData["specialties"])
	assert.Equal(t, req.IsMfaRequired, unmarshaledReq.IsMfaRequired)
}

func TestUpdateResellerRequestStructure(t *testing.T) {
	mfaRequired := true
	req := UpdateResellerRequest{
		Name:        "Updated Reseller",
		Description: "Updated reseller description",
		CustomData: map[string]interface{}{
			"status":      "updated",
			"version":     "2.0",
			"performance": map[string]interface{}{"sales": "excellent", "support": "good"},
		},
		IsMfaRequired: &mfaRequired,
	}

	assert.Equal(t, "Updated Reseller", req.Name)
	assert.Equal(t, "Updated reseller description", req.Description)
	assert.NotNil(t, req.IsMfaRequired)
	assert.True(t, *req.IsMfaRequired)
}

func TestUpdateResellerRequestJSONSerialization(t *testing.T) {
	mfaRequired := false
	req := UpdateResellerRequest{
		Name:        "JSON Update Reseller",
		Description: "JSON update description",
		CustomData: map[string]interface{}{
			"api_version": "v2",
			"capabilities": map[string]bool{
				"customer_management": true,
				"sales_tracking":      true,
				"support_tickets":     false,
			},
			"metrics": map[string]int{
				"customers":      150,
				"monthly_sales":  50000,
				"support_rating": 4,
			},
		},
		IsMfaRequired: &mfaRequired,
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReq UpdateResellerRequest
	err = json.Unmarshal(jsonData, &unmarshaledReq)
	assert.NoError(t, err)

	assert.Equal(t, req.Name, unmarshaledReq.Name)
	assert.Equal(t, req.Description, unmarshaledReq.Description)
	// Compare individual CustomData fields due to JSON type conversion
	assert.Equal(t, req.CustomData["api_version"], unmarshaledReq.CustomData["api_version"])
	// Capabilities map becomes map[string]interface{} after JSON unmarshal
	expectedCapabilities := map[string]interface{}{
		"customer_management": true,
		"sales_tracking":      true,
		"support_tickets":     false,
	}
	assert.Equal(t, expectedCapabilities, unmarshaledReq.CustomData["capabilities"])
	// Metrics map becomes map[string]interface{} after JSON unmarshal
	expectedMetrics := map[string]interface{}{
		"customers":      float64(150),
		"monthly_sales":  float64(50000),
		"support_rating": float64(4),
	}
	assert.Equal(t, expectedMetrics, unmarshaledReq.CustomData["metrics"])
	assert.NotNil(t, unmarshaledReq.IsMfaRequired)
	assert.False(t, *unmarshaledReq.IsMfaRequired)
}

func TestResellerStatusTypes(t *testing.T) {
	statusTypes := []string{"active", "suspended", "inactive"}

	for _, status := range statusTypes {
		t.Run("status_"+status, func(t *testing.T) {
			reseller := Reseller{
				ID:     "status-test-" + status,
				Name:   "Status Test Reseller",
				Status: status,
			}

			assert.Equal(t, status, reseller.Status)

			jsonData, err := json.Marshal(reseller)
			assert.NoError(t, err)

			var unmarshaledReseller Reseller
			err = json.Unmarshal(jsonData, &unmarshaledReseller)
			assert.NoError(t, err)
			assert.Equal(t, status, unmarshaledReseller.Status)
		})
	}
}

func TestResellerRegionTypes(t *testing.T) {
	regionTypes := []string{
		"North America", "South America", "Europe", "Asia Pacific",
		"Middle East", "Africa", "Oceania", "Central America",
	}

	for _, region := range regionTypes {
		t.Run("region_"+region, func(t *testing.T) {
			reseller := Reseller{
				ID:     "region-test-" + region,
				Name:   "Region Test Reseller",
				Region: region,
			}

			assert.Equal(t, region, reseller.Region)

			jsonData, err := json.Marshal(reseller)
			assert.NoError(t, err)

			var unmarshaledReseller Reseller
			err = json.Unmarshal(jsonData, &unmarshaledReseller)
			assert.NoError(t, err)
			assert.Equal(t, region, unmarshaledReseller.Region)
		})
	}
}

func TestResellerWithEmptyFields(t *testing.T) {
	reseller := Reseller{
		ID:          "",
		Name:        "",
		Email:       "",
		CompanyName: "",
		Status:      "",
		Region:      "",
		Metadata:    map[string]string{},
		CreatedAt:   time.Time{},
		UpdatedAt:   time.Time{},
		CreatedBy:   "",
	}

	jsonData, err := json.Marshal(reseller)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReseller Reseller
	err = json.Unmarshal(jsonData, &unmarshaledReseller)
	assert.NoError(t, err)

	assert.Equal(t, "", unmarshaledReseller.ID)
	assert.Equal(t, "", unmarshaledReseller.Name)
	assert.Equal(t, "", unmarshaledReseller.Email)
	assert.Equal(t, "", unmarshaledReseller.CompanyName)
	assert.Equal(t, "", unmarshaledReseller.Status)
	assert.Equal(t, "", unmarshaledReseller.Region)
	assert.Equal(t, map[string]string{}, unmarshaledReseller.Metadata)
	assert.Equal(t, "", unmarshaledReseller.CreatedBy)
}

func TestResellerWithNilMetadata(t *testing.T) {
	reseller := Reseller{
		ID:          "nil-metadata-reseller",
		Name:        "Nil Metadata Reseller",
		Email:       "nilmetadata@example.com",
		CompanyName: "Nil Metadata Corp",
		Status:      "active",
		Region:      "Global",
		Metadata:    nil,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   "nil-distributor",
	}

	jsonData, err := json.Marshal(reseller)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReseller Reseller
	err = json.Unmarshal(jsonData, &unmarshaledReseller)
	assert.NoError(t, err)

	assert.Equal(t, "nil-metadata-reseller", unmarshaledReseller.ID)
	assert.Equal(t, "Nil Metadata Reseller", unmarshaledReseller.Name)
	assert.Nil(t, unmarshaledReseller.Metadata)
}

func TestResellerRequestWithNilMfaRequired(t *testing.T) {
	req := UpdateResellerRequest{
		Name:          "Nil MFA Reseller",
		Description:   "Reseller with nil MFA requirement",
		CustomData:    map[string]interface{}{"test": "nil_mfa"},
		IsMfaRequired: nil,
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReq UpdateResellerRequest
	err = json.Unmarshal(jsonData, &unmarshaledReq)
	assert.NoError(t, err)

	assert.Equal(t, "Nil MFA Reseller", unmarshaledReq.Name)
	assert.Nil(t, unmarshaledReq.IsMfaRequired)
}

func TestResellerRequestWithNilCustomData(t *testing.T) {
	createReq := CreateResellerRequest{
		Name:          "Nil CustomData Create",
		Description:   "Create with nil custom data",
		CustomData:    nil,
		IsMfaRequired: false,
	}

	updateReq := UpdateResellerRequest{
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

func TestResellerCompleteProfile(t *testing.T) {
	now := time.Now()
	reseller := Reseller{
		ID:          "complete-reseller-profile",
		Name:        "Complete Reseller",
		Email:       "complete@reseller.com",
		CompanyName: "Complete Reseller Solutions & Services Ltd",
		Status:      "active",
		Region:      "Multi-Regional Europe",
		Metadata: map[string]string{
			"tier":                "platinum",
			"specialization":      "enterprise_solutions",
			"certification_level": "advanced",
			"support_level":       "premium",
			"contract_type":       "multi_year",
			"customers_limit":     "500",
			"features":            "all_standard",
			"performance_rating":  "excellent",
			"training_completed":  "all_modules",
			"partner_since":       "2020",
			"revenue_tier":        "top_25_percent",
			"focus_industries":    "healthcare,finance,retail",
			"team_size":           "50-100",
			"primary_language":    "english",
			"secondary_languages": "spanish,french,german",
			"time_zone":           "CET",
		},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "complete-distributor-456",
	}

	// Verify struct integrity
	assert.NotEmpty(t, reseller.ID)
	assert.NotEmpty(t, reseller.Name)
	assert.NotEmpty(t, reseller.Email)
	assert.NotEmpty(t, reseller.CompanyName)
	assert.NotEmpty(t, reseller.Status)
	assert.NotEmpty(t, reseller.Region)
	assert.NotEmpty(t, reseller.Metadata)
	assert.NotEmpty(t, reseller.CreatedBy)

	// Verify JSON round-trip
	jsonData, err := json.Marshal(reseller)
	assert.NoError(t, err)

	var unmarshaledReseller Reseller
	err = json.Unmarshal(jsonData, &unmarshaledReseller)
	assert.NoError(t, err)

	// Verify all fields match exactly (except time precision)
	assert.Equal(t, reseller.ID, unmarshaledReseller.ID)
	assert.Equal(t, reseller.Name, unmarshaledReseller.Name)
	assert.Equal(t, reseller.Email, unmarshaledReseller.Email)
	assert.Equal(t, reseller.CompanyName, unmarshaledReseller.CompanyName)
	assert.Equal(t, reseller.Status, unmarshaledReseller.Status)
	assert.Equal(t, reseller.Region, unmarshaledReseller.Region)
	assert.Equal(t, reseller.Metadata, unmarshaledReseller.Metadata)
	assert.Equal(t, reseller.CreatedBy, unmarshaledReseller.CreatedBy)
}

func TestResellerBusinessTiers(t *testing.T) {
	tiers := []string{"bronze", "silver", "gold", "platinum"}

	for _, tier := range tiers {
		t.Run("tier_"+tier, func(t *testing.T) {
			reseller := Reseller{
				ID:       "tier-test-" + tier,
				Name:     "Tier Test Reseller",
				Metadata: map[string]string{"tier": tier},
			}

			assert.Equal(t, tier, reseller.Metadata["tier"])

			jsonData, err := json.Marshal(reseller)
			assert.NoError(t, err)

			var unmarshaledReseller Reseller
			err = json.Unmarshal(jsonData, &unmarshaledReseller)
			assert.NoError(t, err)
			assert.Equal(t, tier, unmarshaledReseller.Metadata["tier"])
		})
	}
}

func TestResellerSpecializations(t *testing.T) {
	specializations := []string{"SMB", "enterprise", "government", "education", "healthcare", "finance", "retail"}

	for _, specialization := range specializations {
		t.Run("specialization_"+specialization, func(t *testing.T) {
			reseller := Reseller{
				ID:       "specialization-test-" + specialization,
				Name:     "Specialization Test Reseller",
				Metadata: map[string]string{"specialization": specialization},
			}

			assert.Equal(t, specialization, reseller.Metadata["specialization"])

			jsonData, err := json.Marshal(reseller)
			assert.NoError(t, err)

			var unmarshaledReseller Reseller
			err = json.Unmarshal(jsonData, &unmarshaledReseller)
			assert.NoError(t, err)
			assert.Equal(t, specialization, unmarshaledReseller.Metadata["specialization"])
		})
	}
}
