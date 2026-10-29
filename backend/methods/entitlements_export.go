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

	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/export"
)

const (
	MaxGrantsExportLimit = 10000 // Maximum grants to export (DoS protection)
)

// ExportEntitlementGrants handles GET /api/entitlements/grants/export — the
// grants report with the same filters as /entitlements/grants, as CSV or PDF.
// With over_limit=true it is the list of add-ons past their tier that
// Nethesis sales follow up.
func ExportEntitlementGrants(c *gin.Context) {
	format := strings.ToLower(c.Query("format"))
	if format != "csv" && format != "pdf" {
		c.JSON(http.StatusBadRequest, response.BadRequest("format parameter required (csv or pdf)", nil))
		return
	}

	u, scope, filter, ok := grantsFilter(c)
	if !ok {
		return
	}

	repo := entities.NewLocalSystemEntitlementRepository()
	rows, total, err := repo.ListGrants(filter, MaxGrantsExportLimit, 0)
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to list grants for export")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve grants for export", nil))
		return
	}
	if total > MaxGrantsExportLimit {
		c.JSON(http.StatusBadRequest, response.BadRequest(
			fmt.Sprintf("too many grants to export (%d). maximum allowed: %d. please apply more filters", total, MaxGrantsExportLimit),
			map[string]interface{}{"total_count": total, "max_limit": MaxGrantsExportLimit},
		))
		return
	}
	for _, row := range rows {
		redactPurchaser(&row.SystemEntitlement, scope)
		redactUsage(&row.SystemEntitlement, u)
	}

	addonNames := map[string]string{}
	if items, err := entities.NewLocalEntitlementCatalogRepository().List(); err == nil {
		for _, item := range items {
			addonNames[item.ID] = item.DisplayName
		}
	}

	filters := grantsExportFilters(filter)
	exportService := export.NewGrantsExportService(addonNames)
	timestamp := time.Now().Format("2006-01-02_150405")
	name := "addons"
	title := "Add-ons"
	if filter.OverLimit {
		name = "addons_over_tier"
		title = "Add-ons over the tier"
	}

	var fileBytes []byte
	var contentType, filename string
	switch format {
	case "csv":
		fileBytes, err = exportService.ExportToCSV(rows)
		contentType = "text/csv"
		filename = fmt.Sprintf("%s_export_%s.csv", name, timestamp)
	case "pdf":
		fileBytes, err = exportService.ExportToPDF(rows, title, filters, fmt.Sprintf("%s (%s)", u.Name, u.Email))
		contentType = "application/pdf"
		filename = fmt.Sprintf("%s_export_%s.pdf", name, timestamp)
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Str("format", format).Msg("Failed to generate grants export")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to generate "+strings.ToUpper(format)+" export", nil))
		return
	}

	logger.RequestLogger(c, "entitlements").Info().
		Str("operation", "export").
		Str("format", format).
		Int("grant_count", len(rows)).
		Interface("filters", filters).
		Msg("Grants exported successfully")

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(fileBytes)))
	c.Data(http.StatusOK, contentType, fileBytes)
}

// grantsExportFilters collects the applied filters for the PDF metadata and the log line
func grantsExportFilters(f entities.GrantsReportFilter) map[string]interface{} {
	filters := make(map[string]interface{})
	if f.Search != "" {
		filters["search"] = f.Search
	}
	if f.Entitlement != "" {
		filters["add-on"] = f.Entitlement
	}
	if f.OrganizationID != "" {
		filters["organization_id"] = f.OrganizationID
	}
	if f.Source != "" {
		filters["source"] = f.Source
	}
	if f.ActiveOnly {
		filters["active"] = "true"
	}
	if f.OverLimit {
		filters["over the tier"] = "true"
	}
	if f.ExpiringBefore != nil {
		filters["expiring before"] = f.ExpiringBefore.Format("2006-01-02")
	}
	return filters
}
