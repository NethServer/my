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

// TestReportScopeClause covers the org-visibility predicate every report
// aggregate carries: absent for the owner org / a Super Admin (nil scope),
// present and numbered after the args already collected otherwise. An empty
// (non-nil) scope must still restrict — a caller whose hierarchy resolved to
// nothing sees nothing, not everything.
func TestReportScopeClause(t *testing.T) {
	tests := []struct {
		name           string
		orgScope       []string
		existingArgs   []interface{}
		expectedClause string
		expectedArgs   int
	}{
		{
			name:           "nil scope adds no predicate and no arg",
			orgScope:       nil,
			expectedClause: "",
			expectedArgs:   0,
		},
		{
			name:           "scope on a query with no other arg is $1",
			orgScope:       []string{"orgA", "orgB"},
			expectedClause: " AND s.organization_id = ANY($1)",
			expectedArgs:   1,
		},
		{
			name:           "scope is numbered after the search arg of the paginated slices",
			orgScope:       []string{"orgA"},
			existingArgs:   []interface{}{"searchTerm"},
			expectedClause: " AND s.organization_id = ANY($2)",
			expectedArgs:   2,
		},
		{
			name:           "empty but non-nil scope still restricts",
			orgScope:       []string{},
			expectedClause: " AND s.organization_id = ANY($1)",
			expectedArgs:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]interface{}{}, tt.existingArgs...)
			clause := reportScopeClause(tt.orgScope, &args)

			assert.Equal(t, tt.expectedClause, clause)
			assert.Len(t, args, tt.expectedArgs)
			if tt.orgScope != nil {
				assert.Equal(t, pq.Array(tt.orgScope), args[len(args)-1])
			}
		})
	}
}
