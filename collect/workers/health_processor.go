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

// HealthProcessor monitors the overall health of the collect system
type HealthProcessor struct {
	id           int
	queueManager *queue.QueueManager
	isHealthy    int32
	lastActivity time.Time
	mu           sync.RWMutex
}

// NewHealthProcessor creates a new health processor
func NewHealthProcessor(id int) *HealthProcessor {
	return &HealthProcessor{
		id:           id,
		queueManager: queue.NewQueueManager(),
		isHealthy:    1,
		lastActivity: time.Now(),
	}
}

// Start starts the health processor
func (hp *HealthProcessor) Start(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	go hp.worker(ctx, wg)
	return nil
}

// Name returns the processor name
func (hp *HealthProcessor) Name() string {
	return "health-processor"
}

// IsHealthy returns the health status
func (hp *HealthProcessor) IsHealthy() bool {
	return atomic.LoadInt32(&hp.isHealthy) == 1
}

// worker runs health checks periodically
func (hp *HealthProcessor) worker(ctx context.Context, wg *sync.WaitGroup) {
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
			hp.runHealthChecks(ctx, workerLogger)
		case <-ticker.C:
			// Run periodic health checks
			hp.runHealthChecks(ctx, workerLogger)
		}
	}
}

// runHealthChecks executes all health checks
func (hp *HealthProcessor) runHealthChecks(ctx context.Context, workerLogger *zerolog.Logger) {
	hp.mu.Lock()
	hp.lastActivity = time.Now()
	hp.mu.Unlock()

	healthy := true

	// Check database connectivity
	if err := hp.checkDatabaseHealth(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Database health check failed")
		healthy = false
	}

	// Check Redis connectivity
	if err := hp.checkRedisHealth(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Redis health check failed")
		healthy = false
	}

	// Check queue health
	if err := hp.checkQueueHealth(ctx, workerLogger); err != nil {
		workerLogger.Error().Err(err).Msg("Queue health check failed")
		healthy = false
	}

	// Check system resources
	if err := hp.checkSystemResources(ctx, workerLogger); err != nil {
		workerLogger.Warn().Err(err).Msg("System resources check warning")
		// Don't mark as unhealthy for resource warnings
	}

	if healthy {
		atomic.StoreInt32(&hp.isHealthy, 1)
		workerLogger.Debug().Msg("All health checks passed")
	} else {
		atomic.StoreInt32(&hp.isHealthy, 0)
		workerLogger.Error().Msg("One or more health checks failed")
	}
}

// checkDatabaseHealth checks PostgreSQL database connectivity and performance
func (hp *HealthProcessor) checkDatabaseHealth(ctx context.Context, workerLogger *zerolog.Logger) error {
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

		if openConnsInt, ok := openConns.(int); ok {
			connectionRatio := float64(openConnsInt) / float64(maxConns)

			if connectionRatio >= 0.9 {
				workerLogger.Error().
					Int("open_connections", openConnsInt).
					Int("max_connections", maxConns).
					Float64("usage_ratio", connectionRatio).
					Msg("CRITICAL: Database connection pool nearly exhausted")

				// Force connection cleanup by setting a very short max idle time
				if database.DB != nil {
					database.DB.SetConnMaxIdleTime(1 * time.Second)
				}
			} else if connectionRatio >= 0.8 {
				workerLogger.Warn().
					Int("open_connections", openConnsInt).
					Int("max_connections", maxConns).
					Float64("usage_ratio", connectionRatio).
					Msg("High database connection usage")
			}
		}

		workerLogger.Debug().
			Interface("stats", dbStats).
			Msg("Database health check passed")
	}

	return nil
}

// checkRedisHealth checks Redis connectivity and performance
func (hp *HealthProcessor) checkRedisHealth(ctx context.Context, workerLogger *zerolog.Logger) error {
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
func (hp *HealthProcessor) checkQueueHealth(ctx context.Context, workerLogger *zerolog.Logger) error {
	stats, err := hp.queueManager.GetQueueStats(ctx)
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
func (hp *HealthProcessor) checkSystemResources(ctx context.Context, workerLogger *zerolog.Logger) error {
	// Check disk space for database directory
	// This is a simplified check - in production, you might want more sophisticated monitoring

	// Check if we can query the database
	testQuery := "SELECT 1"
	var result int
	err := database.DB.QueryRowContext(ctx, testQuery).Scan(&result)
	if err != nil {
		return fmt.Errorf("database query test failed: %w", err)
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
func (hp *HealthProcessor) GetStats() map[string]interface{} {
	hp.mu.RLock()
	defer hp.mu.RUnlock()

	return map[string]interface{}{
		"last_activity":  hp.lastActivity,
		"is_healthy":     hp.IsHealthy(),
		"check_interval": configuration.Config.HealthCheckInterval.String(),
	}
}

// GetSystemHealth returns overall system health status
func (hp *HealthProcessor) GetSystemHealth(ctx context.Context) map[string]interface{} {
	health := make(map[string]interface{})

	// Database health
	dbErr := database.Health()
	health["database"] = map[string]interface{}{
		"healthy": dbErr == nil,
		"error":   getErrorString(dbErr),
		"stats":   database.GetStats(),
	}

	// Queue health
	queueStats, queueErr := hp.queueManager.GetQueueStats(ctx)
	health["queues"] = map[string]interface{}{
		"healthy": queueErr == nil && queueStats.QueueHealth != "critical",
		"error":   getErrorString(queueErr),
		"stats":   queueStats,
	}

	// Overall health
	overallHealthy := dbErr == nil &&
		queueErr == nil &&
		(queueStats == nil || queueStats.QueueHealth != "critical") &&
		hp.IsHealthy()

	health["overall"] = map[string]interface{}{
		"healthy":         overallHealthy,
		"last_check":      hp.lastActivity,
		"monitor_healthy": hp.IsHealthy(),
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
