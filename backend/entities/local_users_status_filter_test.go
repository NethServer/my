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

	"github.com/stretchr/testify/assert"
)

// TestUsersStatusClauses covers the SQL rendered for the users `status` query
// filter: the three exclusive statuses, the virtual additive "no_login" and
// the soft-delete guard that only "deleted" lifts.
func TestUsersStatusClauses(t *testing.T) {
	const (
		hideDeleted   = " AND u.deleted_at IS NULL"
		enabledPart   = "(u.deleted_at IS NULL AND u.suspended_at IS NULL)"
		suspendedPart = "(u.deleted_at IS NULL AND u.suspended_at IS NOT NULL)"
		deletedPart   = "(u.deleted_at IS NOT NULL)"
		noLoginPart   = "(u.latest_login_at IS NULL)"
	)

	tests := []struct {
		name            string
		statuses        []string
		expectedDeleted string
		expectedStatus  string
	}{
		{
			name:            "no filter only hides deleted accounts",
			statuses:        nil,
			expectedDeleted: hideDeleted,
			expectedStatus:  "",
		},
		{
			name:            "default selection",
			statuses:        []string{"enabled", "suspended"},
			expectedDeleted: hideDeleted,
			expectedStatus:  " AND (" + enabledPart + " OR " + suspendedPart + ")",
		},
		{
			name:            "deleted lifts the soft-delete guard",
			statuses:        []string{"deleted"},
			expectedDeleted: "",
			expectedStatus:  " AND (" + deletedPart + ")",
		},
		{
			name:            "no_login alone matches the badge among live accounts",
			statuses:        []string{"no_login"},
			expectedDeleted: hideDeleted,
			expectedStatus:  " AND (" + noLoginPart + ")",
		},
		{
			name:            "no_login adds to the other statuses",
			statuses:        []string{"enabled", "no_login"},
			expectedDeleted: hideDeleted,
			expectedStatus:  " AND (" + enabledPart + " OR " + noLoginPart + ")",
		},
		{
			name:            "no_login with deleted reaches archived accounts too",
			statuses:        []string{"deleted", "no_login"},
			expectedDeleted: "",
			expectedStatus:  " AND (" + deletedPart + " OR " + noLoginPart + ")",
		},
		{
			name:            "values are case-insensitive and unknown ones are ignored",
			statuses:        []string{"Enabled", "bogus", "NO_LOGIN"},
			expectedDeleted: hideDeleted,
			expectedStatus:  " AND (" + enabledPart + " OR " + noLoginPart + ")",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deletedClause, statusClause := usersStatusClauses(tt.statuses)
			assert.Equal(t, tt.expectedDeleted, deletedClause)
			assert.Equal(t, tt.expectedStatus, statusClause)
		})
	}
}
