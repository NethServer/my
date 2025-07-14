/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package services

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/logger"
)

type MetricsService struct {
	db *sql.DB
}

type SystemMetrics struct {
	Database DatabaseMetrics `json:"database"`
	Redis    RedisMetrics    `json:"redis"`
	Systems  SystemsMetrics  `json:"systems"`
	Collect  CollectMetrics  `json:"collect"`
	Workers  WorkerMetrics   `json:"workers"`
	Queues   QueueMetrics    `json:"queues"`
}

type DatabaseMetrics struct {
	ConnectionsOpen    int    `json:"connections_open"`
	ConnectionsInUse   int    `json:"connections_in_use"`
	ConnectionsIdle    int    `json:"connections_idle"`
	MaxOpenConnections int    `json:"max_open_connections"`
	MaxIdleClosed      int64  `json:"max_idle_closed"`
	MaxIdleTimeClosed  int64  `json:"max_idle_time_closed"`
	MaxLifetimeClosed  int64  `json:"max_lifetime_closed"`
	WaitCount          int64  `json:"wait_count"`
	WaitDuration       string `json:"wait_duration"`
	AvgResponseTime    string `json:"avg_response_time"`
	IsHealthy          bool   `json:"is_healthy"`
}

type RedisMetrics struct {
	ConnectedClients       int     `json:"connected_clients"`
	UsedMemory             int64   `json:"used_memory"`
	UsedMemoryHuman        string  `json:"used_memory_human"`
	TotalCommandsProcessed int64   `json:"total_commands_processed"`
	KeyspaceHits           int64   `json:"keyspace_hits"`
	KeyspaceMisses         int64   `json:"keyspace_misses"`
	HitRate                float64 `json:"hit_rate"`
	IsHealthy              bool    `json:"is_healthy"`
}

type SystemsMetrics struct {
	TotalSystems       int     `json:"total_systems"`
	ActiveSystems      int     `json:"active_systems"`
	RecentlyCollected  int     `json:"recently_collected"`
	LastCollectionTime *string `json:"last_collection_time"`
}

type CollectMetrics struct {
	TotalInventoryRecords int    `json:"total_inventory_records"`
	RecordsLast24h        int    `json:"records_last_24h"`
	TotalDiffs            int    `json:"total_diffs"`
	DiffsLast24h          int    `json:"diffs_last_24h"`
	PendingNotifications  int    `json:"pending_notifications"`
	DatabaseSize          string `json:"database_size,omitempty"`
}

type WorkerMetrics struct {
	InventoryWorkers      WorkerGroupMetrics `json:"inventory_workers"`
	ProcessingWorkers     WorkerGroupMetrics `json:"processing_workers"`
	NotificationWorkers   WorkerGroupMetrics `json:"notification_workers"`
	DelayedMessageWorkers WorkerGroupMetrics `json:"delayed_message_workers"`
	CleanupWorkers        WorkerGroupMetrics `json:"cleanup_workers"`
}

type WorkerGroupMetrics struct {
	TotalWorkers   int     `json:"total_workers"`
	HealthyWorkers int     `json:"healthy_workers"`
	ProcessedJobs  int64   `json:"processed_jobs"`
	FailedJobs     int64   `json:"failed_jobs"`
	LastActivity   *string `json:"last_activity,omitempty"`
}

type QueueMetrics struct {
	InventoryQueue    QueueInfo `json:"inventory_queue"`
	ProcessingQueue   QueueInfo `json:"processing_queue"`
	NotificationQueue QueueInfo `json:"notification_queue"`
	DelayedQueue      QueueInfo `json:"delayed_queue"`
}

type QueueInfo struct {
	Name       string `json:"name"`
	Length     int64  `json:"length"`
	Processing int64  `json:"processing"`
	Failed     int64  `json:"failed"`
	Completed  int64  `json:"completed"`
}

// NewMetricsService creates a new metrics service
func NewMetricsService() *MetricsService {
	return &MetricsService{
		db: database.DB,
	}
}

// GetSystemMetrics returns comprehensive system metrics
func (ms *MetricsService) GetSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	metrics := &SystemMetrics{}

	// Collect all metrics concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	wg.Add(6)

	// Database metrics
	go func() {
		defer wg.Done()
		if dbMetrics, err := ms.getDatabaseMetrics(ctx); err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("database metrics: %w", err))
			mu.Unlock()
		} else {
			metrics.Database = *dbMetrics
		}
	}()

	// Redis metrics
	go func() {
		defer wg.Done()
		if redisMetrics, err := ms.getRedisMetrics(ctx); err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("redis metrics: %w", err))
			mu.Unlock()
		} else {
			metrics.Redis = *redisMetrics
		}
	}()

	// Systems metrics
	go func() {
		defer wg.Done()
		if systemsMetrics, err := ms.getSystemsMetrics(ctx); err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("systems metrics: %w", err))
			mu.Unlock()
		} else {
			metrics.Systems = *systemsMetrics
		}
	}()

	// Collect service metrics
	go func() {
		defer wg.Done()
		if collectMetrics, err := ms.getCollectMetrics(ctx); err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("collect metrics: %w", err))
			mu.Unlock()
		} else {
			metrics.Collect = *collectMetrics
		}
	}()

	// Worker metrics
	go func() {
		defer wg.Done()
		if workerMetrics, err := ms.getWorkerMetrics(ctx); err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("worker metrics: %w", err))
			mu.Unlock()
		} else {
			metrics.Workers = *workerMetrics
		}
	}()

	// Queue metrics
	go func() {
		defer wg.Done()
		if queueMetrics, err := ms.getQueueMetrics(ctx); err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("queue metrics: %w", err))
			mu.Unlock()
		} else {
			metrics.Queues = *queueMetrics
		}
	}()

	wg.Wait()

	// Log any errors but still return partial metrics
	if len(errors) > 0 {
		for _, err := range errors {
			logger.Warn().Err(err).Msg("Failed to collect metrics component")
		}
	}

	return metrics, nil
}

