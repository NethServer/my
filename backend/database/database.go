/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"
	"github.com/nethesis/my/backend/logger"
)

var (
	DB *sql.DB
)

// Init initializes the database connection
func Init() error {
	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	// Open database connection
	var err error
	DB, err = sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)

	if err := pingWithRetry(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	logger.ComponentLogger("database").Info().
		Str("database_url", logger.SanitizeConnectionURL(databaseURL)).
		Msg("Database connection established")

	// Initialize database schema (for new installations)
	if err := initSchemaFromFile(); err != nil {
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return nil
}

// RefreshUnifiedOrganizations refreshes the unified_organizations materialized view.
// Uses CONCURRENTLY to avoid locking reads during refresh.
func RefreshUnifiedOrganizations() error {
	_, err := DB.Exec("REFRESH MATERIALIZED VIEW CONCURRENTLY unified_organizations")
	if err != nil {
		return fmt.Errorf("failed to refresh unified_organizations: %w", err)
	}
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
		return DB.Close()
	}
	return nil
}

// HealthCheck performs a database health check
func HealthCheck() error {
	if DB == nil {
		return fmt.Errorf("database connection not initialized")
	}
	return DB.Ping()
}

// initSchemaFromFile initializes the database schema from SQL file
func initSchemaFromFile() error {
	logger.ComponentLogger("database").Info().Msg("Initializing database schema")

	// Check if core tables already exist (meaning migrations have run)
	var tableExists bool
	err := DB.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'distributors')").Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check if tables exist: %w", err)
	}

	if tableExists {
		logger.ComponentLogger("database").Info().Msg("Core tables already exist, skipping schema initialization")
		return nil
	}

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

	logger.ComponentLogger("database").Info().Msg("Database schema initialized successfully")
	return nil
}
