/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package entities

import (
	"strings"
	"testing"
)

func TestOwnedByFilterClause(t *testing.T) {
	tests := []struct {
		name    string
		ownedBy []string
		want    string
	}{
		{
			name:    "empty input",
			ownedBy: nil,
			want:    "",
		},
		{
			name:    "single org id",
			ownedBy: []string{"u03ezlgb5u3h"},
			want:    " AND custom_data->>'createdBy' IN ('u03ezlgb5u3h')",
		},
		{
			name:    "multiple ids with duplicates",
			ownedBy: []string{"u03ezlgb5u3h", "5r14qcvnpoah", "u03ezlgb5u3h"},
			want:    " AND custom_data->>'createdBy' IN ('u03ezlgb5u3h', '5r14qcvnpoah')",
		},
		{
			name:    "injection attempts and empty values are dropped",
			ownedBy: []string{"", "abc'); DROP TABLE customers; --", "id with spaces", "ok_id-1"},
			want:    " AND custom_data->>'createdBy' IN ('ok_id-1')",
		},
		{
			name:    "only invalid values",
			ownedBy: []string{"", "not valid!"},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ownedByFilterClause(tt.ownedBy); got != tt.want {
				t.Errorf("ownedByFilterClause(%v) = %q, want %q", tt.ownedBy, got, tt.want)
			}
		})
	}
}

// TestChildOrganizationsArray pins the parent company lookup to a single
// level across the three organization tables, all bound to one placeholder.
func TestChildOrganizationsArray(t *testing.T) {
	got := childOrganizationsArray("$4")

	if !strings.HasPrefix(got, "ARRAY(") || !strings.HasSuffix(got, ")") {
		t.Fatalf("childOrganizationsArray must render an ARRAY(subquery), got %q", got)
	}
	for _, table := range []string{"distributors", "resellers", "customers"} {
		if !strings.Contains(got, "FROM "+table+" WHERE deleted_at IS NULL") {
			t.Errorf("childOrganizationsArray misses live %s: %q", table, got)
		}
	}
	if n := strings.Count(got, "custom_data->>'createdBy' = ANY($4::text[])"); n != 3 {
		t.Errorf("expected the parent placeholder once per table, got %d in %q", n, got)
	}
}

func TestBuildUsersParentOrgClause(t *testing.T) {
	var args []interface{}
	if got := buildUsersParentOrgClause(nil, &args); got != "" || len(args) != 0 {
		t.Fatalf("no parent filter must add neither clause nor args, got %q %v", got, args)
	}

	args = []interface{}{"already-bound"}
	got := buildUsersParentOrgClause([]string{"eeex9cffzsd7", "obhdyclbfx4t"}, &args)

	if len(args) != 2 {
		t.Fatalf("expected one array argument appended, got %v", args)
	}
	if want := " AND u.organization_id = ANY(" + childOrganizationsArray("$2") + ")"; got != want {
		t.Errorf("buildUsersParentOrgClause = %q, want %q", got, want)
	}
}
