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

func setupSystemRepoMock(t *testing.T) (*LocalSystemRepository, sqlmock.Sqlmock, func()) {
	originalDB := database.DB
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	database.DB = mockDB
	repo := NewLocalSystemRepository()

	cleanup := func() {
		database.DB = originalDB
		_ = mockDB.Close()
	}
	return repo, mock, cleanup
}

// The cascade suspend/reactivate repo methods must return the system_keys of
// every affected row (via RETURNING) so the service layer can invalidate each
// system's cached collect credentials.

func TestSuspendSystemsByOrgID_ReturnsAffectedSystemKeys(t *testing.T) {
	repo, mock, cleanup := setupSystemRepoMock(t)
	defer cleanup()

	mock.ExpectQuery(`UPDATE systems.+suspended_at.+WHERE organization_id = \$1 AND deleted_at IS NULL AND suspended_at IS NULL.+RETURNING system_key`).
		WithArgs("org-1", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"system_key"}).AddRow("SYS-1").AddRow("SYS-2"))

	keys, err := repo.SuspendSystemsByOrgID("org-1")

	assert.NoError(t, err)
	assert.Equal(t, []string{"SYS-1", "SYS-2"}, keys)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSuspendSystemsByMultipleOrgIDs_ReturnsAffectedSystemKeys(t *testing.T) {
	repo, mock, cleanup := setupSystemRepoMock(t)
	defer cleanup()

	mock.ExpectQuery(`UPDATE systems.+WHERE organization_id IN \(\$1,\$2\).+RETURNING system_key`).
		WithArgs("org-1", "org-2", "suspended-by-org", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"system_key"}).AddRow("SYS-A").AddRow("SYS-B").AddRow("SYS-C"))

	keys, err := repo.SuspendSystemsByMultipleOrgIDs([]string{"org-1", "org-2"}, "suspended-by-org")

	assert.NoError(t, err)
	assert.Equal(t, []string{"SYS-A", "SYS-B", "SYS-C"}, keys)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSuspendSystemsByMultipleOrgIDs_EmptyOrgIDsSkipsQuery(t *testing.T) {
	repo, mock, cleanup := setupSystemRepoMock(t)
	defer cleanup()

	// No query expectation registered: an empty org list must short-circuit.
	keys, err := repo.SuspendSystemsByMultipleOrgIDs(nil, "suspended-by-org")

	assert.NoError(t, err)
	assert.Empty(t, keys)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReactivateSystemsByOrgID_ReturnsAffectedSystemKeys(t *testing.T) {
	repo, mock, cleanup := setupSystemRepoMock(t)
	defer cleanup()

	mock.ExpectQuery(`UPDATE systems.+suspended_at = NULL.+WHERE suspended_by_org_id = \$1.+RETURNING system_key`).
		WithArgs("org-1", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"system_key"}).AddRow("SYS-1"))

	keys, err := repo.ReactivateSystemsByOrgID("org-1")

	assert.NoError(t, err)
	assert.Equal(t, []string{"SYS-1"}, keys)
	assert.NoError(t, mock.ExpectationsWereMet())
}
