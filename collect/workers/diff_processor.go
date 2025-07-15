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
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/differ"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/queue"
	"github.com/rs/zerolog"
)

// DiffProcessor processes inventory diffs and detects changes
type DiffProcessor struct {
	id            int
	workerCount   int
	queueManager  *queue.QueueManager
	diffEngine    *differ.DiffEngine
	isHealthy     int32
	processedJobs int64
	failedJobs    int64
	lastActivity  time.Time
	mu            sync.RWMutex
}

// createDiffEngine creates a new diff engine with error handling
func createDiffEngine() *differ.DiffEngine {
	engine, err := differ.NewDefaultDiffEngine()
	if err != nil {
		logger.ComponentLogger("diff-processor").Error().
			Err(err).
			Msg("Failed to create diff engine, using fallback")
		// In case of error, we could return nil and handle it in the worker
		// For now, we'll panic since this is a critical component
		panic(fmt.Sprintf("Failed to create diff engine: %v", err))
	}
	return engine
}

// NewDiffProcessor creates a new diff processor
func NewDiffProcessor(id, workerCount int) *DiffProcessor {
	return &DiffProcessor{
		id:           id,
		workerCount:  workerCount,
		queueManager: queue.NewQueueManager(),
		diffEngine:   createDiffEngine(),
		isHealthy:    1,
		lastActivity: time.Now(),
	}
}

// Start starts the diff processor workers
func (dp *DiffProcessor) Start(ctx context.Context, wg *sync.WaitGroup) error {
	// Start multiple worker goroutines
	for i := 0; i < dp.workerCount; i++ {
		wg.Add(1)
		go dp.worker(ctx, wg, i+1)
	}

	// Start health monitor
	wg.Add(1)
	go dp.healthMonitor(ctx, wg)

	return nil
}

// Name returns the worker name
func (dp *DiffProcessor) Name() string {
	return fmt.Sprintf("diff-processor-%d", dp.id)
}

// IsHealthy returns the health status
func (dp *DiffProcessor) IsHealthy() bool {
	return atomic.LoadInt32(&dp.isHealthy) == 1
}

