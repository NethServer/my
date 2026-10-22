/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package local

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/models"
)

func TestTransferRenewalOutcome(t *testing.T) {
	expiry := time.Now().Add(30 * 24 * time.Hour)
	keep := []string{"owner", "distributor-b", "reseller-b", "customer-1"}

	cases := []struct {
		name  string
		grant models.OrganizationEntitlementGrant
		want  string
	}{
		{"perpetual grant", models.OrganizationEntitlementGrant{PurchasedByOrgID: "reseller-a"}, models.TransferRenewalPerpetual},
		{"already stopped", models.OrganizationEntitlementGrant{ValidUntil: &expiry, RenewalStoppedAt: &expiry, PurchasedByOrgID: "reseller-a"}, models.TransferRenewalAlreadyStopped},
		{"bought by the manager it leaves", models.OrganizationEntitlementGrant{ValidUntil: &expiry, PurchasedByOrgID: "reseller-a"}, models.TransferRenewalStops},
		{"bought by the new manager's distributor", models.OrganizationEntitlementGrant{ValidUntil: &expiry, PurchasedByOrgID: "distributor-b"}, models.TransferRenewalContinues},
		{"bought by the customer itself", models.OrganizationEntitlementGrant{ValidUntil: &expiry, PurchasedByOrgID: "customer-1"}, models.TransferRenewalContinues},
		{"bought by the owner organization", models.OrganizationEntitlementGrant{ValidUntil: &expiry, PurchasedByOrgID: "owner"}, models.TransferRenewalContinues},
		{"unknown buyer", models.OrganizationEntitlementGrant{ValidUntil: &expiry}, models.TransferRenewalStops},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, transferRenewalOutcome(&tc.grant, keep))
		})
	}
}
