/*
 * Copyright (C) 2025 Nethesis S.r.l.
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

	"github.com/nethesis/my/collect/configuration"
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

	// Update activity timestamp
	cw.mu.Lock()
	cw.lastActivity = time.Now()
	cw.mu.Unlock()

	workerLogger.Info().Msg("Starting cleanup operations")

	// Run cleanup operations
	if err := cw.cleanupOldInventoryRecords(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Failed to cleanup old inventory records")
		atomic.StoreInt32(&cw.isHealthy, 0)
		return
	}

	if err := cw.cleanupOldInventoryDiffs(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Failed to cleanup old inventory diffs")
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

// cleanupOldInventoryRecords removes old inventory records but keeps at least 5 per system
func (cw *CleanupWorker) cleanupOldInventoryRecords(ctx context.Context, workerLogger *zerolog.Logger) error {
	maxAge := configuration.Config.InventoryMaxAge
	maxAgeHours := int(maxAge.Hours())

	// Use a more complex query that keeps at least 5 records per system
	// even if they're older than the max age
	query := `
		DELETE FROM inventory_records 
		WHERE id IN (
			SELECT id FROM (
				SELECT id, 
					   ROW_NUMBER() OVER (PARTITION BY system_id ORDER BY created_at DESC) as row_num
				FROM inventory_records
				WHERE created_at < NOW() - INTERVAL '%d hours'
			) ranked
			WHERE row_num > 5
		)
	`

	query = fmt.Sprintf(query, maxAgeHours)

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
			Dur("max_age", maxAge).
			Msg("Cleaned up old inventory records (kept at least 5 per system)")
	}

	return nil
}

// cleanupOldInventoryDiffs removes old inventory diffs
func (cw *CleanupWorker) cleanupOldInventoryDiffs(ctx context.Context, workerLogger *zerolog.Logger) error {
	maxAge := configuration.Config.InventoryMaxAge
	maxAgeHours := int(maxAge.Hours())

	// Remove low/medium severity diffs after configured period
	query := fmt.Sprintf(`
		DELETE FROM inventory_diffs 
		WHERE created_at < NOW() - INTERVAL '%d hours'
		AND severity IN ('low', 'medium')
	`, maxAgeHours)

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
			Msg("Cleaned up old inventory diffs (low/medium severity)")
	}

	// Remove high/critical severity diffs after longer period
	extendedAgeHours := maxAgeHours * 2 // Keep high/critical diffs twice as long
	extendedQuery := fmt.Sprintf(`
		DELETE FROM inventory_diffs 
		WHERE created_at < NOW() - INTERVAL '%d hours'
		AND severity IN ('high', 'critical')
	`, extendedAgeHours)

	result, err = database.DB.ExecContext(ctx, extendedQuery)
	if err != nil {
		return err
	}

	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected > 0 {
		workerLogger.Info().
			Int64("rows_deleted", rowsAffected).
			Msg("Cleaned up old inventory diffs (high/critical severity)")
	}

	return nil
}

// cleanupResolvedAlerts removes old resolved alerts
func (cw *CleanupWorker) cleanupResolvedAlerts(ctx context.Context, workerLogger *zerolog.Logger) error {
	// Remove resolved alerts older than 30 days
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
		query := fmt.Sprintf("VACUUM ANALYZE %s", table)
		_, err := database.DB.ExecContext(ctx, query)
		if err != nil {
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
		"max_age":          configuration.Config.InventoryMaxAge.String(),
	}
}
