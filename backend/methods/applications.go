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
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/logto"
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
	client := logto.NewManagementClient()

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

	// Get user's organization ID
	var userOrganizationID string
	if orgID, exists := c.Get("organization_id"); exists {
		if orgIDStr, ok := orgID.(string); ok {
			userOrganizationID = orgIDStr
		}
	}

	logger.Debug().
		Str("user_id", userIDStr).
		Str("organization_id", userOrganizationID).
		Strs("organization_roles", organizationRoles).
		Strs("user_roles", userRoles).
		Msg("User context for application filtering")

	// Filter applications based on user's roles and organization membership
	filteredLogtoApps := logto.FilterApplicationsByAccess(logtoApplications, organizationRoles, userRoles, userOrganizationID)

	// Get cached domain validation result
	domainValidation := cache.GetDomainValidation()
	isValidDomain := domainValidation.IsValid()

	// Convert filtered applications to our response model using parallel processing
	var responseApplications []models.ThirdPartyApplication
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, app := range filteredLogtoApps {
		wg.Add(1)
		go func(app models.LogtoThirdPartyApp) {
			defer wg.Done()

			// Parallel calls for branding and scopes
			var branding *models.ApplicationSignInExperience
			var scopes []string
			var brandingWg sync.WaitGroup

			brandingWg.Add(2)

			// Get branding information in parallel
			go func() {
				defer brandingWg.Done()
				var err error
				branding, err = client.GetApplicationBranding(app.ID)
				if err != nil {
					logger.Warn().
						Err(err).
						Str("app_id", app.ID).
						Msg("Failed to get branding for app")
				}
			}()

			// Get scopes in parallel
			go func() {
				defer brandingWg.Done()
				var err error
				scopes, err = client.GetApplicationScopes(app.ID)
				if err != nil {
					logger.Warn().
						Err(err).
						Str("app_id", app.ID).
						Msg("Failed to get scopes for app")
				}
			}()

			brandingWg.Wait()

			// Convert to our response model with cached domain validation
			convertedApp := app.ToThirdPartyApplication(branding, scopes, func(appID string, redirectURI string, scopes []string, isValidDomain bool) string {
				return logto.GenerateOAuth2LoginURL(appID, redirectURI, scopes, isValidDomain)
			}, isValidDomain)

			// Thread-safe append
			mu.Lock()
			responseApplications = append(responseApplications, *convertedApp)
			mu.Unlock()
		}(app)
	}

	wg.Wait()

	logger.Info().
		Int("count", len(responseApplications)).
		Str("user_id", userIDStr).
		Msg("Returning applications for user")

	// Return filtered applications
	c.JSON(http.StatusOK, response.Success(http.StatusOK, "Applications retrieved successfully", responseApplications))
}
