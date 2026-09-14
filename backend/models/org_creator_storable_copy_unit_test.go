/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrgCreatorStorableCopy(t *testing.T) {
	// A snapshot as it comes back from a read: OrganizationType was filled in by
	// the resolver, it was never in custom_data.
	read := &OrgCreator{
		UserID:           "kyfy0tlnlk3l",
		Username:         "sviluppo_contratti",
		Name:             "Sviluppo - Integrazioni Esterne",
		Email:            "sviluppo+integrazioni@nethesis.it",
		OrganizationID:   "obhdyclbfx4t",
		OrganizationName: "Nethesis Italia",
		OrganizationType: "reseller",
		OnBehalfOf:       true,
	}

	t.Run("drops the resolved level and keeps the rest", func(t *testing.T) {
		stored := read.StorableCopy()

		assert.Empty(t, stored.OrganizationType)
		assert.Equal(t, "kyfy0tlnlk3l", stored.UserID)
		assert.Equal(t, "obhdyclbfx4t", stored.OrganizationID)
		assert.Equal(t, "Nethesis Italia", stored.OrganizationName)
		assert.True(t, stored.OnBehalfOf)
	})

	t.Run("leaves the read snapshot untouched", func(t *testing.T) {
		_ = read.StorableCopy()

		assert.Equal(t, "reseller", read.OrganizationType)
	})

	t.Run("serialises without the organization_type key", func(t *testing.T) {
		// This is what an organization update writes back into custom_data: a
		// stored level would go stale the moment the org is promoted.
		encoded, err := json.Marshal(read.StorableCopy())
		assert.NoError(t, err)
		assert.NotContains(t, string(encoded), "organization_type")
	})

	t.Run("is nil-safe", func(t *testing.T) {
		var missing *OrgCreator

		assert.Nil(t, missing.StorableCopy())
	})
}
