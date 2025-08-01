/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		assert.Equal(t, "init", initCmd.Use)
		assert.Equal(t, "🚀 Initialize Logto configuration with complete setup", initCmd.Short)
		assert.Contains(t, initCmd.Long, "Initialize Logto with complete configuration for Operation Center")
		assert.NotNil(t, initCmd.RunE)
	})

	t.Run("init flags", func(t *testing.T) {
		flags := []string{
			"force",
			"logto-domain",
			"app-url",
			"tenant-id",
			"backend-app-id",
			"backend-app-secret",
			"owner-username",
			"owner-email",
			"owner-name",
		}

		for _, flagName := range flags {
			flag := initCmd.Flags().Lookup(flagName)
			assert.NotNil(t, flag, "Flag %s should exist", flagName)
		}
	})

	t.Run("default flag values", func(t *testing.T) {
		usernameFlag := initCmd.Flags().Lookup("owner-username")
		assert.Equal(t, "owner", usernameFlag.DefValue)

		emailFlag := initCmd.Flags().Lookup("owner-email")
		assert.Equal(t, "owner@example.com", emailFlag.DefValue)

		nameFlag := initCmd.Flags().Lookup("owner-name")
		assert.Equal(t, "Company Owner", nameFlag.DefValue)
	})
}

func TestInitCommandInit(t *testing.T) {
	t.Run("command is added to root", func(t *testing.T) {
		// Check that initCmd is properly added to rootCmd
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == "init" {
				found = true
				break
			}
		}
		assert.True(t, found, "init command should be added to root command")
	})
}
