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
	"time"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
)

// HeartbeatMonitor checks system heartbeats and updates status accordingly
type HeartbeatMonitor struct {
	db               *sql.DB
	mimirURL         string
	timeoutMinutes   int
	checkIntervalSec int
}

// NewHeartbeatMonitor creates a new heartbeat monitor instance
func NewHeartbeatMonitor() *HeartbeatMonitor {
	return &HeartbeatMonitor{
		db:               database.DB,
		mimirURL:         configuration.Config.MimirURL,
		timeoutMinutes:   configuration.Config.HeartbeatTimeoutMinutes,
		checkIntervalSec: 60, // Check every 60 seconds
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

type mimirAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
}

// checkAndUpdateStatuses checks all system heartbeats, updates statuses, and
// keeps LinkFailed alerts aligned with the current inactive systems.
func (h *HeartbeatMonitor) checkAndUpdateStatuses(ctx context.Context) {
	cutoff := time.Now().Add(-time.Duration(h.timeoutMinutes) * time.Minute)

	// Select systems that will be set to 'active' if they have recent heartbeat and are not currently 'active'
	// This handles: unknown -> active, inactive -> active
	queryActive := `
		SELECT s.system_key, s.organization_id
		FROM systems s
		INNER JOIN system_heartbeats h ON s.id = h.system_id
		WHERE h.last_heartbeat > $1
			AND s.status != 'active'
			AND s.deleted_at IS NULL
	`

	// Update systems to 'active' and resolve LinkFailed alerts
	resultActive, err := h.db.QueryContext(ctx, queryActive, cutoff)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to query systems for active status update")
	} else {
		for resultActive.Next() {
			var systemKey, orgID string
			if err := resultActive.Scan(&systemKey, &orgID); err != nil {
				logger.Error().
					Err(err).
					Str("system_key", systemKey).
					Msg("Failed to scan active system row")
				continue
			} else {
				updatedRows, err := h.db.ExecContext(ctx, `
					UPDATE systems
					SET status = 'active', updated_at = NOW()
					WHERE system_key = $1
				`, systemKey)
				if err != nil {
					logger.Error().
						Err(err).
						Str("system_key", systemKey).
						Msg("Failed to update system to active status")
				} else {
					affected, _ := updatedRows.RowsAffected()
					if affected > 0 {
						logger.Info().
							Str("system_key", systemKey).
							Msg("Updated system to active status")
					}
					if err := h.resolveLinkFailedAlert(systemKey, orgID); err != nil {
						logger.Error().
							Err(err).
							Str("system_key", systemKey).
							Msg("Failed to resolve LinkFailed alert")
					}
				}
			}
		}
	}

	// Update systems to 'inactive' if they have old heartbeat and are currently 'active'
	queryInactive := `
		UPDATE systems s
		SET status = 'inactive', updated_at = NOW()
		FROM system_heartbeats h
		WHERE s.id = h.system_id
		AND h.last_heartbeat <= $1
	`
	resultInactive, err := h.db.Exec(queryInactive, cutoff)
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

	// Select inactive, not-deleted systems to ensure LinkFailed alerts are fired.
	// Only fire alerts for systems that have been inactive for at least 2 check intervals.
	stricterCutoff := time.Now().Add(-time.Duration(h.timeoutMinutes)*time.Minute - time.Duration(h.checkIntervalSec)*time.Second)
	queryLinkFailed := `
		SELECT s.system_key, s.organization_id
		FROM systems s
		INNER JOIN system_heartbeats h ON s.id = h.system_id
		WHERE s.status = 'inactive'
			AND h.last_heartbeat <= $1
			AND s.deleted_at IS NULL
	`
	resultLinkFailed, err := h.db.QueryContext(ctx, queryLinkFailed, stricterCutoff)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to query systems for LinkFailed alert check")
	} else {
		for resultLinkFailed.Next() {
			var systemKey, orgID string
			if err := resultLinkFailed.Scan(&systemKey, &orgID); err != nil {
				logger.Error().
					Err(err).
					Str("system_key", systemKey).
					Msg("Failed to scan inactive system row for LinkFailed alert")
				continue
			} else {
				if err := h.fireLinkFailedAlert(systemKey, orgID, cutoff); err != nil {
					logger.Error().
						Err(err).
						Str("system_key", systemKey).
						Msg("Failed to fire LinkFailed alert")
				}
			}
		}
	}

	logger.Debug().
		Time("cutoff", cutoff).
		Msg("Heartbeat status check completed")
}

func (h *HeartbeatMonitor) fireLinkFailedAlert(systemKey, orgID string, startsAt time.Time) error {
	endsAt := time.Now().Add(time.Duration(h.checkIntervalSec*3) * time.Second)
	return h.postLinkFailedAlert(systemKey, orgID, startsAt, endsAt)
}

func (h *HeartbeatMonitor) resolveLinkFailedAlert(systemKey, orgID string) error {
	// To resolve the alert, post a new alert with the same fingerprint but with an end time in the past.
	return h.postLinkFailedAlert(systemKey, orgID, time.Now().Add(-time.Duration(h.timeoutMinutes)*time.Minute), time.Now())
}

func (h *HeartbeatMonitor) postLinkFailedAlert(systemKey, orgID string, startsAt, endsAt time.Time) error {
	payload := []mimirAlert{
		{
			Labels: map[string]string{
				"alertname":  "LinkFailed",
				"severity":   "critical",
				"system_key": systemKey,
			},
			Annotations: map[string]string{
				"summary_en":     "No heartbeat received from system",
				"summary_it":     "Nessun heartbeat ricevuto dal sistema",
				"description_en": fmt.Sprintf("The system has not communicated with the server since %s. Check the service connection.", startsAt.Format(time.RFC3339)),
				"description_it": fmt.Sprintf("Il sistema non ha comunicato con il server dal %s. Verificare la connessione al servizio.", startsAt.Format(time.RFC3339)),
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
			logger.Error().Err(err).Msg("Failed to close LinkFailed response body")
		}
	}()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("mimir returned status %d", resp.StatusCode)
	}

	return nil
}
