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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// sharedHTTPClient is a package-level HTTP client with connection pooling.
// Reused across all Logto API calls to avoid creating a new TCP connection per request.
var sharedHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
	},
}

// TokenCache holds Management API token with thread safety
type TokenCache struct {
	mu          sync.RWMutex
	accessToken string
	tokenExpiry time.Time
}

// Global token cache shared across all clients
var globalTokenCache = &TokenCache{}

// LogtoManagementClient handles Logto Management API calls
type LogtoManagementClient struct {
	baseURL      string
	clientID     string
	clientSecret string
}

// NewManagementClient creates a new Logto Management API client
func NewManagementClient() *LogtoManagementClient {
	return &LogtoManagementClient{
		baseURL:      configuration.Config.LogtoManagementBaseURL,
		clientID:     configuration.Config.LogtoManagementClientID,
		clientSecret: configuration.Config.LogtoManagementClientSecret,
	}
}

// =============================================================================
// PUBLIC METHODS
// =============================================================================

// makeRequest makes an authenticated request to the Management API with token refresh retry
func (c *LogtoManagementClient) makeRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	return c.makeRequestWithRetry(method, endpoint, body, false)
}

// =============================================================================
// PRIVATE METHODS
// =============================================================================

// getAccessToken obtains an access token for the Management API with enhanced caching
func (c *LogtoManagementClient) getAccessToken() (string, error) {
	// First, try to get token from cache (read lock)
	globalTokenCache.mu.RLock()
	if globalTokenCache.accessToken != "" && time.Now().Before(globalTokenCache.tokenExpiry.Add(-30*time.Second)) {
		// Return cached token if it's valid and has at least 30 seconds left
		token := globalTokenCache.accessToken
		globalTokenCache.mu.RUnlock()
		return token, nil
	}
	globalTokenCache.mu.RUnlock()

	// Need to refresh token - acquire write lock
	globalTokenCache.mu.Lock()
	defer globalTokenCache.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine might have refreshed)
	if globalTokenCache.accessToken != "" && time.Now().Before(globalTokenCache.tokenExpiry.Add(-30*time.Second)) {
		return globalTokenCache.accessToken, nil
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
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := sharedHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp models.LogtoManagementTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	// Update cache
	globalTokenCache.accessToken = tokenResp.AccessToken
	globalTokenCache.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	logger.ComponentLogger("logto").Info().
		Str("operation", "token_obtained").
		Time("expires_at", globalTokenCache.tokenExpiry).
		Msg("Management API token obtained")

	return globalTokenCache.accessToken, nil
}

// invalidateToken clears the cached token to force refresh
func invalidateToken() {
	globalTokenCache.mu.Lock()
	defer globalTokenCache.mu.Unlock()
	globalTokenCache.accessToken = ""
	globalTokenCache.tokenExpiry = time.Time{}
}

// makeRequestWithRetry makes an authenticated request. It transparently retries
// on 401 (refreshing the token once) and on 429 (rate limited) with capped
// exponential backoff + jitter, honoring Retry-After. The body is buffered so
// POST/PUT requests can be safely replayed across retries (a 429 means the
// request was rejected, so replaying creates no duplicates).
func (c *LogtoManagementClient) makeRequestWithRetry(method, endpoint string, body io.Reader, isRetry bool) (*http.Response, error) {
	url := strings.TrimSuffix(c.baseURL, "/") + endpoint

	// Buffer the body once so each attempt gets a fresh reader (io.Reader is single-use).
	var bodyBytes []byte
	if body != nil {
		b, readErr := io.ReadAll(body)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read request body: %w", readErr)
		}
		bodyBytes = b
	}

	const maxRateLimitRetries = 5
	tokenRefreshed := isRetry

	for attempt := 0; ; attempt++ {
		start := time.Now()

		// Ensure we have a valid token
		accessToken, err := c.getAccessToken()
		if err != nil {
			logger.ComponentLogger("logto").Error().
				Err(err).
				Str("operation", "api_call_token_failed").
				Str("method", method).
				Str("endpoint", endpoint).
				Msg("Failed to get access token for Logto API call")
			return nil, fmt.Errorf("failed to get access token: %w", err)
		}

		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequest(method, url, reqBody)
		if err != nil {
			logger.ComponentLogger("logto").Error().
				Err(err).
				Str("operation", "api_call_request_failed").
				Str("method", method).
				Str("endpoint", endpoint).
				Str("url", url).
				Msg("Failed to create HTTP request for Logto API")
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := sharedHTTPClient.Do(req)
		duration := time.Since(start)
		if err != nil {
			logger.ComponentLogger("logto").Error().
				Err(err).
				Str("operation", "api_call_failed").
				Str("method", method).
				Str("endpoint", endpoint).
				Str("url", url).
				Dur("duration", duration).
				Msg("Logto Management API call failed")
			return nil, err
		}

		// 401 Unauthorized - token might be expired; refresh once and retry.
		if resp.StatusCode == 401 && !tokenRefreshed {
			logger.ComponentLogger("logto").Warn().
				Str("operation", "api_call_token_expired").
				Str("method", method).
				Str("endpoint", endpoint).
				Msg("Received 401, invalidating token and retrying")
			_ = resp.Body.Close()
			invalidateToken()
			tokenRefreshed = true
			continue
		}

		// 429 Too Many Requests - back off and retry.
		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxRateLimitRetries {
			wait := rateLimitBackoff(resp, attempt)
			_ = resp.Body.Close()
			logger.ComponentLogger("logto").Warn().
				Str("operation", "api_call_rate_limited").
				Str("method", method).
				Str("endpoint", endpoint).
				Int("attempt", attempt+1).
				Dur("backoff", wait).
				Msg("Received 429 from Logto, backing off and retrying")
			time.Sleep(wait)
			continue
		}

		if resp.StatusCode >= 400 {
			logger.ComponentLogger("logto").Warn().
				Str("operation", "api_call_error_status").
				Str("method", method).
				Str("endpoint", endpoint).
				Int("status_code", resp.StatusCode).
				Dur("duration", duration).
				Msg("Logto Management API returned error status")
		} else {
			logger.ComponentLogger("logto").Info().
				Str("operation", "api_call_success").
				Str("method", method).
				Str("endpoint", endpoint).
				Int("status_code", resp.StatusCode).
				Dur("duration", duration).
				Msg("Logto Management API call completed")
		}

		return resp, nil
	}
}

// rateLimitBackoff computes how long to wait before retrying a 429, honoring the
// Retry-After header when present, otherwise capped exponential backoff. Jitter
// avoids concurrent workers retrying in lockstep.
func rateLimitBackoff(resp *http.Response, attempt int) time.Duration {
	base := 500 * time.Millisecond * (1 << attempt)
	if base > 8*time.Second {
		base = 8 * time.Second
	}
	if ra := strings.TrimSpace(resp.Header.Get("Retry-After")); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
			base = time.Duration(secs) * time.Second
		} else if t, err := http.ParseTime(ra); err == nil {
			if d := time.Until(t); d > 0 {
				base = d
			}
		}
	}
	return base + rand.N(500*time.Millisecond)
}
