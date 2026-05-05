/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nethesis/my/backend/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAppRepoMock(t *testing.T) (*LocalApplicationRepository, sqlmock.Sqlmock, func()) {
	originalDB := database.DB
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	database.DB = mockDB
	repo := NewLocalApplicationRepository()

	cleanup := func() {
		database.DB = originalDB
		_ = mockDB.Close()
	}
	return repo, mock, cleanup
}

func TestUnassignAllForSystem(t *testing.T) {
	repo, mock, cleanup := setupAppRepoMock(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE applications\s+SET organization_id = NULL,\s+organization_type = NULL,\s+status = 'unassigned',\s+updated_at = \$2\s+WHERE system_id = \$1\s+AND deleted_at IS NULL\s+AND organization_id IS NOT NULL`).
		WithArgs("sys-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 3))

	rows, err := repo.UnassignAllForSystem("sys-1")
	require.NoError(t, err)
	assert.Equal(t, int64(3), rows)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUnassignAllForSystemRejectsEmpty(t *testing.T) {
	repo, _, cleanup := setupAppRepoMock(t)
	defer cleanup()

	_, err := repo.UnassignAllForSystem("")
	assert.Error(t, err)
}

func TestUnassignAllForSystemReturnsZeroWhenNothingAssigned(t *testing.T) {
	repo, mock, cleanup := setupAppRepoMock(t)
	defer cleanup()

	// Idempotency: rerun on a system whose apps are already unassigned
	// produces zero matches but no error.
	mock.ExpectExec(`UPDATE applications`).
		WithArgs("sys-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	rows, err := repo.UnassignAllForSystem("sys-1")
	require.NoError(t, err)
	assert.Equal(t, int64(0), rows)
	assert.NoError(t, mock.ExpectationsWereMet())
}
