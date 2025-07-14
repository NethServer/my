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
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/queue"
	"github.com/rs/zerolog"
)

// InventoryProcessor processes incoming inventory data
type InventoryProcessor struct {
	id            int
	workerCount   int
	queueManager  *queue.QueueManager
	isHealthy     int32
	processedJobs int64
	failedJobs    int64
	lastActivity  time.Time
	mu            sync.RWMutex
}

// NewInventoryProcessor creates a new inventory processor
func NewInventoryProcessor(id, workerCount int) *InventoryProcessor {
	return &InventoryProcessor{
		id:           id,
		workerCount:  workerCount,
		queueManager: queue.NewQueueManager(),
		isHealthy:    1,
		lastActivity: time.Now(),
	}
}

// Start starts the inventory processor workers
func (ip *InventoryProcessor) Start(ctx context.Context, wg *sync.WaitGroup) error {
	// Start multiple worker goroutines
	for i := 0; i < ip.workerCount; i++ {
		wg.Add(1)
		go ip.worker(ctx, wg, i+1)
	}

	// Start health monitor
	wg.Add(1)
	go ip.healthMonitor(ctx, wg)

	return nil
}

// Name returns the worker name
func (ip *InventoryProcessor) Name() string {
	return fmt.Sprintf("inventory-processor-%d", ip.id)
}

// IsHealthy returns the health status
func (ip *InventoryProcessor) IsHealthy() bool {
	return atomic.LoadInt32(&ip.isHealthy) == 1
}

