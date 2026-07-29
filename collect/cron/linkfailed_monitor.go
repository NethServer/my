/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cron

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	collectalerting "github.com/nethesis/my/collect/alerting"
	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
)

const (
	linkFailedSyncInterval = 5 * time.Minute
	// Alerts auto-resolve after 2× the sync interval if not refreshed.
	// A system must remain active for a full TTL window before its alert clears,
	// which prevents flapping when heartbeats arrive near the timeout boundary.
	linkFailedAlertTTL = 2 * linkFailedSyncInterval
)

type postAlertsFunc func(orgID string, alerts []models.AlertmanagerPostAlert) error

type linkFailedSystem struct {
	Context       *collectalerting.SystemAlertContext
	LastHeartbeat time.Time
}

// LinkFailedMonitor keeps internally-managed LinkFailed alerts aligned with system status.
type LinkFailedMonitor struct {
	db             *sql.DB
	timeoutMinutes int
	syncInterval   time.Duration
	postAlerts     postAlertsFunc
	// Lost-wave guard: see suppressLostWave.
	lostWaveMinSystems int
	lostWaveWindow     time.Duration
}

// NewLinkFailedMonitor creates a new LinkFailed monitor instance.
func NewLinkFailedMonitor() *LinkFailedMonitor {
	return &LinkFailedMonitor{
		db:                 database.DB,
		timeoutMinutes:     configuration.Config.HeartbeatTimeoutMinutes,
		syncInterval:       linkFailedSyncInterval,
		postAlerts:         collectalerting.PostAlerts,
		lostWaveMinSystems: configuration.Config.LinkFailedLostWaveMinSystems,
		lostWaveWindow:     time.Duration(configuration.Config.LinkFailedLostWaveWindowSeconds) * time.Second,
	}
}

// Start begins the LinkFailed synchronization cron job. It blocks until ctx is cancelled.
func (m *LinkFailedMonitor) Start(ctx context.Context) {
	logger.Info().
		Int("timeout_minutes", m.timeoutMinutes).
		Dur("sync_interval", m.syncInterval).
		Msg("Starting LinkFailed monitor cron job")

	ticker := time.NewTicker(m.syncInterval)
	defer ticker.Stop()

	m.sync(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("LinkFailed monitor stopped")
			return
		case <-ticker.C:
			m.sync(ctx)
		}
	}
}

func (m *LinkFailedMonitor) sync(ctx context.Context) {
	desiredByOrg, err := m.loadInactiveSystems(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("LinkFailed monitor: failed to load inactive systems")
		return
	}

	if suppressed, clusters := suppressLostWave(desiredByOrg, m.lostWaveMinSystems, m.lostWaveWindow); suppressed > 0 {
		clusterStarts := make([]string, 0, len(clusters))
		for _, at := range clusters {
			clusterStarts = append(clusterStarts, at.UTC().Format(time.RFC3339))
		}
		logger.Error().
			Int("systems_suppressed", suppressed).
			Strs("heartbeat_cluster_starts", clusterStarts).
			Dur("cluster_window", m.lostWaveWindow).
			Msg("LinkFailed monitor: suppressed a synchronized heartbeat gap — heartbeat recording broke on our side, these systems are not down")
	}

	for tenantOrgID, systems := range desiredByOrg {
		if err := m.syncOrganization(tenantOrgID, systems); err != nil {
			logger.Error().Err(err).Str("tenant_org_id", tenantOrgID).Msg("LinkFailed monitor: sync failed")
		}
	}
}

