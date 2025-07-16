/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
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
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
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

	logger.ComponentLogger("logto").Debug().
		Str("operation", "request_token").
		Str("resource", managementAPIResource).
		Msg("Requesting Management API token")

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(payload))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
		return fmt.Errorf("failed to request token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp models.LogtoManagementTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	logger.ComponentLogger("logto").Info().
		Str("operation", "token_obtained").
		Time("expires_at", c.tokenExpiry).
		Msg("Management API token obtained")
	return nil
}

// makeRequest makes an authenticated request to the Management API with token refresh retry
func (c *LogtoManagementClient) makeRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	return c.makeRequestWithRetry(method, endpoint, body, false)
}

// makeRequestWithRetry makes an authenticated request with optional retry for token refresh
func (c *LogtoManagementClient) makeRequestWithRetry(method, endpoint string, body io.Reader, isRetry bool) (*http.Response, error) {
	start := time.Now()

	// Ensure we have a valid token
	if err := c.getAccessToken(); err != nil {
		logger.ComponentLogger("logto").Error().
			Err(err).
			Str("operation", "api_call_token_failed").
			Str("method", method).
			Str("endpoint", endpoint).
			Bool("is_retry", isRetry).
			Msg("Failed to get access token for Logto API call")
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	url := strings.TrimSuffix(c.baseURL, "/") + endpoint
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		logger.ComponentLogger("logto").Error().
			Err(err).
			Str("operation", "api_call_request_failed").
			Str("method", method).
			Str("endpoint", endpoint).
			Str("url", url).
			Bool("is_retry", isRetry).
			Msg("Failed to create HTTP request for Logto API")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	logger.ComponentLogger("logto").Debug().
		Str("operation", "api_call_start").
		Str("method", method).
		Str("endpoint", endpoint).
		Str("url", url).
		Bool("is_retry", isRetry).
		Msg("Starting Logto Management API call")

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

	duration := time.Since(start)

	if err != nil {
		logger.ComponentLogger("logto").Error().
			Err(err).
			Str("operation", "api_call_failed").
			Str("method", method).
			Str("endpoint", endpoint).
			Str("url", url).
			Bool("is_retry", isRetry).
			Dur("duration", duration).
			Msg("Logto Management API call failed")
		return nil, err
	}

	// Log successful API calls
	logger.ComponentLogger("logto").Info().
		Str("operation", "api_call_success").
		Str("method", method).
		Str("endpoint", endpoint).
		Int("status_code", resp.StatusCode).
		Bool("is_retry", isRetry).
		Dur("duration", duration).
		Msg("Logto Management API call completed")

	// Handle 401 Unauthorized - token might be expired
	if resp.StatusCode == 401 && !isRetry {
		logger.ComponentLogger("logto").Warn().
			Str("operation", "api_call_token_expired").
			Str("method", method).
			Str("endpoint", endpoint).
			Int("status_code", resp.StatusCode).
			Dur("duration", duration).
			Msg("Received 401, invalidating token and retrying")

		// Close the response body before retry
		_ = resp.Body.Close()

		// Invalidate current token to force refresh
		c.accessToken = ""
		c.tokenExpiry = time.Time{}

		// For retry, we need to recreate the request body since io.Reader can only be read once
		var retryBody io.Reader
		if body != nil {
			// For POST/PUT requests, we need to recreate the body
			// This is a limitation - for now we'll only retry GET requests
			if method != "GET" {
				logger.ComponentLogger("logto").Warn().
					Str("operation", "api_call_retry_skipped").
					Str("method", method).
					Str("endpoint", endpoint).
					Msg("Skipping retry for non-GET request with body")
				return resp, nil
			}
		}

		// Retry once with fresh token
		return c.makeRequestWithRetry(method, endpoint, retryBody, true)
	}

	// Log non-2xx status codes as warnings
	if resp.StatusCode >= 400 {
		logger.ComponentLogger("logto").Warn().
			Str("operation", "api_call_error_status").
			Str("method", method).
			Str("endpoint", endpoint).
			Int("status_code", resp.StatusCode).
			Bool("is_retry", isRetry).
			Dur("duration", duration).
			Msg("Logto Management API returned error status")
	}

	return resp, nil
}
