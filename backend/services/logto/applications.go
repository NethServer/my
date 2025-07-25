/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package logto

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// GetApplications retrieves all applications from Logto
func (c *LogtoManagementClient) GetApplications() ([]models.LogtoThirdPartyApp, error) {
	logger.ComponentLogger("logto").Debug().
		Str("operation", "get_all_applications").
		Msg("Fetching all applications from Logto")

	// Use makeRequest which handles token refresh automatically
	resp, err := c.makeRequest("GET", "/applications", nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get applications: status %d", resp.StatusCode)
	}

	var logtoApps []models.LogtoThirdPartyApp
	if err := json.NewDecoder(resp.Body).Decode(&logtoApps); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.ComponentLogger("logto").Debug().
		Int("total_apps", len(logtoApps)).
		Msg("Fetched all applications from Logto")

	return logtoApps, nil
}

// GetThirdPartyApplications retrieves all third-party applications from Logto
func (c *LogtoManagementClient) GetThirdPartyApplications() ([]models.LogtoThirdPartyApp, error) {
	logger.ComponentLogger("logto").Debug().
		Str("operation", "get_third_party_applications").
		Msg("Fetching third-party applications from Logto")

	// Get all applications first
	logtoApps, err := c.GetApplications()
	if err != nil {
		return nil, fmt.Errorf("failed to get applications: %w", err)
	}

	// Filter only third-party applications
	var logtoThirdPartyApps []models.LogtoThirdPartyApp
	for _, app := range logtoApps {
		if app.IsThirdParty {
			logtoThirdPartyApps = append(logtoThirdPartyApps, app)
		}
	}

	logger.ComponentLogger("logto").Debug().
		Int("third_party_apps", len(logtoThirdPartyApps)).
		Msg("Found third-party applications")
	return logtoThirdPartyApps, nil
}

// GetApplicationBranding retrieves branding information for an application
func (c *LogtoManagementClient) GetApplicationBranding(appID string) (*models.ApplicationSignInExperience, error) {
	// Use makeRequest which handles token refresh automatically
	resp, err := c.makeRequest("GET", "/applications/"+appID+"/sign-in-experience", nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		// Branding not available for this app type
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get branding: status %d", resp.StatusCode)
	}

	var branding models.ApplicationSignInExperience
	if err := json.NewDecoder(resp.Body).Decode(&branding); err != nil {
		return nil, fmt.Errorf("failed to decode branding response: %w", err)
	}

	return &branding, nil
}

