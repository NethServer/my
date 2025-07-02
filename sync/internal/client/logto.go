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

// GetUsers retrieves all users
func (c *LogtoClient) GetUsers() ([]map[string]interface{}, error) {
	resp, err := c.makeRequest("GET", "/api/users", nil)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	return result, c.handlePaginatedResponse(resp, &result)
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
