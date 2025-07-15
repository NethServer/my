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
	"crypto/sha256"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/queue"
	"github.com/rs/zerolog"
)

// BatchProcessor handles high-throughput batch operations
type BatchProcessor struct {
	inventoryBatch chan *models.InventoryData
	batchSize      int
	flushInterval  time.Duration
	stopCh         chan struct{}
	isHealthy      bool
	mu             sync.RWMutex
	processedCount int64
	failedCount    int64
	lastFlush      time.Time
	queueManager   *queue.QueueManager
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize int, flushInterval time.Duration) *BatchProcessor {
	return &BatchProcessor{
		inventoryBatch: make(chan *models.InventoryData, batchSize*2), // Buffer 2x batch size
		batchSize:      batchSize,
		flushInterval:  flushInterval,
		stopCh:         make(chan struct{}),
		isHealthy:      true,
		lastFlush:      time.Now(),
		queueManager:   queue.NewQueueManager(),
	}
}

// Start starts the batch processor
func (bp *BatchProcessor) Start(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	go bp.batchWorker(ctx, wg)
	return nil
}

// Name returns the processor name
func (bp *BatchProcessor) Name() string {
	return "batch-processor"
}

// IsHealthy returns health status
func (bp *BatchProcessor) IsHealthy() bool {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	return bp.isHealthy
}

// AddInventory adds inventory data to the batch for processing
func (bp *BatchProcessor) AddInventory(ctx context.Context, inventory *models.InventoryData) error {
	select {
	case bp.inventoryBatch <- inventory:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1 * time.Second):
		return fmt.Errorf("batch processor queue full, dropping inventory")
	}
}

// batchWorker processes inventory data in batches
func (bp *BatchProcessor) batchWorker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	logger := logger.ComponentLogger("batch-processor")
	logger.Info().
		Int("batch_size", bp.batchSize).
		Dur("flush_interval", bp.flushInterval).
		Msg("Batch processor started")

	ticker := time.NewTicker(bp.flushInterval)
	defer ticker.Stop()

	batch := make([]*models.InventoryData, 0, bp.batchSize)

	for {
		select {
		case <-ctx.Done():
			// Process remaining items in batch before stopping
			if len(batch) > 0 {
				bp.processBatch(ctx, batch, *logger)
			}
			logger.Info().Msg("Batch processor stopped")
			return

		case inventory := <-bp.inventoryBatch:
			batch = append(batch, inventory)

			// Process batch when it reaches target size
			if len(batch) >= bp.batchSize {
				bp.processBatch(ctx, batch, *logger)
				batch = batch[:0] // Reset batch
				bp.updateLastFlush()
			}

		case <-ticker.C:
			// Process batch on timer if it has items
			if len(batch) > 0 {
				bp.processBatch(ctx, batch, *logger)
				batch = batch[:0] // Reset batch
				bp.updateLastFlush()
			}
		}
	}
}

// processBatch processes a batch of inventory data
func (bp *BatchProcessor) processBatch(ctx context.Context, batch []*models.InventoryData, logger zerolog.Logger) {
	if len(batch) == 0 {
		return
	}

	start := time.Now()
	logger.Info().
		Int("batch_size", len(batch)).
		Msg("Processing inventory batch")

	// Get managed connection
	conn, err := database.GetManagedConnection(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Int("batch_size", len(batch)).
			Msg("Failed to acquire database connection for batch")
		bp.recordFailure(int64(len(batch)))
		return
	}
	defer conn.Release()

	// Process batch in transaction
	if err := bp.processBatchInTransaction(ctx, conn, batch, logger); err != nil {
		logger.Error().
			Err(err).
			Int("batch_size", len(batch)).
			Msg("Failed to process batch")
		bp.recordFailure(int64(len(batch)))
		return
	}

	duration := time.Since(start)
	bp.recordSuccess(int64(len(batch)))

	logger.Info().
		Int("batch_size", len(batch)).
		Dur("duration", duration).
		Float64("items_per_second", float64(len(batch))/duration.Seconds()).
		Msg("Batch processed successfully")
}

