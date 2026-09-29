/*
 * Copyright (C) 2025 Nethesis S.r.l.
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

	"github.com/nethesis/my/backend/models"
)

// SystemsExportService handles systems export operations
type SystemsExportService struct{}

// NewSystemsExportService creates a new systems export service
func NewSystemsExportService() *SystemsExportService {
	return &SystemsExportService{}
}

// ExportToCSV exports systems to CSV format
func (s *SystemsExportService) ExportToCSV(systems []*models.System) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{
		"Name",
		"Type",
		"Version",
		"Status",
		"FQDN",
		"IPv4 Address",
		"IPv6 Address",
		"System Key",
		"Notes",
		"Created At",
		"Organization",
		"Organization Type",
		"Created By",
		"Creator Email",
		"Creator Organization",
	}

	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, system := range systems {
		row := []string{
			system.Name,
			safeString(system.Type),
			system.Version,
			system.Status,
			system.FQDN,
			system.IPv4Address,
			system.IPv6Address,
			system.SystemKey,
			system.Notes,
			system.CreatedAt.Format("2006-01-02 15:04:05 MST"),
			system.Organization.Name,
			system.Organization.Type,
			system.CreatedBy.Name,
			system.CreatedBy.Email,
			system.CreatedBy.OrganizationName,
		}

		if err := writer.Write(csvSafeRow(row)); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}

// ExportToPDF exports systems to PDF format
func (s *SystemsExportService) ExportToPDF(systems []*models.System, filters map[string]interface{}, exportedBy string) ([]byte, error) {
	rows := make([][]string, 0, len(systems))
	for _, system := range systems {
		rows = append(rows, []string{
			system.Name,
			safeString(system.Type),
			system.Version,
			titleCase(system.Status),
			systemAddress(system),
			system.Organization.Name,
			system.CreatedBy.Name,
			formatDate(system.CreatedAt),
		})
	}

	return renderTablePDF(pdfReport{
		Title:      "Systems",
		Filters:    filters,
		ExportedBy: exportedBy,
		Columns: []pdfColumn{
			{Title: "Name", Width: 2},
			{Title: "Type", Width: 1},
			{Title: "Version", Width: 1},
			{Title: "Status", Width: 1, Status: true},
			{Title: "FQDN / IP", Width: 2},
			{Title: "Organization", Width: 2},
			{Title: "Created by", Width: 2},
			{Title: "Created", Width: 1},
		},
		Rows:  rows,
		Empty: "No systems found with the applied filters.",
	})
}

// systemAddress is the most readable network identity of a system
func systemAddress(system *models.System) string {
	if system.FQDN != "" {
		return system.FQDN
	}
	if system.IPv4Address != "" {
		return system.IPv4Address
	}
	return system.IPv6Address
}

// Helper functions

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
