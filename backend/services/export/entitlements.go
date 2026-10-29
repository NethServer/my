/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package export

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"github.com/nethesis/my/backend/models"
)

// GrantsExportService exports the add-on grants report
type GrantsExportService struct {
	// addonNames maps a catalog id to its display name
	addonNames map[string]string
}

// NewGrantsExportService creates a grants export service naming the add-ons
// after the catalog
func NewGrantsExportService(addonNames map[string]string) *GrantsExportService {
	return &GrantsExportService{addonNames: addonNames}
}

// ExportToCSV exports the grants to CSV format
func (s *GrantsExportService) ExportToCSV(rows []*models.EntitlementGrantReportRow) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	header := []string{
		"System ID",
		"System key",
		"System",
		"Organization",
		"Reseller",
		"Add-on",
		"Tier",
		"Usage",
		"Limit",
		"Over the tier",
		"Measured at",
		"Order",
		"Status",
		"Valid until",
	}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, row := range rows {
		record := []string{
			row.SystemID,
			row.SystemKey,
			row.SystemName,
			row.OrganizationName,
			row.ResellerName,
			s.addonName(row.Entitlement),
			grantTier(row),
			formatCount(row.Usage),
			formatCount(row.Seats),
			grantOverLimit(row),
			formatOptionalTime(row.UsageAt, timestampLayout),
			grantOrder(row),
			row.Status,
			formatOptionalDate(row.ValidUntil),
		}
		if err := writer.Write(csvSafeRow(record)); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}
	return buf.Bytes(), nil
}

// ExportToPDF exports the grants to PDF format
func (s *GrantsExportService) ExportToPDF(rows []*models.EntitlementGrantReportRow, title string, filters map[string]interface{}, exportedBy string) ([]byte, error) {
	out := make([][]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, []string{
			row.SystemName,
			row.SystemKey,
			row.ResellerName,
			row.OrganizationName,
			s.addonName(row.Entitlement),
			grantTier(row),
			grantUsage(row),
			grantOrder(row),
		})
	}

	return renderTablePDF(pdfReport{
		Title:      title,
		Filters:    filters,
		ExportedBy: exportedBy,
		// 12-unit grid: the system key is the widest value and the one the
		// partners look systems up by; the measure date stays in the CSV
		Columns: []pdfColumn{
			{Title: "System", Width: 2},
			{Title: "System ID", Width: 3},
			{Title: "Reseller", Width: 2},
			{Title: "Organization", Width: 1},
			{Title: "Add-on", Width: 1},
			{Title: "Tier", Width: 1},
			{Title: "Usage", Width: 1, Align: alignRight},
			{Title: "Order", Width: 1},
		},
		Rows:  out,
		Empty: "No add-ons found with the applied filters.",
	})
}

func (s *GrantsExportService) addonName(id string) string {
	if name := s.addonNames[id]; name != "" {
		return name
	}
	return id
}

// grantTier is the shop tier label; manual and untiered grants have none
func grantTier(row *models.EntitlementGrantReportRow) string {
	if label, _ := row.Variant["label"].(string); label != "" {
		return label
	}
	return ""
}

// grantUsage reads "156/30 (+126)", the measured usage against the limit
func grantUsage(row *models.EntitlementGrantReportRow) string {
	if row.Usage == nil {
		return ""
	}
	if row.Seats == nil {
		return strconv.Itoa(*row.Usage)
	}
	usage := fmt.Sprintf("%d/%d", *row.Usage, *row.Seats)
	if over := grantOverLimit(row); over != "" {
		usage += " (+" + over + ")"
	}
	return usage
}

func grantOverLimit(row *models.EntitlementGrantReportRow) string {
	if row.Usage == nil || row.Seats == nil || *row.Usage <= *row.Seats {
		return ""
	}
	return strconv.Itoa(*row.Usage - *row.Seats)
}

// grantOrder is the shop order number of the grant
func grantOrder(row *models.EntitlementGrantReportRow) string {
	return strings.TrimPrefix(row.SourceRef, "wc-order-")
}
