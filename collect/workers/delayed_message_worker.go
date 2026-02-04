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

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/queue"
	"github.com/rs/zerolog"
)

// DelayedMessageWorker processes delayed messages and moves them back to main queues
type DelayedMessageWorker struct {
	id             int
	queueManager   *queue.QueueManager
	isHealthy      int32
	lastRun        time.Time
	processedCount int64
	mu             sync.RWMutex
}

// NewDelayedMessageWorker creates a new delayed message worker
func NewDelayedMessageWorker(id int, queueManager *queue.QueueManager) *DelayedMessageWorker {
	return &DelayedMessageWorker{
		id:           id,
		queueManager: queueManager,
		isHealthy:    1,
		lastRun:      time.Now(),
	}
}

// Start starts the delayed message worker
func (dmw *DelayedMessageWorker) Start(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	go dmw.worker(ctx, wg)
	return nil
}

// Name returns the worker name
func (dmw *DelayedMessageWorker) Name() string {
	return "delayed-message-worker"
}

// IsHealthy returns health status
func (dmw *DelayedMessageWorker) IsHealthy() bool {
	return atomic.LoadInt32(&dmw.isHealthy) == 1
}

// worker runs the delayed message processing
func (dmw *DelayedMessageWorker) worker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	logger := logger.ComponentLogger("delayed-message-worker")
	logger.Info().
		Int("worker_id", dmw.id).
		Msg("Delayed message worker started")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().
				Int("worker_id", dmw.id).
				Msg("Delayed message worker stopped")
			return
		case <-ticker.C:
			dmw.processDelayedMessages(ctx, logger)
		}
	}
}

// processDelayedMessages processes all delayed queues
func (dmw *DelayedMessageWorker) processDelayedMessages(ctx context.Context, logger *zerolog.Logger) {
	start := time.Now()

	dmw.mu.Lock()
	dmw.lastRun = start
	dmw.mu.Unlock()

	queues := []string{
		configuration.Config.QueueInventoryName,
		configuration.Config.QueueProcessingName,
		configuration.Config.QueueNotificationName,
	}

	successCount := 0
	errorCount := 0

	for _, queueName := range queues {
		err := dmw.queueManager.ProcessDelayedMessages(ctx, queueName)
		if err != nil {
			logger.Error().
				Err(err).
				Str("queue", queueName).
				Int("worker_id", dmw.id).
				Msg("Failed to process delayed messages")
			errorCount++
			dmw.recordError()
		} else {
			logger.Debug().
				Str("queue", queueName).
				Int("worker_id", dmw.id).
				Msg("Processed delayed messages")
			successCount++
		}
	}

	duration := time.Since(start)
	dmw.recordSuccess()

	logger.Info().
		Int("worker_id", dmw.id).
		Int("queues_processed", successCount).
		Int("queues_failed", errorCount).
		Dur("duration", duration).
		Msg("Delayed message processing cycle completed")
}

// recordSuccess records successful processing
func (dmw *DelayedMessageWorker) recordSuccess() {
	atomic.AddInt64(&dmw.processedCount, 1)
	atomic.StoreInt32(&dmw.isHealthy, 1)
}

// recordError records processing error
func (dmw *DelayedMessageWorker) recordError() {
	atomic.StoreInt32(&dmw.isHealthy, 0)
}

// GetStats returns delayed message worker statistics
func (dmw *DelayedMessageWorker) GetStats() map[string]interface{} {
	dmw.mu.RLock()
	lastRun := dmw.lastRun
	dmw.mu.RUnlock()

	return map[string]interface{}{
		"worker_id":       dmw.id,
		"processed_count": atomic.LoadInt64(&dmw.processedCount),
		"last_run":        lastRun,
		"is_healthy":      dmw.IsHealthy(),
		"check_interval":  "30s",
	}
}
