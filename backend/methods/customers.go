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
		// Check if it's a validation error from service
		if validationErr := getValidationError(err); validationErr != nil {
			logger.Warn().
				Str("user_id", user.ID).
				Str("customer_name", request.Name).
				Str("validation_reason", validationErr.ErrorData.Errors[0].Message).
				Msg("Customer creation validation failed")

			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", validationErr.ErrorData.Errors))
			return
		}

		// System error - log as error
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("customer_name", request.Name).
			Msg("Failed to create customer")

		// Default to internal server error
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create customer", map[string]interface{}{
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

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get customer", nil))
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

	// Resolve rebranding info
	if customer.LogtoID != nil {
		customer.RebrandingEnabled, customer.RebrandingOrgID = resolveRebranding(*customer.LogtoID)
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

	// Parse pagination and sorting parameters
	page, pageSize, sortBy, sortDirection := helpers.GetPaginationAndSortingFromQuery(c)

	// Parse search and status parameters
	search := c.Query("search")
	statuses := c.QueryArray("status")

	// Create service
	service := local.NewOrganizationService()

	// Get customers based on RBAC
	userOrgRole := strings.ToLower(user.OrgRole)
	customers, totalCount, err := service.ListCustomers(userOrgRole, user.OrganizationID, page, pageSize, search, sortBy, sortDirection, statuses)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("user_org_role", userOrgRole).
			Msg("Failed to list customers")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to list customers", nil))
		return
	}

	// Resolve rebranding info for each customer
	for i := range customers {
		if customers[i].LogtoID != nil {
			customers[i].RebrandingEnabled, customers[i].RebrandingOrgID = resolveRebranding(*customers[i].LogtoID)
		}
	}

	// Log the action
	logger.RequestLogger(c, "customers").Info().
		Str("operation", "list_customers").
		Int("page", page).
		Int("page_size", pageSize).
		Str("search", search).
		Int("total_count", totalCount).
		Int("returned_count", len(customers)).
		Msg("Customers list requested")

	// Return paginated response
	c.JSON(http.StatusOK, response.OK("customers retrieved successfully", gin.H{
		"customers":  customers,
		"pagination": helpers.BuildPaginationInfoWithSorting(page, pageSize, totalCount, sortBy, sortDirection),
	}))
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

	// Get customer to obtain logto_id for hierarchy validation
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
			Msg("Failed to get customer for update validation")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get customer", nil))
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
		// Pass the logto_id, not the local database ID
		if customer.LogtoID != nil {
			canUpdate = userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, *customer.LogtoID)
		}
	case "customer":
		// Customer can only update themselves - compare with logto_id
		if customer.LogtoID != nil && *customer.LogtoID == user.OrganizationID {
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
	customer, err = service.UpdateCustomer(customerID, &request, user.ID, user.OrganizationID)
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
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to update customer", map[string]interface{}{
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

	// Get customer to obtain logto_id for hierarchy validation
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
			Msg("Failed to get customer for deletion validation")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get customer", nil))
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
		// Pass the logto_id, not the local database ID
		if customer.LogtoID != nil {
			canDelete = userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, *customer.LogtoID)
		}
	}

	if !canDelete {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to delete customer", nil))
		return
	}

	// Create service
	service := local.NewOrganizationService()

	// Delete customer
	deletedSystemsCount, deletedUsersCount, err := service.DeleteCustomer(customerID, user.ID, user.OrganizationID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("customer_id", customerID).
			Msg("Failed to delete customer")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to delete customer", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "customers", "delete", "customer", customerID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("customer deleted successfully", map[string]interface{}{
		"deleted_systems_count": deletedSystemsCount,
		"deleted_users_count":   deletedUsersCount,
	}))
}

