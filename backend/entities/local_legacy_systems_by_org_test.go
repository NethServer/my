/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/backend/database"
)

func setupLegacySystemsMock(t *testing.T) (*LocalLegacySystemsByOrgRepository, sqlmock.Sqlmock, func()) {
	originalDB := database.DB
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	database.DB = mockDB
	repo := NewLocalLegacySystemsByOrgRepository()

	cleanup := func() {
		database.DB = originalDB
		_ = mockDB.Close()
	}
	return repo, mock, cleanup
}

func TestLegacySumByOrgIDs_EmptyScopeReturnsZeroWithoutQuerying(t *testing.T) {
	repo, mock, cleanup := setupLegacySystemsMock(t)
	defer cleanup()

	sum, err := repo.SumByOrgIDs(nil)

	assert.NoError(t, err)
	assert.Equal(t, LegacySystemsSum{}, sum)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLegacySumByOrgIDs_AggregatesAcrossTheScope(t *testing.T) {
	repo, mock, cleanup := setupLegacySystemsMock(t)
	defer cleanup()

	oldest := time.Now().Add(-3 * time.Hour).UTC()
	mock.ExpectQuery("SELECT(.|\n)*FROM legacy_systems_by_org").
		WillReturnRows(sqlmock.NewRows([]string{"sum", "min"}).AddRow(42, oldest))

	sum, err := repo.SumByOrgIDs([]string{"org-a", "org-b"})

	assert.NoError(t, err)
	assert.Equal(t, 42, sum.Total)
	assert.WithinDuration(t, oldest, sum.OldestUpdate, time.Second)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// An organization the legacy sync has never pushed simply has no row. That is a
// genuine zero, and it must not be confused with a sync that stopped: the caller
// tells them apart through OldestUpdate, which stays the zero time here.
func TestLegacySumByOrgIDs_NoRowsIsZeroWithNoTimestamp(t *testing.T) {
	repo, mock, cleanup := setupLegacySystemsMock(t)
	defer cleanup()

	mock.ExpectQuery("SELECT(.|\n)*FROM legacy_systems_by_org").
		WillReturnRows(sqlmock.NewRows([]string{"sum", "min"}).AddRow(0, nil))

	sum, err := repo.SumByOrgIDs([]string{"org-never-synced"})

	assert.NoError(t, err)
	assert.Equal(t, 0, sum.Total)
	assert.True(t, sum.OldestUpdate.IsZero())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLegacySumAll_CoversEveryOrganization(t *testing.T) {
	repo, mock, cleanup := setupLegacySystemsMock(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(total\\), 0\\), MIN\\(updated_at\\) FROM legacy_systems_by_org").
		WillReturnRows(sqlmock.NewRows([]string{"sum", "min"}).AddRow(19312, time.Now().UTC()))

	sum, err := repo.SumAll()

	assert.NoError(t, err)
	assert.Equal(t, 19312, sum.Total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// The whole point of replacing rather than upserting: an organization that has
// finished migrating disappears from the payload, and its stale non-zero count
// must disappear with it. The DELETE is what guarantees the counter can fall
// back to zero.
func TestLegacyReplaceAll_ClearsBeforeInserting(t *testing.T) {
	repo, mock, cleanup := setupLegacySystemsMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM legacy_systems_by_org").
		WillReturnResult(sqlmock.NewResult(0, 7))
	mock.ExpectPrepare("INSERT INTO legacy_systems_by_org").
		ExpectExec().
		WithArgs("org-a", 12).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.ReplaceAll(map[string]int{"org-a": 12})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// An empty payload is legitimate and meaningful: every organization migrated, or
// the legacy portal has nothing left to report. It must empty the table, not be
// mistaken for "nothing to do".
func TestLegacyReplaceAll_EmptyPayloadEmptiesTheTable(t *testing.T) {
	repo, mock, cleanup := setupLegacySystemsMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM legacy_systems_by_org").
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectPrepare("INSERT INTO legacy_systems_by_org")
	mock.ExpectCommit()

	err := repo.ReplaceAll(map[string]int{})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLegacyReplaceAll_RollsBackWhenAnInsertFails(t *testing.T) {
	repo, mock, cleanup := setupLegacySystemsMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM legacy_systems_by_org").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectPrepare("INSERT INTO legacy_systems_by_org").
		ExpectExec().
		WithArgs("org-a", 5).
		WillReturnError(assert.AnError)
	mock.ExpectRollback()

	err := repo.ReplaceAll(map[string]int{"org-a": 5})

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
