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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// CustomJwtClaimsPayload represents the payload for custom JWT claims API
type CustomJwtClaimsPayload struct {
	Script string `json:"script"`
}

// UpdateCustomJwtClaims updates the custom JWT claims script in Logto
func (c *LogtoClient) UpdateCustomJwtClaims(scriptPath string) error {
	// Read the script file
	absolutePath, err := filepath.Abs(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to resolve script path: %w", err)
	}

	scriptContent, err := os.ReadFile(absolutePath)
	if err != nil {
		return fmt.Errorf("failed to read script file %s: %w", absolutePath, err)
	}

	// Prepare the payload
	payload := CustomJwtClaimsPayload{
		Script: string(scriptContent),
	}

	// Use the makeRequest method to handle authentication
	resp, err := c.makeRequest("PUT", "/api/configs/jwt-customizer/access-token", payload)
	if err != nil {
		return fmt.Errorf("failed to update custom JWT claims: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update custom JWT claims (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetCustomJwtClaims retrieves the current custom JWT claims script from Logto
func (c *LogtoClient) GetCustomJwtClaims() (string, error) {
	// Use the makeRequest method to handle authentication
	resp, err := c.makeRequest("GET", "/api/configs/jwt-customizer/access-token", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get custom JWT claims: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil // No custom JWT claims configured
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get custom JWT claims (status %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Try to parse as array first (Logto might return an array of configurations)
	var arrayResponse []CustomJwtClaimsPayload
	if err := json.Unmarshal(body, &arrayResponse); err == nil && len(arrayResponse) > 0 {
		return arrayResponse[0].Script, nil
	}

	// Try to parse as single object
	var response CustomJwtClaimsPayload
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response (tried both array and object): %w", err)
	}

	return response.Script, nil
}