// RestoreCustomer handles PATCH /api/customers/:id/restore - restores a soft-deleted customer
func RestoreCustomer(c *gin.Context) {
	customerID := c.Param("id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("customer ID required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get customer (including deleted) for hierarchy validation
	repo := entities.NewLocalCustomerRepository()
	customer, err := repo.GetByIDIncludeDeleted(customerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("customer not found", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get customer", nil))
		return
	}

	// RBAC: owner can always, distributor/reseller check hierarchy
	userOrgRole := strings.ToLower(user.OrgRole)
	canRestore := false
	switch userOrgRole {
	case "owner":
		canRestore = true
	case "distributor", "reseller":
		if customer.LogtoID != nil {
			userService := local.NewUserService()
			canRestore = userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, *customer.LogtoID)
		}
	}

	if !canRestore {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to restore customer", nil))
		return
	}

	service := local.NewOrganizationService()

	restoredSystemsCount, restoredUsersCount, err := service.RestoreCustomer(customerID, user.ID, user.OrganizationID)
	if err != nil {
		if strings.Contains(err.Error(), "not deleted") {
			c.JSON(http.StatusBadRequest, response.BadRequest("customer is not deleted", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("customer_id", customerID).
			Msg("Failed to restore customer")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to restore customer", nil))
		return
	}

	logger.LogBusinessOperation(c, "customers", "restore", "customer", customerID, true, nil)

	c.JSON(http.StatusOK, response.OK("customer restored successfully", map[string]interface{}{
		"restored_systems_count": restoredSystemsCount,
		"restored_users_count":   restoredUsersCount,
	}))
}

// DestroyCustomer handles DELETE /api/customers/:id/destroy - permanently deletes a customer
func DestroyCustomer(c *gin.Context) {
	customerID := c.Param("id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("customer ID required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get customer (including deleted) for hierarchy validation
	repo := entities.NewLocalCustomerRepository()
	customer, err := repo.GetByIDIncludeDeleted(customerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("customer not found", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get customer", nil))
		return
	}

	// RBAC: owner can always, distributor/reseller check hierarchy
	userOrgRole := strings.ToLower(user.OrgRole)
	canDestroy := false
	switch userOrgRole {
	case "owner":
		canDestroy = true
	case "distributor", "reseller":
		if customer.LogtoID != nil {
			userService := local.NewUserService()
			canDestroy = userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, *customer.LogtoID)
		}
	}

	if !canDestroy {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to destroy customer", nil))
		return
	}

	service := local.NewOrganizationService()

	err = service.DestroyCustomer(customerID, user.ID, user.OrganizationID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("customer_id", customerID).
			Msg("Failed to destroy customer")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to destroy customer", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	logger.LogBusinessOperation(c, "customers", "destroy", "customer", customerID, true, nil)

	c.JSON(http.StatusOK, response.OK("customer permanently destroyed", nil))
}

// GetCustomerStats handles GET /api/customers/:id/stats - retrieves users, systems and applications count for a customer
func GetCustomerStats(c *gin.Context) {
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

	// Get customer to obtain logto_id for hierarchy validation
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
			Msg("Failed to get customer for stats")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get customer", nil))
		return
	}

	// Apply hierarchical RBAC validation
	userService := local.NewUserService()
	userOrgRole := strings.ToLower(user.OrgRole)
	canAccess := false

	switch userOrgRole {
	case "owner":
		canAccess = true
	case "distributor", "reseller":
		if customer.LogtoID != nil {
			canAccess = userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, *customer.LogtoID)
		}
	case "customer":
		if customer.LogtoID != nil && *customer.LogtoID == user.OrganizationID {
			canAccess = true
		}
	}

	if !canAccess {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to customer stats", nil))
		return
	}

	// Get stats
	stats, err := repo.GetStats(customerID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("customer_id", customerID).
			Msg("Failed to get customer stats")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get customer stats", nil))
		return
	}

	// Log the action
	logger.RequestLogger(c, "customers").Info().
		Str("operation", "get_customer_stats").
		Str("customer_id", customerID).
		Int("users_count", stats.UsersCount).
		Int("systems_count", stats.SystemsCount).
		Int("applications_count", stats.ApplicationsCount).
		Msg("Customer stats requested")

	// Return stats
	c.JSON(http.StatusOK, response.OK("customer stats retrieved successfully", stats))
}

// SuspendCustomer handles PATCH /api/customers/:id/suspend - suspends a customer and all its users
func SuspendCustomer(c *gin.Context) {
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

	// Get customer to obtain logto_id for hierarchy validation
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
			Msg("Failed to get customer for suspension validation")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get customer", nil))
		return
	}

	// Apply hierarchical RBAC validation - Owner, Distributor, Reseller can suspend
	userService := local.NewUserService()
	userOrgRole := strings.ToLower(user.OrgRole)
	canSuspend := false

	switch userOrgRole {
	case "owner":
		canSuspend = true
	case "distributor", "reseller":
		if customer.LogtoID != nil {
			canSuspend = userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, *customer.LogtoID)
		}
	}

	if !canSuspend {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to suspend customer", nil))
		return
	}

	// Suspend customer
	service := local.NewOrganizationService()
	customer, suspendedUsersCount, suspendedSystemsCount, err := service.SuspendCustomer(customerID, user.ID, user.OrganizationID)
	if err != nil {
		if strings.Contains(err.Error(), "already suspended") {
			c.JSON(http.StatusBadRequest, response.BadRequest("customer is already suspended", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("customer_id", customerID).
			Msg("Failed to suspend customer")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to suspend customer", nil))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "customers", "suspend", "customer", customerID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("customer suspended successfully", map[string]interface{}{
		"customer":                customer,
		"suspended_users_count":   suspendedUsersCount,
		"suspended_systems_count": suspendedSystemsCount,
	}))
}

// ReactivateCustomer handles PATCH /api/customers/:id/reactivate - reactivates a customer and its cascade-suspended users
func ReactivateCustomer(c *gin.Context) {
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

	// Get customer to obtain logto_id for hierarchy validation
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
			Msg("Failed to get customer for reactivation validation")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get customer", nil))
		return
	}

	// Apply hierarchical RBAC validation - Owner, Distributor, Reseller can reactivate
	userService := local.NewUserService()
	userOrgRole := strings.ToLower(user.OrgRole)
	canReactivate := false

	switch userOrgRole {
	case "owner":
		canReactivate = true
	case "distributor", "reseller":
		if customer.LogtoID != nil {
			canReactivate = userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, *customer.LogtoID)
		}
	}

	if !canReactivate {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to reactivate customer", nil))
		return
	}

	// Guard: if customer was cascade-suspended by a higher org, only that org's authority can reactivate
	if customer.SuspendedByOrgID != nil && *customer.SuspendedByOrgID != "" {
		c.JSON(http.StatusForbidden, response.Forbidden("customer is suspended by a parent organization and cannot be reactivated directly", nil))
		return
	}

	// Reactivate customer
	service := local.NewOrganizationService()
	customer, reactivatedUsersCount, reactivatedSystemsCount, err := service.ReactivateCustomer(customerID, user.ID, user.OrganizationID)
	if err != nil {
		if strings.Contains(err.Error(), "not suspended") {
			c.JSON(http.StatusBadRequest, response.BadRequest("customer is not suspended", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("customer_id", customerID).
			Msg("Failed to reactivate customer")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to reactivate customer", nil))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "customers", "reactivate", "customer", customerID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("customer reactivated successfully", map[string]interface{}{
		"customer":                  customer,
		"reactivated_users_count":   reactivatedUsersCount,
		"reactivated_systems_count": reactivatedSystemsCount,
	}))
}
