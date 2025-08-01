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
	AppURL           string
	BackendAppID     string
	BackendAppSecret string
	Mode             string // "env" or "cli"
}

// ValidateAndGetConfig validates and returns the configuration for the init command
func ValidateAndGetConfig(tenantID, backendAppID, backendAppSecret, logtoDomain, appURL string) (*InitConfig, error) {
	// Check if any CLI flags are provided
	hasCLIFlags := tenantID != "" || backendAppID != "" || backendAppSecret != "" || logtoDomain != "" || appURL != ""

	if hasCLIFlags {
		// CLI mode - all flags must be provided
		if tenantID == "" || backendAppID == "" || backendAppSecret == "" || logtoDomain == "" || appURL == "" {
			return nil, fmt.Errorf("when using CLI flags, all must be provided:\n" +
				"  --tenant-id, --backend-app-id, --backend-app-secret, --logto-domain, --app-url\n" +
				"Or use environment variables: TENANT_ID, BACKEND_APP_ID, BACKEND_APP_SECRET, TENANT_DOMAIN, APP_URL")
		}

		logger.Info("Using CLI mode")
		return &InitConfig{
			TenantID:         tenantID,
			TenantDomain:     logtoDomain,
			AppURL:           appURL,
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
	envAppURL := os.Getenv("APP_URL")

	if envTenantID == "" || envBackendAppID == "" || envBackendAppSecret == "" || envTenantDomain == "" || envAppURL == "" {
		return nil, fmt.Errorf("required environment variables missing:\n" +
			"  TENANT_ID, BACKEND_APP_ID, BACKEND_APP_SECRET, TENANT_DOMAIN, APP_URL\n" +
			"Or use CLI flags: --tenant-id, --backend-app-id, --backend-app-secret, --logto-domain, --app-url")
	}

	return &InitConfig{
		TenantID:         envTenantID,
		TenantDomain:     envTenantDomain,
		AppURL:           envAppURL,
		BackendAppID:     envBackendAppID,
		BackendAppSecret: envBackendAppSecret,
		Mode:             "env",
	}, nil
}
