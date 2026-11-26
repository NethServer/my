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
	"github.com/stretchr/testify/require"

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

func TestPlanOrphans(t *testing.T) {
	company := int64(7)
	identities := map[string]Identity{
		"luigi@example.com": {UserID: "u1"},
		"maria@example.com": {UserID: "u2"},
		"anna@example.com":  {UserID: "u3"},
	}
	contacts := []Contact{
		{ID: 1, Email: "Maria@example.com"},
		{ID: 2, Email: "luigi@example.com"},
		{ID: 2, Email: "luigi@example.com"},
		{ID: 3, Email: "anna@example.com", CompanyID: &company},
		{ID: 4, Email: "stranger@example.com"},
	}

	orphans := PlanOrphans(contacts, identities)
	require.Len(t, orphans, 2)
	assert.Equal(t, Orphan{Contact: contacts[1], UserID: "u1"}, orphans[0])
	assert.Equal(t, Orphan{Contact: contacts[0], UserID: "u2"}, orphans[1])
}

func TestCompanyIndexFind(t *testing.T) {
	index := NewCompanyIndex([]Company{
		{ID: 1, Name: "Acme Srl", CustomFields: CompanyCustomFields{VAT: "IT 01234567890"}},
		{ID: 2, Name: "DIGINET1 Communications srl"},
		{ID: 3, Name: "Old Name Spa", CustomFields: CompanyCustomFields{VAT: "09876543210"}},
	})

	company, ok := index.Find("01234567890", "Another Name")
	require.True(t, ok)
	assert.Equal(t, int64(1), company.ID)

	company, ok = index.Find("03830671206", "DIGINET1  communications SRL")
	require.True(t, ok)
	assert.Equal(t, int64(2), company.ID)

	company, ok = index.Find("09876543210", "New Name Spa")
	require.True(t, ok)
	assert.Equal(t, int64(3), company.ID)

	_, ok = index.Find("00502140049", "G.S.C. GENERAL SYSTEMS CUNEO SRL")
	assert.False(t, ok)

	index.Add(Company{ID: 4, Name: "G.S.C. GENERAL SYSTEMS CUNEO SRL", CustomFields: CompanyCustomFields{VAT: "00502140049"}})
	company, ok = index.Find("00502140049", "")
	require.True(t, ok)
	assert.Equal(t, int64(4), company.ID)
}
