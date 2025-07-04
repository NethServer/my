/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package syncmd

import (
	"os"

	"github.com/spf13/viper"

	"github.com/nethesis/my/sync/internal/sync"
)

// OutputResult outputs synchronization results in the specified format
func OutputResult(result *sync.Result) error {
	format := viper.GetString("output")

	switch format {
	case "json":
		return result.OutputJSON(os.Stdout)
	case "yaml":
		return result.OutputYAML(os.Stdout)
	default:
		return result.OutputText(os.Stdout)
	}
}
