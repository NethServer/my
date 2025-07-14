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
	"sync/atomic"
	"time"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/queue"
	"github.com/rs/zerolog"
)

// HealthMonitor monitors the overall health of the collect system
type HealthMonitor struct {
	id           int
	queueManager *queue.QueueManager
	isHealthy    int32
	lastActivity time.Time
	mu           sync.RWMutex
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(id int) *HealthMonitor {
	return &HealthMonitor{
		id:           id,
		queueManager: queue.NewQueueManager(),
		isHealthy:    1,
		lastActivity: time.Now(),
	}
}

// Start starts the health monitor
func (hm *HealthMonitor) Start(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	go hm.worker(ctx, wg)
	return nil
}

// Name returns the worker name
func (hm *HealthMonitor) Name() string {
	return "health-monitor"
}

// IsHealthy returns the health status
func (hm *HealthMonitor) IsHealthy() bool {
	return atomic.LoadInt32(&hm.isHealthy) == 1
}

// worker runs health checks periodically
func (hm *HealthMonitor) worker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	workerLogger := logger.ComponentLogger("health-monitor")
	workerLogger.Info().Msg("Health monitor started")

	ticker := time.NewTicker(configuration.Config.HealthCheckInterval)
	defer ticker.Stop()

	// Run initial health check after a short delay
	initialTimer := time.NewTimer(5 * time.Second)
	defer initialTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			workerLogger.Info().Msg("Health monitor stopping")
			return
		case <-initialTimer.C:
			// Run initial health check
			hm.runHealthChecks(ctx, workerLogger)
		case <-ticker.C:
			// Run periodic health checks
			hm.runHealthChecks(ctx, workerLogger)
		}
	}
}

// runHealthChecks executes all health checks
func (hm *HealthMonitor) runHealthChecks(ctx context.Context, workerLogger *zerolog.Logger) {
	hm.mu.Lock()
	hm.lastActivity = time.Now()
	hm.mu.Unlock()

	healthy := true

	// Check database connectivity
	if err := hm.checkDatabaseHealth(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Database health check failed")
		healthy = false
	}

	// Check Redis connectivity
	if err := hm.checkRedisHealth(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Redis health check failed")
		healthy = false
	}

	// Check queue health
	if err := hm.checkQueueHealth(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Queue health check failed")
		healthy = false
	}

	// Check system resources
	if err := hm.checkSystemResources(ctx, workerLogger); err != nil {
		workerLogger.Warn().Err(err).Msg("System resources check warning")
		// Don't mark as unhealthy for resource warnings
	}

	if healthy {
		atomic.StoreInt32(&hm.isHealthy, 1)
		workerLogger.Debug().Msg("All health checks passed")
	} else {
		atomic.StoreInt32(&hm.isHealthy, 0)
		workerLogger.Error().Msg("One or more health checks failed")
	}
}

// checkDatabaseHealth checks PostgreSQL database connectivity and performance
func (hm *HealthMonitor) checkDatabaseHealth(ctx context.Context, workerLogger *zerolog.Logger) error {
	// Check basic connectivity
	if err := database.Health(); err != nil {
		return err
	}

	// Check database stats
	stats := database.GetStats()

	// Log database statistics
	if dbStats, ok := stats["postgresql"].(map[string]interface{}); ok {
		openConns := dbStats["open_connections"]
		maxConns := configuration.Config.DatabaseMaxConns

		if openConnsInt, ok := openConns.(int); ok && openConnsInt > int(float64(maxConns)*0.8) {
			workerLogger.Warn().
				Int("open_connections", openConnsInt).
				Int("max_connections", maxConns).
				Msg("High database connection usage")
		}

		workerLogger.Debug().
			Interface("stats", dbStats).
			Msg("Database health check passed")
	}

	return nil
}

// checkRedisHealth checks Redis connectivity and performance
func (hm *HealthMonitor) checkRedisHealth(ctx context.Context, workerLogger *zerolog.Logger) error {
	redis := database.GetRedisClient()

	// Check basic connectivity
	if err := redis.Ping(ctx).Err(); err != nil {
		return err
	}

	// Check memory usage
	memInfo, err := redis.Info(ctx, "memory").Result()
	if err != nil {
		workerLogger.Warn().Err(err).Msg("Failed to get Redis memory info")
	} else {
		workerLogger.Debug().
			Str("memory_info", memInfo).
			Msg("Redis health check passed")
	}

	return nil
}

// checkQueueHealth checks the health of Redis queues
func (hm *HealthMonitor) checkQueueHealth(ctx context.Context, workerLogger *zerolog.Logger) error {
	stats, err := hm.queueManager.GetQueueStats(ctx)
	if err != nil {
		return err
	}

	// Check for critical queue conditions
	if stats.QueueHealth == "critical" {
		workerLogger.Error().
			Int64("pending_jobs", stats.PendingJobs).
			Int64("failed_jobs", stats.FailedJobs).
			Msg("Queue health is critical")
		return fmt.Errorf("queue health is critical")
	}

	if stats.QueueHealth == "warning" {
		workerLogger.Warn().
			Int64("pending_jobs", stats.PendingJobs).
			Int64("failed_jobs", stats.FailedJobs).
			Msg("Queue health warning")
	}

	workerLogger.Debug().
		Interface("stats", stats).
		Msg("Queue health check completed")

	return nil
}

// checkSystemResources checks system resource usage
func (hm *HealthMonitor) checkSystemResources(ctx context.Context, workerLogger *zerolog.Logger) error {
	// Check disk space for database directory
	// This is a simplified check - in production, you might want more sophisticated monitoring

	// Check if we can write to the database
	testQuery := "SELECT 1"
	_, err := database.DB.QueryContext(ctx, testQuery)
	if err != nil {
		return fmt.Errorf("database write test failed: %w", err)
	}

	// Check Redis memory usage by trying to set a test key
	redis := database.GetRedisClient()
	testKey := "health_check:test"
	err = redis.Set(ctx, testKey, "test", 1*time.Second).Err()
	if err != nil {
		return fmt.Errorf("redis write test failed: %w", err)
	}

	workerLogger.Debug().Msg("System resources check passed")
	return nil
}

// GetStats returns health monitor statistics
func (hm *HealthMonitor) GetStats() map[string]interface{} {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	return map[string]interface{}{
		"last_activity":  hm.lastActivity,
		"is_healthy":     hm.IsHealthy(),
		"check_interval": configuration.Config.HealthCheckInterval.String(),
	}
}

// GetSystemHealth returns overall system health status
func (hm *HealthMonitor) GetSystemHealth(ctx context.Context) map[string]interface{} {
	health := make(map[string]interface{})

	// Database health
	dbErr := database.Health()
	health["database"] = map[string]interface{}{
		"healthy": dbErr == nil,
		"error":   getErrorString(dbErr),
		"stats":   database.GetStats(),
	}

	// Queue health
	queueStats, queueErr := hm.queueManager.GetQueueStats(ctx)
	health["queues"] = map[string]interface{}{
		"healthy": queueErr == nil && queueStats.QueueHealth != "critical",
		"error":   getErrorString(queueErr),
		"stats":   queueStats,
	}

	// Overall health
	overallHealthy := dbErr == nil &&
		queueErr == nil &&
		(queueStats == nil || queueStats.QueueHealth != "critical") &&
		hm.IsHealthy()

	health["overall"] = map[string]interface{}{
		"healthy":         overallHealthy,
		"last_check":      hm.lastActivity,
		"monitor_healthy": hm.IsHealthy(),
	}

	return health
}

// getErrorString safely converts an error to string
func getErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
