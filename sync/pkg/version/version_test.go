/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package version

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGet(t *testing.T) {
	// Save original values
	originalVersion := Version
	originalCommit := Commit
	originalBuildTime := BuildTime

	// Set test values
	Version = "1.2.3"
	Commit = "abc123def"
	BuildTime = "2023-01-01T12:00:00Z"

	defer func() {
		// Restore original values
		Version = originalVersion
		Commit = originalCommit
		BuildTime = originalBuildTime
	}()

	info := Get()

	assert.Equal(t, "1.2.3", info.Version)
	assert.Equal(t, "abc123def", info.Commit)
	assert.Equal(t, "2023-01-01T12:00:00Z", info.BuildTime)
	assert.Equal(t, runtime.Version(), info.GoVersion)
	assert.Equal(t, fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH), info.Platform)
}

func TestGetWithDefaultValues(t *testing.T) {
	// Test with default values (as they would be without ldflags)
	info := Get()

	// Check that all fields are populated
	assert.NotEmpty(t, info.Version)
	assert.NotEmpty(t, info.Commit)
	assert.NotEmpty(t, info.BuildTime)
	assert.NotEmpty(t, info.GoVersion)
	assert.NotEmpty(t, info.Platform)

	// Check that Go version starts with "go"
	assert.True(t, strings.HasPrefix(info.GoVersion, "go"))

	// Check platform format
	assert.Contains(t, info.Platform, "/")
	parts := strings.Split(info.Platform, "/")
	assert.Len(t, parts, 2)
	assert.NotEmpty(t, parts[0]) // OS
	assert.NotEmpty(t, parts[1]) // Arch
}

func TestInfoString(t *testing.T) {
	info := Info{
		Version:   "1.0.0",
		Commit:    "abcdef123456",
		BuildTime: "2023-06-15T10:30:45Z",
		GoVersion: "go1.20.5",
		Platform:  "linux/amd64",
	}

	result := info.String()
	expected := "1.0.0 (abcdef123456) built at 2023-06-15T10:30:45Z with go1.20.5 for linux/amd64"

	assert.Equal(t, expected, result)
}

func TestInfoStringWithSpecialCharacters(t *testing.T) {
	info := Info{
		Version:   "v1.0.0-beta+123",
		Commit:    "dirty-commit-hash",
		BuildTime: "2023-12-25T00:00:00Z",
		GoVersion: "go1.21.0",
		Platform:  "darwin/arm64",
	}

	result := info.String()
	expected := "v1.0.0-beta+123 (dirty-commit-hash) built at 2023-12-25T00:00:00Z with go1.21.0 for darwin/arm64"

	assert.Equal(t, expected, result)
}

func TestInfoJSONSerialization(t *testing.T) {
	info := Info{
		Version:   "2.1.0",
		Commit:    "fedcba987654",
		BuildTime: "2023-07-20T14:45:30Z",
		GoVersion: "go1.20.6",
		Platform:  "windows/amd64",
	}

	jsonData, err := json.Marshal(info)
	require.NoError(t, err)

	var unmarshaled Info
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, info, unmarshaled)

	// Check that JSON contains expected fields
	jsonStr := string(jsonData)
	assert.Contains(t, jsonStr, `"version":"2.1.0"`)
	assert.Contains(t, jsonStr, `"commit":"fedcba987654"`)
	assert.Contains(t, jsonStr, `"build_time":"2023-07-20T14:45:30Z"`)
	assert.Contains(t, jsonStr, `"go_version":"go1.20.6"`)
	assert.Contains(t, jsonStr, `"platform":"windows/amd64"`)
}

