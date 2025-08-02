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
	"fmt"

	"github.com/spf13/viper"

	"github.com/nethesis/my/sync/internal/config"
	"github.com/nethesis/my/sync/internal/logger"
)

// LoadAndValidateConfig loads configuration from file and validates it
func LoadAndValidateConfig(configFile string) (*config.Config, error) {
	if configFile == "" {
		configFile = viper.ConfigFileUsed()
	}

	if configFile == "" {
		return nil, fmt.Errorf("no configuration file specified or found")
	}

	logger.Info("Loading configuration from: %s", configFile)
	cfg, err := config.LoadFromFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Log configuration loading with structured data
	resourceCount := len(cfg.Resources)
	roleCount := len(cfg.UserRoles) + len(cfg.OrganizationRoles)

	// Validate configuration
	if err := cfg.Validate(); err != nil && !viper.GetBool("force") {
		logger.LogConfigLoad(configFile, resourceCount, roleCount, false)
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	logger.LogConfigLoad(configFile, resourceCount, roleCount, true)
	return cfg, nil
}
