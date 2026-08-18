/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cron

import (
	"fmt"
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

func TestLinkFailedMonitorSyncOrganization_TTLMatchesAlertTTLConstant(t *testing.T) {
	monitor := &LinkFailedMonitor{
		timeoutMinutes: 10,
		postAlerts:     func(_ string, _ []models.AlertmanagerPostAlert) error { return nil },
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

	assert.WithinDuration(t, time.Now().UTC().Add(linkFailedAlertTTL), alert.EndsAt, 5*time.Second)
}

// clusteredSystems builds n inactive systems whose last_heartbeat values are
// spread evenly across span, mimicking one synchronized heartbeat wave.
func clusteredSystems(prefix string, n int, base time.Time, span time.Duration) map[string]linkFailedSystem {
	out := make(map[string]linkFailedSystem, n)
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("%s-%03d", prefix, i)
		offset := time.Duration(0)
		if n > 1 {
			offset = time.Duration(int64(span) * int64(i) / int64(n-1))
		}
		out[key] = linkFailedSystem{
			Context: collectalerting.BuildSystemAlertContext(collectalerting.SystemAlertMetadata{
				SystemID:       key,
				OrganizationID: "org-1",
				SystemKey:      key,
			}),
			LastHeartbeat: base.Add(offset),
		}
	}
	return out
}

func TestSuppressLostWave_SuppressesSynchronizedWaveKeepsChronic(t *testing.T) {
	waveBase := time.Date(2026, 7, 28, 12, 20, 0, 0, time.UTC)

	systems := clusteredSystems("WAVE", 25, waveBase, 61*time.Second)
	// Three machines genuinely offline for days: heartbeats scattered, not clustered.
	for i, age := range []time.Duration{72 * time.Hour, 26 * time.Hour, 8 * time.Hour} {
		key := fmt.Sprintf("CHRONIC-%d", i)
		systems[key] = linkFailedSystem{
			Context: collectalerting.BuildSystemAlertContext(collectalerting.SystemAlertMetadata{
				SystemID: key, OrganizationID: "org-1", SystemKey: key,
			}),
			LastHeartbeat: waveBase.Add(-age),
		}
	}

	byOrg := map[string]map[string]linkFailedSystem{"tenant-1": systems}

	suppressed, clusters := suppressLostWave(byOrg, 20, 3*time.Minute)

	assert.Equal(t, 25, suppressed, "the whole wave must be suppressed")
	assert.Equal(t, []time.Time{waveBase}, clusters, "reported cluster start should be the wave start")
	require.Len(t, byOrg["tenant-1"], 3, "chronic offline systems must survive")
	for i := 0; i < 3; i++ {
		assert.Contains(t, byOrg["tenant-1"], fmt.Sprintf("CHRONIC-%d", i))
	}
}

// A single ingest gap spans several consecutive waves, so by the second monitor
// tick the inactive set holds two distinct clusters. Suppressing only the largest
// would still page every system in the smaller one.
func TestSuppressLostWave_SuppressesEveryClusterAboveFloor(t *testing.T) {
	firstWave := time.Date(2026, 7, 28, 12, 10, 0, 0, time.UTC)
	secondWave := firstWave.Add(10 * time.Minute)

	systems := clusteredSystems("EARLY", 25, firstWave, 57*time.Second)
	for key, system := range clusteredSystems("LATE", 30, secondWave, 61*time.Second) {
		systems[key] = system
	}
	systems["CHRONIC"] = linkFailedSystem{
		Context: collectalerting.BuildSystemAlertContext(collectalerting.SystemAlertMetadata{
			SystemID: "CHRONIC", OrganizationID: "org-1", SystemKey: "CHRONIC",
		}),
		LastHeartbeat: firstWave.Add(-72 * time.Hour),
	}

	byOrg := map[string]map[string]linkFailedSystem{"tenant-1": systems}

	suppressed, clusters := suppressLostWave(byOrg, 20, 3*time.Minute)

	assert.Equal(t, 55, suppressed, "both lost waves must be suppressed, not just the biggest")
	assert.Equal(t, []time.Time{firstWave, secondWave}, clusters, "each cluster reported, oldest first")
	assert.Equal(t, map[string]bool{"CHRONIC": true}, keySet(byOrg["tenant-1"]))
}

func TestSuppressLostWave_IgnoresClusterBelowFloor(t *testing.T) {
	base := time.Date(2026, 7, 28, 12, 20, 0, 0, time.UTC)
	byOrg := map[string]map[string]linkFailedSystem{
		"tenant-1": clusteredSystems("SYS", 5, base, 30*time.Second),
	}

	suppressed, _ := suppressLostWave(byOrg, 20, 3*time.Minute)

	assert.Zero(t, suppressed)
	assert.Len(t, byOrg["tenant-1"], 5, "a handful of simultaneous outages is plausible and must alert")
}

func TestSuppressLostWave_IgnoresHeartbeatsSpreadOverTime(t *testing.T) {
	base := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	// 30 systems, each an hour apart: no wave, just an unlucky fleet.
	byOrg := map[string]map[string]linkFailedSystem{
		"tenant-1": clusteredSystems("SYS", 30, base, 30*time.Hour),
	}

	suppressed, _ := suppressLostWave(byOrg, 20, 3*time.Minute)

	assert.Zero(t, suppressed)
	assert.Len(t, byOrg["tenant-1"], 30)
}

func TestSuppressLostWave_SpansMultipleTenants(t *testing.T) {
	waveBase := time.Date(2026, 7, 28, 12, 20, 0, 0, time.UTC)
	byOrg := map[string]map[string]linkFailedSystem{
		"tenant-1": clusteredSystems("A", 12, waveBase, 40*time.Second),
		"tenant-2": clusteredSystems("B", 12, waveBase.Add(10*time.Second), 40*time.Second),
	}

	suppressed, _ := suppressLostWave(byOrg, 20, 3*time.Minute)

	assert.Equal(t, 24, suppressed, "a platform-side gap crosses tenants and must be detected across them")
	assert.Empty(t, byOrg["tenant-1"])
	assert.Empty(t, byOrg["tenant-2"])
}

func TestSuppressLostWave_DisabledWhenFloorNotPositive(t *testing.T) {
	base := time.Date(2026, 7, 28, 12, 20, 0, 0, time.UTC)
	byOrg := map[string]map[string]linkFailedSystem{
		"tenant-1": clusteredSystems("SYS", 300, base, 61*time.Second),
	}

	suppressed, _ := suppressLostWave(byOrg, 0, 3*time.Minute)

	assert.Zero(t, suppressed, "a non-positive floor disables the guard entirely")
	assert.Len(t, byOrg["tenant-1"], 300)
}

// Fleet-scale check: a lost wave large enough to span several tenants, alongside
// a genuinely-dead machine. No customer alert should leave the process for the
// wave, while the dead machine must still alert.
func TestSuppressLostWave_FleetScaleWave(t *testing.T) {
	waveBase := time.Date(2026, 7, 28, 12, 20, 0, 0, time.UTC)

	tenantA := clusteredSystems("A", 120, waveBase, 61*time.Second)
	tenantB := clusteredSystems("B", 131, waveBase.Add(3*time.Second), 58*time.Second)
	tenantB["DEAD-1"] = linkFailedSystem{
		Context: collectalerting.BuildSystemAlertContext(collectalerting.SystemAlertMetadata{
			SystemID: "DEAD-1", OrganizationID: "org-9", SystemKey: "DEAD-1",
		}),
		LastHeartbeat: waveBase.Add(-96 * time.Hour),
	}

	byOrg := map[string]map[string]linkFailedSystem{"tenant-a": tenantA, "tenant-b": tenantB}

	suppressed, _ := suppressLostWave(byOrg, 20, 3*time.Minute)

	assert.Equal(t, 251, suppressed)
	assert.Empty(t, byOrg["tenant-a"])
	assert.Equal(t, map[string]bool{"DEAD-1": true}, keySet(byOrg["tenant-b"]))
}

// The guard only matters if suppressed systems genuinely never reach the
// Alertmanager fan-out, so assert on what postAlerts receives rather than on the
// intermediate map: a lost wave must produce no customer alerts at all, while a
// genuinely-dead machine in the same cycle must still be alerted on.
func TestLinkFailedMonitorSync_SuppressedWaveReachesNoCustomer(t *testing.T) {
	waveBase := time.Date(2026, 7, 28, 12, 20, 0, 0, time.UTC)

	systems := clusteredSystems("WAVE", 40, waveBase, 61*time.Second)
	systems["DEAD"] = linkFailedSystem{
		Context: collectalerting.BuildSystemAlertContext(collectalerting.SystemAlertMetadata{
			SystemID: "DEAD", OrganizationID: "org-1", SystemKey: "DEAD",
		}),
		LastHeartbeat: waveBase.Add(-96 * time.Hour),
	}
	byOrg := map[string]map[string]linkFailedSystem{"tenant-1": systems}

	var posted []models.AlertmanagerPostAlert
	monitor := &LinkFailedMonitor{
		timeoutMinutes:     30,
		lostWaveMinSystems: 20,
		lostWaveWindow:     3 * time.Minute,
		postAlerts: func(_ string, alerts []models.AlertmanagerPostAlert) error {
			posted = append(posted, alerts...)
			return nil
		},
	}

	suppressed, _ := suppressLostWave(byOrg, monitor.lostWaveMinSystems, monitor.lostWaveWindow)
	require.Equal(t, 40, suppressed)

	for tenantOrgID, remaining := range byOrg {
		require.NoError(t, monitor.syncOrganization(tenantOrgID, remaining))
	}

	require.Len(t, posted, 1, "only the genuinely-dead system may be alerted on")
	assert.Equal(t, "DEAD", posted[0].Labels["system_key"])
}

// Counterpart to the above: with the guard configured but no wave present, every
// inactive system must still be alerted on. Guards against a guard that silently
// swallows normal alerting.
func TestLinkFailedMonitorSync_NoWaveStillAlertsEveryone(t *testing.T) {
	base := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	byOrg := map[string]map[string]linkFailedSystem{
		"tenant-1": clusteredSystems("SYS", 6, base, 30*time.Hour),
	}

	var posted []models.AlertmanagerPostAlert
	monitor := &LinkFailedMonitor{
		timeoutMinutes:     30,
		lostWaveMinSystems: 20,
		lostWaveWindow:     3 * time.Minute,
		postAlerts: func(_ string, alerts []models.AlertmanagerPostAlert) error {
			posted = append(posted, alerts...)
			return nil
		},
	}

	suppressed, clusters := suppressLostWave(byOrg, monitor.lostWaveMinSystems, monitor.lostWaveWindow)
	require.Zero(t, suppressed)
	require.Empty(t, clusters)

	for tenantOrgID, remaining := range byOrg {
		require.NoError(t, monitor.syncOrganization(tenantOrgID, remaining))
	}

	assert.Len(t, posted, 6)
}

func keySet(in map[string]linkFailedSystem) map[string]bool {
	out := make(map[string]bool, len(in))
	for k := range in {
		out[k] = true
	}
	return out
}

func TestLinkFailedMonitorSyncOrganization_NoOpWhenNoInactiveSystems(t *testing.T) {
	called := false
	monitor := &LinkFailedMonitor{
		postAlerts: func(_ string, _ []models.AlertmanagerPostAlert) error {
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
