/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package pushcmd

import (
	"fmt"

	"github.com/spf13/viper"
)

// ValidatePushFlags validates the push command flags for consistency
func ValidatePushFlags() error {
	organizationsOnly := viper.GetBool("organizations-only")
	usersOnly := viper.GetBool("users-only")

	if organizationsOnly && usersOnly {
		return fmt.Errorf("cannot specify both --organizations-only and --users-only")
	}

	// Validate SMTP config when --send-email is requested
	if viper.GetBool("send-email") {
		if viper.GetString("smtp-host") == "" {
			return fmt.Errorf("--smtp-host is required when --send-email is set")
		}
		if viper.GetString("smtp-from") == "" {
			return fmt.Errorf("--smtp-from is required when --send-email is set")
		}
	}

	return nil
}
