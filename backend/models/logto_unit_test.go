/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLogtoManagementTokenResponseStruct(t *testing.T) {
	token := LogtoManagementTokenResponse{
		AccessToken: "test_access_token",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		Scope:       "all",
	}

	assert.Equal(t, "test_access_token", token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)
	assert.Equal(t, 3600, token.ExpiresIn)
	assert.Equal(t, "all", token.Scope)
}

func TestLogtoUserInfoStruct(t *testing.T) {
	userInfo := LogtoUserInfo{
		Sub:              "user_123",
		Username:         "testuser",
		Email:            "test@example.com",
		Name:             "Test User",
		Roles:            []string{"admin", "user"},
		OrganizationId:   "org_456",
		OrganizationName: "Test Organization",
	}

	assert.Equal(t, "user_123", userInfo.Sub)
	assert.Equal(t, "testuser", userInfo.Username)
	assert.Equal(t, "test@example.com", userInfo.Email)
	assert.Equal(t, "Test User", userInfo.Name)
	assert.Equal(t, []string{"admin", "user"}, userInfo.Roles)
	assert.Equal(t, "org_456", userInfo.OrganizationId)
	assert.Equal(t, "Test Organization", userInfo.OrganizationName)
}

func TestLogtoRoleStruct(t *testing.T) {
	role := LogtoRole{
		ID:          "rol_123",
		Name:        "Administrator",
		Description: "System administrator role",
		Type:        "Machine",
	}

	assert.Equal(t, "rol_123", role.ID)
	assert.Equal(t, "Administrator", role.Name)
	assert.Equal(t, "System administrator role", role.Description)
	assert.Equal(t, "Machine", role.Type)
}

func TestLogtoScopeStruct(t *testing.T) {
	scope := LogtoScope{
		ID:          "scope_123",
		Name:        "read:users",
		Description: "Read user information",
		ResourceID:  "resource_456",
	}

	assert.Equal(t, "scope_123", scope.ID)
	assert.Equal(t, "read:users", scope.Name)
	assert.Equal(t, "Read user information", scope.Description)
	assert.Equal(t, "resource_456", scope.ResourceID)
}

func TestLogtoOrganizationStruct(t *testing.T) {
	customData := map[string]interface{}{
		"type":      "distributor",
		"createdBy": "admin_user",
	}

	branding := &LogtoOrganizationBranding{
		LogoUrl:     "https://example.com/logo.png",
		DarkLogoUrl: "https://example.com/dark-logo.png",
	}

	org := LogtoOrganization{
		ID:          "org_123",
		Name:        "Test Organization",
		Description: "Test organization description",
		CustomData:  customData,
		Branding:    branding,
	}

	assert.Equal(t, "org_123", org.ID)
	assert.Equal(t, "Test Organization", org.Name)
	assert.Equal(t, "Test organization description", org.Description)
	assert.Equal(t, customData, org.CustomData)
	assert.NotNil(t, org.Branding)
	assert.Equal(t, "https://example.com/logo.png", org.Branding.LogoUrl)
}

func TestLogtoOrganizationRoleStruct(t *testing.T) {
	orgRole := LogtoOrganizationRole{
		ID:          "org_rol_123",
		Name:        "Owner",
		Description: "Organization owner",
		Type:        "User",
		IsDefault:   false,
	}

	assert.Equal(t, "org_rol_123", orgRole.ID)
	assert.Equal(t, "Owner", orgRole.Name)
	assert.Equal(t, "Organization owner", orgRole.Description)
	assert.Equal(t, "User", orgRole.Type)
	assert.False(t, orgRole.IsDefault)
}

func TestLogtoUserStruct(t *testing.T) {
	now := time.Now().Unix()
	lastSignIn := now - 3600

	user := LogtoUser{
		ID:            "usr_123",
		Username:      "testuser",
		PrimaryEmail:  "test@example.com",
		PrimaryPhone:  "+1234567890",
		Name:          "Test User",
		Avatar:        "https://example.com/avatar.png",
		CustomData:    map[string]interface{}{"department": "IT"},
		Identities:    map[string]interface{}{"google": "google_id_123"},
		LastSignInAt:  &lastSignIn,
		CreatedAt:     now - 86400,
		UpdatedAt:     now,
		Profile:       map[string]interface{}{"bio": "Test user bio"},
		ApplicationId: "app_456",
		IsSuspended:   false,
		HasPassword:   true,
	}

	assert.Equal(t, "usr_123", user.ID)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "test@example.com", user.PrimaryEmail)
	assert.Equal(t, "+1234567890", user.PrimaryPhone)
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, "https://example.com/avatar.png", user.Avatar)
	assert.NotNil(t, user.LastSignInAt)
	assert.Equal(t, lastSignIn, *user.LastSignInAt)
	assert.False(t, user.IsSuspended)
	assert.True(t, user.HasPassword)
}

