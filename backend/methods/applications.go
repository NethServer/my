/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
)

// GetApplications handles GET /api/applications - retrieves all applications with pagination
func GetApplications(c *gin.Context) {
	// Get current user context with organization ID
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Parse pagination and sorting parameters
	page, pageSize, sortBy, sortDirection := helpers.GetPaginationAndSortingFromQuery(c)

	// Override default page size for applications
	if c.Query("page_size") == "" {
		pageSize = 50
	}

	// Parse search parameter
	search := c.Query("search")

	// Parse filter parameters (supporting multiple values)
	filterTypes := c.QueryArray("type")
	filterVersions := c.QueryArray("version")
	filterSystemIDs := c.QueryArray("system_id")
	filterOrgIDs := c.QueryArray("organization_id")
	filterStatuses := c.QueryArray("status")

	// Create applications service
	appsService := local.NewApplicationsService()

	// Get applications with pagination, search, sorting and filters
	apps, totalCount, err := appsService.GetApplications(
		userOrgRole, userOrgID,
		page, pageSize,
		search, sortBy, sortDirection,
		filterTypes, filterVersions, filterSystemIDs, filterOrgIDs, filterStatuses,
	)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Int("page", page).
			Int("page_size", pageSize).
			Msg("Failed to get applications")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve applications", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Batch resolve rebranding info (eliminates N+1 queries)
	var orgIDsForRebranding []string
	for _, app := range apps {
		if app.OrganizationID != nil && *app.OrganizationID != "" {
			orgIDsForRebranding = append(orgIDsForRebranding, *app.OrganizationID)
		}
	}
	rebrandingService := local.NewRebrandingService()
	rebrandingResults := rebrandingService.BatchResolveRebranding(orgIDsForRebranding)
	for i := range apps {
		if apps[i].OrganizationID != nil && *apps[i].OrganizationID != "" {
			if res, ok := rebrandingResults[*apps[i].OrganizationID]; ok && res.Enabled {
				apps[i].RebrandingEnabled = true
				apps[i].RebrandingOrgID = &res.ResolvedOrgID
			}
		}
	}

	// Convert to list items for response
	applications := make([]*models.ApplicationListItem, len(apps))
	for i, app := range apps {
		applications[i] = app.ToListItem()
	}

	c.JSON(http.StatusOK, response.OK("applications retrieved successfully", gin.H{
		"applications": applications,
		"pagination":   helpers.BuildPaginationInfoWithSorting(page, pageSize, totalCount, sortBy, sortDirection),
	}))
}

// GetApplication handles GET /api/applications/:id - retrieves a single application
func GetApplication(c *gin.Context) {
	appID := c.Param("id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("application id is required", nil))
		return
	}

	// Get current user context
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Create applications service
	appsService := local.NewApplicationsService()

	// Get application with access validation
	app, err := appsService.GetApplication(appID, userOrgRole, userOrgID)
	if helpers.HandleAccessError(c, err, "application", appID) {
		return
	}

	// Resolve rebranding info
	if app.OrganizationID != nil && *app.OrganizationID != "" {
		app.RebrandingEnabled, app.RebrandingOrgID = resolveRebranding(*app.OrganizationID)
	}

	c.JSON(http.StatusOK, response.OK("application retrieved successfully", app))
}

