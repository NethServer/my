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
	"os"
)

// GetAPIBaseURL returns the API base URL from environment or derives it from LOGTO_TENANT_DOMAIN
func GetAPIBaseURL() string {
	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		// Derive from LOGTO_TENANT_DOMAIN if API_BASE_URL is not set
		tenantDomain := os.Getenv("LOGTO_TENANT_DOMAIN")
		if tenantDomain != "" {
			apiBaseURL = fmt.Sprintf("https://%s/api", tenantDomain)
		} else {
			// Fallback to localhost only if neither is available
			apiBaseURL = "http://localhost:8080"
		}
	}
	return apiBaseURL
}
