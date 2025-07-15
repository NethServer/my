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
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"github.com/nethesis/my/backend/logger"
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

	// Initialize connection manager
	InitConnectionManager()

	logger.ComponentLogger("database").Info().Msg("Database connections initialized successfully")
	return nil
}

// initPostgreSQL initializes the PostgreSQL connection
func initPostgreSQL() error {
	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	// Parse configuration with defaults - optimized for batch processing
	maxConns := 50 // Optimized for batch processing and connection manager
	if maxConnsStr := os.Getenv("DATABASE_MAX_CONNS"); maxConnsStr != "" {
		if parsed, err := strconv.Atoi(maxConnsStr); err == nil {
			maxConns = parsed
		}
	}

	maxIdle := 5 // Increased from 2 for better connection reuse
	if maxIdleStr := os.Getenv("DATABASE_MAX_IDLE"); maxIdleStr != "" {
		if parsed, err := strconv.Atoi(maxIdleStr); err == nil {
			maxIdle = parsed
		}
	}

	connMaxAge := 15 * time.Minute // Increased from 5 minutes for better connection efficiency
	if connMaxAgeStr := os.Getenv("DATABASE_CONN_MAX_AGE"); connMaxAgeStr != "" {
		if parsed, err := time.ParseDuration(connMaxAgeStr); err == nil {
			connMaxAge = parsed
		}
	}

	// Open database connection
	var err error
	DB, err = sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	DB.SetMaxOpenConns(maxConns)
	DB.SetMaxIdleConns(maxIdle)
	DB.SetConnMaxLifetime(connMaxAge)
	DB.SetConnMaxIdleTime(1 * time.Minute) // Force cleanup of idle connections

	// Test connection
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	logger.ComponentLogger("database").Info().
		Str("database_url", databaseURL).
		Int("max_conns", maxConns).
		Int("max_idle", maxIdle).
		Dur("conn_max_age", connMaxAge).
		Msg("PostgreSQL connection established")

	// Initialize database schema
	if err := initSchemaFromFile(); err != nil {
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return nil
}

// initRedis initializes the Redis connection
func initRedis() error {
	// Get Redis configuration from environment
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Override with environment configuration
	if redisDB := os.Getenv("REDIS_DB"); redisDB != "" {
		if db, err := strconv.Atoi(redisDB); err == nil {
			opt.DB = db
		}
	}

	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		opt.Password = redisPassword
	}

	// Set reasonable defaults
	opt.MaxRetries = 3
	opt.DialTimeout = 5 * time.Second
	opt.ReadTimeout = 3 * time.Second
	opt.WriteTimeout = 3 * time.Second

	Redis = redis.NewClient(opt)

	// Test connection
	ctx := context.Background()
	if err := Redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.ComponentLogger("database").Info().
		Str("redis_url", opt.Addr).
		Int("redis_db", opt.DB).
		Msg("Redis connection established")

	return nil
}

// GetRedisClient returns the Redis client instance
func GetRedisClient() *redis.Client {
	return Redis
}

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

	logger.ComponentLogger("database").Info().Msg("Database connections closed successfully")
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
		return fmt.Errorf("redis health check failed: %w", err)
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
			"in_use":               dbStats.InUse,
			"idle":                 dbStats.Idle,
			"wait_count":           dbStats.WaitCount,
			"wait_duration":        dbStats.WaitDuration,
			"max_idle_closed":      dbStats.MaxIdleClosed,
			"max_idle_time_closed": dbStats.MaxIdleTimeClosed,
			"max_lifetime_closed":  dbStats.MaxLifetimeClosed,
		}
	}

	if Redis != nil {
		poolStats := Redis.PoolStats()
		stats["redis"] = map[string]interface{}{
			"hits":        poolStats.Hits,
			"misses":      poolStats.Misses,
			"timeouts":    poolStats.Timeouts,
			"total_conns": poolStats.TotalConns,
			"idle_conns":  poolStats.IdleConns,
			"stale_conns": poolStats.StaleConns,
		}
	}

	return stats
}

// initSchemaFromFile initializes the database schema from SQL file
func initSchemaFromFile() error {
	logger.ComponentLogger("database").Info().Msg("Initializing collect database schema")

	// Path to the schema file
	schemaFile := filepath.Join("database", "schema.sql")

	// Check if schema file exists
	if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
		logger.ComponentLogger("database").Warn().
			Str("schema_file", schemaFile).
			Msg("Schema file not found, skipping schema initialization")
		return nil
	}

	// Read schema file
	content, err := os.ReadFile(schemaFile)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Execute schema SQL
	if _, err := DB.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	logger.ComponentLogger("database").Info().Msg("Collect database schema initialized successfully")
	return nil
}
