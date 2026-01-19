/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
)

// handleApplicationAccessError handles application access errors with appropriate HTTP status codes
func handleApplicationAccessError(c *gin.Context, err error, appID string) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	if errMsg == "application not found" {
		c.JSON(http.StatusNotFound, response.NotFound("application not found", nil))
		return true
	}

	if strings.Contains(errMsg, "access denied") {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to application", map[string]interface{}{
			"application_id": appID,
		}))
		return true
	}

	// Technical error
	c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to process application request", map[string]interface{}{
		"error": errMsg,
	}))
	return true
}

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

	// Convert to list items for response
	items := make([]*models.ApplicationListItem, len(apps))
	for i, app := range apps {
		items[i] = app.ToListItem()
	}

	// Calculate total pages
	totalPages := (totalCount + pageSize - 1) / pageSize

	c.JSON(http.StatusOK, response.OK("applications retrieved successfully", map[string]interface{}{
		"items":       items,
		"total":       totalCount,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
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
	if handleApplicationAccessError(c, err, appID) {
		return
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
	if handleApplicationAccessError(c, err, appID) {
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
	if handleApplicationAccessError(c, err, appID) {
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
	if handleApplicationAccessError(c, err, appID) {
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
	if handleApplicationAccessError(c, err, appID) {
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

// GetApplicationVersions handles GET /api/applications/versions - returns available versions
func GetApplicationVersions(c *gin.Context) {
	// Get current user context
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Create applications service
	appsService := local.NewApplicationsService()

	// Get versions
	versions, err := appsService.GetApplicationVersions(userOrgRole, userOrgID)
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

	c.JSON(http.StatusOK, response.OK("application versions retrieved successfully", versions))
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
