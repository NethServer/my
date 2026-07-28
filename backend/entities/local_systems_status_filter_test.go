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

// TestStatusFilterClause covers the SQL rendered for the `status` query filter:
// the column match, the two virtual statuses, their overlap rules and
// placeholder numbering.
func TestStatusFilterClause(t *testing.T) {
	const (
		columnMatchExcludingSuspended = `(s.status IN ($1) AND (s.suspended_at IS NULL OR s.deleted_at IS NOT NULL))`
		suspendedPart                 = `(s.suspended_at IS NOT NULL AND s.deleted_at IS NULL)`
		noInventoryPart               = `(s.last_inventory_at IS NULL AND s.status <> 'unknown')`
	)

	tests := []struct {
		name           string
		filterStatuses []string
		argOffset      int
		expectedClause string
		expectedArgs   []interface{}
	}{
		{
			name:           "no filter yields no clause",
			filterStatuses: nil,
			expectedClause: "",
			expectedArgs:   nil,
		},
		{
			name:           "single column status excludes suspended systems",
			filterStatuses: []string{"active"},
			expectedClause: "(" + columnMatchExcludingSuspended + ")",
			expectedArgs:   []interface{}{"active"},
		},
		{
			name:           "several column statuses share one IN list",
			filterStatuses: []string{"active", "inactive", "unknown"},
			expectedClause: `((s.status IN ($1,$2,$3) AND (s.suspended_at IS NULL OR s.deleted_at IS NOT NULL)))`,
			expectedArgs:   []interface{}{"active", "inactive", "unknown"},
		},
		{
			name:           "deleted is a plain column value",
			filterStatuses: []string{"deleted"},
			expectedClause: "(" + columnMatchExcludingSuspended + ")",
			expectedArgs:   []interface{}{"deleted"},
		},
		{
			name:           "suspended alone consumes no placeholder",
			filterStatuses: []string{"suspended"},
			expectedClause: "(" + suspendedPart + ")",
			expectedArgs:   nil,
		},
		{
			name:           "selecting suspended drops the exclusion from the column match",
			filterStatuses: []string{"active", "suspended"},
			expectedClause: `(s.status IN ($1) OR ` + suspendedPart + `)`,
			expectedArgs:   []interface{}{"active"},
		},
		{
			name:           "no_inventory alone consumes no placeholder",
			filterStatuses: []string{"no_inventory"},
			expectedClause: "(" + noInventoryPart + ")",
			expectedArgs:   nil,
		},
		{
			// no_inventory is additive: it does not suppress the suspended
			// exclusion the way selecting "suspended" does.
			name:           "no_inventory is OR'd onto the column match",
			filterStatuses: []string{"active", "no_inventory"},
			expectedClause: "(" + columnMatchExcludingSuspended + " OR " + noInventoryPart + ")",
			expectedArgs:   []interface{}{"active"},
		},
		{
			name:           "all three kinds combine in order",
			filterStatuses: []string{"active", "suspended", "no_inventory"},
			expectedClause: `(s.status IN ($1) OR ` + suspendedPart + ` OR ` + noInventoryPart + `)`,
			expectedArgs:   []interface{}{"active"},
		},
		{
			name:           "placeholders continue after the arguments already consumed",
			filterStatuses: []string{"active", "inactive"},
			argOffset:      3,
			expectedClause: `((s.status IN ($4,$5) AND (s.suspended_at IS NULL OR s.deleted_at IS NOT NULL)))`,
			expectedArgs:   []interface{}{"active", "inactive"},
		},
		{
			name:           "virtual statuses do not shift placeholder numbering",
			filterStatuses: []string{"suspended", "active", "no_inventory"},
			argOffset:      1,
			expectedClause: `(s.status IN ($2) OR ` + suspendedPart + ` OR ` + noInventoryPart + `)`,
			expectedArgs:   []interface{}{"active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clause, args := statusFilterClause(tt.filterStatuses, tt.argOffset)

			assert.Equal(t, tt.expectedClause, clause)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

// TestStatusFilterClause_NoInventoryMatchesWarningIcon pins the predicate to the
// condition behind the missing-inventory warning in the systems table
// (no inventory ever received, and the system is past its pending state). If the
// icon changes, this fails and the filter must be realigned with it.
func TestStatusFilterClause_NoInventoryMatchesWarningIcon(t *testing.T) {
	clause, args := statusFilterClause([]string{"no_inventory"}, 0)

	assert.Equal(t, `((s.last_inventory_at IS NULL AND s.status <> 'unknown'))`, clause)
	assert.Nil(t, args, "the no_inventory predicate is self-contained and takes no argument")
}
