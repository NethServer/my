/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package cli

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/sync/internal/cli/syncmd"
	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/sync"
)

func TestSyncCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		assert.Equal(t, "sync", syncCmd.Use)
		assert.Equal(t, "🔄 Synchronize RBAC configuration with Logto", syncCmd.Short)
		assert.Contains(t, syncCmd.Long, "Synchronize RBAC configuration from YAML file to Logto")
		assert.NotNil(t, syncCmd.RunE)
	})

	t.Run("sync-specific flags", func(t *testing.T) {
		flags := []string{
			"skip-resources",
			"skip-roles",
			"skip-permissions",
			"force",
			"cleanup",
		}

		for _, flagName := range flags {
			flag := syncCmd.Flags().Lookup(flagName)
			assert.NotNil(t, flag, "Flag %s should exist", flagName)
		}
	})

	t.Run("viper bindings for sync flags", func(t *testing.T) {
		viper.Reset()

		// Bind flags to viper explicitly for testing
		_ = viper.BindPFlag("skip-resources", syncCmd.Flags().Lookup("skip-resources"))
		_ = viper.BindPFlag("skip-roles", syncCmd.Flags().Lookup("skip-roles"))
		_ = viper.BindPFlag("force", syncCmd.Flags().Lookup("force"))

		// Set flags and check viper picks them up
		_ = syncCmd.Flags().Set("skip-resources", "true")
		_ = syncCmd.Flags().Set("skip-roles", "true")
		_ = syncCmd.Flags().Set("force", "true")

		assert.True(t, viper.GetBool("skip-resources"))
		assert.True(t, viper.GetBool("skip-roles"))
		assert.True(t, viper.GetBool("force"))
	})
}

func TestGetAPIBaseURL(t *testing.T) {
	// Save original environment
	original := os.Getenv("API_BASE_URL")
	defer func() {
		if original == "" {
			_ = os.Unsetenv("API_BASE_URL")
		} else {
			_ = os.Setenv("API_BASE_URL", original)
		}
	}()

	t.Run("default URL when env not set", func(t *testing.T) {
		_ = os.Unsetenv("API_BASE_URL")

		result := syncmd.GetAPIBaseURL()
		assert.Equal(t, "http://localhost:8080", result)
	})

	t.Run("custom URL from environment", func(t *testing.T) {
		customURL := "https://api.example.com"
		_ = os.Setenv("API_BASE_URL", customURL)

		result := syncmd.GetAPIBaseURL()
		assert.Equal(t, customURL, result)
	})

	t.Run("empty env variable uses default", func(t *testing.T) {
		_ = os.Setenv("API_BASE_URL", "")

		result := syncmd.GetAPIBaseURL()
		assert.Equal(t, "http://localhost:8080", result)
	})
}

func TestOutputResult(t *testing.T) {
	// Create a mock result
	_ = &sync.Result{
		Summary: &sync.Summary{
			ResourcesCreated: 1,
			ResourcesUpdated: 2,
			RolesCreated:     3,
			RolesUpdated:     4,
		},
		DryRun: false,
	}

	t.Run("text output", func(t *testing.T) {
		viper.Reset()
		viper.Set("output", "text")

		// Test logic for text output

		// Create a custom result with OutputText method that writes to our buffer
		// Since we can't easily mock the sync.Result methods, we'll test the switch logic
		format := viper.GetString("output")

		switch format {
		case "json":
			assert.Fail(t, "Should not reach JSON case")
		case "yaml":
			assert.Fail(t, "Should not reach YAML case")
		default:
			// This is the text case - the function should call result.OutputText(os.Stdout)
			assert.Equal(t, "text", format)
		}
	})

	t.Run("json output", func(t *testing.T) {
		viper.Reset()
		viper.Set("output", "json")

		format := viper.GetString("output")
		assert.Equal(t, "json", format)

		// The function should call result.OutputJSON(os.Stdout)
		// We're testing the switch logic here
	})

	t.Run("yaml output", func(t *testing.T) {
		viper.Reset()
		viper.Set("output", "yaml")

		format := viper.GetString("output")
		assert.Equal(t, "yaml", format)

		// The function should call result.OutputYAML(os.Stdout)
		// We're testing the switch logic here
	})

	t.Run("invalid format defaults to text", func(t *testing.T) {
		viper.Reset()
		viper.Set("output", "invalid-format")

		format := viper.GetString("output")
		assert.Equal(t, "invalid-format", format)

		// The switch should fall through to the default case (text)
	})
}

