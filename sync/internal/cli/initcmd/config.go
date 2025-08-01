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
	AppDomain        string
	BackendAppID     string
	BackendAppSecret string
	Mode             string // "env" or "cli"
}

// ValidateAndGetConfig validates and returns the configuration for the init command
func ValidateAndGetConfig(tenantID, backendAppID, backendAppSecret, logtoDomain, appDomain string) (*InitConfig, error) {
	// Check if any CLI flags are provided
	hasCLIFlags := tenantID != "" || backendAppID != "" || backendAppSecret != "" || logtoDomain != "" || appDomain != ""

	if hasCLIFlags {
		// CLI mode - all flags must be provided
		if tenantID == "" || backendAppID == "" || backendAppSecret == "" || logtoDomain == "" || appDomain == "" {
			return nil, fmt.Errorf("when using CLI flags, all must be provided:\n" +
				"  --tenant-id, --backend-app-id, --backend-app-secret, --logto-domain, --app-domain\n" +
				"Or use environment variables: TENANT_ID, BACKEND_APP_ID, BACKEND_APP_SECRET, TENANT_DOMAIN, APP_DOMAIN")
		}

		logger.Info("Using CLI mode")
		return &InitConfig{
			TenantID:         tenantID,
			TenantDomain:     logtoDomain,
			AppDomain:        appDomain,
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
	envAppDomain := os.Getenv("APP_DOMAIN")

	if envTenantID == "" || envBackendAppID == "" || envBackendAppSecret == "" || envTenantDomain == "" || envAppDomain == "" {
		return nil, fmt.Errorf("required environment variables missing:\n" +
			"  TENANT_ID, BACKEND_APP_ID, BACKEND_APP_SECRET, TENANT_DOMAIN, APP_DOMAIN\n" +
			"Or use CLI flags: --tenant-id, --backend-app-id, --backend-app-secret, --logto-domain, --app-domain")
	}

	return &InitConfig{
		TenantID:         envTenantID,
		TenantDomain:     envTenantDomain,
		AppDomain:        envAppDomain,
		BackendAppID:     envBackendAppID,
		BackendAppSecret: envBackendAppSecret,
		Mode:             "env",
	}, nil
}
