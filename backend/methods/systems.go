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

// handleSystemAccessError handles system access errors with appropriate HTTP status codes
func handleSystemAccessError(c *gin.Context, err error, systemID string) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	if errMsg == "system not found" {
		c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
		return true
	}

	if strings.Contains(errMsg, "access denied") {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to system", map[string]interface{}{
			"system_id": systemID,
		}))
		return true
	}

	// Technical error
	c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to validate system access", map[string]interface{}{
		"error": errMsg,
	}))
	return true
}

// CreateSystem handles POST /api/systems - creates a new system
func CreateSystem(c *gin.Context) {
	// Parse request body
	var request models.CreateSystemRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Create systems service
	systemsService := local.NewSystemsService()

	// Create SystemCreator object with detailed user information
	// Use logto_id for consistency across the system (users and organizations both use logto_id)
	userLogtoID := ""
	if user.LogtoID != nil {
		userLogtoID = *user.LogtoID
	}
	creatorInfo := &models.SystemCreator{
		UserID:           userLogtoID,
		Username:         user.Username,
		Name:             user.Name,
		Email:            user.Email,
		OrganizationID:   user.OrganizationID,
		OrganizationName: user.OrganizationName,
	}

	// Create system with automatic secret generation
	system, err := systemsService.CreateSystem(&request, creatorInfo, user.OrgRole, user.OrganizationID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("system_name", request.Name).
			Msg("Failed to create system")

		// Check if it's an access denied error
		if strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusForbidden, response.Forbidden(err.Error(), nil))
			return
		}

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create system", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "systems", "create", "system", system.ID, true, nil)

	// Return success response with system data directly in data
	c.JSON(http.StatusCreated, response.Created("system created successfully", system))
}

// GetSystems handles GET /api/systems - retrieves all systems with pagination
func GetSystems(c *gin.Context) {
	// Get current user context with organization ID
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Parse pagination and sorting parameters
	page, pageSize, sortBy, sortDirection := helpers.GetPaginationAndSortingFromQuery(c)

	// Override default page size for systems
	if c.Query("page_size") == "" {
		pageSize = 50 // Default page size for systems
	}

	// Parse search parameter
	search := c.Query("search")

	// Parse filter parameters (supporting multiple values via checkbox, except name which is text input)
	filterName := c.Query("name")                   // Name filter (single value, text input)
	filterSystemKey := c.Query("system_key")        // System Key filter (single value, exact match)
	filterTypes := c.QueryArray("type")             // Product/Type filter (multiple values)
	filterCreatedBy := c.QueryArray("created_by")   // Created By filter (multiple user IDs)
	filterVersions := c.QueryArray("version")       // Version filter (multiple values)
	filterOrgIDs := c.QueryArray("organization_id") // Organization filter (multiple IDs)
	filterStatuses := c.QueryArray("status")        // Status filter (multiple values)

	// Create systems service
	systemsService := local.NewSystemsService()

	// Get systems with pagination, search, sorting and filters
	systems, totalCount, err := systemsService.GetSystemsByOrganizationPaginated(
		userID, userOrgID, userOrgRole, page, pageSize, search, sortBy, sortDirection,
		filterName, filterSystemKey, filterTypes, filterCreatedBy, filterVersions, filterOrgIDs, filterStatuses,
	)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Int("page", page).
			Int("page_size", pageSize).
			Str("search", search).
			Str("filter_name", filterName).
			Strs("filter_types", filterTypes).
			Strs("filter_created_by", filterCreatedBy).
			Strs("filter_versions", filterVersions).
			Strs("filter_organization_ids", filterOrgIDs).
			Strs("filter_statuses", filterStatuses).
			Str("sort_by", sortBy).
			Str("sort_direction", sortDirection).
			Msg("Failed to retrieve systems")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve systems", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.RequestLogger(c, "systems").Info().
		Str("operation", "list_systems").
		Int("count", len(systems)).
		Int("total", totalCount).
		Int("page", page).
		Int("page_size", pageSize).
		Str("search", search).
		Str("filter_name", filterName).
		Strs("filter_types", filterTypes).
		Strs("filter_created_by", filterCreatedBy).
		Strs("filter_versions", filterVersions).
		Strs("filter_organization_ids", filterOrgIDs).
		Strs("filter_statuses", filterStatuses).
		Str("sort_by", sortBy).
		Str("sort_direction", sortDirection).
		Msg("Systems list requested")

	// Return paginated systems list
	c.JSON(http.StatusOK, response.OK("systems retrieved successfully", gin.H{
		"systems":    systems,
		"pagination": helpers.BuildPaginationInfoWithSorting(page, pageSize, totalCount, sortBy, sortDirection),
	}))
}

// GetSystem handles GET /api/systems/:id - retrieves a single system
func GetSystem(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Create systems service
	systemsService := local.NewSystemsService()

	// Get system with access validation
	system, err := systemsService.GetSystem(systemID, user.OrgRole, user.OrganizationID)
	if handleSystemAccessError(c, err, systemID) {
		return
	}

	// Log the action
	logger.RequestLogger(c, "systems").Info().
		Str("operation", "get_system").
		Str("system_id", systemID).
		Msg("System details requested")

	// Return system
	c.JSON(http.StatusOK, response.OK("system retrieved successfully", system))
}

// UpdateSystem handles PUT /api/systems/:id - updates an existing system
func UpdateSystem(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Parse request body
	var request models.UpdateSystemRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	// Create systems service
	systemsService := local.NewSystemsService()

	// Update system with access validation
	system, err := systemsService.UpdateSystem(systemID, &request, user.ID, user.OrganizationID, user.OrgRole)
	if handleSystemAccessError(c, err, systemID) {
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "systems", "update", "system", systemID, true, nil)

	// Return updated system
	c.JSON(http.StatusOK, response.OK("system updated successfully", system))
}

