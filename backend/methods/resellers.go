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
	"github.com/nethesis/my/backend/repositories"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services"
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
	service := services.NewLocalOrganizationService()

	// Validate permissions
	userOrgRole := strings.ToLower(user.OrgRole)
	if canCreate, reason := service.CanCreateReseller(userOrgRole, user.OrganizationID); !canCreate {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied: "+reason, nil))
		return
	}

	// Create reseller
	reseller, err := service.CreateReseller(&request, user.ID, user.OrganizationID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("reseller_name", request.Name).
			Msg("Failed to create reseller")

		// Check if it's a validation error from service
		if validationErr := getValidationError(err); validationErr != nil {
			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", validationErr.ErrorData.Errors))
			return
		}

		// Default to internal server error
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to create reseller", map[string]interface{}{
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
	repo := repositories.NewLocalResellerRepository()
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

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to get reseller", nil))
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

	// Parse pagination parameters
	page, pageSize := helpers.GetPaginationFromQuery(c)

	// Create service
	service := services.NewLocalOrganizationService()

	// Get resellers based on RBAC
	userOrgRole := strings.ToLower(user.OrgRole)
	resellers, totalCount, err := service.ListResellers(userOrgRole, user.OrganizationID, page, pageSize)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("user_org_role", userOrgRole).
			Msg("Failed to list resellers")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to list resellers", nil))
		return
	}

	// Log the action
	logger.RequestLogger(c, "resellers").Info().
		Str("operation", "list_resellers").
		Int("page", page).
		Int("page_size", pageSize).
		Int("total_count", totalCount).
		Int("returned_count", len(resellers)).
		Msg("Resellers list requested")

	// Return paginated response
	c.JSON(http.StatusOK, response.Paginated("resellers retrieved successfully", "resellers", resellers, totalCount, page, pageSize))
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

	// Apply hierarchical RBAC validation using service layer
	userService := services.NewLocalUserService()
	userOrgRole := strings.ToLower(user.OrgRole)
	canUpdate := false

	switch userOrgRole {
	case "owner":
		canUpdate = true
	case "distributor":
		// Use hierarchical validation - check if reseller organization is in hierarchy
		canUpdate = userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, resellerID)
	case "reseller":
		// Reseller can only update themselves
		if resellerID == user.OrganizationID {
			canUpdate = true
		}
	}

	if !canUpdate {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to update reseller", nil))
		return
	}

	// Create service
	service := services.NewLocalOrganizationService()

	// Update reseller
	reseller, err := service.UpdateReseller(resellerID, &request, user.ID, user.OrganizationID)
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
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to update reseller", map[string]interface{}{
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

	// Apply hierarchical RBAC validation - only creators and above can delete
	userService := services.NewLocalUserService()
	userOrgRole := strings.ToLower(user.OrgRole)
	canDelete := false

	switch userOrgRole {
	case "owner":
		canDelete = true
	case "distributor":
		// Use hierarchical validation - check if reseller organization is in hierarchy
		canDelete = userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, resellerID)
	}

	if !canDelete {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to delete reseller", nil))
		return
	}

	// Create service
	service := services.NewLocalOrganizationService()

	// Delete reseller
	err := service.DeleteReseller(resellerID, user.ID, user.OrganizationID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("reseller_id", resellerID).
			Msg("Failed to delete reseller")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to delete reseller", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "resellers", "delete", "reseller", resellerID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("reseller deleted successfully", nil))
}
