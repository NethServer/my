/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package cron

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
)

type firedAlert struct {
	OrgID    string
	StartsAt time.Time
}

// HeartbeatMonitor checks system heartbeats and updates status accordingly
type HeartbeatMonitor struct {
	db               *sql.DB
	mimirURL         string
	timeoutMinutes   int
	checkIntervalSec int
	firedAlerts      map[string]firedAlert
	mu               sync.Mutex
}

// NewHeartbeatMonitor creates a new heartbeat monitor instance
func NewHeartbeatMonitor() *HeartbeatMonitor {
	return &HeartbeatMonitor{
		db:               database.DB,
		mimirURL:         configuration.Config.MimirURL,
		timeoutMinutes:   configuration.Config.HeartbeatTimeoutMinutes,
		checkIntervalSec: 60, // Check every 60 seconds
		firedAlerts:      make(map[string]firedAlert),
	}
}

// Start begins the heartbeat monitoring cron job. It blocks until ctx is cancelled.
func (h *HeartbeatMonitor) Start(ctx context.Context) {
	logger.Info().
		Int("timeout_minutes", h.timeoutMinutes).
		Int("check_interval_seconds", h.checkIntervalSec).
		Msg("Starting heartbeat monitor cron job")

	ticker := time.NewTicker(time.Duration(h.checkIntervalSec) * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	h.checkAndUpdateStatuses(ctx)

	// Then run on ticker, stopping when context is cancelled
	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Heartbeat monitor stopped")
			return
		case <-ticker.C:
			h.checkAndUpdateStatuses(ctx)
		}
	}
}

type hostDownSystem struct {
	SystemKey string
	OrgID     string
}

type mimirAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
}

// checkAndUpdateStatuses checks all system heartbeats, updates statuses, and
// keeps HostDown alerts aligned with the current inactive systems.
func (h *HeartbeatMonitor) checkAndUpdateStatuses(ctx context.Context) {
	cutoff := time.Now().Add(-time.Duration(h.timeoutMinutes) * time.Minute)

	// Update systems to 'active' if they have recent heartbeat and are not currently 'active'
	// This handles: unknown -> active, inactive -> active
	queryActive := `
		UPDATE systems s
		SET status = 'active', updated_at = NOW()
		FROM system_heartbeats h
		WHERE s.id = h.system_id
			AND h.last_heartbeat > $1
			AND s.status != 'active'
			AND s.deleted_at IS NULL
	`

	resultActive, err := h.db.ExecContext(ctx, queryActive, cutoff)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to update systems to active status")
	} else {
		rowsAffected, _ := resultActive.RowsAffected()
		if rowsAffected > 0 {
			logger.Info().
				Int64("systems_updated", rowsAffected).
				Msg("Updated systems to active status")
		}
	}

	// Update systems to 'inactive' if they have old heartbeat and are currently 'active'
	queryInactive := `
		UPDATE systems s
		SET status = 'inactive', updated_at = NOW()
		FROM system_heartbeats h
		WHERE s.id = h.system_id
			AND h.last_heartbeat <= $1
			AND s.status = 'active'
			AND s.deleted_at IS NULL
	`

	resultInactive, err := h.db.ExecContext(ctx, queryInactive, cutoff)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to update systems to inactive status")
	} else {
		rowsAffected, _ := resultInactive.RowsAffected()
		if rowsAffected > 0 {
			logger.Warn().
				Int64("systems_updated", rowsAffected).
				Msg("Updated systems to inactive status")
		}
	}

	inactiveSystems, err := h.getInactiveSystems(ctx, cutoff)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to load inactive systems for HostDown alerts")
	} else {
		h.syncHostDownAlerts(inactiveSystems)
	}

	logger.Debug().
		Time("cutoff", cutoff).
		Msg("Heartbeat status check completed")
}

func (h *HeartbeatMonitor) getInactiveSystems(ctx context.Context, cutoff time.Time) (map[string]hostDownSystem, error) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT s.system_key, s.organization_id
		FROM systems s
		INNER JOIN system_heartbeats hb ON s.id = hb.system_id
		WHERE hb.last_heartbeat <= $1
			AND s.deleted_at IS NULL
			AND s.suspended_at IS NULL
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close inactive systems rows")
		}
	}()

	inactiveSystems := make(map[string]hostDownSystem)
	for rows.Next() {
		var system hostDownSystem
		if err := rows.Scan(&system.SystemKey, &system.OrgID); err != nil {
			return nil, err
		}
		inactiveSystems[system.SystemKey] = system
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return inactiveSystems, nil
}

func (h *HeartbeatMonitor) syncHostDownAlerts(inactiveSystems map[string]hostDownSystem) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for systemKey, alert := range h.firedAlerts {
		if _, stillInactive := inactiveSystems[systemKey]; stillInactive {
			continue
		}

		if err := h.resolveHostDownAlert(systemKey, alert.OrgID); err != nil {
			logger.Error().
				Err(err).
				Str("system_key", systemKey).
				Msg("Failed to resolve HostDown alert")
			continue
		}

		logger.Info().
			Str("system_key", systemKey).
			Dur("was_down_for", time.Since(alert.StartsAt)).
			Msg("Resolved HostDown alert")
		delete(h.firedAlerts, systemKey)
	}

	for systemKey, system := range inactiveSystems {
		activeAlert, alreadyFired := h.firedAlerts[systemKey]
		startsAt := activeAlert.StartsAt
		if !alreadyFired {
			startsAt = time.Now()
		}

		if err := h.fireHostDownAlert(systemKey, system.OrgID, startsAt); err != nil {
			logger.Error().
				Err(err).
				Str("system_key", systemKey).
				Msg("Failed to fire HostDown alert")
			continue
		}

		if !alreadyFired {
			logger.Warn().
				Str("system_key", systemKey).
				Int("timeout_minutes", h.timeoutMinutes).
				Msg("Fired HostDown alert")
		}

		h.firedAlerts[systemKey] = firedAlert{
			OrgID:    system.OrgID,
			StartsAt: startsAt,
		}
	}
}

func (h *HeartbeatMonitor) fireHostDownAlert(systemKey, orgID string, startsAt time.Time) error {
	endsAt := time.Now().Add(time.Duration(h.checkIntervalSec*3) * time.Second)
	return h.postHostDownAlert(systemKey, orgID, startsAt, endsAt)
}

func (h *HeartbeatMonitor) resolveHostDownAlert(systemKey, orgID string) error {
	return h.postHostDownAlert(systemKey, orgID, time.Now().Add(-time.Duration(h.timeoutMinutes)*time.Minute), time.Now())
}

func (h *HeartbeatMonitor) postHostDownAlert(systemKey, orgID string, startsAt, endsAt time.Time) error {
	payload := []mimirAlert{
		{
			Labels: map[string]string{
				"alertname":  "HostDown",
				"severity":   "critical",
				"system_key": systemKey,
			},
			Annotations: map[string]string{
				"summary": fmt.Sprintf("system %s is down: no heartbeat received", systemKey),
			},
			StartsAt: startsAt,
			EndsAt:   endsAt,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal alert: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/alertmanager/api/v2/alerts", h.mimirURL), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Scope-OrgID", orgID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("post to mimir: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close HostDown response body")
		}
	}()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("mimir returned status %d", resp.StatusCode)
	}

	return nil
}
