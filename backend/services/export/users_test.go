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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/backend/models"
)

func sampleUsers() []*models.LocalUser {
	created := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	login := time.Date(2026, 10, 7, 16, 45, 0, 0, time.UTC)

	return []*models.LocalUser{
		{
			Username:      "alice",
			Email:         "alice@example.com",
			Name:          "Alice",
			CreatedAt:     created,
			LatestLoginAt: &login,
			Organization:  &models.UserOrganization{Name: "ACME"},
			Roles:         []models.UserRole{{Name: "Admin"}},
		},
		{
			Username:  "bob",
			Email:     "bob@example.com",
			Name:      "Bob",
			CreatedAt: created,
		},
	}
}

// TestUsersExportToCSV_NeverLoggedIn pins the last-login column: a date for an
// account that signed in, "never" for one that did not, mirroring the "no
// login" badge of the users table.
func TestUsersExportToCSV_NeverLoggedIn(t *testing.T) {
	out, err := NewUsersExportService().ExportToCSV(sampleUsers())
	require.NoError(t, err)

	records, err := csv.NewReader(bytes.NewReader(out)).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 3)

	header := records[0]
	col := -1
	for i, h := range header {
		if h == "Latest Login At" {
			col = i
		}
	}
	require.NotEqual(t, -1, col, "Latest Login At column missing")

	assert.Equal(t, "2026-10-07 16:45:00 UTC", records[1][col])
	assert.Equal(t, "never", records[2][col])
}

// TestUsersExportToPDF_NeverLoggedIn checks the PDF renders with a mix of
// accounts that did and did not sign in.
func TestUsersExportToPDF_NeverLoggedIn(t *testing.T) {
	out, err := NewUsersExportService().ExportToPDF(sampleUsers(), map[string]interface{}{"status": "no_login"}, "Owner (owner@example.com)")
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(out, []byte("%PDF-")), "output is not a PDF")
}