// DeleteSystem handles DELETE /api/systems/:id - deletes a system
func DeleteSystem(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Create systems service
	systemsService := local.NewSystemsService()

	// Delete system with access validation
	err := systemsService.DeleteSystem(systemID, user.ID, user.OrganizationID, user.OrgRole)
	if handleSystemAccessError(c, err, systemID) {
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "systems", "delete", "system", systemID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("system deleted successfully", nil))
}

// RestoreSystem handles PATCH /api/systems/:id/restore - restores a soft-deleted system
func RestoreSystem(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Create systems service
	systemsService := local.NewSystemsService()

	// Restore system with access validation
	err := systemsService.RestoreSystem(systemID, user.ID, user.OrganizationID, user.OrgRole)
	if err != nil {
		errMsg := err.Error()

		// Check for specific error types
		if strings.Contains(errMsg, "system not found") {
			c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
			return
		}

		if strings.Contains(errMsg, "system is not deleted") {
			c.JSON(http.StatusBadRequest, response.BadRequest("system is not deleted", nil))
			return
		}

		if strings.Contains(errMsg, "access denied") {
			c.JSON(http.StatusForbidden, response.Forbidden("access denied to system", map[string]interface{}{
				"system_id": systemID,
			}))
			return
		}

		// Technical error
		logger.Error().
			Err(err).
			Str("system_id", systemID).
			Str("user_id", user.ID).
			Msg("Failed to restore system")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to restore system", map[string]interface{}{
			"error": errMsg,
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "systems", "restore", "system", systemID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("system restored successfully", nil))
}

// RegenerateSystemSecret handles POST /api/systems/:id/regenerate-secret - regenerates system secret
func RegenerateSystemSecret(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Create systems service
	systemsService := local.NewSystemsService()

	// Regenerate system secret
	system, err := systemsService.RegenerateSystemSecret(systemID, user.ID, user.OrganizationID, user.OrgRole)
	if handleSystemAccessError(c, err, systemID) {
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "systems", "regenerate_secret", "system", systemID, true, nil)

	// Return new secret (only time it's visible)
	c.JSON(http.StatusOK, response.OK("system secret regenerated successfully", system))
}

// RegisterSystem handles POST /api/systems/register - registers a system using system_secret
func RegisterSystem(c *gin.Context) {
	// Parse request body
	var request models.RegisterSystemRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	// Create systems service
	systemsService := local.NewSystemsService()

	// Register system using the secret token
	result, err := systemsService.RegisterSystem(request.SystemSecret)
	if err != nil {
		errMsg := err.Error()

		logger.Warn().
			Err(err).
			Msg("Failed system registration attempt")

		// Map errors to appropriate HTTP status codes
		switch {
		case strings.Contains(errMsg, "invalid system secret format"):
			c.JSON(http.StatusBadRequest, response.BadRequest("invalid system secret format", nil))
			return
		case strings.Contains(errMsg, "invalid system secret"):
			c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid system secret", nil))
			return
		case strings.Contains(errMsg, "system has been deleted"):
			c.JSON(http.StatusForbidden, response.Forbidden("system has been deleted", nil))
			return
		case strings.Contains(errMsg, "system is already registered"):
			c.JSON(http.StatusConflict, response.Conflict("system is already registered", nil))
			return
		default:
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to register system", map[string]interface{}{
				"error": errMsg,
			}))
			return
		}
	}

	// Log successful registration
	logger.Info().
		Str("system_key", result.SystemKey).
		Time("registered_at", result.RegisteredAt).
		Msg("System registered successfully")

	// Return system_key and registration info
	c.JSON(http.StatusOK, response.OK("system registered successfully", result))
}

// SuspendSystem handles PATCH /api/systems/:id/suspend - suspends a system
func SuspendSystem(c *gin.Context) {
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	systemsService := local.NewSystemsService()
	err := systemsService.SuspendSystem(systemID, user.OrgRole, user.OrganizationID)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "system not found" {
			c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
			return
		}
		if strings.Contains(errMsg, "access denied") {
			c.JSON(http.StatusForbidden, response.Forbidden("access denied to system", nil))
			return
		}
		if strings.Contains(errMsg, "already suspended") {
			c.JSON(http.StatusBadRequest, response.BadRequest("system is already suspended", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("system_id", systemID).
			Str("user_id", user.ID).
			Msg("Failed to suspend system")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to suspend system", nil))
		return
	}

	logger.LogBusinessOperation(c, "systems", "suspend", "system", systemID, true, nil)

	c.JSON(http.StatusOK, response.OK("system suspended successfully", nil))
}

// ReactivateSystem handles PATCH /api/systems/:id/reactivate - reactivates a suspended system
func ReactivateSystem(c *gin.Context) {
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	systemsService := local.NewSystemsService()
	err := systemsService.ReactivateSystem(systemID, user.OrgRole, user.OrganizationID)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "system not found" {
			c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
			return
		}
		if strings.Contains(errMsg, "access denied") {
			c.JSON(http.StatusForbidden, response.Forbidden("access denied to system", nil))
			return
		}
		if strings.Contains(errMsg, "not suspended") {
			c.JSON(http.StatusBadRequest, response.BadRequest("system is not suspended", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("system_id", systemID).
			Str("user_id", user.ID).
			Msg("Failed to reactivate system")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to reactivate system", nil))
		return
	}

	logger.LogBusinessOperation(c, "systems", "reactivate", "system", systemID, true, nil)

	c.JSON(http.StatusOK, response.OK("system reactivated successfully", nil))
}
