/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package workers

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/rs/zerolog"
)

// CleanupWorker handles cleanup of old inventory data and maintenance tasks
type CleanupWorker struct {
	id           int
	isHealthy    int32
	lastActivity time.Time
	mu           sync.RWMutex
}

// NewCleanupWorker creates a new cleanup worker
func NewCleanupWorker(id int) *CleanupWorker {
	return &CleanupWorker{
		id:           id,
		isHealthy:    1,
		lastActivity: time.Now(),
	}
}

// Start starts the cleanup worker
func (cw *CleanupWorker) Start(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	go cw.worker(ctx, wg)
	return nil
}

// Name returns the worker name
func (cw *CleanupWorker) Name() string {
	return "cleanup-worker"
}

// IsHealthy returns health status
func (cw *CleanupWorker) IsHealthy() bool {
	return atomic.LoadInt32(&cw.isHealthy) == 1
}

// worker runs cleanup operations periodically
func (cw *CleanupWorker) worker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	workerLogger := logger.ComponentLogger("cleanup-worker").
		With().
		Int("worker_id", cw.id).
		Logger()

	workerLogger.Info().Msg("Cleanup worker started")

	// Run cleanup every hour
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Run initial cleanup after 5 minutes
	initialTimer := time.NewTimer(5 * time.Minute)
	defer initialTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			workerLogger.Info().Msg("Cleanup worker stopped")
			return

		case <-initialTimer.C:
			cw.runCleanup(ctx, &workerLogger)

		case <-ticker.C:
			cw.runCleanup(ctx, &workerLogger)
		}
	}
}

// runCleanup runs all cleanup operations
func (cw *CleanupWorker) runCleanup(ctx context.Context, workerLogger *zerolog.Logger) {
	start := time.Now()

	cw.mu.Lock()
	cw.lastActivity = time.Now()
	cw.mu.Unlock()

	workerLogger.Info().Msg("Starting cleanup operations")

	if err := cw.cleanupInventoryRecordsExponential(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Failed to cleanup inventory records")
		atomic.StoreInt32(&cw.isHealthy, 0)
		return
	}

	if err := cw.cleanupAlertHistory(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Failed to cleanup alert history")
		atomic.StoreInt32(&cw.isHealthy, 0)
		return
	}

	if err := cw.cleanupResolvedAlerts(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Failed to cleanup resolved alerts")
		atomic.StoreInt32(&cw.isHealthy, 0)
		return
	}

	if err := cw.cleanupOrphanAlertAssignments(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Failed to cleanup orphan alert assignments")
		atomic.StoreInt32(&cw.isHealthy, 0)
		return
	}

	if err := cw.vacuumAnalyze(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Failed to vacuum analyze tables")
		atomic.StoreInt32(&cw.isHealthy, 0)
		return
	}

	atomic.StoreInt32(&cw.isHealthy, 1)

	duration := time.Since(start)
	workerLogger.Info().
		Dur("duration", duration).
		Msg("Cleanup operations completed")
}

// alertHistoryRetention is the flat time-based retention horizon for
// alert_history: every row older than this is deleted outright. Unlike inventory
// snapshots, alert rows are discrete events with no "current state" to preserve,
// so a flat TTL — not exponential downsampling — is the right fit. Tune here.
const alertHistoryRetention = "6 months"

// retentionCleanupBatchSize bounds how many rows a single DELETE materializes,
// keeping peak working memory bounded on small postgres tiers.
const retentionCleanupBatchSize = 1000

// retentionCleanupStmtTimeout caps each cleanup statement so a slow plan
// can't hold a connection indefinitely.
const retentionCleanupStmtTimeout = 60 * time.Second

// inventorySystemYieldEvery sets how many systems are pruned between brief yields,
// so the per-system loop never monopolizes a DB connection.
const inventorySystemYieldEvery = 200

