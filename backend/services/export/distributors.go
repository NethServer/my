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

// DistributorsExportService handles distributors export operations
type DistributorsExportService struct{}

// NewDistributorsExportService creates a new distributors export service
func NewDistributorsExportService() *DistributorsExportService {
	return &DistributorsExportService{}
}

// ExportToCSV exports distributors to CSV format
func (s *DistributorsExportService) ExportToCSV(distributors []*models.LocalDistributor) ([]byte, error) {
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
	for _, distributor := range distributors {
		// Format custom data as JSON string
		customDataStr := ""
		if len(distributor.CustomData) > 0 {
			if bytes, err := json.Marshal(distributor.CustomData); err == nil {
				customDataStr = string(bytes)
			}
		}

		// Format dates
		createdAt := distributor.CreatedAt.Format("2006-01-02 15:04:05 MST")
		updatedAt := distributor.UpdatedAt.Format("2006-01-02 15:04:05 MST")
		logtoSyncedAt := ""
		if distributor.LogtoSyncedAt != nil {
			logtoSyncedAt = distributor.LogtoSyncedAt.Format("2006-01-02 15:04:05 MST")
		}

		row := []string{
			distributor.Name,
			distributor.Description,
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

// ExportToPDF exports distributors to PDF format
func (s *DistributorsExportService) ExportToPDF(distributors []*models.LocalDistributor, filters map[string]interface{}, exportedBy string) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageNumber().
		WithOrientation(orientation.Horizontal).
		Build()

	m := maroto.New(cfg)

	// Add header
	s.addPDFHeader(m, len(distributors), filters, exportedBy)

	// Add table with distributors data
	if len(distributors) > 0 {
		s.addPDFTable(m, distributors)
	} else {
		m.AddRow(20,
			col.New(12).Add(text.New("No distributors found with the applied filters.", props.Text{
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
func (s *DistributorsExportService) addPDFHeader(m core.Maroto, totalDistributors int, filters map[string]interface{}, exportedBy string) {
	// Title
	m.AddRow(10,
		col.New(12).Add(text.New("Distributors Export Report", props.Text{
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
		col.New(6).Add(text.New("Total Distributors: "+strconv.Itoa(totalDistributors), props.Text{
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

// addPDFTable adds distributors table to PDF with card-style layout
func (s *DistributorsExportService) addPDFTable(m core.Maroto, distributors []*models.LocalDistributor) {
	for i, distributor := range distributors {
		// Add spacing between distributors
		if i > 0 {
			m.AddRow(3)
		}

		// Main info row: Name (bold) | Organization Info | Created At
		m.AddRow(8,
			col.New(4).Add(text.New(distributor.Name, props.Text{
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
		descriptionText := distributor.Description
		if descriptionText == "" {
			descriptionText = "No description"
		}

		createdDate := distributor.CreatedAt.Format("2006-01-02 15:04")

		m.AddRow(6,
			col.New(4).Add(text.New("", props.Text{Size: 6})),
			col.New(4).Add(text.New("Description: "+truncate(descriptionText, 60), props.Text{Size: 6})),
			col.New(4).Add(text.New(createdDate, props.Text{Size: 6})),
		)

		// Third row: Empty | VAT | Empty
		vatText := "VAT: N/A"
		if distributor.CustomData != nil {
			if vat, ok := distributor.CustomData["vat"].(string); ok && vat != "" {
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
