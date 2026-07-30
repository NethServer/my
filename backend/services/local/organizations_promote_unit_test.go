/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

var resellerColumns = []string{
	"id", "logto_id", "name", "description", "custom_data", "created_at", "updated_at",
	"logto_synced_at", "logto_sync_error", "deleted_at", "suspended_at", "suspended_by_org_id",
}

// setupPromoteMock swaps the package-level DB for a mock. Repositories capture
// database.DB at construction, so the service must be built after the swap.
func setupPromoteMock(t *testing.T) (sqlmock.Sqlmock, func()) {
	originalDB := database.DB
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	database.DB = mockDB
	return mock, func() {
		database.DB = originalDB
		_ = mockDB.Close()
	}
}

// promoter is the actor snapshot the handler builds from the authenticated user.
func promoter() *models.OrgCreator {
	return &models.OrgCreator{
		UserID:           "user-1",
		Username:         "owner",
		Name:             "Nethesis Owner",
		Email:            "owner@nethesis.it",
		OrganizationID:   "owner-1",
		OrganizationName: "Owner",
	}
}

func resellerRow(logtoID interface{}, suspendedAt interface{}) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows(resellerColumns).AddRow(
		"11111111-1111-1111-1111-111111111111", logtoID, "Acme Reseller", "",
		[]byte(`{"vat":"IT00000000001","type":"reseller","createdBy":"dist-1"}`),
		now, now, now, nil, nil, suspendedAt, nil,
	)
}

// A suspended organization cannot be promoted: its cascade state belongs to the
// distributor that suspended it, which the promotion would strand.
func TestPromoteResellerToDistributor_RejectsSuspended(t *testing.T) {
	mock, cleanup := setupPromoteMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM resellers`).WithArgs("res-1").
		WillReturnRows(resellerRow("res-1", time.Now()))

	_, err := NewOrganizationService().PromoteResellerToDistributor("res-1", promoter(), "owner-1")
	assert.ErrorIs(t, err, ErrPromoteResellerSuspended)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Without a logto_id there is no organization to move: the level lives in Logto
// too, so promoting would leave the two sides disagreeing.
func TestPromoteResellerToDistributor_RejectsUnsynced(t *testing.T) {
	mock, cleanup := setupPromoteMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM resellers`).WithArgs("res-1").
		WillReturnRows(resellerRow(nil, nil))

	_, err := NewOrganizationService().PromoteResellerToDistributor("res-1", promoter(), "owner-1")
	assert.ErrorIs(t, err, ErrPromoteResellerNotSynced)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// The Owner has no row in the org tables, which is what identifies the caller's
