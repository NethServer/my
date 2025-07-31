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
	"sort"
	"strings"
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

	// Run database migrations first
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	// Initialize database schema (for new installations)
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

// runMigrations runs all pending database migrations
func runMigrations() error {
	logger.ComponentLogger("database").Info().Msg("Running database migrations")

	// Create migrations table if it doesn't exist
	if err := createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get list of executed migrations
	executedMigrations, err := getExecutedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get executed migrations: %w", err)
	}

	// Get list of migration files
	migrationFiles, err := getMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to get migration files: %w", err)
	}

	// Execute pending migrations
	for _, migrationFile := range migrationFiles {
		migrationName := strings.TrimSuffix(filepath.Base(migrationFile), ".sql")

		// Skip if already executed
		if contains(executedMigrations, migrationName) {
			continue
		}

		logger.ComponentLogger("database").Info().
			Str("migration", migrationName).
			Msg("Executing migration")

		if err := executeMigration(migrationFile, migrationName); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migrationName, err)
		}

		logger.ComponentLogger("database").Info().
			Str("migration", migrationName).
			Msg("Migration executed successfully")
	}

	logger.ComponentLogger("database").Info().Msg("All migrations completed successfully")
	return nil
}

// createMigrationsTable creates the migrations tracking table
func createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			executed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
	`
	_, err := DB.Exec(query)
	return err
}

// getExecutedMigrations returns list of already executed migrations
func getExecutedMigrations() ([]string, error) {
	rows, err := DB.Query("SELECT name FROM migrations ORDER BY executed_at")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.ComponentLogger("database").Error().Err(err).Msg("Failed to close rows")
		}
	}()

	var migrations []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		migrations = append(migrations, name)
	}

	return migrations, rows.Err()
}

// getMigrationFiles returns sorted list of migration files
func getMigrationFiles() ([]string, error) {
	migrationsDir := filepath.Join("database", "migrations")

	// Check if migrations directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		logger.ComponentLogger("database").Info().Msg("No migrations directory found, skipping migrations")
		return []string{}, nil
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return nil, err
	}

	// Sort files to ensure consistent execution order
	sort.Strings(files)
	return files, nil
}

// executeMigration executes a single migration file
func executeMigration(filePath, migrationName string) error {
	// Read migration file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Start transaction
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			logger.ComponentLogger("database").Error().Err(err).Msg("Failed to rollback transaction")
		}
	}()

	// Execute migration SQL
	if _, err := tx.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration as executed
	if _, err := tx.Exec("INSERT INTO migrations (name) VALUES ($1)", migrationName); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	return tx.Commit()
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
