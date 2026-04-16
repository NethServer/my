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
	linkFailedAlertTTL     = 10 * 365 * 24 * time.Hour
)

type listAlertsFunc func(orgID string, filters ...string) ([]models.AlertmanagerAlert, error)
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
	listAlerts     listAlertsFunc
	postAlerts     postAlertsFunc
}

// NewLinkFailedMonitor creates a new LinkFailed monitor instance.
func NewLinkFailedMonitor() *LinkFailedMonitor {
	return &LinkFailedMonitor{
		db:             database.DB,
		timeoutMinutes: configuration.Config.HeartbeatTimeoutMinutes,
		syncInterval:   linkFailedSyncInterval,
		listAlerts:     collectalerting.ListAlerts,
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
	orgIDs, err := m.loadOrganizationIDs(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("LinkFailed monitor: failed to load organizations")
		return
	}

	desiredByOrg, err := m.loadInactiveSystems(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("LinkFailed monitor: failed to load inactive systems")
		return
	}

	for _, orgID := range orgIDs {
		if err := m.syncOrganization(orgID, desiredByOrg[orgID]); err != nil {
			logger.Error().Err(err).Str("organization_id", orgID).Msg("LinkFailed monitor: sync failed")
		}
	}
}

func (m *LinkFailedMonitor) loadOrganizationIDs(ctx context.Context) ([]string, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT DISTINCT organization_id
		FROM systems
		WHERE organization_id IS NOT NULL
		  AND organization_id <> ''
	`)
	if err != nil {
		return nil, fmt.Errorf("query organization ids: %w", err)
	}
	defer func() { _ = rows.Close() }()

	orgIDs := make([]string, 0)
	for rows.Next() {
		var orgID string
		if err := rows.Scan(&orgID); err != nil {
			return nil, fmt.Errorf("scan organization id: %w", err)
		}
		orgIDs = append(orgIDs, orgID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organization ids: %w", err)
	}

	return orgIDs, nil
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

func (m *LinkFailedMonitor) syncOrganization(orgID string, desired map[string]linkFailedSystem) error {
	currentAlerts, err := m.listAlerts(
		orgID,
		fmt.Sprintf(`alertname="%s"`, collectalerting.LinkFailedAlert),
		fmt.Sprintf(`%s="%s"`, collectalerting.ManagedByLabel, collectalerting.ManagedByCollect),
	)
	if err != nil {
		return fmt.Errorf("list managed LinkFailed alerts: %w", err)
	}

	currentBySystemKey := make(map[string]models.AlertmanagerAlert, len(currentAlerts))
	for _, alert := range currentAlerts {
		systemKey := alert.Labels["system_key"]
		if systemKey == "" {
			logger.Warn().
				Str("organization_id", orgID).
				Str("fingerprint", alert.Fingerprint).
				Msg("LinkFailed monitor: skipping managed alert without system_key")
			continue
		}
		if _, exists := currentBySystemKey[systemKey]; exists {
			logger.Warn().
				Str("organization_id", orgID).
				Str("system_key", systemKey).
				Msg("LinkFailed monitor: duplicate managed alert for system")
			continue
		}
		currentBySystemKey[systemKey] = alert
	}

	transitions := make([]models.AlertmanagerPostAlert, 0, len(desired)+len(currentBySystemKey))
	firingCount := 0
	resolvedCount := 0

	for systemKey, system := range desired {
		if _, exists := currentBySystemKey[systemKey]; exists {
			continue
		}

		firingAlert, err := m.buildFiringAlert(system)
		if err != nil {
			return fmt.Errorf("build firing alert for %s: %w", systemKey, err)
		}

		transitions = append(transitions, firingAlert)
		firingCount++
	}

	now := time.Now().UTC()
	for systemKey, alert := range currentBySystemKey {
		if _, exists := desired[systemKey]; exists {
			continue
		}

		transitions = append(transitions, models.AlertmanagerPostAlert{
			Labels:      cloneStringMap(alert.Labels),
			Annotations: cloneStringMap(alert.Annotations),
			StartsAt:    alert.StartsAt,
			EndsAt:      now,
		})
		resolvedCount++
	}

	if len(transitions) == 0 {
		return nil
	}

	if err := m.postAlerts(orgID, transitions); err != nil {
		return fmt.Errorf("post alert transitions: %w", err)
	}

	logger.Info().
		Str("organization_id", orgID).
		Int("firing", firingCount).
		Int("resolved", resolvedCount).
		Msg("LinkFailed monitor: synchronized alert transitions")

	return nil
}

func (m *LinkFailedMonitor) buildFiringAlert(system linkFailedSystem) (models.AlertmanagerPostAlert, error) {
	startsAt := system.LastHeartbeat.Add(time.Duration(m.timeoutMinutes) * time.Minute).UTC()
	if startsAt.IsZero() {
		startsAt = time.Now().UTC()
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

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}

	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func nullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
