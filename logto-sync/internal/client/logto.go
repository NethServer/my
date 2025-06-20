/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
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

	"github.com/nethesis/my/logto-sync/internal/logger"
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
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
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
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API test failed with status %d: %s", resp.StatusCode, string(body))
	}

	logger.Debug("Connection to Logto successful")
	return nil
}

// makeRequest makes an authenticated request to the Logto API
func (c *LogtoClient) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
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
		logger.Debug("%s %s with body: %s", method, endpoint, string(jsonBody))
	} else {
		logger.Debug("%s %s", method, endpoint)
	}

	req, err := http.NewRequest(method, c.BaseURL+endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	logger.Debug("Response status: %d", resp.StatusCode)
	return resp, nil
}

// handleResponse handles common response patterns
func (c *LogtoClient) handleResponse(resp *http.Response, expectedStatus int, target interface{}) error {
	defer resp.Body.Close()

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

// handlePaginatedResponse handles paginated responses
func (c *LogtoClient) handlePaginatedResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

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
