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

func intPtr(n int) *int { return &n }

func sampleGrants() []*models.EntitlementGrantReportRow {
	measured := time.Date(2026, 10, 28, 2, 10, 0, 0, time.UTC)
	over := &models.EntitlementGrantReportRow{
		SystemEntitlement: models.SystemEntitlement{
			SystemID:    "sys-1",
			Entitlement: "nsec-blacklist",
			SourceRef:   "wc-order-103812",
			Status:      "active",
			Variant:     map[string]interface{}{"label": "1-15 device"},
			Seats:       intPtr(15),
			Usage:       intPtr(130),
			UsageAt:     &measured,
		},
		SystemName:       "Novavetro_Firewall2",
		SystemKey:        "NETH-AAAA",
		OrganizationName: "Novavetro",
		ResellerName:     "MIT INFORMATICA",
	}
	unlimited := &models.EntitlementGrantReportRow{
		SystemEntitlement: models.SystemEntitlement{
			SystemID:    "sys-2",
			Entitlement: "crowdsec-advanced-threat-shield",
			Source:      "manual",
			Status:      "active",
			Usage:       intPtr(4),
		},
		SystemName:       "cluster",
		OrganizationName: "ACME",
		ResellerName:     "ACME",
	}
	return []*models.EntitlementGrantReportRow{over, unlimited}
}

func TestGrantsExportToCSV(t *testing.T) {
	svc := NewGrantsExportService(map[string]string{"nsec-blacklist": "Advanced Threat Shield"})
	out, err := svc.ExportToCSV(sampleGrants())
	require.NoError(t, err)

	records, err := csv.NewReader(bytes.NewReader(out)).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 3)

	header, row := records[0], records[1]
	col := func(name string) string {
		for i, h := range header {
			if h == name {
				return row[i]
			}
		}
		t.Fatalf("missing column %q", name)
		return ""
	}
	assert.Equal(t, "sys-1", col("System ID"))
	assert.Equal(t, "Novavetro_Firewall2", col("System"))
	assert.Equal(t, "MIT INFORMATICA", col("Reseller"))
	assert.Equal(t, "Advanced Threat Shield", col("Add-on"))
	assert.Equal(t, "130", col("Usage"))
	assert.Equal(t, "15", col("Limit"))
	assert.Equal(t, "115", col("Over the tier"))
	assert.Equal(t, "103812", col("Order"))

	// an add-on missing from the catalog keeps its id, no limit means no overrun
	assert.Equal(t, "crowdsec-advanced-threat-shield", records[2][5])
	assert.Equal(t, "", records[2][9])
}

func TestGrantUsage(t *testing.T) {
	rows := sampleGrants()
	assert.Equal(t, "130/15 (+115)", grantUsage(rows[0]))
	assert.Equal(t, "4", grantUsage(rows[1]))
	assert.Equal(t, "", grantUsage(&models.EntitlementGrantReportRow{}))
}

func TestGrantsExportToPDF(t *testing.T) {
	out, err := NewGrantsExportService(nil).ExportToPDF(sampleGrants(), "Add-ons over the tier", map[string]interface{}{"over the tier": "true"}, "Owner (owner@example.com)")
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(out, []byte("%PDF")))
}
