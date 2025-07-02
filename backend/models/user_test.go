package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserStructure(t *testing.T) {
	// Test User struct creation and field assignment
	user := User{
		ID:               "test-user-123",
		Username:         "testuser",
		Email:            "test@example.com",
		Name:             "Test User",
		UserRoles:        []string{"Admin", "Support"},
		UserPermissions:  []string{"manage:systems", "view:logs"},
		OrgRole:          "Distributor",
		OrgPermissions:   []string{"create:resellers", "manage:customers"},
		OrganizationID:   "org-123",
		OrganizationName: "Test Organization",
	}

	assert.Equal(t, "test-user-123", user.ID)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, []string{"Admin", "Support"}, user.UserRoles)
	assert.Equal(t, []string{"manage:systems", "view:logs"}, user.UserPermissions)
	assert.Equal(t, "Distributor", user.OrgRole)
	assert.Equal(t, []string{"create:resellers", "manage:customers"}, user.OrgPermissions)
	assert.Equal(t, "org-123", user.OrganizationID)
	assert.Equal(t, "Test Organization", user.OrganizationName)
}

func TestUserJSONSerialization(t *testing.T) {
	user := User{
		ID:               "json-user-456",
		Username:         "jsonuser",
		Email:            "json@example.com",
		Name:             "JSON User",
		UserRoles:        []string{"Admin"},
		UserPermissions:  []string{"manage:systems"},
		OrgRole:          "Customer",
		OrgPermissions:   []string{"view:systems"},
		OrganizationID:   "org-json",
		OrganizationName: "JSON Organization",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var unmarshaledUser User
	err = json.Unmarshal(jsonData, &unmarshaledUser)
	assert.NoError(t, err)

	// Verify all fields are preserved
	assert.Equal(t, user.ID, unmarshaledUser.ID)
	assert.Equal(t, user.Username, unmarshaledUser.Username)
	assert.Equal(t, user.Email, unmarshaledUser.Email)
	assert.Equal(t, user.Name, unmarshaledUser.Name)
	assert.Equal(t, user.UserRoles, unmarshaledUser.UserRoles)
	assert.Equal(t, user.UserPermissions, unmarshaledUser.UserPermissions)
	assert.Equal(t, user.OrgRole, unmarshaledUser.OrgRole)
	assert.Equal(t, user.OrgPermissions, unmarshaledUser.OrgPermissions)
	assert.Equal(t, user.OrganizationID, unmarshaledUser.OrganizationID)
	assert.Equal(t, user.OrganizationName, unmarshaledUser.OrganizationName)
}

func TestUserJSONTags(t *testing.T) {
	user := User{
		ID:               "tag-test-user",
		Username:         "taguser",
		Email:            "tag@example.com",
		Name:             "Tag User",
		UserRoles:        []string{"Support"},
		UserPermissions:  []string{"view:logs"},
		OrgRole:          "Reseller",
		OrgPermissions:   []string{"manage:customers"},
		OrganizationID:   "org-tag",
		OrganizationName: "Tag Organization",
	}

	jsonData, err := json.Marshal(user)
	assert.NoError(t, err)

	// Parse JSON to verify field names match tags
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	assert.NoError(t, err)

	// Verify JSON field names match struct tags
	assert.Contains(t, jsonMap, "id")
	assert.Contains(t, jsonMap, "username")
	assert.Contains(t, jsonMap, "email")
	assert.Contains(t, jsonMap, "name")
	assert.Contains(t, jsonMap, "user_roles")
	assert.Contains(t, jsonMap, "user_permissions")
	assert.Contains(t, jsonMap, "org_role")
	assert.Contains(t, jsonMap, "org_permissions")
	assert.Contains(t, jsonMap, "organization_id")
	assert.Contains(t, jsonMap, "organization_name")

	// Verify values
	assert.Equal(t, "tag-test-user", jsonMap["id"])
	assert.Equal(t, "taguser", jsonMap["username"])
	assert.Equal(t, "tag@example.com", jsonMap["email"])
	assert.Equal(t, "Tag User", jsonMap["name"])
	assert.Equal(t, "Reseller", jsonMap["org_role"])
	assert.Equal(t, "org-tag", jsonMap["organization_id"])
	assert.Equal(t, "Tag Organization", jsonMap["organization_name"])
}

func TestUserWithEmptyFields(t *testing.T) {
	// Test user with empty string fields
	user := User{
		ID:               "",
		Username:         "",
		Email:            "",
		Name:             "",
		UserRoles:        []string{},
		UserPermissions:  []string{},
		OrgRole:          "",
		OrgPermissions:   []string{},
		OrganizationID:   "",
		OrganizationName: "",
	}

	// Should be valid and serializable
	jsonData, err := json.Marshal(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledUser User
	err = json.Unmarshal(jsonData, &unmarshaledUser)
	assert.NoError(t, err)

	assert.Equal(t, "", unmarshaledUser.ID)
	assert.Equal(t, "", unmarshaledUser.Username)
	assert.Equal(t, "", unmarshaledUser.Email)
	assert.Equal(t, "", unmarshaledUser.Name)
	assert.Equal(t, []string{}, unmarshaledUser.UserRoles)
	assert.Equal(t, []string{}, unmarshaledUser.UserPermissions)
	assert.Equal(t, "", unmarshaledUser.OrgRole)
	assert.Equal(t, []string{}, unmarshaledUser.OrgPermissions)
	assert.Equal(t, "", unmarshaledUser.OrganizationID)
	assert.Equal(t, "", unmarshaledUser.OrganizationName)
}

func TestUserWithNilSlices(t *testing.T) {
	// Test user with nil slices
	user := User{
		ID:               "nil-slices-user",
		Username:         "niluser",
		Email:            "nil@example.com",
		Name:             "Nil User",
		UserRoles:        nil,
		UserPermissions:  nil,
		OrgRole:          "Customer",
		OrgPermissions:   nil,
		OrganizationID:   "org-nil",
		OrganizationName: "Nil Organization",
	}

	// Should be valid and serializable
	jsonData, err := json.Marshal(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledUser User
	err = json.Unmarshal(jsonData, &unmarshaledUser)
	assert.NoError(t, err)

	assert.Equal(t, "nil-slices-user", unmarshaledUser.ID)
	assert.Equal(t, "niluser", unmarshaledUser.Username)
	assert.Equal(t, "nil@example.com", unmarshaledUser.Email)
	assert.Equal(t, "Nil User", unmarshaledUser.Name)
	assert.Nil(t, unmarshaledUser.UserRoles)
	assert.Nil(t, unmarshaledUser.UserPermissions)
	assert.Equal(t, "Customer", unmarshaledUser.OrgRole)
	assert.Nil(t, unmarshaledUser.OrgPermissions)
	assert.Equal(t, "org-nil", unmarshaledUser.OrganizationID)
	assert.Equal(t, "Nil Organization", unmarshaledUser.OrganizationName)
}

func TestUserBusinessRoleTypes(t *testing.T) {
	// Test different business role types
	businessRoles := []string{"Owner", "Distributor", "Reseller", "Customer"}

	for _, role := range businessRoles {
		t.Run("org_role_"+role, func(t *testing.T) {
			user := User{
				ID:      "role-test-" + role,
				OrgRole: role,
			}

			assert.Equal(t, role, user.OrgRole)

			// Verify JSON serialization preserves role
			jsonData, err := json.Marshal(user)
			assert.NoError(t, err)

			var unmarshaledUser User
			err = json.Unmarshal(jsonData, &unmarshaledUser)
			assert.NoError(t, err)
			assert.Equal(t, role, unmarshaledUser.OrgRole)
		})
	}
}

func TestUserTechnicalRoleTypes(t *testing.T) {
	// Test different technical role combinations
	technicalRoleCombinations := [][]string{
		{"Admin"},
		{"Support"},
		{"Admin", "Support"},
		{},
		nil,
	}

	for i, roles := range technicalRoleCombinations {
		t.Run("user_roles_combination_"+string(rune(i+'A')), func(t *testing.T) {
			user := User{
				ID:        "tech-role-test-" + string(rune(i+'A')),
				UserRoles: roles,
			}

			assert.Equal(t, roles, user.UserRoles)

			// Verify JSON serialization preserves roles
			jsonData, err := json.Marshal(user)
			assert.NoError(t, err)

			var unmarshaledUser User
			err = json.Unmarshal(jsonData, &unmarshaledUser)
			assert.NoError(t, err)
			assert.Equal(t, roles, unmarshaledUser.UserRoles)
		})
	}
}

func TestUserPermissionTypes(t *testing.T) {
	// Test different permission combinations
	permissionCombinations := [][]string{
		{"manage:systems"},
		{"view:logs"},
		{"admin:accounts"},
		{"manage:systems", "view:logs", "admin:accounts"},
		{"create:distributors", "manage:resellers"},
		{},
		nil,
	}

	for i, permissions := range permissionCombinations {
		t.Run("permissions_combination_"+string(rune(i+'A')), func(t *testing.T) {
			user := User{
				ID:              "perm-test-" + string(rune(i+'A')),
				UserPermissions: permissions,
				OrgPermissions:  permissions, // Test both user and org permissions
			}

			assert.Equal(t, permissions, user.UserPermissions)
			assert.Equal(t, permissions, user.OrgPermissions)

			// Verify JSON serialization preserves permissions
			jsonData, err := json.Marshal(user)
			assert.NoError(t, err)

			var unmarshaledUser User
			err = json.Unmarshal(jsonData, &unmarshaledUser)
			assert.NoError(t, err)
			assert.Equal(t, permissions, unmarshaledUser.UserPermissions)
			assert.Equal(t, permissions, unmarshaledUser.OrgPermissions)
		})
	}
}

func TestUserCompleteProfile(t *testing.T) {
	// Test a complete user profile with all possible fields populated
	user := User{
		ID:               "complete-user-profile",
		Username:         "completeuser",
		Email:            "complete@nethesis.it",
		Name:             "Complete User Profile",
		UserRoles:        []string{"Admin", "Support"},
		UserPermissions:  []string{"manage:systems", "view:logs", "admin:accounts", "manage:users"},
		OrgRole:          "Owner",
		OrgPermissions:   []string{"create:distributors", "manage:resellers", "admin:system", "global:access"},
		OrganizationID:   "org-nethesis-complete",
		OrganizationName: "Nethesis S.r.l. - Complete",
	}

	// Verify struct integrity
	assert.NotEmpty(t, user.ID)
	assert.NotEmpty(t, user.Username)
	assert.NotEmpty(t, user.Email)
	assert.NotEmpty(t, user.Name)
	assert.NotEmpty(t, user.UserRoles)
	assert.NotEmpty(t, user.UserPermissions)
	assert.NotEmpty(t, user.OrgRole)
	assert.NotEmpty(t, user.OrgPermissions)
	assert.NotEmpty(t, user.OrganizationID)
	assert.NotEmpty(t, user.OrganizationName)

	// Verify JSON round-trip
	jsonData, err := json.Marshal(user)
	assert.NoError(t, err)

	var unmarshaledUser User
	err = json.Unmarshal(jsonData, &unmarshaledUser)
	assert.NoError(t, err)

	// Verify all fields match exactly
	assert.Equal(t, user, unmarshaledUser)
}

func TestUserPointerOperations(t *testing.T) {
	// Test operations with User pointers (common in application usage)
	user := &User{
		ID:       "pointer-user",
		Username: "pointeruser",
		Email:    "pointer@example.com",
	}

	assert.NotNil(t, user)
	assert.Equal(t, "pointer-user", user.ID)
	assert.Equal(t, "pointeruser", user.Username)
	assert.Equal(t, "pointer@example.com", user.Email)

	// Test JSON serialization with pointer
	jsonData, err := json.Marshal(user)
	assert.NoError(t, err)

	var unmarshaledUser *User
	err = json.Unmarshal(jsonData, &unmarshaledUser)
	assert.NoError(t, err)
	assert.NotNil(t, unmarshaledUser)
	assert.Equal(t, user.ID, unmarshaledUser.ID)
	assert.Equal(t, user.Username, unmarshaledUser.Username)
	assert.Equal(t, user.Email, unmarshaledUser.Email)
}
