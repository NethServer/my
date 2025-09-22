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

// CreateDistributor handles POST /api/distributors - creates a new distributor locally and syncs to Logto
func CreateDistributor(c *gin.Context) {
	// Parse request body
	var request models.CreateLocalDistributorRequest
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
	if canCreate, reason := service.CanCreateDistributor(userOrgRole, user.OrganizationID); !canCreate {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied: "+reason, nil))
		return
	}

	// Create distributor
	distributor, err := service.CreateDistributor(&request, user.ID, user.OrganizationID)
	if err != nil {
		// Check if it's a validation error from service
		if validationErr := getValidationError(err); validationErr != nil {
			logger.Warn().
				Str("user_id", user.ID).
				Str("distributor_name", request.Name).
				Str("validation_reason", validationErr.ErrorData.Errors[0].Message).
				Msg("Distributor creation validation failed")

			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", validationErr.ErrorData.Errors))
			return
		}

		// System error - log as error
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("distributor_name", request.Name).
			Msg("Failed to create distributor")

		// Default to internal server error
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create distributor", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "distributors", "create", "distributor", distributor.ID, true, nil)

	// Return success response
	c.JSON(http.StatusCreated, response.Created("distributor created successfully", distributor))
}

// GetDistributor handles GET /api/distributors/:id - retrieves a single distributor
func GetDistributor(c *gin.Context) {
	// Get distributor ID from URL parameter
	distributorID := c.Param("id")
	if distributorID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("distributor ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Only Owner can access distributors
	if strings.ToLower(user.OrgRole) != "owner" {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied: only owners can access distributors", nil))
		return
	}

	// Get distributor
	repo := entities.NewLocalDistributorRepository()
	distributor, err := repo.GetByID(distributorID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("distributor not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("distributor_id", distributorID).
			Msg("Failed to get distributor")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get distributor", nil))
		return
	}

	// Log the action
	logger.RequestLogger(c, "distributors").Info().
		Str("operation", "get_distributor").
		Str("distributor_id", distributorID).
		Msg("Distributor details requested")

	// Return distributor
	c.JSON(http.StatusOK, response.OK("distributor retrieved successfully", distributor))
}

// GetDistributors handles GET /api/distributors - list distributors with pagination
func GetDistributors(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Parse pagination and sorting parameters
	page, pageSize, sortBy, sortDirection := helpers.GetPaginationAndSortingFromQuery(c)

	// Parse search parameter
	search := c.Query("search")

	// Create service
	service := local.NewOrganizationService()

	// Get distributors based on RBAC
	userOrgRole := strings.ToLower(user.OrgRole)
	distributors, totalCount, err := service.ListDistributors(userOrgRole, user.OrganizationID, page, pageSize, search, sortBy, sortDirection)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("user_org_role", userOrgRole).
			Msg("Failed to list distributors")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to list distributors", nil))
		return
	}

	// Log the action
	logger.RequestLogger(c, "distributors").Info().
		Str("operation", "list_distributors").
		Int("page", page).
		Int("page_size", pageSize).
		Str("search", search).
		Int("total_count", totalCount).
		Int("returned_count", len(distributors)).
		Msg("Distributors list requested")

	// Return paginated response
	c.JSON(http.StatusOK, response.PaginatedWithSorting("distributors retrieved successfully", "distributors", distributors, totalCount, page, pageSize, sortBy, sortDirection))
}

// UpdateDistributor handles PUT /api/distributors/:id - updates a distributor locally and syncs to Logto
func UpdateDistributor(c *gin.Context) {
	// Get distributor ID from URL parameter
	distributorID := c.Param("id")
	if distributorID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("distributor ID required", nil))
		return
	}

	// Parse request body
	var request models.UpdateLocalDistributorRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Only Owner can update distributors
	if strings.ToLower(user.OrgRole) != "owner" {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied: only owners can update distributors", nil))
		return
	}

	// Create service
	service := local.NewOrganizationService()

	// Update distributor
	distributor, err := service.UpdateDistributor(distributorID, &request, user.ID, user.OrganizationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("distributor not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("distributor_id", distributorID).
			Msg("Failed to update distributor")

		// Check if it's a validation error from service
		if validationErr := getValidationError(err); validationErr != nil {
			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", validationErr.ErrorData.Errors))
			return
		}

		// Default to internal server error
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to update distributor", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "distributors", "update", "distributor", distributorID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("distributor updated successfully", distributor))
}

// DeleteDistributor handles DELETE /api/distributors/:id - soft-deletes a distributor locally and syncs to Logto
func DeleteDistributor(c *gin.Context) {
	// Get distributor ID from URL parameter
	distributorID := c.Param("id")
	if distributorID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("distributor ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Only Owner can delete distributors
	if strings.ToLower(user.OrgRole) != "owner" {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied: only owners can delete distributors", nil))
		return
	}

	// Create service
	service := local.NewOrganizationService()

	// Delete distributor
	err := service.DeleteDistributor(distributorID, user.ID, user.OrganizationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("distributor not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("distributor_id", distributorID).
			Msg("Failed to delete distributor")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to delete distributor", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "distributors", "delete", "distributor", distributorID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("distributor deleted successfully", nil))
}
