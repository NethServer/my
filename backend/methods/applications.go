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

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services"
)

// GetApplications handles GET /api/applications
// Returns third-party applications filtered by user access permissions
func GetApplications(c *gin.Context) {
	// Extract user context
	userID, exists := c.Get("user_id")
	if !exists {
		logger.NewHTTPErrorLogger(c, "applications").LogError(nil, "missing_context", http.StatusUnauthorized, "User context not found in GetApplications")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		logger.NewHTTPErrorLogger(c, "applications").LogError(nil, "invalid_user_id", http.StatusUnauthorized, "Invalid user ID in context")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}

	logger.Info().
		Str("user_id", userIDStr).
		Msg("Fetching applications for user")

	// Create Logto client
	client := services.NewLogtoManagementClient()

	// Fetch all third-party applications from Logto
	logtoApplications, err := client.GetThirdPartyApplications()
	if err != nil {
		logger.NewHTTPErrorLogger(c, "applications").LogError(err, "fetch_applications", http.StatusInternalServerError, "Failed to fetch applications from Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch applications", err.Error()))
		return
	}

	// Get user's organization role
	var organizationRoles []string
	if orgRole, exists := c.Get("org_role"); exists {
		if orgRoleStr, ok := orgRole.(string); ok && orgRoleStr != "" {
			organizationRoles = append(organizationRoles, orgRoleStr)
		}
	}

	// Get user's user roles
	var userRoles []string
	if userRolesData, exists := c.Get("user_roles"); exists {
		if userRolesList, ok := userRolesData.([]string); ok {
			userRoles = userRolesList
		}
	}

	logger.Debug().
		Str("user_id", userIDStr).
		Strs("organization_roles", organizationRoles).
		Strs("user_roles", userRoles).
		Msg("User roles for application filtering")

	// Filter applications based on user's roles
	filteredLogtoApps := services.FilterApplicationsByAccess(logtoApplications, organizationRoles, userRoles)

	// Convert filtered applications to our response model
	var responseApplications []models.ThirdPartyApplication
	for _, app := range filteredLogtoApps {
		// Get branding information
		branding, err := client.GetApplicationBranding(app.ID)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("app_id", app.ID).
				Msg("Failed to get branding for app")
		}

		// Get scopes
		scopes, err := client.GetApplicationScopes(app.ID)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("app_id", app.ID).
				Msg("Failed to get scopes for app")
		}

		// Convert to our response model
		convertedApp := app.ToThirdPartyApplication(branding, scopes, services.GenerateOAuth2LoginURL)
		responseApplications = append(responseApplications, *convertedApp)
	}

	logger.Info().
		Int("count", len(responseApplications)).
		Str("user_id", userIDStr).
		Msg("Returning applications for user")

	// Return filtered applications
	c.JSON(http.StatusOK, response.Success(http.StatusOK, "Applications retrieved successfully", responseApplications))
}
