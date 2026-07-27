/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package version

import (
	_ "embed"
	"fmt"
	"runtime"
	"strings"
)

//go:embed VERSION
var versionRaw string

// Build information. Populated at build-time via ldflags.
var (
	Version   = strings.TrimSpace(versionRaw)
	Commit    = "unknown"
	BuildTime = "unknown"
)

// Info represents version information
type Info struct {
	Version   string `json:"version" yaml:"version"`
	Commit    string `json:"commit" yaml:"commit"`
	BuildTime string `json:"build_time" yaml:"build_time"`
	GoVersion string `json:"go_version" yaml:"go_version"`
	Platform  string `json:"platform" yaml:"platform"`
}

// Get returns version information
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a human-readable version string
func (i Info) String() string {
	return fmt.Sprintf("%s (%s) built at %s with %s for %s",
		i.Version, i.Commit, i.BuildTime, i.GoVersion, i.Platform)
}
