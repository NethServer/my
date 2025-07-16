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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// GetUserInfoFromLogto fetches user information from Logto using access token
func GetUserInfoFromLogto(accessToken string) (*models.LogtoUserInfo, error) {
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
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

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

	logger.ComponentLogger("logto").Debug().
		Str("operation", "userinfo_response").
		Str("response", logger.SanitizeString(string(body))).
		Msg("Logto userinfo response")

	var userInfo models.LogtoUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	logger.ComponentLogger("logto").Debug().
		Str("operation", "userinfo_parsed").
		Str("sub", userInfo.Sub).
		Str("username", userInfo.Username).
		Str("email", logger.SanitizeString(userInfo.Email)).
		Msg("Parsed Logto userinfo")

	return &userInfo, nil
}

// GetUserProfileFromLogto fetches complete user profile from Logto Management API
func GetUserProfileFromLogto(userID string) (*models.LogtoUser, error) {
	client := NewLogtoManagementClient()

	// Use the GetUserByID method we already have
	user, err := client.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	logger.ComponentLogger("logto").Debug().
		Str("operation", "profile_response").
		Str("username", user.Username).
		Str("email", logger.SanitizeString(user.PrimaryEmail)).
		Str("name", user.Name).
		Msg("Logto profile API response")

	return user, nil
}
