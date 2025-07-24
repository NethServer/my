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
	c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to validate system access", map[string]interface{}{
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
	creatorInfo := &models.SystemCreator{
		UserID:           user.ID,
		UserName:         user.Username,
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
			Str("customer_id", request.CustomerID).
			Msg("Failed to create system")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to create system", map[string]interface{}{
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
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Parse pagination parameters
	page := 1
	pageSize := 50 // Default page size for systems
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Create systems service
	systemsService := local.NewSystemsService()

	// Get systems with pagination
	systems, totalCount, err := systemsService.GetSystemsByOrganizationPaginated(userID, userOrgID, userOrgRole, page, pageSize)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Int("page", page).
			Int("page_size", pageSize).
			Msg("Failed to retrieve systems")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to retrieve systems", map[string]interface{}{
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
		Msg("Systems list requested")

	// Return paginated systems list
	c.JSON(http.StatusOK, response.OK("systems retrieved successfully", gin.H{
		"systems": systems,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total_count": totalCount,
			"total_pages": (totalCount + pageSize - 1) / pageSize,
		},
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

	// Get current user context with organization ID
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
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
	system, err := systemsService.UpdateSystem(systemID, &request, userID, userOrgID, userOrgRole)
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

	// Get current user context with organization ID
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Create systems service
	systemsService := local.NewSystemsService()

	// Delete system with access validation
	err := systemsService.DeleteSystem(systemID, userID, userOrgID, userOrgRole)
	if handleSystemAccessError(c, err, systemID) {
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "systems", "delete", "system", systemID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("system deleted successfully", nil))
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
