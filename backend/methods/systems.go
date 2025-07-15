/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services"
)

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
	systemsService := services.NewSystemsService()

	// Create SystemCreator object with detailed user information
	creatorInfo := &models.SystemCreator{
		UserID:           user.ID,
		UserName:         user.Username,
		OrganizationID:   user.OrganizationID,
		OrganizationName: user.OrganizationName,
	}

	// Create system with automatic secret generation
	system, err := systemsService.CreateSystem(&request, creatorInfo)
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

// GetSystems handles GET /api/systems - retrieves all systems
func GetSystems(c *gin.Context) {
	// Get current user context
	userID, userOrgRole, userRole := helpers.GetUserContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Create systems service
	systemsService := services.NewSystemsService()

	// Get systems with proper filtering
	systems, err := systemsService.GetSystemsByOrganization(userID, userOrgRole, userRole)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
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
		Msg("Systems list requested")

	// Return systems list
	c.JSON(http.StatusOK, response.OK("systems retrieved successfully", gin.H{"systems": systems, "count": len(systems)}))
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
	userID, userOrgRole, userRole := helpers.GetUserContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Create systems service
	systemsService := services.NewSystemsService()

	// Get system with access validation
	system, err := systemsService.GetSystemByID(systemID, userID, userOrgRole, userRole)
	if err != nil {
		if err.Error() == "system not found" {
			c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", userID).
			Str("system_id", systemID).
			Msg("Failed to retrieve system")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to retrieve system", map[string]interface{}{
			"error": err.Error(),
		}))
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
	userID, userOrgRole, userRole := helpers.GetUserContext(c)
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
	systemsService := services.NewSystemsService()

	// Update system with access validation
	system, err := systemsService.UpdateSystem(systemID, &request, userID, userOrgRole, userRole)
	if err != nil {
		if err.Error() == "system not found" {
			c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", userID).
			Str("system_id", systemID).
			Msg("Failed to update system")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to update system", map[string]interface{}{
			"error": err.Error(),
		}))
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
	userID, userOrgRole, userRole := helpers.GetUserContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Create systems service
	systemsService := services.NewSystemsService()

	// Delete system with access validation
	err := systemsService.DeleteSystem(systemID, userID, userOrgRole, userRole)
	if err != nil {
		if err.Error() == "system not found" {
			c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", userID).
			Str("system_id", systemID).
			Msg("Failed to delete system")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to delete system", map[string]interface{}{
			"error": err.Error(),
		}))
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
	systemsService := services.NewSystemsService()

	// Get user role (fallback to empty string if no roles)
	userRole := ""
	if len(user.UserRoles) > 0 {
		userRole = user.UserRoles[0]
	}

	// Regenerate system secret
	system, err := systemsService.RegenerateSystemSecret(systemID, user.ID, user.OrgRole, userRole)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("system_id", systemID).
			Msg("Failed to regenerate system secret")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to regenerate system secret", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "systems", "regenerate_secret", "system", systemID, true, nil)

	// Return new secret (only time it's visible)
	c.JSON(http.StatusOK, response.OK("system secret regenerated successfully", system))
}

