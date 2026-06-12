/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
)

// GetApplicationFilters handles GET /api/filters/applications - aggregated
// filters endpoint for the applications views.
//
// Returns the catalog of application types and versions in the caller's scope.
// Systems and organizations dropdowns are populated by /api/systems and
// /api/organizations respectively, which support search + pagination and scale
// to tenants with thousands of rows.
func GetApplicationFilters(c *gin.Context) {
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	appsService := local.NewApplicationsService()

	// Owner can access everything - pass nil to skip RBAC filtering in queries
	var allowedSystemIDs []string
	if strings.ToLower(userOrgRole) != "owner" {
		var err error
		allowedSystemIDs, err = appsService.GetAllowedSystemIDs(userOrgRole, userOrgID)
		if err != nil {
			logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get allowed systems for filters")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve access", nil))
			return
		}
	}

	var (
		types    []models.ApplicationType
		versions map[string]entities.ApplicationVersionGroup

		errTypes, errVersions error
		wg                    sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		types, errTypes = appsService.GetApplicationTypesWithIDs(allowedSystemIDs)
	}()

	go func() {
		defer wg.Done()
		versions, errVersions = appsService.GetApplicationVersionsWithIDs(allowedSystemIDs)
	}()

	wg.Wait()

	for _, e := range []error{errTypes, errVersions} {
		if e != nil {
			logger.Error().Err(e).Str("user_id", userID).Msg("Failed in application filters")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve application filters", nil))
			return
		}
	}

	type ApplicationVersions struct {
		Application string   `json:"application"`
		Name        string   `json:"name"`
		Versions    []string `json:"versions"`
	}

	groupedVersions := make([]ApplicationVersions, 0)
	for application, group := range versions {
		groupedVersions = append(groupedVersions, ApplicationVersions{
			Application: application,
			Name:        group.Name,
			Versions:    group.Versions,
		})
	}
	sort.Slice(groupedVersions, func(i, j int) bool {
		return groupedVersions[i].Application < groupedVersions[j].Application
	})

	c.JSON(http.StatusOK, response.OK("application filters retrieved successfully", gin.H{
		"types":    helpers.EnsureSlice(types),
		"versions": groupedVersions,
	}))
}
