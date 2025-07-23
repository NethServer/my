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

	// Parse filters
	filters := models.OrganizationFilters{
		Search:      c.Query("search"),
		Name:        c.Query("name"),
		Description: c.Query("description"),
		Type:        c.Query("type"),
		CreatedBy:   c.Query("created_by"),
	}

	// Use local organization service for better performance
	service := services.NewLocalOrganizationService()

	// Get organizations with pagination and filters from local database
	result, err := service.GetAllOrganizationsPaginated(userOrgRole, userOrgID, page, pageSize, filters)
	if err != nil {
		httpLogger := logger.NewHTTPErrorLogger(c, "organizations")
		httpLogger.LogError(err, "fetch_organizations", http.StatusInternalServerError, "Failed to fetch organizations")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to fetch organizations", nil))
		return
	}

	// Convert to response format (no additional filtering needed - RBAC already applied by repositories)
	organizations := make([]models.OrganizationSummary, len(result.Data))
	for i, org := range result.Data {
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
		Int("returned_orgs", len(organizations)).
		Int("page", page).
		Int("page_size", pageSize).
		Str("search", filters.Search).
		Msg("Organizations retrieved from local database with pagination")

	c.JSON(http.StatusOK, response.OK("organizations retrieved successfully", models.PaginatedOrganizationsResponse{
		Organizations: organizations,
		Pagination:    result.Pagination,
	}))
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
