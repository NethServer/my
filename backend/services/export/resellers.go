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
			formatCount(reseller.CustomersCount),
			formatCount(reseller.SystemsCount),
			formatCount(reseller.LegacySystemsCount),
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

// ExportToPDF exports resellers to PDF format
func (s *ResellersExportService) ExportToPDF(resellers []*models.LocalReseller, filters map[string]interface{}, exportedBy string) ([]byte, error) {
	rows := make([][]string, 0, len(resellers))
	for _, org := range resellers {
		rows = append(rows, organizationPDFRow(org.Name, org.Description, org.CustomData, org.CreatedBy, org.SuspendedAt, org.CreatedAt, org.CustomersCount, org.SystemsCount))
	}

	return renderTablePDF(pdfReport{
		Title:      "Resellers",
		Filters:    filters,
		ExportedBy: exportedBy,
		Columns:    resellerPDFColumns,
		Rows:       rows,
		Empty:      "No resellers found with the applied filters.",
	})
}
