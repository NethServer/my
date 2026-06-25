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
	"context"
	"database/sql"
	"time"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
)

// HeartbeatMonitor checks system heartbeats and updates status accordingly
type HeartbeatMonitor struct {
	db               *sql.DB
	timeoutMinutes   int
	checkIntervalSec int
}

// defaultHeartbeatCheckIntervalSec is used when the interval is unconfigured
// (config not loaded, e.g. in tests) so the ticker never gets a non-positive
// duration.
const defaultHeartbeatCheckIntervalSec = 300

// NewHeartbeatMonitor creates a new heartbeat monitor instance
func NewHeartbeatMonitor() *HeartbeatMonitor {
	interval := configuration.Config.HeartbeatCheckIntervalSeconds
	if interval <= 0 {
		interval = defaultHeartbeatCheckIntervalSec
	}
	return &HeartbeatMonitor{
		db:               database.DB,
		timeoutMinutes:   configuration.Config.HeartbeatTimeoutMinutes,
		checkIntervalSec: interval,
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

// checkAndUpdateStatuses checks all system heartbeats and updates statuses.
func (h *HeartbeatMonitor) checkAndUpdateStatuses(ctx context.Context) {
	cutoff := time.Now().Add(-time.Duration(h.timeoutMinutes) * time.Minute)

	// Note: updated_at is intentionally NOT touched here. Status is a
	// system-driven liveness flag, not a user edit; bumping updated_at on every
	// flip would churn idx_systems_updated_at each cycle for no benefit.
	queryActive := `
		UPDATE systems s
		SET status = 'active'
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
		SET status = 'inactive'
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

	logger.Debug().
		Time("cutoff", cutoff).
		Msg("Heartbeat status check completed")
}
