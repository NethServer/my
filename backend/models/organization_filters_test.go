/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrganizationFiltersStatus(t *testing.T) {
	cases := []struct {
		name          string
		statuses      []string
		onlyEnabled   bool
		onlySuspended bool
	}{
		{"no filter keeps both", nil, false, false},
		{"enabled alone", []string{"enabled"}, true, false},
		{"suspended alone", []string{"suspended"}, false, true},
		{"both keep both", []string{"enabled", "suspended"}, false, false},
		{"case insensitive", []string{"Enabled"}, true, false},
		{"unknown values filter nothing", []string{"deleted"}, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := OrganizationFilters{Statuses: tc.statuses}
			assert.Equal(t, tc.onlyEnabled, f.OnlyEnabled())
			assert.Equal(t, tc.onlySuspended, f.OnlySuspended())
		})
	}
}
