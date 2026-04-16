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

// NewHeartbeatMonitor creates a new heartbeat monitor instance
func NewHeartbeatMonitor() *HeartbeatMonitor {
	return &HeartbeatMonitor{
		db:               database.DB,
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

// checkAndUpdateStatuses checks all system heartbeats and updates statuses.
func (h *HeartbeatMonitor) checkAndUpdateStatuses(ctx context.Context) {
	cutoff := time.Now().Add(-time.Duration(h.timeoutMinutes) * time.Minute)

	activeRows, err := h.db.QueryContext(ctx, `
		UPDATE systems s
		SET status = 'active', updated_at = NOW()
		FROM system_heartbeats h
		WHERE s.id = h.system_id
			AND h.last_heartbeat > $1
			AND s.status != 'active'
			AND s.deleted_at IS NULL
		RETURNING s.system_key
	`, cutoff)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to update systems to active status")
	} else {
		var activatedCount int64
		for activeRows.Next() {
			var systemKey string
			if err := activeRows.Scan(&systemKey); err != nil {
				logger.Error().Err(err).Msg("Failed to scan activated system key")
				continue
			}
			logger.Debug().Str("system_key", systemKey).Msg("System status changed to active")
			activatedCount++
		}
		if err := activeRows.Err(); err != nil {
			logger.Error().Err(err).Msg("Failed to iterate activated systems")
		}
		_ = activeRows.Close()
		if activatedCount > 0 {
			logger.Info().
				Int64("systems_updated", activatedCount).
				Msg("Updated systems to active status")
		}
	}

	inactiveRows, err := h.db.QueryContext(ctx, `
		UPDATE systems s
		SET status = 'inactive', updated_at = NOW()
		FROM system_heartbeats h
		WHERE s.id = h.system_id
			AND h.last_heartbeat <= $1
			AND s.status = 'active'
			AND s.deleted_at IS NULL
		RETURNING s.system_key
	`, cutoff)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to update systems to inactive status")
	} else {
		var deactivatedCount int64
		for inactiveRows.Next() {
			var systemKey string
			if err := inactiveRows.Scan(&systemKey); err != nil {
				logger.Error().Err(err).Msg("Failed to scan deactivated system key")
				continue
			}
			logger.Debug().Str("system_key", systemKey).Msg("System status changed to inactive")
			deactivatedCount++
		}
		if err := inactiveRows.Err(); err != nil {
			logger.Error().Err(err).Msg("Failed to iterate deactivated systems")
		}
		_ = inactiveRows.Close()
		if deactivatedCount > 0 {
			logger.Warn().
				Int64("systems_updated", deactivatedCount).
				Msg("Updated systems to inactive status")
		}
	}

	logger.Debug().
		Time("cutoff", cutoff).
		Msg("Heartbeat status check completed")
}
