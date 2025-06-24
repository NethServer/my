/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logs"
	"github.com/nethesis/my/backend/models"
)

// LogtoUserInfo represents the user info returned by Logto
type LogtoUserInfo struct {
	Sub              string   `json:"sub"`
	Username         string   `json:"username"`
	Email            string   `json:"email"`
	Name             string   `json:"name"`
	Roles            []string `json:"roles"`
	OrganizationId   string   `json:"organization_id"`
	OrganizationName string   `json:"organization_name"`
	// Add other fields as needed
}

// Management API structures
type LogtoRole struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type LogtoScope struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ResourceID  string `json:"resourceId"`
}

type LogtoOrganization struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	CustomData   map[string]interface{} `json:"customData"`
	IsMfaRequired bool                   `json:"isMfaRequired"`
	Branding     *LogtoOrganizationBranding `json:"branding"`
}

type LogtoOrganizationBranding struct {
	LogoUrl     string `json:"logoUrl"`
	DarkLogoUrl string `json:"darkLogoUrl"`
	Favicon     string `json:"favicon"`
	DarkFavicon string `json:"darkFavicon"`
}

type LogtoOrganizationRole struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type LogtoManagementTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// LogtoManagementClient handles Logto Management API calls
type LogtoManagementClient struct {
	baseURL      string
	clientID     string
	clientSecret string
	accessToken  string
	tokenExpiry  time.Time
}

// GetUserInfoFromLogto fetches user information from Logto using access token
func GetUserInfoFromLogto(accessToken string) (*LogtoUserInfo, error) {
	// Create request to Logto userinfo endpoint
	userInfoURL := configuration.Config.LogtoIssuer + "/oidc/me"
	
	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("logto userinfo request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	logs.Logs.Printf("[DEBUG][LOGTO] Userinfo response: %s", string(body))

	var userInfo LogtoUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	logs.Logs.Printf("[DEBUG][LOGTO] Parsed userinfo: sub=%s, username=%s, email=%s", userInfo.Sub, userInfo.Username, userInfo.Email)

	return &userInfo, nil
}

// NewLogtoManagementClient creates a new Logto Management API client
func NewLogtoManagementClient() *LogtoManagementClient {
	return &LogtoManagementClient{
		baseURL:      configuration.Config.LogtoManagementBaseURL,
		clientID:     configuration.Config.LogtoManagementClientID,
		clientSecret: configuration.Config.LogtoManagementClientSecret,
	}
}

// getAccessToken obtains an access token for the Management API
func (c *LogtoManagementClient) getAccessToken() error {
	// Check if we have a valid token
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return nil
	}

	// Request new token
	tokenURL := strings.TrimSuffix(configuration.Config.LogtoIssuer, "/") + "/oidc/token"
	
	// Management API resource indicator
	managementAPIResource := strings.TrimSuffix(configuration.Config.LogtoIssuer, "/") + "/api"
	
	payload := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s&resource=%s&scope=all",
		c.clientID, c.clientSecret, managementAPIResource)

	logs.Logs.Printf("[DEBUG][LOGTO] Requesting Management API token for resource: %s", managementAPIResource)

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(payload))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp LogtoManagementTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	logs.Logs.Printf("[INFO][LOGTO] Management API token obtained, expires at %v", c.tokenExpiry)
	return nil
}

// makeRequest makes an authenticated request to the Management API
func (c *LogtoManagementClient) makeRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	// Ensure we have a valid token
	if err := c.getAccessToken(); err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	url := strings.TrimSuffix(c.baseURL, "/") + endpoint
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

// GetUserRoles fetches user roles from Logto Management API
func (c *LogtoManagementClient) GetUserRoles(userID string) ([]LogtoRole, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/users/%s/roles", userID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user roles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user roles, status %d: %s", resp.StatusCode, string(body))
	}

	var roles []LogtoRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode user roles: %w", err)
	}

	return roles, nil
}

// GetRoleScopes fetches scopes for a role
func (c *LogtoManagementClient) GetRoleScopes(roleID string) ([]LogtoScope, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/roles/%s/scopes", roleID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch role scopes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch role scopes, status %d: %s", resp.StatusCode, string(body))
	}

	var scopes []LogtoScope
	if err := json.NewDecoder(resp.Body).Decode(&scopes); err != nil {
		return nil, fmt.Errorf("failed to decode role scopes: %w", err)
	}

	return scopes, nil
}

// GetUserOrganizations fetches organizations the user belongs to
func (c *LogtoManagementClient) GetUserOrganizations(userID string) ([]LogtoOrganization, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/users/%s/organizations", userID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user organizations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user organizations, status %d: %s", resp.StatusCode, string(body))
	}

	var orgs []LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return nil, fmt.Errorf("failed to decode user organizations: %w", err)
	}

	return orgs, nil
}

