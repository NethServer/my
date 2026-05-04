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
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"

	"github.com/nethesis/my/collect/logger"
)

var (
	DB *sql.DB
)

// Init initializes the database connections
func Init() error {
	// Initialize PostgreSQL connection
	if err := initPostgreSQL(); err != nil {
		return fmt.Errorf("failed to initialize PostgreSQL: %w", err)
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

	maxIdle := 15 // Keep idle connections ready under concurrent load
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

	if err := pingWithRetry(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	logger.ComponentLogger("database").Info().
		Str("database_url", logger.SanitizeConnectionURL(databaseURL)).
		Int("max_conns", maxConns).
		Int("max_idle", maxIdle).
		Dur("conn_max_age", connMaxAge).
		Msg("PostgreSQL connection established")

	return nil
}

func pingWithRetry() error {
	budget := 90 * time.Second
	if v := os.Getenv("DATABASE_PING_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			budget = d
		}
	}
	interval := 2 * time.Second
	if v := os.Getenv("DATABASE_PING_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}

	deadline := time.Now().Add(budget)
	attempt := 0
	for {
		attempt++
		err := DB.Ping()
		if err == nil {
			if attempt > 1 {
				logger.ComponentLogger("database").Info().
					Int("attempts", attempt).
					Msg("PostgreSQL became reachable")
			}
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("database unreachable after %d attempts in %s: %w", attempt, budget, err)
		}
		logger.ComponentLogger("database").Warn().
			Err(err).
			Int("attempt", attempt).
			Dur("retry_in", interval).
			Msg("PostgreSQL not ready, retrying")
		time.Sleep(interval)
	}
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		if err := DB.Close(); err != nil {
			return fmt.Errorf("failed to close PostgreSQL connection: %w", err)
		}
	}

	logger.ComponentLogger("database").Info().Msg("Database connection closed successfully")
	return nil
}

// Health checks the health of the database connection
func Health() error {
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("PostgreSQL health check failed: %w", err)
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

	return stats
}
