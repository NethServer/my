/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/nethesis/my/backend/logger"
)

var (
	DB *sql.DB
)

// envInt reads a positive integer from the environment, falling back to def
// when the variable is unset, unparseable, or non-positive.
// defaultStatementTimeout bounds a single SQL statement of the API. It is well
// above anything a healthy endpoint runs (the slowest legitimate statement
// observed on production, the organizations view refresh, takes 30 s under
// load and gets its own exemption) and well below the minutes a runaway list
// query used to hold the 0.1-CPU tier. Override with DATABASE_STATEMENT_TIMEOUT.
const defaultStatementTimeout = 60 * time.Second

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return def
}

// withStatementTimeout adds statement_timeout (in milliseconds, the unit
// Postgres assumes for a bare number) to a DSN unless it already carries one.
// Both forms lib/pq accepts are handled: a postgres:// URL and the
// space-separated key=value string.
func withStatementTimeout(dsn string, d time.Duration) (string, error) {
	ms := strconv.FormatInt(d.Milliseconds(), 10)
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return "", err
		}
		q := u.Query()
		if q.Has("statement_timeout") {
			return dsn, nil
		}
		q.Set("statement_timeout", ms)
		u.RawQuery = q.Encode()
		return u.String(), nil
	}
	if strings.Contains(dsn, "statement_timeout=") {
		return dsn, nil
	}
	return strings.TrimSpace(dsn) + " statement_timeout=" + ms, nil
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// Init initializes the database connection
func Init() error {
	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	// Cap every statement server-side. A request that nginx has already
	// answered with 504 (30 s) otherwise keeps its query running to the end:
	// the reseller list once reached 188 s on production while users retried.
	// lib/pq forwards unknown DSN keys to the server as session settings.
	databaseURL, err := withStatementTimeout(databaseURL, envDuration("DATABASE_STATEMENT_TIMEOUT", defaultStatementTimeout))
	if err != nil {
		return fmt.Errorf("invalid DATABASE_URL: %w", err)
	}

	// Open database connection
	DB, err = sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool. Sized small on purpose and tunable via env: the
	// managed Postgres tier runs on tight RAM (256MB) where every backend process
	// costs a few MB, so an oversized pool is the fast path to OOM, not throughput.
	// SetConnMaxIdleTime closes connections that go idle so the pool deflates
	// between bursts instead of pinning MaxIdleConns backends open forever.
	DB.SetMaxOpenConns(envInt("DATABASE_MAX_CONNS", 6))
	DB.SetMaxIdleConns(envInt("DATABASE_MAX_IDLE", 2))
	DB.SetConnMaxLifetime(5 * time.Minute)
	DB.SetConnMaxIdleTime(90 * time.Second)

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
	// REFRESH ... CONCURRENTLY cannot run inside a transaction, so the
	// statement_timeout exemption is set on a dedicated pooled connection and
	// reset before the connection goes back to the pool.
	ctx := context.Background()
	conn, err := DB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection for unified_organizations refresh: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.ExecContext(ctx, "SET statement_timeout = 0"); err != nil {
		return fmt.Errorf("failed to lift statement_timeout for unified_organizations refresh: %w", err)
	}
	_, err = conn.ExecContext(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY unified_organizations")
	if _, resetErr := conn.ExecContext(ctx, "RESET statement_timeout"); resetErr != nil && err == nil {
		err = resetErr
	}
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
