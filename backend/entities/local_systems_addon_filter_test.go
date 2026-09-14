/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package entities

import (
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

// TestAddonFilterClause covers the SQL rendered for the `addon` query filter:
// the validity rule that decides whether a system "has" an add-on, the single
// placeholder shared by every selected add-on, and placeholder numbering.
func TestAddonFilterClause(t *testing.T) {
	tests := []struct {
		name           string
		filterAddons   []string
		argOffset      int
		expectedClause string
		expectedArgs   []interface{}
	}{
		{
			name:           "no filter yields no clause",
			filterAddons:   nil,
			expectedClause: "",
			expectedArgs:   nil,
		},
		{
			name:           "empty list yields no clause",
			filterAddons:   []string{},
			expectedClause: "",
			expectedArgs:   nil,
		},
		{
			name:           "one add-on matches live grants only",
			filterAddons:   []string{"nsec-blacklist"},
			expectedClause: `EXISTS (SELECT 1 FROM system_entitlements se WHERE se.system_id = s.id AND se.entitlement = ANY($1::text[]) AND se.revoked_at IS NULL AND (se.valid_until IS NULL OR se.valid_until > NOW()))`,
			expectedArgs:   []interface{}{pq.Array([]string{"nsec-blacklist"})},
		},
		{
			// Several add-ons travel in one array argument: selecting more of
			// them widens the match (OR) without adding placeholders.
			name:           "several add-ons share one array placeholder",
			filterAddons:   []string{"nsec-blacklist", "nsec-ha", "nethvoice-chat"},
			expectedClause: `EXISTS (SELECT 1 FROM system_entitlements se WHERE se.system_id = s.id AND se.entitlement = ANY($1::text[]) AND se.revoked_at IS NULL AND (se.valid_until IS NULL OR se.valid_until > NOW()))`,
			expectedArgs:   []interface{}{pq.Array([]string{"nsec-blacklist", "nsec-ha", "nethvoice-chat"})},
		},
		{
			name:           "placeholder follows the arguments already bound",
			filterAddons:   []string{"nsec-ha"},
			argOffset:      7,
			expectedClause: `EXISTS (SELECT 1 FROM system_entitlements se WHERE se.system_id = s.id AND se.entitlement = ANY($8::text[]) AND se.revoked_at IS NULL AND (se.valid_until IS NULL OR se.valid_until > NOW()))`,
			expectedArgs:   []interface{}{pq.Array([]string{"nsec-ha"})},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clause, args := addonFilterClause(tt.filterAddons, tt.argOffset)

			assert.Equal(t, tt.expectedClause, clause)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}
