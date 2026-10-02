/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package entities

import "testing"

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

func TestPartnerOrderClause(t *testing.T) {
	tests := []struct {
		name      string
		sortBy    string
		direction string
		want      string
	}{
		{
			name: "empty falls back to newest first",
			want: "ORDER BY created_at DESC",
		},
		{
			name:      "unknown key falls back to newest first",
			sortBy:    "name; DROP TABLE customers",
			direction: "asc",
			want:      "ORDER BY created_at DESC",
		},
		{
			name:      "name ascending",
			sortBy:    "name",
			direction: "asc",
			want:      "ORDER BY LOWER(name) ASC",
		},
		{
			name:      "direction is case insensitive",
			sortBy:    "suspended_at",
			direction: "desc",
			want:      "ORDER BY suspended_at DESC",
		},
		{
			name:      "managed by breaks ties on the name",
			sortBy:    "managed_by",
			direction: "DESC",
			want:      "ORDER BY LOWER(custom_data->'createdByUser'->>'organization_name') DESC, LOWER(name), id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := partnerOrderClause(tt.sortBy, tt.direction); got != tt.want {
				t.Errorf("partnerOrderClause(%q, %q) = %q, want %q", tt.sortBy, tt.direction, got, tt.want)
			}
		})
	}
}
