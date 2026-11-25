/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

func setupSystemFavoritesMock(t *testing.T) (sqlmock.Sqlmock, func()) {
	originalDB := database.DB
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	database.DB = mockDB
	return mock, func() {
		database.DB = originalDB
		_ = mockDB.Close()
	}
}

func TestSystemFavoriteAdd_IsIdempotent(t *testing.T) {
	mock, cleanup := setupSystemFavoritesMock(t)
	defer cleanup()

	mock.ExpectExec(`INSERT INTO system_favorites \(user_logto_id, system_id\) VALUES \(\$1, \$2\)\s+ON CONFLICT \(user_logto_id, system_id\) DO NOTHING`).
		WithArgs("user-1", "sys-1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	require.NoError(t, NewLocalSystemFavoriteRepository().Add("user-1", "sys-1"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSystemFavoriteRemove_DeletesOnlyTheCallersRow(t *testing.T) {
	mock, cleanup := setupSystemFavoritesMock(t)
	defer cleanup()

	mock.ExpectExec(`DELETE FROM system_favorites WHERE user_logto_id = \$1 AND system_id = \$2`).
		WithArgs("user-1", "sys-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, NewLocalSystemFavoriteRepository().Remove("user-1", "sys-1"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSystemFavoriteIsFavorite(t *testing.T) {
	mock, cleanup := setupSystemFavoritesMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS \(SELECT 1 FROM system_favorites WHERE user_logto_id = \$1 AND system_id = \$2\)`).
		WithArgs("user-1", "sys-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	got, err := NewLocalSystemFavoriteRepository().IsFavorite("user-1", "sys-1")
	require.NoError(t, err)
	assert.True(t, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSystemFavoriteAdd_WrapsError(t *testing.T) {
	mock, cleanup := setupSystemFavoritesMock(t)
	defer cleanup()

	mock.ExpectExec(`INSERT INTO system_favorites`).WillReturnError(errors.New("fk violation"))

	err := NewLocalSystemFavoriteRepository().Add("user-1", "sys-1")
	assert.ErrorContains(t, err, "failed to add system favorite")
}

// captureSystemsListQuery runs the systems list as the Owner (no RBAC filter,
// so the only bound arguments are the favorites ones and paging), checks the
// arguments it bound and returns the SQL it sent.
func captureSystemsListQuery(t *testing.T, sortBy, sortDirection string, f models.SystemListFilters, wantArgs ...driver.Value) string {
	t.Helper()
	var gotSQL string
	matcher := sqlmock.QueryMatcherFunc(func(_ string, actual string) error {
		gotSQL = actual
		return nil
	})
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	mock.ExpectQuery("").WithArgs(wantArgs...).WillReturnError(errors.New("stop"))

	repo := &LocalSystemRepository{db: mockDB}
	_, _, err = repo.ListByCreatedByOrganizations(context.Background(), nil, 1, 50, "", sortBy, sortDirection, f)
	require.ErrorContains(t, err, "stop")
	return gotSQL
}

func TestListSystems_FavoritesJoinIsScopedToTheCaller(t *testing.T) {
	sql := captureSystemsListQuery(t, "", "asc", models.SystemListFilters{FavoritesOf: "user-1"}, "user-1", 50, 0)

	assert.Contains(t, sql, "LEFT JOIN system_favorites sf ON sf.system_id = s.id AND sf.user_logto_id = $1")
	assert.Contains(t, sql, "(sf.system_id IS NOT NULL) as is_favorite")
	assert.NotContains(t, sql, "AND (sf.system_id IS NOT NULL)")
}

func TestListSystems_WithoutCallerNothingIsStarred(t *testing.T) {
	sql := captureSystemsListQuery(t, "", "asc", models.SystemListFilters{}, 50, 0)

	assert.NotContains(t, sql, "system_favorites")
	assert.Contains(t, sql, "FALSE as is_favorite")
}

func TestListSystems_FavoritesOnlyFilter(t *testing.T) {
	sql := captureSystemsListQuery(t, "", "asc", models.SystemListFilters{FavoritesOf: "user-1", FavoritesOnly: true}, "user-1", 50, 0)

	assert.Contains(t, sql, "s.deleted_at IS NULL AND (sf.system_id IS NOT NULL)")
}

func TestListSystems_FavoritesOnlyWithoutCallerMatchesNothing(t *testing.T) {
	sql := captureSystemsListQuery(t, "", "asc", models.SystemListFilters{FavoritesOnly: true}, 50, 0)

	assert.Contains(t, sql, "s.deleted_at IS NULL AND FALSE")
}

func TestListSystems_SortByFavoriteAscendingListsFavoritesFirst(t *testing.T) {
	sql := captureSystemsListQuery(t, "favorite", "asc", models.SystemListFilters{FavoritesOf: "user-1"}, "user-1", 50, 0)

	assert.Contains(t, sql, "ORDER BY (sf.system_id IS NULL) ASC, LOWER(s.name), s.id")
}

func TestListSystems_SortByFavoriteDescendingListsFavoritesLast(t *testing.T) {
	sql := captureSystemsListQuery(t, "favorite", "desc", models.SystemListFilters{FavoritesOf: "user-1"}, "user-1", 50, 0)

	assert.Contains(t, sql, "ORDER BY (sf.system_id IS NULL) DESC, LOWER(s.name), s.id")
}

func TestUserHardDelete_RemovesTheUsersFavorites(t *testing.T) {
	mock, cleanup := setupSystemFavoritesMock(t)
	defer cleanup()

	mock.ExpectQuery(`WITH deleted AS \(\s+DELETE FROM users WHERE logto_id = \$1 RETURNING logto_id\s+\), favorites AS \(\s+DELETE FROM system_favorites WHERE user_logto_id IN \(SELECT logto_id FROM deleted\)\s+\)\s+SELECT COUNT\(\*\) FROM deleted`).
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	require.NoError(t, NewLocalUserRepository().HardDelete("user-1"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserHardDelete_UnknownUser(t *testing.T) {
	mock, cleanup := setupSystemFavoritesMock(t)
	defer cleanup()

	mock.ExpectQuery(`WITH deleted AS`).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	assert.EqualError(t, NewLocalUserRepository().HardDelete("missing"), "user not found")
}

func TestUserHardDeleteByOrgID_RemovesTheirFavorites(t *testing.T) {
	mock, cleanup := setupSystemFavoritesMock(t)
	defer cleanup()

	mock.ExpectQuery(`WITH deleted AS \(\s+DELETE FROM users WHERE organization_id = \$1 RETURNING logto_id\s+\), favorites AS \(\s+DELETE FROM system_favorites\s+WHERE user_logto_id IN \(SELECT logto_id FROM deleted WHERE logto_id IS NOT NULL\)\s+\)\s+SELECT logto_id FROM deleted WHERE logto_id IS NOT NULL`).
		WithArgs("org-1").
		WillReturnRows(sqlmock.NewRows([]string{"logto_id"}).AddRow("user-1").AddRow("user-2"))

	ids, err := NewLocalUserRepository().HardDeleteByOrgID("org-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"user-1", "user-2"}, ids)
	assert.NoError(t, mock.ExpectationsWereMet())
}
