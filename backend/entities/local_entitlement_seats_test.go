/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package entities

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/nethesis/my/backend/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSeatsMock(t *testing.T) (*LocalEntitlementSeatRepository, sqlmock.Sqlmock, func()) {
	originalDB := database.DB
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	database.DB = mockDB
	repo := NewLocalEntitlementSeatRepository()

	cleanup := func() {
		database.DB = originalDB
		_ = mockDB.Close()
	}
	return repo, mock, cleanup
}

const (
	lockGrantQuery    = `SELECT seats FROM system_entitlements WHERE id = \$1 FOR UPDATE`
	liveInstanceQuery = `SELECT EXISTS \(SELECT 1 FROM applications\s+WHERE system_id = \$1 AND module_id = \$2 AND instance_of = \$3 AND deleted_at IS NULL\)`
	heldSeatsQuery    = `SELECT COUNT\(\*\) FROM entitlement_seats WHERE grant_id = \$1 AND released_at IS NULL`
	alreadyHeldQuery  = `SELECT EXISTS \(SELECT 1 FROM entitlement_seats WHERE grant_id = \$1 AND module_id = \$2 AND released_at IS NULL\)`
	insertSeatQuery   = `INSERT INTO entitlement_seats \(grant_id, module_id, assigned_by\)`
)

func TestSeatAssign_GivesAFreeSeat(t *testing.T) {
	repo, mock, cleanup := setupSeatsMock(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(lockGrantQuery).WithArgs("grant-1").
		WillReturnRows(sqlmock.NewRows([]string{"seats"}).AddRow(10))
	mock.ExpectQuery(liveInstanceQuery).WithArgs("sys-1", "nethvoice5", "nethvoice").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(heldSeatsQuery).WithArgs("grant-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(insertSeatQuery).WithArgs("grant-1", "nethvoice5", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "assigned_at"}).AddRow("seat-1", now))
	mock.ExpectCommit()

	seat, err := repo.Assign("grant-1", "sys-1", "nethvoice", "nethvoice5", map[string]interface{}{"user_id": "u1"})

	require.NoError(t, err)
	assert.Equal(t, "seat-1", seat.ID)
	assert.Equal(t, "nethvoice5", seat.ModuleID)
	assert.True(t, seat.InQuota)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// An instance of another application, or one that left the inventory, cannot
// take a seat.
func TestSeatAssign_RefusesAnInstanceNotLive(t *testing.T) {
	repo, mock, cleanup := setupSeatsMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectQuery(lockGrantQuery).WithArgs("grant-1").
		WillReturnRows(sqlmock.NewRows([]string{"seats"}).AddRow(10))
	mock.ExpectQuery(liveInstanceQuery).WithArgs("sys-1", "webtop1", "nethvoice").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectRollback()

	_, err := repo.Assign("grant-1", "sys-1", "nethvoice", "webtop1", nil)

	assert.ErrorIs(t, err, ErrInstanceNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSeatAssign_RefusesWhenAllSeatsAreTaken(t *testing.T) {
	repo, mock, cleanup := setupSeatsMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectQuery(lockGrantQuery).WithArgs("grant-1").
		WillReturnRows(sqlmock.NewRows([]string{"seats"}).AddRow(2))
	mock.ExpectQuery(liveInstanceQuery).WithArgs("sys-1", "nethvoice8", "nethvoice").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(heldSeatsQuery).WithArgs("grant-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(alreadyHeldQuery).WithArgs("grant-1", "nethvoice8").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectRollback()

	_, err := repo.Assign("grant-1", "sys-1", "nethvoice", "nethvoice8", nil)

	assert.ErrorIs(t, err, ErrNoFreeSeat)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// With every seat taken, asking again for an instance that already holds one
// says so instead of reporting a full grant.
func TestSeatAssign_FullGrantStillReportsTheSeatAlreadyHeld(t *testing.T) {
	repo, mock, cleanup := setupSeatsMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectQuery(lockGrantQuery).WithArgs("grant-1").
		WillReturnRows(sqlmock.NewRows([]string{"seats"}).AddRow(1))
	mock.ExpectQuery(liveInstanceQuery).WithArgs("sys-1", "nethvoice1", "nethvoice").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(heldSeatsQuery).WithArgs("grant-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(alreadyHeldQuery).WithArgs("grant-1", "nethvoice1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectRollback()

	_, err := repo.Assign("grant-1", "sys-1", "nethvoice", "nethvoice1", nil)

	assert.ErrorIs(t, err, ErrSeatAlreadyHeld)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSeatAssign_DuplicateInsertIsSeatAlreadyHeld(t *testing.T) {
	repo, mock, cleanup := setupSeatsMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectQuery(lockGrantQuery).WithArgs("grant-1").
		WillReturnRows(sqlmock.NewRows([]string{"seats"}).AddRow(10))
	mock.ExpectQuery(liveInstanceQuery).WithArgs("sys-1", "nethvoice1", "nethvoice").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(heldSeatsQuery).WithArgs("grant-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(insertSeatQuery).WithArgs("grant-1", "nethvoice1", sqlmock.AnyArg()).
		WillReturnError(&pq.Error{Code: "23505"})
	mock.ExpectRollback()

	_, err := repo.Assign("grant-1", "sys-1", "nethvoice", "nethvoice1", nil)

	assert.ErrorIs(t, err, ErrSeatAlreadyHeld)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSeatRelease(t *testing.T) {
	repo, mock, cleanup := setupSeatsMock(t)
	defer cleanup()

	const releaseQuery = `UPDATE entitlement_seats\s+SET released_at = NOW\(\), released_reason = \$3, released_by = \$4\s+WHERE grant_id = \$1 AND module_id = \$2 AND released_at IS NULL`

	mock.ExpectExec(releaseQuery).WithArgs("grant-1", "nethvoice5", SeatReleasedManual, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	assert.NoError(t, repo.Release("grant-1", "nethvoice5", map[string]interface{}{"user_id": "u1"}))

	mock.ExpectExec(releaseQuery).WithArgs("grant-1", "nethvoice9", SeatReleasedManual, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	assert.ErrorIs(t, repo.Release("grant-1", "nethvoice9", nil), ErrSeatNotFound)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSeatListByGrants(t *testing.T) {
	repo, mock, cleanup := setupSeatsMock(t)
	defer cleanup()

	out, err := repo.ListByGrants(nil)
	assert.NoError(t, err)
	assert.Empty(t, out)

	now := time.Now()
	mock.ExpectQuery(`SELECT s.grant_id, s.id, s.module_id, s.assigned_at, s.assigned_by,`).
		WillReturnRows(sqlmock.NewRows([]string{"grant_id", "id", "module_id", "assigned_at", "assigned_by", "in_quota"}).
			AddRow("grant-1", "seat-1", "nethvoice1", now, []byte(`{"user_id":"u1"}`), true).
			AddRow("grant-1", "seat-2", "nethvoice5", now, nil, false))

	out, err = repo.ListByGrants([]string{"grant-1", "grant-2"})

	require.NoError(t, err)
	require.Len(t, out["grant-1"], 2)
	assert.Equal(t, "u1", out["grant-1"][0].AssignedBy["user_id"])
	assert.True(t, out["grant-1"][0].InQuota)
	assert.False(t, out["grant-1"][1].InQuota)
	assert.Empty(t, out["grant-2"])
	assert.NoError(t, mock.ExpectationsWereMet())
}
