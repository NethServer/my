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
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadAndValidateConfig(t *testing.T) {
	// Save original viper settings
	origForce := viper.GetBool("force")
	defer viper.Set("force", origForce)

	t.Run("valid config file loads successfully", func(t *testing.T) {
		// Create a temporary valid config file
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "config.yml")

		validConfig := `metadata:
  name: "test-config"
  version: "1.0.0"

resources:
  - name: "systems"
    actions: ["read", "manage"]

user_roles:
  - id: "admin"
    name: "Admin"
    permissions:
      - id: "manage:systems"

organization_roles:
  - id: "owner"
    name: "Owner"
    permissions:
      - id: "read:systems"`

		err := os.WriteFile(configFile, []byte(validConfig), 0644)
		require.NoError(t, err)

		cfg, err := LoadAndValidateConfig(configFile)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "test-config", cfg.Metadata.Name)
		assert.Equal(t, "1.0.0", cfg.Metadata.Version)
	})

	t.Run("empty config file path returns error", func(t *testing.T) {
		// Clear viper config file
		viper.SetConfigFile("")

		cfg, err := LoadAndValidateConfig("")
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "no configuration file specified")
	})

	t.Run("non-existent config file returns error", func(t *testing.T) {
		cfg, err := LoadAndValidateConfig("/non/existent/config.yml")
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to load configuration")
	})

	t.Run("invalid config validation fails without force", func(t *testing.T) {
		viper.Set("force", false)

		// Create a temporary invalid config file
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "invalid_config.yml")

		invalidConfig := `metadata:
  name: ""  # Invalid: empty name
  version: "1.0.0"`

		err := os.WriteFile(configFile, []byte(invalidConfig), 0644)
		require.NoError(t, err)

		cfg, err := LoadAndValidateConfig(configFile)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "configuration validation failed")
	})

	t.Run("invalid config passes validation with force", func(t *testing.T) {
		viper.Set("force", true)

		// Create a temporary invalid config file
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "invalid_config.yml")

		invalidConfig := `metadata:
  name: ""  # Invalid: empty name
  version: "1.0.0"`

		err := os.WriteFile(configFile, []byte(invalidConfig), 0644)
		require.NoError(t, err)

		cfg, err := LoadAndValidateConfig(configFile)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "", cfg.Metadata.Name) // Invalid name but force allows it
	})
}
