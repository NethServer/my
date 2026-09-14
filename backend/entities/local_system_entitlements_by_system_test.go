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
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nethesis/my/backend/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEntitlementsBySystemMock(t *testing.T) (*LocalSystemEntitlementRepository, sqlmock.Sqlmock, func()) {
	originalDB := database.DB
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	database.DB = mockDB
	repo := NewLocalSystemEntitlementRepository()

	cleanup := func() {
		database.DB = originalDB
		_ = mockDB.Close()
	}
	return repo, mock, cleanup
}

const activeEntitlementsQuery = `SELECT system_id, entitlement\s+FROM system_entitlements\s+WHERE system_id = ANY\(\$1::text\[\]\)\s+AND revoked_at IS NULL\s+AND \(valid_until IS NULL OR valid_until > NOW\(\)\)\s+ORDER BY system_id, entitlement`

func TestActiveEntitlementsBySystems_EmptyInputRunsNoQuery(t *testing.T) {
	repo, mock, cleanup := setupEntitlementsBySystemMock(t)
	defer cleanup()

	out, err := repo.ActiveEntitlementsBySystems(nil)

	assert.NoError(t, err)
	assert.Empty(t, out)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestActiveEntitlementsBySystems_GroupsBySystem(t *testing.T) {
	repo, mock, cleanup := setupEntitlementsBySystemMock(t)
	defer cleanup()

	mock.ExpectQuery(activeEntitlementsQuery).
		WillReturnRows(sqlmock.NewRows([]string{"system_id", "entitlement"}).
			AddRow("sys-1", "nsec-blacklist").
			AddRow("sys-1", "nsec-ha").
			AddRow("sys-3", "nethvoice-chat"))

	out, err := repo.ActiveEntitlementsBySystems([]string{"sys-1", "sys-2", "sys-3"})

	assert.NoError(t, err)
	assert.Equal(t, []string{"nsec-blacklist", "nsec-ha"}, out["sys-1"])
	assert.Equal(t, []string{"nethvoice-chat"}, out["sys-3"])
	// A system with no live grant is absent, not an empty list.
	_, present := out["sys-2"]
	assert.False(t, present)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// One add-on granted on several scopes of the same system is one add-on to
// name: the module add-ons of an NS8 cluster carry one row per application
// instance.
func TestActiveEntitlementsBySystems_CollapsesScopedDuplicates(t *testing.T) {
	repo, mock, cleanup := setupEntitlementsBySystemMock(t)
	defer cleanup()

	mock.ExpectQuery(activeEntitlementsQuery).
		WillReturnRows(sqlmock.NewRows([]string{"system_id", "entitlement"}).
			AddRow("sys-1", "nethvoice-chat").
			AddRow("sys-1", "nethvoice-chat").
			AddRow("sys-1", "nethvoice-whatsapp"))

	out, err := repo.ActiveEntitlementsBySystems([]string{"sys-1"})

	assert.NoError(t, err)
	assert.Equal(t, []string{"nethvoice-chat", "nethvoice-whatsapp"}, out["sys-1"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestActiveEntitlementsBySystems_QueryErrorPropagates(t *testing.T) {
	repo, mock, cleanup := setupEntitlementsBySystemMock(t)
	defer cleanup()

	mock.ExpectQuery(activeEntitlementsQuery).WillReturnError(errors.New("connection reset"))

	out, err := repo.ActiveEntitlementsBySystems([]string{"sys-1"})

	assert.Error(t, err)
	assert.Nil(t, out)
	assert.NoError(t, mock.ExpectationsWereMet())
}
