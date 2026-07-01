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

func TestOrgCreatorAttributeToOrg(t *testing.T) {
	// A creator built from the acting user (e.g. the import/distributor account).
	base := func() *OrgCreator {
		return &OrgCreator{
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

		// Org re-pointed to the attributed owner...
		assert.Equal(t, "eeex9cffzsd7", c.OrganizationID)
		assert.Equal(t, "Nethesis Diretta", c.OrganizationName)
		// ...but the actual actor is untouched.
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
	})

	t.Run("nil receiver is safe", func(t *testing.T) {
		var c *OrgCreator
		assert.NotPanics(t, func() { c.AttributeToOrg("x", "y") })
	})
}

// TestOrgCreatorAttributionRoundTrip verifies the full snapshot flow: an attributed
// creator stored under custom_data.createdByUser surfaces via ExtractOrgCreator as
// the top-level created_by with the owning org and the real acting user.
func TestOrgCreatorAttributionRoundTrip(t *testing.T) {
	creator := NewOrgCreatorFromUser(User{
		LogtoID:          strPtr("kyfy0tlnlk3l"),
		Username:         "sviluppo_contratti",
		Name:             "Sviluppo - Integrazioni Esterne",
		Email:            "sviluppo+integrazioni@nethesis.it",
		OrganizationID:   "obhdyclbfx4t",
		OrganizationName: "Nethesis Italia",
	})
	creator.AttributeToOrg("eeex9cffzsd7", "Nethesis Diretta")

	customData := map[string]interface{}{
		"type":      "customer",
		"createdBy": "eeex9cffzsd7",
		// json round-trip: mirror how the service stores the snapshot.
		"createdByUser": map[string]interface{}{
			"user_id":           creator.UserID,
			"username":          creator.Username,
			"name":              creator.Name,
			"email":             creator.Email,
			"organization_id":   creator.OrganizationID,
			"organization_name": creator.OrganizationName,
		},
	}

	extracted := ExtractOrgCreator(customData)
	assert.NotNil(t, extracted)
	assert.Equal(t, "eeex9cffzsd7", extracted.OrganizationID)
	assert.Equal(t, "Nethesis Diretta", extracted.OrganizationName)
	assert.Equal(t, "kyfy0tlnlk3l", extracted.UserID)
	assert.Equal(t, "sviluppo_contratti", extracted.Username)
	// Extraction removes the raw key so it is only exposed as the typed field.
	_, stillThere := customData["createdByUser"]
	assert.False(t, stillThere)
}
