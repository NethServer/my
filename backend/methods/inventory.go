/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
)

// GetSystemInventoryHistory handles GET /api/systems/:id/inventory - retrieves paginated inventory history
func GetSystemInventoryHistory(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	// Get current user context for access validation
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Validate system access
	systemsService := local.NewSystemsService()
	_, err := systemsService.GetSystem(systemID, userOrgRole, userOrgID)
	if handleSystemAccessError(c, err, systemID) {
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
	inventoryService := local.NewInventoryService()
	records, totalCount, err := inventoryService.GetInventoryHistory(systemID, page, pageSize, fromDate, toDate)
	if err != nil {
		logger.Error().
			Err(err).
			Str("system_id", systemID).
			Int("page", page).
			Int("page_size", pageSize).
			Msg("Failed to retrieve inventory history")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve inventory history", map[string]interface{}{
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
		"records":    records,
		"pagination": helpers.BuildPaginationInfo(page, pageSize, totalCount),
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
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Validate system access
	systemsService := local.NewSystemsService()
	_, err := systemsService.GetSystem(systemID, userOrgRole, userOrgID)
	if handleSystemAccessError(c, err, systemID) {
		return
	}

	// Get latest inventory
	inventoryService := local.NewInventoryService()
	record, err := inventoryService.GetLatestInventory(systemID)
	if err != nil {
		if err.Error() == "no inventory found for system "+systemID {
			// Return 200 with null when no inventory exists (not an error condition)
			logger.RequestLogger(c, "inventory").Info().
				Str("operation", "get_latest_inventory").
				Str("system_id", systemID).
				Msg("No inventory available for system")

			c.JSON(http.StatusOK, response.OK("no inventory available", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("system_id", systemID).
			Msg("Failed to retrieve latest inventory")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve latest inventory", map[string]interface{}{
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
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Validate system access
	systemsService := local.NewSystemsService()
	_, err := systemsService.GetSystem(systemID, userOrgRole, userOrgID)
	if handleSystemAccessError(c, err, systemID) {
		return
	}

	// Get changes summary
	inventoryService := local.NewInventoryService()
	summary, err := inventoryService.GetChangesSummary(systemID)
	if err != nil {
		if err.Error() == "no inventory found for system "+systemID {
			// Return 200 with null when no inventory exists (not an error condition)
			logger.RequestLogger(c, "inventory").Info().
				Str("operation", "get_changes_summary").
				Str("system_id", systemID).
				Msg("No inventory available for system")

			c.JSON(http.StatusOK, response.OK("no inventory available", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("system_id", systemID).
			Msg("Failed to retrieve changes summary")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve changes summary", map[string]interface{}{
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
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Validate system access
	systemsService := local.NewSystemsService()
	_, err := systemsService.GetSystem(systemID, userOrgRole, userOrgID)
	if handleSystemAccessError(c, err, systemID) {
		return
	}

	// Get latest changes summary
	inventoryService := local.NewInventoryService()
	summary, err := inventoryService.GetLatestInventoryChangesSummary(systemID)
	if err != nil {
		if err.Error() == "no inventory found for system "+systemID {
			// Return 200 with null when no inventory exists (not an error condition)
			logger.RequestLogger(c, "inventory").Info().
				Str("operation", "get_latest_inventory_changes").
				Str("system_id", systemID).
				Msg("No inventory available for system")

			c.JSON(http.StatusOK, response.OK("no inventory available", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("system_id", systemID).
			Msg("Failed to retrieve latest inventory changes summary")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve latest inventory changes summary", map[string]interface{}{
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
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Validate system access
	systemsService := local.NewSystemsService()
	_, err := systemsService.GetSystem(systemID, userOrgRole, userOrgID)
	if handleSystemAccessError(c, err, systemID) {
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
	inventoryService := local.NewInventoryService()
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

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve inventory diffs", map[string]interface{}{
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
		"diffs":      diffs,
		"pagination": helpers.BuildPaginationInfo(page, pageSize, totalCount),
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
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Validate system access
	systemsService := local.NewSystemsService()
	_, err := systemsService.GetSystem(systemID, userOrgRole, userOrgID)
	if handleSystemAccessError(c, err, systemID) {
		return
	}

	// Get latest diffs batch
	inventoryService := local.NewInventoryService()
	diffs, err := inventoryService.GetLatestInventoryDiffs(systemID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("system_id", systemID).
			Msg("Failed to retrieve latest inventory diffs")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve latest inventory diffs", map[string]interface{}{
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
