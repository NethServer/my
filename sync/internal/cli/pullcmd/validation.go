/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package pullcmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Flags holds the pull-specific flags. They are read from the command, not
// from viper: push and sync define flags with the same names, and viper keeps
// a single binding per key, so a viper read would return their values.
type Flags struct {
	OrganizationsOnly bool
	UsersOnly         bool
	ResourcesOnly     bool
	ConflictStrategy  string
	OverwriteAll      bool
	Force             bool
}

// FlagsFromCommand reads the pull-specific flags of cmd
func FlagsFromCommand(cmd *cobra.Command) (Flags, error) {
	var flags Flags
	var err error
	f := cmd.Flags()

	if flags.OrganizationsOnly, err = f.GetBool("organizations-only"); err != nil {
		return flags, err
	}
	if flags.UsersOnly, err = f.GetBool("users-only"); err != nil {
		return flags, err
	}
	if flags.ResourcesOnly, err = f.GetBool("resources-only"); err != nil {
		return flags, err
	}
	if flags.ConflictStrategy, err = f.GetString("conflict-strategy"); err != nil {
		return flags, err
	}
	if flags.OverwriteAll, err = f.GetBool("overwrite-all"); err != nil {
		return flags, err
	}
	if flags.Force, err = f.GetBool("force"); err != nil {
		return flags, err
	}
	return flags, nil
}

// ValidatePullFlags validates the pull command flags for consistency
func ValidatePullFlags(flags Flags) error {
	// Validate conflict strategy
	validStrategies := []string{"skip", "overwrite", "merge"}

	isValidStrategy := false
	for _, strategy := range validStrategies {
		if flags.ConflictStrategy == strategy {
			isValidStrategy = true
			break
		}
	}

	if !isValidStrategy {
		return fmt.Errorf("invalid conflict strategy '%s'. Valid options: skip, overwrite, merge", flags.ConflictStrategy)
	}

	// Validate mutually exclusive flags
	exclusiveCount := 0
	if flags.OrganizationsOnly {
		exclusiveCount++
	}
	if flags.UsersOnly {
		exclusiveCount++
	}
	if flags.ResourcesOnly {
		exclusiveCount++
	}

	if exclusiveCount > 1 {
		return fmt.Errorf("cannot specify multiple exclusive flags: --organizations-only, --users-only, --resources-only")
	}

	// Warn about dangerous options
	if flags.OverwriteAll && !flags.Force {
		return fmt.Errorf("--overwrite-all requires --force flag due to its destructive nature")
	}

	return nil
}
