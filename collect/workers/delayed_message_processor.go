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
	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/queue"
	"github.com/rs/zerolog"
)

// DelayedMessageProcessor handles processing of delayed messages in Redis queues
type DelayedMessageProcessor struct {
	id            int
	queueManager  *queue.QueueManager
	isHealthy     int32
	processedJobs int64
	lastActivity  time.Time
	mu            sync.RWMutex
}

// NewDelayedMessageProcessor creates a new delayed message processor
func NewDelayedMessageProcessor(id int) *DelayedMessageProcessor {
	return &DelayedMessageProcessor{
		id:           id,
		queueManager: queue.NewQueueManager(),
		isHealthy:    1,
		lastActivity: time.Now(),
	}
}

// Start starts the delayed message processor
func (dmp *DelayedMessageProcessor) Start(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	go dmp.worker(ctx, wg)
	return nil
}

// Name returns the worker name
func (dmp *DelayedMessageProcessor) Name() string {
	return "delayed-message-processor"
}

// IsHealthy returns the health status
func (dmp *DelayedMessageProcessor) IsHealthy() bool {
	return atomic.LoadInt32(&dmp.isHealthy) == 1
}

// worker processes delayed messages periodically
func (dmp *DelayedMessageProcessor) worker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	workerLogger := logger.ComponentLogger("delayed-message-processor")
	workerLogger.Info().Msg("Delayed message processor started")

	// Process delayed messages every 10 seconds
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			workerLogger.Info().Msg("Delayed message processor stopping")
			return
		case <-ticker.C:
			dmp.processDelayedMessages(ctx, workerLogger)
		}
	}
}

// processDelayedMessages processes delayed messages for all queues
func (dmp *DelayedMessageProcessor) processDelayedMessages(ctx context.Context, workerLogger *zerolog.Logger) {
	dmp.mu.Lock()
	dmp.lastActivity = time.Now()
	dmp.mu.Unlock()

	queues := []string{
		configuration.Config.QueueInventoryName,
		configuration.Config.QueueProcessingName,
		configuration.Config.QueueNotificationName,
	}

	totalProcessed := int64(0)

	for _, queueName := range queues {
		if err := dmp.queueManager.ProcessDelayedMessages(ctx, queueName); err != nil {
			workerLogger.Error().
				Err(err).
				Str("queue", queueName).
				Msg("Failed to process delayed messages")

			atomic.StoreInt32(&dmp.isHealthy, 0)
			continue
		}

		totalProcessed++
	}

	if totalProcessed > 0 {
		atomic.AddInt64(&dmp.processedJobs, totalProcessed)
		atomic.StoreInt32(&dmp.isHealthy, 1)

		workerLogger.Debug().
			Int64("queues_processed", totalProcessed).
			Msg("Processed delayed messages for all queues")
	}
}

// GetStats returns processor statistics
func (dmp *DelayedMessageProcessor) GetStats() map[string]interface{} {
	dmp.mu.RLock()
	defer dmp.mu.RUnlock()

	return map[string]interface{}{
		"processed_cycles": atomic.LoadInt64(&dmp.processedJobs),
		"last_activity":    dmp.lastActivity,
		"is_healthy":       dmp.IsHealthy(),
	}
}