// GetUserOrganizationRoles fetches user's roles in an organization
func (c *LogtoManagementClient) GetUserOrganizationRoles(orgID, userID string) ([]LogtoOrganizationRole, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organizations/%s/users/%s/roles", orgID, userID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user organization roles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user organization roles, status %d: %s", resp.StatusCode, string(body))
	}

	var roles []LogtoOrganizationRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode user organization roles: %w", err)
	}

	return roles, nil
}

// GetOrganizationRoleScopes fetches scopes for an organization role
func (c *LogtoManagementClient) GetOrganizationRoleScopes(roleID string) ([]LogtoScope, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organization-roles/%s/scopes", roleID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization role scopes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organization role scopes, status %d: %s", resp.StatusCode, string(body))
	}

	var scopes []LogtoScope
	if err := json.NewDecoder(resp.Body).Decode(&scopes); err != nil {
		return nil, fmt.Errorf("failed to decode organization role scopes: %w", err)
	}

	return scopes, nil
}

// EnrichUserWithRolesAndPermissions fetches complete roles and permissions from Logto Management API
func EnrichUserWithRolesAndPermissions(userID string) (*models.User, error) {
	logs.Logs.Printf("[DEBUG][LOGTO] Starting enrichment for user: %s", userID)
	client := NewLogtoManagementClient()

	// Initialize user
	user := &models.User{
		ID:               userID,
		UserRoles:        []string{},
		UserPermissions:  []string{},
		OrgRole:          "",
		OrgPermissions:   []string{},
		OrganizationID:   "",
		OrganizationName: "",
	}

	// Fetch user roles (technical capabilities)
	logs.Logs.Printf("[DEBUG][LOGTO] Fetching user roles for: %s", userID)
	userRoles, err := client.GetUserRoles(userID)
	if err != nil {
		logs.Logs.Printf("[WARN][LOGTO] Failed to fetch user roles for %s: %v", userID, err)
	} else {
		logs.Logs.Printf("[DEBUG][LOGTO] Found %d user roles for %s", len(userRoles), userID)
		// Extract role names
		for _, role := range userRoles {
			user.UserRoles = append(user.UserRoles, role.Name)
		}

		// Fetch permissions for each user role
		for _, role := range userRoles {
			scopes, err := client.GetRoleScopes(role.ID)
			if err != nil {
				logs.Logs.Printf("[WARN][LOGTO] Failed to fetch scopes for role %s: %v", role.ID, err)
				continue
			}
			for _, scope := range scopes {
				user.UserPermissions = append(user.UserPermissions, scope.Name)
			}
		}
	}

	// Fetch user organizations
	logs.Logs.Printf("[DEBUG][LOGTO] Fetching user organizations for: %s", userID)
	orgs, err := client.GetUserOrganizations(userID)
	if err != nil {
		logs.Logs.Printf("[WARN][LOGTO] Failed to fetch user organizations for %s: %v", userID, err)
	} else {
		logs.Logs.Printf("[DEBUG][LOGTO] Found %d organizations for %s", len(orgs), userID)
		if len(orgs) > 0 {
			// Use first organization as primary
			primaryOrg := orgs[0]
			user.OrganizationID = primaryOrg.ID
			user.OrganizationName = primaryOrg.Name

			// Fetch user's roles in this organization
			orgRoles, err := client.GetUserOrganizationRoles(primaryOrg.ID, userID)
			if err != nil {
				logs.Logs.Printf("[WARN][LOGTO] Failed to fetch organization roles for %s in org %s: %v", userID, primaryOrg.ID, err)
			} else if len(orgRoles) > 0 {
				// Use first organization role as primary
				primaryOrgRole := orgRoles[0]
				user.OrgRole = primaryOrgRole.Name

				// Fetch permissions for organization role
				orgScopes, err := client.GetOrganizationRoleScopes(primaryOrgRole.ID)
				if err != nil {
					logs.Logs.Printf("[WARN][LOGTO] Failed to fetch organization role scopes for %s: %v", primaryOrgRole.ID, err)
				} else {
					for _, scope := range orgScopes {
						user.OrgPermissions = append(user.OrgPermissions, scope.Name)
					}
				}
			}
		}
	}

	// Remove duplicates from permissions
	user.UserPermissions = removeDuplicates(user.UserPermissions)
	user.OrgPermissions = removeDuplicates(user.OrgPermissions)

	logs.Logs.Printf("[INFO][LOGTO] Enriched user %s with %d user roles, %d user permissions, org role '%s', %d org permissions",
		userID, len(user.UserRoles), len(user.UserPermissions), user.OrgRole, len(user.OrgPermissions))

	return user, nil
}

// removeDuplicates removes duplicate strings from a slice
func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	return result
}

// GetAllOrganizations fetches all organizations from Logto
func (c *LogtoManagementClient) GetAllOrganizations() ([]LogtoOrganization, error) {
	resp, err := c.makeRequest("GET", "/organizations", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organizations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organizations, status %d: %s", resp.StatusCode, string(body))
	}

	var orgs []LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return nil, fmt.Errorf("failed to decode organizations: %w", err)
	}

	return orgs, nil
}