// worker processes diff computation messages from the queue
func (dp *DiffProcessor) worker(ctx context.Context, wg *sync.WaitGroup, workerID int) {
	defer wg.Done()

	workerLogger := logger.ComponentLogger("diff-processor").
		With().
		Int("worker_id", workerID).
		Logger()

	workerLogger.Info().Msg("Diff processor worker started")

	for {
		select {
		case <-ctx.Done():
			workerLogger.Info().Msg("Diff processor worker stopping")
			return
		default:
			// Process messages from queue
			if err := dp.processNextMessage(ctx, &workerLogger); err != nil {
				workerLogger.Error().Err(err).Msg("Error processing diff message")
				atomic.AddInt64(&dp.failedJobs, 1)

				// Brief pause on error to prevent tight error loops
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// processNextMessage processes the next message from the processing queue
func (dp *DiffProcessor) processNextMessage(ctx context.Context, workerLogger *zerolog.Logger) error {
	// Update activity timestamp
	dp.mu.Lock()
	dp.lastActivity = time.Now()
	dp.mu.Unlock()

	// Get message from queue with timeout
	message, err := dp.queueManager.DequeueMessage(ctx, configuration.Config.QueueProcessingName, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to dequeue message: %w", err)
	}

	if message == nil {
		// No message available, this is normal
		return nil
	}

	// Parse processing job
	var job models.InventoryProcessingJob
	if err := json.Unmarshal(message.Data, &job); err != nil {
		workerLogger.Error().
			Err(err).
			Str("message_id", message.ID).
			Msg("Failed to unmarshal processing job")
		return fmt.Errorf("failed to unmarshal processing job: %w", err)
	}

	// Process the diff computation
	if err := dp.processInventoryDiff(ctx, &job, workerLogger); err != nil {
		// Requeue the message for retry
		if requeueErr := dp.queueManager.RequeueMessage(ctx, configuration.Config.QueueProcessingName, message, err); requeueErr != nil {
			workerLogger.Error().
				Err(requeueErr).
				Str("message_id", message.ID).
				Msg("Failed to requeue message")
		}
		return fmt.Errorf("failed to process inventory diff: %w", err)
	}

	atomic.AddInt64(&dp.processedJobs, 1)
	workerLogger.Debug().
		Str("system_id", job.SystemID).
		Str("message_id", message.ID).
		Int64("inventory_id", job.InventoryRecord.ID).
		Msg("Inventory diff processed successfully")

	return nil
}

// processInventoryDiff computes and stores differences for an inventory record
func (dp *DiffProcessor) processInventoryDiff(ctx context.Context, job *models.InventoryProcessingJob, workerLogger *zerolog.Logger) error {
	// Add timeout to prevent hanging database operations
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second) // Longer timeout for diff computation
	defer cancel()

	start := time.Now()
	defer func() {
		workerLogger.Debug().
			Str("system_id", job.SystemID).
			Int64("inventory_id", job.InventoryRecord.ID).
			Dur("total_diff_time", time.Since(start)).
			Msg("Diff processing completed")
	}()

	// Get previous inventory record
	previousRecord, err := dp.getPreviousInventoryRecord(ctx, job.SystemID, job.InventoryRecord.ID)
	if err != nil {
		return fmt.Errorf("failed to get previous inventory record: %w", err)
	}

	if previousRecord == nil {
		// No previous record, mark as processed without diff
		if err := dp.markInventoryProcessed(ctx, job.InventoryRecord.ID, false, 0); err != nil {
			return fmt.Errorf("failed to mark inventory as processed: %w", err)
		}

		workerLogger.Info().
			Str("system_id", job.SystemID).
			Int64("inventory_id", job.InventoryRecord.ID).
			Msg("No previous inventory found, marked as processed")
		return nil
	}

	// Compute differences
	diffs, err := dp.diffEngine.ComputeDiff(job.SystemID, previousRecord, job.InventoryRecord)
	if err != nil {
		return fmt.Errorf("failed to compute diff: %w", err)
	}

	// Filter significant changes
	significantDiffs, err := differ.FilterSignificantChanges(diffs)
	if err != nil {
		return fmt.Errorf("failed to filter significant changes: %w", err)
	}

	// Store differences in database with reduced transaction time
	storedDiffs, err := dp.storeDifferences(ctx, significantDiffs)
	if err != nil {
		return fmt.Errorf("failed to store differences: %w", err)
	}

	// Mark inventory as processed
	hasChanges := len(storedDiffs) > 0
	if err := dp.markInventoryProcessed(ctx, job.InventoryRecord.ID, hasChanges, len(storedDiffs)); err != nil {
		return fmt.Errorf("failed to mark inventory as processed: %w", err)
	}

	// Queue notifications for significant changes
	if len(storedDiffs) > 0 {
		notificationJob := &models.NotificationJob{
			Type:     "diff",
			SystemID: job.SystemID,
			Diffs:    storedDiffs,
			Message:  fmt.Sprintf("Detected %d changes in system inventory", len(storedDiffs)),
			Severity: dp.determineDiffSeverity(storedDiffs),
		}

		if err := dp.queueManager.EnqueueNotification(ctx, notificationJob); err != nil {
			workerLogger.Error().
				Err(err).
				Str("system_id", job.SystemID).
				Int("diff_count", len(storedDiffs)).
				Msg("Failed to enqueue notification job")
			// Continue processing even if notification fails
		}
	}

	workerLogger.Info().
		Str("system_id", job.SystemID).
		Int64("inventory_id", job.InventoryRecord.ID).
		Int64("previous_id", previousRecord.ID).
		Int("total_diffs", len(diffs)).
		Int("significant_diffs", len(storedDiffs)).
		Bool("has_changes", hasChanges).
		Msg("Inventory diff computed and stored")

	return nil
}

// getPreviousInventoryRecord gets the most recent previous inventory record
func (dp *DiffProcessor) getPreviousInventoryRecord(ctx context.Context, systemID string, currentID int64) (*models.InventoryRecord, error) {
	query := `
		SELECT id, system_id, timestamp, data, data_hash, data_size,
		       processed_at, has_changes, change_count, created_at, updated_at
		FROM inventory_records 
		WHERE system_id = $1 AND id < $2 
		ORDER BY timestamp DESC, id DESC 
		LIMIT 1
	`

	record := &models.InventoryRecord{}
	err := database.DB.QueryRowContext(ctx, query, systemID, currentID).Scan(
		&record.ID,
		&record.SystemID,
		&record.Timestamp,
		&record.Data,
		&record.DataHash,
		&record.DataSize,
		&record.ProcessedAt,
		&record.HasChanges,
		&record.ChangeCount,
		&record.CreatedAt,
		&record.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return record, nil
}

// storeDifferences stores the computed differences in the database using batch inserts
func (dp *DiffProcessor) storeDifferences(ctx context.Context, diffs []models.InventoryDiff) ([]models.InventoryDiff, error) {
	if len(diffs) == 0 {
		return diffs, nil
	}

	// Process in batches to reduce transaction time and prevent timeouts
	const batchSize = 100
	for i := 0; i < len(diffs); i += batchSize {
		end := i + batchSize
		if end > len(diffs) {
			end = len(diffs)
		}

		batch := diffs[i:end]
		if err := dp.storeDiffBatch(ctx, batch); err != nil {
			return nil, fmt.Errorf("failed to store diff batch %d-%d: %w", i, end, err)
		}
	}

	return diffs, nil
}

// storeDiffBatch stores a batch of diffs in a single optimized transaction
func (dp *DiffProcessor) storeDiffBatch(ctx context.Context, diffs []models.InventoryDiff) error {
	if len(diffs) == 0 {
		return nil
	}

	// Add shorter timeout for batch operations
	batchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Start transaction with shorter duration
	tx, err := database.DB.BeginTx(batchCtx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Ignore error on rollback
	}()

	// Prepare statement for better performance
	stmt, err := tx.PrepareContext(batchCtx, `
		INSERT INTO inventory_diffs 
		(system_id, previous_id, current_id, diff_type, field_path, previous_value, current_value, severity, category, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		RETURNING id
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	// Insert all diffs in batch
	for i := range diffs {
		err := stmt.QueryRowContext(
			batchCtx,
			diffs[i].SystemID,
			diffs[i].PreviousID,
			diffs[i].CurrentID,
			diffs[i].DiffType,
			diffs[i].FieldPath,
			diffs[i].PreviousValue,
			diffs[i].CurrentValue,
			diffs[i].Severity,
			diffs[i].Category,
		).Scan(&diffs[i].ID)

		if err != nil {
			return fmt.Errorf("failed to insert diff: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// markInventoryProcessed marks an inventory record as processed
func (dp *DiffProcessor) markInventoryProcessed(ctx context.Context, inventoryID int64, hasChanges bool, changeCount int) error {
	// Add timeout for this operation
	updateCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		UPDATE inventory_records 
		SET processed_at = NOW(), has_changes = $2, change_count = $3, updated_at = NOW()
		WHERE id = $1
	`

	_, err := database.DB.ExecContext(updateCtx, query, inventoryID, hasChanges, changeCount)
	return err
}

// determineDiffSeverity determines the overall severity of a batch of diffs
func (dp *DiffProcessor) determineDiffSeverity(diffs []models.InventoryDiff) string {
	severityLevels := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}

	maxSeverity := "low"
	maxLevel := 1

	for _, diff := range diffs {
		if level, exists := severityLevels[diff.Severity]; exists && level > maxLevel {
			maxLevel = level
			maxSeverity = diff.Severity
		}
	}

	return maxSeverity
}

// healthMonitor monitors the health of the diff processor
func (dp *DiffProcessor) healthMonitor(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(configuration.Config.WorkerHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dp.checkHealth()
		}
	}
}

// checkHealth checks the health of the processor
func (dp *DiffProcessor) checkHealth() {
	dp.mu.RLock()
	lastActivity := dp.lastActivity
	dp.mu.RUnlock()

	// Consider unhealthy if no activity for too long
	if time.Since(lastActivity) > 5*configuration.Config.WorkerHeartbeatInterval {
		atomic.StoreInt32(&dp.isHealthy, 0)
		logger.Warn().
			Str("worker", dp.Name()).
			Time("last_activity", lastActivity).
			Msg("Worker marked as unhealthy due to inactivity")
	} else {
		atomic.StoreInt32(&dp.isHealthy, 1)
	}
}

// GetStats returns processor statistics
func (dp *DiffProcessor) GetStats() map[string]interface{} {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	return map[string]interface{}{
		"worker_count":   dp.workerCount,
		"processed_jobs": atomic.LoadInt64(&dp.processedJobs),
		"failed_jobs":    atomic.LoadInt64(&dp.failedJobs),
		"last_activity":  dp.lastActivity,
		"is_healthy":     dp.IsHealthy(),
	}
}