// worker processes inventory messages from the queue
func (ip *InventoryProcessor) worker(ctx context.Context, wg *sync.WaitGroup, workerID int) {
	defer wg.Done()

	workerLogger := logger.ComponentLogger("inventory-processor").
		With().
		Int("worker_id", workerID).
		Logger()

	workerLogger.Info().Msg("Inventory processor worker started")

	for {
		select {
		case <-ctx.Done():
			workerLogger.Info().Msg("Inventory processor worker stopping")
			return
		default:
			// Process messages from queue
			if err := ip.processNextMessage(ctx, &workerLogger); err != nil {
				workerLogger.Error().Err(err).Msg("Error processing inventory message")
				atomic.AddInt64(&ip.failedJobs, 1)

				// Brief pause on error to prevent tight error loops
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// processNextMessage processes the next message from the inventory queue
func (ip *InventoryProcessor) processNextMessage(ctx context.Context, workerLogger *zerolog.Logger) error {
	// Always update activity timestamp when checking queue
	ip.mu.Lock()
	ip.lastActivity = time.Now()
	ip.mu.Unlock()

	// Get message from queue with timeout
	message, err := ip.queueManager.DequeueMessage(ctx, configuration.Config.QueueInventoryName, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to dequeue message: %w", err)
	}

	if message == nil {
		// No message available, this is normal
		return nil
	}

	// Parse inventory data
	var inventoryData models.InventoryData
	if err := json.Unmarshal(message.Data, &inventoryData); err != nil {
		workerLogger.Error().
			Err(err).
			Str("message_id", message.ID).
			Msg("Failed to unmarshal inventory data")
		return fmt.Errorf("failed to unmarshal inventory data: %w", err)
	}

	// Process the inventory data
	if err := ip.processInventoryData(ctx, &inventoryData, workerLogger); err != nil {
		// Requeue the message for retry
		if requeueErr := ip.queueManager.RequeueMessage(ctx, configuration.Config.QueueInventoryName, message, err); requeueErr != nil {
			workerLogger.Error().
				Err(requeueErr).
				Str("message_id", message.ID).
				Msg("Failed to requeue message")
		}
		return fmt.Errorf("failed to process inventory data: %w", err)
	}

	atomic.AddInt64(&ip.processedJobs, 1)
	workerLogger.Debug().
		Str("system_id", inventoryData.SystemID).
		Str("message_id", message.ID).
		Msg("Inventory data processed successfully")

	return nil
}

// processInventoryData processes a single inventory data record
func (ip *InventoryProcessor) processInventoryData(ctx context.Context, inventoryData *models.InventoryData, workerLogger *zerolog.Logger) error {
	// Add timeout to prevent hanging database operations
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	start := time.Now()
	defer func() {
		workerLogger.Debug().
			Str("system_id", inventoryData.SystemID).
			Dur("total_processing_time", time.Since(start)).
			Msg("Inventory processing completed")
	}()

	// Calculate data hash for deduplication
	dataHash := ip.calculateDataHash(inventoryData.Data)

	// Check if we already have this exact data
	var existingID int64
	query := `
		SELECT id FROM inventory_records 
		WHERE system_id = $1 AND data_hash = $2 
		LIMIT 1
	`
	err := database.DB.QueryRowContext(ctx, query, inventoryData.SystemID, dataHash).Scan(&existingID)

	if err == nil {
		// Duplicate data found, skip processing
		workerLogger.Debug().
			Str("system_id", inventoryData.SystemID).
			Str("data_hash", dataHash).
			Int64("existing_id", existingID).
			Msg("Duplicate inventory data detected, skipping")
		return nil
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check for existing inventory: %w", err)
	}

	// Insert new inventory record
	inventoryRecord, err := ip.insertInventoryRecord(ctx, inventoryData, dataHash)
	if err != nil {
		return fmt.Errorf("failed to insert inventory record: %w", err)
	}

	// Find previous inventory record for this system to compute diff
	previousRecord, err := ip.getPreviousInventoryRecord(ctx, inventoryData.SystemID, inventoryRecord.ID)
	if err != nil {
		workerLogger.Warn().
			Err(err).
			Str("system_id", inventoryData.SystemID).
			Msg("Failed to get previous inventory record, will skip diff computation")
	}

	// Queue for diff processing if we have a previous record
	if previousRecord != nil {
		processingJob := &models.InventoryProcessingJob{
			InventoryRecord: inventoryRecord,
			SystemID:        inventoryData.SystemID,
			ForceProcess:    false,
		}

		if err := ip.queueManager.EnqueueProcessing(ctx, processingJob); err != nil {
			workerLogger.Error().
				Err(err).
				Str("system_id", inventoryData.SystemID).
				Int64("inventory_id", inventoryRecord.ID).
				Msg("Failed to enqueue processing job")
			// Continue processing even if queuing fails
		}
	}

	workerLogger.Info().
		Str("system_id", inventoryData.SystemID).
		Int64("inventory_id", inventoryRecord.ID).
		Int64("data_size", inventoryRecord.DataSize).
		Bool("has_previous", previousRecord != nil).
		Msg("Inventory record stored successfully")

	return nil
}

// insertInventoryRecord inserts a new inventory record into the database
func (ip *InventoryProcessor) insertInventoryRecord(ctx context.Context, inventoryData *models.InventoryData, dataHash string) (*models.InventoryRecord, error) {
	dataSize := int64(len(inventoryData.Data))

	query := `
		INSERT INTO inventory_records 
		(system_id, timestamp, data, data_hash, data_size, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	record := &models.InventoryRecord{
		SystemID:  inventoryData.SystemID,
		Timestamp: inventoryData.Timestamp,
		Data:      inventoryData.Data,
		DataHash:  dataHash,
		DataSize:  dataSize,
	}

	err := database.DB.QueryRowContext(
		ctx, query,
		record.SystemID,
		record.Timestamp,
		record.Data,
		record.DataHash,
		record.DataSize,
	).Scan(&record.ID, &record.CreatedAt, &record.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to insert inventory record: %w", err)
	}

	return record, nil
}

// getPreviousInventoryRecord gets the most recent previous inventory record
func (ip *InventoryProcessor) getPreviousInventoryRecord(ctx context.Context, systemID string, currentID int64) (*models.InventoryRecord, error) {
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

// calculateDataHash calculates SHA-256 hash of the inventory data
func (ip *InventoryProcessor) calculateDataHash(data json.RawMessage) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// healthMonitor monitors the health of the inventory processor
func (ip *InventoryProcessor) healthMonitor(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(configuration.Config.WorkerHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ip.checkHealth()
		}
	}
}

// checkHealth checks the health of the processor
func (ip *InventoryProcessor) checkHealth() {
	ip.mu.RLock()
	lastActivity := ip.lastActivity
	ip.mu.RUnlock()

	// Consider unhealthy if no activity for too long
	if time.Since(lastActivity) > 5*configuration.Config.WorkerHeartbeatInterval {
		atomic.StoreInt32(&ip.isHealthy, 0)
		logger.Warn().
			Str("worker", ip.Name()).
			Time("last_activity", lastActivity).
			Msg("Worker marked as unhealthy due to inactivity")
	} else {
		atomic.StoreInt32(&ip.isHealthy, 1)
	}
}

// GetStats returns processor statistics
func (ip *InventoryProcessor) GetStats() map[string]interface{} {
	ip.mu.RLock()
	defer ip.mu.RUnlock()

	return map[string]interface{}{
		"worker_count":   ip.workerCount,
		"processed_jobs": atomic.LoadInt64(&ip.processedJobs),
		"failed_jobs":    atomic.LoadInt64(&ip.failedJobs),
		"last_activity":  ip.lastActivity,
		"is_healthy":     ip.IsHealthy(),
	}
}