// GetSystemInventoryHistory handles GET /api/systems/:id/inventory - retrieves paginated inventory history
func GetSystemInventoryHistory(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	// Get current user context for access validation
	userID, userOrgRole, userRole := helpers.GetUserContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Validate system access
	systemsService := services.NewSystemsService()
	_, err := systemsService.GetSystemByID(systemID, userID, userOrgRole, userRole)
	if err != nil {
		if err.Error() == "system not found" {
			c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to validate system access", nil))
		return
	}

	// Parse pagination parameters
	page := 1
	pageSize := 20
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

	// Parse date filters
	var fromDate, toDate *time.Time
	if fromStr := c.Query("from_date"); fromStr != "" {
		if fd, err := time.Parse(time.RFC3339, fromStr); err == nil {
			fromDate = &fd
		}
	}
	if toStr := c.Query("to_date"); toStr != "" {
		if td, err := time.Parse(time.RFC3339, toStr); err == nil {
			toDate = &td
		}
	}

	// Get inventory history
	inventoryService := services.NewInventoryService()
	records, totalCount, err := inventoryService.GetInventoryHistory(systemID, page, pageSize, fromDate, toDate)
	if err != nil {
		logger.Error().
			Err(err).
			Str("system_id", systemID).
			Int("page", page).
			Int("page_size", pageSize).
			Msg("Failed to retrieve inventory history")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to retrieve inventory history", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.RequestLogger(c, "inventory").Info().
		Str("operation", "get_inventory_history").
		Str("system_id", systemID).
		Int("count", len(records)).
		Int("total", totalCount).
		Msg("Inventory history requested")

	// Return paginated results
	c.JSON(http.StatusOK, response.OK("inventory history retrieved successfully", gin.H{
		"records": records,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total_count": totalCount,
			"total_pages": (totalCount + pageSize - 1) / pageSize,
		},
	}))
}

// GetSystemLatestInventory handles GET /api/systems/:id/inventory/latest - retrieves latest inventory
func GetSystemLatestInventory(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	// Get current user context for access validation
	userID, userOrgRole, userRole := helpers.GetUserContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Validate system access
	systemsService := services.NewSystemsService()
	_, err := systemsService.GetSystemByID(systemID, userID, userOrgRole, userRole)
	if err != nil {
		if err.Error() == "system not found" {
			c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to validate system access", nil))
		return
	}

	// Get latest inventory
	inventoryService := services.NewInventoryService()
	record, err := inventoryService.GetLatestInventory(systemID)
	if err != nil {
		if err.Error() == "no inventory found for system "+systemID {
			c.JSON(http.StatusNotFound, response.NotFound("no inventory found for system", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("system_id", systemID).
			Msg("Failed to retrieve latest inventory")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to retrieve latest inventory", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.RequestLogger(c, "inventory").Info().
		Str("operation", "get_latest_inventory").
		Str("system_id", systemID).
		Int64("inventory_id", record.ID).
		Msg("Latest inventory requested")

	// Return latest inventory
	c.JSON(http.StatusOK, response.OK("latest inventory retrieved successfully", record))
}

// GetSystemInventoryChanges handles GET /api/systems/:id/inventory/changes - retrieves changes summary
func GetSystemInventoryChanges(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	// Get current user context for access validation
	userID, userOrgRole, userRole := helpers.GetUserContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Validate system access
	systemsService := services.NewSystemsService()
	_, err := systemsService.GetSystemByID(systemID, userID, userOrgRole, userRole)
	if err != nil {
		if err.Error() == "system not found" {
			c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to validate system access", nil))
		return
	}

	// Get changes summary
	inventoryService := services.NewInventoryService()
	summary, err := inventoryService.GetChangesSummary(systemID)
	if err != nil {
		if err.Error() == "no inventory found for system "+systemID {
			c.JSON(http.StatusNotFound, response.NotFound("no inventory found for system", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("system_id", systemID).
			Msg("Failed to retrieve changes summary")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to retrieve changes summary", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.RequestLogger(c, "inventory").Info().
		Str("operation", "get_changes_summary").
		Str("system_id", systemID).
		Int("total_changes", summary.TotalChanges).
		Msg("Changes summary requested")

	// Return changes summary
	c.JSON(http.StatusOK, response.OK("changes summary retrieved successfully", summary))
}

// GetSystemLatestInventoryChanges handles GET /api/systems/:id/inventory/changes/latest - retrieves latest batch changes summary
func GetSystemLatestInventoryChanges(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	// Get current user context for access validation
	userID, userOrgRole, userRole := helpers.GetUserContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Validate system access
	systemsService := services.NewSystemsService()
	_, err := systemsService.GetSystemByID(systemID, userID, userOrgRole, userRole)
	if err != nil {
		if err.Error() == "system not found" {
			c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to validate system access", nil))
		return
	}

	// Get latest changes summary
	inventoryService := services.NewInventoryService()
	summary, err := inventoryService.GetLatestInventoryChangesSummary(systemID)
	if err != nil {
		if err.Error() == "no inventory found for system "+systemID {
			c.JSON(http.StatusNotFound, response.NotFound("no inventory found for system", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("system_id", systemID).
			Msg("Failed to retrieve latest inventory changes summary")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to retrieve latest inventory changes summary", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.RequestLogger(c, "inventory").Info().
		Str("operation", "get_latest_inventory_changes").
		Str("system_id", systemID).
		Int("total_changes", summary.TotalChanges).
		Msg("Latest inventory changes summary requested")

	// Return latest changes summary
	c.JSON(http.StatusOK, response.OK("latest inventory changes summary retrieved successfully", summary))
}

// GetSystemInventoryDiffs handles GET /api/systems/:id/inventory/diffs - retrieves paginated diffs
func GetSystemInventoryDiffs(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	// Get current user context for access validation
	userID, userOrgRole, userRole := helpers.GetUserContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Validate system access
	systemsService := services.NewSystemsService()
	_, err := systemsService.GetSystemByID(systemID, userID, userOrgRole, userRole)
	if err != nil {
		if err.Error() == "system not found" {
			c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to validate system access", nil))
		return
	}

	// Parse pagination parameters
	page := 1
	pageSize := 20
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

	// Parse filters
	severity := c.Query("severity")
	category := c.Query("category")
	diffType := c.Query("diff_type")

	// Parse date filters
	var fromDate, toDate *time.Time
	if fromStr := c.Query("from_date"); fromStr != "" {
		if fd, err := time.Parse(time.RFC3339, fromStr); err == nil {
			fromDate = &fd
		}
	}
	if toStr := c.Query("to_date"); toStr != "" {
		if td, err := time.Parse(time.RFC3339, toStr); err == nil {
			toDate = &td
		}
	}

	// Get inventory diffs
	inventoryService := services.NewInventoryService()
	diffs, totalCount, err := inventoryService.GetInventoryDiffs(systemID, page, pageSize, severity, category, diffType, fromDate, toDate)
	if err != nil {
		logger.Error().
			Err(err).
			Str("system_id", systemID).
			Int("page", page).
			Int("page_size", pageSize).
			Str("severity", severity).
			Str("category", category).
			Str("diff_type", diffType).
			Msg("Failed to retrieve inventory diffs")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to retrieve inventory diffs", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.RequestLogger(c, "inventory").Info().
		Str("operation", "get_inventory_diffs").
		Str("system_id", systemID).
		Int("count", len(diffs)).
		Int("total", totalCount).
		Str("severity", severity).
		Str("category", category).
		Msg("Inventory diffs requested")

	// Return paginated results
	c.JSON(http.StatusOK, response.OK("inventory diffs retrieved successfully", gin.H{
		"diffs": diffs,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total_count": totalCount,
			"total_pages": (totalCount + pageSize - 1) / pageSize,
		},
	}))
}

// GetSystemLatestInventoryDiff handles GET /api/systems/:id/inventory/diffs/latest - retrieves latest diffs batch
func GetSystemLatestInventoryDiff(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	// Get current user context for access validation
	userID, userOrgRole, userRole := helpers.GetUserContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Validate system access
	systemsService := services.NewSystemsService()
	_, err := systemsService.GetSystemByID(systemID, userID, userOrgRole, userRole)
	if err != nil {
		if err.Error() == "system not found" {
			c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to validate system access", nil))
		return
	}

	// Get latest diffs batch
	inventoryService := services.NewInventoryService()
	diffs, err := inventoryService.GetLatestInventoryDiffs(systemID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("system_id", systemID).
			Msg("Failed to retrieve latest inventory diffs")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to retrieve latest inventory diffs", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logEvent := logger.RequestLogger(c, "inventory").Info().
		Str("operation", "get_latest_inventory_diffs").
		Str("system_id", systemID).
		Int("count", len(diffs))

	if len(diffs) > 0 {
		logEvent.Int64("current_id", diffs[0].CurrentID)
	}

	logEvent.Msg("Latest inventory diffs batch requested")

	// Prepare response data
	responseData := gin.H{
		"diffs": diffs,
		"count": len(diffs),
	}

	// Add current_inventory_id only if there are diffs
	if len(diffs) > 0 {
		responseData["current_inventory_id"] = diffs[0].CurrentID
	}

	// Return latest diffs batch
	c.JSON(http.StatusOK, response.OK("latest inventory diffs retrieved successfully", responseData))
}