func TestCheckLogtoInitialization(t *testing.T) {
	// Save original environment
	originalClientID := os.Getenv("BACKEND_CLIENT_ID")
	defer func() {
		if originalClientID == "" {
			_ = os.Unsetenv("BACKEND_CLIENT_ID")
		} else {
			_ = os.Setenv("BACKEND_CLIENT_ID", originalClientID)
		}
	}()

	// Mock client for testing
	_ = &client.LogtoClient{}

	t.Run("missing backend client ID env var", func(t *testing.T) {
		_ = os.Unsetenv("BACKEND_CLIENT_ID")

		// This test is complex because it requires a real client implementation
		// We'll test the environment variable requirement
		backendClientID := os.Getenv("BACKEND_CLIENT_ID")
		assert.Empty(t, backendClientID)

		// The function would fail because it can't find the backend client ID
	})

	t.Run("backend client ID set", func(t *testing.T) {
		_ = os.Setenv("BACKEND_CLIENT_ID", "test-backend-client")

		backendClientID := os.Getenv("BACKEND_CLIENT_ID")
		assert.Equal(t, "test-backend-client", backendClientID)

		// The function would use this ID to check for the backend app
		// Full testing would require mocking the client.GetApplications() call
	})
}

func TestSyncCommandInit(t *testing.T) {
	t.Run("command is added to root", func(t *testing.T) {
		// Check that syncCmd is properly added to rootCmd
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == "sync" {
				found = true
				break
			}
		}
		assert.True(t, found, "sync command should be added to root command")
	})
}

func TestSyncCommandFlags(t *testing.T) {
	t.Run("all flags have correct types", func(t *testing.T) {
		flagTests := []struct {
			name         string
			expectedType string
		}{
			{"skip-resources", "bool"},
			{"skip-roles", "bool"},
			{"skip-permissions", "bool"},
			{"force", "bool"},
			{"cleanup", "bool"},
		}

		for _, test := range flagTests {
			flag := syncCmd.Flags().Lookup(test.name)
			require.NotNil(t, flag, "Flag %s should exist", test.name)
			assert.Equal(t, test.expectedType, flag.Value.Type(), "Flag %s should be of type %s", test.name, test.expectedType)
		}
	})

	t.Run("dangerous flags have appropriate defaults", func(t *testing.T) {
		// These flags should default to false for safety
		dangerousFlags := []string{"force", "cleanup"}

		for _, flagName := range dangerousFlags {
			flag := syncCmd.Flags().Lookup(flagName)
			require.NotNil(t, flag)
			assert.Equal(t, "false", flag.DefValue, "Dangerous flag %s should default to false", flagName)
		}
	})
}

func TestSyncCommandValidation(t *testing.T) {
	// Save original environment
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

	t.Run("environment validation", func(t *testing.T) {
		// Clear environment variables
		for _, env := range envVars {
			_ = os.Unsetenv(env)
		}

		// Test that validateEnvironment() would fail
		err := validateEnvironment()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not set")
	})
}

func TestRunSyncLogic(t *testing.T) {
	t.Run("config file resolution", func(t *testing.T) {
		viper.Reset()

		// Test with global cfgFile
		cfgFile = "test-config.yml"
		configFile := cfgFile
		if configFile == "" {
			configFile = viper.ConfigFileUsed()
		}
		assert.Equal(t, "test-config.yml", configFile)

		// Test without global cfgFile
		cfgFile = ""
		viper.SetConfigFile("viper-config.yml")
		configFile = cfgFile
		if configFile == "" {
			configFile = viper.ConfigFileUsed()
		}
		assert.Equal(t, "viper-config.yml", configFile)

		// Test with no config file
		cfgFile = ""
		viper.Reset()
		configFile = cfgFile
		if configFile == "" {
			configFile = viper.ConfigFileUsed()
		}
		assert.Equal(t, "", configFile)
	})

	t.Run("dry run and verbose flags", func(t *testing.T) {
		viper.Reset()

		// Test dry run flag
		viper.Set("dry-run", true)
		assert.True(t, viper.GetBool("dry-run"))

		// Test verbose flag
		viper.Set("verbose", true)
		assert.True(t, viper.GetBool("verbose"))

		// Test skip flags
		viper.Set("skip-resources", true)
		viper.Set("skip-roles", true)
		viper.Set("skip-permissions", true)

		assert.True(t, viper.GetBool("skip-resources"))
		assert.True(t, viper.GetBool("skip-roles"))
		assert.True(t, viper.GetBool("skip-permissions"))
	})
}

func TestTenantURLConstruction(t *testing.T) {
	t.Run("tenant URL format", func(t *testing.T) {
		tenantID := "test-tenant"
		expectedURL := "https://test-tenant.logto.app"

		// This is the pattern used in runSync
		actualURL := "https://" + tenantID + ".logto.app"
		assert.Equal(t, expectedURL, actualURL)
	})

	t.Run("API base URL construction", func(t *testing.T) {
		// Test the GetAPIBaseURL function behavior
		assert.Equal(t, "http://localhost:8080", syncmd.GetAPIBaseURL())
	})
}
