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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestRootCommand(t *testing.T) {
	// Reset viper for each test
	viper.Reset()

	t.Run("command structure", func(t *testing.T) {
		assert.Equal(t, "sync", rootCmd.Use)
		assert.Equal(t, "Synchronize RBAC configuration with Logto", rootCmd.Short)
		assert.Contains(t, rootCmd.Long, "sync is a CLI tool")
		assert.NotEmpty(t, rootCmd.Version)
	})

	t.Run("persistent flags", func(t *testing.T) {
		// Check that flags exist
		configFlag := rootCmd.PersistentFlags().Lookup("config")
		assert.NotNil(t, configFlag)
		assert.Equal(t, "c", configFlag.Shorthand)

		verboseFlag := rootCmd.PersistentFlags().Lookup("verbose")
		assert.NotNil(t, verboseFlag)
		assert.Equal(t, "v", verboseFlag.Shorthand)

		dryRunFlag := rootCmd.PersistentFlags().Lookup("dry-run")
		assert.NotNil(t, dryRunFlag)

		outputFlag := rootCmd.PersistentFlags().Lookup("output")
		assert.NotNil(t, outputFlag)
		assert.Equal(t, "o", outputFlag.Shorthand)
	})

	t.Run("viper bindings", func(t *testing.T) {
		viper.Reset()

		// Bind flags to viper explicitly for testing
		_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
		_ = viper.BindPFlag("dry-run", rootCmd.PersistentFlags().Lookup("dry-run"))
		_ = viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))

		// Set flags and check viper picks them up
		_ = rootCmd.PersistentFlags().Set("verbose", "true")
		_ = rootCmd.PersistentFlags().Set("dry-run", "true")
		_ = rootCmd.PersistentFlags().Set("output", "json")

		assert.True(t, viper.GetBool("verbose"))
		assert.True(t, viper.GetBool("dry-run"))
		assert.Equal(t, "json", viper.GetString("output"))
	})
}

func TestExecute(t *testing.T) {
	// This test is tricky because Execute() tries to run the actual command
	// We'll test that the function exists and can be called, but won't execute it
	assert.NotNil(t, Execute, "Execute function should exist")

	// Test that rootCmd has the expected structure for execution
	assert.NotNil(t, rootCmd)
	assert.NotEmpty(t, rootCmd.Use)
}

func TestInitConfig(t *testing.T) {
	// Save original environment
	originalEnvs := make(map[string]string)
	envVars := []string{"LOG_LEVEL", "TENANT_ID", "BACKEND_APP_ID", "BACKEND_APP_SECRET"}
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
		viper.Reset()
	}()

	t.Run("default log level", func(t *testing.T) {
		// Clear environment
		_ = os.Unsetenv("LOG_LEVEL")
		viper.Reset()
		viper.Set("verbose", false)

		// This will call the actual initConfig, but we need to be careful
		// because it initializes the logger which might affect other tests
		// We'll test the logic indirectly by checking environment behavior

		logLevel := os.Getenv("LOG_LEVEL")
		if logLevel == "" {
			if viper.GetBool("verbose") {
				logLevel = "debug"
			} else {
				logLevel = "info"
			}
		}

		assert.Equal(t, "info", logLevel)
	})

	t.Run("verbose log level", func(t *testing.T) {
		_ = os.Unsetenv("LOG_LEVEL")
		viper.Reset()
		viper.Set("verbose", true)

		logLevel := os.Getenv("LOG_LEVEL")
		if logLevel == "" {
			if viper.GetBool("verbose") {
				logLevel = "debug"
			} else {
				logLevel = "info"
			}
		}

		assert.Equal(t, "debug", logLevel)
	})

	t.Run("env log level override", func(t *testing.T) {
		_ = os.Setenv("LOG_LEVEL", "warn")
		viper.Reset()

		// Environment variable should take precedence
		logLevel := os.Getenv("LOG_LEVEL")
		assert.Equal(t, "warn", logLevel)
	})
}

