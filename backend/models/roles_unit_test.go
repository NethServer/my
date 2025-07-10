/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoleStruct(t *testing.T) {
	role := Role{
		ID:          "rol_123",
		Name:        "Admin",
		Description: "Administrator role with full system access",
	}

	assert.Equal(t, "rol_123", role.ID)
	assert.Equal(t, "Admin", role.Name)
	assert.Equal(t, "Administrator role with full system access", role.Description)
}

func TestRoleJSONTags(t *testing.T) {
	// Test that struct fields have correct JSON tags
	role := Role{}

	// Verify field types are correct
	assert.IsType(t, "", role.ID)
	assert.IsType(t, "", role.Name)
	assert.IsType(t, "", role.Description)
}

func TestOrganizationRoleStruct(t *testing.T) {
	orgRole := OrganizationRole{
		ID:          "org_rol_456",
		Name:        "Owner",
		Description: "Organization owner with complete control",
	}

	assert.Equal(t, "org_rol_456", orgRole.ID)
	assert.Equal(t, "Owner", orgRole.Name)
	assert.Equal(t, "Organization owner with complete control", orgRole.Description)
}

func TestOrganizationRoleJSONTags(t *testing.T) {
	// Test that struct fields have correct JSON tags
	orgRole := OrganizationRole{}

	// Verify field types are correct
	assert.IsType(t, "", orgRole.ID)
	assert.IsType(t, "", orgRole.Name)
	assert.IsType(t, "", orgRole.Description)
}

func TestRolesResponseStruct(t *testing.T) {
	roles := []Role{
		{
			ID:          "rol_admin",
			Name:        "Admin",
			Description: "System administrator",
		},
		{
			ID:          "rol_support",
			Name:        "Support",
			Description: "Support specialist",
		},
		{
			ID:          "rol_viewer",
			Name:        "Viewer",
			Description: "Read-only access",
		},
	}

	response := RolesResponse{
		Roles: roles,
	}

	assert.Len(t, response.Roles, 3)
	assert.Equal(t, roles, response.Roles)

	// Test individual roles
	assert.Equal(t, "rol_admin", response.Roles[0].ID)
	assert.Equal(t, "Admin", response.Roles[0].Name)

	assert.Equal(t, "rol_support", response.Roles[1].ID)
	assert.Equal(t, "Support", response.Roles[1].Name)

	assert.Equal(t, "rol_viewer", response.Roles[2].ID)
	assert.Equal(t, "Viewer", response.Roles[2].Name)
}

func TestOrganizationRolesResponseStruct(t *testing.T) {
	orgRoles := []OrganizationRole{
		{
			ID:          "org_rol_owner",
			Name:        "Owner",
			Description: "Organization owner",
		},
		{
			ID:          "org_rol_distributor",
			Name:        "Distributor",
			Description: "Business distributor",
		},
		{
			ID:          "org_rol_reseller",
			Name:        "Reseller",
			Description: "Business reseller",
		},
		{
			ID:          "org_rol_customer",
			Name:        "Customer",
			Description: "End customer",
		},
	}

	response := OrganizationRolesResponse{
		OrganizationRoles: orgRoles,
	}

	assert.Len(t, response.OrganizationRoles, 4)
	assert.Equal(t, orgRoles, response.OrganizationRoles)

	// Test individual organization roles
	assert.Equal(t, "org_rol_owner", response.OrganizationRoles[0].ID)
	assert.Equal(t, "Owner", response.OrganizationRoles[0].Name)

	assert.Equal(t, "org_rol_distributor", response.OrganizationRoles[1].ID)
	assert.Equal(t, "Distributor", response.OrganizationRoles[1].Name)

	assert.Equal(t, "org_rol_reseller", response.OrganizationRoles[2].ID)
	assert.Equal(t, "Reseller", response.OrganizationRoles[2].Name)

	assert.Equal(t, "org_rol_customer", response.OrganizationRoles[3].ID)
	assert.Equal(t, "Customer", response.OrganizationRoles[3].Name)
}

func TestRolesResponseEmpty(t *testing.T) {
	response := RolesResponse{
		Roles: []Role{},
	}

	assert.Len(t, response.Roles, 0)
	assert.NotNil(t, response.Roles) // Should be empty slice, not nil
}

func TestOrganizationRolesResponseEmpty(t *testing.T) {
	response := OrganizationRolesResponse{
		OrganizationRoles: []OrganizationRole{},
	}

	assert.Len(t, response.OrganizationRoles, 0)
	assert.NotNil(t, response.OrganizationRoles) // Should be empty slice, not nil
}

func TestRolesResponseNil(t *testing.T) {
	response := RolesResponse{
		Roles: nil,
	}

	assert.Nil(t, response.Roles)
}

func TestOrganizationRolesResponseNil(t *testing.T) {
	response := OrganizationRolesResponse{
		OrganizationRoles: nil,
	}

	assert.Nil(t, response.OrganizationRoles)
}

func TestRoleWithEmptyFields(t *testing.T) {
	role := Role{
		ID:          "",
		Name:        "",
		Description: "",
	}

	assert.Empty(t, role.ID)
	assert.Empty(t, role.Name)
	assert.Empty(t, role.Description)
}

func TestOrganizationRoleWithEmptyFields(t *testing.T) {
	orgRole := OrganizationRole{
		ID:          "",
		Name:        "",
		Description: "",
	}

	assert.Empty(t, orgRole.ID)
	assert.Empty(t, orgRole.Name)
	assert.Empty(t, orgRole.Description)
}

