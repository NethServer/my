/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemCreatorAttributeToOrg(t *testing.T) {
	// A creator built from the acting user (e.g. the import/distributor account).
	base := func() *SystemCreator {
		return &SystemCreator{
			UserID:           "kyfy0tlnlk3l",
			Username:         "sviluppo_contratti",
			Name:             "Sviluppo - Integrazioni Esterne",
			Email:            "sviluppo+integrazioni@nethesis.it",
			OrganizationID:   "obhdyclbfx4t",
			OrganizationName: "Nethesis Italia",
		}
	}

	t.Run("attributes org and preserves the user identity", func(t *testing.T) {
		c := base()
		c.AttributeToOrg("eeex9cffzsd7", "Nethesis Diretta")

		assert.Equal(t, "eeex9cffzsd7", c.OrganizationID)
		assert.Equal(t, "Nethesis Diretta", c.OrganizationName)
		assert.True(t, c.OnBehalfOf)
		assert.Equal(t, "kyfy0tlnlk3l", c.UserID)
		assert.Equal(t, "sviluppo_contratti", c.Username)
		assert.Equal(t, "Sviluppo - Integrazioni Esterne", c.Name)
		assert.Equal(t, "sviluppo+integrazioni@nethesis.it", c.Email)
	})

	t.Run("no-op when orgID is empty (default own-org path)", func(t *testing.T) {
		c := base()
		c.AttributeToOrg("", "Whatever")
		assert.Equal(t, "obhdyclbfx4t", c.OrganizationID)
		assert.Equal(t, "Nethesis Italia", c.OrganizationName)
	})

	t.Run("no-op when orgName is empty", func(t *testing.T) {
		c := base()
		c.AttributeToOrg("eeex9cffzsd7", "")
		assert.Equal(t, "obhdyclbfx4t", c.OrganizationID)
		assert.Equal(t, "Nethesis Italia", c.OrganizationName)
	})

	t.Run("no-op when the attributed org matches the creator org", func(t *testing.T) {
		c := base()
		c.AttributeToOrg("obhdyclbfx4t", "Renamed But Same ID")
		assert.Equal(t, "obhdyclbfx4t", c.OrganizationID)
		assert.Equal(t, "Nethesis Italia", c.OrganizationName)
		assert.False(t, c.OnBehalfOf)
	})

	t.Run("nil receiver is safe", func(t *testing.T) {
		var c *SystemCreator
		assert.NotPanics(t, func() { c.AttributeToOrg("x", "y") })
	})
}