func (m *LinkFailedMonitor) loadInactiveSystems(ctx context.Context) (map[string]map[string]linkFailedSystem, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT s.id::text,
		       s.organization_id,
		       s.system_key,
		       s.name,
		       s.type,
		       s.fqdn,
		       s.ipv4_address::text,
		       COALESCE(d.name, r.name, c.name),
		       COALESCE(d.custom_data->>'vat', r.custom_data->>'vat', c.custom_data->>'vat'),
		       CASE
		           WHEN d.logto_id IS NOT NULL THEN 'distributor'
		           WHEN r.logto_id IS NOT NULL THEN 'reseller'
		           WHEN c.logto_id IS NOT NULL THEN 'customer'
		           ELSE NULL
		       END,
		       COALESCE(NULLIF(c.custom_data->>'createdBy', ''), s.organization_id),
		       h.last_heartbeat
		FROM systems s
		INNER JOIN system_heartbeats h ON s.id = h.system_id
		LEFT JOIN distributors d ON (s.organization_id = d.logto_id OR s.organization_id = d.id) AND d.deleted_at IS NULL
		LEFT JOIN resellers r ON (s.organization_id = r.logto_id OR s.organization_id = r.id) AND r.deleted_at IS NULL
		LEFT JOIN customers c ON (s.organization_id = c.logto_id OR s.organization_id = c.id) AND c.deleted_at IS NULL
		WHERE s.status = 'inactive'
		  AND s.deleted_at IS NULL
		  -- Never alert on suspended systems. s.suspended_at also covers org
		  -- cascade suspension (SuspendSystemsByMultipleOrgIDs writes it down to
		  -- the system); the COALESCE on the org row is a belt-and-suspenders
		  -- guard for any system the cascade missed (e.g. created under an
		  -- already-suspended org).
		  AND s.suspended_at IS NULL
		  AND COALESCE(d.suspended_at, r.suspended_at, c.suspended_at) IS NULL
		  AND s.organization_id IS NOT NULL
		  AND s.organization_id <> ''
	`)
	if err != nil {
		return nil, fmt.Errorf("query inactive systems: %w", err)
	}
	defer func() { _ = rows.Close() }()

	systemsByOrg := make(map[string]map[string]linkFailedSystem)
	for rows.Next() {
		var (
			metadata         collectalerting.SystemAlertMetadata
			systemType       sql.NullString
			systemFQDN       sql.NullString
			systemIPv4       sql.NullString
			organizationName sql.NullString
			organizationVAT  sql.NullString
			organizationType sql.NullString
			lastHeartbeat    time.Time
		)

		if err := rows.Scan(
			&metadata.SystemID,
			&metadata.OrganizationID,
			&metadata.SystemKey,
			&metadata.SystemName,
			&systemType,
			&systemFQDN,
			&systemIPv4,
			&organizationName,
			&organizationVAT,
			&organizationType,
			&metadata.ResellerOrgID,
			&lastHeartbeat,
		); err != nil {
			return nil, fmt.Errorf("scan inactive system: %w", err)
		}

		metadata.SystemType = nullStringValue(systemType)
		metadata.SystemFQDN = nullStringValue(systemFQDN)
		metadata.SystemIPv4 = nullStringValue(systemIPv4)
		metadata.OrganizationName = nullStringValue(organizationName)
		metadata.OrganizationVAT = nullStringValue(organizationVAT)
		metadata.OrganizationType = nullStringValue(organizationType)

		// Group by the reseller/managing org: that is the Mimir tenant
		// (X-Scope-OrgID). The alert's organization_id label stays the customer
		// org (set in BuildSystemAlertContext), so per-customer routing/filtering
		// still works inside the shared reseller tenant.
		if systemsByOrg[metadata.ResellerOrgID] == nil {
			systemsByOrg[metadata.ResellerOrgID] = make(map[string]linkFailedSystem)
		}
		systemsByOrg[metadata.ResellerOrgID][metadata.SystemKey] = linkFailedSystem{
			Context:       collectalerting.BuildSystemAlertContext(metadata),
			LastHeartbeat: lastHeartbeat.UTC(),
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inactive systems: %w", err)
	}

	return systemsByOrg, nil
}

// lostWaveEntry locates one system inside the desired-alert map alongside its
// heartbeat time, so a detected cluster can be removed in place.
type lostWaveEntry struct {
	tenantOrgID string
	systemKey   string
	at          time.Time
}

// suppressLostWave drops inactive systems whose last_heartbeat values all fall
// inside a single narrow window.
//
// Clients send heartbeats on wall-clock-aligned waves, so when the platform
// briefly stops recording them every affected system is left with the same
// last_heartbeat to within a few seconds. Hundreds of machines do not genuinely
// lose connectivity in the same second: a tight cluster means our own ingest
// gapped, and firing on it pages every customer at once for a fault on our side.
// A machine that is really dead stopped beating at an arbitrary moment, so it
// never joins the cluster and still alerts normally.
//
// The scan deliberately runs across tenants: an ingest gap hits every tenant at
// once, which is what separates it from one reseller losing an uplink.
//
// One ingest gap usually spans several consecutive waves, so every qualifying
// cluster is removed, not just the largest. Once two waves have been lost the
// inactive set holds both, and suppressing only the biggest would still page the
// systems belonging to the smaller one.
//
// Known trade-off: a wave can straddle the staleness cutoff and so reach the
// inactive set across two ticks. If the first slice lands below minSystems it
// alerts, and when the rest arrives the now-qualifying cluster is suppressed, so
// those few alerts stop being refreshed and expire into a spurious "resolved"
// notification one TTL later. Bounded by minSystems and self-correcting, so it is
// accepted rather than tracked with per-alert state.
//
// Returns how many systems were dropped and the start of each offending window,
// oldest first. A minSystems or window of zero or less disables the guard.
func suppressLostWave(byOrg map[string]map[string]linkFailedSystem, minSystems int, window time.Duration) (int, []time.Time) {
	if minSystems <= 0 || window <= 0 {
		return 0, nil
	}

	entries := make([]lostWaveEntry, 0, len(byOrg))
	for tenantOrgID, systems := range byOrg {
		for systemKey, system := range systems {
			entries = append(entries, lostWaveEntry{
				tenantOrgID: tenantOrgID,
				systemKey:   systemKey,
				at:          system.LastHeartbeat,
			})
		}
	}
	if len(entries) < minSystems {
		return 0, nil
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].at.Before(entries[j].at) })

	// Sweep once, carving out every maximal run that fits inside one window and
	// meets the floor. Advancing past a suppressed run rather than restarting
	// keeps this linear and stops a run from being counted twice.
	var (
		suppressed int
		clusters   []time.Time
		start      int
	)
	for start < len(entries) {
		end := start
		for end+1 < len(entries) && entries[end+1].at.Sub(entries[start].at) <= window {
			end++
		}

		if count := end - start + 1; count >= minSystems {
			for _, entry := range entries[start : end+1] {
				delete(byOrg[entry.tenantOrgID], entry.systemKey)
			}
			suppressed += count
			clusters = append(clusters, entries[start].at)
			start = end + 1
			continue
		}

		start++
	}

	return suppressed, clusters
}

// syncOrganization pushes the firing alerts for one Mimir tenant. tenantOrgID
// is the reseller/managing org (X-Scope-OrgID); the systems it carries may
// belong to several customer orgs, each identified by the organization_id
// label on its alerts.
func (m *LinkFailedMonitor) syncOrganization(tenantOrgID string, inactive map[string]linkFailedSystem) error {
	if len(inactive) == 0 {
		return nil
	}

	alerts := make([]models.AlertmanagerPostAlert, 0, len(inactive))
	for systemKey, system := range inactive {
		firingAlert, err := m.buildFiringAlert(system)
		if err != nil {
			return fmt.Errorf("build firing alert for %s: %w", systemKey, err)
		}
		alerts = append(alerts, firingAlert)

		logger.Debug().
			Str("tenant_org_id", tenantOrgID).
			Str("system_key", systemKey).
			Msg("LinkFailed monitor: refreshing alert for inactive system")
	}

	if err := m.postAlerts(tenantOrgID, alerts); err != nil {
		return fmt.Errorf("post alerts: %w", err)
	}

	logger.Info().
		Str("tenant_org_id", tenantOrgID).
		Int("alerts_refreshed", len(alerts)).
		Msg("LinkFailed monitor: refreshed alerts for inactive systems")

	return nil
}

func (m *LinkFailedMonitor) buildFiringAlert(system linkFailedSystem) (models.AlertmanagerPostAlert, error) {
	startsAt := system.LastHeartbeat.Add(time.Duration(m.timeoutMinutes) * time.Minute).UTC()
	now := time.Now().UTC()
	// Cap to now: if last_heartbeat was updated after the system was marked inactive
	// (race between heartbeat ingestion and heartbeat_monitor tick), startsAt may land
	// in the future. Alertmanager rejects resolutions where EndsAt < StartsAt.
	if startsAt.IsZero() || startsAt.After(now) {
		startsAt = now
	}

	enrichedAlerts, err := collectalerting.EnrichAlerts([]models.AlertmanagerPostAlert{
		{
			Labels: map[string]string{
				"alertname":                    collectalerting.LinkFailedAlert,
				"severity":                     "critical",
				collectalerting.ManagedByLabel: collectalerting.ManagedByCollect,
			},
			Annotations: map[string]string{
				"summary_en":     "No heartbeat received from system",
				"summary_it":     "Nessun heartbeat ricevuto dal sistema",
				"description_en": fmt.Sprintf("The system has not communicated with My Nethesis since %s. The alert is raised after %d minutes without a heartbeat. Check the system connection.", system.LastHeartbeat.Format(time.RFC3339), m.timeoutMinutes),
				"description_it": fmt.Sprintf("Il sistema non ha comunicato con My Nethesis dal %s. L'allarme viene generato dopo %d minuti senza heartbeat. Verificare la connessione del sistema.", system.LastHeartbeat.Format(time.RFC3339), m.timeoutMinutes),
			},
			StartsAt: startsAt,
			EndsAt:   time.Now().UTC().Add(linkFailedAlertTTL),
		},
	}, system.Context)
	if err != nil {
		return models.AlertmanagerPostAlert{}, err
	}

	return enrichedAlerts[0], nil
}

func nullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
