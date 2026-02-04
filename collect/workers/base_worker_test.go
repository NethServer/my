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
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/logger"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Silent logger for tests to avoid JSON noise in test output
	logger.Logger = zerolog.New(io.Discard)
	log.Logger = logger.Logger
	_ = os.Setenv("DATABASE_URL", "postgres://localhost:5432/testdb")
	configuration.Init()
}

func TestNewBaseWorker(t *testing.T) {
	bw := NewBaseWorker(1, "test-worker", 5)

	assert.Equal(t, 1, bw.id)
	assert.Equal(t, "test-worker", bw.name)
	assert.Equal(t, 5, bw.workerCount)
	assert.True(t, bw.IsHealthy(), "new worker is healthy by default")
	assert.False(t, bw.lastActivity.IsZero(), "lastActivity is set on creation")
}

func TestBaseWorkerName(t *testing.T) {
	bw := NewBaseWorker(1, "my-worker", 3)
	assert.Equal(t, "my-worker", bw.Name())
}

func TestBaseWorkerIsHealthy(t *testing.T) {
	bw := NewBaseWorker(1, "test", 1)
	assert.True(t, bw.IsHealthy())

	// Mark unhealthy via atomic
	bw.isHealthy = 0
	assert.False(t, bw.IsHealthy())

	// Mark healthy again
	bw.isHealthy = 1
	assert.True(t, bw.IsHealthy())
}

func TestBaseWorkerRecordSuccess(t *testing.T) {
	bw := NewBaseWorker(1, "test", 1)

	bw.RecordSuccess()
	bw.RecordSuccess()
	bw.RecordSuccess()

	stats := bw.GetBaseStats()
	assert.Equal(t, int64(3), stats["processed_jobs"])
	assert.Equal(t, int64(0), stats["failed_jobs"])
}

func TestBaseWorkerRecordFailure(t *testing.T) {
	bw := NewBaseWorker(1, "test", 1)

	bw.RecordFailure()
	bw.RecordFailure()

	stats := bw.GetBaseStats()
	assert.Equal(t, int64(0), stats["processed_jobs"])
	assert.Equal(t, int64(2), stats["failed_jobs"])
}

func TestBaseWorkerUpdateActivity(t *testing.T) {
	bw := NewBaseWorker(1, "test", 1)
	before := bw.lastActivity

	time.Sleep(1 * time.Millisecond)
	bw.UpdateActivity()

	bw.mu.RLock()
	after := bw.lastActivity
	bw.mu.RUnlock()

	assert.True(t, after.After(before), "UpdateActivity advances the timestamp")
}

func TestBaseWorkerGetBaseStats(t *testing.T) {
	bw := NewBaseWorker(1, "stats-worker", 4)
	bw.RecordSuccess()
	bw.RecordSuccess()
	bw.RecordFailure()

	stats := bw.GetBaseStats()

	assert.Equal(t, 4, stats["worker_count"])
	assert.Equal(t, int64(2), stats["processed_jobs"])
	assert.Equal(t, int64(1), stats["failed_jobs"])
	assert.Equal(t, true, stats["is_healthy"])
	assert.NotNil(t, stats["last_activity"])
}

func TestBaseWorkerCheckHealthMarksUnhealthy(t *testing.T) {
	bw := NewBaseWorker(1, "test", 1)

	// Set lastActivity far in the past
	bw.mu.Lock()
	bw.lastActivity = time.Now().Add(-1 * time.Hour)
	bw.mu.Unlock()

	bw.checkHealth()

	assert.False(t, bw.IsHealthy(), "worker with stale activity is marked unhealthy")
}

func TestBaseWorkerCheckHealthMarksHealthy(t *testing.T) {
	bw := NewBaseWorker(1, "test", 1)

	// Ensure lastActivity is recent
	bw.UpdateActivity()
	bw.checkHealth()

	assert.True(t, bw.IsHealthy(), "worker with recent activity is healthy")
}

func TestBaseWorkerHealthMonitorStopsOnContextCancel(t *testing.T) {
	// Use a short heartbeat interval for testing
	original := configuration.Config.WorkerHeartbeatInterval
	configuration.Config.WorkerHeartbeatInterval = 10 * time.Millisecond
	defer func() { configuration.Config.WorkerHeartbeatInterval = original }()

	bw := NewBaseWorker(1, "test", 1)
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go bw.HealthMonitor(ctx, &wg)

	// Let it run a bit
	time.Sleep(50 * time.Millisecond)
	cancel()
	wg.Wait() // Should return promptly
}

func TestBaseWorkerConcurrentAccess(t *testing.T) {
	bw := NewBaseWorker(1, "concurrent-test", 1)
	var wg sync.WaitGroup
	iterations := 100

	// Concurrent RecordSuccess
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bw.RecordSuccess()
		}()
	}

	// Concurrent RecordFailure
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bw.RecordFailure()
		}()
	}

	// Concurrent UpdateActivity
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bw.UpdateActivity()
		}()
	}

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = bw.IsHealthy()
			_ = bw.GetBaseStats()
		}()
	}

	wg.Wait()

	stats := bw.GetBaseStats()
	assert.Equal(t, int64(iterations), stats["processed_jobs"])
	assert.Equal(t, int64(iterations), stats["failed_jobs"])
}