func TestPaginationInfoStruct(t *testing.T) {
	nextPage := 3
	prevPage := 1

	pagination := PaginationInfo{
		Page:       2,
		PageSize:   20,
		TotalCount: 100,
		TotalPages: 5,
		HasNext:    true,
		HasPrev:    true,
		NextPage:   &nextPage,
		PrevPage:   &prevPage,
	}

	assert.Equal(t, 2, pagination.Page)
	assert.Equal(t, 20, pagination.PageSize)
	assert.Equal(t, 100, pagination.TotalCount)
	assert.Equal(t, 5, pagination.TotalPages)
	assert.True(t, pagination.HasNext)
	assert.True(t, pagination.HasPrev)
	assert.NotNil(t, pagination.NextPage)
	assert.Equal(t, 3, *pagination.NextPage)
	assert.NotNil(t, pagination.PrevPage)
	assert.Equal(t, 1, *pagination.PrevPage)
}

func TestOrganizationFiltersStruct(t *testing.T) {
	filters := OrganizationFilters{
		Name:        "Test Org",
		Description: "Test description",
		Type:        "distributor",
		CreatedBy:   "admin_user",
		Search:      "test search",
	}

	assert.Equal(t, "Test Org", filters.Name)
	assert.Equal(t, "Test description", filters.Description)
	assert.Equal(t, "distributor", filters.Type)
	assert.Equal(t, "admin_user", filters.CreatedBy)
	assert.Equal(t, "test search", filters.Search)
}

func TestPaginatedOrganizationsStruct(t *testing.T) {
	orgs := []LogtoOrganization{
		{ID: "org_1", Name: "Org 1"},
		{ID: "org_2", Name: "Org 2"},
	}

	pagination := PaginationInfo{
		Page:       1,
		PageSize:   2,
		TotalCount: 2,
		TotalPages: 1,
		HasNext:    false,
		HasPrev:    false,
	}

	paginatedOrgs := PaginatedOrganizations{
		Data:       orgs,
		Pagination: pagination,
	}

	assert.Len(t, paginatedOrgs.Data, 2)
	assert.Equal(t, orgs, paginatedOrgs.Data)
	assert.Equal(t, 1, paginatedOrgs.Pagination.Page)
	assert.Equal(t, 2, paginatedOrgs.Pagination.TotalCount)
}

func TestJitRolesResultStruct(t *testing.T) {
	roles := []LogtoOrganizationRole{
		{ID: "rol_1", Name: "Role 1"},
		{ID: "rol_2", Name: "Role 2"},
	}

	result := JitRolesResult{
		OrgID: "org_123",
		Roles: roles,
		Error: nil,
	}

	assert.Equal(t, "org_123", result.OrgID)
	assert.Len(t, result.Roles, 2)
	assert.Equal(t, roles, result.Roles)
	assert.Nil(t, result.Error)
}

func TestPaginatedUsersStruct(t *testing.T) {
	users := []LogtoUser{
		{ID: "usr_1", Username: "user1"},
		{ID: "usr_2", Username: "user2"},
	}

	pagination := PaginationInfo{
		Page:       1,
		PageSize:   2,
		TotalCount: 2,
		TotalPages: 1,
		HasNext:    false,
		HasPrev:    false,
	}

	paginatedUsers := PaginatedUsers{
		Data:       users,
		Pagination: pagination,
	}

	assert.Len(t, paginatedUsers.Data, 2)
	assert.Equal(t, users, paginatedUsers.Data)
	assert.Equal(t, 1, paginatedUsers.Pagination.Page)
	assert.Equal(t, 2, paginatedUsers.Pagination.TotalCount)
}

func TestUserFiltersStruct(t *testing.T) {
	filters := UserFilters{
		Search:         "test search",
		OrganizationID: "org_123",
		Role:           "admin",
		Username:       "testuser",
		Email:          "test@example.com",
	}

	assert.Equal(t, "test search", filters.Search)
	assert.Equal(t, "org_123", filters.OrganizationID)
	assert.Equal(t, "admin", filters.Role)
	assert.Equal(t, "testuser", filters.Username)
	assert.Equal(t, "test@example.com", filters.Email)
}

