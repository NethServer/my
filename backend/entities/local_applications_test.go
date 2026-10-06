/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"context"
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

// Soft-deleting a system does not cascade to its applications (they must
// survive a restore), so the read paths hide them themselves: otherwise the
// detail endpoint answers 403 (the system lookup fails) instead of 404 and
// the owner, who has no system filter, keeps seeing phantom apps in the list.
func TestGetByIDIgnoresApplicationsOnDeletedSystems(t *testing.T) {
	repo, mock, cleanup := setupAppRepoMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM applications a\s+LEFT JOIN systems s ON a\.system_id = s\.id.*WHERE a\.id = \$1 AND a\.deleted_at IS NULL AND EXISTS \(SELECT 1 FROM systems s2 WHERE s2\.id = a\.system_id AND s2\.deleted_at IS NULL\)`).
		WithArgs("sys-1-nextcloud1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	_, err := repo.GetByID("sys-1-nextcloud1")
	assert.EqualError(t, err, "application not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListOwnerScopeRequiresLiveSystem(t *testing.T) {
	repo, mock, cleanup := setupAppRepoMock(t)
	defer cleanup()

	// nil allowedSystemIDs = owner: no RBAC system filter, so the live-system
	// invariant is the only thing keeping deleted systems' apps out.
	mock.ExpectQuery(`FROM applications a\s+LEFT JOIN systems s ON a\.system_id = s\.id.*WHERE a\.deleted_at IS NULL AND EXISTS \(SELECT 1 FROM systems s2 WHERE s2\.id = a\.system_id AND s2\.deleted_at IS NULL\) AND a\.is_user_facing = TRUE`).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	apps, total, err := repo.List(context.Background(), nil, 1, 20, "", "", "", nil, nil, nil, nil, nil, nil, true)
	require.NoError(t, err)
	assert.Empty(t, apps)
	assert.Equal(t, 0, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListSearchMatchesModuleFQDNsLiterally(t *testing.T) {
	repo, mock, cleanup := setupAppRepoMock(t)
	defer cleanup()

	// One escaped parameter feeds every ILIKE, including the one over the
	// module FQDNs in inventory_data: "_" is a literal underscore, not a
	// single-character wildcard.
	mock.ExpectQuery(`OR s\.name ILIKE \$1 ESCAPE '\\' OR EXISTS \(SELECT 1 FROM jsonb_array_elements_text\(CASE WHEN jsonb_typeof\(a\.inventory_data->'fqdns'\) = 'array' THEN a\.inventory_data->'fqdns' END\) AS module_fqdn\(value\) WHERE module_fqdn\.value ILIKE \$1 ESCAPE '\\'\)\)`).
		WithArgs(`%cti\_acme.example.it%`, 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	apps, total, err := repo.List(context.Background(), nil, 1, 20, "cti_acme.example.it", "", "", nil, nil, nil, nil, nil, nil, true)
	require.NoError(t, err)
	assert.Empty(t, apps)
	assert.Equal(t, 0, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListSearchFollowsRBACSystemFilterPlaceholder(t *testing.T) {
	repo, mock, cleanup := setupAppRepoMock(t)
	defer cleanup()

	// Non-owner: the allowed system ids take $1, so every search ILIKE
	// (FQDN subquery included) must read $2.
	mock.ExpectQuery(`a\.system_id = ANY\(\$1::text\[\]\).*a\.module_id ILIKE \$2 ESCAPE '\\'.*WHERE module_fqdn\.value ILIKE \$2 ESCAPE '\\'\)\)`).
		WithArgs(sqlmock.AnyArg(), "%cti.acme%", 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	apps, total, err := repo.List(context.Background(), []string{"sys-1"}, 1, 20, "cti.acme", "", "", nil, nil, nil, nil, nil, nil, true)
	require.NoError(t, err)
	assert.Empty(t, apps)
	assert.Equal(t, 0, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}
