package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCustomerStructure(t *testing.T) {
	now := time.Now()
	customer := Customer{
		ID:          "customer-123",
		Name:        "Test Customer",
		Email:       "customer@example.com",
		CompanyName: "Test Customer Inc.",
		Status:      "active",
		Tier:        "premium",
		ResellerID:  "reseller-456",
		Metadata:    map[string]string{"region": "US", "support_level": "enterprise"},
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   "admin-789",
	}

	assert.Equal(t, "customer-123", customer.ID)
	assert.Equal(t, "Test Customer", customer.Name)
	assert.Equal(t, "customer@example.com", customer.Email)
	assert.Equal(t, "Test Customer Inc.", customer.CompanyName)
	assert.Equal(t, "active", customer.Status)
	assert.Equal(t, "premium", customer.Tier)
	assert.Equal(t, "reseller-456", customer.ResellerID)
	assert.Equal(t, map[string]string{"region": "US", "support_level": "enterprise"}, customer.Metadata)
	assert.Equal(t, now, customer.CreatedAt)
	assert.Equal(t, now, customer.UpdatedAt)
	assert.Equal(t, "admin-789", customer.CreatedBy)
}

func TestCustomerJSONSerialization(t *testing.T) {
	now := time.Now()
	customer := Customer{
		ID:          "json-customer-456",
		Name:        "JSON Test Customer",
		Email:       "jsoncustomer@example.com",
		CompanyName: "JSON Customer Corp",
		Status:      "suspended",
		Tier:        "basic",
		ResellerID:  "json-reseller-789",
		Metadata:    map[string]string{"plan": "starter", "users": "50"},
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   "json-admin-123",
	}

	jsonData, err := json.Marshal(customer)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledCustomer Customer
	err = json.Unmarshal(jsonData, &unmarshaledCustomer)
	assert.NoError(t, err)

	assert.Equal(t, customer.ID, unmarshaledCustomer.ID)
	assert.Equal(t, customer.Name, unmarshaledCustomer.Name)
	assert.Equal(t, customer.Email, unmarshaledCustomer.Email)
	assert.Equal(t, customer.CompanyName, unmarshaledCustomer.CompanyName)
	assert.Equal(t, customer.Status, unmarshaledCustomer.Status)
	assert.Equal(t, customer.Tier, unmarshaledCustomer.Tier)
	assert.Equal(t, customer.ResellerID, unmarshaledCustomer.ResellerID)
	assert.Equal(t, customer.Metadata, unmarshaledCustomer.Metadata)
	assert.Equal(t, customer.CreatedAt.Unix(), unmarshaledCustomer.CreatedAt.Unix())
	assert.Equal(t, customer.UpdatedAt.Unix(), unmarshaledCustomer.UpdatedAt.Unix())
	assert.Equal(t, customer.CreatedBy, unmarshaledCustomer.CreatedBy)
}

