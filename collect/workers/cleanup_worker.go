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

	if err := cw.cleanupResolvedAlerts(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Failed to cleanup resolved alerts")
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

// cleanupInventoryRecordsExponential applies exponential retention to inventory_records.
//
// The strategy preserves historical coverage while controlling storage growth:
//   - Always keep: the first record per system (baseline) and the latest (current state)
//   - Last 7 days: keep all records (daily or more frequent granularity)
//   - 7 days – 1 month: keep 1 per day
//   - 1 month – 3 months: keep 1 per week
//   - 3 months – 1 year: keep 1 per month
//   - Older than 1 year: keep 1 per quarter
//
// inventory_diffs are never deleted — they are the timeline source of truth and
// are self-contained (field_path, previous_value, current_value). The FK from
// inventory_diffs to inventory_records uses ON DELETE SET NULL so diffs survive
// when their referenced snapshot is pruned.
func (cw *CleanupWorker) cleanupInventoryRecordsExponential(ctx context.Context, workerLogger *zerolog.Logger) error {
	// Each tier: delete records in the age window that are NOT the representative
	// for their bucket (day/week/month/quarter), and NOT the first or last per system.
	//
	// We use ROW_NUMBER() within each bucket, keeping only row 1 (most recent within bucket).
	// The first and last record per system are always excluded from deletion via a subquery.
	query := `
		DELETE FROM inventory_records
		WHERE id IN (
			SELECT id FROM (
				SELECT
					id,
					created_at,
					system_id,
					-- Bucket label varies by age tier
					CASE
						WHEN created_at >= NOW() - INTERVAL '7 days'
							THEN NULL -- never delete records from last 7 days
						WHEN created_at >= NOW() - INTERVAL '1 month'
							THEN DATE_TRUNC('day', created_at)::text
						WHEN created_at >= NOW() - INTERVAL '3 months'
							THEN DATE_TRUNC('week', created_at)::text
						WHEN created_at >= NOW() - INTERVAL '1 year'
							THEN DATE_TRUNC('month', created_at)::text
						ELSE
							DATE_TRUNC('quarter', created_at)::text
					END AS bucket,
					ROW_NUMBER() OVER (
						PARTITION BY system_id,
						CASE
							WHEN created_at >= NOW() - INTERVAL '7 days'
								THEN NULL
							WHEN created_at >= NOW() - INTERVAL '1 month'
								THEN DATE_TRUNC('day', created_at)::text
							WHEN created_at >= NOW() - INTERVAL '3 months'
								THEN DATE_TRUNC('week', created_at)::text
							WHEN created_at >= NOW() - INTERVAL '1 year'
								THEN DATE_TRUNC('month', created_at)::text
							ELSE
								DATE_TRUNC('quarter', created_at)::text
						END
						ORDER BY created_at DESC
					) AS rn
				FROM inventory_records
				-- Never touch the first or last record per system
				WHERE id NOT IN (
					SELECT DISTINCT ON (system_id) id
					FROM inventory_records
					ORDER BY system_id, created_at ASC
				)
				AND id NOT IN (
					SELECT DISTINCT ON (system_id) id
					FROM inventory_records
					ORDER BY system_id, created_at DESC
				)
			) ranked
			-- Keep row 1 per bucket (most recent in bucket), delete the rest
			-- Also skip the last-7-days tier (bucket IS NULL)
			WHERE rn > 1 AND bucket IS NOT NULL
		)
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
			Msg("Cleaned up inventory records (exponential retention)")
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

// vacuumAnalyze runs VACUUM ANALYZE on tables for performance optimization
func (cw *CleanupWorker) vacuumAnalyze(ctx context.Context, workerLogger *zerolog.Logger) error {
	tables := []string{
		"inventory_records",
		"inventory_diffs",
		"inventory_alerts",
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
		"last_activity":    cw.lastActivity,
		"is_healthy":       cw.IsHealthy(),
		"cleanup_interval": "1h",
		"retention_policy": "exponential: 7d=all, 1m=daily, 3m=weekly, 1y=monthly, older=quarterly",
	}
}