func TestInfoYAMLSerialization(t *testing.T) {
	info := Info{
		Version:   "3.0.0",
		Commit:    "1234567890ab",
		BuildTime: "2023-08-15T09:15:22Z",
		GoVersion: "go1.21.1",
		Platform:  "linux/arm64",
	}

	yamlData, err := yaml.Marshal(info)
	require.NoError(t, err)

	var unmarshaled Info
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, info, unmarshaled)

	// Check that YAML contains expected fields
	yamlStr := string(yamlData)
	assert.Contains(t, yamlStr, "version: 3.0.0")
	assert.Contains(t, yamlStr, "commit: 1234567890ab")
	assert.Contains(t, yamlStr, "build_time:")
	assert.Contains(t, yamlStr, "2023-08-15T09:15:22Z")
	assert.Contains(t, yamlStr, "go_version: go1.21.1")
	assert.Contains(t, yamlStr, "platform: linux/arm64")
}

func TestPlatformFormat(t *testing.T) {
	info := Get()

	// Test that platform has the expected format
	parts := strings.Split(info.Platform, "/")
	assert.Len(t, parts, 2, "Platform should be in format OS/ARCH")

	os := parts[0]
	arch := parts[1]

	// Common OS values
	validOS := []string{"linux", "darwin", "windows", "freebsd", "openbsd", "netbsd", "dragonfly", "solaris", "plan9", "js", "aix", "android", "illumos"}
	assert.Contains(t, validOS, os, "OS should be a valid GOOS value")

	// Common architecture values
	validArch := []string{"amd64", "386", "arm", "arm64", "ppc64", "ppc64le", "mips", "mipsle", "mips64", "mips64le", "s390x", "riscv64", "wasm"}
	assert.Contains(t, validArch, arch, "Architecture should be a valid GOARCH value")
}

func TestRuntimeValues(t *testing.T) {
	info := Get()

	// Test that runtime values are reasonable
	assert.True(t, strings.HasPrefix(info.GoVersion, "go"), "Go version should start with 'go'")

	// Go version should have at least major.minor format
	versionParts := strings.Split(strings.TrimPrefix(info.GoVersion, "go"), ".")
	assert.True(t, len(versionParts) >= 2, "Go version should have at least major.minor components")
}

func TestEmptyValues(t *testing.T) {
	info := Info{
		Version:   "",
		Commit:    "",
		BuildTime: "",
		GoVersion: "",
		Platform:  "",
	}

	result := info.String()
	expected := " () built at  with  for "

	assert.Equal(t, expected, result)
}

func TestBuildTimeVariation(t *testing.T) {
	// Save original values
	originalVersion := Version
	originalCommit := Commit
	originalBuildTime := BuildTime

	testCases := []struct {
		name      string
		version   string
		commit    string
		buildTime string
	}{
		{
			name:      "development build",
			version:   "dev",
			commit:    "unknown",
			buildTime: "unknown",
		},
		{
			name:      "release build",
			version:   "v1.0.0",
			commit:    "1a2b3c4d5e6f7890",
			buildTime: "2023-09-01T12:00:00Z",
		},
		{
			name:      "snapshot build",
			version:   "1.1.0-SNAPSHOT",
			commit:    "dirty",
			buildTime: "2023-09-01T12:05:30.123Z",
		},
	}

	defer func() {
		// Restore original values
		Version = originalVersion
		Commit = originalCommit
		BuildTime = originalBuildTime
	}()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			Version = tc.version
			Commit = tc.commit
			BuildTime = tc.buildTime

			info := Get()

			assert.Equal(t, tc.version, info.Version)
			assert.Equal(t, tc.commit, info.Commit)
			assert.Equal(t, tc.buildTime, info.BuildTime)

			// Ensure String() method works with all variations
			result := info.String()
			assert.Contains(t, result, tc.version)
			assert.Contains(t, result, tc.commit)
			assert.Contains(t, result, tc.buildTime)
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	// Test that Get() is safe for concurrent access
	done := make(chan Info, 10)

	for i := 0; i < 10; i++ {
		go func() {
			done <- Get()
		}()
	}

	// Collect all results
	var results []Info
	for i := 0; i < 10; i++ {
		results = append(results, <-done)
	}

	// All results should be identical
	first := results[0]
	for i := 1; i < len(results); i++ {
		assert.Equal(t, first, results[i], "Concurrent calls should return identical results")
	}
}