// cleanupInventoryRecordsExponential applies exponential retention to inventory_records,
// processing one system at a time. Each statement touches only that system's handful of
// rows (via the system_id index) — no table-wide ROW_NUMBER/sort — so peak memory stays
// tiny and constant regardless of table size. This is what lets it run safely on small
// Postgres tiers; see pruneSystemInventory for the policy.
//
// inventory_diffs are never deleted — they are the timeline source of truth and are
// self-contained (field_path, previous_value, current_value). Both FKs from
// inventory_diffs to inventory_records use ON DELETE SET NULL (and both columns are
// nullable as of migration 028), so diffs survive when a referenced snapshot is pruned:
// their previous_id/current_id is simply set to NULL.
func (cw *CleanupWorker) cleanupInventoryRecordsExponential(ctx context.Context, workerLogger *zerolog.Logger) error {
	systemIDs, err := loadInventorySystemIDs(ctx)
	if err != nil {
		return fmt.Errorf("load system ids: %w", err)
	}

	totalDeleted := int64(0)
	for i, systemID := range systemIDs {
		deleted, err := pruneSystemInventory(ctx, systemID)
		if err != nil {
			return fmt.Errorf("system %s: %w", systemID, err)
		}
		totalDeleted += deleted

		if (i+1)%inventorySystemYieldEvery == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(50 * time.Millisecond):
			}
		}
	}

	if totalDeleted > 0 {
		workerLogger.Info().
			Int("systems", len(systemIDs)).
			Int64("rows_deleted", totalDeleted).
			Msg("Cleaned up inventory records (exponential retention)")
	}
	return nil
}