// GetApplicationScopes retrieves scopes for an application
func (c *LogtoManagementClient) GetApplicationScopes(appID string) ([]string, error) {
	// Use makeRequest which handles token refresh automatically
	resp, err := c.makeRequest("GET", "/applications/"+appID+"/user-consent-scopes", nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		// Scopes not available for this app type
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get scopes: status %d", resp.StatusCode)
	}

	var scopeResponse struct {
		UserScopes []string `json:"userScopes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&scopeResponse); err != nil {
		return nil, fmt.Errorf("failed to decode scopes response: %w", err)
	}

	return scopeResponse.UserScopes, nil
}

// FilterApplicationsByAccess filters applications based on user's organization and user roles
func FilterApplicationsByAccess(logtoApps []models.LogtoThirdPartyApp, organizationRoles []string, userRoles []string) []models.LogtoThirdPartyApp {
	var filteredApps []models.LogtoThirdPartyApp

	for _, app := range logtoApps {
		if canAccessApplication(app, organizationRoles, userRoles) {
			filteredApps = append(filteredApps, app)
		}
	}

	logger.ComponentLogger("access_control").Debug().
		Int("total", len(logtoApps)).
		Int("filtered", len(filteredApps)).
		Msg("Filtered applications based on user access")
	return filteredApps
}

// canAccessApplication checks if a user with given roles can access an application
func canAccessApplication(app models.LogtoThirdPartyApp, organizationRoles []string, userRoles []string) bool {
	// Extract access control from custom data
	accessControl := app.ExtractAccessControlFromCustomData()

	// If no access control is defined, deny access by default
	if accessControl == nil {
		return false
	}

	// Check organization roles
	if len(accessControl.OrganizationRoles) > 0 {
		hasOrgRole := false
		for _, userOrgRole := range organizationRoles {
			for _, requiredOrgRole := range accessControl.OrganizationRoles {
				if strings.EqualFold(userOrgRole, requiredOrgRole) {
					hasOrgRole = true
					break
				}
			}
			if hasOrgRole {
				break
			}
		}
		if !hasOrgRole {
			return false
		}
	}

	// Check user roles
	if len(accessControl.UserRoles) > 0 {
		hasUserRole := false
		for _, userRole := range userRoles {
			for _, requiredUserRole := range accessControl.UserRoles {
				if strings.EqualFold(userRole, requiredUserRole) {
					hasUserRole = true
					break
				}
			}
			if hasUserRole {
				break
			}
		}
		if !hasUserRole {
			return false
		}
	}

	return true
}

// GenerateOAuth2LoginURL generates the OAuth2 login URL for a third-party application
func GenerateOAuth2LoginURL(appID string, redirectURI string, scopes []string, isValidDomain bool) string {
	return GenerateOAuth2LoginURLWithDomainValidation(appID, redirectURI, scopes, isValidDomain)
}

// GenerateOAuth2LoginURLWithDomainValidation generates the OAuth2 login URL with pre-validated domain status
func GenerateOAuth2LoginURLWithDomainValidation(appID string, redirectURI string, scopes []string, isValidDomain bool) string {
	// Get domain configuration
	tenantDomain := configuration.Config.TenantDomain
	tenantID := configuration.Config.TenantID

	// Use pre-validated domain status
	var issuerHost string
	if isValidDomain {
		// Domain is valid, use custom domain
		issuerHost = fmt.Sprintf("https://%s", tenantDomain)
		logger.ComponentLogger("oauth").Debug().
			Str("domain", tenantDomain).
			Msg("Using custom domain in login URL")
	} else {
		// Domain is not valid, use tenant ID
		issuerHost = fmt.Sprintf("https://%s.logto.app", tenantID)
		logger.ComponentLogger("oauth").Debug().
			Str("tenant_id", tenantID).
			Msg("Using tenant ID in login URL")
	}

	// Use provided scopes, or fallback to basic scopes if none provided
	if len(scopes) == 0 {
		scopes = []string{
			"openid",
			"profile",
			"email",
		}
	}

	// Generate a random state string
	state := generateRandomState()

	// Build OAuth2 authorization URL
	authURL := fmt.Sprintf("%s/oidc/auth", issuerHost)

	// Create URL with query parameters
	u, err := url.Parse(authURL)
	if err != nil {
		logger.ComponentLogger("oauth").Error().
			Err(err).
			Msg("Failed to parse auth URL")
		return ""
	}

	q := u.Query()
	q.Set("client_id", appID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(scopes, " "))
	q.Set("state", state)

	u.RawQuery = q.Encode()

	return u.String()
}

// ValidateDomain checks if a domain is valid using Logto's domains API
func (c *LogtoManagementClient) ValidateDomain(domain string) bool {
	logger.ComponentLogger("logto").Info().
		Str("domain", domain).
		Msg("Validating domain with Logto")

	// Use makeRequest which handles token refresh automatically
	resp, err := c.makeRequest("GET", "/domains", nil)
	if err != nil {
		logger.ComponentLogger("logto").Error().
			Err(err).
			Str("domain", domain).
			Msg("Failed to fetch domains from Logto")
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		logger.ComponentLogger("logto").Error().
			Int("status", resp.StatusCode).
			Str("domain", domain).
			Msg("Failed to get domains from Logto")
		return false
	}

	// First, let's read the raw response to see what we get
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ComponentLogger("logto").Error().
			Err(err).
			Str("domain", domain).
			Msg("Failed to read domains response body")
		return false
	}

	logger.ComponentLogger("logto").Info().
		Str("domain", domain).
		Str("response", string(body)).
		Msg("Raw domains API response")

	var domains []struct {
		ID     string `json:"id"`
		Domain string `json:"domain"`
		Status string `json:"status"`
	}

	if err := json.Unmarshal(body, &domains); err != nil {
		logger.ComponentLogger("logto").Error().
			Err(err).
			Str("domain", domain).
			Str("response", string(body)).
			Msg("Failed to decode domains response")
		return false
	}

	logger.ComponentLogger("logto").Info().
		Str("domain", domain).
		Int("domains_count", len(domains)).
		Msg("Successfully parsed domains response")

	// Check if the domain exists and is active (status should be "Active" according to API docs)
	for _, d := range domains {
		logger.ComponentLogger("logto").Info().
			Str("checking_domain", d.Domain).
			Str("status", d.Status).
			Str("target_domain", domain).
			Msg("Checking domain match")

		if d.Domain == domain && d.Status == "Active" {
			logger.ComponentLogger("logto").Info().
				Str("domain", domain).
				Msg("Domain is valid and active")
			return true
		}
	}

	logger.ComponentLogger("logto").Info().
		Str("domain", domain).
		Msg("Domain is not valid or not active")
	return false
}

// generateRandomState generates a random state string for OAuth2
func generateRandomState() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a simple timestamp-based state
		return "random-state-string"
	}
	return hex.EncodeToString(bytes)
}
