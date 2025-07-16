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

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/queue"
)

// Manager manages workers efficiently for high-throughput scenarios
type Manager struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	wg                   sync.WaitGroup
	inventoryWorker      *InventoryWorker
	diffWorker           *DiffWorker
	notificationWorker   *NotificationWorker
	cleanupWorker        *CleanupWorker
	queueMonitorWorker   *QueueMonitorWorker
	delayedMessageWorker *DelayedMessageWorker
	queueManager         *queue.QueueManager
	backpressure         *BackpressureManager
	isStarted            bool
	mu                   sync.RWMutex
	metrics              *WorkerMetrics
}

// WorkerMetrics tracks worker performance metrics
type WorkerMetrics struct {
	QueueLength         int64
	ProcessingRate      float64
	ErrorRate           float64
	ConnectionPoolUsage float64
	LastUpdate          time.Time
	mu                  sync.RWMutex
}

// BackpressureManager handles system overload scenarios
type BackpressureManager struct {
	maxQueueSize   int
	dropThreshold  float64
	circuitBreaker *CircuitBreaker
}

// CircuitBreaker prevents system overload
type CircuitBreaker struct {
	maxFailures     int
	resetTimeout    time.Duration
	failureCount    int
	lastFailureTime time.Time
	state           CircuitState
	mu              sync.RWMutex
}

type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// NewManager creates a new worker manager
func NewManager() *Manager {
	inventoryWorker := NewInventoryWorker(100, 5*time.Second)
	queueManager := queue.NewQueueManager()

	return &Manager{
		inventoryWorker:      inventoryWorker,
		diffWorker:           NewDiffWorker(2, 1),                                     // 2 ID, 1 worker for diff processing
		notificationWorker:   NewNotificationWorker(3, 2),                             // 3 ID, 2 workers for notifications
		cleanupWorker:        NewCleanupWorker(4),                                     // 4 ID for cleanup operations
		queueMonitorWorker:   NewQueueMonitorWorker(5, queueManager, inventoryWorker), // 5 ID for queue monitoring
		delayedMessageWorker: NewDelayedMessageWorker(6, queueManager),                // 6 ID for delayed message processing
		queueManager:         queueManager,
		backpressure:         NewBackpressureManager(10000, 0.8), // 10k max queue, 80% drop threshold
		metrics:              &WorkerMetrics{},
	}
}

// NewBackpressureManager creates a new backpressure manager
func NewBackpressureManager(maxQueueSize int, dropThreshold float64) *BackpressureManager {
	return &BackpressureManager{
		maxQueueSize:  maxQueueSize,
		dropThreshold: dropThreshold,
		circuitBreaker: &CircuitBreaker{
			maxFailures:  5,
			resetTimeout: 30 * time.Second,
			state:        CircuitClosed,
		},
	}
}

// Start starts the worker manager
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isStarted {
		return fmt.Errorf("scalable manager already started")
	}

	m.ctx, m.cancel = context.WithCancel(ctx)

	// Start inventory worker
	if err := m.inventoryWorker.Start(m.ctx, &m.wg); err != nil {
		return fmt.Errorf("failed to start inventory worker: %w", err)
	}

	// Start diff worker
	if err := m.diffWorker.Start(m.ctx, &m.wg); err != nil {
		return fmt.Errorf("failed to start diff worker: %w", err)
	}

	// Start notification worker
	if err := m.notificationWorker.Start(m.ctx, &m.wg); err != nil {
		return fmt.Errorf("failed to start notification worker: %w", err)
	}

	// Start cleanup worker
	if err := m.cleanupWorker.Start(m.ctx, &m.wg); err != nil {
		return fmt.Errorf("failed to start cleanup worker: %w", err)
	}

	// Start queue monitor worker
	if err := m.queueMonitorWorker.Start(m.ctx, &m.wg); err != nil {
		return fmt.Errorf("failed to start queue monitor worker: %w", err)
	}

	// Start delayed message worker
	if err := m.delayedMessageWorker.Start(m.ctx, &m.wg); err != nil {
		return fmt.Errorf("failed to start delayed message worker: %w", err)
	}

	// Start metrics collector
	m.wg.Add(1)
	go m.metricsCollector(m.ctx, &m.wg)

	// Start event-driven queue consumer (replaces always-on workers)
	m.wg.Add(1)
	go m.eventDrivenConsumer(m.ctx, &m.wg)

	m.isStarted = true
	logger.Info().Msg("Worker manager started")

	return nil
}

// Stop stops the worker manager
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isStarted {
		return fmt.Errorf("worker manager not started")
	}

	logger.Info().Msg("Stopping worker manager...")

	// Cancel context
	m.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info().Msg("Worker manager stopped gracefully")
	case <-time.After(30 * time.Second):
		logger.Warn().Msg("Worker manager stop timeout")
	}

	m.isStarted = false
	return nil
}

