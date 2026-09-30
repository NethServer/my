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
