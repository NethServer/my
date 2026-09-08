/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
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

	userOrgRole := strings.ToLower(currentUserOrgRole.(string))
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

	// Parse sorting parameters
	sortBy, sortDirection := helpers.GetSortingFromQuery(c)

	// Parse filters
	filters := models.OrganizationFilters{
		Search:      c.Query("search"),
		Name:        c.Query("name"),
		Description: c.Query("description"),
		Types:       c.QueryArray("type"),
		CreatedBy:   c.Query("created_by"),
	}

	// Use local organization service for better performance
	service := local.NewOrganizationService()

	// Get organizations with pagination and filters from local database
	result, err := service.GetAllOrganizationsPaginated(userOrgRole, userOrgID, page, pageSize, sortBy, sortDirection, filters)
	if err != nil {
		httpLogger := logger.NewHTTPErrorLogger(c, "organizations")
		httpLogger.LogError(err, "fetch_organizations", http.StatusInternalServerError, "Failed to fetch organizations")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch organizations", nil))
		return
	}

	// Pin the Owner organization itself on the first page for callers holding
	// the Owner user role (the only ones who can create users there): it exists
	// only in Logto (never in the local tables), and it is the org where
	// Staff/Owner users are created. Synthetic entry — no local UUID, logto_id
	// is the caller's own organization. Skipped when the filters exclude it.
	callerUserRoles := []string{}
	if v, exists := c.Get("user_roles"); exists {
		if roles, isSlice := v.([]string); isSlice {
			callerUserRoles = roles
		}
	}
	var ownerEntry []models.OrganizationSummary
	if page == 1 && userOrgRole == "owner" && models.HasOwnerUserRole(callerUserRoles) && ownerEntryMatchesFilters(c, filters) {
		ownerName := "Owner"
		if n, exists := c.Get("organization_name"); exists {
			if s, isStr := n.(string); isStr && s != "" {
				ownerName = s
			}
		}
		ownerEntry = []models.OrganizationSummary{{
			ID:      "",
			LogtoID: userOrgID,
			Name:    ownerName,
			Type:    "owner",
		}}
	}

	// Convert to response format (no additional filtering needed - RBAC already applied by repositories)
	organizations := make([]models.OrganizationSummary, 0, len(result.Data)+len(ownerEntry))
	organizations = append(organizations, ownerEntry...)
	for _, org := range result.Data {
		// Extract database_id from CustomData
		databaseID := ""
		if org.CustomData != nil {
			if dbID, ok := org.CustomData["database_id"].(string); ok {
				databaseID = dbID
			}
		}

		organizations = append(organizations, models.OrganizationSummary{
			ID:          databaseID, // Database UUID
			LogtoID:     org.ID,     // Logto ID
			Name:        org.Name,
			Description: org.Description,
			Type:        getOrganizationType(org),
		})
	}

	logger.ComponentLogger("organizations").Info().
		Str("operation", "get_organizations").
		Str("user_org_role", userOrgRole).
		Str("user_org_id", userOrgID).
		Int("returned_orgs", len(organizations)).
		Int("page", page).
		Int("page_size", pageSize).
		Str("sort_by", sortBy).
		Str("sort_direction", sortDirection).
		Str("search", filters.Search).
		Msg("Organizations retrieved from local database with pagination")

	c.JSON(http.StatusOK, response.OK("organizations retrieved successfully", models.PaginatedOrganizationsResponse{
		Organizations: organizations,
		Pagination:    result.Pagination,
	}))
}

// ownerEntryMatchesFilters reports whether the synthetic Owner entry survives
// the active list filters: it is excluded by a type filter without "owner", by
// a search/name term that does not match the org name, and by any created_by
// or description filter (the Owner organization has neither).
func ownerEntryMatchesFilters(c *gin.Context, filters models.OrganizationFilters) bool {
	if filters.CreatedBy != "" || filters.Description != "" {
		return false
	}
	if len(filters.Types) > 0 {
		found := false
		for _, t := range filters.Types {
			if strings.EqualFold(t, "owner") {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	ownerName := "Owner"
	if n, exists := c.Get("organization_name"); exists {
		if s, isStr := n.(string); isStr && s != "" {
			ownerName = s
		}
	}
	for _, term := range []string{filters.Search, filters.Name} {
		if term != "" && !strings.Contains(strings.ToLower(ownerName), strings.ToLower(term)) {
			return false
		}
	}
	return true
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
