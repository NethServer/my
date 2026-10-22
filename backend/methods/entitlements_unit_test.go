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

	// An add-on without seats drops whatever the shop sends.
	got, msg = seatsFor(plain, &ten, true)
	assert.Empty(t, msg)
	assert.Nil(t, got)
}