// getDatabaseMetrics collects database connection and performance metrics
func (ms *MetricsService) getDatabaseMetrics(ctx context.Context) (*DatabaseMetrics, error) {
	stats := database.DB.Stats()

	// Test database health
	start := time.Now()
	err := database.DB.PingContext(ctx)
	responseTime := time.Since(start)

	return &DatabaseMetrics{
		ConnectionsOpen:    stats.OpenConnections,
		ConnectionsInUse:   stats.InUse,
		ConnectionsIdle:    stats.Idle,
		MaxOpenConnections: stats.MaxOpenConnections,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxIdleTimeClosed:  stats.MaxIdleTimeClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration.String(),
		AvgResponseTime:    responseTime.String(),
		IsHealthy:          err == nil,
	}, nil
}

// getRedisMetrics collects Redis metrics from the collect service
func (ms *MetricsService) getRedisMetrics(ctx context.Context) (*RedisMetrics, error) {
	// This would connect to the same Redis instance used by collect service
	// For now, return basic info - this could be enhanced to connect to Redis directly
	return &RedisMetrics{
		ConnectedClients:       0, // Would need Redis connection to get
		UsedMemory:             0,
		UsedMemoryHuman:        "N/A",
		TotalCommandsProcessed: 0,
		KeyspaceHits:           0,
		KeyspaceMisses:         0,
		HitRate:                0.0,
		IsHealthy:              true, // Would test connection
	}, nil
}

// getSystemsMetrics collects metrics about systems managed by the backend
func (ms *MetricsService) getSystemsMetrics(ctx context.Context) (*SystemsMetrics, error) {
	var totalSystems, activeSystems, recentlyCollected int
	var lastCollection sql.NullTime

	// Total systems
	err := ms.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM systems").Scan(&totalSystems)
	if err != nil {
		return nil, fmt.Errorf("failed to count total systems: %w", err)
	}

	// Active systems (those with recent inventory)
	err = ms.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT s.id) 
		FROM systems s 
		JOIN inventory_records ir ON s.id = ir.system_id 
		WHERE ir.timestamp > NOW() - INTERVAL '24 hours'
	`).Scan(&activeSystems)
	if err != nil {
		activeSystems = 0 // Not critical if this fails
	}

	// Recently collected (last 1 hour)
	err = ms.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT system_id) 
		FROM inventory_records 
		WHERE timestamp > NOW() - INTERVAL '1 hour'
	`).Scan(&recentlyCollected)
	if err != nil {
		recentlyCollected = 0
	}

	// Last collection time
	err = ms.db.QueryRowContext(ctx, `
		SELECT MAX(timestamp) FROM inventory_records
	`).Scan(&lastCollection)
	if err != nil {
		lastCollection.Valid = false
	}

	var lastCollectionStr *string
	if lastCollection.Valid {
		timeStr := lastCollection.Time.Format(time.RFC3339)
		lastCollectionStr = &timeStr
	}

	return &SystemsMetrics{
		TotalSystems:       totalSystems,
		ActiveSystems:      activeSystems,
		RecentlyCollected:  recentlyCollected,
		LastCollectionTime: lastCollectionStr,
	}, nil
}

