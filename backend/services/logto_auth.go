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
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logs"
)

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

// GetUserProfileFromLogto fetches complete user profile from Logto Management API
func GetUserProfileFromLogto(userID string) (*LogtoUser, error) {
	client := NewLogtoManagementClient()

	// Use the GetUserByID method we already have
	user, err := client.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	logs.Logs.Printf("[DEBUG][LOGTO] Profile API response: username=%s, email=%s, name=%s",
		user.Username, user.PrimaryEmail, user.Name)

	return user, nil
}
