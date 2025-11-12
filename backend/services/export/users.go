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
	"strings"
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

// ExportToPDF exports users to PDF format
func (s *UsersExportService) ExportToPDF(users []*models.LocalUser, filters map[string]interface{}, exportedBy string) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageNumber().
		WithOrientation(orientation.Horizontal).
		Build()

	m := maroto.New(cfg)

	// Add header
	s.addPDFHeader(m, len(users), filters, exportedBy)

	// Add table with users data
	if len(users) > 0 {
		s.addPDFTable(m, users)
	} else {
		m.AddRow(20,
			col.New(12).Add(text.New("No users found with the applied filters.", props.Text{
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
func (s *UsersExportService) addPDFHeader(m core.Maroto, totalUsers int, filters map[string]interface{}, exportedBy string) {
	// Title
	m.AddRow(10,
		col.New(12).Add(text.New("Users Export Report", props.Text{
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
		col.New(6).Add(text.New("Total Users: "+strconv.Itoa(totalUsers), props.Text{
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

// addPDFTable adds users table to PDF with card-style layout
func (s *UsersExportService) addPDFTable(m core.Maroto, users []*models.LocalUser) {
	for i, user := range users {
		// Add spacing between users
		if i > 0 {
			m.AddRow(3)
		}

		// Build organization name
		orgName := "No Organization"
		if user.Organization != nil {
			orgName = user.Organization.Name
		}

		// Build roles string
		rolesStr := "No Roles"
		if len(user.Roles) > 0 {
			roleNames := make([]string, len(user.Roles))
			for j, role := range user.Roles {
				roleNames[j] = role.Name
			}
			rolesStr = strings.Join(roleNames, ", ")
		}

		// Main info row: Username (bold) | User Info | Created At
		m.AddRow(8,
			col.New(4).Add(text.New(user.Username, props.Text{
				Size:  9,
				Style: fontstyle.Bold,
			})),
			col.New(4).Add(text.New("User Info", props.Text{
				Size:  7,
				Style: fontstyle.Bold,
			})),
			col.New(4).Add(text.New("Created At", props.Text{
				Size:  7,
				Style: fontstyle.Bold,
			})),
		)

		// Second row: Name | Roles | Created date
		createdDate := user.CreatedAt.Format("2006-01-02 15:04")

		m.AddRow(6,
			col.New(4).Add(text.New("Name: "+user.Name, props.Text{Size: 6})),
			col.New(4).Add(text.New("Roles: "+truncate(rolesStr, 40), props.Text{Size: 6})),
			col.New(4).Add(text.New(createdDate, props.Text{Size: 6})),
		)

		// Third row: Email | Phone | Organization
		phoneText := "N/A"
		if user.Phone != nil && *user.Phone != "" {
			phoneText = *user.Phone
		}

		m.AddRow(6,
			col.New(4).Add(text.New("Email: "+truncate(user.Email, 40), props.Text{Size: 6})),
			col.New(4).Add(text.New("Phone: "+phoneText, props.Text{Size: 6})),
			col.New(4).Add(text.New("Org: "+truncate(orgName, 40), props.Text{Size: 6})),
		)

		// Separator line
		m.AddRow(1)
	}
}

// Helper functions

func safeStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
