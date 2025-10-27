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
	"strconv"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontfamily"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/orientation"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

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

		if err := writer.Write(row); err != nil {
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
	cfg := config.NewBuilder().
		WithPageNumber().
		WithOrientation(orientation.Horizontal).
		Build()

	m := maroto.New(cfg)

	// Add header
	s.addPDFHeader(m, len(systems), filters, exportedBy)

	// Add table with systems data
	if len(systems) > 0 {
		s.addPDFTable(m, systems)
	} else {
		m.AddRow(20,
			col.New(12).Add(text.New("No systems found with the applied filters.", props.Text{
				Align: align.Center,
				Size:  12,
			})),
		)
	}

	// Generate PDF
	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return doc.GetBytes(), nil
}

// addPDFHeader adds header section to PDF
func (s *SystemsExportService) addPDFHeader(m core.Maroto, totalSystems int, filters map[string]interface{}, exportedBy string) {
	// Title
	m.AddRow(10,
		col.New(12).Add(text.New("Systems Export Report", props.Text{
			Size:   16,
			Family: fontfamily.Helvetica,
			Align:  align.Center,
			Style:  fontstyle.Bold,
		})),
	)

	// Metadata
	m.AddRow(6,
		col.New(6).Add(text.New("Generated: "+time.Now().Format("2006-01-02 15:04:05 MST"), props.Text{
			Size: 8,
		})),
		col.New(6).Add(text.New("Total Systems: "+strconv.Itoa(totalSystems), props.Text{
			Size:  8,
			Align: align.Right,
		})),
	)

	// Filters applied
	if len(filters) > 0 {
		filtersText := "Filters Applied: " + formatFilters(filters)
		m.AddRow(6,
			col.New(12).Add(text.New(filtersText, props.Text{
				Size: 8,
			})),
		)
	}

	// Exported by
	if exportedBy != "" {
		m.AddRow(6,
			col.New(12).Add(text.New("Exported by: "+exportedBy, props.Text{
				Size: 8,
			})),
		)
	}

	// Spacing
	m.AddRow(5)
}

// addPDFTable adds systems table to PDF with card-style layout
func (s *SystemsExportService) addPDFTable(m core.Maroto, systems []*models.System) {
	for i, system := range systems {
		// Add spacing between systems
		if i > 0 {
			m.AddRow(3)
		}

		// Main info row: Name (bold) on first column
		m.AddRow(8,
			col.New(4).Add(text.New(system.Name, props.Text{
				Size:  9,
				Style: fontstyle.Bold,
			})),
			col.New(4).Add(text.New("System Info", props.Text{
				Size:  7,
				Style: fontstyle.Bold,
			})),
			col.New(4).Add(text.New("Created At", props.Text{
				Size:  7,
				Style: fontstyle.Bold,
			})),
		)

		// Second row: Type, Version, Status | System Key | Created date
		typeVersionStatus := fmt.Sprintf("Type: %s | Version: %s | Status: %s",
			safeString(system.Type), system.Version, system.Status)
		systemKeyText := fmt.Sprintf("Key: %s", system.SystemKey)
		createdDate := system.CreatedAt.Format("2006-01-02 15:04")

		m.AddRow(6,
			col.New(4).Add(text.New(typeVersionStatus, props.Text{Size: 6})),
			col.New(4).Add(text.New(systemKeyText, props.Text{Size: 6})),
			col.New(4).Add(text.New(createdDate, props.Text{Size: 6})),
		)

		// Third row: Network info | Organization | Created By
		orgText := fmt.Sprintf("Org: %s (%s)", system.Organization.Name, system.Organization.Type)
		networkInfo := ""
		if system.FQDN != "" {
			networkInfo = "FQDN: " + system.FQDN
		}
		if system.IPv4Address != "" {
			if networkInfo != "" {
				networkInfo += " | "
			}
			networkInfo += "IPv4: " + system.IPv4Address
		}
		if system.IPv6Address != "" {
			if networkInfo != "" {
				networkInfo += " | "
			}
			networkInfo += "IPv6: " + truncate(system.IPv6Address, 25)
		}
		if networkInfo == "" {
			networkInfo = "No network info"
		}

		createdByText := fmt.Sprintf("By: %s (%s)", system.CreatedBy.Name, system.CreatedBy.Email)

		m.AddRow(6,
			col.New(4).Add(text.New(networkInfo, props.Text{Size: 6})),
			col.New(4).Add(text.New(orgText, props.Text{Size: 6})),
			col.New(4).Add(text.New(createdByText, props.Text{Size: 6})),
		)

		// Notes row (if present)
		if system.Notes != "" {
			m.AddRow(6,
				col.New(12).Add(text.New("Notes: "+truncate(system.Notes, 120), props.Text{
					Size:  6,
					Style: fontstyle.Italic,
				})),
			)
		}

		// Separator line
		m.AddRow(1)
	}
}

// Helper functions

func formatFilters(filters map[string]interface{}) string {
	if len(filters) == 0 {
		return "None"
	}

	result := ""
	for key, value := range filters {
		if result != "" {
			result += ", "
		}
		result += fmt.Sprintf("%s=%v", key, value)
	}
	return result
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