// UpdateApplication handles PUT /api/applications/:id - updates an application
func UpdateApplication(c *gin.Context) {
	appID := c.Param("id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("application id is required", nil))
		return
	}

	// Parse request body
	var request models.UpdateApplicationRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	// Get current user context
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Create applications service
	appsService := local.NewApplicationsService()

	// Update application
	err := appsService.UpdateApplication(appID, &request, userOrgRole, userOrgID)
	if helpers.HandleAccessError(c, err, "application", appID) {
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "applications", "update", "application", appID, true, nil)

	// Get updated application
	app, err := appsService.GetApplication(appID, userOrgRole, userOrgID)
	if err != nil {
		c.JSON(http.StatusOK, response.OK("application updated successfully", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("application updated successfully", app))
}

// AssignApplicationOrganization handles PATCH /api/applications/:id/assign - assigns organization
func AssignApplicationOrganization(c *gin.Context) {
	appID := c.Param("id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("application id is required", nil))
		return
	}

	// Parse request body
	var request models.AssignApplicationRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	// Get current user context
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Create applications service
	appsService := local.NewApplicationsService()

	// Assign organization
	err := appsService.AssignOrganization(appID, &request, userOrgRole, userOrgID)
	if helpers.HandleAccessError(c, err, "application", appID) {
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "applications", "assign", "application", appID, true, nil)

	// Get updated application
	app, err := appsService.GetApplication(appID, userOrgRole, userOrgID)
	if err != nil {
		c.JSON(http.StatusOK, response.OK("organization assigned successfully", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("organization assigned successfully", app))
}

// UnassignApplicationOrganization handles PATCH /api/applications/:id/unassign - removes organization
func UnassignApplicationOrganization(c *gin.Context) {
	appID := c.Param("id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("application id is required", nil))
		return
	}

	// Get current user context
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Create applications service
	appsService := local.NewApplicationsService()

	// Unassign organization
	err := appsService.UnassignOrganization(appID, userOrgRole, userOrgID)
	if helpers.HandleAccessError(c, err, "application", appID) {
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "applications", "unassign", "application", appID, true, nil)

	// Get updated application
	app, err := appsService.GetApplication(appID, userOrgRole, userOrgID)
	if err != nil {
		c.JSON(http.StatusOK, response.OK("organization unassigned successfully", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("organization unassigned successfully", app))
}

// DeleteApplication handles DELETE /api/applications/:id - soft deletes an application
func DeleteApplication(c *gin.Context) {
	appID := c.Param("id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("application id is required", nil))
		return
	}

	// Get current user context
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Create applications service
	appsService := local.NewApplicationsService()

	// Delete application
	err := appsService.DeleteApplication(appID, userOrgRole, userOrgID)
	if helpers.HandleAccessError(c, err, "application", appID) {
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "applications", "delete", "application", appID, true, nil)

	c.JSON(http.StatusOK, response.OK("application deleted successfully", nil))
}

// GetApplicationTotals handles GET /api/applications/totals - returns statistics
func GetApplicationTotals(c *gin.Context) {
	// Get current user context
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Create applications service
	appsService := local.NewApplicationsService()

	// Get totals
	totals, err := appsService.GetApplicationTotals(userOrgRole, userOrgID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to get application totals")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve application totals", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	c.JSON(http.StatusOK, response.OK("application totals retrieved successfully", totals))
}

// GetApplicationTypes handles GET /api/applications/types - returns available types
func GetApplicationTypes(c *gin.Context) {
	// Get current user context
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Create applications service
	appsService := local.NewApplicationsService()

	// Get types
	types, err := appsService.GetApplicationTypes(userOrgRole, userOrgID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to get application types")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve application types", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	c.JSON(http.StatusOK, response.OK("application types retrieved successfully", types))
}

// GetApplicationVersions handles GET /api/filters/applications/versions - returns available versions grouped by application type
func GetApplicationVersions(c *gin.Context) {
	// Get current user context
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Create applications service
	appsService := local.NewApplicationsService()

	// Get versions grouped by instance_of
	versionsByProduct, err := appsService.GetApplicationVersions(userOrgRole, userOrgID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to get application versions")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve application versions", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Convert map to array of ApplicationVersions
	type ApplicationVersions struct {
		Application string   `json:"application"`
		Name        string   `json:"name"`
		Versions    []string `json:"versions"`
	}

	groupedVersions := make([]ApplicationVersions, 0)
	for application, group := range versionsByProduct {
		groupedVersions = append(groupedVersions, ApplicationVersions{
			Application: application,
			Name:        group.Name,
			Versions:    group.Versions,
		})
	}

	// Sort by application name for consistent output
	sort.Slice(groupedVersions, func(i, j int) bool {
		return groupedVersions[i].Application < groupedVersions[j].Application
	})

	result := gin.H{
		"versions": groupedVersions,
	}

	logger.Info().
		Str("component", "filters").
		Str("operation", "application_versions_filters").
		Str("user_org_id", userOrgID).
		Str("user_org_role", userOrgRole).
		Int("product_count", len(groupedVersions)).
		Msg("application version filters retrieved")

	c.JSON(http.StatusOK, response.OK("application versions retrieved successfully", result))
}

// GetApplicationSystems handles GET /api/applications/systems - returns available systems for filter
func GetApplicationSystems(c *gin.Context) {
	// Get current user context
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Create applications service
	appsService := local.NewApplicationsService()

	// Get systems
	systems, err := appsService.GetAvailableSystems(userOrgRole, userOrgID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to get available systems")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve systems", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	c.JSON(http.StatusOK, response.OK("systems retrieved successfully", systems))
}

// GetApplicationOrganizations handles GET /api/applications/organizations - returns available orgs for assignment
func GetApplicationOrganizations(c *gin.Context) {
	// Get current user context
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Create applications service
	appsService := local.NewApplicationsService()

	// Get organizations
	orgs, err := appsService.GetAvailableOrganizations(userOrgRole, userOrgID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to get available organizations")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve organizations", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	c.JSON(http.StatusOK, response.OK("organizations retrieved successfully", orgs))
}

// GetApplicationTypeSummary handles GET /api/applications/summary - returns applications grouped by type
func GetApplicationTypeSummary(c *gin.Context) {
	// Get current user context
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Optional filters
	organizationID := c.Query("organization_id")
	systemID := c.Query("system_id")
	includeHierarchy := c.Query("include_hierarchy") == "true"

	// Pagination parameters (0 means no pagination)
	page := 0
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	pageSize := 0
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Sorting parameters
	sortBy := c.DefaultQuery("sort_by", "count")
	sortDirection := c.DefaultQuery("sort_direction", "desc")

	// Create applications service
	appsService := local.NewApplicationsService()

	// Get type summary
	summary, err := appsService.GetApplicationTypeSummary(userOrgRole, userOrgID, organizationID, systemID, includeHierarchy, page, pageSize, sortBy, sortDirection)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Str("organization_id", organizationID).
			Msg("Failed to get application type summary")

		if strings.Contains(err.Error(), "access denied") {
			msg := "access denied to organization"
			if systemID != "" {
				msg = "access denied to system"
			}
			c.JSON(http.StatusForbidden, response.Forbidden(msg, nil))
			return
		}

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve application type summary", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Build response with pagination if requested
	responseData := gin.H{
		"total":   summary.Total,
		"by_type": helpers.EnsureSlice(summary.ByType),
	}

	if pageSize > 0 {
		responseData["pagination"] = helpers.BuildPaginationInfoWithSorting(page, pageSize, summary.TotalTypes, sortBy, sortDirection)
	}

	c.JSON(http.StatusOK, response.OK("application type summary retrieved successfully", responseData))
}
