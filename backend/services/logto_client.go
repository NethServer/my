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
)

// LogtoManagementClient handles Logto Management API calls
type LogtoManagementClient struct {
	baseURL      string
	clientID     string
	clientSecret string
	accessToken  string
	tokenExpiry  time.Time
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
