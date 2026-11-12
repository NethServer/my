/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package pullcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePullFlags(t *testing.T) {
	tests := []struct {
		name    string
		flags   Flags
		wantErr string
	}{
		{name: "defaults", flags: Flags{ConflictStrategy: "skip"}},
		{name: "one exclusive flag", flags: Flags{ConflictStrategy: "merge", OrganizationsOnly: true}},
		{name: "overwrite-all with force", flags: Flags{ConflictStrategy: "skip", OverwriteAll: true, Force: true}},
		{
			name:    "unknown conflict strategy",
			flags:   Flags{ConflictStrategy: "ignore"},
			wantErr: "invalid conflict strategy",
		},
		{
			name:    "two exclusive flags",
			flags:   Flags{ConflictStrategy: "skip", OrganizationsOnly: true, UsersOnly: true},
			wantErr: "cannot specify multiple exclusive flags",
		},
		{
			name:    "overwrite-all without force",
			flags:   Flags{ConflictStrategy: "skip", OverwriteAll: true},
			wantErr: "--overwrite-all requires --force",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePullFlags(tt.flags)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
