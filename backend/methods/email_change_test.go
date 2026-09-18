/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeEmail(t *testing.T) {
	assert.Equal(t, "john.doe@example.com", normalizeEmail("  John.Doe@Example.COM "))
	assert.Equal(t, normalizeEmail("a@b.co"), normalizeEmail("A@B.CO"), "case never counts as a change")
}

func TestEmailShape(t *testing.T) {
	for _, ok := range []string{"a@b.co", "john+tag@example.com", "x@sub.domain.example"} {
		assert.True(t, emailShape.MatchString(ok), ok)
	}
	for _, bad := range []string{"", "john", "john@", "@example.com", "john@example", "john doe@example.com", "john@exa mple.com"} {
		assert.False(t, emailShape.MatchString(bad), bad)
	}
}
