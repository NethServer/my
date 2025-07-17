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

	// Test the connection
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	logger.ComponentLogger("database").Info().
		Str("database_url", logger.SanitizeConnectionURL(databaseURL)).
		Msg("Database connection established")

	// Initialize database schema
	if err := initSchemaFromFile(); err != nil {
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return nil
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
