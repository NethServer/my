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
	"fmt"
	"os"

	"github.com/nethesis/my/sync/internal/logger"
)

// InitConfig holds the configuration for the init command
type InitConfig struct {
	TenantID         string
	TenantDomain     string
	BackendAppID     string
	BackendAppSecret string
	Mode             string // "env" or "cli"
}

// ValidateAndGetConfig validates and returns the configuration for the init command
func ValidateAndGetConfig(tenantID, backendAppID, backendAppSecret, domain string) (*InitConfig, error) {
	// Check if any CLI flags are provided
	hasCLIFlags := tenantID != "" || backendAppID != "" || backendAppSecret != "" || domain != ""

	if hasCLIFlags {
		// CLI mode - all flags must be provided
		if tenantID == "" || backendAppID == "" || backendAppSecret == "" || domain == "" {
			return nil, fmt.Errorf("when using CLI flags, all must be provided:\n" +
				"  --tenant-id, --backend-app-id, --backend-app-secret, --domain\n" +
				"Or use environment variables: TENANT_ID, BACKEND_APP_ID, BACKEND_APP_SECRET, TENANT_DOMAIN")
		}

		logger.Info("Using CLI mode")
		return &InitConfig{
			TenantID:         tenantID,
			TenantDomain:     domain,
			BackendAppID:     backendAppID,
			BackendAppSecret: backendAppSecret,
			Mode:             "cli",
		}, nil
	}

	// Environment mode - check all required env vars
	envTenantID := os.Getenv("TENANT_ID")
	envBackendAppID := os.Getenv("BACKEND_APP_ID")
	envBackendAppSecret := os.Getenv("BACKEND_APP_SECRET")
	envTenantDomain := os.Getenv("TENANT_DOMAIN")

	if envTenantID == "" || envBackendAppID == "" || envBackendAppSecret == "" || envTenantDomain == "" {
		return nil, fmt.Errorf("required environment variables missing:\n" +
			"  TENANT_ID, BACKEND_APP_ID, BACKEND_APP_SECRET, TENANT_DOMAIN\n" +
			"Or use CLI flags: --tenant-id, --backend-app-id, --backend-app-secret, --domain")
	}

	logger.Info("Using environment variables mode")
	return &InitConfig{
		TenantID:         envTenantID,
		TenantDomain:     envTenantDomain,
		BackendAppID:     envBackendAppID,
		BackendAppSecret: envBackendAppSecret,
		Mode:             "env",
	}, nil
}
