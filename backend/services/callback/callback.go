/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package callback

import (
	"net/http"
	"net/url"
	"time"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// CallbackService handles HTTP callbacks for system creation
type CallbackService struct {
	httpClient *http.Client
}

// NewCallbackService creates a new callback service with configured timeouts
func NewCallbackService() *CallbackService {
	return &CallbackService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExecuteSystemCreationCallback sends system data to the callback URL via GET redirect
func (cs *CallbackService) ExecuteSystemCreationCallback(callbackURL, state string, system models.System) bool {
	// Build callback URL with query parameters (OAuth style)
	callbackURLWithParams, err := cs.buildCallbackURL(callbackURL, state, system)
	if err != nil {
		logger.Error().
			Err(err).
			Str("component", "callback_service").
			Str("callback_url", callbackURL).
			Msg("Failed to build callback URL")
		return false
	}

	// Create HTTP GET request (OAuth style redirect)
	req, err := http.NewRequest("GET", callbackURLWithParams, nil)
	if err != nil {
		logger.Error().
			Err(err).
			Str("component", "callback_service").
			Str("callback_url", callbackURLWithParams).
			Msg("Failed to create callback request")
		return false
	}

	// Set headers
	req.Header.Set("User-Agent", "My-Nethesis/1.0")

	// Execute request
	resp, err := cs.httpClient.Do(req)
	if err != nil {
		logger.Error().
			Err(err).
			Str("component", "callback_service").
			Str("callback_url", callbackURL).
			Str("system_id", system.ID).
			Msg("Callback request failed")
		return false
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.Warn().
				Err(closeErr).
				Str("component", "callback_service").
				Msg("Failed to close response body")
		}
	}()

	// Check response status
	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	if success {
		logger.Info().
			Str("component", "callback_service").
			Str("callback_url", callbackURL).
			Str("system_id", system.ID).
			Str("state", state).
			Int("status_code", resp.StatusCode).
			Msg("Callback executed successfully")
	} else {
		logger.Warn().
			Str("component", "callback_service").
			Str("callback_url", callbackURL).
			Str("system_id", system.ID).
			Str("state", state).
			Int("status_code", resp.StatusCode).
			Msg("Callback returned non-success status")
	}

	return success
}

// buildCallbackURL constructs the callback URL with OAuth-style query parameters
func (cs *CallbackService) buildCallbackURL(baseURL, state string, system models.System) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	// Add query parameters (OAuth style) - only non-sensitive data
	query := u.Query()
	query.Set("state", state)
	query.Set("system_id", system.ID)
	query.Set("system_name", system.Name)
	query.Set("system_type", system.Type)
	// system_secret intentionally omitted for security (visible only in creation confirmation)
	query.Set("timestamp", time.Now().Format(time.RFC3339))

	u.RawQuery = query.Encode()
	return u.String(), nil
}
