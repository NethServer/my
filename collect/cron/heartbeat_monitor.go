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

	collectalerting "github.com/nethesis/my/collect/alerting"
	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
)

// HeartbeatMonitor checks system heartbeats and updates status accordingly
type HeartbeatMonitor struct {
	db               *sql.DB
	timeoutMinutes   int
	checkIntervalSec int
	// postAlerts resolves LinkFailed alerts when a system recovers. Injectable
	// for tests; defaults to the real Mimir client.
	postAlerts postAlertsFunc
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
		postAlerts:       collectalerting.PostAlerts,
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
	//
	// The "to active" transition is split in two so we can react to recoveries:
	//   - inactive -> active is a RECOVERY: the system had a firing LinkFailed
	//     alert that must be resolved, so we RETURN the recovered ids.
	//   - unknown  -> active is a system reporting for the FIRST time (pending);
	//     it never fired LinkFailed, so nothing to resolve.
	// Together they cover the same rows the old `status != 'active'` update did.
	queryRecover := `
		UPDATE systems s
		SET status = 'active'
		FROM system_heartbeats h
		WHERE s.id = h.system_id
			AND h.last_heartbeat > $1
			AND s.status = 'inactive'
			AND s.deleted_at IS NULL
		RETURNING s.id::text
	`

	recoveredIDs := make([]string, 0)
	rows, err := h.db.QueryContext(ctx, queryRecover, cutoff)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to update inactive systems to active status")
	} else {
		for rows.Next() {
			var id string
			if scanErr := rows.Scan(&id); scanErr != nil {
				logger.Error().Err(scanErr).Msg("Failed to scan recovered system id")
				continue
			}
			recoveredIDs = append(recoveredIDs, id)
		}
		_ = rows.Close()
		if len(recoveredIDs) > 0 {
			logger.Info().
				Int("systems_recovered", len(recoveredIDs)).
				Msg("Recovered systems to active status")
		}
	}

	queryWake := `
		UPDATE systems s
		SET status = 'active'
		FROM system_heartbeats h
		WHERE s.id = h.system_id
			AND h.last_heartbeat > $1
			AND s.status = 'unknown'
			AND s.deleted_at IS NULL
	`
	if resultWake, wakeErr := h.db.ExecContext(ctx, queryWake, cutoff); wakeErr != nil {
		logger.Error().Err(wakeErr).Msg("Failed to update pending systems to active status")
	} else if n, _ := resultWake.RowsAffected(); n > 0 {
		logger.Info().Int64("systems_updated", n).Msg("Activated pending systems on first contact")
	}

	// Resolve LinkFailed alerts for the systems that just came back. Without this
	// the alert lingers in Mimir until its TTL — and re-fires before expiry when
	// the heartbeat interval flirts with the timeout, so it never clears.
	h.resolveRecovered(ctx, recoveredIDs)

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

// resolveRecovered posts a resolved LinkFailed alert for each system that just
// transitioned inactive -> active, batched per organization. The alert is built
// from the system's current DB labels (same path as the firing alert) so its
// fingerprint matches and Alertmanager clears the firing alert. Bounded by the
// number of recoveries per cycle; on the rare mass-recovery it is a handful of
// indexed lookups, not a fan-out over every alert.
func (h *HeartbeatMonitor) resolveRecovered(ctx context.Context, ids []string) {
	if len(ids) == 0 || h.postAlerts == nil {
		return
	}

	byOrg := make(map[string][]models.AlertmanagerPostAlert)
	for _, id := range ids {
		systemContext, err := collectalerting.LookupSystemAlertContext(ctx, h.db, id)
		if err != nil {
			// System deleted between the status flip and this lookup, or a DB
			// hiccup: skip. collect stops refreshing it anyway, so the TTL clears it.
			logger.Warn().Err(err).Str("system_id", id).
				Msg("heartbeat monitor: failed to load context for recovered system")
			continue
		}
		if systemContext.OrganizationID == "" {
			continue
		}
		alert, err := collectalerting.BuildResolvedLinkFailedAlert(systemContext)
		if err != nil {
			logger.Warn().Err(err).Str("system_key", systemContext.SystemKey).
				Msg("heartbeat monitor: failed to build resolved alert")
			continue
		}
		byOrg[systemContext.OrganizationID] = append(byOrg[systemContext.OrganizationID], alert)
	}

	resolved := 0
	for orgID, alerts := range byOrg {
		if err := h.postAlerts(orgID, alerts); err != nil {
			logger.Warn().Err(err).
				Str("organization_id", orgID).
				Int("count", len(alerts)).
				Msg("heartbeat monitor: failed to post resolved LinkFailed alerts")
			continue
		}
		resolved += len(alerts)
	}
	if resolved > 0 {
		logger.Info().Int("alerts_resolved", resolved).
			Msg("heartbeat monitor: resolved LinkFailed alerts for recovered systems")
	}
}
