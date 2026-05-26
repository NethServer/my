/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nethesis/my/backend/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAlertsTotalsMock(t *testing.T) (*LocalAlertsTotalsByOrgRepository, sqlmock.Sqlmock, func()) {
	originalDB := database.DB
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	database.DB = mockDB
	repo := NewLocalAlertsTotalsByOrgRepository()

	cleanup := func() {
		database.DB = originalDB
		_ = mockDB.Close()
	}
	return repo, mock, cleanup
}

func TestSumByOrgIDs_EmptyInputReturnsZeroNoQuery(t *testing.T) {
	repo, mock, cleanup := setupAlertsTotalsMock(t)
	defer cleanup()

	// No query expected — empty scope short-circuits.
	sum, err := repo.SumByOrgIDs(nil)

	assert.NoError(t, err)
	assert.Equal(t, AlertsTotalsSum{}, sum)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSumByOrgIDs_AggregatesAcrossOrgs(t *testing.T) {
	repo, mock, cleanup := setupAlertsTotalsMock(t)
	defer cleanup()

	oldest := time.Now().Add(-30 * time.Second).UTC()

	mock.ExpectQuery(`SELECT\s+COALESCE\(SUM\(active\),\s+0\),\s+COALESCE\(SUM\(critical\),\s+0\),\s+COALESCE\(SUM\(warning\),\s+0\),\s+COALESCE\(SUM\(info\),\s+0\),\s+COALESCE\(SUM\(muted\),\s+0\),\s+MIN\(updated_at\)\s+FROM alerts_totals_by_org\s+WHERE organization_id = ANY\(\$1\)`).
		WillReturnRows(sqlmock.NewRows([]string{"active", "critical", "warning", "info", "muted", "oldest"}).
			AddRow(527, 510, 12, 5, 18, oldest))

	sum, err := repo.SumByOrgIDs([]string{"org-1", "org-2"})

	assert.NoError(t, err)
	assert.Equal(t, 527, sum.Active)
	assert.Equal(t, 510, sum.Critical)
	assert.Equal(t, 12, sum.Warning)
	assert.Equal(t, 5, sum.Info)
	assert.Equal(t, 18, sum.Muted)
	assert.True(t, sum.OldestUpdate.Equal(oldest), "OldestUpdate should match")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSumByOrgIDs_NoMatchingRowsReturnsZeroAndNullOldest(t *testing.T) {
	repo, mock, cleanup := setupAlertsTotalsMock(t)
	defer cleanup()

	// COALESCE returns 0 for the counts; MIN(updated_at) yields SQL NULL when
	// no rows match, which the repo maps to a zero time.Time.
	mock.ExpectQuery(`FROM alerts_totals_by_org`).
		WillReturnRows(sqlmock.NewRows([]string{"active", "critical", "warning", "info", "muted", "oldest"}).
			AddRow(0, 0, 0, 0, 0, nil))

	sum, err := repo.SumByOrgIDs([]string{"unknown-org"})

	assert.NoError(t, err)
	assert.Equal(t, AlertsTotalsSum{}, sum)
	assert.True(t, sum.OldestUpdate.IsZero(), "OldestUpdate should be zero time for empty result")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSumByOrgIDs_QueryErrorPropagates(t *testing.T) {
	repo, mock, cleanup := setupAlertsTotalsMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM alerts_totals_by_org`).
		WillReturnError(errors.New("boom"))

	sum, err := repo.SumByOrgIDs([]string{"org-1"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sum alerts_totals_by_org")
	assert.Equal(t, AlertsTotalsSum{}, sum)
	assert.NoError(t, mock.ExpectationsWereMet())
}