// loadInventorySystemIDs returns the distinct system_ids present in inventory_records,
// backed by the (system_id, id) index (index-only scan, low memory).
func loadInventorySystemIDs(ctx context.Context) ([]string, error) {
	rows, err := database.DB.QueryContext(ctx, `SELECT DISTINCT system_id FROM inventory_records`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	ids := make([]string, 0, 1024)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// pruneSystemInventory deletes one system's redundant inventory snapshots per the
// exponential policy, in a single statement scoped to that system_id. Because it only
// ever touches one system's rows, the GROUP BY runs over a handful of rows in memory —
// no disk spill, safe on a 256MB tier.
//
// Kept per system:
//   - every record from the last 7 days (the retention floor)
//   - the latest record (MAX id) per time bucket for older records, the bucket widening
//     by age: daily (7d–1m), weekly (1m–3m), monthly (3m–1y), quarterly (>1y)
//   - the first (baseline) and last (current state) record, always
func pruneSystemInventory(ctx context.Context, systemID string) (int64, error) {
	const query = `
		DELETE FROM inventory_records
		WHERE system_id = $1
		  AND created_at < NOW() - INTERVAL '7 days'
		  AND id NOT IN (
		      SELECT MAX(id) FROM inventory_records
		      WHERE system_id = $1 AND created_at < NOW() - INTERVAL '7 days'
		      GROUP BY DATE_TRUNC(
		          CASE
		              WHEN created_at >= NOW() - INTERVAL '1 month'  THEN 'day'
		              WHEN created_at >= NOW() - INTERVAL '3 months' THEN 'week'
		              WHEN created_at >= NOW() - INTERVAL '1 year'   THEN 'month'
		              ELSE 'quarter'
		          END, created_at)
		  )
		  AND id NOT IN (
		      SELECT MIN(id) FROM inventory_records WHERE system_id = $1
		      UNION ALL
		      SELECT MAX(id) FROM inventory_records WHERE system_id = $1
		  )
	`
	stmtCtx, cancel := context.WithTimeout(ctx, retentionCleanupStmtTimeout)
	defer cancel()
	result, err := database.DB.ExecContext(stmtCtx, query, systemID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// cleanupAlertHistory enforces a flat time-based retention on alert_history:
// every row older than alertHistoryRetention is deleted. Batched by
// retentionCleanupBatchSize and bounded by retentionCleanupStmtTimeout per
// statement so the job never holds a connection for long on small postgres tiers.
func (cw *CleanupWorker) cleanupAlertHistory(ctx context.Context, workerLogger *zerolog.Logger) error {
	const query = `
		DELETE FROM alert_history
		WHERE id IN (
			SELECT id FROM alert_history
			WHERE created_at < NOW() - $1::interval
			LIMIT $2
		)
	`
	totalDeleted := int64(0)
	for {
		stmtCtx, cancel := context.WithTimeout(ctx, retentionCleanupStmtTimeout)
		result, err := database.DB.ExecContext(stmtCtx, query, alertHistoryRetention, retentionCleanupBatchSize)
		cancel()
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		totalDeleted += n
		if n < retentionCleanupBatchSize {
			break
		}
		// Yield to the DB between batches so concurrent traffic gets connections back.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}

	if totalDeleted > 0 {
		workerLogger.Info().
			Int64("rows_deleted", totalDeleted).
			Str("retention", alertHistoryRetention).
			Msg("Cleaned up alert history (time-based retention)")
	}
	return nil
}

// cleanupResolvedAlerts removes old resolved alerts
func (cw *CleanupWorker) cleanupResolvedAlerts(ctx context.Context, workerLogger *zerolog.Logger) error {
	query := `
		DELETE FROM inventory_alerts
		WHERE is_resolved = true
		AND resolved_at < NOW() - INTERVAL '30 days'
	`

	result, err := database.DB.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected > 0 {
		workerLogger.Info().
			Int64("rows_deleted", rowsAffected).
			Msg("Cleaned up resolved alerts")
	}

	return nil
}

// cleanupOrphanAlertAssignments sweeps assignments whose alert resolved but
// whose auto-release in the webhook path didn't land (e.g. the DELETE failed
// after the history insert succeeded). An assignment is orphaned when a
// resolved history row for the same (organization_id, fingerprint) was written
// after the assignment was made — a refire + re-assign updates assigned_at, so
// current work on a flapping alert is never swept. Mirrors the webhook path:
// each released row appends a system-driven `unassigned` event to the timeline.
func (cw *CleanupWorker) cleanupOrphanAlertAssignments(ctx context.Context, workerLogger *zerolog.Logger) error {
	const query = `
		WITH released AS (
		    DELETE FROM alert_assignments a
		    WHERE EXISTS (
		        SELECT 1 FROM alert_history h
		        WHERE h.organization_id = a.organization_id
		          AND h.fingerprint     = a.fingerprint
		          AND h.status          = 'resolved'
		          AND h.created_at      > a.assigned_at
		    )
		    RETURNING a.organization_id, a.fingerprint, a.assigned_user_id, a.assigned_user_name, a.assigned_user_org_id, a.assigned_user_org_name
		)
		INSERT INTO alert_activity (organization_id, fingerprint, action, details)
		SELECT organization_id, fingerprint, 'unassigned',
		       jsonb_build_object(
		           'reason', 'resolved',
		           'assigned_user_id', assigned_user_id,
		           'assigned_user_name', assigned_user_name,
		           'assigned_user_org_id', assigned_user_org_id,
		           'assigned_user_org_name', assigned_user_org_name
		       )
		FROM released
	`
	stmtCtx, cancel := context.WithTimeout(ctx, retentionCleanupStmtTimeout)
	defer cancel()
	result, err := database.DB.ExecContext(stmtCtx, query)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected > 0 {
		workerLogger.Info().
			Int64("rows_deleted", rowsAffected).
			Msg("Cleaned up orphan alert assignments (resolved but not auto-released)")
	}
	return nil
}

// vacuumAnalyze runs VACUUM ANALYZE on tables for performance optimization
func (cw *CleanupWorker) vacuumAnalyze(ctx context.Context, workerLogger *zerolog.Logger) error {
	tables := []string{
		"inventory_records",
		"inventory_diffs",
		"inventory_alerts",
		"alert_history",
	}

	for _, table := range tables {
		if _, err := database.DB.ExecContext(ctx, "VACUUM ANALYZE "+table); err != nil {
			workerLogger.Warn().
				Err(err).
				Str("table", table).
				Msg("Failed to vacuum analyze table")
			continue
		}

		workerLogger.Debug().
			Str("table", table).
			Msg("Vacuum analyze completed")
	}

	return nil
}

// GetStats returns cleanup worker statistics
func (cw *CleanupWorker) GetStats() map[string]interface{} {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	return map[string]interface{}{
		"last_activity":           cw.lastActivity,
		"is_healthy":              cw.IsHealthy(),
		"cleanup_interval":        "1h",
		"inventory_retention":     "exponential: 7d=all, 1m=daily, 3m=weekly, 1y=monthly, older=quarterly",
		"alert_history_retention": "time-based: delete older than " + alertHistoryRetention,
	}
}
