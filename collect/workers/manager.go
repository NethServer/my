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
	"fmt"
	"sync"
	"time"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/collect/configuration"
)

// Manager coordinates all background workers
type Manager struct {
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	workers    []Worker
	started    bool
	mu         sync.RWMutex
}

// Worker interface for all background workers
type Worker interface {
	Start(ctx context.Context, wg *sync.WaitGroup) error
	Name() string
	IsHealthy() bool
}

// NewManager creates a new worker manager
func NewManager() *Manager {
	return &Manager{
		workers: make([]Worker, 0),
		started: false,
	}
}

// Start starts all workers
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("worker manager already started")
	}

	m.ctx, m.cancel = context.WithCancel(ctx)

	// Initialize workers based on configuration
	m.workers = []Worker{
		// Inventory processing workers
		NewInventoryProcessor(1, configuration.Config.WorkerInventoryCount),
		
		// Diff processing workers  
		NewDiffProcessor(2, configuration.Config.WorkerProcessingCount),
		
		// Notification workers
		NewNotificationProcessor(3, configuration.Config.WorkerNotificationCount),
		
		// Cleanup worker (single instance)
		NewCleanupWorker(4),
		
		// Delayed message processor (single instance)
		NewDelayedMessageProcessor(5),
		
		// Health monitor (single instance)
		NewHealthMonitor(6),
	}

	// Start all workers
	for _, worker := range m.workers {
		logger.Info().
			Str("worker", worker.Name()).
			Msg("Starting worker")

		if err := worker.Start(m.ctx, &m.wg); err != nil {
			logger.Error().
				Err(err).
				Str("worker", worker.Name()).
				Msg("Failed to start worker")
			
			// Cancel context and wait for already started workers
			m.cancel()
			m.wg.Wait()
			return fmt.Errorf("failed to start worker %s: %w", worker.Name(), err)
		}
	}

	m.started = true
	logger.Info().
		Int("worker_count", len(m.workers)).
		Msg("All workers started successfully")

	return nil
}

// Stop gracefully stops all workers
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return fmt.Errorf("worker manager not started")
	}

	logger.Info().Msg("Stopping all workers...")

	// Cancel context to signal workers to stop
	m.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info().Msg("All workers stopped gracefully")
	case <-time.After(configuration.Config.WorkerShutdownTimeout):
		logger.Warn().
			Dur("timeout", configuration.Config.WorkerShutdownTimeout).
			Msg("Worker shutdown timeout reached")
	}

	m.started = false
	return nil
}

// GetStatus returns the status of all workers
func (m *Manager) GetStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]interface{})
	status["started"] = m.started
	status["worker_count"] = len(m.workers)

	workers := make(map[string]interface{})
	for _, worker := range m.workers {
		workers[worker.Name()] = map[string]interface{}{
			"healthy": worker.IsHealthy(),
		}
	}
	status["workers"] = workers

	return status
}

// GetHealthyWorkerCount returns the count of healthy workers
func (m *Manager) GetHealthyWorkerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	healthy := 0
	for _, worker := range m.workers {
		if worker.IsHealthy() {
			healthy++
		}
	}
	return healthy
}

// IsHealthy returns true if all workers are healthy
func (m *Manager) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started {
		return false
	}

	for _, worker := range m.workers {
		if !worker.IsHealthy() {
			return false
		}
	}
	return true
}

// RestartUnhealthyWorkers attempts to restart workers that are not healthy
func (m *Manager) RestartUnhealthyWorkers() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return fmt.Errorf("worker manager not started")
	}

	for _, worker := range m.workers {
		if !worker.IsHealthy() {
			logger.Warn().
				Str("worker", worker.Name()).
				Msg("Attempting to restart unhealthy worker")

			// For now, we'll just log the unhealthy worker
			// In a more sophisticated implementation, we could implement
			// worker restart logic here
		}
	}

	return nil
}

// LogWorkerStats logs statistics about worker performance
func (m *Manager) LogWorkerStats() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started {
		return
	}

	healthy := 0
	for _, worker := range m.workers {
		if worker.IsHealthy() {
			healthy++
		}
	}

	logger.Info().
		Int("total_workers", len(m.workers)).
		Int("healthy_workers", healthy).
		Int("unhealthy_workers", len(m.workers)-healthy).
		Bool("overall_healthy", healthy == len(m.workers)).
		Msg("Worker statistics")
}