// own org as the parent to record.
func TestResolveOwnerOrgID_CallerFromOwnerOrg(t *testing.T) {
	mock, cleanup := setupPromoteMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("owner-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	ownerOrgID, err := NewOrganizationService().resolveOwnerOrgID("owner-1")
	assert.NoError(t, err)
	assert.Equal(t, "owner-1", ownerOrgID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// A Super Admin signing in from a partner org is not the Owner: the parent comes
// from the distributors, which the Owner creates.
func TestResolveOwnerOrgID_FromExistingDistributors(t *testing.T) {
	mock, cleanup := setupPromoteMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("dist-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`FROM distributors`).
		WillReturnRows(sqlmock.NewRows([]string{"createdBy"}).AddRow("owner-1"))

	ownerOrgID, err := NewOrganizationService().resolveOwnerOrgID("dist-1")
	assert.NoError(t, err)
	assert.Equal(t, "owner-1", ownerOrgID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// No distributor to derive the Owner from: fail closed rather than record an
// empty parent, which would detach the promoted org from the alerting chain.
func TestResolveOwnerOrgID_Unresolvable(t *testing.T) {
	mock, cleanup := setupPromoteMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("dist-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`FROM distributors`).WillReturnError(sql.ErrNoRows)

	_, err := NewOrganizationService().resolveOwnerOrgID("dist-1")
	assert.ErrorIs(t, err, ErrPromoteOwnerOrgUnknown)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Soft-deleted members keep their Logto role assignment, so they take part in
// the switch: a restore must not bring back the old level.
func TestOrganizationMemberLogtoIDs_IncludesSoftDeleted(t *testing.T) {
	mock, cleanup := setupPromoteMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM users`).WithArgs("res-1").
		WillReturnRows(sqlmock.NewRows([]string{"logto_id"}).AddRow("user-1").AddRow("user-2"))

	members, err := NewOrganizationService().organizationMemberLogtoIDs("res-1")
	assert.NoError(t, err)
	assert.Equal(t, []string{"user-1", "user-2"}, members)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// The creator snapshot lives in custom_data.createdByUser but the read hands it
// back on CreatedBy: both payloads have to carry it, or the promotion erases the
// organization's "created by".
func TestPromotedCustomData_KeepsCreatorSnapshot(t *testing.T) {
	creator := &models.OrgCreator{
		UserID:           "user-1",
		Username:         "distadmin",
		OrganizationID:   "dist-1",
		OrganizationName: "Acme Distribution",
	}
	reseller := &models.LocalReseller{
		CustomData: map[string]interface{}{
			"vat":       "IT00000000001",
			"type":      "reseller",
			"createdBy": "dist-1",
		},
		CreatedBy: creator,
	}

	promoted, original := promotedCustomData(reseller, "owner-1", nil)

	assert.Equal(t, "distributor", promoted["type"])
	assert.Equal(t, "owner-1", promoted["createdBy"])
	assert.Equal(t, creator, promoted["createdByUser"])
	assert.Equal(t, "IT00000000001", promoted["vat"])

	assert.Equal(t, "reseller", original["type"])
	assert.Equal(t, "dist-1", original["createdBy"])
	assert.Equal(t, creator, original["createdByUser"])

	// The reseller's own map stays untouched: the undo payload must not inherit
	// the promoted values.
	assert.Equal(t, "reseller", reseller.CustomData["type"])
	assert.NotContains(t, reseller.CustomData, "createdByUser")
}

// An organization with no creator snapshot must not gain an empty one.
func TestPromotedCustomData_WithoutCreator(t *testing.T) {
	reseller := &models.LocalReseller{CustomData: map[string]interface{}{"type": "reseller"}}

	promoted, original := promotedCustomData(reseller, "owner-1", nil)

	assert.NotContains(t, promoted, "createdByUser")
	assert.NotContains(t, original, "createdByUser")
	assert.Equal(t, "distributor", promoted["type"])
}

// The promotion snapshot is what tells a promoted distributor apart from one
// created at distributor level: it goes on the stored payload only, since the
// restore payload describes an organization that never moved.
func TestPromotedCustomData_CarriesPromotionSnapshot(t *testing.T) {
	reseller := &models.LocalReseller{
		CustomData: map[string]interface{}{"type": "reseller", "createdBy": "dist-1"},
	}
	promotion := &models.OrgPromotion{
		Level:                      "reseller",
		At:                         "2026-07-30T10:00:00Z",
		DetachedFromOrganizationID: "dist-1",
		By:                         promoter(),
	}

	promoted, original := promotedCustomData(reseller, "owner-1", promotion)

	assert.Equal(t, promotion, promoted["promotedFrom"])
	assert.NotContains(t, original, "promotedFrom")
}

// A request cannot claim a promotion: the snapshot the promotion writes wins over
// whatever the organization carries.
func TestPromotedCustomData_OverwritesClaimedPromotion(t *testing.T) {
	reseller := &models.LocalReseller{
		CustomData: map[string]interface{}{
			"type":         "reseller",
			"promotedFrom": map[string]interface{}{"level": "customer"},
		},
	}
	promotion := &models.OrgPromotion{Level: "reseller", At: "2026-07-30T10:00:00Z"}

	promoted, _ := promotedCustomData(reseller, "owner-1", promotion)

	assert.Equal(t, promotion, promoted["promotedFrom"])
}

// The promotion snapshot round-trips through custom_data the same way the creator
// snapshot does: typed value out, raw key gone from the map.
func TestExtractOrgPromotion(t *testing.T) {
	customData := map[string]interface{}{
		"vat": "IT00000000001",
		"promotedFrom": map[string]interface{}{
			"level":                         "reseller",
			"at":                            "2026-07-30T10:00:00Z",
			"detached_from_organization_id": "dist-1",
			"by":                            map[string]interface{}{"user_id": "user-1", "name": "Nethesis Owner"},
		},
	}

	promotion := models.ExtractOrgPromotion(customData)

	require.NotNil(t, promotion)
	assert.Equal(t, "reseller", promotion.Level)
	assert.Equal(t, "2026-07-30T10:00:00Z", promotion.At)
	assert.Equal(t, "dist-1", promotion.DetachedFromOrganizationID)
	require.NotNil(t, promotion.By)
	assert.Equal(t, "user-1", promotion.By.UserID)
	assert.NotContains(t, customData, "promotedFrom")
	assert.Equal(t, "IT00000000001", customData["vat"])

	assert.Nil(t, models.ExtractOrgPromotion(map[string]interface{}{"vat": "IT00000000001"}))
	assert.Nil(t, models.ExtractOrgPromotion(nil))
}

func TestCustomDataString(t *testing.T) {
	assert.Equal(t, "dist-1", customDataString(map[string]interface{}{"createdBy": "dist-1"}, "createdBy"))
	assert.Equal(t, "", customDataString(map[string]interface{}{"createdBy": 42}, "createdBy"))
	assert.Equal(t, "", customDataString(nil, "createdBy"))
}

func TestPromoteGuardrailsAreDistinctErrors(t *testing.T) {
	assert.False(t, errors.Is(ErrPromoteResellerSuspended, ErrPromoteResellerNotSynced))
	assert.False(t, errors.Is(ErrPromoteOwnerOrgUnknown, ErrPromoteResellerSuspended))
}
