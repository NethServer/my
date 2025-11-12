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
	"encoding/json"
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

// ResellersExportService handles resellers export operations
type ResellersExportService struct{}

// NewResellersExportService creates a new resellers export service
func NewResellersExportService() *ResellersExportService {
	return &ResellersExportService{}
}

// ExportToCSV exports resellers to CSV format
func (s *ResellersExportService) ExportToCSV(resellers []*models.LocalReseller) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{
		"Name",
		"Description",
		"Custom Data",
		"Created At",
		"Updated At",
		"Logto Synced At",
	}

	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, reseller := range resellers {
		// Format custom data as JSON string
		customDataStr := ""
		if len(reseller.CustomData) > 0 {
			if bytes, err := json.Marshal(reseller.CustomData); err == nil {
				customDataStr = string(bytes)
			}
		}

		// Format dates
		createdAt := reseller.CreatedAt.Format("2006-01-02 15:04:05 MST")
		updatedAt := reseller.UpdatedAt.Format("2006-01-02 15:04:05 MST")
		logtoSyncedAt := ""
		if reseller.LogtoSyncedAt != nil {
			logtoSyncedAt = reseller.LogtoSyncedAt.Format("2006-01-02 15:04:05 MST")
		}

		row := []string{
			reseller.Name,
			reseller.Description,
			customDataStr,
			createdAt,
			updatedAt,
			logtoSyncedAt,
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

// ExportToPDF exports resellers to PDF format
func (s *ResellersExportService) ExportToPDF(resellers []*models.LocalReseller, filters map[string]interface{}, exportedBy string) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageNumber().
		WithOrientation(orientation.Horizontal).
		Build()

	m := maroto.New(cfg)

	// Add header
	s.addPDFHeader(m, len(resellers), filters, exportedBy)

	// Add table with resellers data
	if len(resellers) > 0 {
		s.addPDFTable(m, resellers)
	} else {
		m.AddRow(20,
			col.New(12).Add(text.New("No resellers found with the applied filters.", props.Text{
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
func (s *ResellersExportService) addPDFHeader(m core.Maroto, totalResellers int, filters map[string]interface{}, exportedBy string) {
	// Title
	m.AddRow(10,
		col.New(12).Add(text.New("Resellers Export Report", props.Text{
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
		col.New(6).Add(text.New("Total Resellers: "+strconv.Itoa(totalResellers), props.Text{
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

// addPDFTable adds resellers table to PDF with card-style layout
func (s *ResellersExportService) addPDFTable(m core.Maroto, resellers []*models.LocalReseller) {
	for i, reseller := range resellers {
		// Add spacing between resellers
		if i > 0 {
			m.AddRow(3)
		}

		// Main info row: Name (bold) | Organization Info | Created At
		m.AddRow(8,
			col.New(4).Add(text.New(reseller.Name, props.Text{
				Size:  9,
				Style: fontstyle.Bold,
			})),
			col.New(4).Add(text.New("Organization Info", props.Text{
				Size:  7,
				Style: fontstyle.Bold,
			})),
			col.New(4).Add(text.New("Created At", props.Text{
				Size:  7,
				Style: fontstyle.Bold,
			})),
		)

		// Second row: Empty | Description | Created date
		descriptionText := reseller.Description
		if descriptionText == "" {
			descriptionText = "No description"
		}

		createdDate := reseller.CreatedAt.Format("2006-01-02 15:04")

		m.AddRow(6,
			col.New(4).Add(text.New("", props.Text{Size: 6})),
			col.New(4).Add(text.New("Description: "+truncate(descriptionText, 60), props.Text{Size: 6})),
			col.New(4).Add(text.New(createdDate, props.Text{Size: 6})),
		)

		// Third row: Empty | VAT | Empty
		vatText := "VAT: N/A"
		if reseller.CustomData != nil {
			if vat, ok := reseller.CustomData["vat"].(string); ok && vat != "" {
				vatText = "VAT: " + vat
			}
		}

		m.AddRow(6,
			col.New(4).Add(text.New("", props.Text{Size: 6})),
			col.New(4).Add(text.New(vatText, props.Text{Size: 6})),
			col.New(4).Add(text.New("", props.Text{Size: 6})),
		)

		// Separator line
		m.AddRow(1)
	}
}
