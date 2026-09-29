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
	"strings"

	"github.com/nethesis/my/backend/models"
)

// UsersExportService handles users export operations
type UsersExportService struct{}

// NewUsersExportService creates a new users export service
func NewUsersExportService() *UsersExportService {
	return &UsersExportService{}
}

// ExportToCSV exports users to CSV format
func (s *UsersExportService) ExportToCSV(users []*models.LocalUser) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{
		"Username",
		"Email",
		"Name",
		"Phone",
		"Organization",
		"User Roles",
		"Status",
		"Created At",
		"Latest Login At",
		"Suspended At",
	}

	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, user := range users {
		// Build roles string
		var rolesStr string
		if len(user.Roles) > 0 {
			roleNames := make([]string, len(user.Roles))
			for i, role := range user.Roles {
				roleNames[i] = role.Name
			}
			rolesStr = strings.Join(roleNames, ", ")
		}

		// Determine status
		status := "active"
		if user.DeletedAt != nil {
			status = "deleted"
		} else if user.SuspendedAt != nil {
			status = "suspended"
		}

		// Build organization name
		orgName := ""
		if user.Organization != nil {
			orgName = user.Organization.Name
		}

		// Format dates
		createdAt := user.CreatedAt.Format("2006-01-02 15:04:05 MST")
		latestLoginAt := ""
		if user.LatestLoginAt != nil {
			latestLoginAt = user.LatestLoginAt.Format("2006-01-02 15:04:05 MST")
		}
		suspendedAt := ""
		if user.SuspendedAt != nil {
			suspendedAt = user.SuspendedAt.Format("2006-01-02 15:04:05 MST")
		}

		row := []string{
			user.Username,
			user.Email,
			user.Name,
			safeStringPtr(user.Phone),
			orgName,
			rolesStr,
			status,
			createdAt,
			latestLoginAt,
			suspendedAt,
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

// ExportToPDF exports users to PDF format
func (s *UsersExportService) ExportToPDF(users []*models.LocalUser, filters map[string]interface{}, exportedBy string) ([]byte, error) {
	rows := make([][]string, 0, len(users))
	for _, user := range users {
		orgName := ""
		if user.Organization != nil {
			orgName = user.Organization.Name
		}

		roleNames := make([]string, 0, len(user.Roles))
		for _, role := range user.Roles {
			roleNames = append(roleNames, role.Name)
		}

		status := "Active"
		if user.DeletedAt != nil {
			status = "Deleted"
		} else if user.SuspendedAt != nil {
			status = "Suspended"
		}

		rows = append(rows, []string{
			user.Name,
			user.Email,
			orgName,
			strings.Join(roleNames, ", "),
			status,
			formatDate(user.CreatedAt),
			formatOptionalDate(user.LatestLoginAt),
		})
	}

	return renderTablePDF(pdfReport{
		Title:      "Users",
		Filters:    filters,
		ExportedBy: exportedBy,
		Columns: []pdfColumn{
			{Title: "Name", Width: 2},
			{Title: "Email", Width: 3},
			{Title: "Organization", Width: 2},
			{Title: "Roles", Width: 2},
			{Title: "Status", Width: 1, Status: true},
			{Title: "Created", Width: 1},
			{Title: "Last login", Width: 1},
		},
		Rows:  rows,
		Empty: "No users found with the applied filters.",
	})
}

// Helper functions

func safeStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
