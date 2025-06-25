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

	"github.com/gin-gonic/gin"
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

// ExchangeToken handles the token exchange endpoint
// POST /auth/exchange
func ExchangeToken(c *gin.Context) {
	var req TokenExchangeRequest

	// Bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		logs.Logs.Println("[ERROR][AUTH] Invalid request body:", err.Error())
		c.JSON(http.StatusBadRequest, response.StatusBadRequest{
			Code:    400,
			Message: "Invalid request body: " + err.Error(),
			Data:    nil,
		})
		return
	}

	// Get user info from Logto using the access token (for basic validation and getting user ID)
	userInfo, err := services.GetUserInfoFromLogto(req.AccessToken)
	if err != nil {
		logs.Logs.Println("[ERROR][AUTH] Failed to get user info from Logto:", err.Error())
		c.JSON(http.StatusUnauthorized, response.StatusUnauthorized{
			Code:    401,
			Message: "Invalid access token: " + err.Error(),
			Data:    nil,
		})
		return
	}

	// Get complete user profile from Management API
	userProfile, err := services.GetUserProfileFromLogto(userInfo.Sub)
	if err != nil {
		logs.Logs.Printf("[WARN][AUTH] Failed to get user profile from Management API: %v", err)
		// Fallback to basic userinfo
		userProfile = nil
	}

	// Create user with complete profile data
	user := models.User{
		ID: userInfo.Sub,
	}

	// Use Management API profile data if available, otherwise fallback to userinfo
	if userProfile != nil {
		user.Username = userProfile.Username
		user.Email = userProfile.PrimaryEmail
		user.Name = userProfile.Name
		logs.Logs.Printf("[DEBUG][AUTH] Using Management API profile: username=%s, email=%s, name=%s",
			user.Username, user.Email, user.Name)
	} else {
		user.Username = userInfo.Username
		user.Email = userInfo.Email
		user.Name = userInfo.Name
		logs.Logs.Printf("[DEBUG][AUTH] Using basic userinfo: username=%s, email=%s, name=%s",
			user.Username, user.Email, user.Name)
	}

	// Enrich user with roles and permissions
	// TODO: Implement full role/permission enrichment from Management API
	enrichedUser, err := services.EnrichUserWithRolesAndPermissions(userInfo.Sub)
	if err != nil {
		logs.Logs.Println("[WARN][AUTH] Failed to enrich user with roles:", err.Error())
		// Continue with basic user info for now
	} else {
		// Merge enriched data
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
		c.JSON(http.StatusInternalServerError, response.StatusInternalServerError{
			Code:    500,
			Message: "Failed to generate token: " + err.Error(),
			Data:    nil,
		})
		return
	}

	// Generate refresh token
	refreshToken, err := jwt.GenerateRefreshToken(user.ID)
	if err != nil {
		logs.Logs.Println("[ERROR][AUTH] Failed to generate refresh token:", err.Error())
		c.JSON(http.StatusInternalServerError, response.StatusInternalServerError{
			Code:    500,
			Message: "Failed to generate refresh token: " + err.Error(),
			Data:    nil,
		})
		return
	}

	// Calculate expiration (24h in seconds)
	expiresIn := int64(24 * 60 * 60) // 24 hours

	logs.Logs.Printf("[INFO][AUTH] Token exchange successful for user: %s", user.ID)

	// Return custom token and refresh token
	c.JSON(http.StatusOK, TokenExchangeResponse{
		Token:        customToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		User:         user,
	})
}

