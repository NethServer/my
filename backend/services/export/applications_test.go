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
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/backend/models"
)

func strPtr(s string) *string { return &s }

func sampleApplications() []*models.Application {
	created := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	inventory := time.Date(2026, 9, 29, 8, 30, 0, 0, time.UTC)
	nodeID := 1

	assigned := &models.Application{
		ID:              "app-1",
		SystemID:        "sys-1",
		ModuleID:        "mail1",
		InstanceOf:      "mail",
		Name:            strPtr("Mail"),
		DisplayName:     strPtr("Company mail"),
		NodeID:          &nodeID,
		NodeLabel:       strPtr("node-1"),
		Version:         strPtr("1.2.3"),
		Status:          "assigned",
		URL:             strPtr("https://mail.example.com"),
		Notes:           strPtr("=cmd, with comma"),
		ServicesData:    json.RawMessage(`{"services":[],"has_errors":true,"error_count":1}`),
		CreatedAt:       created,
		FirstSeenAt:     created,
		LastInventoryAt: &inventory,
		System:          &models.SystemSummary{ID: "sys-1", Name: "ns8-hq"},
		Organization:    &models.OrganizationSummary{ID: "org-1", LogtoID: "l1", Name: "ACME", Type: "customer"},
	}

	unassigned := &models.Application{
		ID:          "app-2",
		SystemID:    "sys-2",
		ModuleID:    "webtop2",
		InstanceOf:  "webtop",
		Status:      "unassigned",
		CreatedAt:   created,
		FirstSeenAt: created,
		System:      &models.SystemSummary{ID: "sys-2", Name: "ns8-branch"},
	}

	return []*models.Application{assigned, unassigned}
}

func TestApplicationsExportToCSV(t *testing.T) {
	out, err := NewApplicationsExportService().ExportToCSV(sampleApplications())
	require.NoError(t, err)

	rows, err := csv.NewReader(bytes.NewReader(out)).ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 3)

	assert.Equal(t, []string{
		"Name", "Module ID", "Type", "Type ID", "Version", "Status", "Node", "URL", "Notes",
		"Has Errors", "System", "Organization", "Organization Type",
		"First Seen At", "Last Inventory At", "Created At",
	}, rows[0])

	assert.Equal(t, []string{
		"Company mail", "mail1", "Mail", "mail", "1.2.3", "assigned", "node-1",
		"https://mail.example.com", "'=cmd, with comma", "true", "ns8-hq", "ACME", "customer",
		"2026-09-01 10:00:00 UTC", "2026-09-29 08:30:00 UTC", "2026-09-01 10:00:00 UTC",
	}, rows[1])

	// display name falls back to the module id, type to its id, and every
	// optional field stays empty rather than rendering "<nil>"
	assert.Equal(t, []string{
		"webtop2", "webtop2", "webtop", "webtop", "", "unassigned", "", "", "", "false",
		"ns8-branch", "", "", "2026-09-01 10:00:00 UTC", "", "2026-09-01 10:00:00 UTC",
	}, rows[2])
}

func TestApplicationsExportToPDF(t *testing.T) {
	svc := NewApplicationsExportService()

	out, err := svc.ExportToPDF(sampleApplications(), map[string]interface{}{"type": "mail"}, "Owner (owner@example.com)")
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(out, []byte("%PDF")), "output must be a PDF document")

	empty, err := svc.ExportToPDF(nil, nil, "")
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(empty, []byte("%PDF")))
}