// GetOrganizationJitRoles fetches default organization roles (just-in-time provisioning)
func (c *LogtoManagementClient) GetOrganizationJitRoles(orgID string) ([]LogtoOrganizationRole, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organizations/%s/jit/roles", orgID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization JIT roles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organization JIT roles, status %d: %s", resp.StatusCode, string(body))
	}

	var roles []LogtoOrganizationRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode organization JIT roles: %w", err)
	}

	return roles, nil
}

// GetOrganizationsByRole fetches organizations that have specific default organization roles (JIT)
// This is used to filter distributors, resellers, customers based on their JIT role configuration
func GetOrganizationsByRole(roleType string) ([]LogtoOrganization, error) {
	client := NewLogtoManagementClient()
	
	// Get all organizations
	allOrgs, err := client.GetAllOrganizations()
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}

	var filteredOrgs []LogtoOrganization
	
	// For each organization, check if it has the specified default role (JIT)
	for _, org := range allOrgs {
		jitRoles, err := client.GetOrganizationJitRoles(org.ID)
		if err != nil {
			logs.Logs.Printf("[WARN][LOGTO] Failed to get JIT roles for org %s: %v", org.ID, err)
			continue
		}
		
		// Check if this organization has the target role as default
		hasRole := false
		for _, role := range jitRoles {
			logs.Logs.Printf("[DEBUG][LOGTO] Org %s (%s) has JIT role: %s", org.ID, org.Name, role.Name)
			if role.Name == roleType {
				hasRole = true
				break
			}
		}
		
		if hasRole {
			logs.Logs.Printf("[INFO][LOGTO] Org %s (%s) matches role %s", org.ID, org.Name, roleType)
			filteredOrgs = append(filteredOrgs, org)
		}
	}
	
	logs.Logs.Printf("[INFO][LOGTO] Found %d organizations with JIT role '%s'", len(filteredOrgs), roleType)
	return filteredOrgs, nil
}

// FilterOrganizationsByVisibility filters organizations based on user's visibility permissions
func FilterOrganizationsByVisibility(orgs []LogtoOrganization, userOrgRole, userOrgID string, targetRole string) []LogtoOrganization {
	// God can see everything
	if userOrgRole == "God" {
		logs.Logs.Printf("[INFO][LOGTO] God user - showing all %d %ss", len(orgs), targetRole)
		return orgs
	}

	var filteredOrgs []LogtoOrganization

	switch targetRole {
	case "Distributor":
		// Only God should access distributors (already protected by middleware)
		logs.Logs.Printf("[INFO][LOGTO] Non-God user accessing distributors - should be blocked by middleware")
		return filteredOrgs

	case "Reseller":
		// Distributors see only resellers they created
		if userOrgRole == "Distributor" {
			for _, org := range orgs {
				if org.CustomData != nil {
					if createdBy, ok := org.CustomData["createdBy"].(string); ok && createdBy == userOrgID {
						filteredOrgs = append(filteredOrgs, org)
					}
				}
			}
			logs.Logs.Printf("[INFO][LOGTO] Distributor %s can see %d/%d resellers", userOrgID, len(filteredOrgs), len(orgs))
		}

	case "Customer":
		if userOrgRole == "Distributor" {
			// Distributors see customers created by their resellers
			// First, get all resellers created by this distributor
			distributorResellers, err := GetOrganizationsByRole("Reseller")
			if err != nil {
				logs.Logs.Printf("[ERROR][LOGTO] Failed to get distributor's resellers: %v", err)
				return filteredOrgs
			}

			// Get IDs of resellers created by this distributor
			var resellerIDs []string
			for _, reseller := range distributorResellers {
				if reseller.CustomData != nil {
					if createdBy, ok := reseller.CustomData["createdBy"].(string); ok && createdBy == userOrgID {
						resellerIDs = append(resellerIDs, reseller.ID)
					}
				}
			}

			// Filter customers created by these resellers
			for _, org := range orgs {
				if org.CustomData != nil {
					if createdBy, ok := org.CustomData["createdBy"].(string); ok {
						for _, resellerID := range resellerIDs {
							if createdBy == resellerID {
								filteredOrgs = append(filteredOrgs, org)
								break
							}
						}
					}
				}
			}
			logs.Logs.Printf("[INFO][LOGTO] Distributor %s can see %d/%d customers (via %d resellers)", userOrgID, len(filteredOrgs), len(orgs), len(resellerIDs))

		} else if userOrgRole == "Reseller" {
			// Resellers see only customers they created
			for _, org := range orgs {
				if org.CustomData != nil {
					if createdBy, ok := org.CustomData["createdBy"].(string); ok && createdBy == userOrgID {
						filteredOrgs = append(filteredOrgs, org)
					}
				}
			}
			logs.Logs.Printf("[INFO][LOGTO] Reseller %s can see %d/%d customers", userOrgID, len(filteredOrgs), len(orgs))
		}
	}

	return filteredOrgs
}