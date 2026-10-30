/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserProfileFromName(t *testing.T) {
	assert.Nil(t, UserProfileFromName("  "))
	assert.Equal(t, map[string]interface{}{"givenName": "Mario"}, UserProfileFromName("Mario"))
	assert.Equal(t,
		map[string]interface{}{"givenName": "Sebastian Marius", "familyName": "Besel"},
		UserProfileFromName(" Sebastian  Marius Besel "))
}
