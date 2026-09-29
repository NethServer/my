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
	"time"

	"github.com/nethesis/my/backend/models"
)

const timestampLayout = "2006-01-02 15:04:05 MST"

// ApplicationsExportService handles applications export operations
type ApplicationsExportService struct{}

// NewApplicationsExportService creates a new applications export service
func NewApplicationsExportService() *ApplicationsExportService {
	return &ApplicationsExportService{}
}

// ExportToCSV exports applications to CSV format
func (s *ApplicationsExportService) ExportToCSV(apps []*models.Application) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	header := []string{
		"Name",
		"Module ID",
		"Type",
		"Type ID",
		"Version",
		"Status",
		"Node",
		"URL",
		"Notes",
		"Has Errors",
		"System",
		"Organization",
		"Organization Type",
		"First Seen At",
		"Last Inventory At",
		"Created At",
	}

	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, app := range apps {
		orgName, orgType := applicationOrganization(app)
		row := []string{
			app.GetEffectiveDisplayName(),
			app.ModuleID,
			applicationTypeName(app),
			app.InstanceOf,
			safeString(app.Version),
			app.Status,
			applicationNode(app),
			safeString(app.URL),
			safeString(app.Notes),
			strconv.FormatBool(app.HasServiceErrors()),
			applicationSystemName(app),
			orgName,
			orgType,
			app.FirstSeenAt.Format(timestampLayout),
			formatOptionalTime(app.LastInventoryAt, timestampLayout),
			app.CreatedAt.Format(timestampLayout),
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

// ExportToPDF exports applications to PDF format
func (s *ApplicationsExportService) ExportToPDF(apps []*models.Application, filters map[string]interface{}, exportedBy string) ([]byte, error) {
	rows := make([][]string, 0, len(apps))
	for _, app := range apps {
		orgName, _ := applicationOrganization(app)
		rows = append(rows, []string{
			app.GetEffectiveDisplayName(),
			applicationTypeName(app),
			safeString(app.Version),
			titleCase(app.Status),
			applicationNode(app),
			applicationSystemName(app),
			orgName,
			formatOptionalDate(app.LastInventoryAt),
			formatDate(app.CreatedAt),
		})
	}

	return renderTablePDF(pdfReport{
		Title:      "Applications",
		Filters:    filters,
		ExportedBy: exportedBy,
		Columns: []pdfColumn{
			{Title: "Name", Width: 2},
			{Title: "Type", Width: 1},
			{Title: "Version", Width: 1},
			{Title: "Status", Width: 1, Status: true},
			{Title: "Node", Width: 1},
			{Title: "System", Width: 2},
			{Title: "Organization", Width: 2},
			{Title: "Last inventory", Width: 1},
			{Title: "Created", Width: 1},
		},
		Rows:  rows,
		Empty: "No applications found with the applied filters.",
	})
}

// applicationTypeName is the catalog name of the application type, falling back to its id
func applicationTypeName(app *models.Application) string {
	if name := safeString(app.Name); name != "" {
		return name
	}
	return app.InstanceOf
}

func applicationNode(app *models.Application) string {
	if label := safeString(app.NodeLabel); label != "" {
		return label
	}
	if app.NodeID != nil {
		return strconv.Itoa(*app.NodeID)
	}
	return ""
}

func applicationSystemName(app *models.Application) string {
	if app.System != nil && app.System.Name != "" {
		return app.System.Name
	}
	return app.SystemID
}

func applicationOrganization(app *models.Application) (name, orgType string) {
	if app.Organization == nil {
		return "", ""
	}
	return app.Organization.Name, app.Organization.Type
}

func formatOptionalTime(t *time.Time, layout string) string {
	if t == nil {
		return ""
	}
	return t.Format(layout)
}
