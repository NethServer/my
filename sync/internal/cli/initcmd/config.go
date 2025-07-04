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
	TenantID            string
	TenantDomain        string
	BackendClientID     string
	BackendClientSecret string
	Mode                string // "env" or "cli"
}

// ValidateAndGetConfig validates and returns the configuration for the init command
func ValidateAndGetConfig(tenantID, backendClientID, backendClientSecret, domain string) (*InitConfig, error) {
	// Check if any CLI flags are provided
	hasCLIFlags := tenantID != "" || backendClientID != "" || backendClientSecret != "" || domain != ""

	if hasCLIFlags {
		// CLI mode - all flags must be provided
		if tenantID == "" || backendClientID == "" || backendClientSecret == "" || domain == "" {
			return nil, fmt.Errorf("when using CLI flags, all must be provided:\n" +
				"  --tenant-id, --backend-client-id, --backend-client-secret, --domain\n" +
				"Or use environment variables: TENANT_ID, BACKEND_CLIENT_ID, BACKEND_CLIENT_SECRET, TENANT_DOMAIN")
		}

		logger.Info("Using CLI mode")
		return &InitConfig{
			TenantID:            tenantID,
			TenantDomain:        domain,
			BackendClientID:     backendClientID,
			BackendClientSecret: backendClientSecret,
			Mode:                "cli",
		}, nil
	}

	// Environment mode - check all required env vars
	envTenantID := os.Getenv("TENANT_ID")
	envBackendClientID := os.Getenv("BACKEND_CLIENT_ID")
	envBackendClientSecret := os.Getenv("BACKEND_CLIENT_SECRET")
	envTenantDomain := os.Getenv("TENANT_DOMAIN")

	if envTenantID == "" || envBackendClientID == "" || envBackendClientSecret == "" || envTenantDomain == "" {
		return nil, fmt.Errorf("required environment variables missing:\n" +
			"  TENANT_ID, BACKEND_CLIENT_ID, BACKEND_CLIENT_SECRET, TENANT_DOMAIN\n" +
			"Or use CLI flags: --tenant-id, --backend-client-id, --backend-client-secret, --domain")
	}

	logger.Info("Using environment variables mode")
	return &InitConfig{
		TenantID:            envTenantID,
		TenantDomain:        envTenantDomain,
		BackendClientID:     envBackendClientID,
		BackendClientSecret: envBackendClientSecret,
		Mode:                "env",
	}, nil
}
