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
	MaxExportLimit = 10000 // Maximum systems to export (DoS protection)
)

// ExportSystems handles GET /api/systems/export - exports systems with applied filters
func ExportSystems(c *gin.Context) {
	// Get export format
	format := strings.ToLower(c.Query("format"))
	if format != "csv" && format != "pdf" {
		c.JSON(http.StatusBadRequest, response.BadRequest("format parameter required (csv or pdf)", nil))
		return
	}

	// Get current user context
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Get user info for export metadata
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Parse filter parameters (same as GetSystems)
	search := c.Query("search")
	filterName := c.Query("name")
	filterSystemKey := c.Query("system_key")
	filterTypes := c.QueryArray("type")
	filterCreatedBy := c.QueryArray("created_by")
	filterVersions := c.QueryArray("version")
	filterOrgIDs := c.QueryArray("organization_id")
	filterStatuses := c.QueryArray("status")

	// For export, we don't use pagination - get all matching systems (with limit)
	// Use page=1 and page_size=MaxExportLimit
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortDirection := c.DefaultQuery("sort_direction", "desc")

	// Create systems service
	systemsService := local.NewSystemsService()

	// Get systems without pagination limit (but with max export limit)
	systems, totalCount, err := systemsService.GetSystemsByOrganizationPaginated(
		userID, userOrgID, userOrgRole, 1, MaxExportLimit, search, sortBy, sortDirection,
		filterName, filterSystemKey, filterTypes, filterCreatedBy, filterVersions, filterOrgIDs, filterStatuses,
	)

	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Str("format", format).
			Msg("Failed to retrieve systems for export")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve systems for export", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Check if we hit the export limit
	if totalCount > MaxExportLimit {
		logger.Warn().
			Str("user_id", userID).
			Int("total_count", totalCount).
			Int("max_limit", MaxExportLimit).
			Msg("Export limit exceeded")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			fmt.Sprintf("too many systems to export (%d). maximum allowed: %d. please apply more filters", totalCount, MaxExportLimit),
			map[string]interface{}{
				"total_count": totalCount,
				"max_limit":   MaxExportLimit,
			},
		))
		return
	}

	// Build filters map for PDF metadata
	filters := buildFiltersMap(search, filterName, filterSystemKey, filterTypes, filterCreatedBy, filterVersions, filterOrgIDs, filterStatuses)

	// Create export service
	exportService := export.NewSystemsExportService()

	// Generate export based on format
	var fileBytes []byte
	var contentType string
	var filename string

	timestamp := time.Now().Format("2006-01-02_150405")

	switch format {
	case "csv":
		fileBytes, err = exportService.ExportToCSV(systems)
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
		filename = fmt.Sprintf("systems_export_%s.csv", timestamp)

	case "pdf":
		exportedBy := fmt.Sprintf("%s (%s)", user.Name, user.Email)
		fileBytes, err = exportService.ExportToPDF(systems, filters, exportedBy)
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
		filename = fmt.Sprintf("systems_export_%s.pdf", timestamp)
	}

	// Log the export operation
	logger.RequestLogger(c, "systems").Info().
		Str("operation", "export").
		Str("format", format).
		Int("system_count", len(systems)).
		Interface("filters", filters).
		Msg("Systems exported successfully")

	// Set response headers for file download
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(fileBytes)))

	// Send file
	c.Data(http.StatusOK, contentType, fileBytes)
}

// buildFiltersMap builds a map of applied filters for metadata
func buildFiltersMap(search, filterName, filterSystemKey string, filterTypes, filterCreatedBy, filterVersions, filterOrgIDs, filterStatuses []string) map[string]interface{} {
	filters := make(map[string]interface{})

	if search != "" {
		filters["search"] = search
	}
	if filterName != "" {
		filters["name"] = filterName
	}
	if filterSystemKey != "" {
		filters["system_key"] = filterSystemKey
	}
	if len(filterTypes) > 0 {
		filters["type"] = strings.Join(filterTypes, ", ")
	}
	if len(filterCreatedBy) > 0 {
		filters["created_by"] = strings.Join(filterCreatedBy, ", ")
	}
	if len(filterVersions) > 0 {
		filters["version"] = strings.Join(filterVersions, ", ")
	}
	if len(filterOrgIDs) > 0 {
		filters["organization_id"] = strings.Join(filterOrgIDs, ", ")
	}
	if len(filterStatuses) > 0 {
		filters["status"] = strings.Join(filterStatuses, ", ")
	}

	return filters
}
