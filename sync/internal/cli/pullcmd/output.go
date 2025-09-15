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
	"os"

	"github.com/spf13/viper"

	"github.com/nethesis/my/sync/internal/sync"
)

// OutputResult outputs the pull result in the specified format
func OutputResult(result *sync.PullResult) error {
	outputFormat := viper.GetString("output")

	switch outputFormat {
	case "json":
		return result.OutputJSON(os.Stdout)
	case "yaml":
		return result.OutputYAML(os.Stdout)
	case "text":
		return result.OutputText(os.Stdout)
	default:
		return fmt.Errorf("unsupported output format: %s (supported: text, json, yaml)", outputFormat)
	}
}
