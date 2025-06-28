/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
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
	"github.com/nethesis/my/backend/logs"
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
		logs.Logs.Println("[ERROR][AUTH] Invalid request body:", err.Error())
		c.JSON(http.StatusBadRequest, response.BadRequest(
			"Invalid request body: "+err.Error(),
			nil,
		))
		return
	}

	// Get user info from Logto
	userInfo, err := services.GetUserInfoFromLogto(req.AccessToken)
	if err != nil {
		logs.Logs.Println("[ERROR][AUTH] Failed to get user info from Logto:", err.Error())
		c.JSON(http.StatusUnauthorized, response.Unauthorized(
			"Invalid access token: "+err.Error(),
			nil,
		))
		return
	}

	// Get complete user profile from Management API
	userProfile, err := services.GetUserProfileFromLogto(userInfo.Sub)
	if err != nil {
		logs.Logs.Printf("[WARN][AUTH] Failed to get user profile: %v", err)
		userProfile = nil
	}

	// Create user with profile data
	user := models.User{ID: userInfo.Sub}

	if userProfile != nil {
		user.Username = userProfile.Username
		user.Email = userProfile.PrimaryEmail
		user.Name = userProfile.Name
	} else {
		user.Username = userInfo.Username
		user.Email = userInfo.Email
		user.Name = userInfo.Name
	}

	// Enrich user with roles and permissions
	enrichedUser, err := services.EnrichUserWithRolesAndPermissions(userInfo.Sub)
	if err != nil {
		logs.Logs.Println("[WARN][AUTH] Failed to enrich user with roles:", err.Error())
	} else {
		user.UserRoles = enrichedUser.UserRoles
		user.UserPermissions = enrichedUser.UserPermissions
		user.OrgRole = enrichedUser.OrgRole
		user.OrgPermissions = enrichedUser.OrgPermissions
		user.OrganizationID = enrichedUser.OrganizationID
		user.OrganizationName = enrichedUser.OrganizationName
	}

	// Generate custom JWT token
	customToken, err := jwt.GenerateCustomToken(user)
	if err != nil {
		logs.Logs.Println("[ERROR][AUTH] Failed to generate custom token:", err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"Failed to generate token: "+err.Error(),
			nil,
		))
		return
	}

	// Generate refresh token
	refreshToken, err := jwt.GenerateRefreshToken(user.ID)
	if err != nil {
		logs.Logs.Println("[ERROR][AUTH] Failed to generate refresh token:", err.Error())
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

	logs.Logs.Printf("[INFO][AUTH] Token exchange successful for user: %s", user.ID)

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
		logs.Logs.Println("[ERROR][AUTH] Invalid refresh request body:", err.Error())
		c.JSON(http.StatusBadRequest, response.BadRequest(
			"Invalid request body: "+err.Error(),
			nil,
		))
		return
	}

	// Validate refresh token
	refreshClaims, err := jwt.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		logs.Logs.Println("[ERROR][AUTH] Invalid refresh token:", err.Error())
		c.JSON(http.StatusUnauthorized, response.Unauthorized(
			"Invalid refresh token: "+err.Error(),
			nil,
		))
		return
	}

	// Get fresh user information
	enrichedUser, err := services.EnrichUserWithRolesAndPermissions(refreshClaims.UserID)
	if err != nil {
		logs.Logs.Println("[ERROR][AUTH] Failed to enrich user during refresh:", err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"Failed to retrieve user information: "+err.Error(),
			nil,
		))
		return
	}

	// Get complete user profile
	userProfile, err := services.GetUserProfileFromLogto(refreshClaims.UserID)
	if err != nil {
		logs.Logs.Printf("[WARN][AUTH] Failed to get user profile during refresh: %v", err)
		userProfile = nil
	}

	// Create fresh user object
	user := models.User{ID: refreshClaims.UserID}

	if userProfile != nil {
		user.Username = userProfile.Username
		user.Email = userProfile.PrimaryEmail
		user.Name = userProfile.Name
	}

	user.UserRoles = enrichedUser.UserRoles
	user.UserPermissions = enrichedUser.UserPermissions
	user.OrgRole = enrichedUser.OrgRole
	user.OrgPermissions = enrichedUser.OrgPermissions
	user.OrganizationID = enrichedUser.OrganizationID
	user.OrganizationName = enrichedUser.OrganizationName

	// Generate new tokens
	newAccessToken, err := jwt.GenerateCustomToken(user)
	if err != nil {
		logs.Logs.Println("[ERROR][AUTH] Failed to generate new access token:", err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"Failed to generate new access token: "+err.Error(),
			nil,
		))
		return
	}

	newRefreshToken, err := jwt.GenerateRefreshToken(user.ID)
	if err != nil {
		logs.Logs.Println("[ERROR][AUTH] Failed to generate new refresh token:", err.Error())
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

	logs.Logs.Printf("[INFO][AUTH] Token refresh successful for user: %s", user.ID)

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
		"userRoles":        user.UserRoles,
		"userPermissions":  user.UserPermissions,
		"orgRole":          user.OrgRole,
		"orgPermissions":   user.OrgPermissions,
		"organizationId":   user.OrganizationID,
		"organizationName": user.OrganizationName,
	}

	logs.Logs.Printf("[INFO][AUTH] User info requested: %s (org: %s)", user.ID, user.OrganizationID)

	c.JSON(http.StatusOK, response.OK(
		"user information retrieved successfully",
		userData,
	))
}
