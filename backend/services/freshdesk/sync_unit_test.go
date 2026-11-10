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

func TestNamesByEmail(t *testing.T) {
	names := NamesByEmail([]models.LogtoUser{
		{PrimaryEmail: " Luigi@Example.com ", Name: " Luigi Rossi "},
		{PrimaryEmail: "luigi@example.com", Name: "Duplicate"},
		{PrimaryEmail: "noname@example.com", Name: "  "},
		{PrimaryEmail: "", Name: "No Email"},
	})

	assert.Equal(t, map[string]string{"luigi@example.com": "Luigi Rossi"}, names)
}

func TestPlanRenames(t *testing.T) {
	names := map[string]string{
		"luigi@example.com":  "Luigi Rossi",
		"maria@example.com":  "Maria Grazia De Luca",
		"sameok@example.com": "Same Name",
	}
	contacts := []Contact{
		{ID: 3, Name: "maria_12", Email: "Maria@example.com"},
		{ID: 1, Name: "luigi_30", Email: "luigi@example.com"},
		{ID: 2, Name: "Same Name", Email: "sameok@example.com"},
		{ID: 4, Name: "Unknown", Email: "unknown@example.com"},
		{ID: 5, Name: "No Email", Email: ""},
	}

	assert.Equal(t, []Rename{
		{Contact: contacts[1], Name: "Luigi Rossi"},
		{Contact: contacts[0], Name: "Maria Grazia De Luca"},
	}, PlanRenames(contacts, names))
	assert.Empty(t, PlanRenames(nil, names))
}
