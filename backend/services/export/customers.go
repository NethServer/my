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

// CustomersExportService handles customers export operations
type CustomersExportService struct{}

// NewCustomersExportService creates a new customers export service
func NewCustomersExportService() *CustomersExportService {
	return &CustomersExportService{}
}

// ExportToCSV exports customers to CSV format
func (s *CustomersExportService) ExportToCSV(customers []*models.LocalCustomer) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{
		"Name",
		"Description",
		"Systems",
		"Custom Data",
		"Created At",
		"Updated At",
		"Logto Synced At",
	}

	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, customer := range customers {
		// Format custom data as JSON string
		customDataStr := ""
		if len(customer.CustomData) > 0 {
			if bytes, err := json.Marshal(customer.CustomData); err == nil {
				customDataStr = string(bytes)
			}
		}

		// Format dates
		createdAt := customer.CreatedAt.Format("2006-01-02 15:04:05 MST")
		updatedAt := customer.UpdatedAt.Format("2006-01-02 15:04:05 MST")
		logtoSyncedAt := ""
		if customer.LogtoSyncedAt != nil {
			logtoSyncedAt = customer.LogtoSyncedAt.Format("2006-01-02 15:04:05 MST")
		}

		row := []string{
			customer.Name,
			customer.Description,
			formatCount(customer.SystemsCount),
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

// ExportToPDF exports customers to PDF format
func (s *CustomersExportService) ExportToPDF(customers []*models.LocalCustomer, filters map[string]interface{}, exportedBy string) ([]byte, error) {
	rows := make([][]string, 0, len(customers))
	for _, org := range customers {
		rows = append(rows, organizationPDFRow(org.Name, org.Description, org.CustomData, org.CreatedBy, org.SuspendedAt, org.CreatedAt, org.SystemsCount))
	}

	return renderTablePDF(pdfReport{
		Title:      "Customers",
		Filters:    filters,
		ExportedBy: exportedBy,
		Columns:    customerPDFColumns,
		Rows:       rows,
		Empty:      "No customers found with the applied filters.",
	})
}
