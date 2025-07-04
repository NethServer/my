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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAPIBaseURL(t *testing.T) {
	// Save original environment
	originalAPIBaseURL := os.Getenv("API_BASE_URL")
	defer func() {
		if originalAPIBaseURL == "" {
			_ = os.Unsetenv("API_BASE_URL")
		} else {
			_ = os.Setenv("API_BASE_URL", originalAPIBaseURL)
		}
	}()

	t.Run("returns default when env var not set", func(t *testing.T) {
		_ = os.Unsetenv("API_BASE_URL")

		baseURL := GetAPIBaseURL()
		assert.Equal(t, "http://localhost:8080", baseURL)
	})

	t.Run("returns custom URL from environment", func(t *testing.T) {
		testURL := "https://api.example.com"
		_ = os.Setenv("API_BASE_URL", testURL)

		baseURL := GetAPIBaseURL()
		assert.Equal(t, testURL, baseURL)
	})

	t.Run("returns default when env var is empty", func(t *testing.T) {
		_ = os.Setenv("API_BASE_URL", "")

		baseURL := GetAPIBaseURL()
		assert.Equal(t, "http://localhost:8080", baseURL)
	})

	t.Run("handles various URL formats", func(t *testing.T) {
		testCases := []struct {
			envValue string
			expected string
		}{
			{"https://api.production.com", "https://api.production.com"},
			{"http://localhost:3000", "http://localhost:3000"},
			{"https://staging-api.company.io:8443", "https://staging-api.company.io:8443"},
		}

		for _, tc := range testCases {
			_ = os.Setenv("API_BASE_URL", tc.envValue)
			baseURL := GetAPIBaseURL()
			assert.Equal(t, tc.expected, baseURL, "Expected %s for env value %s", tc.expected, tc.envValue)
		}
	})
}