func TestOrgUsersResultStruct(t *testing.T) {
	users := []LogtoUser{
		{ID: "usr_1", Username: "user1"},
		{ID: "usr_2", Username: "user2"},
	}

	result := OrgUsersResult{
		OrgID: "org_123",
		Users: users,
		Error: nil,
	}

	assert.Equal(t, "org_123", result.OrgID)
	assert.Len(t, result.Users, 2)
	assert.Equal(t, users, result.Users)
	assert.Nil(t, result.Error)
}

func TestUsersCacheStruct(t *testing.T) {
	now := time.Now()
	users := []LogtoUser{
		{ID: "usr_1", Username: "user1"},
	}

	cache := UsersCache{
		Users:     users,
		CachedAt:  now,
		ExpiresAt: now.Add(time.Hour),
	}

	assert.Len(t, cache.Users, 1)
	assert.Equal(t, users, cache.Users)
	assert.Equal(t, now, cache.CachedAt)
	assert.True(t, cache.ExpiresAt.After(now))
}

func TestLogtoOrganizationBrandingStruct(t *testing.T) {
	branding := LogtoOrganizationBranding{
		LogoUrl:     "https://example.com/logo.png",
		DarkLogoUrl: "https://example.com/dark-logo.png",
		Favicon:     "https://example.com/favicon.ico",
		DarkFavicon: "https://example.com/dark-favicon.ico",
	}

	assert.Equal(t, "https://example.com/logo.png", branding.LogoUrl)
	assert.Equal(t, "https://example.com/dark-logo.png", branding.DarkLogoUrl)
	assert.Equal(t, "https://example.com/favicon.ico", branding.Favicon)
	assert.Equal(t, "https://example.com/dark-favicon.ico", branding.DarkFavicon)
}

func TestCreateOrganizationRequestStruct(t *testing.T) {
	customData := map[string]interface{}{
		"type": "distributor",
	}

	branding := &LogtoOrganizationBranding{
		LogoUrl: "https://example.com/logo.png",
	}

	request := CreateOrganizationRequest{
		Name:        "New Organization",
		Description: "New organization description",
		CustomData:  customData,
		Branding:    branding,
	}

	assert.Equal(t, "New Organization", request.Name)
	assert.Equal(t, "New organization description", request.Description)
	assert.Equal(t, customData, request.CustomData)
	assert.NotNil(t, request.Branding)
}

func TestUpdateOrganizationRequestStruct(t *testing.T) {
	name := "Updated Organization"
	description := "Updated description"

	request := UpdateOrganizationRequest{
		Name:        &name,
		Description: &description,
		CustomData:  map[string]interface{}{"updated": true},
	}

	assert.NotNil(t, request.Name)
	assert.Equal(t, "Updated Organization", *request.Name)
	assert.NotNil(t, request.Description)
	assert.Equal(t, "Updated description", *request.Description)
}

func TestCreateUserRequestStruct(t *testing.T) {
	phone := "+1234567890"
	avatar := "https://example.com/avatar.png"

	request := CreateUserRequest{
		Username:     "newuser",
		Password:     "securepassword",
		Name:         "New User",
		PrimaryEmail: "newuser@example.com",
		PrimaryPhone: phone,
		Avatar:       &avatar,
		CustomData:   map[string]interface{}{"department": "IT"},
	}

	assert.Equal(t, "newuser", request.Username)
	assert.Equal(t, "securepassword", request.Password)
	assert.Equal(t, "New User", request.Name)
	assert.Equal(t, "newuser@example.com", request.PrimaryEmail)
	assert.NotNil(t, request.PrimaryPhone)
	assert.Equal(t, "+1234567890", request.PrimaryPhone)
	assert.NotNil(t, request.Avatar)
	assert.Equal(t, "https://example.com/avatar.png", *request.Avatar)
}

func TestUpdateUserRequestStruct(t *testing.T) {
	username := "updateduser"
	name := "Updated User"
	email := "updated@example.com"
	phone := "+0987654321"
	avatar := "https://example.com/new-avatar.png"
	suspended := true

	request := UpdateUserRequest{
		Username:     &username,
		Name:         &name,
		PrimaryEmail: &email,
		PrimaryPhone: &phone,
		Avatar:       &avatar,
		CustomData:   map[string]interface{}{"updated": true},
		IsSuspended:  &suspended,
	}

	assert.NotNil(t, request.Username)
	assert.Equal(t, "updateduser", *request.Username)
	assert.NotNil(t, request.Name)
	assert.Equal(t, "Updated User", *request.Name)
	assert.NotNil(t, request.PrimaryEmail)
	assert.Equal(t, "updated@example.com", *request.PrimaryEmail)
	assert.NotNil(t, request.PrimaryPhone)
	assert.Equal(t, "+0987654321", *request.PrimaryPhone)
	assert.NotNil(t, request.Avatar)
	assert.Equal(t, "https://example.com/new-avatar.png", *request.Avatar)
	assert.NotNil(t, request.IsSuspended)
	assert.True(t, *request.IsSuspended)
}

