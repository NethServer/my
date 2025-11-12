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
	MaxResellersExportLimit = 10000 // Maximum resellers to export (DoS protection)
)

// ExportResellers handles GET /api/resellers/export - exports resellers with applied filters
func ExportResellers(c *gin.Context) {
	// Get export format
	format := strings.ToLower(c.Query("format"))
	if format != "csv" && format != "pdf" {
		c.JSON(http.StatusBadRequest, response.BadRequest("format parameter required (csv or pdf)", nil))
		return
	}

	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Parse search parameter
	search := c.Query("search")

	// For export, we don't use pagination - get all matching resellers (with limit)
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortDirection := c.DefaultQuery("sort_direction", "desc")

	// Create service
	service := local.NewOrganizationService()

	// Get resellers based on RBAC without pagination limit (but with max export limit)
	userOrgRole := strings.ToLower(user.OrgRole)
	resellers, totalCount, err := service.ListResellers(userOrgRole, user.OrganizationID, 1, MaxResellersExportLimit, search, sortBy, sortDirection)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("format", format).
			Msg("Failed to retrieve resellers for export")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve resellers for export", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Check if we hit the export limit
	if totalCount > MaxResellersExportLimit {
		logger.Warn().
			Str("user_id", user.ID).
			Int("total_count", totalCount).
			Int("max_limit", MaxResellersExportLimit).
			Msg("Export limit exceeded")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			fmt.Sprintf("too many resellers to export (%d). maximum allowed: %d. please apply more filters", totalCount, MaxResellersExportLimit),
			map[string]interface{}{
				"total_count": totalCount,
				"max_limit":   MaxResellersExportLimit,
			},
		))
		return
	}

	// Build filters map for PDF metadata
	filters := make(map[string]interface{})
	if search != "" {
		filters["search"] = search
	}

	// Create export service
	exportService := export.NewResellersExportService()

	// Generate export based on format
	var fileBytes []byte
	var contentType string
	var filename string

	timestamp := time.Now().Format("2006-01-02_150405")

	switch format {
	case "csv":
		fileBytes, err = exportService.ExportToCSV(resellers)
		if err != nil {
			logger.Error().
				Err(err).
				Str("user_id", user.ID).
				Msg("Failed to generate CSV export")

			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to generate CSV export", map[string]interface{}{
				"error": err.Error(),
			}))
			return
		}
		contentType = "text/csv"
		filename = fmt.Sprintf("resellers_export_%s.csv", timestamp)

	case "pdf":
		exportedBy := fmt.Sprintf("%s (%s)", user.Name, user.Email)
		fileBytes, err = exportService.ExportToPDF(resellers, filters, exportedBy)
		if err != nil {
			logger.Error().
				Err(err).
				Str("user_id", user.ID).
				Msg("Failed to generate PDF export")

			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to generate PDF export", map[string]interface{}{
				"error": err.Error(),
			}))
			return
		}
		contentType = "application/pdf"
		filename = fmt.Sprintf("resellers_export_%s.pdf", timestamp)
	}

	// Log the export operation
	logger.RequestLogger(c, "resellers").Info().
		Str("operation", "export").
		Str("format", format).
		Int("reseller_count", len(resellers)).
		Interface("filters", filters).
		Msg("Resellers exported successfully")

	// Set response headers for file download
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(fileBytes)))

	// Send file
	c.Data(http.StatusOK, contentType, fileBytes)
}
