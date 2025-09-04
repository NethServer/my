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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/jwt"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
	"github.com/nethesis/my/backend/services/logto"
)

// TokenExchangeRequest represents the request body for token exchange
type TokenExchangeRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}

// RefreshTokenRequest represents the request body for token refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ImpersonateRequest represents the request body for user impersonation
type ImpersonateRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// TokenExchangeResponse represents the response for token exchange
type TokenExchangeResponse struct {
	Token        string      `json:"token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    int64       `json:"expires_in"`
	User         models.User `json:"user"`
}

// ExchangeToken converts Logto access token to custom JWT
// POST /auth/exchange
func ExchangeToken(c *gin.Context) {
	var req TokenExchangeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.NewHTTPErrorLogger(c, "auth").LogError(err, "parse_request", http.StatusBadRequest, "Invalid request body")
		c.JSON(http.StatusBadRequest, response.BadRequest(
			"invalid request body: "+err.Error(),
			nil,
		))
		return
	}

	// Get user info from Logto
	userInfo, err := logto.GetUserInfoFromLogto(req.AccessToken)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "auth").LogError(err, "get_userinfo", http.StatusUnauthorized, "Failed to get user info from Logto")
		c.JSON(http.StatusUnauthorized, response.Unauthorized(
			"invalid access token: "+err.Error(),
			nil,
		))
		return
	}

	// Get complete user profile from Management API
	userProfile, err := logto.GetUserProfileFromLogto(userInfo.Sub)
	if err != nil {
		logger.RequestLogger(c, "auth").Warn().
			Err(err).
			Str("operation", "get_profile").
			Msg("Failed to get user profile")
		userProfile = nil
	}

	// Try to get user from local database first
	userService := local.NewUserService()
	var user models.User

	if localUser, err := userService.GetUserByLogtoID(userInfo.Sub); err == nil {
		// User exists in local DB, use local ID as primary ID
		user = models.User{
			ID:      localUser.ID,      // Local database ID as primary ID
			LogtoID: localUser.LogtoID, // Logto ID for reference
		}

		// Update latest login timestamp
		if updateErr := userService.UpdateLatestLogin(localUser.ID); updateErr != nil {
			logger.RequestLogger(c, "auth").Warn().
				Err(updateErr).
				Str("operation", "update_latest_login").
				Str("user_id", localUser.ID).
				Msg("Failed to update latest login timestamp")
			// Don't fail the request if this update fails
		}
	} else {
		// User not in local DB, create temporary user with empty local ID
		user = models.User{
			ID:      "",            // No local ID available
			LogtoID: &userInfo.Sub, // Logto ID for reference
		}
	}

	if userProfile != nil {
		user.Username = userProfile.Username
		user.Email = userProfile.PrimaryEmail
		user.Name = userProfile.Name
		if userProfile.PrimaryPhone != "" {
			user.Phone = &userProfile.PrimaryPhone
		}
	} else {
		user.Username = userInfo.Username
		user.Email = userInfo.Email
		user.Name = userInfo.Name
		// userInfo doesn't have phone, keep it nil
	}

	// Enrich user with roles and permissions
	enrichedUser, err := logto.EnrichUserWithRolesAndPermissions(userInfo.Sub)
	if err != nil {
		logger.RequestLogger(c, "auth").Warn().
			Err(err).
			Str("operation", "enrich_user").
			Msg("Failed to enrich user with roles")
	} else {
		user.UserRoles = enrichedUser.UserRoles
		user.UserRoleIDs = enrichedUser.UserRoleIDs
		user.UserPermissions = enrichedUser.UserPermissions
		user.OrgRole = enrichedUser.OrgRole
		user.OrgRoleID = enrichedUser.OrgRoleID
		user.OrgPermissions = enrichedUser.OrgPermissions
		user.OrganizationID = enrichedUser.OrganizationID
		user.OrganizationName = enrichedUser.OrganizationName
	}

	// Generate custom JWT token
	customToken, err := jwt.GenerateCustomToken(user)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "auth").LogError(err, "generate_token", http.StatusInternalServerError, "Failed to generate custom token")
		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to generate token: "+err.Error(),
			nil,
		))
		return
	}

	// Generate refresh token using Logto ID (for auth compatibility)
	refreshToken, err := jwt.GenerateRefreshToken(*user.LogtoID)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "auth").LogError(err, "generate_refresh_token", http.StatusInternalServerError, "Failed to generate refresh token")
		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to generate refresh token: "+err.Error(),
			nil,
		))
		return
	}

	// Calculate expiration in seconds
	expDuration, err := time.ParseDuration(configuration.Config.JWTExpiration)
	if err != nil {
		expDuration = 24 * time.Hour // Default fallback
	}
	expiresIn := int64(expDuration.Seconds())

	logger.LogTokenExchange(c, "auth", "access_token", true, nil)

	c.JSON(http.StatusOK, response.OK(
		"token exchange successful",
		gin.H{
			"token":         customToken,
			"refresh_token": refreshToken,
			"expires_in":    expiresIn,
			"user":          user,
		},
	))
}

// RefreshToken refreshes access token using refresh token
// POST /auth/refresh
func RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.NewHTTPErrorLogger(c, "auth").LogError(err, "parse_refresh_request", http.StatusBadRequest, "Invalid refresh request body")
		c.JSON(http.StatusBadRequest, response.BadRequest(
			"invalid request body: "+err.Error(),
			nil,
		))
		return
	}

	// Validate refresh token
	refreshClaims, err := jwt.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "auth").LogError(err, "validate_refresh_token", http.StatusUnauthorized, "Invalid refresh token")
		c.JSON(http.StatusUnauthorized, response.Unauthorized(
			"invalid refresh token: "+err.Error(),
			nil,
		))
		return
	}

	// Get fresh user information
	enrichedUser, err := logto.EnrichUserWithRolesAndPermissions(refreshClaims.UserID)
	if err != nil {
		logger.RequestLogger(c, "auth").Error().
			Err(err).
			Str("operation", "enrich_user_refresh").
			Msg("Failed to enrich user during refresh")
		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to retrieve user information: "+err.Error(),
			nil,
		))
		return
	}

	// Get complete user profile
	userProfile, err := logto.GetUserProfileFromLogto(refreshClaims.UserID)
	if err != nil {
		logger.RequestLogger(c, "auth").Warn().
			Err(err).
			Str("operation", "get_profile_refresh").
			Msg("Failed to get user profile during refresh")
		userProfile = nil
	}

	// Try to get user from local database first
	userService := local.NewUserService()
	var user models.User

	if localUser, err := userService.GetUserByLogtoID(refreshClaims.UserID); err == nil {
		// User exists in local DB, use local ID as primary ID
		user = models.User{
			ID:      localUser.ID,      // Local database ID as primary ID
			LogtoID: localUser.LogtoID, // Logto ID for reference
		}
	} else {
		// User not in local DB, create temporary user with empty local ID
		user = models.User{
			ID:      "",                    // No local ID available
			LogtoID: &refreshClaims.UserID, // Logto ID for reference
		}
	}

	if userProfile != nil {
		user.Username = userProfile.Username
		user.Email = userProfile.PrimaryEmail
		user.Name = userProfile.Name
		if userProfile.PrimaryPhone != "" {
			user.Phone = &userProfile.PrimaryPhone
		}
	}

	user.UserRoles = enrichedUser.UserRoles
	user.UserRoleIDs = enrichedUser.UserRoleIDs
	user.UserPermissions = enrichedUser.UserPermissions
	user.OrgRole = enrichedUser.OrgRole
	user.OrgRoleID = enrichedUser.OrgRoleID
	user.OrgPermissions = enrichedUser.OrgPermissions
	user.OrganizationID = enrichedUser.OrganizationID
	user.OrganizationName = enrichedUser.OrganizationName

	// Generate new tokens
	newAccessToken, err := jwt.GenerateCustomToken(user)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "auth").LogError(err, "generate_new_token", http.StatusInternalServerError, "Failed to generate new access token")
		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to generate new access token: "+err.Error(),
			nil,
		))
		return
	}

	newRefreshToken, err := jwt.GenerateRefreshToken(*user.LogtoID)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "auth").LogError(err, "generate_new_refresh_token", http.StatusInternalServerError, "Failed to generate new refresh token")
		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to generate new refresh token: "+err.Error(),
			nil,
		))
		return
	}

	// Calculate expiration in seconds
	expDuration, err := time.ParseDuration(configuration.Config.JWTExpiration)
	if err != nil {
		expDuration = 24 * time.Hour // Default fallback
	}
	expiresIn := int64(expDuration.Seconds())

	logger.LogTokenExchange(c, "auth", "refresh_token", true, nil)

	c.JSON(http.StatusOK, response.OK(
		"token refresh successful",
		gin.H{
			"token":         newAccessToken,
			"refresh_token": newRefreshToken,
			"expires_in":    expiresIn,
			"user":          user,
		},
	))
}

// GetCurrentUser returns current user information from JWT token
// GET /me
func GetCurrentUser(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	userData := gin.H{
		"id":                user.ID,
		"logto_id":          user.LogtoID,
		"username":          user.Username,
		"email":             user.Email,
		"name":              user.Name,
		"phone":             user.Phone,
		"user_roles":        user.UserRoles,
		"user_role_ids":     user.UserRoleIDs,
		"user_permissions":  user.UserPermissions,
		"org_role":          user.OrgRole,
		"org_role_id":       user.OrgRoleID,
		"org_permissions":   user.OrgPermissions,
		"organization_id":   user.OrganizationID,
		"organization_name": user.OrganizationName,
	}

	logger.RequestLogger(c, "auth").Info().
		Str("operation", "get_current_user").
		Str("user_id", user.ID).
		Str("organization_id", user.OrganizationID).
		Msg("User info requested")

	c.JSON(http.StatusOK, response.OK(
		"user information retrieved successfully",
		userData,
	))
}

// ChangePassword allows the current user to change their own password
// POST /me/change-password
func ChangePassword(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Check if user has a Logto ID (required for password operations)
	if user.LogtoID == nil || *user.LogtoID == "" {
		logger.RequestLogger(c, "auth").Warn().
			Str("operation", "change_password").
			Str("user_id", user.ID).
			Msg("User attempted password change without Logto ID")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			"password change not available for this user",
			nil,
		))
		return
	}

	// Parse request body
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.RequestLogger(c, "auth").Warn().
			Err(err).
			Str("operation", "change_password").
			Str("user_id", user.ID).
			Msg("Invalid change password request JSON")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			"invalid request body: "+err.Error(),
			nil,
		))
		return
	}

	// Manual validation with proper field names
	var validationErrors []gin.H
	if req.CurrentPassword == "" {
		validationErrors = append(validationErrors, gin.H{
			"key":     "current_password",
			"message": "required",
			"value":   "",
		})
	}
	if req.NewPassword == "" {
		validationErrors = append(validationErrors, gin.H{
			"key":     "new_password",
			"message": "required",
			"value":   "",
		})
	}

	if len(validationErrors) > 0 {
		logger.RequestLogger(c, "auth").Warn().
			Str("operation", "change_password").
			Str("user_id", user.ID).
			Msg("Missing required fields for password change")

		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "validation failed",
			"data": gin.H{
				"type":   "validation_error",
				"errors": validationErrors,
			},
		})
		return
	}

	// Validate new password strength
	isValid, passwordErrors := helpers.ValidatePasswordStrength(req.NewPassword)
	if !isValid {
		logger.RequestLogger(c, "auth").Warn().
			Strs("validation_errors", passwordErrors).
			Str("operation", "change_password").
			Str("user_id", user.ID).
			Msg("New password failed validation")

		// Convert validation errors to standard format
		var errors []gin.H
		for _, validationError := range passwordErrors {
			errors = append(errors, gin.H{
				"key":     "new_password",
				"message": validationError,
				"value":   "", // Don't expose the actual password value
			})
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "validation failed",
			"data": gin.H{
				"type":   "validation_error",
				"errors": errors,
			},
		})
		return
	}

	// Create Logto client
	logtoClient := logto.NewManagementClient()

	// Verify current password
	err := logtoClient.VerifyUserPassword(*user.LogtoID, req.CurrentPassword)
	if err != nil {
		logger.RequestLogger(c, "auth").Warn().
			Err(err).
			Str("operation", "change_password").
			Str("user_id", user.ID).
			Str("logto_id", *user.LogtoID).
			Msg("Current password verification failed")

		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "validation failed",
			"data": gin.H{
				"type": "validation_error",
				"errors": []gin.H{
					{
						"key":     "current_password",
						"message": "incorrect_password",
						"value":   "", // Don't expose the actual password value
					},
				},
			},
		})
		return
	}

	// Update password in Logto
	err = logtoClient.UpdateUserPassword(*user.LogtoID, req.NewPassword)
	if err != nil {
		logger.RequestLogger(c, "auth").Error().
			Err(err).
			Str("operation", "change_password").
			Str("user_id", user.ID).
			Str("logto_id", *user.LogtoID).
			Msg("Failed to update password in Logto")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to update password",
			map[string]interface{}{
				"error": err.Error(),
			},
		))
		return
	}

	// Log successful password change
	logger.LogAccountOperation(c, "change_password", user.ID, user.OrganizationID, user.ID, user.OrganizationID, true, nil)

	logger.RequestLogger(c, "auth").Info().
		Str("operation", "change_password").
		Str("user_id", user.ID).
		Str("logto_id", *user.LogtoID).
		Msg("Password changed successfully")

	c.JSON(http.StatusOK, response.OK(
		"password changed successfully",
		nil,
	))
}

// ChangeInfo allows the current user to change their own personal information
// POST /me/change-info
func ChangeInfo(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Check if user has a Logto ID (required for profile operations)
	if user.LogtoID == nil || *user.LogtoID == "" {
		logger.RequestLogger(c, "auth").Warn().
			Str("operation", "change_info").
			Str("user_id", user.ID).
			Msg("User attempted info change without Logto ID")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			"profile update not available for this user",
			nil,
		))
		return
	}

	// Parse request body
	var req models.ChangeInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.RequestLogger(c, "auth").Warn().
			Err(err).
			Str("operation", "change_info").
			Str("user_id", user.ID).
			Msg("Invalid change info request JSON")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			"invalid request body: "+err.Error(),
			nil,
		))
		return
	}

	// Validate that at least one field is provided
	if req.Name == nil && req.Email == nil && req.Phone == nil {
		logger.RequestLogger(c, "auth").Warn().
			Str("operation", "change_info").
			Str("user_id", user.ID).
			Msg("No fields provided for info change")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			"at least one field (name, email, or phone) must be provided",
			nil,
		))
		return
	}

	// Create Logto client
	logtoClient := logto.NewManagementClient()

	// Manual validation with proper field names
	var validationErrors []gin.H

	if req.Name != nil && *req.Name == "" {
		validationErrors = append(validationErrors, gin.H{
			"key":     "name",
			"message": "name cannot be empty",
			"value":   "",
		})
	}

	if req.Email != nil && *req.Email == "" {
		validationErrors = append(validationErrors, gin.H{
			"key":     "email",
			"message": "email cannot be empty",
			"value":   "",
		})
	}

	if len(validationErrors) > 0 {
		logger.RequestLogger(c, "auth").Warn().
			Str("operation", "change_info").
			Str("user_id", user.ID).
			Msg("Validation failed for info change")

		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "validation failed",
			"data": gin.H{
				"type":   "validation_error",
				"errors": validationErrors,
			},
		})
		return
	}

	// Prepare update data
	updateData := models.UpdateUserRequest{}
	changedFields := []string{}

	if req.Name != nil {
		updateData.Name = req.Name
		changedFields = append(changedFields, "name")
	}

	if req.Email != nil {
		updateData.PrimaryEmail = req.Email
		changedFields = append(changedFields, "email")
	}

	if req.Phone != nil {
		if *req.Phone == "" {
			// Remove phone number
			emptyPhone := ""
			updateData.PrimaryPhone = &emptyPhone
			changedFields = append(changedFields, "phone (removed)")
		} else {
			updateData.PrimaryPhone = req.Phone
			changedFields = append(changedFields, "phone")
		}
	}

	// Update user profile in Logto
	_, err := logtoClient.UpdateUser(*user.LogtoID, updateData)
	if err != nil {
		logger.RequestLogger(c, "auth").Error().
			Err(err).
			Str("operation", "change_info").
			Str("user_id", user.ID).
			Str("logto_id", *user.LogtoID).
			Strs("fields", changedFields).
			Msg("Failed to update user profile in Logto")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to update profile",
			map[string]interface{}{
				"error": err.Error(),
			},
		))
		return
	}

	// Log successful profile change
	logger.LogAccountOperation(c, "change_info", user.ID, user.OrganizationID, user.ID, user.OrganizationID, true, nil)

	logger.RequestLogger(c, "auth").Info().
		Str("operation", "change_info").
		Str("user_id", user.ID).
		Str("logto_id", *user.LogtoID).
		Strs("changed_fields", changedFields).
		Msg("Profile updated successfully")

	c.JSON(http.StatusOK, response.OK(
		"profile updated successfully",
		gin.H{
			"updated_fields": changedFields,
		},
	))
}

// Logout invalidates the current JWT token by adding it to the blacklist
// POST /api/auth/logout
func Logout(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Extract token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		logger.RequestLogger(c, "auth").Warn().
			Str("operation", "logout").
			Str("user_id", user.ID).
			Msg("Logout called without authorization header")
		c.JSON(http.StatusBadRequest, response.BadRequest("authorization header required", nil))
		return
	}

	// Check Bearer prefix and extract token
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		logger.RequestLogger(c, "auth").Warn().
			Str("operation", "logout").
			Str("user_id", user.ID).
			Msg("Invalid authorization header format for logout")
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid authorization header format", nil))
		return
	}

	tokenString := authHeader[len(bearerPrefix):]
	if tokenString == "" {
		logger.RequestLogger(c, "auth").Warn().
			Str("operation", "logout").
			Str("user_id", user.ID).
			Msg("Empty token provided for logout")
		c.JSON(http.StatusBadRequest, response.BadRequest("token not provided", nil))
		return
	}

	// Get blacklist service
	blacklist := cache.GetTokenBlacklist()

	// Blacklist the current token
	err := blacklist.BlacklistToken(tokenString, "user logout")
	if err != nil {
		logger.RequestLogger(c, "auth").Error().
			Err(err).
			Str("operation", "logout").
			Str("user_id", user.ID).
			Msg("Failed to blacklist token during logout")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"logout failed",
			map[string]interface{}{
				"error": err.Error(),
			},
		))
		return
	}

	// Log successful logout
	logger.RequestLogger(c, "auth").Info().
		Str("operation", "logout_success").
		Str("user_id", user.ID).
		Str("username", user.Username).
		Msg("User successfully logged out and token blacklisted")

	c.JSON(http.StatusOK, response.OK(
		"logout successful",
		gin.H{
			"message": "token has been invalidated",
		},
	))
}
