/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/sync/internal/cli/pullcmd"
)

// push and sync define --organizations-only, --users-only and --force too:
// pull must read its own values, whatever the other commands hold
func TestPullReadsItsOwnFlags(t *testing.T) {
	t.Cleanup(func() {
		for _, name := range []string{"organizations-only", "users-only", "force"} {
			_ = pullCmd.Flags().Set(name, "false")
			_ = pushCmd.Flags().Set(name, "false")
			_ = syncCmd.Flags().Set(name, "false")
		}
	})

	require.NoError(t, pullCmd.ParseFlags([]string{"--organizations-only", "--force"}))
	require.NoError(t, pushCmd.ParseFlags([]string{"--users-only"}))

	flags, err := pullcmd.FlagsFromCommand(pullCmd)
	require.NoError(t, err)

	assert.True(t, flags.OrganizationsOnly)
	assert.False(t, flags.UsersOnly)
	assert.True(t, flags.Force)
	assert.Equal(t, "skip", flags.ConflictStrategy)
}
