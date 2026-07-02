/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nethesis/my/backend/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTenantMock(t *testing.T) (sqlmock.Sqlmock, func()) {
	originalDB := database.DB
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	database.DB = mockDB
	return mock, func() {
		database.DB = originalDB
		_ = mockDB.Close()
	}
}

const customerParentQuery = `SELECT custom_data->>'createdBy' FROM customers WHERE logto_id = \$1`

// A customer maps to its managing parent (reseller/distributor) recorded in createdBy.
func TestTenantForOrg_CustomerReturnsReseller(t *testing.T) {
	mock, cleanup := setupTenantMock(t)
	defer cleanup()

	mock.ExpectQuery(customerParentQuery).WithArgs("cust-1").
		WillReturnRows(sqlmock.NewRows([]string{"createdBy"}).AddRow("reseller-1"))

	tenant, err := TenantForOrg("cust-1")
	assert.NoError(t, err)
	assert.Equal(t, "reseller-1", tenant)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// A non-customer org (reseller/distributor/owner) has no customers row → tenant is itself.
func TestTenantForOrg_NonCustomerReturnsSelf(t *testing.T) {
	mock, cleanup := setupTenantMock(t)
	defer cleanup()

	mock.ExpectQuery(customerParentQuery).WithArgs("reseller-1").
		WillReturnRows(sqlmock.NewRows([]string{"createdBy"}))

	tenant, err := TenantForOrg("reseller-1")
	assert.NoError(t, err)
	assert.Equal(t, "reseller-1", tenant)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// A customer whose createdBy is NULL falls back to itself (never a dangling tenant).
func TestTenantForOrg_CustomerWithoutParentReturnsSelf(t *testing.T) {
	mock, cleanup := setupTenantMock(t)
	defer cleanup()

	mock.ExpectQuery(customerParentQuery).WithArgs("cust-orphan").
		WillReturnRows(sqlmock.NewRows([]string{"createdBy"}).AddRow(nil))

	tenant, err := TenantForOrg("cust-orphan")
	assert.NoError(t, err)
	assert.Equal(t, "cust-orphan", tenant)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantForOrg_EmptyOrgIDErrors(t *testing.T) {
	_, cleanup := setupTenantMock(t)
	defer cleanup()

	_, err := TenantForOrg("")
	assert.Error(t, err)
}
