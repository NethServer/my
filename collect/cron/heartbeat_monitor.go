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

// Start begins the heartbeat monitoring cron job
func (h *HeartbeatMonitor) Start() {
	logger.Info().
		Int("timeout_minutes", h.timeoutMinutes).
		Int("check_interval_seconds", h.checkIntervalSec).
		Msg("Starting heartbeat monitor cron job")

	ticker := time.NewTicker(time.Duration(h.checkIntervalSec) * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	h.checkAndUpdateStatuses()

	// Then run on ticker
	for range ticker.C {
		h.checkAndUpdateStatuses()
	}
}

// checkAndUpdateStatuses checks all system heartbeats and updates statuses
func (h *HeartbeatMonitor) checkAndUpdateStatuses() {
	cutoff := time.Now().Add(-time.Duration(h.timeoutMinutes) * time.Minute)

	// Update systems to 'online' if they have recent heartbeat and are not currently 'online'
	// This handles: unknown -> online, offline -> online
	queryOnline := `
		UPDATE systems s
		SET status = 'online', updated_at = NOW()
		FROM system_heartbeats h
		WHERE s.id = h.system_id
			AND h.last_heartbeat > $1
			AND s.status != 'online'
			AND s.deleted_at IS NULL
	`

	resultOnline, err := h.db.Exec(queryOnline, cutoff)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to update systems to online status")
	} else {
		rowsAffected, _ := resultOnline.RowsAffected()
		if rowsAffected > 0 {
			logger.Info().
				Int64("systems_updated", rowsAffected).
				Msg("Updated systems to online status")
		}
	}

	// Update systems to 'offline' if they have old heartbeat and are currently 'online'
	queryOffline := `
		UPDATE systems s
		SET status = 'offline', updated_at = NOW()
		FROM system_heartbeats h
		WHERE s.id = h.system_id
			AND h.last_heartbeat <= $1
			AND s.status = 'online'
			AND s.deleted_at IS NULL
	`

	resultOffline, err := h.db.Exec(queryOffline, cutoff)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to update systems to offline status")
	} else {
		rowsAffected, _ := resultOffline.RowsAffected()
		if rowsAffected > 0 {
			logger.Warn().
				Int64("systems_updated", rowsAffected).
				Msg("Updated systems to offline status")
		}
	}

	logger.Debug().
		Time("cutoff", cutoff).
		Msg("Heartbeat status check completed")
}
