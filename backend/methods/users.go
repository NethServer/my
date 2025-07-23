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
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
)

// CreateUser handles POST /api/users - creates a new user locally and syncs to Logto
func CreateUser(c *gin.Context) {
	// Parse request body
	var request models.CreateLocalUserRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	// Validate password using our secure validator
	isValid, errorCodes := helpers.ValidatePasswordStrength(request.Password)
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
	service := local.NewUserService()

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
			Str("user_username", request.Username).
			Msg("Failed to create user")

		// Check if it's a validation error from Logto
		if validationErr := getValidationError(err); validationErr != nil {
			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", validationErr.ErrorData.Errors))
			return
		}

		// Default to internal server error
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to create user", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "users", "create", "user", account.ID, true, nil)

	// Return success response
	c.JSON(http.StatusCreated, response.Created("user created successfully", account))
}

// GetUser handles GET /api/users/:id - retrieves a single user account
func GetUser(c *gin.Context) {
	// Get user ID from URL parameter
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("user ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get user account
	repo := entities.NewLocalUserRepository()
	account, err := repo.GetByID(userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("user not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("current_user_id", user.ID).
			Str("target_user_id", userID).
			Msg("Failed to get user")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to get user", nil))
		return
	}

	// Apply RBAC validation
	userOrgRole := strings.ToLower(user.OrgRole)
	canAccess := false

	// Users can always see themselves
	if userID == user.ID {
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
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to user", nil))
		return
	}

	// Log the action
	logger.RequestLogger(c, "users").Info().
		Str("operation", "get_user").
		Str("user_id", userID).
		Msg("User details requested")

	// Return user account
	c.JSON(http.StatusOK, response.OK("user retrieved successfully", account))
}

// GetUsers handles GET /api/users - list accounts with pagination
func GetUsers(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Parse pagination parameters
	page, pageSize := helpers.GetPaginationFromQuery(c)

	// Create service
	service := local.NewUserService()

	// Get users based on RBAC
	userOrgRole := strings.ToLower(user.OrgRole)
	accounts, totalCount, err := service.ListUsers(userOrgRole, user.OrganizationID, page, pageSize)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("user_org_role", userOrgRole).
			Msg("Failed to list users")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to list users", nil))
		return
	}

	// Log the action
	logger.RequestLogger(c, "users").Info().
		Str("operation", "list_users").
		Int("page", page).
		Int("page_size", pageSize).
		Int("total_count", totalCount).
		Int("returned_count", len(accounts)).
		Msg("Users list requested")

	// Return paginated response
	c.JSON(http.StatusOK, response.Paginated("users retrieved successfully", "users", accounts, totalCount, page, pageSize))
}

// UpdateUser handles PUT /api/users/:id - updates a user account locally and syncs to Logto
func UpdateUser(c *gin.Context) {
	// Get user ID from URL parameter
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("user ID required", nil))
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

	// Get current user account for RBAC validation
	repo := entities.NewLocalUserRepository()
	currentAccount, err := repo.GetByID(userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("user not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("current_user_id", user.ID).
			Str("target_user_id", userID).
			Msg("Failed to get user for update validation")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to get user", nil))
		return
	}

	// Apply RBAC validation
	userOrgRole := strings.ToLower(user.OrgRole)
	canUpdate := false

	// Users can always update themselves (with restrictions)
	if userID == user.ID {
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

		service := local.NewUserService()
		if canUpdateUser, reason := service.CanUpdateUser(userOrgRole, user.OrganizationID, targetOrgID); canUpdateUser {
			canUpdate = true
		} else {
			c.JSON(http.StatusForbidden, response.Forbidden("access denied: "+reason, nil))
			return
		}
	}

	if !canUpdate {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to update user", nil))
		return
	}

	// Create service
	service := local.NewUserService()

	// Update user
	account, err := service.UpdateUser(userID, &request, user.ID, user.OrganizationID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("current_user_id", user.ID).
			Str("target_user_id", userID).
			Msg("Failed to update user")

		// Check if it's a validation error from Logto
		if validationErr := getValidationError(err); validationErr != nil {
			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", validationErr.ErrorData.Errors))
			return
		}

		// Default to internal server error
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to update user", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "users", "update", "user", userID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("user updated successfully", account))
}

// DeleteUser handles DELETE /api/users/:id - soft-deletes a user account locally and syncs to Logto
func DeleteUser(c *gin.Context) {
	// Get user ID from URL parameter
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("user ID required", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Users cannot delete themselves
	if userID == user.ID {
		c.JSON(http.StatusForbidden, response.Forbidden("users cannot delete their own account", nil))
		return
	}

	// Get current user account for RBAC validation
	repo := entities.NewLocalUserRepository()
	currentAccount, err := repo.GetByID(userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("user not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("current_user_id", user.ID).
			Str("target_user_id", userID).
			Msg("Failed to get user for delete validation")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to get user", nil))
		return
	}

	// Apply RBAC validation
	targetOrgID := ""
	if currentAccount.OrganizationID != nil {
		targetOrgID = *currentAccount.OrganizationID
	}

	userOrgRole := strings.ToLower(user.OrgRole)
	service := local.NewUserService()

	if canDelete, reason := service.CanDeleteUser(userOrgRole, user.OrganizationID, targetOrgID); !canDelete {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied: "+reason, nil))
		return
	}

	// Delete user
	err = service.DeleteUser(userID, user.ID, user.OrganizationID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("current_user_id", user.ID).
			Str("target_user_id", userID).
			Msg("Failed to delete user")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to delete user", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "users", "delete", "user", userID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("user deleted successfully", nil))
}

// ResetUserPassword handles PATCH /api/users/:id/password - resets user password
func ResetUserPassword(c *gin.Context) {
	// Get user ID from URL parameter
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("user ID required", nil))
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
	isValid, errorCodes := helpers.ValidatePasswordStrength(request.Password)
	if !isValid {
		c.JSON(http.StatusBadRequest, response.PasswordValidationBadRequest(errorCodes))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get target user account for RBAC validation
	repo := entities.NewLocalUserRepository()
	targetAccount, err := repo.GetByID(userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("user not found", nil))
			return
		}

		logger.Error().
			Err(err).
			Str("current_user_id", user.ID).
			Str("target_user_id", userID).
			Msg("Failed to get user for password reset")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to get user", nil))
		return
	}

	// Apply RBAC validation - similar to update but more restrictive
	userOrgRole := strings.ToLower(user.OrgRole)
	canReset := false

	// Users can reset their own password
	if userID == user.ID {
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
		c.JSON(http.StatusBadRequest, response.BadRequest("user not synced to Logto", nil))
		return
	}

	// Create service and reset password using Logto ID
	service := local.NewUserService()
	err = service.ResetUserPassword(*targetAccount.LogtoID, request.Password)
	if err != nil {
		logger.Error().
			Err(err).
			Str("current_user_id", user.ID).
			Str("target_user_id", userID).
			Str("logto_user_id", *targetAccount.LogtoID).
			Msg("Failed to reset user password")

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
	logger.LogBusinessOperation(c, "users", "password_reset", "user", userID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("password reset successfully", nil))
}

// getValidationError checks if the error chain contains a ValidationError and returns it
func getValidationError(err error) *local.ValidationError {
	var validationErr *local.ValidationError
	if errors.As(err, &validationErr) {
		return validationErr
	}
	return nil
}
