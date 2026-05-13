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

func TestQueryAlertHistory_PerSystem(t *testing.T) {
	repo, mock, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	now := time.Now().UTC()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM alert_history WHERE organization_id IN \(\$1\) AND system_key IN \(\$2\)`).
		WithArgs("org-1", "SYS-001").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	rows := sqlmock.NewRows([]string{
		"id", "system_key", "alertname", "severity", "status", "fingerprint",
		"starts_at", "ends_at", "summary", "labels", "annotations", "receiver", "created_at",
	}).
		AddRow(1, "SYS-001", "DiskFull", "critical", "resolved", "abc123",
			now.Add(-time.Hour), now, "Disk full", `{"alertname":"DiskFull"}`, `{"summary":"Disk full"}`, "default", now).
		AddRow(2, "SYS-001", "HighCPU", nil, "resolved", "def456",
			now.Add(-2*time.Hour), nil, nil, `{}`, `{}`, nil, now)

	mock.ExpectQuery(`SELECT id, system_key, alertname, severity, status, fingerprint`).
		WithArgs("org-1", "SYS-001", 20, 0).
		WillReturnRows(rows)

	records, totalCount, err := repo.QueryAlertHistory(AlertHistoryQuery{
		OrgIDs: []string{"org-1"}, SystemKeys: []string{"SYS-001"},
		Page: 1, PageSize: 20, SortBy: "created_at", SortDirection: "desc",
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, totalCount)
	assert.Len(t, records, 2)
	assert.Equal(t, "DiskFull", records[0].Alertname)
	assert.NotNil(t, records[0].Severity)
	assert.Equal(t, "critical", *records[0].Severity)
	assert.Nil(t, records[1].Severity)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryAlertHistory_EmptyOrgIDs(t *testing.T) {
	repo, _, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	// Empty OrgIDs short-circuits without hitting the DB.
	records, totalCount, err := repo.QueryAlertHistory(AlertHistoryQuery{
		OrgIDs: []string{}, Page: 1, PageSize: 20,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, totalCount)
	assert.Empty(t, records)
	assert.NotNil(t, records)
}

func TestQueryAlertHistory_MultiOrgWithDateRangeAndFilters(t *testing.T) {
	repo, mock, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	from := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM alert_history WHERE organization_id IN \(\$1,\$2\) AND alertname IN \(\$3\) AND severity IN \(\$4,\$5\) AND created_at >= \$6 AND created_at < \$7`).
		WithArgs("org-A", "org-B", "CVE-2024-1234", "critical", "warning", from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`SELECT id, system_key, alertname.*ORDER BY created_at desc`).
		WithArgs("org-A", "org-B", "CVE-2024-1234", "critical", "warning", from, to, 50, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "system_key", "alertname", "severity", "status", "fingerprint",
			"starts_at", "ends_at", "summary", "labels", "annotations", "receiver", "created_at",
		}))

	_, total, err := repo.QueryAlertHistory(AlertHistoryQuery{
		OrgIDs:        []string{"org-A", "org-B"},
		Alertnames:    []string{"CVE-2024-1234"},
		Severities:    []string{"critical", "warning"},
		From:          &from,
		To:            &to,
		Page:          1,
		PageSize:      50,
		SortBy:        "created_at",
		SortDirection: "desc",
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryAlertHistory_SortValidation(t *testing.T) {
	repo, mock, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT COUNT`).
		WithArgs("org-1", "SYS-001").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`ORDER BY created_at desc`).
		WithArgs("org-1", "SYS-001", 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "system_key", "alertname", "severity", "status", "fingerprint",
			"starts_at", "ends_at", "summary", "labels", "annotations", "receiver", "created_at",
		}))

	// Invalid sort_by/sort_direction must fall back to created_at desc.
	_, _, err := repo.QueryAlertHistory(AlertHistoryQuery{
		OrgIDs: []string{"org-1"}, SystemKeys: []string{"SYS-001"},
		Page: 1, PageSize: 10, SortBy: "invalid_column", SortDirection: "INVALID",
	})
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

func TestReassignSystemAlertHistory(t *testing.T) {
	repo, mock, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE alert_history\s+SET organization_id = \$1\s+WHERE system_key = \$2 AND organization_id = \$3`).
		WithArgs("org-new", "SYS-001", "org-old").
		WillReturnResult(sqlmock.NewResult(0, 7))

	rows, err := repo.ReassignSystemAlertHistory("SYS-001", "org-old", "org-new")
	require.NoError(t, err)
	assert.Equal(t, int64(7), rows)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReassignSystemAlertHistoryNoopWhenSameOrg(t *testing.T) {
	repo, _, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	// Same source and destination: must not hit the database at all
	// (any unexpected Exec would be flagged by sqlmock on close).
	rows, err := repo.ReassignSystemAlertHistory("SYS-001", "org-1", "org-1")
	require.NoError(t, err)
	assert.Equal(t, int64(0), rows)
}

func TestReassignSystemAlertHistoryRejectsEmptyArgs(t *testing.T) {
	repo, _, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	cases := []struct{ key, from, to string }{
		{"", "org-old", "org-new"},
		{"SYS-001", "", "org-new"},
		{"SYS-001", "org-old", ""},
	}
	for _, tc := range cases {
		_, err := repo.ReassignSystemAlertHistory(tc.key, tc.from, tc.to)
		assert.Error(t, err)
	}
}

func TestReassignSystemAlertHistoryReturnsZeroWhenNoMatch(t *testing.T) {
	repo, mock, cleanup := setupAlertHistoryMock(t)
	defer cleanup()

	// Idempotency: a re-run on a system whose history has already moved
	// matches no rows but is still a successful no-op.
	mock.ExpectExec(`UPDATE alert_history`).
		WithArgs("org-new", "SYS-001", "org-old").
		WillReturnResult(sqlmock.NewResult(0, 0))

	rows, err := repo.ReassignSystemAlertHistory("SYS-001", "org-old", "org-new")
	require.NoError(t, err)
	assert.Equal(t, int64(0), rows)
	assert.NoError(t, mock.ExpectationsWereMet())
}
