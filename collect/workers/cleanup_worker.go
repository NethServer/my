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

	"github.com/lib/pq"
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

// retentionTier describes one age band of the exponential retention policy:
// records older than minAge but younger than maxAge keep a single representative
// per (system_id, bucket). maxAge="" means "no upper bound" (the oldest tier).
type retentionTier struct {
	name   string // for logging
	minAge string // pg interval, e.g. "7 days"
	maxAge string // pg interval, e.g. "1 month"; empty for the oldest tier
	bucket string // DATE_TRUNC unit: day | week | month | quarter
}

// inventoryRetentionTiers preserves the original semantics:
//   - 7 days – 1 month: keep 1 per day
//   - 1 month – 3 months: keep 1 per week
//   - 3 months – 1 year: keep 1 per month
//   - Older than 1 year: keep 1 per quarter
var inventoryRetentionTiers = []retentionTier{
	{name: "7d-1m", minAge: "7 days", maxAge: "1 month", bucket: "day"},
	{name: "1m-3m", minAge: "1 month", maxAge: "3 months", bucket: "week"},
	{name: "3m-1y", minAge: "3 months", maxAge: "1 year", bucket: "month"},
	{name: "1y+", minAge: "1 year", maxAge: "", bucket: "quarter"},
}

// inventoryCleanupBatchSize bounds how many rows a single DELETE materializes,
// keeping peak working memory bounded on small postgres tiers.
const inventoryCleanupBatchSize = 1000

// inventoryCleanupStmtTimeout caps each cleanup statement so a slow plan
// can't hold a connection indefinitely.
const inventoryCleanupStmtTimeout = 60 * time.Second

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
//
// Implementation: per-tier batched DELETEs with a precomputed edge set, so
// peak working memory per statement stays bounded.
func (cw *CleanupWorker) cleanupInventoryRecordsExponential(ctx context.Context, workerLogger *zerolog.Logger) error {
	edges, err := loadInventoryEdgeIDs(ctx)
	if err != nil {
		return fmt.Errorf("load edge ids: %w", err)
	}
	workerLogger.Debug().
		Int("edge_ids", len(edges)).
		Msg("Loaded first/last record ids per system")

	totalDeleted := int64(0)
	for _, tier := range inventoryRetentionTiers {
		deleted, err := deleteInventoryTier(ctx, tier, edges, workerLogger)
		if err != nil {
			return fmt.Errorf("tier %s: %w", tier.name, err)
		}
		totalDeleted += deleted
	}

	if totalDeleted > 0 {
		workerLogger.Info().
			Int64("rows_deleted", totalDeleted).
			Msg("Cleaned up inventory records (exponential retention)")
	}
	return nil
}

// loadInventoryEdgeIDs returns the set of record ids that must always be
// preserved: the first and last record per system. Computed once per cleanup
// run and reused across all tier DELETEs. id is BIGSERIAL (monotonic), so
// MIN/MAX(id) per system_id matches "first/last record" chronologically.
func loadInventoryEdgeIDs(ctx context.Context) ([]int64, error) {
	const q = `
		SELECT MIN(id) AS edge_id FROM inventory_records GROUP BY system_id
		UNION
		SELECT MAX(id) FROM inventory_records GROUP BY system_id
	`
	rows, err := database.DB.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	ids := make([]int64, 0, 1024)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// deleteInventoryTier removes records for a single age tier, batched by
// inventoryCleanupBatchSize and bounded by inventoryCleanupStmtTimeout per
// statement, looping until a batch returns fewer rows than the limit.
func deleteInventoryTier(ctx context.Context, tier retentionTier, edges []int64, workerLogger *zerolog.Logger) (int64, error) {
	// $1=bucket unit, $2=minAge interval, $3=excluded edge ids,
	// $4=batch size, and (when present) $5=maxAge interval.
	upperBound := ""
	if tier.maxAge != "" {
		upperBound = "AND created_at >= NOW() - $5::interval"
	}
	query := fmt.Sprintf(`
		DELETE FROM inventory_records
		WHERE id IN (
			SELECT id FROM (
				SELECT
					id,
					ROW_NUMBER() OVER (
						PARTITION BY system_id, DATE_TRUNC($1, created_at)
						ORDER BY created_at DESC, id DESC
					) AS rn
				FROM inventory_records
				WHERE created_at < NOW() - $2::interval
					%s
					AND id <> ALL($3::bigint[])
			) ranked
			WHERE rn > 1
			LIMIT $4
		)
	`, upperBound)

	args := []interface{}{tier.bucket, tier.minAge, pq.Array(edges), inventoryCleanupBatchSize}
	if tier.maxAge != "" {
		args = append(args, tier.maxAge)
	}

	totalDeleted := int64(0)
	for {
		stmtCtx, cancel := context.WithTimeout(ctx, inventoryCleanupStmtTimeout)
		result, err := database.DB.ExecContext(stmtCtx, query, args...)
		cancel()
		if err != nil {
			return totalDeleted, err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return totalDeleted, err
		}
		totalDeleted += n
		if n < inventoryCleanupBatchSize {
			break
		}
		// Yield to the DB between batches so concurrent traffic
		// (heartbeats, auth lookups) gets connections back.
		select {
		case <-ctx.Done():
			return totalDeleted, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}

	if totalDeleted > 0 {
		workerLogger.Debug().
			Str("tier", tier.name).
			Int64("rows_deleted", totalDeleted).
			Msg("Tier cleanup completed")
	}
	return totalDeleted, nil
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