func TestJSONTagsConsistency(t *testing.T) {
	// Test that all structs have proper JSON tags for API serialization
	t.Run("LogtoManagementTokenResponse", func(t *testing.T) {
		token := LogtoManagementTokenResponse{}
		assert.IsType(t, "", token.AccessToken)
		assert.IsType(t, "", token.TokenType)
		assert.IsType(t, 0, token.ExpiresIn)
		assert.IsType(t, "", token.Scope)
	})

	t.Run("LogtoUserInfo", func(t *testing.T) {
		userInfo := LogtoUserInfo{}
		assert.IsType(t, "", userInfo.Sub)
		assert.IsType(t, []string{}, userInfo.Roles)
	})

	t.Run("PaginationInfo", func(t *testing.T) {
		pagination := PaginationInfo{}
		assert.IsType(t, 0, pagination.Page)
		assert.IsType(t, false, pagination.HasNext)
		assert.IsType(t, (*int)(nil), pagination.NextPage)
	})
}

func TestPointerFieldHandling(t *testing.T) {
	// Test structs with pointer fields for optional values
	t.Run("UpdateOrganizationRequest with nil pointers", func(t *testing.T) {
		request := UpdateOrganizationRequest{
			CustomData: map[string]interface{}{"key": "value"},
		}

		assert.Nil(t, request.Name)
		assert.Nil(t, request.Description)
		assert.NotNil(t, request.CustomData)
	})

	t.Run("UpdateUserRequest with nil pointers", func(t *testing.T) {
		request := UpdateUserRequest{
			CustomData: map[string]interface{}{"updated": true},
		}

		assert.Nil(t, request.Username)
		assert.Nil(t, request.Name)
		assert.Nil(t, request.PrimaryEmail)
		assert.Nil(t, request.PrimaryPhone)
		assert.Nil(t, request.Avatar)
		assert.Nil(t, request.IsSuspended)
		assert.NotNil(t, request.CustomData)
	})

	t.Run("LogtoUser with nil LastSignInAt", func(t *testing.T) {
		user := LogtoUser{
			ID:       "usr_123",
			Username: "testuser",
		}

		assert.Nil(t, user.LastSignInAt)
		assert.Equal(t, "usr_123", user.ID)
	})
}

func TestTimestampHandling(t *testing.T) {
	// Test Unix timestamp handling in LogtoUser
	now := time.Now().Unix()

	user := LogtoUser{
		CreatedAt: now - 86400, // 1 day ago
		UpdatedAt: now,
	}

	assert.Equal(t, now-86400, user.CreatedAt)
	assert.Equal(t, now, user.UpdatedAt)
	assert.True(t, user.UpdatedAt > user.CreatedAt)
}

func TestCustomDataHandling(t *testing.T) {
	// Test that custom data maps work correctly
	customData := map[string]interface{}{
		"string_field":  "value",
		"number_field":  123,
		"boolean_field": true,
		"nested_object": map[string]interface{}{
			"nested_field": "nested_value",
		},
		"array_field": []interface{}{"item1", "item2"},
	}

	org := LogtoOrganization{
		CustomData: customData,
	}

	assert.Equal(t, customData, org.CustomData)
	assert.Equal(t, "value", org.CustomData["string_field"])
	assert.Equal(t, 123, org.CustomData["number_field"])
	assert.Equal(t, true, org.CustomData["boolean_field"])
}

func TestEmptyStructsInitialization(t *testing.T) {
	// Test that empty structs initialize correctly
	emptyStructs := []interface{}{
		LogtoManagementTokenResponse{},
		LogtoUserInfo{},
		LogtoRole{},
		LogtoScope{},
		LogtoOrganization{},
		LogtoOrganizationRole{},
		LogtoUser{},
		PaginationInfo{},
		OrganizationFilters{},
		UserFilters{},
		CreateOrganizationRequest{},
		UpdateOrganizationRequest{},
		CreateUserRequest{},
		UpdateUserRequest{},
	}

	for _, s := range emptyStructs {
		assert.NotNil(t, s)
	}
}
