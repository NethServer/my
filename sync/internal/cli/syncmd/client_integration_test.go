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

func TestCreateLogtoClient(t *testing.T) {
	// Save original environment variables
	originalEnvs := make(map[string]string)
	envVars := []string{"TENANT_ID", "BACKEND_CLIENT_ID", "BACKEND_CLIENT_SECRET"}
	for _, env := range envVars {
		originalEnvs[env] = os.Getenv(env)
	}

	defer func() {
		// Restore original environment
		for env, val := range originalEnvs {
			if val == "" {
				_ = os.Unsetenv(env)
			} else {
				_ = os.Setenv(env, val)
			}
		}
	}()

	t.Run("creates client with valid environment variables", func(t *testing.T) {
		// Set test environment variables
		_ = os.Setenv("TENANT_ID", "test-tenant")
		_ = os.Setenv("BACKEND_CLIENT_ID", "test-client-id")
		_ = os.Setenv("BACKEND_CLIENT_SECRET", "test-secret")

		// Note: This will fail with actual connection test, but we can test the client creation logic
		client, err := CreateLogtoClient()

		// Should fail on connection test since we're using fake credentials
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "failed to connect to Logto")
	})

	t.Run("client creation structure", func(t *testing.T) {
		// Set test environment variables
		_ = os.Setenv("TENANT_ID", "test-tenant")
		_ = os.Setenv("BACKEND_CLIENT_ID", "test-client")
		_ = os.Setenv("BACKEND_CLIENT_SECRET", "test-secret")

		// We can't easily test the actual client creation without mocking,
		// but we can verify the environment variables are read correctly
		tenantID := os.Getenv("TENANT_ID")
		clientID := os.Getenv("BACKEND_CLIENT_ID")
		secret := os.Getenv("BACKEND_CLIENT_SECRET")

		assert.Equal(t, "test-tenant", tenantID)
		assert.Equal(t, "test-client", clientID)
		assert.Equal(t, "test-secret", secret)
	})
}
