package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLocalDistributorStructure(t *testing.T) {
	now := time.Now()
	logtoID := "logto-dist-123"
	syncedAt := now.Add(-1 * time.Hour)
	syncError := "sync error message"
	customData := map[string]interface{}{
		"region":   "europe",
		"priority": 1,
	}

	distributor := LocalDistributor{
		ID:             "dist-123",
		LogtoID:        &logtoID,
		Name:           "Test Distributor",
		Description:    "A test distributor",
		CustomData:     customData,
		CreatedAt:      now,
		UpdatedAt:      now,
		LogtoSyncedAt:  &syncedAt,
		LogtoSyncError: &syncError,
		Active:         true,
	}

	assert.Equal(t, "dist-123", distributor.ID)
	assert.Equal(t, &logtoID, distributor.LogtoID)
	assert.Equal(t, "Test Distributor", distributor.Name)
	assert.Equal(t, "A test distributor", distributor.Description)
	assert.Equal(t, customData, distributor.CustomData)
	assert.Equal(t, now, distributor.CreatedAt)
	assert.Equal(t, now, distributor.UpdatedAt)
	assert.Equal(t, &syncedAt, distributor.LogtoSyncedAt)
	assert.Equal(t, &syncError, distributor.LogtoSyncError)
	assert.True(t, distributor.Active)
}

