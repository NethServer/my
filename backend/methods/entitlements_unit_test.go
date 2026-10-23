/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"testing"

	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
)

func TestRedactSeatAssigners(t *testing.T) {
	e := &models.SystemEntitlement{SeatAssignments: []models.EntitlementSeat{
		{ModuleID: "nethvoice1", AssignedBy: map[string]interface{}{"user_name": "Mario", "organization_id": "org-in"}},
		{ModuleID: "nethvoice2", AssignedBy: map[string]interface{}{"user_name": "Nethesis staff", "organization_id": "org-owner"}},
	}}

	redactSeatAssigners(e, []string{"org-in"})

	assert.Equal(t, "Mario", e.SeatAssignments[0].AssignedBy["user_name"])
	assert.Equal(t, map[string]interface{}{"out_of_scope": true}, e.SeatAssignments[1].AssignedBy)

	// The Owner organization sees every assigner.
	full := &models.SystemEntitlement{SeatAssignments: []models.EntitlementSeat{
		{AssignedBy: map[string]interface{}{"user_name": "Anyone", "organization_id": "org-x"}},
	}}
	redactSeatAssigners(full, nil)
	assert.Equal(t, "Anyone", full.SeatAssignments[0].AssignedBy["user_name"])
}

func TestSeatsFor(t *testing.T) {
	seated := &models.EntitlementCatalogItem{ID: "nethvoice-chat", Seated: true}
	plain := &models.EntitlementCatalogItem{ID: "crowdsec-advanced-threat-shield"}
	ten, zero := 10, 0

	got, msg := seatsFor(seated, &ten, true)
	assert.Empty(t, msg)
	assert.Equal(t, 10, *got)

	_, msg = seatsFor(seated, nil, true)
	assert.Equal(t, "seats required for this add-on", msg)

	got, msg = seatsFor(seated, nil, false)
	assert.Empty(t, msg)
	assert.Nil(t, got)

	_, msg = seatsFor(seated, &zero, true)
	assert.Equal(t, "seats must be greater than zero", msg)

	tooMany := maxSeats + 1
	_, msg = seatsFor(seated, &tooMany, true)
	assert.Equal(t, "seats must be at most 10000", msg)

	// An add-on without seats drops whatever the shop sends.
	got, msg = seatsFor(plain, &ten, true)
	assert.Empty(t, msg)
	assert.Nil(t, got)
}

// A buyer that cannot be placed, and the Owner organization, never trip the
// hierarchy check: only a known partner buying outside its tree does.
func TestBuyerOutsideHierarchy_UnplacedAndOwnerPass(t *testing.T) {
	outside, err := buyerOutsideHierarchy(map[string]interface{}{"email": "old-shop-account@example.com"}, "org-x")
	assert.NoError(t, err)
	assert.False(t, outside)

	outside, err = buyerOutsideHierarchy(map[string]interface{}{"organization_id": "owner-org", "org_role": "Owner"}, "org-x")
	assert.NoError(t, err)
	assert.False(t, outside)
}
