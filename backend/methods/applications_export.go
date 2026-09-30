/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/export"
	"github.com/nethesis/my/backend/services/local"
)

const (
	MaxApplicationsExportLimit = 10000 // Maximum applications to export (DoS protection)
)

// ExportApplications handles GET /api/applications/export - exports applications with applied filters
func ExportApplications(c *gin.Context) {
	format := strings.ToLower(c.Query("format"))
	if format != "csv" && format != "pdf" {
		c.JSON(http.StatusBadRequest, response.BadRequest("format parameter required (csv or pdf)", nil))
		return
	}

	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Same filter set as GetApplications
	search := c.Query("search")
	filterTypes := c.QueryArray("type")
	filterVersions := c.QueryArray("version")
	filterSystemIDs := c.QueryArray("system_id")
	filterOrgIDs := c.QueryArray("organization_id")
	filterStatuses := c.QueryArray("status")

	// include_hierarchy expands each organization_id filter to the org plus its
	// whole subtree (resellers/customers); the applications RBAC scope still
	// applies on top, so the expansion can never widen visibility.
	if c.Query("include_hierarchy") == "true" && len(filterOrgIDs) > 0 {
		expanded, err := local.NewOrganizationService().ExpandOrganizationIDs(filterOrgIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to expand organization hierarchy", nil))
			return
		}
		filterOrgIDs = expanded
	}

	filterManagedByOrgIDs, ok := managedByOrganizationIDs(c)
	if !ok {
		return
	}

	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortDirection := c.DefaultQuery("sort_direction", "desc")

	appsService := local.NewApplicationsService()

	// No pagination for export: a single page capped at the export limit
	apps, totalCount, err := appsService.GetApplications(
		c.Request.Context(),
		userOrgRole, userOrgID,
		1, MaxApplicationsExportLimit,
		search, sortBy, sortDirection,
		filterTypes, filterVersions, filterSystemIDs, filterOrgIDs, filterManagedByOrgIDs, filterStatuses,
	)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Str("format", format).
			Msg("Failed to retrieve applications for export")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve applications for export", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	if totalCount > MaxApplicationsExportLimit {
		logger.Warn().
			Str("user_id", userID).
			Int("total_count", totalCount).
			Int("max_limit", MaxApplicationsExportLimit).
			Msg("Export limit exceeded")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			fmt.Sprintf("too many applications to export (%d). maximum allowed: %d. please apply more filters", totalCount, MaxApplicationsExportLimit),
			map[string]interface{}{
				"total_count": totalCount,
				"max_limit":   MaxApplicationsExportLimit,
			},
		))
		return
	}

	filters := buildApplicationsFiltersMap(search, filterTypes, filterVersions, filterSystemIDs, filterOrgIDs, filterStatuses)
	if parents := c.QueryArray("parent_organization_id"); len(parents) > 0 {
		filters["parent_organization_id"] = strings.Join(parents, ", ")
	}

	exportService := export.NewApplicationsExportService()

	var fileBytes []byte
	var contentType string
	var filename string

	timestamp := time.Now().Format("2006-01-02_150405")

	switch format {
	case "csv":
		fileBytes, err = exportService.ExportToCSV(apps)
		if err != nil {
			logger.Error().
				Err(err).
				Str("user_id", userID).
				Msg("Failed to generate CSV export")

			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to generate CSV export", map[string]interface{}{
				"error": err.Error(),
			}))
			return
		}
		contentType = "text/csv"
		filename = fmt.Sprintf("applications_export_%s.csv", timestamp)

	case "pdf":
		exportedBy := fmt.Sprintf("%s (%s)", user.Name, user.Email)
		fileBytes, err = exportService.ExportToPDF(apps, filters, exportedBy)
		if err != nil {
			logger.Error().
				Err(err).
				Str("user_id", userID).
				Msg("Failed to generate PDF export")

			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to generate PDF export", map[string]interface{}{
				"error": err.Error(),
			}))
			return
		}
		contentType = "application/pdf"
		filename = fmt.Sprintf("applications_export_%s.pdf", timestamp)
	}

	logger.RequestLogger(c, "applications").Info().
		Str("operation", "export").
		Str("format", format).
		Int("application_count", len(apps)).
		Interface("filters", filters).
		Msg("Applications exported successfully")

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(fileBytes)))

	c.Data(http.StatusOK, contentType, fileBytes)
}

// buildApplicationsFiltersMap collects the applied filters for the PDF metadata and the log line
func buildApplicationsFiltersMap(search string, filterTypes, filterVersions, filterSystemIDs, filterOrgIDs, filterStatuses []string) map[string]interface{} {
	filters := make(map[string]interface{})

	if search != "" {
		filters["search"] = search
	}
	if len(filterTypes) > 0 {
		filters["type"] = strings.Join(filterTypes, ", ")
	}
	if len(filterVersions) > 0 {
		filters["version"] = strings.Join(filterVersions, ", ")
	}
	if len(filterSystemIDs) > 0 {
		filters["system_id"] = strings.Join(filterSystemIDs, ", ")
	}
	if len(filterOrgIDs) > 0 {
		filters["organization_id"] = strings.Join(filterOrgIDs, ", ")
	}
	if len(filterStatuses) > 0 {
		filters["status"] = strings.Join(filterStatuses, ", ")
	}

	return filters
}
