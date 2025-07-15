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

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/rs/zerolog"
)

// CleanupProcessor handles cleanup of old inventory data and maintenance tasks
type CleanupProcessor struct {
	id           int
	isHealthy    int32
	lastActivity time.Time
	mu           sync.RWMutex
}

// NewCleanupProcessor creates a new cleanup processor
func NewCleanupProcessor(id int) *CleanupProcessor {
	return &CleanupProcessor{
		id:           id,
		isHealthy:    1,
		lastActivity: time.Now(),
	}
}

// Start starts the cleanup processor
func (cp *CleanupProcessor) Start(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	go cp.worker(ctx, wg)
	return nil
}

// Name returns the processor name
func (cp *CleanupProcessor) Name() string {
	return "cleanup-processor"
}

// IsHealthy returns the health status
func (cp *CleanupProcessor) IsHealthy() bool {
	return atomic.LoadInt32(&cp.isHealthy) == 1
}

// worker runs the cleanup tasks periodically
func (cp *CleanupProcessor) worker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	workerLogger := logger.ComponentLogger("cleanup-worker")
	workerLogger.Info().Msg("Cleanup worker started")

	ticker := time.NewTicker(configuration.Config.InventoryCleanupInterval)
	defer ticker.Stop()

	// Run initial cleanup after a short delay
	initialTimer := time.NewTimer(30 * time.Second)
	defer initialTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			workerLogger.Info().Msg("Cleanup worker stopping")
			return
		case <-initialTimer.C:
			// Run initial cleanup
			cp.runCleanupTasks(ctx, workerLogger)
		case <-ticker.C:
			// Run periodic cleanup
			cp.runCleanupTasks(ctx, workerLogger)
		}
	}
}

// runCleanupTasks executes all cleanup tasks
func (cp *CleanupProcessor) runCleanupTasks(ctx context.Context, workerLogger *zerolog.Logger) {
	cp.mu.Lock()
	cp.lastActivity = time.Now()
	cp.mu.Unlock()

	workerLogger.Info().Msg("Running cleanup tasks")

	// Cleanup old inventory records
	if err := cp.cleanupOldInventoryRecords(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Failed to cleanup old inventory records")
		atomic.StoreInt32(&cp.isHealthy, 0)
		return
	}

	// Cleanup old inventory diffs
	if err := cp.cleanupOldInventoryDiffs(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Failed to cleanup old inventory diffs")
		atomic.StoreInt32(&cp.isHealthy, 0)
		return
	}

	// Cleanup resolved alerts
	if err := cp.cleanupResolvedAlerts(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Failed to cleanup resolved alerts")
		atomic.StoreInt32(&cp.isHealthy, 0)
		return
	}

	// Vacuum analyze databases for performance
	if err := cp.vacuumAnalyze(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Failed to vacuum analyze database")
		// Don't mark as unhealthy for vacuum failures
	}

	atomic.StoreInt32(&cp.isHealthy, 1)
	workerLogger.Info().Msg("Cleanup tasks completed successfully")
}

// cleanupOldInventoryRecords removes inventory records older than the configured age
func (cp *CleanupProcessor) cleanupOldInventoryRecords(ctx context.Context, workerLogger *zerolog.Logger) error {
	// Keep at least 10 records per system, regardless of age
	maxAgeHours := int(configuration.Config.InventoryMaxAge.Hours())
	query := fmt.Sprintf(`
		DELETE FROM inventory_records 
		WHERE id IN (
			SELECT id FROM (
				SELECT id, 
				       ROW_NUMBER() OVER (PARTITION BY system_id ORDER BY timestamp DESC) as rn
				FROM inventory_records 
				WHERE created_at < NOW() - INTERVAL '%d hours'
			) ranked 
			WHERE rn > 10
		)
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
			Int("max_age_hours", maxAgeHours).
			Msg("Cleaned up old inventory records")
	}

	return nil
}

// cleanupOldInventoryDiffs removes old diff records
func (cp *CleanupProcessor) cleanupOldInventoryDiffs(ctx context.Context, workerLogger *zerolog.Logger) error {
	// Remove diffs older than max age, but keep critical and high severity diffs longer
	maxAgeHours := int(configuration.Config.InventoryMaxAge.Hours())
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
func (cp *CleanupProcessor) cleanupResolvedAlerts(ctx context.Context, workerLogger *zerolog.Logger) error {
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
func (cp *CleanupProcessor) vacuumAnalyze(ctx context.Context, workerLogger *zerolog.Logger) error {
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
func (cp *CleanupProcessor) GetStats() map[string]interface{} {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return map[string]interface{}{
		"last_activity":    cp.lastActivity,
		"is_healthy":       cp.IsHealthy(),
		"cleanup_interval": configuration.Config.InventoryCleanupInterval.String(),
		"max_age":          configuration.Config.InventoryMaxAge.String(),
	}
}
