/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nethesis/my/backend/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAlertHistoryMock(t *testing.T) (*LocalAlertHistoryRepository, sqlmock.Sqlmock, func()) {
	originalDB := database.DB
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	database.DB = mockDB
	repo := NewLocalAlertHistoryRepository()

	cleanup := func() {
		database.DB = originalDB
		_ = mockDB.Close()
	}
	return repo, mock, cleanup
}

func TestGetAlertHistoryBySystemKey(t *testing.T) {
	repo, mock, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	now := time.Now().UTC()

	// Expect count query
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM alert_history WHERE system_key = \$1 AND organization_id = \$2`).
		WithArgs("SYS-001", "org-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Expect data query
	rows := sqlmock.NewRows([]string{
		"id", "system_key", "alertname", "severity", "status", "fingerprint",
		"starts_at", "ends_at", "summary", "labels", "annotations", "receiver", "created_at",
	}).
		AddRow(1, "SYS-001", "DiskFull", "critical", "resolved", "abc123",
			now.Add(-time.Hour), now, "Disk full", `{"alertname":"DiskFull"}`, `{"summary":"Disk full"}`, "default", now).
		AddRow(2, "SYS-001", "HighCPU", nil, "resolved", "def456",
			now.Add(-2*time.Hour), nil, nil, `{}`, `{}`, nil, now)

	mock.ExpectQuery(`SELECT id, system_key, alertname, severity, status, fingerprint`).
		WithArgs("SYS-001", "org-1", 20, 0).
		WillReturnRows(rows)

	records, totalCount, err := repo.GetAlertHistoryBySystemKey("SYS-001", "org-1", 1, 20, "created_at", "desc")

	assert.NoError(t, err)
	assert.Equal(t, 2, totalCount)
	assert.Len(t, records, 2)

	// First record has all fields
	assert.Equal(t, "DiskFull", records[0].Alertname)
	assert.NotNil(t, records[0].Severity)
	assert.Equal(t, "critical", *records[0].Severity)
	assert.NotNil(t, records[0].EndsAt)
	assert.NotNil(t, records[0].Summary)
	assert.NotNil(t, records[0].Receiver)

	// Second record has nullable fields as nil
	assert.Nil(t, records[1].Severity)
	assert.Nil(t, records[1].EndsAt)
	assert.Nil(t, records[1].Summary)
	assert.Nil(t, records[1].Receiver)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAlertHistoryBySystemKey_EmptyResult(t *testing.T) {
	repo, mock, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM alert_history WHERE system_key = \$1 AND organization_id = \$2`).
		WithArgs("SYS-EMPTY", "org-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	rows := sqlmock.NewRows([]string{
		"id", "system_key", "alertname", "severity", "status", "fingerprint",
		"starts_at", "ends_at", "summary", "labels", "annotations", "receiver", "created_at",
	})

	mock.ExpectQuery(`SELECT id, system_key, alertname`).
		WithArgs("SYS-EMPTY", "org-1", 20, 0).
		WillReturnRows(rows)

	records, totalCount, err := repo.GetAlertHistoryBySystemKey("SYS-EMPTY", "org-1", 1, 20, "created_at", "desc")

	assert.NoError(t, err)
	assert.Equal(t, 0, totalCount)
	assert.Empty(t, records)
	assert.NotNil(t, records) // Empty slice, not nil
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAlertHistoryBySystemKey_SortValidation(t *testing.T) {
	repo, mock, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT COUNT`).
		WithArgs("SYS-001", "org-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`ORDER BY created_at desc`).
		WithArgs("SYS-001", "org-1", 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "system_key", "alertname", "severity", "status", "fingerprint",
			"starts_at", "ends_at", "summary", "labels", "annotations", "receiver", "created_at",
		}))

	// Invalid sort_by should fallback to created_at
	_, _, err := repo.GetAlertHistoryBySystemKey("SYS-001", "org-1", 1, 10, "invalid_column", "desc")
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAlertHistoryBySystemKey_SortDirectionValidation(t *testing.T) {
	repo, mock, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT COUNT`).
		WithArgs("SYS-001", "org-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`ORDER BY created_at desc`).
		WithArgs("SYS-001", "org-1", 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "system_key", "alertname", "severity", "status", "fingerprint",
			"starts_at", "ends_at", "summary", "labels", "annotations", "receiver", "created_at",
		}))

	// Invalid sort_direction should fallback to desc
	_, _, err := repo.GetAlertHistoryBySystemKey("SYS-001", "org-1", 1, 10, "created_at", "INVALID")
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAlertHistoryTotals_Scoped(t *testing.T) {
	repo, mock, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM alert_history WHERE organization_id = \$1`).
		WithArgs("dist-org").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

	total, err := repo.GetAlertHistoryTotals("dist-org")

	assert.NoError(t, err)
	assert.Equal(t, 10, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAlertHistoryTotals_AllTenants(t *testing.T) {
	repo, mock, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	// Empty orgID means "all tenants" — no filter, no args
	mock.ExpectQuery(`^SELECT COUNT\(\*\) FROM alert_history$`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	total, err := repo.GetAlertHistoryTotals("")

	assert.NoError(t, err)
	assert.Equal(t, 42, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAlertHistoryTrend_Up(t *testing.T) {
	repo, mock, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM alert_history WHERE created_at >= \$1 AND organization_id = \$2`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM alert_history WHERE created_at >= \$1 AND created_at < \$2 AND organization_id = \$3`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	today := time.Now().UTC().Truncate(24 * time.Hour)
	mock.ExpectQuery(`SELECT DATE\(created_at\) AS day, COUNT\(\*\) AS count FROM alert_history`).
		WillReturnRows(sqlmock.NewRows([]string{"day", "count"}).
			AddRow(today, 2))

	trend, err := repo.GetAlertHistoryTrend(7, "org-1")

	require.NoError(t, err)
	assert.Equal(t, 7, trend.Period)
	assert.Equal(t, "7 days", trend.PeriodLabel)
	assert.Equal(t, 5, trend.CurrentTotal)
	assert.Equal(t, 3, trend.PreviousTotal)
	assert.Equal(t, 2, trend.Delta)
	assert.Equal(t, "up", trend.Trend)
	assert.Len(t, trend.DataPoints, 7)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAlertHistoryTrend_Stable(t *testing.T) {
	repo, mock, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT COUNT`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT COUNT`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT DATE`).WillReturnRows(sqlmock.NewRows([]string{"day", "count"}))

	trend, err := repo.GetAlertHistoryTrend(7, "org-1")

	assert.NoError(t, err)
	assert.Equal(t, "stable", trend.Trend)
	assert.Equal(t, 0, trend.Delta)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAlertHistoryTrend_AllTenants(t *testing.T) {
	repo, mock, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	// No organization_id filter when orgID is empty
	mock.ExpectQuery(`^SELECT COUNT\(\*\) FROM alert_history WHERE created_at >= \$1$`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))
	mock.ExpectQuery(`^SELECT COUNT\(\*\) FROM alert_history WHERE created_at >= \$1 AND created_at < \$2$`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))
	mock.ExpectQuery(`^SELECT DATE\(created_at\) AS day, COUNT\(\*\) AS count FROM alert_history WHERE created_at >= \$1 GROUP BY day ORDER BY day$`).
		WillReturnRows(sqlmock.NewRows([]string{"day", "count"}))

	trend, err := repo.GetAlertHistoryTrend(7, "")

	assert.NoError(t, err)
	assert.Equal(t, 7, trend.CurrentTotal)
	assert.Equal(t, 4, trend.PreviousTotal)
	assert.Equal(t, "up", trend.Trend)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAlertHistoryTrend_Down(t *testing.T) {
	repo, mock, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT COUNT`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`SELECT COUNT`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery(`SELECT DATE`).WillReturnRows(sqlmock.NewRows([]string{"day", "count"}))

	trend, err := repo.GetAlertHistoryTrend(30, "org-1")

	assert.NoError(t, err)
	assert.Equal(t, "down", trend.Trend)
	assert.Equal(t, -3, trend.Delta)
	assert.Equal(t, -60.0, trend.DeltaPercentage)
	assert.Equal(t, "30 days", trend.PeriodLabel)
	assert.Len(t, trend.DataPoints, 30)
	assert.NoError(t, mock.ExpectationsWereMet())
}
