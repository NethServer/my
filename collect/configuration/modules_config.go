/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package configuration

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/nethesis/my/collect/logger"
)

// ModulesConfig holds the module visibility configuration
type ModulesConfig struct {
	Modules struct {
		SystemModules []string `yaml:"system_modules"`
	} `yaml:"modules"`
	ApplicationURL struct {
		Pattern         string `yaml:"pattern"`
		FallbackPattern string `yaml:"fallback_pattern"`
	} `yaml:"application_url"`
	Inventory struct {
		Types struct {
			NS8  string `yaml:"ns8"`
			NSEC string `yaml:"nsec"`
		} `yaml:"types"`
	} `yaml:"inventory"`
}

var (
	modulesConfig     *ModulesConfig
	modulesConfigOnce sync.Once
	systemModulesSet  map[string]bool
)

// LoadModulesConfig loads the modules configuration from config.yml
func LoadModulesConfig() (*ModulesConfig, error) {
	var loadErr error

	modulesConfigOnce.Do(func() {
		modulesConfig = &ModulesConfig{}

		// Search for config.yml in standard locations
		configPaths := []string{
			"config.yml",
			"./config.yml",
			"/etc/collect/config.yml",
		}

		// Add path relative to executable
		if execPath, err := os.Executable(); err == nil {
			configPaths = append(configPaths, filepath.Join(filepath.Dir(execPath), "config.yml"))
		}

		var configData []byte
		var configPath string

		for _, path := range configPaths {
			if data, err := os.ReadFile(path); err == nil {
				configData = data
				configPath = path
				break
			}
		}

		if configData == nil {
			loadErr = fmt.Errorf("config.yml not found in any of the search paths")
			logger.Error().Msg("config.yml not found: the file is required")
			return
		}

		if err := yaml.Unmarshal(configData, modulesConfig); err != nil {
			loadErr = fmt.Errorf("failed to parse config.yml: %w", err)
			logger.Error().Err(err).Str("path", configPath).Msg("Failed to parse config.yml")
			return
		}

		// Build system modules set for fast lookup
		systemModulesSet = make(map[string]bool)
		for _, module := range modulesConfig.Modules.SystemModules {
			systemModulesSet[module] = true
		}

		logger.Info().
			Str("path", configPath).
			Int("system_modules", len(modulesConfig.Modules.SystemModules)).
			Msg("Loaded modules configuration")
	})

	return modulesConfig, loadErr
}

// GetModulesConfig returns the loaded modules configuration
func GetModulesConfig() *ModulesConfig {
	if modulesConfig == nil {
		_, _ = LoadModulesConfig()
	}
	return modulesConfig
}

// IsSystemModule checks if a module is a system module (not user-facing)
func IsSystemModule(moduleName string) bool {
	if systemModulesSet == nil {
		_, _ = LoadModulesConfig()
	}
	return systemModulesSet[moduleName]
}

// IsUserFacingModule checks if a module should be shown in the UI
func IsUserFacingModule(moduleName string) bool {
	return !IsSystemModule(moduleName)
}

// GetApplicationURL generates the URL for an application
func GetApplicationURL(fqdn, moduleID string) string {
	if fqdn == "" {
		return ""
	}

	config := GetModulesConfig()
	if config.ApplicationURL.Pattern == "" {
		return ""
	}

	// Simple template replacement
	url := config.ApplicationURL.Pattern
	url = replaceAll(url, "{fqdn}", fqdn)
	url = replaceAll(url, "{module_id}", moduleID)

	return url
}

// replaceAll is a simple string replacement helper
func replaceAll(s, old, new string) string {
	result := s
	for {
		idx := indexOf(result, old)
		if idx < 0 {
			break
		}
		result = result[:idx] + new + result[idx+len(old):]
	}
	return result
}

// indexOf finds the index of a substring
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