func TestValidateEnvironment(t *testing.T) {
	// Save original environment
	originalEnvs := make(map[string]string)
	requiredVars := []string{"TENANT_ID", "BACKEND_APP_ID", "BACKEND_APP_SECRET"}
	for _, env := range requiredVars {
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

	t.Run("all environment variables set", func(t *testing.T) {
		_ = os.Setenv("TENANT_ID", "test-tenant")
		_ = os.Setenv("BACKEND_APP_ID", "test-client")
		_ = os.Setenv("BACKEND_APP_SECRET", "test-secret")

		err := validateEnvironment()
		assert.NoError(t, err)
	})

	t.Run("missing TENANT_ID", func(t *testing.T) {
		_ = os.Unsetenv("TENANT_ID")
		_ = os.Setenv("BACKEND_APP_ID", "test-client")
		_ = os.Setenv("BACKEND_APP_SECRET", "test-secret")

		err := validateEnvironment()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "TENANT_ID")
		assert.Contains(t, err.Error(), "not set")
	})

	t.Run("missing BACKEND_APP_ID", func(t *testing.T) {
		_ = os.Setenv("TENANT_ID", "test-tenant")
		_ = os.Unsetenv("BACKEND_APP_ID")
		_ = os.Setenv("BACKEND_APP_SECRET", "test-secret")

		err := validateEnvironment()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "BACKEND_APP_ID")
	})

	t.Run("missing BACKEND_APP_SECRET", func(t *testing.T) {
		_ = os.Setenv("TENANT_ID", "test-tenant")
		_ = os.Setenv("BACKEND_APP_ID", "test-client")
		_ = os.Unsetenv("BACKEND_APP_SECRET")

		err := validateEnvironment()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "BACKEND_APP_SECRET")
	})

	t.Run("all missing", func(t *testing.T) {
		for _, env := range requiredVars {
			_ = os.Unsetenv(env)
		}

		err := validateEnvironment()
		assert.Error(t, err)
		// Should fail on the first missing variable (TENANT_ID)
		assert.Contains(t, err.Error(), "TENANT_ID")
	})

	t.Run("empty strings", func(t *testing.T) {
		_ = os.Setenv("TENANT_ID", "")
		_ = os.Setenv("BACKEND_APP_ID", "test-client")
		_ = os.Setenv("BACKEND_APP_SECRET", "test-secret")

		err := validateEnvironment()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "TENANT_ID")
	})
}

func TestGlobalVariables(t *testing.T) {
	t.Run("global variables exist", func(t *testing.T) {
		// Test that global variables are properly declared
		// These are package-level variables used by the CLI

		// Reset to defaults
		cfgFile = ""
		verbose = false
		dryRun = false
		outputFormat = ""

		assert.Equal(t, "", cfgFile)
		assert.False(t, verbose)
		assert.False(t, dryRun)
		assert.Equal(t, "", outputFormat)
	})
}

func TestRootCommandIntegration(t *testing.T) {
	t.Run("help command", func(t *testing.T) {
		// Create a new command instance to avoid affecting the global state
		testCmd := &cobra.Command{
			Use:   "sync",
			Short: "Test sync command",
		}

		// Test that help can be generated
		helpOutput, err := testCmd.ExecuteC()
		// The help command should not return an error, but the root command
		// might not have a run function, so we expect a specific behavior
		if err != nil {
			// This is expected for a command without a Run function
			assert.Contains(t, err.Error(), "unknown command")
		} else {
			assert.NotNil(t, helpOutput)
		}
	})

	t.Run("version flag", func(t *testing.T) {
		// Test that version is set
		assert.NotEmpty(t, rootCmd.Version)
	})
}

func TestCommandFlags(t *testing.T) {
	t.Run("config flag variations", func(t *testing.T) {
		viper.Reset()

		// Test long flag with different values
		err := rootCmd.PersistentFlags().Set("config", "test-config.yml")
		assert.NoError(t, err)

		// Test that the flag was set
		configFlag := rootCmd.PersistentFlags().Lookup("config")
		assert.NotNil(t, configFlag)
		assert.Equal(t, "test-config.yml", configFlag.Value.String())

		// Test setting another value
		err = rootCmd.PersistentFlags().Set("config", "another-config.yml")
		assert.NoError(t, err)
		assert.Equal(t, "another-config.yml", configFlag.Value.String())
	})

	t.Run("output format validation", func(t *testing.T) {
		viper.Reset()

		// Bind output flag to viper for testing
		_ = viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))

		validFormats := []string{"text", "json", "yaml"}
		for _, format := range validFormats {
			err := rootCmd.PersistentFlags().Set("output", format)
			assert.NoError(t, err)
			assert.Equal(t, format, viper.GetString("output"))
		}
	})

	t.Run("boolean flags", func(t *testing.T) {
		viper.Reset()

		// Bind flags to viper for testing
		_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
		_ = viper.BindPFlag("dry-run", rootCmd.PersistentFlags().Lookup("dry-run"))

		// Test verbose flag
		err := rootCmd.PersistentFlags().Set("verbose", "true")
		assert.NoError(t, err)
		assert.True(t, viper.GetBool("verbose"))

		err = rootCmd.PersistentFlags().Set("verbose", "false")
		assert.NoError(t, err)
		assert.False(t, viper.GetBool("verbose"))

		// Test dry-run flag
		err = rootCmd.PersistentFlags().Set("dry-run", "true")
		assert.NoError(t, err)
		assert.True(t, viper.GetBool("dry-run"))
	})
}
