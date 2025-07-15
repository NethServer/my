package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateAccountRequestStructure(t *testing.T) {
	req := CreateAccountRequest{
		Username:       "testuser",
		Email:          "test@example.com",
		Name:           "Test User",
		Phone:          "+1234567890",
		Password:       "securepass123",
		UserRoleIDs:    []string{"role-123", "role-456"},
		OrganizationID: "org-456",
		Avatar:         "https://example.com/avatar.jpg",
		CustomData:     map[string]interface{}{"department": "IT"},
	}

	assert.Equal(t, "testuser", req.Username)
	assert.Equal(t, "test@example.com", req.Email)
	assert.Equal(t, "Test User", req.Name)
	assert.Equal(t, "+1234567890", req.Phone)
	assert.Equal(t, "securepass123", req.Password)
	assert.Equal(t, []string{"role-123", "role-456"}, req.UserRoleIDs)
	assert.Equal(t, "org-456", req.OrganizationID)
	assert.Equal(t, "https://example.com/avatar.jpg", req.Avatar)
	assert.Equal(t, map[string]interface{}{"department": "IT"}, req.CustomData)
}

func TestCreateAccountRequestJSONSerialization(t *testing.T) {
	req := CreateAccountRequest{
		Username:       "jsonuser",
		Email:          "json@example.com",
		Name:           "JSON User",
		Phone:          "+9876543210",
		Password:       "jsonpass123",
		UserRoleIDs:    []string{"role-json"},
		OrganizationID: "org-json",
		Avatar:         "https://example.com/json-avatar.jpg",
		CustomData:     map[string]interface{}{"team": "Backend"},
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReq CreateAccountRequest
	err = json.Unmarshal(jsonData, &unmarshaledReq)
	assert.NoError(t, err)

	assert.Equal(t, req.Username, unmarshaledReq.Username)
	assert.Equal(t, req.Email, unmarshaledReq.Email)
	assert.Equal(t, req.Name, unmarshaledReq.Name)
	assert.Equal(t, req.Phone, unmarshaledReq.Phone)
	assert.Equal(t, req.Password, unmarshaledReq.Password)
	assert.Equal(t, req.UserRoleIDs, unmarshaledReq.UserRoleIDs)
	assert.Equal(t, req.OrganizationID, unmarshaledReq.OrganizationID)
	assert.Equal(t, req.Avatar, unmarshaledReq.Avatar)
	assert.Equal(t, req.CustomData, unmarshaledReq.CustomData)
}

func TestCreateAccountRequestJSONTags(t *testing.T) {
	req := CreateAccountRequest{
		Username:       "taguser",
		Email:          "tag@example.com",
		Name:           "Tag User",
		Phone:          "+1111111111",
		Password:       "tagpass123",
		UserRoleIDs:    []string{"role-tag"},
		OrganizationID: "org-tag",
		Avatar:         "https://example.com/tag-avatar.jpg",
		CustomData:     map[string]interface{}{"role": "Tester"},
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)

	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	assert.NoError(t, err)

	assert.Contains(t, jsonMap, "username")
	assert.Contains(t, jsonMap, "email")
	assert.Contains(t, jsonMap, "name")
	assert.Contains(t, jsonMap, "phone")
	assert.Contains(t, jsonMap, "password")
	assert.Contains(t, jsonMap, "userRoleIds")
	assert.Contains(t, jsonMap, "organizationId")
	assert.Contains(t, jsonMap, "avatar")
	assert.Contains(t, jsonMap, "customData")

	assert.Equal(t, "taguser", jsonMap["username"])
	assert.Equal(t, "tag@example.com", jsonMap["email"])
	assert.Equal(t, []interface{}{"role-tag"}, jsonMap["userRoleIds"])
	assert.Equal(t, "org-tag", jsonMap["organizationId"])
}

func TestUpdateAccountRequestStructure(t *testing.T) {
	req := UpdateAccountRequest{
		Username:       "updateduser",
		Email:          "updated@example.com",
		Name:           "Updated User",
		Phone:          "+2222222222",
		UserRoleIDs:    &[]string{"role-updated"},
		OrganizationID: "org-updated",
		Avatar:         "https://example.com/updated-avatar.jpg",
		CustomData:     map[string]interface{}{"status": "updated"},
	}

	assert.Equal(t, "updateduser", req.Username)
	assert.Equal(t, "updated@example.com", req.Email)
	assert.Equal(t, "Updated User", req.Name)
	assert.Equal(t, "+2222222222", req.Phone)
	assert.Equal(t, []string{"role-updated"}, *req.UserRoleIDs)
	assert.Equal(t, "org-updated", req.OrganizationID)
	assert.Equal(t, "https://example.com/updated-avatar.jpg", req.Avatar)
	assert.Equal(t, map[string]interface{}{"status": "updated"}, req.CustomData)
}

func TestUpdateAccountRequestJSONSerialization(t *testing.T) {
	req := UpdateAccountRequest{
		Username:       "updatejson",
		Email:          "updatejson@example.com",
		Name:           "Update JSON User",
		Phone:          "+3333333333",
		UserRoleIDs:    &[]string{"role-updatejson"},
		OrganizationID: "org-updatejson",
		Avatar:         "https://example.com/updatejson-avatar.jpg",
		CustomData:     map[string]interface{}{"version": "2.0"},
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReq UpdateAccountRequest
	err = json.Unmarshal(jsonData, &unmarshaledReq)
	assert.NoError(t, err)

	assert.Equal(t, req, unmarshaledReq)
}

func TestAccountResponseStructure(t *testing.T) {
	now := time.Now()
	lastSignIn := now.Add(-24 * time.Hour)

	resp := AccountResponse{
		ID:               "account-123",
		Username:         "responseuser",
		Email:            "response@example.com",
		Name:             "Response User",
		Phone:            "+4444444444",
		Avatar:           "https://example.com/response-avatar.jpg",
		UserRoleIDs:      []string{"Admin"},
		OrganizationID:   "org-response",
		OrganizationName: "Response Organization",
		OrganizationRole: "Owner",
		IsSuspended:      false,
		LastSignInAt:     &lastSignIn,
		CreatedAt:        now,
		UpdatedAt:        now,
		CustomData:       map[string]interface{}{"source": "api"},
	}

	assert.Equal(t, "account-123", resp.ID)
	assert.Equal(t, "responseuser", resp.Username)
	assert.Equal(t, "response@example.com", resp.Email)
	assert.Equal(t, "Response User", resp.Name)
	assert.Equal(t, "+4444444444", resp.Phone)
	assert.Equal(t, "https://example.com/response-avatar.jpg", resp.Avatar)
	assert.Equal(t, []string{"Admin"}, resp.UserRoleIDs)
	assert.Equal(t, "org-response", resp.OrganizationID)
	assert.Equal(t, "Response Organization", resp.OrganizationName)
	assert.Equal(t, "Owner", resp.OrganizationRole)
	assert.False(t, resp.IsSuspended)
	assert.Equal(t, &lastSignIn, resp.LastSignInAt)
	assert.Equal(t, now, resp.CreatedAt)
	assert.Equal(t, now, resp.UpdatedAt)
	assert.Equal(t, map[string]interface{}{"source": "api"}, resp.CustomData)
}

func TestAccountResponseJSONSerialization(t *testing.T) {
	now := time.Now()
	lastSignIn := now.Add(-48 * time.Hour)

	resp := AccountResponse{
		ID:               "json-account-456",
		Username:         "jsonresponse",
		Email:            "jsonresponse@example.com",
		Name:             "JSON Response User",
		Phone:            "+5555555555",
		Avatar:           "https://example.com/jsonresponse-avatar.jpg",
		UserRoleIDs:      []string{"Support"},
		OrganizationID:   "org-jsonresponse",
		OrganizationName: "JSON Response Organization",
		OrganizationRole: "Distributor",
		IsSuspended:      true,
		LastSignInAt:     &lastSignIn,
		CreatedAt:        now,
		UpdatedAt:        now,
		CustomData:       map[string]interface{}{"api_version": "v1"},
	}

	jsonData, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledResp AccountResponse
	err = json.Unmarshal(jsonData, &unmarshaledResp)
	assert.NoError(t, err)

	assert.Equal(t, resp.ID, unmarshaledResp.ID)
	assert.Equal(t, resp.Username, unmarshaledResp.Username)
	assert.Equal(t, resp.Email, unmarshaledResp.Email)
	assert.Equal(t, resp.Name, unmarshaledResp.Name)
	assert.Equal(t, resp.Phone, unmarshaledResp.Phone)
	assert.Equal(t, resp.Avatar, unmarshaledResp.Avatar)
	assert.Equal(t, resp.UserRoleIDs, unmarshaledResp.UserRoleIDs)
	assert.Equal(t, resp.OrganizationID, unmarshaledResp.OrganizationID)
	assert.Equal(t, resp.OrganizationName, unmarshaledResp.OrganizationName)
	assert.Equal(t, resp.OrganizationRole, unmarshaledResp.OrganizationRole)
	assert.Equal(t, resp.IsSuspended, unmarshaledResp.IsSuspended)
	assert.Equal(t, resp.CreatedAt.Unix(), unmarshaledResp.CreatedAt.Unix())
	assert.Equal(t, resp.UpdatedAt.Unix(), unmarshaledResp.UpdatedAt.Unix())
	assert.Equal(t, resp.CustomData, unmarshaledResp.CustomData)
}

func TestAccountResponseWithNilLastSignIn(t *testing.T) {
	now := time.Now()

	resp := AccountResponse{
		ID:               "nil-signin-account",
		Username:         "nilsignin",
		Email:            "nilsignin@example.com",
		Name:             "Nil SignIn User",
		UserRoleIDs:      []string{"Admin"},
		OrganizationID:   "org-nilsignin",
		OrganizationName: "Nil SignIn Organization",
		OrganizationRole: "Customer",
		IsSuspended:      false,
		LastSignInAt:     nil,
		CreatedAt:        now,
		UpdatedAt:        now,
		CustomData:       map[string]interface{}{},
	}

	jsonData, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledResp AccountResponse
	err = json.Unmarshal(jsonData, &unmarshaledResp)
	assert.NoError(t, err)

	assert.Nil(t, unmarshaledResp.LastSignInAt)
	assert.Equal(t, resp.ID, unmarshaledResp.ID)
	assert.Equal(t, resp.Username, unmarshaledResp.Username)
	assert.False(t, unmarshaledResp.IsSuspended)
}

func TestAccountModelsWithEmptyCustomData(t *testing.T) {
	createReq := CreateAccountRequest{
		Username:       "emptymetauser",
		Email:          "emptymeta@example.com",
		Name:           "Empty Meta User",
		Password:       "emptypass123",
		UserRoleIDs:    []string{"role-emptymeta"},
		OrganizationID: "org-emptymeta",
		CustomData:     map[string]interface{}{},
	}

	updateReq := UpdateAccountRequest{
		Username:   "updatedemptymetauser",
		Email:      "updatedemptymeta@example.com",
		Name:       "Updated Empty Meta User",
		CustomData: map[string]interface{}{},
	}

	resp := AccountResponse{
		ID:               "empty-meta-resp",
		Username:         "emptymetaresp",
		Email:            "emptymetaresp@example.com",
		Name:             "Empty Meta Response User",
		UserRoleIDs:      []string{"Support"},
		OrganizationID:   "org-emptymetaresp",
		OrganizationName: "Empty Meta Organization",
		OrganizationRole: "Reseller",
		IsSuspended:      false,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		CustomData:       map[string]interface{}{},
	}

	// Test JSON serialization for all structures
	createJson, err := json.Marshal(createReq)
	assert.NoError(t, err)
	assert.NotEmpty(t, createJson)

	updateJson, err := json.Marshal(updateReq)
	assert.NoError(t, err)
	assert.NotEmpty(t, updateJson)

	respJson, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, respJson)

	assert.NotNil(t, createReq.CustomData)
	assert.NotNil(t, updateReq.CustomData)
	assert.NotNil(t, resp.CustomData)
	assert.Len(t, createReq.CustomData, 0)
	assert.Len(t, updateReq.CustomData, 0)
	assert.Len(t, resp.CustomData, 0)
}

func TestAccountModelsWithNilCustomData(t *testing.T) {
	createReq := CreateAccountRequest{
		Username:       "nilmetauser",
		Email:          "nilmeta@example.com",
		Name:           "Nil Meta User",
		Password:       "nilpass123",
		UserRoleIDs:    []string{"role-nilmeta"},
		OrganizationID: "org-nilmeta",
		CustomData:     nil,
	}

	updateReq := UpdateAccountRequest{
		Username:   "updatednilmetauser",
		Email:      "updatednilmeta@example.com",
		Name:       "Updated Nil Meta User",
		CustomData: nil,
	}

	resp := AccountResponse{
		ID:               "nil-meta-resp",
		Username:         "nilmetaresp",
		Email:            "nilmetaresp@example.com",
		Name:             "Nil Meta Response User",
		UserRoleIDs:      []string{"Admin"},
		OrganizationID:   "org-nilmetaresp",
		OrganizationName: "Nil Meta Organization",
		OrganizationRole: "Owner",
		IsSuspended:      true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		CustomData:       nil,
	}

	// Test JSON serialization with nil customData
	createJson, err := json.Marshal(createReq)
	assert.NoError(t, err)
	assert.NotEmpty(t, createJson)

	updateJson, err := json.Marshal(updateReq)
	assert.NoError(t, err)
	assert.NotEmpty(t, updateJson)

	respJson, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, respJson)

	assert.Nil(t, createReq.CustomData)
	assert.Nil(t, updateReq.CustomData)
	assert.Nil(t, resp.CustomData)
}

func TestAccountSuspensionStates(t *testing.T) {
	suspensionStates := []bool{true, false}

	for i, suspended := range suspensionStates {
		t.Run("suspension_state_"+string(rune(i+'A')), func(t *testing.T) {
			resp := AccountResponse{
				ID:               "suspension-test-" + string(rune(i+'A')),
				Username:         "suspensiontest",
				Email:            "suspension@example.com",
				Name:             "Suspension Test User",
				UserRoleIDs:      []string{"Support"},
				OrganizationID:   "org-suspension",
				OrganizationName: "Suspension Test Organization",
				OrganizationRole: "Customer",
				IsSuspended:      suspended,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
				CustomData:       map[string]interface{}{"test": "suspension"},
			}

			assert.Equal(t, suspended, resp.IsSuspended)

			jsonData, err := json.Marshal(resp)
			assert.NoError(t, err)

			var unmarshaledResp AccountResponse
			err = json.Unmarshal(jsonData, &unmarshaledResp)
			assert.NoError(t, err)
			assert.Equal(t, suspended, unmarshaledResp.IsSuspended)
		})
	}
}
