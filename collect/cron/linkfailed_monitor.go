/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cron

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	collectalerting "github.com/nethesis/my/collect/alerting"
	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
)

const (
	linkFailedSyncInterval = 5 * time.Minute
	// Alerts auto-resolve after 3× the sync interval if not refreshed.
	// A system must remain active for a full TTL window before its alert clears,
	// which prevents flapping when heartbeats arrive near the timeout boundary.
	linkFailedAlertTTL = 3 * linkFailedSyncInterval
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
}

// NewLinkFailedMonitor creates a new LinkFailed monitor instance.
func NewLinkFailedMonitor() *LinkFailedMonitor {
	return &LinkFailedMonitor{
		db:             database.DB,
		timeoutMinutes: configuration.Config.HeartbeatTimeoutMinutes,
		syncInterval:   linkFailedSyncInterval,
		postAlerts:     collectalerting.PostAlerts,
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

	for orgID, systems := range desiredByOrg {
		if err := m.syncOrganization(orgID, systems); err != nil {
			logger.Error().Err(err).Str("organization_id", orgID).Msg("LinkFailed monitor: sync failed")
		}
	}
}

func (m *LinkFailedMonitor) loadInactiveSystems(ctx context.Context) (map[string]map[string]linkFailedSystem, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT s.id::text,
		       s.organization_id,
		       s.system_key,
		       s.name,
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
		       h.last_heartbeat
		FROM systems s
		INNER JOIN system_heartbeats h ON s.id = h.system_id
		LEFT JOIN distributors d ON (s.organization_id = d.logto_id OR s.organization_id = d.id) AND d.deleted_at IS NULL
		LEFT JOIN resellers r ON (s.organization_id = r.logto_id OR s.organization_id = r.id) AND r.deleted_at IS NULL
		LEFT JOIN customers c ON (s.organization_id = c.logto_id OR s.organization_id = c.id) AND c.deleted_at IS NULL
		WHERE s.status = 'inactive'
		  AND s.deleted_at IS NULL
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
			&systemFQDN,
			&systemIPv4,
			&organizationName,
			&organizationVAT,
			&organizationType,
			&lastHeartbeat,
		); err != nil {
			return nil, fmt.Errorf("scan inactive system: %w", err)
		}

		metadata.SystemFQDN = nullStringValue(systemFQDN)
		metadata.SystemIPv4 = nullStringValue(systemIPv4)
		metadata.OrganizationName = nullStringValue(organizationName)
		metadata.OrganizationVAT = nullStringValue(organizationVAT)
		metadata.OrganizationType = nullStringValue(organizationType)

		if systemsByOrg[metadata.OrganizationID] == nil {
			systemsByOrg[metadata.OrganizationID] = make(map[string]linkFailedSystem)
		}
		systemsByOrg[metadata.OrganizationID][metadata.SystemKey] = linkFailedSystem{
			Context:       collectalerting.BuildSystemAlertContext(metadata),
			LastHeartbeat: lastHeartbeat.UTC(),
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inactive systems: %w", err)
	}

	return systemsByOrg, nil
}

func (m *LinkFailedMonitor) syncOrganization(orgID string, inactive map[string]linkFailedSystem) error {
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
			Str("organization_id", orgID).
			Str("system_key", systemKey).
			Msg("LinkFailed monitor: refreshing alert for inactive system")
	}

	if err := m.postAlerts(orgID, alerts); err != nil {
		return fmt.Errorf("post alerts: %w", err)
	}

	logger.Info().
		Str("organization_id", orgID).
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
				"description_en": fmt.Sprintf("The system has not communicated with the server since %s. Check the service connection.", system.LastHeartbeat.Format(time.RFC3339)),
				"description_it": fmt.Sprintf("Il sistema non ha comunicato con il server dal %s. Verificare la connessione al servizio.", system.LastHeartbeat.Format(time.RFC3339)),
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
