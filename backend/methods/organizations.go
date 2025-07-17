/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services"
)

// GetOrganizations returns organizations the current user can assign users to with pagination and search
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

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Parse filters
	filters := models.OrganizationFilters{
		Search:      c.Query("search"),
		Name:        c.Query("name"),
		Description: c.Query("description"),
		Type:        c.Query("type"),
		CreatedBy:   c.Query("created_by"),
	}

	// Connect to Logto Management API
	client := services.NewLogtoManagementClient()

	// Get organizations with pagination and filters
	result, err := client.GetOrganizationsPaginated(page, pageSize, filters)
	if err != nil {
		httpLogger := logger.NewHTTPErrorLogger(c, "organizations")
		httpLogger.LogError(err, "fetch_organizations", http.StatusInternalServerError, "Failed to fetch organizations")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to fetch organizations", nil))
		return
	}

	// Filter organizations based on user's role and hierarchy
	filteredOrgs := filterOrganizationsByHierarchy(result.Data, userOrgRole, userOrgID)

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

	// Update pagination info based on filtering
	updatedPagination := result.Pagination
	updatedPagination.TotalCount = len(organizations)
	updatedPagination.TotalPages = (updatedPagination.TotalCount + pageSize - 1) / pageSize

	logger.ComponentLogger("organizations").Info().
		Str("operation", "get_organizations").
		Str("user_org_role", userOrgRole).
		Str("user_org_id", userOrgID).
		Int("total_orgs", len(result.Data)).
		Int("filtered_orgs", len(organizations)).
		Int("page", page).
		Int("page_size", pageSize).
		Str("search", filters.Search).
		Msg("Organizations filtered and returned with pagination")

	c.JSON(http.StatusOK, response.OK("organizations retrieved successfully", models.PaginatedOrganizationsResponse{
		Organizations: organizations,
		Pagination:    updatedPagination,
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
	// Check for explicit type field in custom data
	if org.CustomData != nil {
		if orgType, exists := org.CustomData["type"]; exists {
			if typeStr, ok := orgType.(string); ok {
				return typeStr
			}
		}
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
