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
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// GetThirdPartyApplications retrieves all third-party applications from Logto
func (c *LogtoManagementClient) GetThirdPartyApplications() ([]models.LogtoThirdPartyApp, error) {
	err := c.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	logger.ComponentLogger("logto").Debug().
		Str("operation", "get_applications").
		Msg("Fetching third-party applications from Logto")

	// Get all applications
	req, err := http.NewRequest("GET", c.baseURL+"/applications", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives:     true, // Disable connection reuse to handle network changes
			MaxIdleConnsPerHost:   0,    // No idle connections
			IdleConnTimeout:       0,    // No idle timeout
			ResponseHeaderTimeout: 15 * time.Second,
		},
	}
	resp, err := client.Do(req)
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
	req, err := http.NewRequest("GET", c.baseURL+"/applications/"+appID+"/sign-in-experience", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives:     true, // Disable connection reuse to handle network changes
			MaxIdleConnsPerHost:   0,    // No idle connections
			IdleConnTimeout:       0,    // No idle timeout
			ResponseHeaderTimeout: 15 * time.Second,
		},
	}
	resp, err := client.Do(req)
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
	req, err := http.NewRequest("GET", c.baseURL+"/applications/"+appID+"/user-consent-scopes", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives:     true, // Disable connection reuse to handle network changes
			MaxIdleConnsPerHost:   0,    // No idle connections
			IdleConnTimeout:       0,    // No idle timeout
			ResponseHeaderTimeout: 15 * time.Second,
		},
	}
	resp, err := client.Do(req)
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
func GenerateOAuth2LoginURL(appID string, redirectURI string, scopes []string) string {
	// Get Logto issuer from configuration (e.g., "https://tree6d.logto.app")
	logtoIssuer := configuration.Config.LogtoIssuer

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
	authURL := fmt.Sprintf("%s/oidc/auth", logtoIssuer)

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

// generateRandomState generates a random state string for OAuth2
func generateRandomState() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a simple timestamp-based state
		return "random-state-string"
	}
	return hex.EncodeToString(bytes)
}
