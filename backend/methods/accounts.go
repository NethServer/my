/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"errors"
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
	"github.com/nethesis/my/backend/validation"
)

// CreateAccount handles POST /api/accounts - creates a new user locally and syncs to Logto
func CreateAccount(c *gin.Context) {
	// Parse request body
	var request models.CreateLocalUserRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	// Validate password using our secure validator
	isValid, errorCodes := validation.ValidatePasswordStrength(request.Password)
	if !isValid {
		c.JSON(http.StatusBadRequest, response.PasswordValidationBadRequest(errorCodes))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Create service
	service := services.NewLocalUserService()

	// Validate permissions
	userOrgRole := strings.ToLower(user.OrgRole)
	if canCreate, reason := service.CanCreateUser(userOrgRole, user.OrganizationID, &request); !canCreate {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied: "+reason, nil))
		return
	}

	// Create user
	account, err := service.CreateUser(&request, user.ID, user.OrganizationID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("account_username", request.Username).
			Msg("Failed to create account")

		// Check if it's a validation error from Logto
		if validationErr := getValidationError(err); validationErr != nil {
			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", validationErr.ErrorData.Errors))
			return
		}

		// Default to internal server error
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to create account", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "accounts", "create", "account", account.ID, true, nil)

	// Return success response
	c.JSON(http.StatusCreated, response.Created("account created successfully", account))
}

// GetAccount handles GET /api/accounts/:id - retrieves a single user account
func GetAccount(c *gin.Context) {
	// Get account ID from URL parameter
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("account ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get account
	repo := repositories.NewLocalUserRepository()
	account, err := repo.GetByID(accountID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("account not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("account_id", accountID).
			Msg("Failed to get account")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to get account", nil))
		return
	}

	// Apply RBAC validation
	userOrgRole := strings.ToLower(user.OrgRole)
	canAccess := false

	// Users can always see themselves
	if accountID == user.ID {
		canAccess = true
	} else {
		// Check organization-based access
		targetOrgID := ""
		if account.OrganizationID != nil {
			targetOrgID = *account.OrganizationID
		}

		switch userOrgRole {
		case "owner":
			canAccess = true
		case "distributor":
			// Distributor can see users in their organization and customer organizations they manage
			if targetOrgID == user.OrganizationID {
				canAccess = true
			}
			// Additional logic needed to check customer organizations
		case "reseller":
			// Reseller can see users in their organization
			if targetOrgID == user.OrganizationID {
				canAccess = true
			}
		case "customer":
			// Customer can only see users in their own organization
			if targetOrgID == user.OrganizationID {
				canAccess = true
			}
		}
	}

	if !canAccess {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to account", nil))
		return
	}

	// Log the action
	logger.RequestLogger(c, "accounts").Info().
		Str("operation", "get_account").
		Str("account_id", accountID).
		Msg("Account details requested")

	// Return account
	c.JSON(http.StatusOK, response.OK("account retrieved successfully", account))
}

// GetAccounts handles GET /api/accounts - list accounts with pagination
func GetAccounts(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Parse pagination parameters
	page, pageSize := helpers.GetPaginationFromQuery(c)

	// Create service
	service := services.NewLocalUserService()

	// Get accounts based on RBAC
	userOrgRole := strings.ToLower(user.OrgRole)
	accounts, totalCount, err := service.ListUsers(userOrgRole, user.OrganizationID, page, pageSize)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("user_org_role", userOrgRole).
			Msg("Failed to list accounts")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to list accounts", nil))
		return
	}

	// Log the action
	logger.RequestLogger(c, "accounts").Info().
		Str("operation", "list_accounts").
		Int("page", page).
		Int("page_size", pageSize).
		Int("total_count", totalCount).
		Int("returned_count", len(accounts)).
		Msg("Accounts list requested")

	// Return paginated response
	c.JSON(http.StatusOK, response.Paginated("accounts retrieved successfully", "accounts", accounts, totalCount, page, pageSize))
}

// UpdateAccount handles PUT /api/accounts/:id - updates a user account locally and syncs to Logto
func UpdateAccount(c *gin.Context) {
	// Get account ID from URL parameter
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("account ID required", nil))
		return
	}

	// Parse request body
	var request models.UpdateLocalUserRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get current account for RBAC validation
	repo := repositories.NewLocalUserRepository()
	currentAccount, err := repo.GetByID(accountID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("account not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("account_id", accountID).
			Msg("Failed to get account for update validation")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to get account", nil))
		return
	}

	// Apply RBAC validation
	userOrgRole := strings.ToLower(user.OrgRole)
	canUpdate := false

	// Users can always update themselves (with restrictions)
	if accountID == user.ID {
		canUpdate = true
		// Users can only update certain fields about themselves
		// Organization-related fields should be restricted
		if request.OrganizationID != nil || request.UserRoleIDs != nil {
			c.JSON(http.StatusForbidden, response.Forbidden("users cannot modify their own organization or role information", nil))
			return
		}
	} else {
		// Check organization-based permissions for updating other users
		targetOrgID := ""
		if currentAccount.OrganizationID != nil {
			targetOrgID = *currentAccount.OrganizationID
		}

		service := services.NewLocalUserService()
		if canUpdateUser, reason := service.CanUpdateUser(userOrgRole, user.OrganizationID, targetOrgID); canUpdateUser {
			canUpdate = true
		} else {
			c.JSON(http.StatusForbidden, response.Forbidden("access denied: "+reason, nil))
			return
		}
	}

	if !canUpdate {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to update account", nil))
		return
	}

	// Create service
	service := services.NewLocalUserService()

	// Update account
	account, err := service.UpdateUser(accountID, &request, user.ID, user.OrganizationID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("account_id", accountID).
			Msg("Failed to update account")

		// Check if it's a validation error from Logto
		if validationErr := getValidationError(err); validationErr != nil {
			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", validationErr.ErrorData.Errors))
			return
		}

		// Default to internal server error
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to update account", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "accounts", "update", "account", accountID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("account updated successfully", account))
}

// DeleteAccount handles DELETE /api/accounts/:id - soft-deletes a user account locally and syncs to Logto
func DeleteAccount(c *gin.Context) {
	// Get account ID from URL parameter
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("account ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Users cannot delete themselves
	if accountID == user.ID {
		c.JSON(http.StatusForbidden, response.Forbidden("users cannot delete their own account", nil))
		return
	}

	// Get current account for RBAC validation
	repo := repositories.NewLocalUserRepository()
	currentAccount, err := repo.GetByID(accountID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("account not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("account_id", accountID).
			Msg("Failed to get account for delete validation")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to get account", nil))
		return
	}

	// Apply RBAC validation
	targetOrgID := ""
	if currentAccount.OrganizationID != nil {
		targetOrgID = *currentAccount.OrganizationID
	}

	userOrgRole := strings.ToLower(user.OrgRole)
	service := services.NewLocalUserService()

	if canDelete, reason := service.CanDeleteUser(userOrgRole, user.OrganizationID, targetOrgID); !canDelete {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied: "+reason, nil))
		return
	}

	// Delete account
	err = service.DeleteUser(accountID, user.ID, user.OrganizationID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("account_id", accountID).
			Msg("Failed to delete account")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to delete account", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "accounts", "delete", "account", accountID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("account deleted successfully", nil))
}

// ResetAccountPassword handles PATCH /api/accounts/:id/password - resets account password
func ResetAccountPassword(c *gin.Context) {
	// Get account ID from URL parameter
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("account ID required", nil))
		return
	}

	// Parse request body for password (as per OpenAPI spec)
	var request struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid request body", nil))
		return
	}

	// Validate password using our secure validator
	isValid, errorCodes := validation.ValidatePasswordStrength(request.Password)
	if !isValid {
		c.JSON(http.StatusBadRequest, response.PasswordValidationBadRequest(errorCodes))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get target account for RBAC validation
	repo := repositories.NewLocalUserRepository()
	targetAccount, err := repo.GetByID(accountID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("account not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("account_id", accountID).
			Msg("Failed to get account for password reset")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to get account", nil))
		return
	}

	// Apply RBAC validation - similar to update but more restrictive
	userOrgRole := strings.ToLower(user.OrgRole)
	canReset := false

	// Users can reset their own password
	if accountID == user.ID {
		canReset = true
	} else {
		// Only higher-level roles can reset other users' passwords
		targetOrgID := ""
		if targetAccount.OrganizationID != nil {
			targetOrgID = *targetAccount.OrganizationID
		}

		switch userOrgRole {
		case "owner":
			canReset = true
		case "distributor":
			// Distributor can reset passwords in organizations they manage
			if targetOrgID == user.OrganizationID {
				canReset = true
			}
		case "reseller":
			// Reseller can reset passwords in their organization
			if targetOrgID == user.OrganizationID {
				canReset = true
			}
		}
	}

	if !canReset {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to reset password", nil))
		return
	}

	// Check if user is synced to Logto
	if targetAccount.LogtoID == nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("account not synced to Logto", nil))
		return
	}

	// Create service and reset password using Logto ID
	service := services.NewLocalUserService()
	err = service.ResetUserPassword(*targetAccount.LogtoID, request.Password)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("account_id", accountID).
			Str("logto_user_id", *targetAccount.LogtoID).
			Msg("Failed to reset account password")

		// Check if it's a validation error from Logto (same as other endpoints)
		if validationErr := getValidationError(err); validationErr != nil {
			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", validationErr.ErrorData.Errors))
			return
		}

		// Default to internal server error for non-validation errors
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to reset password", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "accounts", "password_reset", "account", accountID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("password reset successfully", nil))
}

// getValidationError checks if the error chain contains a ValidationError and returns it
func getValidationError(err error) *services.ValidationError {
	var validationErr *services.ValidationError
	if errors.As(err, &validationErr) {
		return validationErr
	}
	return nil
}
