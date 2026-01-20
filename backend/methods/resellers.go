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

	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
)

// CreateReseller handles POST /api/resellers - creates a new reseller locally and syncs to Logto
func CreateReseller(c *gin.Context) {
	// Parse request body
	var request models.CreateLocalResellerRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Create service
	service := local.NewOrganizationService()

	// Validate permissions
	userOrgRole := strings.ToLower(user.OrgRole)
	if canCreate, reason := service.CanCreateReseller(userOrgRole, user.OrganizationID); !canCreate {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied: "+reason, nil))
		return
	}

	// Create reseller
	reseller, err := service.CreateReseller(&request, user.ID, user.OrganizationID)
	if err != nil {
		// Check if it's a validation error from service
		if validationErr := getValidationError(err); validationErr != nil {
			logger.Warn().
				Str("user_id", user.ID).
				Str("reseller_name", request.Name).
				Str("validation_reason", validationErr.ErrorData.Errors[0].Message).
				Msg("Reseller creation validation failed")

			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", validationErr.ErrorData.Errors))
			return
		}

		// System error - log as error
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("reseller_name", request.Name).
			Msg("Failed to create reseller")

		// Default to internal server error
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create reseller", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "resellers", "create", "reseller", reseller.ID, true, nil)

	// Return success response
	c.JSON(http.StatusCreated, response.Created("reseller created successfully", reseller))
}

// GetReseller handles GET /api/resellers/:id - retrieves a single reseller
func GetReseller(c *gin.Context) {
	// Get reseller ID from URL parameter
	resellerID := c.Param("id")
	if resellerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("reseller ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get reseller
	repo := entities.NewLocalResellerRepository()
	reseller, err := repo.GetByID(resellerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("reseller not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("reseller_id", resellerID).
			Msg("Failed to get reseller")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get reseller", nil))
		return
	}

	// Apply RBAC validation
	userOrgRole := strings.ToLower(user.OrgRole)
	canAccess := false
	switch userOrgRole {
	case "owner":
		canAccess = true
	case "distributor":
		// Check if this reseller was created by the distributor (via CustomData)
		if reseller.CustomData != nil {
			if createdBy, ok := reseller.CustomData["createdBy"].(string); ok && createdBy == user.OrganizationID {
				canAccess = true
			}
		}
	case "reseller":
		// Reseller can only access themselves
		if resellerID == user.OrganizationID {
			canAccess = true
		}
	}

	if !canAccess {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to reseller", nil))
		return
	}

	// Log the action
	logger.RequestLogger(c, "resellers").Info().
		Str("operation", "get_reseller").
		Str("reseller_id", resellerID).
		Msg("Reseller details requested")

	// Return reseller
	c.JSON(http.StatusOK, response.OK("reseller retrieved successfully", reseller))
}

// GetResellers handles GET /api/resellers - list resellers with pagination
func GetResellers(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Parse pagination and sorting parameters
	page, pageSize, sortBy, sortDirection := helpers.GetPaginationAndSortingFromQuery(c)

	// Parse search and status parameters
	search := c.Query("search")
	status := c.Query("status")

	// Create service
	service := local.NewOrganizationService()

	// Get resellers based on RBAC
	userOrgRole := strings.ToLower(user.OrgRole)
	resellers, totalCount, err := service.ListResellers(userOrgRole, user.OrganizationID, page, pageSize, search, sortBy, sortDirection, status)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("user_org_role", userOrgRole).
			Msg("Failed to list resellers")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to list resellers", nil))
		return
	}

	// Log the action
	logger.RequestLogger(c, "resellers").Info().
		Str("operation", "list_resellers").
		Int("page", page).
		Int("page_size", pageSize).
		Str("search", search).
		Int("total_count", totalCount).
		Int("returned_count", len(resellers)).
		Msg("Resellers list requested")

	// Return paginated response
	c.JSON(http.StatusOK, response.OK("resellers retrieved successfully", gin.H{
		"resellers":  resellers,
		"pagination": helpers.BuildPaginationInfoWithSorting(page, pageSize, totalCount, sortBy, sortDirection),
	}))
}

// UpdateReseller handles PUT /api/resellers/:id - updates a reseller locally and syncs to Logto
func UpdateReseller(c *gin.Context) {
	// Get reseller ID from URL parameter
	resellerID := c.Param("id")
	if resellerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("reseller ID required", nil))
		return
	}

	// Parse request body
	var request models.UpdateLocalResellerRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get reseller to obtain logto_id for hierarchy validation
	repo := entities.NewLocalResellerRepository()
	reseller, err := repo.GetByID(resellerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("reseller not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("reseller_id", resellerID).
			Msg("Failed to get reseller for update validation")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get reseller", nil))
		return
	}

	// Apply hierarchical RBAC validation using service layer
	userService := local.NewUserService()
	userOrgRole := strings.ToLower(user.OrgRole)
	canUpdate := false

	switch userOrgRole {
	case "owner":
		canUpdate = true
	case "distributor":
		// Use hierarchical validation - check if reseller organization is in hierarchy
		// Pass the logto_id, not the local database ID
		if reseller.LogtoID != nil {
			canUpdate = userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, *reseller.LogtoID)
		}
	case "reseller":
		// Reseller can only update themselves - compare with logto_id
		if reseller.LogtoID != nil && *reseller.LogtoID == user.OrganizationID {
			canUpdate = true
		}
	}

	if !canUpdate {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to update reseller", nil))
		return
	}

	// Create service
	service := local.NewOrganizationService()

	// Update reseller
	reseller, err = service.UpdateReseller(resellerID, &request, user.ID, user.OrganizationID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("reseller_id", resellerID).
			Msg("Failed to update reseller")

		// Check if it's a validation error from service
		if validationErr := getValidationError(err); validationErr != nil {
			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", validationErr.ErrorData.Errors))
			return
		}

		// Default to internal server error
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to update reseller", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "resellers", "update", "reseller", resellerID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("reseller updated successfully", reseller))
}

// DeleteReseller handles DELETE /api/resellers/:id - soft-deletes a reseller locally and syncs to Logto
func DeleteReseller(c *gin.Context) {
	// Get reseller ID from URL parameter
	resellerID := c.Param("id")
	if resellerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("reseller ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get reseller to obtain logto_id for hierarchy validation
	repo := entities.NewLocalResellerRepository()
	reseller, err := repo.GetByID(resellerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("reseller not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("reseller_id", resellerID).
			Msg("Failed to get reseller for deletion validation")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get reseller", nil))
		return
	}

	// Apply hierarchical RBAC validation - only creators and above can delete
	userService := local.NewUserService()
	userOrgRole := strings.ToLower(user.OrgRole)
	canDelete := false

	switch userOrgRole {
	case "owner":
		canDelete = true
	case "distributor":
		// Use hierarchical validation - check if reseller organization is in hierarchy
		// Pass the logto_id, not the local database ID
		if reseller.LogtoID != nil {
			canDelete = userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, *reseller.LogtoID)
		}
	}

	if !canDelete {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to delete reseller", nil))
		return
	}

	// Create service
	service := local.NewOrganizationService()

	// Delete reseller
	err = service.DeleteReseller(resellerID, user.ID, user.OrganizationID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("reseller_id", resellerID).
			Msg("Failed to delete reseller")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to delete reseller", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "resellers", "delete", "reseller", resellerID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("reseller deleted successfully", nil))
}

// GetResellerStats handles GET /api/resellers/:id/stats - retrieves users and systems count for a reseller
func GetResellerStats(c *gin.Context) {
	// Get reseller ID from URL parameter
	resellerID := c.Param("id")
	if resellerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("reseller ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get reseller to obtain logto_id for hierarchy validation
	repo := entities.NewLocalResellerRepository()
	reseller, err := repo.GetByID(resellerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("reseller not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("reseller_id", resellerID).
			Msg("Failed to get reseller for stats")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get reseller", nil))
		return
	}

	// Apply hierarchical RBAC validation
	userService := local.NewUserService()
	userOrgRole := strings.ToLower(user.OrgRole)
	canAccess := false

	switch userOrgRole {
	case "owner":
		canAccess = true
	case "distributor":
		if reseller.LogtoID != nil {
			canAccess = userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, *reseller.LogtoID)
		}
	case "reseller":
		if reseller.LogtoID != nil && *reseller.LogtoID == user.OrganizationID {
			canAccess = true
		}
	}

	if !canAccess {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to reseller stats", nil))
		return
	}

	// Get stats
	stats, err := repo.GetStats(resellerID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("reseller_id", resellerID).
			Msg("Failed to get reseller stats")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get reseller stats", nil))
		return
	}

	// Log the action
	logger.RequestLogger(c, "resellers").Info().
		Str("operation", "get_reseller_stats").
		Str("reseller_id", resellerID).
		Int("users_count", stats.UsersCount).
		Int("systems_count", stats.SystemsCount).
		Int("customers_count", stats.CustomersCount).
		Int("applications_count", stats.ApplicationsCount).
		Int("applications_hierarchy_count", stats.ApplicationsHierarchyCount).
		Msg("Reseller stats requested")

	// Return stats
	c.JSON(http.StatusOK, response.OK("reseller stats retrieved successfully", stats))
}

// SuspendReseller handles PATCH /api/resellers/:id/suspend - suspends a reseller and all its users
func SuspendReseller(c *gin.Context) {
	// Get reseller ID from URL parameter
	resellerID := c.Param("id")
	if resellerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("reseller ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get reseller to obtain logto_id for hierarchy validation
	repo := entities.NewLocalResellerRepository()
	reseller, err := repo.GetByID(resellerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("reseller not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("reseller_id", resellerID).
			Msg("Failed to get reseller for suspension validation")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get reseller", nil))
		return
	}

	// Apply hierarchical RBAC validation - only Owner and Distributor can suspend
	userService := local.NewUserService()
	userOrgRole := strings.ToLower(user.OrgRole)
	canSuspend := false

	switch userOrgRole {
	case "owner":
		canSuspend = true
	case "distributor":
		if reseller.LogtoID != nil {
			canSuspend = userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, *reseller.LogtoID)
		}
	}

	if !canSuspend {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to suspend reseller", nil))
		return
	}

	// Suspend reseller
	service := local.NewOrganizationService()
	reseller, suspendedUsersCount, err := service.SuspendReseller(resellerID, user.ID, user.OrganizationID)
	if err != nil {
		if strings.Contains(err.Error(), "already suspended") {
			c.JSON(http.StatusBadRequest, response.BadRequest("reseller is already suspended", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("reseller_id", resellerID).
			Msg("Failed to suspend reseller")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to suspend reseller", nil))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "resellers", "suspend", "reseller", resellerID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("reseller suspended successfully", map[string]interface{}{
		"reseller":              reseller,
		"suspended_users_count": suspendedUsersCount,
	}))
}

// ReactivateReseller handles PATCH /api/resellers/:id/reactivate - reactivates a reseller and its cascade-suspended users
func ReactivateReseller(c *gin.Context) {
	// Get reseller ID from URL parameter
	resellerID := c.Param("id")
	if resellerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("reseller ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get reseller to obtain logto_id for hierarchy validation
	repo := entities.NewLocalResellerRepository()
	reseller, err := repo.GetByID(resellerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("reseller not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("reseller_id", resellerID).
			Msg("Failed to get reseller for reactivation validation")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get reseller", nil))
		return
	}

	// Apply hierarchical RBAC validation - only Owner and Distributor can reactivate
	userService := local.NewUserService()
	userOrgRole := strings.ToLower(user.OrgRole)
	canReactivate := false

	switch userOrgRole {
	case "owner":
		canReactivate = true
	case "distributor":
		if reseller.LogtoID != nil {
			canReactivate = userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, *reseller.LogtoID)
		}
	}

	if !canReactivate {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to reactivate reseller", nil))
		return
	}

	// Reactivate reseller
	service := local.NewOrganizationService()
	reseller, reactivatedUsersCount, err := service.ReactivateReseller(resellerID, user.ID, user.OrganizationID)
	if err != nil {
		if strings.Contains(err.Error(), "not suspended") {
			c.JSON(http.StatusBadRequest, response.BadRequest("reseller is not suspended", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("reseller_id", resellerID).
			Msg("Failed to reactivate reseller")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to reactivate reseller", nil))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "resellers", "reactivate", "reseller", resellerID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("reseller reactivated successfully", map[string]interface{}{
		"reseller":                reseller,
		"reactivated_users_count": reactivatedUsersCount,
	}))
}
