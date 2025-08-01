/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package initcmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAndGetConfig(t *testing.T) {
	// Save original environment
	originalEnvs := make(map[string]string)
	envVars := []string{"TENANT_ID", "BACKEND_APP_ID", "BACKEND_APP_SECRET", "TENANT_DOMAIN", "APP_URL"}
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

	t.Run("CLI mode - all flags provided", func(t *testing.T) {
		// Clear environment
		for _, env := range envVars {
			_ = os.Unsetenv(env)
		}

		config, err := ValidateAndGetConfig("cli-tenant", "cli-client", "cli-secret", "cli-domain.com", "https://cli-app.com")
		require.NoError(t, err)

		assert.Equal(t, "cli-tenant", config.TenantID)
		assert.Equal(t, "cli-domain.com", config.TenantDomain)
		assert.Equal(t, "https://cli-app.com", config.AppURL)
		assert.Equal(t, "cli-client", config.BackendAppID)
		assert.Equal(t, "cli-secret", config.BackendAppSecret)
		assert.Equal(t, "cli", config.Mode)
	})

	t.Run("CLI mode - missing flags", func(t *testing.T) {
		// Clear environment
		for _, env := range envVars {
			_ = os.Unsetenv(env)
		}

		config, err := ValidateAndGetConfig("cli-tenant", "", "", "", "")
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "all must be provided")
	})

	t.Run("environment mode - all vars set", func(t *testing.T) {
		// Set environment variables
		_ = os.Setenv("TENANT_ID", "env-tenant")
		_ = os.Setenv("TENANT_DOMAIN", "env-domain.com")
		_ = os.Setenv("BACKEND_APP_ID", "env-client")
		_ = os.Setenv("BACKEND_APP_SECRET", "env-secret")
		_ = os.Setenv("APP_URL", "https://env-app.com")

		config, err := ValidateAndGetConfig("", "", "", "", "")
		require.NoError(t, err)

		assert.Equal(t, "env-tenant", config.TenantID)
		assert.Equal(t, "env-domain.com", config.TenantDomain)
		assert.Equal(t, "https://env-app.com", config.AppURL)
		assert.Equal(t, "env-client", config.BackendAppID)
		assert.Equal(t, "env-secret", config.BackendAppSecret)
		assert.Equal(t, "env", config.Mode)
	})

	t.Run("environment mode - missing vars", func(t *testing.T) {
		// Clear environment
		for _, env := range envVars {
			_ = os.Unsetenv(env)
		}

		config, err := ValidateAndGetConfig("", "", "", "", "")
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "required environment variables missing")
	})

	t.Run("CLI flags take precedence over environment", func(t *testing.T) {
		// Set environment variables
		_ = os.Setenv("TENANT_ID", "env-tenant")
		_ = os.Setenv("TENANT_DOMAIN", "env-domain.com")
		_ = os.Setenv("BACKEND_APP_ID", "env-client")
		_ = os.Setenv("BACKEND_APP_SECRET", "env-secret")
		_ = os.Setenv("APP_URL", "https://env-app.com")

		// Override with CLI flags
		config, err := ValidateAndGetConfig("cli-tenant", "cli-client", "cli-secret", "cli-domain.com", "https://cli-app.com")
		require.NoError(t, err)

		// Should use CLI values, not environment
		assert.Equal(t, "cli-tenant", config.TenantID)
		assert.Equal(t, "cli-domain.com", config.TenantDomain)
		assert.Equal(t, "https://cli-app.com", config.AppURL)
		assert.Equal(t, "cli-client", config.BackendAppID)
		assert.Equal(t, "cli-secret", config.BackendAppSecret)
		assert.Equal(t, "cli", config.Mode)
	})
}

func TestInitConfig(t *testing.T) {
	t.Run("InitConfig structure", func(t *testing.T) {
		config := InitConfig{
			TenantID:         "test-tenant",
			TenantDomain:     "test-domain.com",
			AppURL:           "https://test-app.com",
			BackendAppID:     "test-client",
			BackendAppSecret: "test-secret",
			Mode:             "cli",
		}

		assert.Equal(t, "test-tenant", config.TenantID)
		assert.Equal(t, "test-domain.com", config.TenantDomain)
		assert.Equal(t, "https://test-app.com", config.AppURL)
		assert.Equal(t, "test-client", config.BackendAppID)
		assert.Equal(t, "test-secret", config.BackendAppSecret)
		assert.Equal(t, "cli", config.Mode)
	})
}
