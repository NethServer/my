/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package workers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBoolField(t *testing.T) {
	yes := boolField(map[string]interface{}{"update_available": true}, "update_available")
	if assert.NotNil(t, yes) {
		assert.True(t, *yes)
	}
	no := boolField(map[string]interface{}{"image_updates_available": false}, "image_updates_available")
	if assert.NotNil(t, no) {
		assert.False(t, *no)
	}
	assert.Nil(t, boolField(map[string]interface{}{}, "update_available"), "missing key is unknown")
	assert.Nil(t, boolField(map[string]interface{}{"update_available": "true"}, "update_available"), "a string is not a flag")
	assert.Nil(t, boolField(nil, "update_available"), "no leader node is unknown")
}
