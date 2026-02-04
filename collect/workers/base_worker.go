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
)

// BaseWorker provides shared health monitoring and statistics tracking for workers
type BaseWorker struct {
	id            int
	name          string
	workerCount   int
	isHealthy     int32
	processedJobs int64
	failedJobs    int64
	lastActivity  time.Time
	mu            sync.RWMutex
}

// NewBaseWorker creates a new BaseWorker with the given parameters
func NewBaseWorker(id int, name string, workerCount int) BaseWorker {
	return BaseWorker{
		id:           id,
		name:         name,
		workerCount:  workerCount,
		isHealthy:    1,
		lastActivity: time.Now(),
	}
}

// IsHealthy returns the health status using atomic load
func (bw *BaseWorker) IsHealthy() bool {
	return atomic.LoadInt32(&bw.isHealthy) == 1
}

// Name returns the worker name
func (bw *BaseWorker) Name() string {
	return bw.name
}

// RecordSuccess increments the processed jobs counter
func (bw *BaseWorker) RecordSuccess() {
	atomic.AddInt64(&bw.processedJobs, 1)
}

// RecordFailure increments the failed jobs counter
func (bw *BaseWorker) RecordFailure() {
	atomic.AddInt64(&bw.failedJobs, 1)
}

// UpdateActivity updates the last activity timestamp
func (bw *BaseWorker) UpdateActivity() {
	bw.mu.Lock()
	bw.lastActivity = time.Now()
	bw.mu.Unlock()
}

// GetBaseStats returns common worker statistics
func (bw *BaseWorker) GetBaseStats() map[string]interface{} {
	bw.mu.RLock()
	defer bw.mu.RUnlock()

	return map[string]interface{}{
		"worker_count":   bw.workerCount,
		"processed_jobs": atomic.LoadInt64(&bw.processedJobs),
		"failed_jobs":    atomic.LoadInt64(&bw.failedJobs),
		"last_activity":  bw.lastActivity,
		"is_healthy":     bw.IsHealthy(),
	}
}

// HealthMonitor monitors the health of the worker based on activity
func (bw *BaseWorker) HealthMonitor(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(configuration.Config.WorkerHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bw.checkHealth()
		}
	}
}

// checkHealth checks the health of the worker based on last activity
func (bw *BaseWorker) checkHealth() {
	bw.mu.RLock()
	lastActivity := bw.lastActivity
	bw.mu.RUnlock()

	if time.Since(lastActivity) > 5*configuration.Config.WorkerHeartbeatInterval {
		atomic.StoreInt32(&bw.isHealthy, 0)
		logger.Warn().
			Str("worker", bw.name).
			Time("last_activity", lastActivity).
			Msg("Worker marked as unhealthy due to inactivity")
	} else {
		atomic.StoreInt32(&bw.isHealthy, 1)
	}
}
