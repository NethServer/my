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

	"github.com/spf13/viper"
)

// ValidatePullFlags validates the pull command flags for consistency
func ValidatePullFlags() error {
	// Validate conflict strategy
	conflictStrategy := viper.GetString("conflict-strategy")
	validStrategies := []string{"skip", "overwrite", "merge"}

	isValidStrategy := false
	for _, strategy := range validStrategies {
		if conflictStrategy == strategy {
			isValidStrategy = true
			break
		}
	}

	if !isValidStrategy {
		return fmt.Errorf("invalid conflict strategy '%s'. Valid options: skip, overwrite, merge", conflictStrategy)
	}

	// Validate mutually exclusive flags
	organizationsOnly := viper.GetBool("organizations-only")
	usersOnly := viper.GetBool("users-only")
	resourcesOnly := viper.GetBool("resources-only")

	exclusiveCount := 0
	if organizationsOnly {
		exclusiveCount++
	}
	if usersOnly {
		exclusiveCount++
	}
	if resourcesOnly {
		exclusiveCount++
	}

	if exclusiveCount > 1 {
		return fmt.Errorf("cannot specify multiple exclusive flags: --organizations-only, --users-only, --resources-only")
	}

	// Warn about dangerous options
	if viper.GetBool("overwrite-all") && !viper.GetBool("force") {
		return fmt.Errorf("--overwrite-all requires --force flag due to its destructive nature")
	}

	return nil
}
