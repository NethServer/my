/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"encoding/json"
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
		SystemType:       "ns8",
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
		"system_type":       "ns8",
		"system_fqdn":       "web-01.example.com",
		"system_ipv4":       "192.0.2.10",
		"organization_id":   "org-1",
		"organization_name": "Acme Corp",
		"organization_vat":  "IT00000000001",
		"organization_type": "customer",
	}, context.Labels)
}

func TestBuildSystemAlertContext_EmptyFields(t *testing.T) {
	ctx := BuildSystemAlertContext(SystemAlertMetadata{
		SystemID:       "system-1",
		OrganizationID: "org-1",
		SystemKey:      "SYS-001",
		// All other fields empty
	})

	require.NotNil(t, ctx)
	// All identity labels must be present so InjectLabels can strip
	// client-supplied spoofed values for keys the server has no value for.
	assert.Contains(t, ctx.Labels, "system_type")
	assert.Contains(t, ctx.Labels, "system_fqdn")
	assert.Contains(t, ctx.Labels, "organization_name")
	assert.Equal(t, "", ctx.Labels["system_type"])
	assert.Equal(t, "", ctx.Labels["system_fqdn"])
	assert.Equal(t, "", ctx.Labels["organization_name"])
}

func TestInjectLabels_EmptyValueStripsClientLabel(t *testing.T) {
	body := []byte(`[{"labels":{"alertname":"Test","organization_name":"FAKE","system_fqdn":"evil.local"}}]`)
	result := InjectLabels(body, map[string]string{
		"system_id":         "uuid-1",
		"system_key":        "SYS-1",
		"organization_name": "",
		"system_fqdn":       "",
	})

	var alerts []map[string]interface{}
	err := json.Unmarshal(result, &alerts)
	require.NoError(t, err)
	labels := alerts[0]["labels"].(map[string]interface{})

	_, hasOrgName := labels["organization_name"]
	_, hasFQDN := labels["system_fqdn"]
	assert.False(t, hasOrgName, "organization_name must be stripped when server has no value")
	assert.False(t, hasFQDN, "system_fqdn must be stripped when server has no value")
	assert.Equal(t, "uuid-1", labels["system_id"])
	assert.Equal(t, "SYS-1", labels["system_key"])
}

func TestInjectLabels_OverwritesClientLabels(t *testing.T) {
	body := []byte(`[{"labels":{"alertname":"Test","system_key":"SPOOFED","system_name":"FAKE"}}]`)
	result := InjectLabels(body, map[string]string{
		"system_key":  "REAL-KEY",
		"system_name": "real-host",
	})

	var alerts []map[string]interface{}
	err := json.Unmarshal(result, &alerts)
	require.NoError(t, err)
	labels := alerts[0]["labels"].(map[string]interface{})

	assert.Equal(t, "REAL-KEY", labels["system_key"])
	assert.Equal(t, "real-host", labels["system_name"])
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

func TestBuildResolvedLinkFailedAlert(t *testing.T) {
	// SystemFQDN/IPv4/VAT left empty on purpose: EnrichAlerts must strip them so
	// the resolve fingerprint matches the firing alert (which also strips empties).
	systemContext := BuildSystemAlertContext(SystemAlertMetadata{
		SystemID:         "sys-1",
		OrganizationID:   "org-1",
		SystemKey:        "SYS-001",
		SystemName:       "web-01",
		SystemType:       "ns8",
		OrganizationName: "Reseller X",
	})

	alert, err := BuildResolvedLinkFailedAlert(systemContext)
	require.NoError(t, err)

	// Identity labels reproduced exactly -> same fingerprint as the firing alert.
	assert.Equal(t, LinkFailedAlert, alert.Labels["alertname"])
	assert.Equal(t, "critical", alert.Labels["severity"])
	assert.Equal(t, ManagedByCollect, alert.Labels[ManagedByLabel])
	assert.Equal(t, "sys-1", alert.Labels["system_id"])
	assert.Equal(t, "SYS-001", alert.Labels["system_key"])
	assert.Equal(t, "web-01", alert.Labels["system_name"])
	assert.Equal(t, "ns8", alert.Labels["system_type"])
	assert.Equal(t, "org-1", alert.Labels["organization_id"])
	assert.Equal(t, "Reseller X", alert.Labels["organization_name"])

	// Empty values must be stripped, exactly as the firing path does.
	_, hasFQDN := alert.Labels["system_fqdn"]
	assert.False(t, hasFQDN, "empty system_fqdn must be stripped to keep fingerprint parity")
	_, hasVAT := alert.Labels["organization_vat"]
	assert.False(t, hasVAT, "empty organization_vat must be stripped")

	// Resolve semantics: EndsAt is in the past and after StartsAt.
	now := time.Now().UTC()
	assert.False(t, alert.EndsAt.After(now.Add(time.Second)), "EndsAt must be <= now to resolve")
	assert.True(t, alert.StartsAt.Before(alert.EndsAt), "StartsAt must precede EndsAt")
}
