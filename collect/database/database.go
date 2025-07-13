/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/collect/configuration"
)

var (
	DB    *sql.DB
	Redis *redis.Client
)

// Init initializes the database connections
func Init() error {
	// Initialize PostgreSQL connection
	if err := initPostgreSQL(); err != nil {
		return fmt.Errorf("failed to initialize PostgreSQL: %w", err)
	}

	// Initialize Redis connection
	if err := initRedis(); err != nil {
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}

	// Create database tables if they don't exist
	if err := createTables(); err != nil {
		return fmt.Errorf("failed to create database tables: %w", err)
	}

	logger.Info().Msg("Database connections initialized successfully")
	return nil
}

// initPostgreSQL initializes the PostgreSQL connection
func initPostgreSQL() error {
	var err error
	DB, err = sql.Open("postgres", configuration.Config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	DB.SetMaxOpenConns(configuration.Config.DatabaseMaxConns)
	DB.SetMaxIdleConns(configuration.Config.DatabaseMaxIdle)
	DB.SetConnMaxLifetime(configuration.Config.DatabaseConnMaxAge)

	// Test connection
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info().
		Int("max_conns", configuration.Config.DatabaseMaxConns).
		Int("max_idle", configuration.Config.DatabaseMaxIdle).
		Dur("conn_max_age", configuration.Config.DatabaseConnMaxAge).
		Msg("PostgreSQL connection established")

	return nil
}

// initRedis initializes the Redis connection
func initRedis() error {
	opt, err := redis.ParseURL(configuration.Config.RedisURL)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Override with configuration values
	opt.DB = configuration.Config.RedisDB
	opt.Password = configuration.Config.RedisPassword
	opt.MaxRetries = configuration.Config.RedisMaxRetries
	opt.DialTimeout = configuration.Config.RedisDialTimeout
	opt.ReadTimeout = configuration.Config.RedisReadTimeout
	opt.WriteTimeout = configuration.Config.RedisWriteTimeout

	Redis = redis.NewClient(opt)

	// Test connection
	ctx := context.Background()
	if err := Redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info().
		Str("redis_url", opt.Addr).
		Int("redis_db", opt.DB).
		Msg("Redis connection established")

	return nil
}

// GetRedisClient returns the Redis client instance
func GetRedisClient() *redis.Client {
	return Redis
}

// createTables creates the necessary database tables
func createTables() error {
	tables := []string{
		createSystemCredentialsTable,
		createInventoryRecordsTable,
		createInventoryDiffsTable,
		createInventoryMonitoringTable,
		createInventoryAlertsTable,
		createIndexes,
	}

	for _, query := range tables {
		if _, err := DB.Exec(query); err != nil {
			return fmt.Errorf("failed to execute table creation query: %w", err)
		}
	}

	logger.Info().Msg("Database tables created/verified successfully")
	return nil
}

// SQL table creation statements
const createSystemCredentialsTable = `
CREATE TABLE IF NOT EXISTS system_credentials (
    system_id VARCHAR(255) PRIMARY KEY,
    secret_hash VARCHAR(64) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_used TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
`

const createInventoryRecordsTable = `
CREATE TABLE IF NOT EXISTS inventory_records (
    id BIGSERIAL PRIMARY KEY,
    system_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    data JSONB NOT NULL,
    data_hash VARCHAR(64) NOT NULL,
    data_size BIGINT NOT NULL,
    compressed BOOLEAN NOT NULL DEFAULT false,
    processed_at TIMESTAMP WITH TIME ZONE,
    has_changes BOOLEAN NOT NULL DEFAULT false,
    change_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(system_id, data_hash)
);
`

const createInventoryDiffsTable = `
CREATE TABLE IF NOT EXISTS inventory_diffs (
    id BIGSERIAL PRIMARY KEY,
    system_id VARCHAR(255) NOT NULL,
    previous_id BIGINT REFERENCES inventory_records(id),
    current_id BIGINT NOT NULL REFERENCES inventory_records(id),
    diff_type VARCHAR(50) NOT NULL CHECK (diff_type IN ('create', 'update', 'delete')),
    field_path TEXT NOT NULL,
    previous_value TEXT,
    current_value TEXT,
    severity VARCHAR(50) NOT NULL DEFAULT 'low' CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    category VARCHAR(100) NOT NULL DEFAULT 'general',
    notification_sent BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
`

const createInventoryMonitoringTable = `
CREATE TABLE IF NOT EXISTS inventory_monitoring (
    id BIGSERIAL PRIMARY KEY,
    system_id VARCHAR(255), -- NULL for global rules
    field_path TEXT NOT NULL,
    monitor_type VARCHAR(50) NOT NULL CHECK (monitor_type IN ('threshold', 'change', 'pattern')),
    threshold TEXT,
    pattern TEXT,
    severity VARCHAR(50) NOT NULL DEFAULT 'medium' CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    description TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
`

const createInventoryAlertsTable = `
CREATE TABLE IF NOT EXISTS inventory_alerts (
    id BIGSERIAL PRIMARY KEY,
    system_id VARCHAR(255) NOT NULL,
    monitoring_id BIGINT NOT NULL REFERENCES inventory_monitoring(id),
    diff_id BIGINT REFERENCES inventory_diffs(id),
    alert_type VARCHAR(50) NOT NULL CHECK (alert_type IN ('threshold', 'change', 'pattern')),
    message TEXT NOT NULL,
    severity VARCHAR(50) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    is_resolved BOOLEAN NOT NULL DEFAULT false,
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
`

const createIndexes = `
-- Indexes for performance optimization
CREATE INDEX IF NOT EXISTS idx_inventory_records_system_id_timestamp ON inventory_records(system_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_inventory_records_data_hash ON inventory_records(data_hash);
CREATE INDEX IF NOT EXISTS idx_inventory_records_processed_at ON inventory_records(processed_at);

CREATE INDEX IF NOT EXISTS idx_inventory_diffs_system_id_created_at ON inventory_diffs(system_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_current_id ON inventory_diffs(current_id);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_severity ON inventory_diffs(severity);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_category ON inventory_diffs(category);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_notification_sent ON inventory_diffs(notification_sent) WHERE notification_sent = false;

CREATE INDEX IF NOT EXISTS idx_inventory_monitoring_system_id ON inventory_monitoring(system_id);
CREATE INDEX IF NOT EXISTS idx_inventory_monitoring_field_path ON inventory_monitoring(field_path);
CREATE INDEX IF NOT EXISTS idx_inventory_monitoring_enabled ON inventory_monitoring(is_enabled) WHERE is_enabled = true;

CREATE INDEX IF NOT EXISTS idx_inventory_alerts_system_id_created_at ON inventory_alerts(system_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inventory_alerts_monitoring_id ON inventory_alerts(monitoring_id);
CREATE INDEX IF NOT EXISTS idx_inventory_alerts_resolved ON inventory_alerts(is_resolved) WHERE is_resolved = false;
CREATE INDEX IF NOT EXISTS idx_inventory_alerts_severity ON inventory_alerts(severity);

CREATE INDEX IF NOT EXISTS idx_system_credentials_active ON system_credentials(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_system_credentials_last_used ON system_credentials(last_used DESC);
`

// Close closes all database connections
func Close() error {
	var dbErr, redisErr error

	if DB != nil {
		dbErr = DB.Close()
	}

	if Redis != nil {
		redisErr = Redis.Close()
	}

	if dbErr != nil {
		return fmt.Errorf("failed to close PostgreSQL connection: %w", dbErr)
	}

	if redisErr != nil {
		return fmt.Errorf("failed to close Redis connection: %w", redisErr)
	}

	logger.Info().Msg("Database connections closed successfully")
	return nil
}

// Health checks the health of database connections
func Health() error {
	// Check PostgreSQL
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("PostgreSQL health check failed: %w", err)
	}

	// Check Redis
	ctx := context.Background()
	if err := Redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis health check failed: %w", err)
	}

	return nil
}

// GetStats returns database connection statistics
func GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	if DB != nil {
		dbStats := DB.Stats()
		stats["postgresql"] = map[string]interface{}{
			"open_connections":     dbStats.OpenConnections,
			"in_use":              dbStats.InUse,
			"idle":                dbStats.Idle,
			"wait_count":          dbStats.WaitCount,
			"wait_duration":       dbStats.WaitDuration,
			"max_idle_closed":     dbStats.MaxIdleClosed,
			"max_idle_time_closed": dbStats.MaxIdleTimeClosed,
			"max_lifetime_closed": dbStats.MaxLifetimeClosed,
		}
	}

	if Redis != nil {
		poolStats := Redis.PoolStats()
		stats["redis"] = map[string]interface{}{
			"hits":         poolStats.Hits,
			"misses":       poolStats.Misses,
			"timeouts":     poolStats.Timeouts,
			"total_conns":  poolStats.TotalConns,
			"idle_conns":   poolStats.IdleConns,
			"stale_conns":  poolStats.StaleConns,
		}
	}

	return stats
}