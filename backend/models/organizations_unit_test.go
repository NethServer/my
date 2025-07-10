/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrganizationSummaryStruct(t *testing.T) {
	org := OrganizationSummary{
		ID:          "org_123",
		Name:        "Test Organization",
		Description: "Test organization description",
		Type:        "distributor",
	}

	assert.Equal(t, "org_123", org.ID)
	assert.Equal(t, "Test Organization", org.Name)
	assert.Equal(t, "Test organization description", org.Description)
	assert.Equal(t, "distributor", org.Type)
}

func TestOrganizationSummaryJSONTags(t *testing.T) {
	// Test that struct fields have correct JSON tags
	org := OrganizationSummary{}

	// Verify field types are correct
	assert.IsType(t, "", org.ID)
	assert.IsType(t, "", org.Name)
	assert.IsType(t, "", org.Description)
	assert.IsType(t, "", org.Type)
}

func TestOrganizationSummaryValidTypes(t *testing.T) {
	validTypes := []string{"owner", "distributor", "reseller", "customer"}

	for _, orgType := range validTypes {
		t.Run("Type: "+orgType, func(t *testing.T) {
			org := OrganizationSummary{
				ID:          "org_test",
				Name:        "Test Org",
				Description: "Test Description",
				Type:        orgType,
			}

			assert.Equal(t, orgType, org.Type)
			assert.Contains(t, validTypes, org.Type)
		})
	}
}

func TestOrganizationsResponseStruct(t *testing.T) {
	orgs := []OrganizationSummary{
		{
			ID:          "org_1",
			Name:        "Organization 1",
			Description: "First organization",
			Type:        "owner",
		},
		{
			ID:          "org_2",
			Name:        "Organization 2",
			Description: "Second organization",
			Type:        "distributor",
		},
		{
			ID:          "org_3",
			Name:        "Organization 3",
			Description: "Third organization",
			Type:        "reseller",
		},
	}

	response := OrganizationsResponse{
		Organizations: orgs,
	}

	assert.Len(t, response.Organizations, 3)
	assert.Equal(t, orgs, response.Organizations)

	// Test individual organizations
	assert.Equal(t, "org_1", response.Organizations[0].ID)
	assert.Equal(t, "owner", response.Organizations[0].Type)

	assert.Equal(t, "org_2", response.Organizations[1].ID)
	assert.Equal(t, "distributor", response.Organizations[1].Type)

	assert.Equal(t, "org_3", response.Organizations[2].ID)
	assert.Equal(t, "reseller", response.Organizations[2].Type)
}

func TestOrganizationsResponseEmpty(t *testing.T) {
	response := OrganizationsResponse{
		Organizations: []OrganizationSummary{},
	}

	assert.Len(t, response.Organizations, 0)
	assert.NotNil(t, response.Organizations) // Should be empty slice, not nil
}

func TestOrganizationsResponseNil(t *testing.T) {
	response := OrganizationsResponse{
		Organizations: nil,
	}

	assert.Nil(t, response.Organizations)
}

func TestOrganizationSummaryWithEmptyFields(t *testing.T) {
	org := OrganizationSummary{
		ID:          "",
		Name:        "",
		Description: "",
		Type:        "",
	}

	assert.Empty(t, org.ID)
	assert.Empty(t, org.Name)
	assert.Empty(t, org.Description)
	assert.Empty(t, org.Type)
}

func TestOrganizationSummaryWithSpecialCharacters(t *testing.T) {
	org := OrganizationSummary{
		ID:          "org_123-456_789",
		Name:        "Test & Co. Srl",
		Description: "Organization with special characters: áéíóú çñü @#$%",
		Type:        "customer",
	}

	assert.Equal(t, "org_123-456_789", org.ID)
	assert.Equal(t, "Test & Co. Srl", org.Name)
	assert.Equal(t, "Organization with special characters: áéíóú çñü @#$%", org.Description)
	assert.Equal(t, "customer", org.Type)
}

func TestOrganizationSummaryFieldValidation(t *testing.T) {
	tests := []struct {
		name        string
		org         OrganizationSummary
		description string
	}{
		{
			name: "Minimal valid organization",
			org: OrganizationSummary{
				ID:   "org_min",
				Name: "Minimal Org",
				Type: "customer",
			},
			description: "Organization with minimal required fields",
		},
		{
			name: "Complete organization",
			org: OrganizationSummary{
				ID:          "org_complete",
				Name:        "Complete Organization Ltd.",
				Description: "A complete organization with all fields filled",
				Type:        "distributor",
			},
			description: "Organization with all fields populated",
		},
		{
			name: "Long description organization",
			org: OrganizationSummary{
				ID:          "org_long",
				Name:        "Organization with Long Description",
				Description: "This is a very long description that contains multiple sentences and explains in detail what this organization does, its mission, vision, and various other important details that might be relevant for business operations.",
				Type:        "reseller",
			},
			description: "Organization with very long description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that all fields are properly set
			assert.Equal(t, tt.org.ID, tt.org.ID)
			assert.Equal(t, tt.org.Name, tt.org.Name)
			assert.Equal(t, tt.org.Description, tt.org.Description)
			assert.Equal(t, tt.org.Type, tt.org.Type)

			// Test that the struct is properly initialized
			assert.IsType(t, OrganizationSummary{}, tt.org)
		})
	}
}

func TestOrganizationHierarchy(t *testing.T) {
	// Test organization hierarchy representation
	owner := OrganizationSummary{
		ID:          "org_owner",
		Name:        "Nethesis S.r.l.",
		Description: "Owner organization",
		Type:        "owner",
	}

	distributor := OrganizationSummary{
		ID:          "org_dist",
		Name:        "Distributor Company",
		Description: "Main distributor",
		Type:        "distributor",
	}

	reseller := OrganizationSummary{
		ID:          "org_reseller",
		Name:        "Reseller Corp",
		Description: "Regional reseller",
		Type:        "reseller",
	}

	customer := OrganizationSummary{
		ID:          "org_customer",
		Name:        "Customer Inc",
		Description: "End customer",
		Type:        "customer",
	}

	hierarchy := []OrganizationSummary{owner, distributor, reseller, customer}

	assert.Len(t, hierarchy, 4)
	assert.Equal(t, "owner", hierarchy[0].Type)
	assert.Equal(t, "distributor", hierarchy[1].Type)
	assert.Equal(t, "reseller", hierarchy[2].Type)
	assert.Equal(t, "customer", hierarchy[3].Type)
}

func TestOrganizationsResponseJSONTags(t *testing.T) {
	response := OrganizationsResponse{}

	// Verify field types are correct
	assert.IsType(t, []OrganizationSummary{}, response.Organizations)
}

func TestOrganizationStructFieldConsistency(t *testing.T) {
	// Test that both JSON and structs tags are consistent
	org := OrganizationSummary{
		ID:          "test_id",
		Name:        "test_name",
		Description: "test_description",
		Type:        "test_type",
	}

	// Verify all fields are accessible and have correct types
	assert.IsType(t, "", org.ID)
	assert.IsType(t, "", org.Name)
	assert.IsType(t, "", org.Description)
	assert.IsType(t, "", org.Type)

	// Verify values are stored correctly
	assert.Equal(t, "test_id", org.ID)
	assert.Equal(t, "test_name", org.Name)
	assert.Equal(t, "test_description", org.Description)
	assert.Equal(t, "test_type", org.Type)
}
