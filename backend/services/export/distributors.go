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
		"Resellers",
		"Customers",
		"Systems",
		"Legacy Systems",
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
			formatCount(distributor.ResellersCount),
			formatCount(distributor.CustomersCount),
			formatCount(distributor.SystemsCount),
			formatCount(distributor.LegacySystemsCount),
			customDataStr,
			createdAt,
			updatedAt,
			logtoSyncedAt,
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

// ExportToPDF exports distributors to PDF format
func (s *DistributorsExportService) ExportToPDF(distributors []*models.LocalDistributor, filters map[string]interface{}, exportedBy string) ([]byte, error) {
	rows := make([][]string, 0, len(distributors))
	for _, org := range distributors {
		rows = append(rows, organizationPDFRow(org.Name, org.Description, org.CustomData, org.CreatedBy, org.SuspendedAt, org.CreatedAt, org.ResellersCount, org.CustomersCount, org.SystemsCount))
	}

	return renderTablePDF(pdfReport{
		Title:      "Distributors",
		Filters:    filters,
		ExportedBy: exportedBy,
		Columns:    distributorPDFColumns,
		Rows:       rows,
		Empty:      "No distributors found with the applied filters.",
	})
}