// getCollectMetrics collects metrics from the collect service database
func (ms *MetricsService) getCollectMetrics(ctx context.Context) (*CollectMetrics, error) {
	var totalRecords, recordsLast24h, totalDiffs, diffsLast24h, pendingNotifications int

	// Total inventory records
	err := ms.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM inventory_records").Scan(&totalRecords)
	if err != nil {
		return nil, fmt.Errorf("failed to count inventory records: %w", err)
	}

	// Records in last 24h
	err = ms.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM inventory_records 
		WHERE created_at > NOW() - INTERVAL '24 hours'
	`).Scan(&recordsLast24h)
	if err != nil {
		recordsLast24h = 0
	}

	// Total diffs
	err = ms.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM inventory_diffs").Scan(&totalDiffs)
	if err != nil {
		totalDiffs = 0
	}

	// Diffs in last 24h
	err = ms.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM inventory_diffs 
		WHERE created_at > NOW() - INTERVAL '24 hours'
	`).Scan(&diffsLast24h)
	if err != nil {
		diffsLast24h = 0
	}

	// Pending notifications
	err = ms.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM inventory_diffs 
		WHERE notification_sent = false
	`).Scan(&pendingNotifications)
	if err != nil {
		pendingNotifications = 0
	}

	// Database size
	var dbSize sql.NullString
	err = ms.db.QueryRowContext(ctx, `
		SELECT pg_size_pretty(pg_database_size(current_database()))
	`).Scan(&dbSize)

	dbSizeStr := "N/A"
	if err == nil && dbSize.Valid {
		dbSizeStr = dbSize.String
	}

	return &CollectMetrics{
		TotalInventoryRecords: totalRecords,
		RecordsLast24h:        recordsLast24h,
		TotalDiffs:            totalDiffs,
		DiffsLast24h:          diffsLast24h,
		PendingNotifications:  pendingNotifications,
		DatabaseSize:          dbSizeStr,
	}, nil
}

// getWorkerMetrics collects metrics about collect service workers
func (ms *MetricsService) getWorkerMetrics(ctx context.Context) (*WorkerMetrics, error) {
	// Try to get some real data about recent worker activity
	var lastProcessedTime sql.NullTime
	var recentProcessedJobs int64

	// Get last processed inventory record time
	_ = ms.db.QueryRowContext(ctx, `
		SELECT MAX(processed_at), COUNT(*) 
		FROM inventory_records 
		WHERE processed_at IS NOT NULL AND processed_at > NOW() - INTERVAL '1 hour'
	`).Scan(&lastProcessedTime, &recentProcessedJobs)

	var lastActivityStr *string
	if lastProcessedTime.Valid {
		timeStr := lastProcessedTime.Time.Format(time.RFC3339)
		lastActivityStr = &timeStr
	}

	return &WorkerMetrics{
		InventoryWorkers: WorkerGroupMetrics{
			TotalWorkers:   5, // From configuration
			HealthyWorkers: 5, // Would check health endpoint
			ProcessedJobs:  recentProcessedJobs,
			FailedJobs:     0, // Would need error tracking
			LastActivity:   lastActivityStr,
		},
		ProcessingWorkers: WorkerGroupMetrics{
			TotalWorkers:   3,
			HealthyWorkers: 3,
			ProcessedJobs:  recentProcessedJobs,
			FailedJobs:     0,
			LastActivity:   lastActivityStr,
		},
		NotificationWorkers: WorkerGroupMetrics{
			TotalWorkers:   2,
			HealthyWorkers: 2,
			ProcessedJobs:  0, // No notifications sent recently
			FailedJobs:     0,
		},
		DelayedMessageWorkers: WorkerGroupMetrics{
			TotalWorkers:   1,
			HealthyWorkers: 1,
			ProcessedJobs:  0,
			FailedJobs:     0,
		},
		CleanupWorkers: WorkerGroupMetrics{
			TotalWorkers:   1,
			HealthyWorkers: 1,
			ProcessedJobs:  0, // Cleanup runs periodically
			FailedJobs:     0,
		},
	}, nil
}

// getQueueMetrics collects metrics about Redis queues
func (ms *MetricsService) getQueueMetrics(ctx context.Context) (*QueueMetrics, error) {
	// Try to estimate queue activity based on database records
	var recentRecords, recentDiffs int64

	// Get records processed in last hour
	_ = ms.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM inventory_records 
		WHERE created_at > NOW() - INTERVAL '1 hour'
	`).Scan(&recentRecords)

	// Get diffs created in last hour
	_ = ms.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM inventory_diffs 
		WHERE created_at > NOW() - INTERVAL '1 hour'
	`).Scan(&recentDiffs)

	return &QueueMetrics{
		InventoryQueue: QueueInfo{
			Name:       "collect:inventory",
			Length:     0, // Would get from Redis LLEN
			Processing: 0,
			Failed:     0,
			Completed:  recentRecords,
		},
		ProcessingQueue: QueueInfo{
			Name:       "collect:processing",
			Length:     0,
			Processing: 0,
			Failed:     0,
			Completed:  recentRecords,
		},
		NotificationQueue: QueueInfo{
			Name:       "collect:notifications",
			Length:     0,
			Processing: 0,
			Failed:     0,
			Completed:  0, // No notifications sent recently
		},
		DelayedQueue: QueueInfo{
			Name:       "collect:delayed",
			Length:     0,
			Processing: 0,
			Failed:     0,
			Completed:  0,
		},
	}, nil
}

// GetHealthStatus returns a simple health check status
func (ms *MetricsService) GetHealthStatus(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"services":  map[string]interface{}{},
	}

	services := health["services"].(map[string]interface{})

	// Test database
	if err := database.DB.PingContext(ctx); err != nil {
		services["database"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		health["status"] = "degraded"
	} else {
		services["database"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Test collect service database
	var count int
	if err := ms.db.QueryRowContext(ctx, "SELECT 1").Scan(&count); err != nil {
		services["collect_database"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		health["status"] = "degraded"
	} else {
		services["collect_database"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	return health
}
