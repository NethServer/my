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
	"time"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/differ"
	"github.com/nethesis/my/collect/helpers"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/queue"
	"github.com/rs/zerolog"
)

// DiffWorker processes inventory diffs and detects changes
type DiffWorker struct {
	BaseWorker
	queueManager *queue.QueueManager
	diffEngine   *differ.DiffEngine
}

// createDiffEngine creates a new diff engine with error handling
func createDiffEngine() *differ.DiffEngine {
	engine, err := differ.NewDefaultDiffEngine()
	if err != nil {
		logger.ComponentLogger("diff-worker").Error().
			Err(err).
			Msg("Failed to create diff engine, using fallback")
		// In case of error, we could return nil and handle it in the worker
		// For now, we'll panic since this is a critical component
		panic(fmt.Sprintf("Failed to create diff engine: %v", err))
	}
	return engine
}

// NewDiffWorker creates a new diff worker
func NewDiffWorker(id, workerCount int, queueManager *queue.QueueManager) *DiffWorker {
	return &DiffWorker{
		BaseWorker:   NewBaseWorker(id, fmt.Sprintf("diff-worker-%d", id), workerCount),
		queueManager: queueManager,
		diffEngine:   createDiffEngine(),
	}
}

// Start starts the diff worker workers
func (dw *DiffWorker) Start(ctx context.Context, wg *sync.WaitGroup) error {
	// Start multiple worker goroutines
	for i := 0; i < dw.workerCount; i++ {
		wg.Add(1)
		go dw.worker(ctx, wg, i+1)
	}

	// Start health monitor
	wg.Add(1)
	go dw.HealthMonitor(ctx, wg)

	return nil
}

// worker processes diff computation messages from the queue
func (dw *DiffWorker) worker(ctx context.Context, wg *sync.WaitGroup, workerID int) {
	defer wg.Done()

	workerLogger := logger.ComponentLogger("diff-worker").
		With().
		Int("worker_id", workerID).
		Logger()

	workerLogger.Info().Msg("Diff worker started")

	for {
		select {
		case <-ctx.Done():
			workerLogger.Info().Msg("Diff worker stopping")
			return
		default:
			// Process messages from queue
			if err := dw.processNextMessage(ctx, &workerLogger); err != nil {
				workerLogger.Error().Err(err).Msg("Error processing diff message")
				dw.RecordFailure()

				// Brief pause on error to prevent tight error loops
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// processNextMessage processes the next message from the processing queue
func (dw *DiffWorker) processNextMessage(ctx context.Context, workerLogger *zerolog.Logger) error {
	// Update activity timestamp
	dw.UpdateActivity()

	// Get message from queue with timeout
	message, err := dw.queueManager.DequeueMessage(ctx, configuration.Config.QueueProcessingName, 5*time.Second)
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
	if err := dw.processInventoryDiff(ctx, &job, workerLogger); err != nil {
		// Requeue the message for retry
		if requeueErr := dw.queueManager.RequeueMessage(ctx, configuration.Config.QueueProcessingName, message, err); requeueErr != nil {
			workerLogger.Error().
				Err(requeueErr).
				Str("message_id", message.ID).
				Msg("Failed to requeue message")
		}
		return fmt.Errorf("failed to process inventory diff: %w", err)
	}

	dw.RecordSuccess()
	workerLogger.Debug().
		Str("system_id", job.SystemID).
		Str("message_id", message.ID).
		Int64("inventory_id", job.InventoryRecord.ID).
		Msg("Inventory diff processed successfully")

	return nil
}

// processInventoryDiff computes and stores differences for an inventory record
func (dw *DiffWorker) processInventoryDiff(ctx context.Context, job *models.InventoryProcessingJob, workerLogger *zerolog.Logger) error {
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

	// Load the full current record from DB (the job only carries ID and SystemID)
	currentRecord, err := GetInventoryRecordByID(ctx, job.InventoryRecord.ID)
	if err != nil {
		return fmt.Errorf("failed to load current inventory record: %w", err)
	}

	// Get previous inventory record
	previousRecord, err := GetPreviousInventoryRecord(ctx, job.SystemID, currentRecord.ID)
	if err != nil {
		return fmt.Errorf("failed to get previous inventory record: %w", err)
	}

	if previousRecord == nil {
		// No previous record, mark as processed without diff
		if err := dw.markInventoryProcessed(ctx, currentRecord.ID, false, 0); err != nil {
			return fmt.Errorf("failed to mark inventory as processed: %w", err)
		}

		workerLogger.Info().
			Str("system_id", job.SystemID).
			Int64("inventory_id", currentRecord.ID).
			Msg("No previous inventory found, marked as processed")
		return nil
	}

	// Compute differences
	diffs, err := dw.diffEngine.ComputeDiff(job.SystemID, previousRecord, currentRecord)
	if err != nil {
		return fmt.Errorf("failed to compute diff: %w", err)
	}

	// Filter significant changes
	significantDiffs, err := differ.FilterSignificantChanges(diffs)
	if err != nil {
		return fmt.Errorf("failed to filter significant changes: %w", err)
	}

	// Store differences in database with reduced transaction time
	storedDiffs, err := dw.storeDifferences(ctx, significantDiffs)
	if err != nil {
		return fmt.Errorf("failed to store differences: %w", err)
	}

	// Mark inventory as processed
	hasChanges := len(storedDiffs) > 0
	if err := dw.markInventoryProcessed(ctx, currentRecord.ID, hasChanges, len(storedDiffs)); err != nil {
		return fmt.Errorf("failed to mark inventory as processed: %w", err)
	}

	// Queue notifications for significant changes
	if len(storedDiffs) > 0 {
		notificationJob := &models.NotificationJob{
			Type:     "diff",
			SystemID: job.SystemID,
			Diffs:    storedDiffs,
			Message:  fmt.Sprintf("Detected %d changes in system inventory", len(storedDiffs)),
			Severity: dw.determineDiffSeverity(storedDiffs),
		}

		if err := dw.queueManager.EnqueueNotification(ctx, notificationJob); err != nil {
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
		Int64("inventory_id", currentRecord.ID).
		Int64("previous_id", previousRecord.ID).
		Int("total_diffs", len(diffs)).
		Int("significant_diffs", len(storedDiffs)).
		Bool("has_changes", hasChanges).
		Msg("Inventory diff computed and stored")

	return nil
}

// storeDifferences stores the computed differences in the database using batch inserts
func (dw *DiffWorker) storeDifferences(ctx context.Context, diffs []models.InventoryDiff) ([]models.InventoryDiff, error) {
	if len(diffs) == 0 {
		return diffs, nil
	}

	if err := helpers.ProcessInBatches(diffs, 100, func(batch []models.InventoryDiff) error {
		return dw.storeDiffBatch(ctx, batch)
	}); err != nil {
		return nil, err
	}

	return diffs, nil
}

// storeDiffBatch stores a batch of diffs in a single optimized transaction
func (dw *DiffWorker) storeDiffBatch(ctx context.Context, diffs []models.InventoryDiff) error {
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
func (dw *DiffWorker) markInventoryProcessed(ctx context.Context, inventoryID int64, hasChanges bool, changeCount int) error {
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
func (dw *DiffWorker) determineDiffSeverity(diffs []models.InventoryDiff) string {
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

// GetStats returns worker statistics
func (dw *DiffWorker) GetStats() map[string]interface{} {
	return dw.GetBaseStats()
}