func TestLocalDistributorJSONSerialization(t *testing.T) {
	now := time.Now()
	logtoID := "json-dist-456"
	customData := map[string]interface{}{
		"location": "italy",
		"tier":     "premium",
	}

	distributor := LocalDistributor{
		ID:          "json-dist-456",
		LogtoID:     &logtoID,
		Name:        "JSON Distributor",
		Description: "JSON test distributor",
		CustomData:  customData,
		CreatedAt:   now,
		UpdatedAt:   now,
		Active:      true,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(distributor)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var unmarshaledDistributor LocalDistributor
	err = json.Unmarshal(jsonData, &unmarshaledDistributor)
	assert.NoError(t, err)

	assert.Equal(t, distributor.ID, unmarshaledDistributor.ID)
	assert.Equal(t, *distributor.LogtoID, *unmarshaledDistributor.LogtoID)
	assert.Equal(t, distributor.Name, unmarshaledDistributor.Name)
	assert.Equal(t, distributor.Description, unmarshaledDistributor.Description)
	assert.Equal(t, distributor.CustomData, unmarshaledDistributor.CustomData)
	assert.Equal(t, distributor.Active, unmarshaledDistributor.Active)
}

func TestLocalResellerStructure(t *testing.T) {
	now := time.Now()
	logtoID := "logto-reseller-789"
	customData := map[string]interface{}{
		"segment": "small-business",
		"contact": "email@example.com",
	}

	reseller := LocalReseller{
		ID:          "reseller-789",
		LogtoID:     &logtoID,
		Name:        "Test Reseller",
		Description: "A test reseller",
		CustomData:  customData,
		CreatedAt:   now,
		UpdatedAt:   now,
		Active:      true,
	}

	assert.Equal(t, "reseller-789", reseller.ID)
	assert.Equal(t, &logtoID, reseller.LogtoID)
	assert.Equal(t, "Test Reseller", reseller.Name)
	assert.Equal(t, "A test reseller", reseller.Description)
	assert.Equal(t, customData, reseller.CustomData)
	assert.Equal(t, now, reseller.CreatedAt)
	assert.Equal(t, now, reseller.UpdatedAt)
	assert.True(t, reseller.Active)
}

func TestLocalCustomerStructure(t *testing.T) {
	now := time.Now()
	logtoID := "logto-customer-101"
	customData := map[string]interface{}{
		"industry": "healthcare",
		"size":     "enterprise",
	}

	customer := LocalCustomer{
		ID:          "customer-101",
		LogtoID:     &logtoID,
		Name:        "Test Customer",
		Description: "A test customer",
		CustomData:  customData,
		CreatedAt:   now,
		UpdatedAt:   now,
		Active:      false,
	}

	assert.Equal(t, "customer-101", customer.ID)
	assert.Equal(t, &logtoID, customer.LogtoID)
	assert.Equal(t, "Test Customer", customer.Name)
	assert.Equal(t, "A test customer", customer.Description)
	assert.Equal(t, customData, customer.CustomData)
	assert.Equal(t, now, customer.CreatedAt)
	assert.Equal(t, now, customer.UpdatedAt)
	assert.False(t, customer.Active)
}

func TestUserOrganizationStructure(t *testing.T) {
	userOrg := UserOrganization{
		ID:      "org-user-123",
		LogtoID: "logto-org-123",
		Name:    "Test Organization",
	}

	assert.Equal(t, "org-user-123", userOrg.ID)
	assert.Equal(t, "logto-org-123", userOrg.LogtoID)
	assert.Equal(t, "Test Organization", userOrg.Name)
}

func TestUserRoleStructure(t *testing.T) {
	userRole := UserRole{
		ID:   "role-admin",
		Name: "Administrator",
	}

	assert.Equal(t, "role-admin", userRole.ID)
	assert.Equal(t, "Administrator", userRole.Name)
}

func TestLocalUserStructure(t *testing.T) {
	now := time.Now()
	logtoID := "logto-user-456"
	phone := "+1234567890"
	organization := &UserOrganization{
		ID:      "org-123",
		LogtoID: "logto-org-123",
		Name:    "Test Org",
	}
	roles := []UserRole{
		{ID: "admin", Name: "Admin"},
		{ID: "support", Name: "Support"},
	}
	customData := map[string]interface{}{
		"department": "IT",
		"level":      "senior",
	}
	suspendedAt := now.Add(-2 * time.Hour)
	userRoleIDs := []string{"admin", "support"}
	orgID := "local-org-123"
	orgName := "Local Org Name"
	orgLocalID := "local-123"

	user := LocalUser{
		ID:                  "user-456",
		LogtoID:             &logtoID,
		Username:            "testuser456",
		Email:               "test@example.com",
		Name:                "Test User",
		Phone:               &phone,
		Organization:        organization,
		Roles:               roles,
		CustomData:          customData,
		CreatedAt:           now,
		UpdatedAt:           now,
		LogtoSyncedAt:       nil,
		DeletedAt:           nil,
		SuspendedAt:         &suspendedAt,
		UserRoleIDs:         userRoleIDs,
		OrganizationID:      &orgID,
		OrganizationName:    &orgName,
		OrganizationLocalID: &orgLocalID,
	}

	assert.Equal(t, "user-456", user.ID)
	assert.Equal(t, &logtoID, user.LogtoID)
	assert.Equal(t, "testuser456", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, &phone, user.Phone)
	assert.Equal(t, organization, user.Organization)
	assert.Equal(t, roles, user.Roles)
	assert.Equal(t, customData, user.CustomData)
	assert.Equal(t, &suspendedAt, user.SuspendedAt)
	assert.Equal(t, userRoleIDs, user.UserRoleIDs)
	assert.Equal(t, &orgID, user.OrganizationID)
}

func TestLocalUserActiveMethods(t *testing.T) {
	now := time.Now()

	// Test active user (no deleted_at, no suspended_at)
	activeUser := LocalUser{
		ID:          "active-user",
		DeletedAt:   nil,
		SuspendedAt: nil,
	}
	assert.True(t, activeUser.Active())
	assert.False(t, activeUser.IsDeleted())
	assert.False(t, activeUser.IsSuspended())

	// Test deleted user
	deletedAt := now
	deletedUser := LocalUser{
		ID:          "deleted-user",
		DeletedAt:   &deletedAt,
		SuspendedAt: nil,
	}
	assert.False(t, deletedUser.Active())
	assert.True(t, deletedUser.IsDeleted())
	assert.False(t, deletedUser.IsSuspended())

	// Test suspended user
	suspendedAt := now
	suspendedUser := LocalUser{
		ID:          "suspended-user",
		DeletedAt:   nil,
		SuspendedAt: &suspendedAt,
	}
	assert.False(t, suspendedUser.Active())
	assert.False(t, suspendedUser.IsDeleted())
	assert.True(t, suspendedUser.IsSuspended())

	// Test deleted and suspended user (both flags set)
	deletedAndSuspendedUser := LocalUser{
		ID:          "deleted-suspended-user",
		DeletedAt:   &deletedAt,
		SuspendedAt: &suspendedAt,
	}
	assert.False(t, deletedAndSuspendedUser.Active())
	assert.True(t, deletedAndSuspendedUser.IsDeleted())
	assert.True(t, deletedAndSuspendedUser.IsSuspended())
}

func TestSystemTotalsStructure(t *testing.T) {
	totals := SystemTotals{
		Total:          100,
		Alive:          80,
		Dead:           15,
		Zombie:         5,
		TimeoutMinutes: 30,
	}

	assert.Equal(t, 100, totals.Total)
	assert.Equal(t, 80, totals.Alive)
	assert.Equal(t, 15, totals.Dead)
	assert.Equal(t, 5, totals.Zombie)
	assert.Equal(t, 30, totals.TimeoutMinutes)
}

func TestCreateLocalDistributorRequest(t *testing.T) {
	customData := map[string]interface{}{
		"region": "europe",
	}

	request := CreateLocalDistributorRequest{
		Name:        "New Distributor",
		Description: "New distributor description",
		CustomData:  customData,
	}

	assert.Equal(t, "New Distributor", request.Name)
	assert.Equal(t, "New distributor description", request.Description)
	assert.Equal(t, customData, request.CustomData)
}

func TestCreateLocalResellerRequest(t *testing.T) {
	customData := map[string]interface{}{
		"segment": "enterprise",
	}

	request := CreateLocalResellerRequest{
		Name:        "New Reseller",
		Description: "New reseller description",
		CustomData:  customData,
	}

	assert.Equal(t, "New Reseller", request.Name)
	assert.Equal(t, "New reseller description", request.Description)
	assert.Equal(t, customData, request.CustomData)
}

func TestCreateLocalCustomerRequest(t *testing.T) {
	customData := map[string]interface{}{
		"industry": "fintech",
	}

	request := CreateLocalCustomerRequest{
		Name:        "New Customer",
		Description: "New customer description",
		CustomData:  customData,
	}

	assert.Equal(t, "New Customer", request.Name)
	assert.Equal(t, "New customer description", request.Description)
	assert.Equal(t, customData, request.CustomData)
}

func TestUpdateLocalDistributorRequest(t *testing.T) {
	name := "Updated Distributor"
	description := "Updated description"
	customData := map[string]interface{}{
		"updated": true,
	}

	request := UpdateLocalDistributorRequest{
		Name:        &name,
		Description: &description,
		CustomData:  &customData,
	}

	assert.Equal(t, &name, request.Name)
	assert.Equal(t, &description, request.Description)
	assert.Equal(t, &customData, request.CustomData)
}

func TestUpdateLocalResellerRequest(t *testing.T) {
	name := "Updated Reseller"
	description := "Updated description"
	customData := map[string]interface{}{
		"updated": true,
	}

	request := UpdateLocalResellerRequest{
		Name:        &name,
		Description: &description,
		CustomData:  &customData,
	}

	assert.Equal(t, &name, request.Name)
	assert.Equal(t, &description, request.Description)
	assert.Equal(t, &customData, request.CustomData)
}

func TestUpdateLocalCustomerRequest(t *testing.T) {
	name := "Updated Customer"
	description := "Updated description"
	customData := map[string]interface{}{
		"updated": true,
	}

	request := UpdateLocalCustomerRequest{
		Name:        &name,
		Description: &description,
		CustomData:  &customData,
	}

	assert.Equal(t, &name, request.Name)
	assert.Equal(t, &description, request.Description)
	assert.Equal(t, &customData, request.CustomData)
}

func TestCreateLocalUserRequest(t *testing.T) {
	phone := "+1234567890"
	userRoleIDs := []string{"admin", "support"}
	orgID := "org-123"
	customData := map[string]interface{}{
		"department": "engineering",
	}

	request := CreateLocalUserRequest{
		Username:       "newuser",
		Email:          "newuser@example.com",
		Name:           "New User",
		Phone:          &phone,
		UserRoleIDs:    userRoleIDs,
		OrganizationID: &orgID,
		CustomData:     customData,
	}

	assert.Equal(t, "newuser", request.Username)
	assert.Equal(t, "newuser@example.com", request.Email)
	assert.Equal(t, "New User", request.Name)
	assert.Equal(t, &phone, request.Phone)
	assert.Equal(t, userRoleIDs, request.UserRoleIDs)
	assert.Equal(t, &orgID, request.OrganizationID)
	assert.Equal(t, customData, request.CustomData)
}

func TestUpdateLocalUserRequest(t *testing.T) {
	username := "updateduser"
	email := "updated@example.com"
	name := "Updated User"
	phone := "+9876543210"
	userRoleIDs := []string{"support"}
	orgID := "org-456"
	customData := map[string]interface{}{
		"department": "support",
	}

	request := UpdateLocalUserRequest{
		Username:       &username,
		Email:          &email,
		Name:           &name,
		Phone:          &phone,
		UserRoleIDs:    &userRoleIDs,
		OrganizationID: &orgID,
		CustomData:     &customData,
	}

	assert.Equal(t, &username, request.Username)
	assert.Equal(t, &email, request.Email)
	assert.Equal(t, &name, request.Name)
	assert.Equal(t, &phone, request.Phone)
	assert.Equal(t, &userRoleIDs, request.UserRoleIDs)
	assert.Equal(t, &orgID, request.OrganizationID)
	assert.Equal(t, &customData, request.CustomData)
}

func TestSuspendUserRequest(t *testing.T) {
	reason := "Policy violation"

	request := SuspendUserRequest{
		Reason: &reason,
	}

	assert.Equal(t, &reason, request.Reason)

	// Test with nil reason
	nilRequest := SuspendUserRequest{
		Reason: nil,
	}
	assert.Nil(t, nilRequest.Reason)
}

func TestReactivateUserRequest(t *testing.T) {
	reason := "Issue resolved"

	request := ReactivateUserRequest{
		Reason: &reason,
	}

	assert.Equal(t, &reason, request.Reason)

	// Test with nil reason
	nilRequest := ReactivateUserRequest{
		Reason: nil,
	}
	assert.Nil(t, nilRequest.Reason)
}

func TestLocalEntitiesJSONSerialization(t *testing.T) {
	// Test JSON serialization for all major structs
	now := time.Now()

	// Test LocalDistributor
	distributor := LocalDistributor{
		ID:   "dist-json-test",
		Name: "JSON Distributor",
	}
	distributorJSON, err := json.Marshal(distributor)
	assert.NoError(t, err)
	assert.NotEmpty(t, distributorJSON)

	// Test LocalReseller
	reseller := LocalReseller{
		ID:   "reseller-json-test",
		Name: "JSON Reseller",
	}
	resellerJSON, err := json.Marshal(reseller)
	assert.NoError(t, err)
	assert.NotEmpty(t, resellerJSON)

	// Test LocalCustomer
	customer := LocalCustomer{
		ID:   "customer-json-test",
		Name: "JSON Customer",
	}
	customerJSON, err := json.Marshal(customer)
	assert.NoError(t, err)
	assert.NotEmpty(t, customerJSON)

	// Test LocalUser
	user := LocalUser{
		ID:        "user-json-test",
		Username:  "jsonuser",
		Email:     "json@example.com",
		Name:      "JSON User",
		CreatedAt: now,
		UpdatedAt: now,
	}
	userJSON, err := json.Marshal(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, userJSON)

	// Test SystemTotals
	totals := SystemTotals{
		Total: 10,
		Alive: 8,
	}
	totalsJSON, err := json.Marshal(totals)
	assert.NoError(t, err)
	assert.NotEmpty(t, totalsJSON)
}

func TestLocalEntitiesJSONTags(t *testing.T) {
	// Test that JSON tags are properly configured
	now := time.Now()
	customData := map[string]interface{}{
		"test": "value",
	}

	distributor := LocalDistributor{
		ID:         "tag-test-dist",
		Name:       "Tag Test",
		CustomData: customData,
		CreatedAt:  now,
		Active:     true,
	}

	jsonData, err := json.Marshal(distributor)
	assert.NoError(t, err)

	// Parse JSON to verify field names
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	assert.NoError(t, err)

	// Verify JSON field names match struct tags
	expectedFields := []string{
		"id", "logto_id", "name", "description", "custom_data",
		"created_at", "updated_at", "logto_synced_at", "logto_sync_error", "active",
	}

	for _, field := range expectedFields {
		assert.Contains(t, jsonMap, field)
	}

	// Verify values
	assert.Equal(t, "tag-test-dist", jsonMap["id"])
	assert.Equal(t, "Tag Test", jsonMap["name"])
	assert.Equal(t, true, jsonMap["active"])
}

func TestLocalUserJSONTagsSpecial(t *testing.T) {
	// Test LocalUser JSON tags, especially the special fields
	now := time.Now()
	phone := "+1234567890"
	organization := &UserOrganization{
		ID:   "org-123",
		Name: "Test Org",
	}
	roles := []UserRole{
		{ID: "admin", Name: "Admin"},
	}
	customData := map[string]interface{}{
		"test": "data",
	}

	user := LocalUser{
		ID:           "user-json-tags",
		Username:     "jsonuser",
		Email:        "json@example.com",
		Name:         "JSON User",
		Phone:        &phone,
		Organization: organization,
		Roles:        roles,
		CustomData:   customData,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	jsonData, err := json.Marshal(user)
	assert.NoError(t, err)

	// Parse JSON to verify field names
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	assert.NoError(t, err)

	// Verify JSON field names match struct tags
	expectedFields := []string{
		"id", "logto_id", "username", "email", "name", "phone",
		"organization", "roles", "custom_data", "created_at", "updated_at",
		"logto_synced_at", "deleted_at", "suspended_at",
	}

	for _, field := range expectedFields {
		assert.Contains(t, jsonMap, field)
	}

	// Verify that internal fields are NOT in JSON (they have json:"-" tags)
	internalFields := []string{
		"user_role_ids", "organization_id", "organization_name", "organization_local_id",
	}

	for _, field := range internalFields {
		assert.NotContains(t, jsonMap, field)
	}

	// Verify values
	assert.Equal(t, "user-json-tags", jsonMap["id"])
	assert.Equal(t, "jsonuser", jsonMap["username"])
	assert.Equal(t, "json@example.com", jsonMap["email"])
	assert.Equal(t, "custom_data", "custom_data") // custom_data field uses snake_case
}

func TestLocalEntitiesPointerOperations(t *testing.T) {
	now := time.Now()

	// Test pointer operations with LocalUser
	user := &LocalUser{
		ID:        "pointer-user",
		Username:  "pointeruser",
		Email:     "pointer@example.com",
		CreatedAt: now,
	}

	assert.NotNil(t, user)
	assert.Equal(t, "pointer-user", user.ID)
	assert.True(t, user.Active()) // Should be active by default

	// Test JSON serialization with pointer
	jsonData, err := json.Marshal(user)
	assert.NoError(t, err)

	var unmarshaledUser *LocalUser
	err = json.Unmarshal(jsonData, &unmarshaledUser)
	assert.NoError(t, err)
	assert.NotNil(t, unmarshaledUser)
	assert.Equal(t, user.ID, unmarshaledUser.ID)
	assert.Equal(t, user.Username, unmarshaledUser.Username)
}