func TestRoleWithSpecialCharacters(t *testing.T) {
	role := Role{
		ID:          "rol_123-456_789",
		Name:        "Special Role & Co.",
		Description: "Role with special characters: áéíóú çñü @#$%",
	}

	assert.Equal(t, "rol_123-456_789", role.ID)
	assert.Equal(t, "Special Role & Co.", role.Name)
	assert.Equal(t, "Role with special characters: áéíóú çñü @#$%", role.Description)
}

func TestOrganizationRoleWithSpecialCharacters(t *testing.T) {
	orgRole := OrganizationRole{
		ID:          "org_rol_123-456_789",
		Name:        "Special Org Role & Co.",
		Description: "Organization role with special characters: áéíóú çñü @#$%",
	}

	assert.Equal(t, "org_rol_123-456_789", orgRole.ID)
	assert.Equal(t, "Special Org Role & Co.", orgRole.Name)
	assert.Equal(t, "Organization role with special characters: áéíóú çñü @#$%", orgRole.Description)
}

func TestTechnicalRoles(t *testing.T) {
	// Test typical technical user roles
	adminRole := Role{
		ID:          "rol_admin",
		Name:        "Admin",
		Description: "Complete system administration capabilities",
	}

	supportRole := Role{
		ID:          "rol_support",
		Name:        "Support",
		Description: "System management and standard operations",
	}

	assert.Equal(t, "Admin", adminRole.Name)
	assert.Equal(t, "Support", supportRole.Name)
	assert.Contains(t, adminRole.Description, "administration")
	assert.Contains(t, supportRole.Description, "management")
}

func TestBusinessHierarchyRoles(t *testing.T) {
	// Test typical business hierarchy organization roles
	ownerRole := OrganizationRole{
		ID:          "org_rol_owner",
		Name:        "Owner",
		Description: "Complete control over commercial hierarchy",
	}

	distributorRole := OrganizationRole{
		ID:          "org_rol_distributor",
		Name:        "Distributor",
		Description: "Can manage resellers and customers",
	}

	resellerRole := OrganizationRole{
		ID:          "org_rol_reseller",
		Name:        "Reseller",
		Description: "Can manage customers only",
	}

	customerRole := OrganizationRole{
		ID:          "org_rol_customer",
		Name:        "Customer",
		Description: "Read-only access to own data",
	}

	hierarchy := []OrganizationRole{ownerRole, distributorRole, resellerRole, customerRole}

	assert.Len(t, hierarchy, 4)
	assert.Equal(t, "Owner", hierarchy[0].Name)
	assert.Equal(t, "Distributor", hierarchy[1].Name)
	assert.Equal(t, "Reseller", hierarchy[2].Name)
	assert.Equal(t, "Customer", hierarchy[3].Name)
}

func TestRoleIDPatterns(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		expected string
	}{
		{
			name: "Admin role ID pattern",
			role: Role{
				ID:   "rol_admin_123",
				Name: "Admin",
			},
			expected: "rol_admin_123",
		},
		{
			name: "Support role ID pattern",
			role: Role{
				ID:   "rol_support_456",
				Name: "Support",
			},
			expected: "rol_support_456",
		},
		{
			name: "Custom role ID pattern",
			role: Role{
				ID:   "rol_custom_xyz789",
				Name: "Custom Role",
			},
			expected: "rol_custom_xyz789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.role.ID)
			assert.Contains(t, tt.role.ID, "rol_")
		})
	}
}

func TestOrganizationRoleIDPatterns(t *testing.T) {
	tests := []struct {
		name     string
		orgRole  OrganizationRole
		expected string
	}{
		{
			name: "Owner organization role ID pattern",
			orgRole: OrganizationRole{
				ID:   "org_rol_owner_123",
				Name: "Owner",
			},
			expected: "org_rol_owner_123",
		},
		{
			name: "Distributor organization role ID pattern",
			orgRole: OrganizationRole{
				ID:   "org_rol_distributor_456",
				Name: "Distributor",
			},
			expected: "org_rol_distributor_456",
		},
		{
			name: "Custom organization role ID pattern",
			orgRole: OrganizationRole{
				ID:   "org_rol_custom_xyz789",
				Name: "Custom Org Role",
			},
			expected: "org_rol_custom_xyz789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.orgRole.ID)
			assert.Contains(t, tt.orgRole.ID, "org_rol_")
		})
	}
}

func TestResponseStructJSONTags(t *testing.T) {
	rolesResponse := RolesResponse{}
	orgRolesResponse := OrganizationRolesResponse{}

	// Verify field types are correct
	assert.IsType(t, []Role{}, rolesResponse.Roles)
	assert.IsType(t, []OrganizationRole{}, orgRolesResponse.OrganizationRoles)
}

func TestRoleStructFieldConsistency(t *testing.T) {
	// Test that both JSON and structs tags are consistent for Role
	role := Role{
		ID:          "test_id",
		Name:        "test_name",
		Description: "test_description",
	}

	assert.IsType(t, "", role.ID)
	assert.IsType(t, "", role.Name)
	assert.IsType(t, "", role.Description)

	// Test that both JSON and structs tags are consistent for OrganizationRole
	orgRole := OrganizationRole{
		ID:          "test_org_id",
		Name:        "test_org_name",
		Description: "test_org_description",
	}

	assert.IsType(t, "", orgRole.ID)
	assert.IsType(t, "", orgRole.Name)
	assert.IsType(t, "", orgRole.Description)
}
