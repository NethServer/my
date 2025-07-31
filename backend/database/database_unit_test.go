/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package database

import (
	"database/sql"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_MissingDatabaseURL(t *testing.T) {
	// Ensure DATABASE_URL is not set
	err := os.Unsetenv("DATABASE_URL")
	require.NoError(t, err)

	err = Init()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL environment variable is required")
}

func TestInit_InvalidDatabaseURL(t *testing.T) {
	// Set invalid database URL
	t.Setenv("DATABASE_URL", "invalid-url")

	err := Init()
	assert.Error(t, err)
	// The actual error message may vary depending on the postgres driver
	assert.NotNil(t, err, "Should return an error for invalid database URL")
}

func TestInit_Success(t *testing.T) {
	// This test is more complex as Init() creates its own connection
	// We'll test the individual components instead
	t.Skip("Init() function involves real database connection - testing individual components instead")
}

func TestClose_WithNilDB(t *testing.T) {
	// Store original DB
	originalDB := DB
	defer func() { DB = originalDB }()

	// Set DB to nil
	DB = nil

	err := Close()
	assert.NoError(t, err)
}

func TestClose_WithValidDB(t *testing.T) {
	// Create mock database
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	// Store original DB
	originalDB := DB
	defer func() { DB = originalDB }()

	// Set our mock as the global DB
	DB = db

	// Mock expects close to be called
	mock.ExpectClose()

	err = Close()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHealthCheck_WithNilDB(t *testing.T) {
	// Store original DB
	originalDB := DB
	defer func() { DB = originalDB }()

	// Set DB to nil
	DB = nil

	err := HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection not initialized")
}

func TestHealthCheck_WithValidDB(t *testing.T) {
	// Create mock database with ping monitoring enabled
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer func() {
		err := db.Close()
		assert.NoError(t, err)
	}()

	// Store original DB
	originalDB := DB
	defer func() { DB = originalDB }()

	// Set our mock as the global DB
	DB = db

	// Mock successful ping
	mock.ExpectPing()

	err = HealthCheck()
	assert.NoError(t, err)

	// Expect close to be called by defer
	mock.ExpectClose()
	// Close expectations will be checked when defer runs
}

func TestHealthCheck_WithFailedPing(t *testing.T) {
	// Create mock database with ping monitoring enabled
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer func() {
		err := db.Close()
		assert.NoError(t, err)
	}()

	// Store original DB
	originalDB := DB
	defer func() { DB = originalDB }()

	// Set our mock as the global DB
	DB = db

	// Mock failed ping
	mock.ExpectPing().WillReturnError(sql.ErrConnDone)

	err = HealthCheck()
	assert.Error(t, err)
	assert.Equal(t, sql.ErrConnDone, err)

	// Expect close to be called by defer
	mock.ExpectClose()
	// Close expectations will be checked when defer runs
}

func TestInitSchemaFromFile_NoSchemaFile(t *testing.T) {
	// Create mock database
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() {
		err := db.Close()
		assert.NoError(t, err)
	}()

	// Store original DB
	originalDB := DB
	defer func() { DB = originalDB }()

	// Set our mock as the global DB
	DB = db

	// Expect the table existence check
	mock.ExpectQuery(`SELECT EXISTS \(SELECT FROM information_schema\.tables WHERE table_name = 'distributors'\)`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// Change to a temporary directory where schema.sql doesn't exist
	tempDir := t.TempDir()
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalWd)
		assert.NoError(t, err)
	}()
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// This should not return an error - it should handle missing schema gracefully
	err = initSchemaFromFile()
	assert.NoError(t, err)

	// Expect close to be called by defer
	mock.ExpectClose()
	// Close expectations will be checked when defer runs
}

func TestInitSchemaFromFile_WithSchemaFile(t *testing.T) {
	// Create mock database
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() {
		err := db.Close()
		assert.NoError(t, err)
	}()

	// Store original DB
	originalDB := DB
	defer func() { DB = originalDB }()

	// Set our mock as the global DB
	DB = db

	// Expect the table existence check
	mock.ExpectQuery(`SELECT EXISTS \(SELECT FROM information_schema\.tables WHERE table_name = 'distributors'\)`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// Create temporary directory with schema file
	tempDir := t.TempDir()
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalWd)
		assert.NoError(t, err)
	}()
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create database subdirectory and schema file
	err = os.Mkdir("database", 0755)
	require.NoError(t, err)

	schemaContent := `
		CREATE TABLE IF NOT EXISTS test_table (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		);
	`
	err = os.WriteFile("database/schema.sql", []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Mock the schema execution
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS test_table").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = initSchemaFromFile()
	assert.NoError(t, err)

	// Expect close to be called by defer
	mock.ExpectClose()
	// Close expectations will be checked when defer runs
}

func TestInitSchemaFromFile_SchemaExecutionFails(t *testing.T) {
	// Create mock database
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() {
		err := db.Close()
		assert.NoError(t, err)
	}()

	// Store original DB
	originalDB := DB
	defer func() { DB = originalDB }()

	// Set our mock as the global DB
	DB = db

	// Expect the table existence check
	mock.ExpectQuery(`SELECT EXISTS \(SELECT FROM information_schema\.tables WHERE table_name = 'distributors'\)`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// Create temporary directory with schema file
	tempDir := t.TempDir()
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalWd)
		assert.NoError(t, err)
	}()
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create database subdirectory and schema file
	err = os.Mkdir("database", 0755)
	require.NoError(t, err)

	schemaContent := "INVALID SQL STATEMENT;"
	err = os.WriteFile("database/schema.sql", []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Mock the schema execution to fail
	mock.ExpectExec("INVALID SQL STATEMENT").
		WillReturnError(sql.ErrTxDone)

	err = initSchemaFromFile()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute schema")

	// Expect close to be called by defer
	mock.ExpectClose()
	// Close expectations will be checked when defer runs
}
