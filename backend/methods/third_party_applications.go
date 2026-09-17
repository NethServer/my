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
	"sort"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
	"github.com/nethesis/my/backend/services/logto"
)

// GetThirdPartyApplications handles GET /api/third-party-applications
// Returns third-party applications filtered by user access permissions
func GetThirdPartyApplications(c *gin.Context) {
	// Extract user context
	userID, exists := c.Get("user_id")
	if !exists {
		logger.NewHTTPErrorLogger(c, "third-party-applications").LogError(nil, "missing_context", http.StatusUnauthorized, "User context not found in GetThirdPartyApplications")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		logger.NewHTTPErrorLogger(c, "third-party-applications").LogError(nil, "invalid_user_id", http.StatusUnauthorized, "Invalid user ID in context")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}

	logger.Info().
		Str("user_id", userIDStr).
		Msg("Fetching third-party applications for user")

	// Get user's organization role
	var organizationRoles []string
	if orgRole, exists := c.Get("org_role"); exists {
		if orgRoleStr, ok := orgRole.(string); ok && orgRoleStr != "" {
			organizationRoles = append(organizationRoles, orgRoleStr)
		}
	}

	// Get user's user role IDs for access control matching
	var userRoleIDs []string
	if userRoleIDsData, exists := c.Get("user_role_ids"); exists {
		if userRoleIDsList, ok := userRoleIDsData.([]string); ok {
			userRoleIDs = userRoleIDsList
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
		Strs("user_role_ids", userRoleIDs).
		Msg("User context for application filtering")

	// For a reseller or customer, the distributor at the top of its branch
	// decides which portals it may use. Resolved first: a branch with no
	// portal at all is answered without a round trip to Logto.
	userOrgRole := ""
	if len(organizationRoles) > 0 {
		userOrgRole = organizationRoles[0]
	}
	allowedApps, restricted, err := local.NewThirdPartyAppsService().ResolveAllowedApps(userOrgRole, userOrganizationID)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "third-party-applications").LogError(err, "resolve_allowed_apps", http.StatusInternalServerError, "Failed to resolve the portals allowed for the user's hierarchy")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch third-party applications", err.Error()))
		return
	}
	if restricted && len(allowedApps) == 0 {
		logger.Info().
			Str("user_id", userIDStr).
			Str("organization_id", userOrganizationID).
			Msg("No portal enabled for the user's hierarchy")
		c.JSON(http.StatusOK, response.Success(http.StatusOK, "third-party applications retrieved successfully", []models.ThirdPartyApplication{}))
		return
	}

	// Create Logto client
	client := logto.NewManagementClient()

	// Fetch all third-party applications from Logto
	logtoApplications, err := client.GetThirdPartyApplications()
	if err != nil {
		logger.NewHTTPErrorLogger(c, "third-party-applications").LogError(err, "fetch_applications", http.StatusInternalServerError, "Failed to fetch third-party applications from Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch third-party applications", err.Error()))
		return
	}

	// Filter applications based on user's roles and organization membership
	filteredLogtoApps := logto.FilterApplicationsByAccess(logtoApplications, organizationRoles, userRoleIDs, userOrganizationID)
	if restricted {
		filteredLogtoApps = logto.FilterApplicationsByNames(filteredLogtoApps, allowedApps)
	}

	// Get cached domain validation result
	domainValidation := cache.GetDomainValidation()
	isValidDomain := domainValidation.IsValid()

	// Convert filtered applications to our response model using parallel processing
	responseApplications := make([]models.ThirdPartyApplication, 0)
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, 10)

	for _, app := range filteredLogtoApps {
		wg.Add(1)
		sem <- struct{}{}
		go func(app models.LogtoThirdPartyApp) {
			defer func() { <-sem }()
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
			convertedApp := app.ToThirdPartyApplication(branding, scopes, func(appID string, redirectURI string, scopes []string, isValidDomain bool) (string, error) {
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
		Msg("Returning third-party applications for user")

	// Return filtered applications
	c.JSON(http.StatusOK, response.Success(http.StatusOK, "third-party applications retrieved successfully", responseApplications))
}

// GetThirdPartyApplicationsCatalog handles GET /api/third-party-applications/catalog
// Returns the portals a distributor's resellers and customers can be granted: every
// third-party application whose own access_control admits a partner
// organization role. Owner organization only: this is the picker the Owner
// uses when it fills in a distributor's portal list, not a user-facing list.
func GetThirdPartyApplicationsCatalog(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	if !models.IsGlobalOrgRole(user.OrgRole) {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied: only the owner organization can list the portal catalogue", nil))
		return
	}

	client := logto.NewManagementClient()
	logtoApplications, err := client.GetThirdPartyApplications()
	if err != nil {
		logger.NewHTTPErrorLogger(c, "third-party-applications").LogError(err, "fetch_applications", http.StatusInternalServerError, "Failed to fetch third-party applications from Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch third-party applications", err.Error()))
		return
	}

	items := make([]models.ThirdPartyApplicationCatalogItem, 0, len(logtoApplications))
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, 10)

	for _, app := range logtoApplications {
		if !logto.IsPartnerAccessible(app) {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(app models.LogtoThirdPartyApp) {
			defer func() { <-sem }()
			defer wg.Done()

			item := models.ThirdPartyApplicationCatalogItem{
				Name:        app.Name,
				DisplayName: app.Name,
				Description: app.Description,
			}
			branding, err := client.GetApplicationBranding(app.ID)
			if err != nil {
				logger.Warn().
					Err(err).
					Str("app_id", app.ID).
					Msg("Failed to get branding for app")
			} else if branding != nil && branding.DisplayName != "" {
				item.DisplayName = branding.DisplayName
			}

			mu.Lock()
			items = append(items, item)
			mu.Unlock()
		}(app)
	}
	wg.Wait()

	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	logger.Info().
		Int("count", len(items)).
		Str("user_id", user.ID).
		Msg("Returning third-party applications catalogue")

	c.JSON(http.StatusOK, response.Success(http.StatusOK, "third-party applications catalogue retrieved successfully", items))
}
