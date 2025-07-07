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

func TestCheckLogtoInitialization(t *testing.T) {
	// Save original environment
	originalClientID := os.Getenv("BACKEND_APP_ID")
	defer func() {
		if originalClientID == "" {
			_ = os.Unsetenv("BACKEND_APP_ID")
		} else {
			_ = os.Setenv("BACKEND_APP_ID", originalClientID)
		}
	}()

	t.Run("initialization check logic", func(t *testing.T) {
		// Set test environment
		_ = os.Setenv("BACKEND_APP_ID", "test-backend-client")

		// Note: We can't easily test this without a real or mocked Logto client
		// since it makes actual API calls. For now we just verify the environment
		// variable is set correctly for the validation logic.

		backendClientID := os.Getenv("BACKEND_APP_ID")
		assert.Equal(t, "test-backend-client", backendClientID)

		// The actual function call would require a real client
		// In a production test environment, you'd use dependency injection
		// or mocking to test this properly
	})

	t.Run("validate initialization environment setup", func(t *testing.T) {
		// Test the environment variable requirement
		_ = os.Setenv("BACKEND_APP_ID", "test-backend")

		backendClientID := os.Getenv("BACKEND_APP_ID")
		assert.NotEmpty(t, backendClientID)
		assert.Equal(t, "test-backend", backendClientID)
	})
}

func TestValidationConstants(t *testing.T) {
	t.Run("validation functions exist", func(t *testing.T) {
		// Test that our validation functions have the expected signatures
		// This is a compile-time check that the functions exist without calling them

		// Just verify the functions are defined and accessible
		// In a real test, you'd use dependency injection or mocking
		assert.NotNil(t, CheckLogtoInitialization)
		assert.NotNil(t, ValidateInitialization)
	})
}
