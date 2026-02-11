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
	MaxUsersExportLimit = 10000 // Maximum users to export (DoS protection)
)

// ExportUsers handles GET /api/users/export - exports users with applied filters
func ExportUsers(c *gin.Context) {
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

	// Parse filter parameters
	organizationFilter := c.QueryArray("organization_id")
	statuses := c.QueryArray("status")
	roleFilter := c.QueryArray("role")

	// For export, we don't use pagination - get all matching users (with limit)
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortDirection := c.DefaultQuery("sort_direction", "desc")

	// Create service
	service := local.NewUserService()

	// Get users based on RBAC (exclude current user) without pagination limit (but with max export limit)
	userOrgRole := strings.ToLower(user.OrgRole)
	users, totalCount, err := service.ListUsers(userOrgRole, user.OrganizationID, user.ID, 1, MaxUsersExportLimit, search, sortBy, sortDirection, organizationFilter, statuses, roleFilter)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", user.ID).
			Str("format", format).
			Msg("Failed to retrieve users for export")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve users for export", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Check if we hit the export limit
	if totalCount > MaxUsersExportLimit {
		logger.Warn().
			Str("user_id", user.ID).
			Int("total_count", totalCount).
			Int("max_limit", MaxUsersExportLimit).
			Msg("Export limit exceeded")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			fmt.Sprintf("too many users to export (%d). maximum allowed: %d. please apply more filters", totalCount, MaxUsersExportLimit),
			map[string]interface{}{
				"total_count": totalCount,
				"max_limit":   MaxUsersExportLimit,
			},
		))
		return
	}

	// Build filters map for PDF metadata
	filters := make(map[string]interface{})
	if search != "" {
		filters["search"] = search
	}
	if len(organizationFilter) > 0 {
		filters["organization_id"] = strings.Join(organizationFilter, ",")
	}
	if len(statuses) > 0 {
		filters["status"] = strings.Join(statuses, ",")
	}
	if len(roleFilter) > 0 {
		filters["role"] = strings.Join(roleFilter, ",")
	}

	// Create export service
	exportService := export.NewUsersExportService()

	// Generate export based on format
	var fileBytes []byte
	var contentType string
	var filename string

	timestamp := time.Now().Format("2006-01-02_150405")

	switch format {
	case "csv":
		fileBytes, err = exportService.ExportToCSV(users)
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
		filename = fmt.Sprintf("users_export_%s.csv", timestamp)

	case "pdf":
		exportedBy := fmt.Sprintf("%s (%s)", user.Name, user.Email)
		fileBytes, err = exportService.ExportToPDF(users, filters, exportedBy)
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
		filename = fmt.Sprintf("users_export_%s.pdf", timestamp)
	}

	// Log the export operation
	logger.RequestLogger(c, "users").Info().
		Str("operation", "export").
		Str("format", format).
		Int("user_count", len(users)).
		Interface("filters", filters).
		Msg("Users exported successfully")

	// Set response headers for file download
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(fileBytes)))

	// Send file
	c.Data(http.StatusOK, contentType, fileBytes)
}
