/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskAPIKeyPermissions(t *testing.T) {
	all := []string{
		"read:systems", "read:users", "read:alerts",
		"manage:systems", "manage:users", "manage:applications", "manage:alerts",
		"destroy:systems", "destroy:users",
		"impersonate:users",
		"config:alerts",
	}

	t.Run("read mode keeps only read:*", func(t *testing.T) {
		got := maskAPIKeyPermissions(all, "read")
		assert.ElementsMatch(t, []string{"read:systems", "read:users", "read:alerts"}, got)
	})

	t.Run("write mode keeps read:* and manage:* including manage:alerts", func(t *testing.T) {
		got := maskAPIKeyPermissions(all, "write")
		assert.ElementsMatch(t, []string{
			"read:systems", "read:users", "read:alerts",
			"manage:systems", "manage:users", "manage:applications", "manage:alerts",
		}, got)
	})

	t.Run("denylist and non read/manage verbs never pass", func(t *testing.T) {
		for _, mode := range []string{"read", "write"} {
			got := maskAPIKeyPermissions(all, mode)
			assert.NotContains(t, got, "impersonate:users")
			assert.NotContains(t, got, "config:alerts")
			assert.NotContains(t, got, "destroy:systems")
			assert.NotContains(t, got, "destroy:users")
		}
	})

	t.Run("empty input yields empty, non-nil slice", func(t *testing.T) {
		got := maskAPIKeyPermissions(nil, "write")
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}
