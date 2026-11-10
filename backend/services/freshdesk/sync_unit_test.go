/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package freshdesk

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/models"
)

func TestIdentitiesByEmail(t *testing.T) {
	identities := IdentitiesByEmail([]models.LogtoUser{
		{PrimaryEmail: " Luigi@Example.com ", Username: " luigi_30 ", Name: " Luigi  Rossi "},
		{PrimaryEmail: "luigi@example.com", Username: "dup", Name: "Duplicate"},
		{PrimaryEmail: "noname@example.com", Username: "noname", Name: "  "},
		{PrimaryEmail: "nousername@example.com", Username: "", Name: "No Username"},
		{PrimaryEmail: "", Username: "noemail", Name: "No Email"},
	})

	assert.Equal(t, map[string]Identity{
		"luigi@example.com": {Username: "luigi_30", Name: "Luigi Rossi"},
	}, identities)
}

func TestPlanRenames(t *testing.T) {
	identities := map[string]Identity{
		"luigi@example.com":  {Username: "luigi_30", Name: "Luigi Rossi"},
		"maria@example.com":  {Username: "Maria_12", Name: "Maria Grazia De Luca"},
		"edited@example.com": {Username: "edited_1", Name: "Edited In My"},
		"same@example.com":   {Username: "same", Name: "same"},
	}
	contacts := []Contact{
		{ID: 3, Name: "maria_12", Email: "Maria@example.com"},
		{ID: 1, Name: " luigi_30 ", Email: "luigi@example.com"},
		{ID: 2, Name: "Edited In Freshdesk", Email: "edited@example.com"},
		{ID: 4, Name: "same", Email: "same@example.com"},
		{ID: 5, Name: "unknown_1", Email: "unknown@example.com"},
		{ID: 6, Name: "noemail_1", Email: ""},
	}

	assert.Equal(t, []Rename{
		{Contact: contacts[1], Name: "Luigi Rossi"},
		{Contact: contacts[0], Name: "Maria Grazia De Luca"},
	}, PlanRenames(contacts, identities))
	assert.Empty(t, PlanRenames(nil, identities))
}
