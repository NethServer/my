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
	"sync"
	"sync/atomic"
	"time"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/collect/queue"
	"github.com/rs/zerolog"
)

// QueueMonitorWorker monitors queue health and performance
type QueueMonitorWorker struct {
	id              int
	queueManager    *queue.QueueManager
	inventoryWorker *InventoryWorker
	isHealthy       int32
	lastActivity    time.Time
	mu              sync.RWMutex
}

// NewQueueMonitorWorker creates a new queue monitor worker
func NewQueueMonitorWorker(id int, queueManager *queue.QueueManager, inventoryWorker *InventoryWorker) *QueueMonitorWorker {
	return &QueueMonitorWorker{
		id:              id,
		queueManager:    queueManager,
		inventoryWorker: inventoryWorker,
		isHealthy:       1,
		lastActivity:    time.Now(),
	}
}

// Start starts the queue monitor worker
func (qm *QueueMonitorWorker) Start(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	go qm.worker(ctx, wg)
	return nil
}

// Name returns the worker name
func (qm *QueueMonitorWorker) Name() string {
	return "queue-monitor-worker"
}

// IsHealthy returns health status
func (qm *QueueMonitorWorker) IsHealthy() bool {
	return atomic.LoadInt32(&qm.isHealthy) == 1
}

// worker runs queue monitoring
func (qm *QueueMonitorWorker) worker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	logger := logger.ComponentLogger("queue-monitor")
	logger.Info().Msg("Queue monitor started")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Queue monitor stopped")
			return
		case <-ticker.C:
			qm.logQueueStats(logger)
		}
	}
}

// logQueueStats logs queue statistics
func (qm *QueueMonitorWorker) logQueueStats(logger *zerolog.Logger) {
	// Update activity timestamp
	qm.mu.Lock()
	qm.lastActivity = time.Now()
	qm.mu.Unlock()

	stats, err := qm.queueManager.GetQueueStats(context.Background())
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get queue stats")
		atomic.StoreInt32(&qm.isHealthy, 0)
		return
	}

	inventoryStats := qm.inventoryWorker.GetStats()

	logger.Info().
		Int64("pending_jobs", stats.PendingJobs).
		Int64("failed_jobs", stats.FailedJobs).
		Str("queue_health", stats.QueueHealth).
		Int("batch_queue_length", inventoryStats["queue_length"].(int)).
		Int("batch_queue_capacity", inventoryStats["queue_capacity"].(int)).
		Int64("batch_processed", inventoryStats["processed_count"].(int64)).
		Int64("batch_failed", inventoryStats["failed_count"].(int64)).
		Msg("Queue statistics")

	atomic.StoreInt32(&qm.isHealthy, 1)
}

// GetStats returns queue monitor statistics
func (qm *QueueMonitorWorker) GetStats() map[string]interface{} {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	return map[string]interface{}{
		"last_activity":    qm.lastActivity,
		"is_healthy":       qm.IsHealthy(),
		"monitor_interval": "30s",
	}
}