// GetCurrentUser handles GET /api/auth/me - returns current user information from JWT token
func GetCurrentUser(c *gin.Context) {
	// Extract user information from JWT token (set by middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "user ID not found in token",
			"data":    nil,
		})
		return
	}

	username, _ := c.Get("username")
	email, _ := c.Get("email")
	name, _ := c.Get("name")
	userRoles, _ := c.Get("user_roles")
	userPermissions, _ := c.Get("user_permissions")
	orgRole, _ := c.Get("org_role")
	orgPermissions, _ := c.Get("org_permissions")
	organizationID, _ := c.Get("organization_id")
	organizationName, _ := c.Get("organization_name")

	// Build response with current user information
	userData := gin.H{
		"id":               userID,
		"username":         username,
		"email":            email,
		"name":             name,
		"userRoles":        userRoles,
		"userPermissions":  userPermissions,
		"orgRole":          orgRole,
		"orgPermissions":   orgPermissions,
		"organizationId":   organizationID,
		"organizationName": organizationName,
	}

	logs.Logs.Printf("[INFO][AUTH] User info requested: %s (org: %s, role: %s)", userID, organizationID, orgRole)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "user information retrieved successfully",
		"data":    userData,
	})
}

// RefreshToken handles POST /auth/refresh - refreshes an access token using a refresh token
func RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest

	// Bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		logs.Logs.Println("[ERROR][AUTH] Invalid refresh request body:", err.Error())
		c.JSON(http.StatusBadRequest, response.StatusBadRequest{
			Code:    400,
			Message: "Invalid request body: " + err.Error(),
			Data:    nil,
		})
		return
	}

	// Validate refresh token
	refreshClaims, err := jwt.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		logs.Logs.Println("[ERROR][AUTH] Invalid refresh token:", err.Error())
		c.JSON(http.StatusUnauthorized, response.StatusUnauthorized{
			Code:    401,
			Message: "Invalid refresh token: " + err.Error(),
			Data:    nil,
		})
		return
	}

	// Get user information using the user ID from refresh token
	enrichedUser, err := services.EnrichUserWithRolesAndPermissions(refreshClaims.UserID)
	if err != nil {
		logs.Logs.Println("[ERROR][AUTH] Failed to enrich user during refresh:", err.Error())
		c.JSON(http.StatusInternalServerError, response.StatusInternalServerError{
			Code:    500,
			Message: "Failed to retrieve user information: " + err.Error(),
			Data:    nil,
		})
		return
	}

	// Get complete user profile from Management API
	userProfile, err := services.GetUserProfileFromLogto(refreshClaims.UserID)
	if err != nil {
		logs.Logs.Printf("[WARN][AUTH] Failed to get user profile during refresh: %v", err)
		userProfile = nil
	}

	// Create user object with fresh data
	user := models.User{
		ID: refreshClaims.UserID,
	}

	// Use Management API profile data if available
	if userProfile != nil {
		user.Username = userProfile.Username
		user.Email = userProfile.PrimaryEmail
		user.Name = userProfile.Name
	}

	// Add enriched role and permission data
	user.UserRoles = enrichedUser.UserRoles
	user.UserPermissions = enrichedUser.UserPermissions
	user.OrgRole = enrichedUser.OrgRole
	user.OrgPermissions = enrichedUser.OrgPermissions
	user.OrganizationID = enrichedUser.OrganizationID
	user.OrganizationName = enrichedUser.OrganizationName

	// Generate new access token
	newAccessToken, err := jwt.GenerateCustomToken(user)
	if err != nil {
		logs.Logs.Println("[ERROR][AUTH] Failed to generate new access token:", err.Error())
		c.JSON(http.StatusInternalServerError, response.StatusInternalServerError{
			Code:    500,
			Message: "Failed to generate new access token: " + err.Error(),
			Data:    nil,
		})
		return
	}

	// Generate new refresh token
	newRefreshToken, err := jwt.GenerateRefreshToken(user.ID)
	if err != nil {
		logs.Logs.Println("[ERROR][AUTH] Failed to generate new refresh token:", err.Error())
		c.JSON(http.StatusInternalServerError, response.StatusInternalServerError{
			Code:    500,
			Message: "Failed to generate new refresh token: " + err.Error(),
			Data:    nil,
		})
		return
	}

	// Calculate expiration (24h in seconds)
	expiresIn := int64(24 * 60 * 60) // 24 hours

	logs.Logs.Printf("[INFO][AUTH] Token refresh successful for user: %s", user.ID)

	// Return new tokens
	c.JSON(http.StatusOK, TokenExchangeResponse{
		Token:        newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    expiresIn,
		User:         user,
	})
}
