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

func TestLinkFailedMonitorSyncOrganization_FiresMissingInactiveAlert(t *testing.T) {
	lastHeartbeat := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)

	var (
		gotOrgID string
		filters  []string
		posted   []models.AlertmanagerPostAlert
	)

	monitor := &LinkFailedMonitor{
		timeoutMinutes: 10,
		listAlerts: func(orgID string, requestedFilters ...string) ([]models.AlertmanagerAlert, error) {
			gotOrgID = orgID
			filters = requestedFilters
			return []models.AlertmanagerAlert{}, nil
		},
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
	assert.Equal(t, []string{
		`alertname="LinkFailed"`,
		`managed_by="my-collect"`,
	}, filters)
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

func TestLinkFailedMonitorSyncOrganization_ResolvesOrphanedAlert(t *testing.T) {
	startsAt := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	var posted []models.AlertmanagerPostAlert
	monitor := &LinkFailedMonitor{
		listAlerts: func(string, ...string) ([]models.AlertmanagerAlert, error) {
			return []models.AlertmanagerAlert{
				{
					Labels: map[string]string{
						"alertname":  "LinkFailed",
						"managed_by": "my-collect",
						"system_key": "SYS-001",
					},
					Annotations: map[string]string{
						"summary_en": "No heartbeat received from system",
					},
					StartsAt: startsAt,
				},
			}, nil
		},
		postAlerts: func(_ string, alerts []models.AlertmanagerPostAlert) error {
			posted = alerts
			return nil
		},
	}

	err := monitor.syncOrganization("org-1", nil)
	require.NoError(t, err)
	require.Len(t, posted, 1)

	assert.Equal(t, "LinkFailed", posted[0].Labels["alertname"])
	assert.Equal(t, "my-collect", posted[0].Labels["managed_by"])
	assert.Equal(t, startsAt, posted[0].StartsAt)
	assert.WithinDuration(t, time.Now().UTC(), posted[0].EndsAt, time.Second)
}

func TestLinkFailedMonitorSyncOrganization_ResolvesAlertWithFutureStartsAt(t *testing.T) {
	// Simulate an alert previously posted with a StartsAt in the future (race condition
	// between heartbeat ingestion and heartbeat_monitor tick). The resolve transition must
	// set EndsAt > StartsAt to avoid a 400 from Alertmanager.
	futureStartsAt := time.Now().UTC().Add(5 * time.Minute)

	var posted []models.AlertmanagerPostAlert
	monitor := &LinkFailedMonitor{
		listAlerts: func(string, ...string) ([]models.AlertmanagerAlert, error) {
			return []models.AlertmanagerAlert{
				{
					Labels: map[string]string{
						"alertname":  "LinkFailed",
						"managed_by": "my-collect",
						"system_key": "SYS-001",
					},
					Annotations: map[string]string{
						"summary_en": "No heartbeat received from system",
					},
					StartsAt: futureStartsAt,
				},
			}, nil
		},
		postAlerts: func(_ string, alerts []models.AlertmanagerPostAlert) error {
			posted = alerts
			return nil
		},
	}

	err := monitor.syncOrganization("org-1", nil)
	require.NoError(t, err)
	require.Len(t, posted, 1)

	assert.Equal(t, futureStartsAt, posted[0].StartsAt)
	assert.True(t, posted[0].EndsAt.After(posted[0].StartsAt),
		"EndsAt must be after StartsAt even when StartsAt is in the future")
}

func TestLinkFailedMonitorBuildFiringAlert_CapsStartsAtToNow(t *testing.T) {
	// If last_heartbeat was updated recently (race with heartbeat_monitor tick),
	// startsAt may be in the future. buildFiringAlert must cap it to now.
	recentHeartbeat := time.Now().UTC().Add(-1 * time.Minute) // 1 min ago

	monitor := &LinkFailedMonitor{
		timeoutMinutes: 10, // startsAt would be recentHeartbeat+10min = 9min in the future
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

func TestLinkFailedMonitorSyncOrganization_NoOpWhenAlertAlreadyActive(t *testing.T) {
	called := false
	monitor := &LinkFailedMonitor{
		listAlerts: func(string, ...string) ([]models.AlertmanagerAlert, error) {
			return []models.AlertmanagerAlert{
				{
					Labels: map[string]string{
						"alertname":  "LinkFailed",
						"managed_by": "my-collect",
						"system_key": "SYS-001",
					},
				},
			}, nil
		},
		postAlerts: func(_ string, alerts []models.AlertmanagerPostAlert) error {
			called = true
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
			LastHeartbeat: time.Now().UTC(),
		},
	})
	require.NoError(t, err)
	assert.False(t, called)
}

func TestLinkFailedMonitorSyncOrganization_NoOpWhenAlertAlreadyResolved(t *testing.T) {
	called := false
	monitor := &LinkFailedMonitor{
		listAlerts: func(string, ...string) ([]models.AlertmanagerAlert, error) {
			return []models.AlertmanagerAlert{}, nil
		},
		postAlerts: func(_ string, alerts []models.AlertmanagerPostAlert) error {
			called = true
			return nil
		},
	}

	err := monitor.syncOrganization("org-1", nil)
	require.NoError(t, err)
	assert.False(t, called)
}
