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
	"github.com/rs/zerolog"
)

// ScalableManager manages workers efficiently for high-throughput scenarios
type ScalableManager struct {
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	batchProcessor *BatchProcessor
	queueManager   *queue.QueueManager
	backpressure   *BackpressureManager
	isStarted      bool
	mu             sync.RWMutex
	metrics        *WorkerMetrics
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

// NewScalableManager creates a new scalable worker manager
func NewScalableManager() *ScalableManager {
	return &ScalableManager{
		batchProcessor: NewBatchProcessor(100, 5*time.Second), // 100 items or 5 seconds
		queueManager:   queue.NewQueueManager(),
		backpressure:   NewBackpressureManager(10000, 0.8), // 10k max queue, 80% drop threshold
		metrics:        &WorkerMetrics{},
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

// Start starts the scalable worker manager
func (sm *ScalableManager) Start(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.isStarted {
		return fmt.Errorf("scalable manager already started")
	}

	sm.ctx, sm.cancel = context.WithCancel(ctx)

	// Start batch processor
	if err := sm.batchProcessor.Start(sm.ctx, &sm.wg); err != nil {
		return fmt.Errorf("failed to start batch processor: %w", err)
	}

	// Start queue monitoring
	sm.wg.Add(1)
	go sm.queueMonitor(sm.ctx, &sm.wg)

	// Start metrics collector
	sm.wg.Add(1)
	go sm.metricsCollector(sm.ctx, &sm.wg)

	// Start event-driven queue consumer (replaces always-on workers)
	sm.wg.Add(1)
	go sm.eventDrivenConsumer(sm.ctx, &sm.wg)

	sm.isStarted = true
	logger.Info().Msg("Scalable worker manager started")

	return nil
}

// Stop stops the scalable worker manager
func (sm *ScalableManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.isStarted {
		return fmt.Errorf("scalable manager not started")
	}

	logger.Info().Msg("Stopping scalable worker manager...")

	// Cancel context
	sm.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		sm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info().Msg("Scalable worker manager stopped gracefully")
	case <-time.After(30 * time.Second):
		logger.Warn().Msg("Scalable worker manager stop timeout")
	}

	sm.isStarted = false
	return nil
}

// eventDrivenConsumer consumes from queue only when items are available
func (sm *ScalableManager) eventDrivenConsumer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	logger := logger.ComponentLogger("scalable-manager")
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
			if sm.backpressure.circuitBreaker.IsOpen() {
				logger.Warn().Msg("Circuit breaker open, skipping queue consumption")
				time.Sleep(5 * time.Second)
				continue
			}

			// Consume from queue
			message, err := sm.queueManager.DequeueMessage(ctx, configuration.Config.QueueInventoryName, timeout)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to dequeue message")
				sm.backpressure.circuitBreaker.RecordFailure()
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
			if sm.shouldDropMessage(&inventoryData) {
				logger.Warn().
					Str("system_id", inventoryData.SystemID).
					Msg("Dropping message due to backpressure")
				continue
			}

			// Add to batch processor
			if err := sm.batchProcessor.AddInventory(ctx, &inventoryData); err != nil {
				logger.Error().
					Err(err).
					Str("system_id", inventoryData.SystemID).
					Msg("Failed to add inventory to batch processor")

				// Requeue message
				if requeueErr := sm.queueManager.RequeueMessage(ctx, configuration.Config.QueueInventoryName, message, err); requeueErr != nil {
					logger.Error().
						Err(requeueErr).
						Str("message_id", message.ID).
						Msg("Failed to requeue message")
				}
				continue
			}

			sm.backpressure.circuitBreaker.RecordSuccess()
		}
	}
}

// shouldDropMessage determines if a message should be dropped due to backpressure
func (sm *ScalableManager) shouldDropMessage(inventory *models.InventoryData) bool {
	// Check batch processor queue length
	stats := sm.batchProcessor.GetStats()
	queueLength := stats["queue_length"].(int)
	queueCapacity := stats["queue_capacity"].(int)

	usage := float64(queueLength) / float64(queueCapacity)
	return usage > sm.backpressure.dropThreshold
}

// queueMonitor monitors queue health and performance
func (sm *ScalableManager) queueMonitor(ctx context.Context, wg *sync.WaitGroup) {
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
			sm.logQueueStats(logger)
		}
	}
}

// logQueueStats logs queue statistics
func (sm *ScalableManager) logQueueStats(logger *zerolog.Logger) {
	stats, err := sm.queueManager.GetQueueStats(context.Background())
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get queue stats")
		return
	}

	batchStats := sm.batchProcessor.GetStats()

	logger.Info().
		Int64("pending_jobs", stats.PendingJobs).
		Int64("failed_jobs", stats.FailedJobs).
		Str("queue_health", stats.QueueHealth).
		Int("batch_queue_length", batchStats["queue_length"].(int)).
		Int("batch_queue_capacity", batchStats["queue_capacity"].(int)).
		Int64("batch_processed", batchStats["processed_count"].(int64)).
		Int64("batch_failed", batchStats["failed_count"].(int64)).
		Msg("Queue statistics")
}

// metricsCollector collects and updates system metrics
func (sm *ScalableManager) metricsCollector(ctx context.Context, wg *sync.WaitGroup) {
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
			sm.updateMetrics()
		}
	}
}

// updateMetrics updates system metrics
func (sm *ScalableManager) updateMetrics() {
	sm.metrics.mu.Lock()
	defer sm.metrics.mu.Unlock()

	// Update connection pool usage
	if connMetrics := database.GetConnectionMetrics(); connMetrics != nil {
		sm.metrics.ConnectionPoolUsage = float64(connMetrics.AcquiredConnections-connMetrics.ReleasedConnections) / 40.0 // 40 max managed connections
	}

	// Update processing rate
	batchStats := sm.batchProcessor.GetStats()
	sm.metrics.ProcessingRate = float64(batchStats["processed_count"].(int64)) / time.Since(sm.metrics.LastUpdate).Seconds()

	sm.metrics.LastUpdate = time.Now()
}

// GetStatus returns current system status
func (sm *ScalableManager) GetStatus() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sm.metrics.mu.RLock()
	defer sm.metrics.mu.RUnlock()

	return map[string]interface{}{
		"is_started":            sm.isStarted,
		"batch_processor_stats": sm.batchProcessor.GetStats(),
		"connection_pool_usage": sm.metrics.ConnectionPoolUsage,
		"processing_rate":       sm.metrics.ProcessingRate,
		"circuit_breaker_state": sm.backpressure.circuitBreaker.GetState(),
		"last_metrics_update":   sm.metrics.LastUpdate,
	}
}

// IsHealthy returns overall system health
func (sm *ScalableManager) IsHealthy() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.isStarted && sm.batchProcessor.IsHealthy() && !sm.backpressure.circuitBreaker.IsOpen()
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
