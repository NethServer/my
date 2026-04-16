/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cron

import (
	"testing"
	"time"

	collectalerting "github.com/nethesis/my/collect/alerting"
	"github.com/nethesis/my/collect/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkFailedMonitorSyncOrganization_PostsAlertForInactiveSystem(t *testing.T) {
	lastHeartbeat := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)

	var (
		gotOrgID string
		posted   []models.AlertmanagerPostAlert
	)

	monitor := &LinkFailedMonitor{
		timeoutMinutes: 10,
		postAlerts: func(orgID string, alerts []models.AlertmanagerPostAlert) error {
			gotOrgID = orgID
			posted = alerts
			return nil
		},
	}

	err := monitor.syncOrganization("org-1", map[string]linkFailedSystem{
		"SYS-001": {
			Context: collectalerting.BuildSystemAlertContext(collectalerting.SystemAlertMetadata{
				SystemID:       "system-1",
				OrganizationID: "org-1",
				SystemKey:      "SYS-001",
				SystemName:     "web-01",
			}),
			LastHeartbeat: lastHeartbeat,
		},
	})
	require.NoError(t, err)

	require.Equal(t, "org-1", gotOrgID)
	require.Len(t, posted, 1)

	alert := posted[0]
	assert.Equal(t, "LinkFailed", alert.Labels["alertname"])
	assert.Equal(t, "critical", alert.Labels["severity"])
	assert.Equal(t, "my-collect", alert.Labels["managed_by"])
	assert.Equal(t, "system-1", alert.Labels["system_id"])
	assert.Equal(t, "SYS-001", alert.Labels["system_key"])
	assert.Equal(t, "web-01", alert.Labels["system_name"])
	assert.Equal(t, "No heartbeat received from system", alert.Annotations["summary_en"])
	assert.Contains(t, alert.Annotations["description_en"], lastHeartbeat.Format(time.RFC3339))
	assert.Equal(t, lastHeartbeat.Add(10*time.Minute), alert.StartsAt)
	assert.WithinDuration(t, time.Now().UTC().Add(linkFailedAlertTTL), alert.EndsAt, time.Minute)
}

func TestLinkFailedMonitorSyncOrganization_PostsMultipleInactiveSystems(t *testing.T) {
	lastHeartbeat := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)

	var posted []models.AlertmanagerPostAlert
	monitor := &LinkFailedMonitor{
		timeoutMinutes: 10,
		postAlerts: func(_ string, alerts []models.AlertmanagerPostAlert) error {
			posted = alerts
			return nil
		},
	}

	err := monitor.syncOrganization("org-1", map[string]linkFailedSystem{
		"SYS-001": {
			Context: collectalerting.BuildSystemAlertContext(collectalerting.SystemAlertMetadata{
				SystemID:       "system-1",
				OrganizationID: "org-1",
				SystemKey:      "SYS-001",
			}),
			LastHeartbeat: lastHeartbeat,
		},
		"SYS-002": {
			Context: collectalerting.BuildSystemAlertContext(collectalerting.SystemAlertMetadata{
				SystemID:       "system-2",
				OrganizationID: "org-1",
				SystemKey:      "SYS-002",
			}),
			LastHeartbeat: lastHeartbeat,
		},
	})
	require.NoError(t, err)
	assert.Len(t, posted, 2)
}

func TestLinkFailedMonitorSyncOrganization_TTLIsThreeSyncIntervals(t *testing.T) {
	monitor := &LinkFailedMonitor{
		timeoutMinutes: 10,
		postAlerts: func(_ string, alerts []models.AlertmanagerPostAlert) error {
			return nil
		},
	}

	system := linkFailedSystem{
		Context: collectalerting.BuildSystemAlertContext(collectalerting.SystemAlertMetadata{
			SystemID:       "system-1",
			OrganizationID: "org-1",
			SystemKey:      "SYS-001",
		}),
		LastHeartbeat: time.Now().UTC().Add(-30 * time.Minute),
	}

	alert, err := monitor.buildFiringAlert(system)
	require.NoError(t, err)

	expectedTTL := 3 * linkFailedSyncInterval
	assert.WithinDuration(t, time.Now().UTC().Add(expectedTTL), alert.EndsAt, 5*time.Second)
}

func TestLinkFailedMonitorBuildFiringAlert_CapsStartsAtToNow(t *testing.T) {
	recentHeartbeat := time.Now().UTC().Add(-1 * time.Minute)

	monitor := &LinkFailedMonitor{
		timeoutMinutes: 10,
	}

	system := linkFailedSystem{
		Context: collectalerting.BuildSystemAlertContext(collectalerting.SystemAlertMetadata{
			SystemID:       "system-1",
			OrganizationID: "org-1",
			SystemKey:      "SYS-001",
			SystemName:     "web-01",
		}),
		LastHeartbeat: recentHeartbeat,
	}

	alert, err := monitor.buildFiringAlert(system)
	require.NoError(t, err)

	assert.False(t, alert.StartsAt.After(time.Now().UTC()),
		"StartsAt must not be in the future")
	assert.True(t, alert.EndsAt.After(alert.StartsAt),
		"EndsAt must be after StartsAt")
}

func TestLinkFailedMonitorSyncOrganization_NoOpWhenNoInactiveSystems(t *testing.T) {
	called := false
	monitor := &LinkFailedMonitor{
		postAlerts: func(_ string, alerts []models.AlertmanagerPostAlert) error {
			called = true
			return nil
		},
	}

	err := monitor.syncOrganization("org-1", nil)
	require.NoError(t, err)
	assert.False(t, called)

	err = monitor.syncOrganization("org-1", map[string]linkFailedSystem{})
	require.NoError(t, err)
	assert.False(t, called)
}