// eventDrivenConsumer consumes from queue only when items are available
func (m *Manager) eventDrivenConsumer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	logger := logger.ComponentLogger("manager")
	logger.Info().Msg("Event-driven consumer started")

	// Use longer timeout to reduce Redis load
	timeout := 10 * time.Second

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Event-driven consumer stopped")
			return
		default:
			// Check circuit breaker
			if m.backpressure.circuitBreaker.IsOpen() {
				logger.Warn().Msg("Circuit breaker open, skipping queue consumption")
				time.Sleep(5 * time.Second)
				continue
			}

			// Consume from queue
			message, err := m.queueManager.DequeueMessage(ctx, configuration.Config.QueueInventoryName, timeout)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to dequeue message")
				m.backpressure.circuitBreaker.RecordFailure()
				continue
			}

			if message == nil {
				// No message, continue polling
				continue
			}

			// Parse inventory data
			var inventoryData models.InventoryData
			if err := json.Unmarshal(message.Data, &inventoryData); err != nil {
				logger.Error().
					Err(err).
					Str("message_id", message.ID).
					Msg("Failed to unmarshal inventory data")
				continue
			}

			// Apply backpressure
			if m.shouldDropMessage(&inventoryData) {
				logger.Warn().
					Str("system_id", inventoryData.SystemID).
					Msg("Dropping message due to backpressure")
				continue
			}

			// Add to batch processor
			if err := m.inventoryWorker.AddInventory(ctx, &inventoryData); err != nil {
				logger.Error().
					Err(err).
					Str("system_id", inventoryData.SystemID).
					Msg("Failed to add inventory to inventory worker")

				// Requeue message
				if requeueErr := m.queueManager.RequeueMessage(ctx, configuration.Config.QueueInventoryName, message, err); requeueErr != nil {
					logger.Error().
						Err(requeueErr).
						Str("message_id", message.ID).
						Msg("Failed to requeue message")
				}
				continue
			}

			m.backpressure.circuitBreaker.RecordSuccess()
		}
	}
}

// shouldDropMessage determines if a message should be dropped due to backpressure
func (m *Manager) shouldDropMessage(inventory *models.InventoryData) bool {
	// Check inventory worker queue length
	stats := m.inventoryWorker.GetStats()
	queueLength := stats["queue_length"].(int)
	queueCapacity := stats["queue_capacity"].(int)

	usage := float64(queueLength) / float64(queueCapacity)
	return usage > m.backpressure.dropThreshold
}

// metricsCollector collects and updates system metrics
func (m *Manager) metricsCollector(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	logger := logger.ComponentLogger("metrics-collector")
	logger.Info().Msg("Metrics collector started")

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Metrics collector stopped")
			return
		case <-ticker.C:
			m.updateMetrics()
		}
	}
}

// updateMetrics updates system metrics
func (m *Manager) updateMetrics() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	// Update connection pool usage
	if connMetrics := database.GetConnectionMetrics(); connMetrics != nil {
		m.metrics.ConnectionPoolUsage = float64(connMetrics.AcquiredConnections-connMetrics.ReleasedConnections) / 40.0 // 40 max managed connections
	}

	// Update processing rate
	inventoryStats := m.inventoryWorker.GetStats()
	m.metrics.ProcessingRate = float64(inventoryStats["processed_count"].(int64)) / time.Since(m.metrics.LastUpdate).Seconds()

	m.metrics.LastUpdate = time.Now()
}

// GetStatus returns current system status
func (m *Manager) GetStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	return map[string]interface{}{
		"is_started":                   m.isStarted,
		"inventory_worker_stats":       m.inventoryWorker.GetStats(),
		"diff_worker_stats":            m.diffWorker.GetStats(),
		"notification_worker_stats":    m.notificationWorker.GetStats(),
		"cleanup_worker_stats":         m.cleanupWorker.GetStats(),
		"queue_monitor_stats":          m.queueMonitorWorker.GetStats(),
		"delayed_message_worker_stats": m.delayedMessageWorker.GetStats(),
		"connection_pool_usage":        m.metrics.ConnectionPoolUsage,
		"processing_rate":              m.metrics.ProcessingRate,
		"circuit_breaker_state":        m.backpressure.circuitBreaker.GetState(),
		"last_metrics_update":          m.metrics.LastUpdate,
	}
}

// IsHealthy returns overall system health
func (m *Manager) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.isStarted && m.inventoryWorker.IsHealthy() && m.diffWorker.IsHealthy() && m.notificationWorker.IsHealthy() && m.cleanupWorker.IsHealthy() && m.queueMonitorWorker.IsHealthy() && m.delayedMessageWorker.IsHealthy() && !m.backpressure.circuitBreaker.IsOpen()
}

// Circuit breaker methods
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.state == CircuitOpen {
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = CircuitHalfOpen
			return false
		}
		return true
	}
	return false
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.maxFailures {
		cb.state = CircuitOpen
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	cb.state = CircuitClosed
}

func (cb *CircuitBreaker) GetState() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}