// processBatchInTransaction processes a batch within a single transaction
func (bp *BatchProcessor) processBatchInTransaction(ctx context.Context, conn *database.ManagedConnection, batch []*models.InventoryData, logger zerolog.Logger) error {
	// Start transaction with timeout
	txCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tx, err := conn.BeginTx(txCtx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			logger.Error().Err(err).Msg("Failed to rollback transaction")
		}
	}()

	// Prepare statement for batch insert
	stmt, err := tx.PrepareContext(txCtx, `
		INSERT INTO inventory_records 
		(system_id, timestamp, data, data_hash, data_size, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (system_id, data_hash) 
		DO UPDATE SET 
			timestamp = EXCLUDED.timestamp,
			updated_at = NOW()
		RETURNING id
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare batch insert statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close prepared statement")
		}
	}()

	// Track inserted records for diff processing
	var insertedRecords []models.InventoryRecord

	// Process each item in batch
	for _, inventory := range batch {
		dataHash := bp.calculateDataHash(inventory.Data)
		dataSize := int64(len(inventory.Data))

		var recordID int64
		err := stmt.QueryRowContext(txCtx,
			inventory.SystemID,
			inventory.Timestamp,
			inventory.Data,
			dataHash,
			dataSize,
		).Scan(&recordID)

		if err != nil {
			return fmt.Errorf("failed to insert inventory for system %s: %w", inventory.SystemID, err)
		}

		// Create record for diff processing
		insertedRecords = append(insertedRecords, models.InventoryRecord{
			ID:        recordID,
			SystemID:  inventory.SystemID,
			Timestamp: inventory.Timestamp,
			Data:      inventory.Data,
			DataHash:  dataHash,
			DataSize:  dataSize,
		})

		// Log large payloads for monitoring
		if dataSize > 1024*1024 { // 1MB
			logger.Warn().
				Str("system_id", inventory.SystemID).
				Int64("data_size", dataSize).
				Int64("record_id", recordID).
				Msg("Large inventory payload processed")
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch transaction: %w", err)
	}

	// After successful commit, trigger diff processing asynchronously
	go bp.triggerDiffProcessingAsync(ctx, insertedRecords, logger)

	return nil
}

// calculateDataHash calculates SHA-256 hash of inventory data
func (bp *BatchProcessor) calculateDataHash(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// updateLastFlush updates the last flush timestamp
func (bp *BatchProcessor) updateLastFlush() {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.lastFlush = time.Now()
}

// recordSuccess records successful batch processing
func (bp *BatchProcessor) recordSuccess(count int64) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.processedCount += count
	bp.isHealthy = true
}

// recordFailure records failed batch processing
func (bp *BatchProcessor) recordFailure(count int64) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.failedCount += count
	bp.isHealthy = false
}

// GetStats returns batch processor statistics
func (bp *BatchProcessor) GetStats() map[string]interface{} {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	return map[string]interface{}{
		"processed_count": bp.processedCount,
		"failed_count":    bp.failedCount,
		"queue_length":    len(bp.inventoryBatch),
		"queue_capacity":  cap(bp.inventoryBatch),
		"last_flush":      bp.lastFlush,
		"is_healthy":      bp.isHealthy,
		"batch_size":      bp.batchSize,
		"flush_interval":  bp.flushInterval,
	}
}

// triggerDiffProcessingAsync triggers diff processing asynchronously for inserted inventory records
func (bp *BatchProcessor) triggerDiffProcessingAsync(ctx context.Context, insertedRecords []models.InventoryRecord, logger zerolog.Logger) {
	// Create a new context with timeout to prevent goroutine leaks
	asyncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	successCount := 0
	failedCount := 0

	for _, record := range insertedRecords {
		// Check if there's a previous record for this system
		previousRecord, err := bp.getPreviousInventoryRecord(asyncCtx, record.SystemID, record.ID)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("system_id", record.SystemID).
				Int64("record_id", record.ID).
				Msg("Failed to get previous inventory record for diff processing")
			failedCount++
			continue
		}

		// Only trigger diff processing if there's a previous record
		if previousRecord != nil {
			processingJob := &models.InventoryProcessingJob{
				InventoryRecord: &record,
				SystemID:        record.SystemID,
				ForceProcess:    false,
			}

			// Use a timeout context for each enqueue operation
			enqueueCtx, enqueueCancel := context.WithTimeout(asyncCtx, 30*time.Second)
			err := bp.queueManager.EnqueueProcessing(enqueueCtx, processingJob)
			enqueueCancel()

			if err != nil {
				logger.Warn().
					Err(err).
					Str("system_id", record.SystemID).
					Int64("inventory_id", record.ID).
					Msg("Failed to enqueue processing job for diff computation (non-blocking)")
				failedCount++
			} else {
				logger.Debug().
					Str("system_id", record.SystemID).
					Int64("inventory_id", record.ID).
					Int64("previous_id", previousRecord.ID).
					Msg("Queued diff processing job")
				successCount++
			}
		}
	}

	logger.Info().
		Int("total_records", len(insertedRecords)).
		Int("diff_jobs_queued", successCount).
		Int("diff_jobs_failed", failedCount).
		Msg("Async diff processing completed")
}

// getPreviousInventoryRecord gets the most recent previous inventory record for a system
func (bp *BatchProcessor) getPreviousInventoryRecord(ctx context.Context, systemID string, currentID int64) (*models.InventoryRecord, error) {
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

	if err == sql.ErrNoRows {
		return nil, nil // No previous record found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get previous inventory record: %w", err)
	}

	return record, nil
}
