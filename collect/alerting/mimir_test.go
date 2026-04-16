/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"testing"
	"time"

	"github.com/nethesis/my/collect/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSystemAlertContext(t *testing.T) {
	context := BuildSystemAlertContext(SystemAlertMetadata{
		SystemID:         "system-1",
		OrganizationID:   "org-1",
		SystemKey:        "SYS-001",
		SystemName:       "web-01",
		SystemFQDN:       "web-01.example.com",
		SystemIPv4:       "192.0.2.10",
		OrganizationName: "Acme Corp",
		OrganizationVAT:  "IT00000000001",
		OrganizationType: "customer",
	})

	require.NotNil(t, context)
	assert.Equal(t, "org-1", context.OrganizationID)
	assert.Equal(t, "system-1", context.SystemID)
	assert.Equal(t, "SYS-001", context.SystemKey)
	assert.Equal(t, map[string]string{
		"system_id":         "system-1",
		"system_key":        "SYS-001",
		"system_name":       "web-01",
		"system_fqdn":       "web-01.example.com",
		"system_ipv4":       "192.0.2.10",
		"organization_name": "Acme Corp",
		"organization_vat":  "IT00000000001",
		"organization_type": "customer",
	}, context.Labels)
}

func TestEnrichAlerts(t *testing.T) {
	startsAt := time.Unix(100, 0).UTC()
	endsAt := startsAt.Add(time.Hour)

	alerts, err := EnrichAlerts([]models.AlertmanagerPostAlert{
		{
			Labels: map[string]string{
				"alertname": "LinkFailed",
				"severity":  "critical",
			},
			Annotations: map[string]string{
				"summary": "Alert on {{.system_key}}",
			},
			StartsAt: startsAt,
			EndsAt:   endsAt,
		},
	}, BuildSystemAlertContext(SystemAlertMetadata{
		SystemID:       "system-1",
		OrganizationID: "org-1",
		SystemKey:      "SYS-001",
		SystemName:     "web-01",
	}))
	require.NoError(t, err)
	require.Len(t, alerts, 1)

	assert.Equal(t, "system-1", alerts[0].Labels["system_id"])
	assert.Equal(t, "SYS-001", alerts[0].Labels["system_key"])
	assert.Equal(t, "web-01", alerts[0].Labels["system_name"])
	assert.Equal(t, "Alert on SYS-001", alerts[0].Annotations["summary"])
	assert.Equal(t, startsAt, alerts[0].StartsAt)
	assert.Equal(t, endsAt, alerts[0].EndsAt)
}