func TestCustomerJSONTags(t *testing.T) {
	customer := Customer{
		ID:          "tag-customer",
		Name:        "Tag Customer",
		Email:       "tagcustomer@example.com",
		CompanyName: "Tag Customer Ltd",
		Status:      "active",
		Tier:        "enterprise",
		ResellerID:  "tag-reseller",
		Metadata:    map[string]string{"test": "tags"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   "tag-admin",
	}

	jsonData, err := json.Marshal(customer)
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
	assert.Contains(t, jsonMap, "tier")
	assert.Contains(t, jsonMap, "reseller_id")
	assert.Contains(t, jsonMap, "metadata")
	assert.Contains(t, jsonMap, "created_at")
	assert.Contains(t, jsonMap, "updated_at")
	assert.Contains(t, jsonMap, "created_by")

	// Verify values
	assert.Equal(t, "tag-customer", jsonMap["id"])
	assert.Equal(t, "Tag Customer", jsonMap["name"])
	assert.Equal(t, "tagcustomer@example.com", jsonMap["email"])
	assert.Equal(t, "Tag Customer Ltd", jsonMap["company_name"])
	assert.Equal(t, "active", jsonMap["status"])
	assert.Equal(t, "enterprise", jsonMap["tier"])
	assert.Equal(t, "tag-reseller", jsonMap["reseller_id"])
	assert.Equal(t, "tag-admin", jsonMap["created_by"])
}

func TestCreateCustomerRequestStructure(t *testing.T) {
	req := CreateCustomerRequest{
		Name:          "New Customer",
		Description:   "A new customer organization",
		CustomData:    map[string]interface{}{"email": "newcustomer@example.com", "tier": "basic"},
		IsMfaRequired: false,
	}

	assert.Equal(t, "New Customer", req.Name)
	assert.Equal(t, "A new customer organization", req.Description)
	assert.Equal(t, map[string]interface{}{"email": "newcustomer@example.com", "tier": "basic"}, req.CustomData)
	assert.False(t, req.IsMfaRequired)
}

func TestCreateCustomerRequestJSONSerialization(t *testing.T) {
	req := CreateCustomerRequest{
		Name:          "JSON Create Customer",
		Description:   "JSON customer description",
		CustomData:    map[string]interface{}{"region": "EU", "support": "premium", "users": 100},
		IsMfaRequired: true,
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReq CreateCustomerRequest
	err = json.Unmarshal(jsonData, &unmarshaledReq)
	assert.NoError(t, err)

	assert.Equal(t, req.Name, unmarshaledReq.Name)
	assert.Equal(t, req.Description, unmarshaledReq.Description)
	// Compare individual CustomData fields due to JSON type conversion
	assert.Equal(t, req.CustomData["region"], unmarshaledReq.CustomData["region"])
	assert.Equal(t, req.CustomData["support"], unmarshaledReq.CustomData["support"])
	assert.Equal(t, float64(100), unmarshaledReq.CustomData["users"]) // JSON converts int to float64
	assert.Equal(t, req.IsMfaRequired, unmarshaledReq.IsMfaRequired)
}

func TestUpdateCustomerRequestStructure(t *testing.T) {
	mfaRequired := true
	req := UpdateCustomerRequest{
		Name:          "Updated Customer",
		Description:   "Updated customer description",
		CustomData:    map[string]interface{}{"status": "updated", "version": "2.0"},
		IsMfaRequired: &mfaRequired,
	}

	assert.Equal(t, "Updated Customer", req.Name)
	assert.Equal(t, "Updated customer description", req.Description)
	assert.Equal(t, map[string]interface{}{"status": "updated", "version": "2.0"}, req.CustomData)
	assert.NotNil(t, req.IsMfaRequired)
	assert.True(t, *req.IsMfaRequired)
}

func TestUpdateCustomerRequestJSONSerialization(t *testing.T) {
	mfaRequired := false
	req := UpdateCustomerRequest{
		Name:          "JSON Update Customer",
		Description:   "JSON update description",
		CustomData:    map[string]interface{}{"api_version": "v2", "features": []string{"sso", "audit"}},
		IsMfaRequired: &mfaRequired,
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReq UpdateCustomerRequest
	err = json.Unmarshal(jsonData, &unmarshaledReq)
	assert.NoError(t, err)

	assert.Equal(t, req.Name, unmarshaledReq.Name)
	assert.Equal(t, req.Description, unmarshaledReq.Description)
	// Compare individual CustomData fields due to JSON type conversion
	assert.Equal(t, req.CustomData["api_version"], unmarshaledReq.CustomData["api_version"])
	// Features slice becomes []interface{} after JSON unmarshal
	expectedFeatures := []interface{}{"sso", "audit"}
	assert.Equal(t, expectedFeatures, unmarshaledReq.CustomData["features"])
	assert.NotNil(t, unmarshaledReq.IsMfaRequired)
	assert.False(t, *unmarshaledReq.IsMfaRequired)
}

func TestCustomerStatusTypes(t *testing.T) {
	statusTypes := []string{"active", "suspended", "inactive"}

	for _, status := range statusTypes {
		t.Run("status_"+status, func(t *testing.T) {
			customer := Customer{
				ID:     "status-test-" + status,
				Name:   "Status Test Customer",
				Status: status,
			}

			assert.Equal(t, status, customer.Status)

			jsonData, err := json.Marshal(customer)
			assert.NoError(t, err)

			var unmarshaledCustomer Customer
			err = json.Unmarshal(jsonData, &unmarshaledCustomer)
			assert.NoError(t, err)
			assert.Equal(t, status, unmarshaledCustomer.Status)
		})
	}
}

func TestCustomerTierTypes(t *testing.T) {
	tierTypes := []string{"basic", "premium", "enterprise"}

	for _, tier := range tierTypes {
		t.Run("tier_"+tier, func(t *testing.T) {
			customer := Customer{
				ID:   "tier-test-" + tier,
				Name: "Tier Test Customer",
				Tier: tier,
			}

			assert.Equal(t, tier, customer.Tier)

			jsonData, err := json.Marshal(customer)
			assert.NoError(t, err)

			var unmarshaledCustomer Customer
			err = json.Unmarshal(jsonData, &unmarshaledCustomer)
			assert.NoError(t, err)
			assert.Equal(t, tier, unmarshaledCustomer.Tier)
		})
	}
}

func TestCustomerWithEmptyFields(t *testing.T) {
	customer := Customer{
		ID:          "",
		Name:        "",
		Email:       "",
		CompanyName: "",
		Status:      "",
		Tier:        "",
		ResellerID:  "",
		Metadata:    map[string]string{},
		CreatedAt:   time.Time{},
		UpdatedAt:   time.Time{},
		CreatedBy:   "",
	}

	jsonData, err := json.Marshal(customer)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledCustomer Customer
	err = json.Unmarshal(jsonData, &unmarshaledCustomer)
	assert.NoError(t, err)

	assert.Equal(t, "", unmarshaledCustomer.ID)
	assert.Equal(t, "", unmarshaledCustomer.Name)
	assert.Equal(t, "", unmarshaledCustomer.Email)
	assert.Equal(t, "", unmarshaledCustomer.CompanyName)
	assert.Equal(t, "", unmarshaledCustomer.Status)
	assert.Equal(t, "", unmarshaledCustomer.Tier)
	assert.Equal(t, "", unmarshaledCustomer.ResellerID)
	assert.Equal(t, map[string]string{}, unmarshaledCustomer.Metadata)
	assert.Equal(t, "", unmarshaledCustomer.CreatedBy)
}

func TestCustomerWithNilMetadata(t *testing.T) {
	customer := Customer{
		ID:          "nil-metadata-customer",
		Name:        "Nil Metadata Customer",
		Email:       "nilmetadata@example.com",
		CompanyName: "Nil Metadata Corp",
		Status:      "active",
		Tier:        "basic",
		ResellerID:  "nil-reseller",
		Metadata:    nil,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   "nil-admin",
	}

	jsonData, err := json.Marshal(customer)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledCustomer Customer
	err = json.Unmarshal(jsonData, &unmarshaledCustomer)
	assert.NoError(t, err)

	assert.Equal(t, "nil-metadata-customer", unmarshaledCustomer.ID)
	assert.Equal(t, "Nil Metadata Customer", unmarshaledCustomer.Name)
	assert.Nil(t, unmarshaledCustomer.Metadata)
}

func TestCustomerRequestWithNilMfaRequired(t *testing.T) {
	req := UpdateCustomerRequest{
		Name:          "Nil MFA Customer",
		Description:   "Customer with nil MFA requirement",
		CustomData:    map[string]interface{}{"test": "nil_mfa"},
		IsMfaRequired: nil,
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReq UpdateCustomerRequest
	err = json.Unmarshal(jsonData, &unmarshaledReq)
	assert.NoError(t, err)

	assert.Equal(t, "Nil MFA Customer", unmarshaledReq.Name)
	assert.Nil(t, unmarshaledReq.IsMfaRequired)
}

func TestCustomerRequestWithNilCustomData(t *testing.T) {
	createReq := CreateCustomerRequest{
		Name:          "Nil CustomData Create",
		Description:   "Create with nil custom data",
		CustomData:    nil,
		IsMfaRequired: false,
	}

	updateReq := UpdateCustomerRequest{
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

func TestCustomerCompleteProfile(t *testing.T) {
	now := time.Now()
	customer := Customer{
		ID:          "complete-customer-profile",
		Name:        "Complete Customer",
		Email:       "complete@customer.com",
		CompanyName: "Complete Customer Solutions Ltd",
		Status:      "active",
		Tier:        "enterprise",
		ResellerID:  "complete-reseller-123",
		Metadata: map[string]string{
			"region":        "North America",
			"support_level": "premium",
			"contract_type": "annual",
			"users_limit":   "unlimited",
			"features":      "all",
		},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "complete-admin-456",
	}

	// Verify struct integrity
	assert.NotEmpty(t, customer.ID)
	assert.NotEmpty(t, customer.Name)
	assert.NotEmpty(t, customer.Email)
	assert.NotEmpty(t, customer.CompanyName)
	assert.NotEmpty(t, customer.Status)
	assert.NotEmpty(t, customer.Tier)
	assert.NotEmpty(t, customer.ResellerID)
	assert.NotEmpty(t, customer.Metadata)
	assert.NotEmpty(t, customer.CreatedBy)

	// Verify JSON round-trip
	jsonData, err := json.Marshal(customer)
	assert.NoError(t, err)

	var unmarshaledCustomer Customer
	err = json.Unmarshal(jsonData, &unmarshaledCustomer)
	assert.NoError(t, err)

	// Verify all fields match exactly (except time precision)
	assert.Equal(t, customer.ID, unmarshaledCustomer.ID)
	assert.Equal(t, customer.Name, unmarshaledCustomer.Name)
	assert.Equal(t, customer.Email, unmarshaledCustomer.Email)
	assert.Equal(t, customer.CompanyName, unmarshaledCustomer.CompanyName)
	assert.Equal(t, customer.Status, unmarshaledCustomer.Status)
	assert.Equal(t, customer.Tier, unmarshaledCustomer.Tier)
	assert.Equal(t, customer.ResellerID, unmarshaledCustomer.ResellerID)
	assert.Equal(t, customer.Metadata, unmarshaledCustomer.Metadata)
	assert.Equal(t, customer.CreatedBy, unmarshaledCustomer.CreatedBy)
}
