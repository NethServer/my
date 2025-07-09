/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services"
)

// GetOrganizations returns organizations the current user can assign users to
func GetOrganizations(c *gin.Context) {
	// Get current user context
	currentUserOrgRole, _ := c.Get("org_role")
	currentUserOrgID, _ := c.Get("organization_id")

	// Validate required user context
	if currentUserOrgRole == nil || currentUserOrgID == nil {
		logger.NewHTTPErrorLogger(c, "organizations").LogError(nil, "validate_user_context", http.StatusUnauthorized, "Missing user context")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("incomplete user context in token", nil))
		return
	}

	userOrgRole := currentUserOrgRole.(string)
	userOrgID := currentUserOrgID.(string)

	// Connect to Logto Management API
	client := services.NewLogtoManagementClient()

	// Get all organizations
	allOrgs, err := client.GetOrganizations()
	if err != nil {
		httpLogger := logger.NewHTTPErrorLogger(c, "organizations")
		httpLogger.LogError(err, "fetch_organizations", http.StatusInternalServerError, "Failed to fetch organizations")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to fetch organizations", nil))
		return
	}

	// Filter organizations based on user's role and hierarchy
	filteredOrgs := filterOrganizationsByHierarchy(allOrgs, userOrgRole, userOrgID)

	// Convert to response format
	organizations := make([]models.OrganizationSummary, len(filteredOrgs))
	for i, org := range filteredOrgs {
		organizations[i] = models.OrganizationSummary{
			ID:          org.ID,
			Name:        org.Name,
			Description: org.Description,
			Type:        getOrganizationType(org),
		}
	}

	logger.ComponentLogger("organizations").Info().
		Str("operation", "get_organizations").
		Str("user_org_role", userOrgRole).
		Str("user_org_id", userOrgID).
		Int("total_orgs", len(allOrgs)).
		Int("filtered_orgs", len(organizations)).
		Msg("Organizations filtered and returned")

	c.JSON(http.StatusOK, response.OK("organizations retrieved successfully", models.OrganizationsResponse{
		Organizations: organizations,
	}))
}

// filterOrganizationsByHierarchy filters organizations based on user hierarchy
func filterOrganizationsByHierarchy(orgs []models.LogtoOrganization, userOrgRole, userOrgID string) []models.LogtoOrganization {
	var filtered []models.LogtoOrganization

	for _, org := range orgs {
		orgType := getOrganizationType(org)

		switch userOrgRole {
		case "Owner":
			// Owner can assign users to any organization
			filtered = append(filtered, org)

		case "Distributor":
			// Distributor can assign users to:
			// - Their own organization
			// - Reseller organizations they created
			// - Customer organizations (directly or through resellers they created)
			if org.ID == userOrgID {
				// Own organization
				filtered = append(filtered, org)
			} else if orgType == "reseller" || orgType == "customer" {
				// Check if this organization was created by the current distributor
				if isOrganizationCreatedBy(org, userOrgID) {
					filtered = append(filtered, org)
				}
			}

		case "Reseller":
			// Reseller can assign users to:
			// - Their own organization
			// - Customer organizations they created
			if org.ID == userOrgID {
				// Own organization
				filtered = append(filtered, org)
			} else if orgType == "customer" {
				// Check if this customer organization was created by the current reseller
				if isOrganizationCreatedBy(org, userOrgID) {
					filtered = append(filtered, org)
				}
			}

		case "Customer":
			// Customer can only assign users to their own organization
			if org.ID == userOrgID {
				filtered = append(filtered, org)
			}
		}
	}

	return filtered
}

// getOrganizationType determines the type of organization based on custom data
func getOrganizationType(org models.LogtoOrganization) string {
	// Priority 1: Check for explicit type field in custom data
	if org.CustomData != nil {
		if orgType, exists := org.CustomData["type"]; exists {
			if typeStr, ok := orgType.(string); ok {
				return typeStr
			}
		}

		// Priority 2: Check for organizationType field in custom data (legacy)
		if orgType, exists := org.CustomData["organizationType"]; exists {
			if typeStr, ok := orgType.(string); ok {
				return typeStr
			}
		}
	}

	// Priority 3: Fallback to name-based detection (legacy)
	orgName := strings.ToLower(org.Name)
	if strings.Contains(orgName, "distributor") {
		return "distributor"
	}
	if strings.Contains(orgName, "reseller") {
		return "reseller"
	}
	if strings.Contains(orgName, "customer") {
		return "customer"
	}

	// Default to customer type if no type can be determined
	return "customer"
}

// isOrganizationCreatedBy checks if an organization was created by the specified creator
func isOrganizationCreatedBy(org models.LogtoOrganization, creatorOrgID string) bool {
	if org.CustomData != nil {
		if createdBy, exists := org.CustomData["createdBy"]; exists {
			if createdByStr, ok := createdBy.(string); ok {
				return createdByStr == creatorOrgID
			}
		}
	}
	return false
}
