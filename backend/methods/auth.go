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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/jwt"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services"
)

// TokenExchangeRequest represents the request body for token exchange
type TokenExchangeRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}

// RefreshTokenRequest represents the request body for token refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
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
			"Invalid request body: "+err.Error(),
			nil,
		))
		return
	}

	// Get user info from Logto
	userInfo, err := services.GetUserInfoFromLogto(req.AccessToken)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "auth").LogError(err, "get_userinfo", http.StatusUnauthorized, "Failed to get user info from Logto")
		c.JSON(http.StatusUnauthorized, response.Unauthorized(
			"Invalid access token: "+err.Error(),
			nil,
		))
		return
	}

	// Get complete user profile from Management API
	userProfile, err := services.GetUserProfileFromLogto(userInfo.Sub)
	if err != nil {
		logger.RequestLogger(c, "auth").Warn().
			Err(err).
			Str("operation", "get_profile").
			Msg("Failed to get user profile")
		userProfile = nil
	}

	// Create user with profile data
	user := models.User{ID: userInfo.Sub}

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
	enrichedUser, err := services.EnrichUserWithRolesAndPermissions(userInfo.Sub)
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
			"Failed to generate token: "+err.Error(),
			nil,
		))
		return
	}

	// Generate refresh token
	refreshToken, err := jwt.GenerateRefreshToken(user.ID)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "auth").LogError(err, "generate_refresh_token", http.StatusInternalServerError, "Failed to generate refresh token")
		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"Failed to generate refresh token: "+err.Error(),
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
			"Invalid request body: "+err.Error(),
			nil,
		))
		return
	}

	// Validate refresh token
	refreshClaims, err := jwt.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "auth").LogError(err, "validate_refresh_token", http.StatusUnauthorized, "Invalid refresh token")
		c.JSON(http.StatusUnauthorized, response.Unauthorized(
			"Invalid refresh token: "+err.Error(),
			nil,
		))
		return
	}

	// Get fresh user information
	enrichedUser, err := services.EnrichUserWithRolesAndPermissions(refreshClaims.UserID)
	if err != nil {
		logger.RequestLogger(c, "auth").Error().
			Err(err).
			Str("operation", "enrich_user_refresh").
			Msg("Failed to enrich user during refresh")
		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"Failed to retrieve user information: "+err.Error(),
			nil,
		))
		return
	}

	// Get complete user profile
	userProfile, err := services.GetUserProfileFromLogto(refreshClaims.UserID)
	if err != nil {
		logger.RequestLogger(c, "auth").Warn().
			Err(err).
			Str("operation", "get_profile_refresh").
			Msg("Failed to get user profile during refresh")
		userProfile = nil
	}

	// Create fresh user object
	user := models.User{ID: refreshClaims.UserID}

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
			"Failed to generate new access token: "+err.Error(),
			nil,
		))
		return
	}

	newRefreshToken, err := jwt.GenerateRefreshToken(user.ID)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "auth").LogError(err, "generate_new_refresh_token", http.StatusInternalServerError, "Failed to generate new refresh token")
		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"Failed to generate new refresh token: "+err.Error(),
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
// GET /auth/me
func GetCurrentUser(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	userData := gin.H{
		"id":               user.ID,
		"username":         user.Username,
		"email":            user.Email,
		"name":             user.Name,
		"phone":            user.Phone,
		"userRoles":        user.UserRoles,
		"userRoleIds":      user.UserRoleIDs,
		"userPermissions":  user.UserPermissions,
		"orgRole":          user.OrgRole,
		"orgRoleId":        user.OrgRoleID,
		"orgPermissions":   user.OrgPermissions,
		"organizationId":   user.OrganizationID,
		"organizationName": user.OrganizationName,
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
