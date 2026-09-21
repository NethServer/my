/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/backend/models"
)

func creatorOrgTypeRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"logto_id", "org_type", "live"})
}

func TestFillCreatorOrgTypes_LabelsEachOrganizationByItsRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	system := &models.SystemCreator{OrganizationID: "dist-1"}
	reseller := &models.OrgCreator{OrganizationID: "res-1"}
	deleted := &models.OrgCreator{OrganizationID: "cust-deleted"}
	owner := &models.OrgCreator{OrganizationID: "owner-org"}
	sameDistributor := &models.OrgCreator{OrganizationID: "dist-1"}
	var noSnapshot *models.OrgCreator

	// Every distinct organization is asked for once, in first-seen order.
	mock.ExpectQuery("FROM distributors").
		WithArgs(pq.Array([]string{"dist-1", "res-1", "cust-deleted", "owner-org"})).
		WillReturnRows(creatorOrgTypeRows().
			AddRow("dist-1", "distributor", true).
			AddRow("res-1", "reseller", true).
			AddRow("cust-deleted", "customer", false))

	fillCreatorOrgTypes(db, system, reseller, deleted, owner, sameDistributor, noSnapshot)

	assert.Equal(t, "distributor", system.OrganizationType)
	assert.Equal(t, "reseller", reseller.OrganizationType)
	assert.Equal(t, "distributor", sameDistributor.OrganizationType)
	assert.Empty(t, deleted.OrganizationType, "a deleted organization has no level to assert")
	assert.Equal(t, "owner", owner.OrganizationType, "no row in any organization table means the Owner organization")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// A logto_id that shows up both deleted and live is labelled by the live row:
// the deleted one is history, the live one is what the link would open.
func TestFillCreatorOrgTypes_LiveRowWinsOverDeletedOne(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	creator := &models.OrgCreator{OrganizationID: "org-1"}

	mock.ExpectQuery("FROM distributors").
		WithArgs(pq.Array([]string{"org-1"})).
		WillReturnRows(creatorOrgTypeRows().
			AddRow("org-1", "reseller", false).
			AddRow("org-1", "customer", true))

	fillCreatorOrgTypes(db, creator)

	assert.Equal(t, "customer", creator.OrganizationType)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// The enrichment never fails a read: on a query error the field stays empty
// for everyone, the Owner organization included, rather than being guessed.
func TestFillCreatorOrgTypes_QueryErrorLeavesTheFieldEmpty(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	creator := &models.SystemCreator{OrganizationID: "owner-org"}

	mock.ExpectQuery("FROM distributors").
		WithArgs(pq.Array([]string{"owner-org"})).
		WillReturnError(errors.New("connection reset"))

	fillCreatorOrgTypes(db, creator)

	assert.Empty(t, creator.OrganizationType)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFillCreatorOrgTypes_NothingToResolveSkipsTheQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	var noSnapshot *models.OrgCreator
	empty := &models.SystemCreator{}

	fillCreatorOrgTypes(db, noSnapshot, empty, nil)
	fillCreatorOrgTypes(nil, &models.OrgCreator{OrganizationID: "owner-org"})

	assert.Empty(t, empty.OrganizationType)
	assert.NoError(t, mock.ExpectationsWereMet())
}
