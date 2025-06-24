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

// TokenExchangeResponse represents the response for token exchange
type TokenExchangeResponse struct {
	Token     string      `json:"token"`
	ExpiresIn int64       `json:"expires_in"`
	User      models.User `json:"user"`
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

	// Get user info from Logto using the access token
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

	// Create basic user from Logto info
	user := models.User{
		ID: userInfo.Sub,
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

	// Calculate expiration (24h in seconds)
	expiresIn := int64(24 * 60 * 60) // 24 hours

	logs.Logs.Printf("[INFO][AUTH] Token exchange successful for user: %s", user.ID)

	// Return custom token
	c.JSON(http.StatusOK, TokenExchangeResponse{
		Token:     customToken,
		ExpiresIn: expiresIn,
		User:      user,
	})
}