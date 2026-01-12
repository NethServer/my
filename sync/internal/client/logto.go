/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nethesis/my/sync/internal/constants"
	"github.com/nethesis/my/sync/internal/logger"
)

// LogtoClient handles communication with Logto API
type LogtoClient struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	accessToken  string
	tokenExpiry  time.Time
	HTTPClient   *http.Client
}

// NewLogtoClient creates a new Logto client
func NewLogtoClient(baseURL, clientID, clientSecret string) *LogtoClient {
	return &LogtoClient{
		BaseURL:      baseURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		HTTPClient:   &http.Client{Timeout: constants.DefaultHTTPTimeout * time.Second},
	}
}

// TestConnection tests the connection to Logto
func (c *LogtoClient) TestConnection() error {
	logger.Debug("Testing connection to Logto at %s", c.BaseURL)

	if err := c.getAccessToken(); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	// Try to make a simple API call
	resp, err := c.makeRequest("GET", "/api/resources", nil)
	if err != nil {
		return fmt.Errorf("failed to make test request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API test failed with status %d: %s", resp.StatusCode, string(body))
	}

	logger.Debug("Connection to Logto successful")
	return nil
}

// makeRequest makes an authenticated request to the Logto API
func (c *LogtoClient) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	start := time.Now()

	// Ensure we have a valid access token
	if err := c.getAccessToken(); err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
		apiLogger := logger.ComponentLogger("api-client")
		apiLogger.Debug().
			Str("method", method).
			Str("endpoint", endpoint).
			Str("body", logger.SanitizeMessage(string(jsonBody))).
			Msg("Making API request with body")
	} else {
		apiLogger := logger.ComponentLogger("api-client")
		apiLogger.Debug().
			Str("method", method).
			Str("endpoint", endpoint).
			Msg("Making API request")
	}

	req, err := http.NewRequest(method, c.BaseURL+endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	duration := time.Since(start)

	if err != nil {
		apiLogger := logger.ComponentLogger("api-client")
		apiLogger.Error().
			Str("method", method).
			Str("endpoint", endpoint).
			Dur("duration", duration).
			Err(err).
			Msg("API request failed")
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Log the API call with structured logging
	logger.LogAPICall(method, endpoint, resp.StatusCode, duration)

	return resp, nil
}

// handleResponse handles common response patterns
func (c *LogtoClient) handleResponse(resp *http.Response, expectedStatus int, target interface{}) error {
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	if target != nil && len(body) > 0 {
		if err := json.Unmarshal(body, target); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// handleCreationResponse handles creation responses that can return either 200 or 201
func (c *LogtoClient) handleCreationResponse(resp *http.Response, target interface{}) error {
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	if target != nil && len(body) > 0 {
		if err := json.Unmarshal(body, target); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// createEntitySimple creates an entity using a simple map structure
func (c *LogtoClient) createEntitySimple(endpoint string, entityData map[string]interface{}, entityType string) error {
	logger.Debug("Creating %s: %v", entityType, entityData["name"])

	resp, err := c.makeRequest("POST", endpoint, entityData)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", entityType, err)
	}

	return c.handleCreationResponse(resp, nil)
}

// FindEntityByField finds an entity in a slice by matching a specific field value
func (c *LogtoClient) FindEntityByField(entities []map[string]interface{}, fieldName string, fieldValue interface{}) (map[string]interface{}, bool) {
	for _, entity := range entities {
		if value, ok := entity[fieldName]; ok && value == fieldValue {
			return entity, true
		}
	}
	return nil, false
}

// FindEntityID finds an entity ID by matching a specific field value
func (c *LogtoClient) FindEntityID(entities []map[string]interface{}, fieldName string, fieldValue interface{}) (string, bool) {
	if entity, found := c.FindEntityByField(entities, fieldName, fieldValue); found {
		if id, ok := entity["id"].(string); ok {
			return id, true
		}
	}
	return "", false
}

// handlePaginatedResponse handles paginated responses
func (c *LogtoClient) handlePaginatedResponse(resp *http.Response, target interface{}) error {
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	// Try to parse as paginated response first
	var paginatedResp struct {
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(body, &paginatedResp); err == nil && paginatedResp.Data != nil {
		return json.Unmarshal(paginatedResp.Data, target)
	}

	// Try to parse as direct array
	return json.Unmarshal(body, target)
}

// GetApplications retrieves all applications
func (c *LogtoClient) GetApplications() ([]map[string]interface{}, error) {
	resp, err := c.makeRequest("GET", "/api/applications", nil)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	return result, c.handlePaginatedResponse(resp, &result)
}

// CreateApplication creates a new application
func (c *LogtoClient) CreateApplication(app map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.makeRequest("POST", "/api/applications", app)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	// Some Logto instances return 200 instead of 201 for creation
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return result, c.handleResponse(resp, resp.StatusCode, &result)
	}

	return result, c.handleResponse(resp, http.StatusCreated, &result)
}

// GetUsers retrieves all users (first page only)
func (c *LogtoClient) GetUsers() ([]map[string]interface{}, error) {
	resp, err := c.makeRequest("GET", "/api/users", nil)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	return result, c.handlePaginatedResponse(resp, &result)
}

// GetAllUsers retrieves all users across all pages
func (c *LogtoClient) GetAllUsers() ([]map[string]interface{}, error) {
	allUsers := []map[string]interface{}{}
	page := 1
	pageSize := 100

	for {
		resp, err := c.makeRequest("GET", fmt.Sprintf("/api/users?page=%d&page_size=%d", page, pageSize), nil)
		if err != nil {
			return nil, err
		}

		var users []map[string]interface{}
		if err := c.handlePaginatedResponse(resp, &users); err != nil {
			return nil, err
		}

		if len(users) == 0 {
			break
		}

		allUsers = append(allUsers, users...)

		// If we got fewer results than page size, we're done
		if len(users) < pageSize {
			break
		}

		page++
	}

	return allUsers, nil
}

// GetUserByUsername searches for a user by username using the search parameter
func (c *LogtoClient) GetUserByUsername(username string) (map[string]interface{}, error) {
	// Use search parameter with % wildcards as per Logto API
	resp, err := c.makeRequest("GET", fmt.Sprintf("/api/users?search=%%%s%%", username), nil)
	if err != nil {
		return nil, err
	}

	var users []map[string]interface{}
	if err := c.handlePaginatedResponse(resp, &users); err != nil {
		return nil, err
	}

	// Find exact username match
	for _, user := range users {
		if userUsername, ok := user["username"].(string); ok && userUsername == username {
			return user, nil
		}
	}

	return nil, fmt.Errorf("user with username '%s' not found", username)
}

// DeleteUser deletes a user by ID
func (c *LogtoClient) DeleteUser(userID string) error {
	resp, err := c.makeRequest("DELETE", fmt.Sprintf("/api/users/%s", userID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return c.handleResponse(resp, http.StatusNoContent, nil)
}

// GetUserRoles retrieves the roles for a specific user
func (c *LogtoClient) GetUserRoles(userID string) ([]string, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/api/users/%s/roles", userID), nil)
	if err != nil {
		return nil, err
	}

	var roles []LogtoRole
	if err := c.handlePaginatedResponse(resp, &roles); err != nil {
		return nil, err
	}

	// Extract role IDs
	roleIDs := make([]string, len(roles))
	for i, role := range roles {
		roleIDs[i] = role.ID
	}

	return roleIDs, nil
}

// UserOrganization represents a user's organization membership from Logto
type UserOrganization struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	CustomData  map[string]interface{} `json:"customData"`
}

// GetUserOrganizations retrieves the organizations a user belongs to
func (c *LogtoClient) GetUserOrganizations(userID string) ([]UserOrganization, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/api/users/%s/organizations", userID), nil)
	if err != nil {
		return nil, err
	}

	var orgs []UserOrganization
	if err := c.handlePaginatedResponse(resp, &orgs); err != nil {
		return nil, err
	}

	return orgs, nil
}

// CreateUser creates a new user
func (c *LogtoClient) CreateUser(user map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.makeRequest("POST", "/api/users", user)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	// Some Logto instances return 200 instead of 201 for creation
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return result, c.handleResponse(resp, resp.StatusCode, &result)
	}

	return result, c.handleResponse(resp, http.StatusCreated, &result)
}

// SetUserPassword sets a user's password
func (c *LogtoClient) SetUserPassword(userID, password string) error {
	data := map[string]interface{}{
		"password": password,
	}

	resp, err := c.makeRequest("PATCH", "/api/users/"+userID+"/password", data)
	if err != nil {
		return err
	}

	return c.handleResponse(resp, http.StatusOK, nil)
}

// GetDomains retrieves all custom domains
func (c *LogtoClient) GetDomains() ([]map[string]interface{}, error) {
	resp, err := c.makeRequest("GET", "/api/domains", nil)
	if err != nil {
		return nil, err
	}

	var domains []map[string]interface{}
	return domains, c.handlePaginatedResponse(resp, &domains)
}

// CreateDomain creates a new custom domain
func (c *LogtoClient) CreateDomain(domain map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.makeRequest("POST", "/api/domains", domain)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	// Some Logto instances return 200 instead of 201 for creation
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return result, c.handleResponse(resp, resp.StatusCode, &result)
	}

	return result, c.handleResponse(resp, http.StatusCreated, &result)
}

// ThirdPartyApplication represents a third-party application
type ThirdPartyApplication struct {
	ID                   string                 `json:"id,omitempty"`
	Name                 string                 `json:"name"`
	Type                 string                 `json:"type"`
	Description          string                 `json:"description"`
	IsThirdParty         bool                   `json:"isThirdParty"`
	OidcClientMetadata   *OidcClientMetadata    `json:"oidcClientMetadata,omitempty"`
	CustomClientMetadata *CustomClientMetadata  `json:"customClientMetadata,omitempty"`
	CustomData           map[string]interface{} `json:"customData,omitempty"`
}

// OidcClientMetadata represents OIDC client metadata
type OidcClientMetadata struct {
	RedirectUris           []string `json:"redirectUris,omitempty"`
	PostLogoutRedirectUris []string `json:"postLogoutRedirectUris,omitempty"`
}

// CustomClientMetadata represents custom client metadata
type CustomClientMetadata struct {
	CorsAllowedOrigins []string `json:"corsAllowedOrigins,omitempty"`
}

// ApplicationSignInExperience represents application sign-in experience settings
type ApplicationSignInExperience struct {
	DisplayName string `json:"displayName"`
}

// GetThirdPartyApplications retrieves only third-party applications (isThirdParty: true)
func (c *LogtoClient) GetThirdPartyApplications() ([]ThirdPartyApplication, error) {
	resp, err := c.makeRequest("GET", "/api/applications", nil)
	if err != nil {
		return nil, err
	}

	var allApps []ThirdPartyApplication
	if err := c.handlePaginatedResponse(resp, &allApps); err != nil {
		return nil, err
	}

	// Filter only third-party applications
	var thirdPartyApps []ThirdPartyApplication
	for _, app := range allApps {
		if app.IsThirdParty {
			thirdPartyApps = append(thirdPartyApps, app)
		}
	}

	logger.Debug("Found %d third-party applications out of %d total applications", len(thirdPartyApps), len(allApps))
	return thirdPartyApps, nil
}

// CreateThirdPartyApplication creates a new third-party application
func (c *LogtoClient) CreateThirdPartyApplication(app ThirdPartyApplication) (ThirdPartyApplication, error) {
	resp, err := c.makeRequest("POST", "/api/applications", app)
	if err != nil {
		return ThirdPartyApplication{}, err
	}

	var result ThirdPartyApplication
	// Some Logto instances return 200 instead of 201 for creation
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return result, c.handleResponse(resp, resp.StatusCode, &result)
	}

	return result, c.handleResponse(resp, http.StatusCreated, &result)
}

// UpdateThirdPartyApplication updates an existing third-party application
func (c *LogtoClient) UpdateThirdPartyApplication(appID string, app ThirdPartyApplication) error {
	resp, err := c.makeRequest("PATCH", "/api/applications/"+appID, app)
	if err != nil {
		return err
	}

	return c.handleResponse(resp, http.StatusOK, nil)
}

// UpdateThirdPartyApplicationScopes updates third-party application scopes
func (c *LogtoClient) UpdateThirdPartyApplicationScopes(appID string, scopes []string) error {
	if len(scopes) == 0 {
		logger.Debug("No scopes provided for application %s, skipping", appID)
		return nil
	}

	payload := map[string]interface{}{
		"userScopes": scopes,
	}

	resp, err := c.makeRequest("POST", "/api/applications/"+appID+"/user-consent-scopes", payload)
	if err != nil {
		return err
	}

	// Check for various success codes and handle errors gracefully
	if resp.StatusCode == 404 || resp.StatusCode == 405 || resp.StatusCode == 422 {
		_ = resp.Body.Close()
		logger.Debug("User consent scopes not supported for application %s (status: %d), skipping", appID, resp.StatusCode)
		return nil
	}

	return c.handleResponse(resp, http.StatusCreated, nil)
}

// UpdateThirdPartyApplicationBranding updates third-party application branding
func (c *LogtoClient) UpdateThirdPartyApplicationBranding(appID, displayName string) error {
	payload := ApplicationSignInExperience{
		DisplayName: displayName,
	}

	resp, err := c.makeRequest("PUT", "/api/applications/"+appID+"/sign-in-experience", payload)
	if err != nil {
		return err
	}

	if resp.StatusCode == 405 {
		_ = resp.Body.Close()
		logger.Debug("Branding not supported for application %s, skipping", appID)
		return nil
	}

	return c.handleCreationResponse(resp, nil)
}

// DeleteThirdPartyApplication deletes a third-party application
func (c *LogtoClient) DeleteThirdPartyApplication(appID string) error {
	resp, err := c.makeRequest("DELETE", "/api/applications/"+appID, nil)
	if err != nil {
		return err
	}

	return c.handleResponse(resp, http.StatusNoContent, nil)
}

// SignInExperienceMFA represents MFA configuration for sign-in experience
type SignInExperienceMFA struct {
	Policy  string   `json:"policy"`
	Factors []string `json:"factors"`
}

// SignInExperienceColor represents color configuration
type SignInExperienceColor struct {
	PrimaryColor      string `json:"primaryColor"`
	IsDarkModeEnabled bool   `json:"isDarkModeEnabled"`
	DarkPrimaryColor  string `json:"darkPrimaryColor"`
}

// SignInExperienceBranding represents branding configuration
type SignInExperienceBranding struct {
	LogoURL     string `json:"logoUrl"`
	DarkLogoURL string `json:"darkLogoUrl"`
	Favicon     string `json:"favicon"`
	DarkFavicon string `json:"darkFavicon"`
}

// SignInExperienceLanguageInfo represents language configuration
type SignInExperienceLanguageInfo struct {
	AutoDetect       bool   `json:"autoDetect"`
	FallbackLanguage string `json:"fallbackLanguage"`
}

// SignInExperienceSignInMethod represents sign-in method configuration
type SignInExperienceSignInMethod struct {
	Identifier        string `json:"identifier"`
	Password          bool   `json:"password"`
	VerificationCode  bool   `json:"verificationCode"`
	IsPasswordPrimary bool   `json:"isPasswordPrimary"`
}

// SignInExperienceSignIn represents sign-in configuration
type SignInExperienceSignIn struct {
	Methods []SignInExperienceSignInMethod `json:"methods"`
}

// SignInExperienceSignUp represents sign-up configuration
type SignInExperienceSignUp struct {
	Identifiers          []string `json:"identifiers"`
	Password             bool     `json:"password"`
	Verify               bool     `json:"verify"`
	SecondaryIdentifiers []string `json:"secondaryIdentifiers"`
}

// SignInExperienceConfig represents the complete sign-in experience configuration
type SignInExperienceConfig struct {
	Color        *SignInExperienceColor        `json:"color,omitempty"`
	Branding     *SignInExperienceBranding     `json:"branding,omitempty"`
	CustomCSS    string                        `json:"customCss,omitempty"`
	LanguageInfo *SignInExperienceLanguageInfo `json:"languageInfo,omitempty"`
	SignIn       *SignInExperienceSignIn       `json:"signIn,omitempty"`
	SignUp       *SignInExperienceSignUp       `json:"signUp,omitempty"`
	SocialSignIn map[string]interface{}        `json:"socialSignIn,omitempty"`
}

// UpdateSignInExperienceMFA configures MFA settings using the sign-in experience API
func (c *LogtoClient) UpdateSignInExperienceMFA(policy string, factors []string) error {
	mfaConfig := map[string]interface{}{
		"mfa": SignInExperienceMFA{
			Policy:  policy,
			Factors: factors,
		},
	}

	logger.Debug("Configuring MFA with policy: %s, factors: %v", policy, factors)

	resp, err := c.makeRequest("PATCH", "/api/sign-in-exp", mfaConfig)
	if err != nil {
		return fmt.Errorf("failed to update MFA configuration: %w", err)
	}

	return c.handleResponse(resp, http.StatusOK, nil)
}

// UpdateSignInExperience updates the complete sign-in experience configuration
func (c *LogtoClient) UpdateSignInExperience(config SignInExperienceConfig) error {
	logger.Debug("Updating sign-in experience configuration")

	resp, err := c.makeRequest("PATCH", "/api/sign-in-exp", config)
	if err != nil {
		return fmt.Errorf("failed to update sign-in experience: %w", err)
	}

	return c.handleResponse(resp, http.StatusOK, nil)
}
