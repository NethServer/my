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

// CreateCustomer handles POST /api/customers - creates a new customer locally and syncs to Logto
func CreateCustomer(c *gin.Context) {
	// Parse request body
	var request models.CreateLocalCustomerRequest
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
	if canCreate, reason := service.CanCreateCustomer(userOrgRole, user.OrganizationID); !canCreate {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied: "+reason, nil))
		return
	}

	// Create customer
	customer, err := service.CreateCustomer(&request, user.ID, user.OrganizationID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("customer_name", request.Name).
			Msg("Failed to create customer")

		// Check if it's a validation error from service
		if validationErr := getValidationError(err); validationErr != nil {
			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", validationErr.ErrorData.Errors))
			return
		}

		// Default to internal server error
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to create customer", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "customers", "create", "customer", customer.ID, true, nil)

	// Return success response
	c.JSON(http.StatusCreated, response.Created("customer created successfully", customer))
}

// GetCustomer handles GET /api/customers/:id - retrieves a single customer
func GetCustomer(c *gin.Context) {
	// Get customer ID from URL parameter
	customerID := c.Param("id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("customer ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get customer
	repo := entities.NewLocalCustomerRepository()
	customer, err := repo.GetByID(customerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("customer not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("customer_id", customerID).
			Msg("Failed to get customer")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to get customer", nil))
		return
	}

	// Apply RBAC validation
	userOrgRole := strings.ToLower(user.OrgRole)
	canAccess := false
	switch userOrgRole {
	case "owner":
		canAccess = true
	case "distributor":
		// Check if customer was created by this distributor (via CustomData)
		if customer.CustomData != nil {
			if createdBy, ok := customer.CustomData["createdBy"].(string); ok && createdBy == user.OrganizationID {
				canAccess = true
			}
		}
	case "reseller":
		// Check if customer was created by this reseller (via CustomData)
		if customer.CustomData != nil {
			if createdBy, ok := customer.CustomData["createdBy"].(string); ok && createdBy == user.OrganizationID {
				canAccess = true
			}
		}
	case "customer":
		// Customer can only access themselves
		if customerID == user.OrganizationID {
			canAccess = true
		}
	}

	if !canAccess {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to customer", nil))
		return
	}

	// Log the action
	logger.RequestLogger(c, "customers").Info().
		Str("operation", "get_customer").
		Str("customer_id", customerID).
		Msg("Customer details requested")

	// Return customer
	c.JSON(http.StatusOK, response.OK("customer retrieved successfully", customer))
}

// GetCustomers handles GET /api/customers - list customers with pagination
func GetCustomers(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Parse pagination parameters
	page, pageSize := helpers.GetPaginationFromQuery(c)

	// Create service
	service := local.NewOrganizationService()

	// Get customers based on RBAC
	userOrgRole := strings.ToLower(user.OrgRole)
	customers, totalCount, err := service.ListCustomers(userOrgRole, user.OrganizationID, page, pageSize)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("user_org_role", userOrgRole).
			Msg("Failed to list customers")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to list customers", nil))
		return
	}

	// Log the action
	logger.RequestLogger(c, "customers").Info().
		Str("operation", "list_customers").
		Int("page", page).
		Int("page_size", pageSize).
		Int("total_count", totalCount).
		Int("returned_count", len(customers)).
		Msg("Customers list requested")

	// Return paginated response
	c.JSON(http.StatusOK, response.Paginated("customers retrieved successfully", "customers", customers, totalCount, page, pageSize))
}

// UpdateCustomer handles PUT /api/customers/:id - updates a customer locally and syncs to Logto
func UpdateCustomer(c *gin.Context) {
	// Get customer ID from URL parameter
	customerID := c.Param("id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("customer ID required", nil))
		return
	}

	// Parse request body
	var request models.UpdateLocalCustomerRequest
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
	userService := local.NewUserService()
	userOrgRole := strings.ToLower(user.OrgRole)
	canUpdate := false

	switch userOrgRole {
	case "owner":
		canUpdate = true
	case "distributor", "reseller":
		// Use hierarchical validation - check if customer organization is in hierarchy
		canUpdate = userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, customerID)
	case "customer":
		// Customer can only update themselves
		if customerID == user.OrganizationID {
			canUpdate = true
		}
	}

	if !canUpdate {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to update customer", nil))
		return
	}

	// Create service
	service := local.NewOrganizationService()

	// Update customer
	customer, err := service.UpdateCustomer(customerID, &request, user.ID, user.OrganizationID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("customer_id", customerID).
			Msg("Failed to update customer")

		// Check if it's a validation error from service
		if validationErr := getValidationError(err); validationErr != nil {
			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", validationErr.ErrorData.Errors))
			return
		}

		// Default to internal server error
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to update customer", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "customers", "update", "customer", customerID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("customer updated successfully", customer))
}

// DeleteCustomer handles DELETE /api/customers/:id - soft-deletes a customer locally and syncs to Logto
func DeleteCustomer(c *gin.Context) {
	// Get customer ID from URL parameter
	customerID := c.Param("id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("customer ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Apply hierarchical RBAC validation - only creators and above can delete
	userService := local.NewUserService()
	userOrgRole := strings.ToLower(user.OrgRole)
	canDelete := false

	switch userOrgRole {
	case "owner":
		canDelete = true
	case "distributor", "reseller":
		// Use hierarchical validation - check if customer organization is in hierarchy
		canDelete = userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, customerID)
	}

	if !canDelete {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to delete customer", nil))
		return
	}

	// Create service
	service := local.NewOrganizationService()

	// Delete customer
	err := service.DeleteCustomer(customerID, user.ID, user.OrganizationID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("customer_id", customerID).
			Msg("Failed to delete customer")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to delete customer", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "customers", "delete", "customer", customerID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("customer deleted successfully", nil))
}
