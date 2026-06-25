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
	"strings"
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

// retentionTier describes one age band of an exponential retention policy:
// rows older than minAge but younger than maxAge keep a single representative
// per (partition key, bucket). maxAge="" means "no upper bound" (the oldest tier).
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
// are self-contained (field_path, previous_value, current_value). Both FKs from
// inventory_diffs to inventory_records use ON DELETE SET NULL (and both columns
// are nullable as of migration 028), so diffs survive when a referenced snapshot
// is pruned: their previous_id/current_id is simply set to NULL.
func (cw *CleanupWorker) cleanupInventoryRecordsExponential(ctx context.Context, workerLogger *zerolog.Logger) error {
	edges, err := loadEdgeIDs(ctx, "inventory_records", "system_id")
	if err != nil {
		return fmt.Errorf("load edge ids: %w", err)
	}
	workerLogger.Debug().
		Int("edge_ids", len(edges)).
		Msg("Loaded first/last record ids per system")

	totalDeleted := int64(0)
	for _, tier := range inventoryRetentionTiers {
		deleted, err := deleteRetentionTier(ctx, "inventory_records", "system_id", tier, edges, workerLogger)
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

// loadEdgeIDs returns the set of row ids that must always be preserved: the
// first and last row per partition key. Computed once per cleanup run and reused
// across all tier DELETEs. id is BIGSERIAL (monotonic), so MIN/MAX(id) per
// partition matches "first/last row" chronologically.
//
// partitionKey is a trusted, caller-supplied column list (never user input) and
// is interpolated directly into the query.
func loadEdgeIDs(ctx context.Context, table, partitionKey string) ([]int64, error) {
	q := fmt.Sprintf(`
		SELECT MIN(id) AS edge_id FROM %[1]s GROUP BY %[2]s
		UNION
		SELECT MAX(id) FROM %[1]s GROUP BY %[2]s
	`, table, partitionKey)
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

// deleteRetentionTier removes rows for a single age tier of a table, keeping one
// representative per (partitionKey, bucket). Batched by retentionCleanupBatchSize
// and bounded by retentionCleanupStmtTimeout per statement, looping until a batch
// returns fewer rows than the limit.
//
// table and partitionKey are trusted constants (never user input) and are
// interpolated into the SQL; all values are passed as bind parameters.
// edges (optional) is a set of row ids that must never be deleted; pass nil to
// skip edge protection.
func deleteRetentionTier(ctx context.Context, table, partitionKey string, tier retentionTier, edges []int64, workerLogger *zerolog.Logger) (int64, error) {
	// $1=bucket unit, $2=minAge interval; optional maxAge / edges / and the
	// batch-size limit follow, numbered dynamically.
	conds := []string{"created_at < NOW() - $2::interval"}
	args := []interface{}{tier.bucket, tier.minAge}
	argN := 3

	if tier.maxAge != "" {
		conds = append(conds, fmt.Sprintf("created_at >= NOW() - $%d::interval", argN))
		args = append(args, tier.maxAge)
		argN++
	}
	if len(edges) > 0 {
		conds = append(conds, fmt.Sprintf("id <> ALL($%d::bigint[])", argN))
		args = append(args, pq.Array(edges))
		argN++
	}
	limitIdx := argN
	args = append(args, retentionCleanupBatchSize)

	query := fmt.Sprintf(`
		DELETE FROM %[1]s
		WHERE id IN (
			SELECT id FROM (
				SELECT
					id,
					ROW_NUMBER() OVER (
						PARTITION BY %[2]s, DATE_TRUNC($1, created_at)
						ORDER BY created_at DESC, id DESC
					) AS rn
				FROM %[1]s
				WHERE %[3]s
			) ranked
			WHERE rn > 1
			LIMIT $%[4]d
		)
	`, table, partitionKey, strings.Join(conds, "\n\t\t\t\t\t"), limitIdx)

	totalDeleted := int64(0)
	for {
		stmtCtx, cancel := context.WithTimeout(ctx, retentionCleanupStmtTimeout)
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
		if n < retentionCleanupBatchSize {
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
			Str("table", table).
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
