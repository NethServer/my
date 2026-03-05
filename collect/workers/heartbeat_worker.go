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

	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/queue"
)

// HeartbeatWorker processes heartbeat updates from Redis queue in batches
type HeartbeatWorker struct {
	id             int
	batchSize      int
	flushInterval  time.Duration
	queueManager   *queue.QueueManager
	isHealthy      int32
	processedCount int64
	failedCount    int64
}

// NewHeartbeatWorker creates a new heartbeat worker
func NewHeartbeatWorker(id int, batchSize int, flushInterval time.Duration, queueManager *queue.QueueManager) *HeartbeatWorker {
	return &HeartbeatWorker{
		id:            id,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		queueManager:  queueManager,
		isHealthy:     1,
	}
}

// Start starts the heartbeat worker
func (hw *HeartbeatWorker) Start(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	go hw.run(ctx, wg)
	return nil
}

// Name returns the worker name
func (hw *HeartbeatWorker) Name() string {
	return "heartbeat-worker"
}

// IsHealthy returns health status
func (hw *HeartbeatWorker) IsHealthy() bool {
	return atomic.LoadInt32(&hw.isHealthy) == 1
}

// GetStats returns heartbeat worker statistics
func (hw *HeartbeatWorker) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"processed_count": atomic.LoadInt64(&hw.processedCount),
		"failed_count":    atomic.LoadInt64(&hw.failedCount),
		"is_healthy":      hw.IsHealthy(),
		"batch_size":      hw.batchSize,
		"flush_interval":  hw.flushInterval,
	}
}

// run is the main loop that polls Redis and flushes batches to DB
func (hw *HeartbeatWorker) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	log := logger.ComponentLogger("heartbeat-worker")
	log.Info().
		Int("batch_size", hw.batchSize).
		Dur("flush_interval", hw.flushInterval).
		Msg("Heartbeat worker started")

	ticker := time.NewTicker(hw.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Drain remaining items before stopping
			hw.drainQueue(ctx)
			log.Info().Msg("Heartbeat worker stopped")
			return

		case <-ticker.C:
			hw.processBatch(ctx)
		}
	}
}

// processBatch dequeues a batch from Redis and writes to DB
func (hw *HeartbeatWorker) processBatch(ctx context.Context) {
	log := logger.ComponentLogger("heartbeat-worker")

	heartbeats, err := hw.queueManager.DequeueHeartbeatBatch(ctx, hw.batchSize)
	if err != nil {
		log.Error().Err(err).Msg("Failed to dequeue heartbeat batch")
		atomic.StoreInt32(&hw.isHealthy, 0)
		return
	}

	if len(heartbeats) == 0 {
		return
	}

	// Deduplicate: keep only the latest heartbeat per system_id
	latest := make(map[string]*models.SystemHeartbeat)
	for _, hb := range heartbeats {
		if existing, ok := latest[hb.SystemID]; !ok || hb.LastHeartbeat.After(existing.LastHeartbeat) {
			latest[hb.SystemID] = hb
		}
	}

	if err := hw.bulkUpsert(ctx, latest); err != nil {
		log.Error().Err(err).Int("count", len(latest)).Msg("Failed to bulk upsert heartbeats")
		atomic.AddInt64(&hw.failedCount, int64(len(latest)))
		atomic.StoreInt32(&hw.isHealthy, 0)
		return
	}

	atomic.AddInt64(&hw.processedCount, int64(len(latest)))
	atomic.StoreInt32(&hw.isHealthy, 1)

	log.Debug().
		Int("dequeued", len(heartbeats)).
		Int("unique_systems", len(latest)).
		Msg("Heartbeat batch processed")
}

// bulkUpsert writes multiple heartbeats in a single SQL statement
func (hw *HeartbeatWorker) bulkUpsert(ctx context.Context, heartbeats map[string]*models.SystemHeartbeat) error {
	if len(heartbeats) == 0 {
		return nil
	}

	// Build bulk INSERT ... ON CONFLICT query
	var sb strings.Builder
	sb.WriteString("INSERT INTO system_heartbeats (system_id, last_heartbeat) VALUES ")

	args := make([]interface{}, 0, len(heartbeats)*2)
	i := 0
	for _, hb := range heartbeats {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "($%d, $%d)", i*2+1, i*2+2)
		args = append(args, hb.SystemID, hb.LastHeartbeat)
		i++
	}

	sb.WriteString(" ON CONFLICT (system_id) DO UPDATE SET last_heartbeat = EXCLUDED.last_heartbeat")

	_, err := database.DB.ExecContext(ctx, sb.String(), args...)
	return err
}

// drainQueue processes all remaining items in the queue
func (hw *HeartbeatWorker) drainQueue(_ context.Context) {
	for {
		drainCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		heartbeats, err := hw.queueManager.DequeueHeartbeatBatch(drainCtx, hw.batchSize)
		cancel()
		if err != nil || len(heartbeats) == 0 {
			return
		}

		latest := make(map[string]*models.SystemHeartbeat)
		for _, hb := range heartbeats {
			if existing, ok := latest[hb.SystemID]; !ok || hb.LastHeartbeat.After(existing.LastHeartbeat) {
				latest[hb.SystemID] = hb
			}
		}

		upsertCtx, upsertCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = hw.bulkUpsert(upsertCtx, latest)
		upsertCancel()
	}
}
