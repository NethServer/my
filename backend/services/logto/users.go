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
	"net/http"

	"github.com/nethesis/my/backend/models"
)

// GetUserByID fetches a specific user by ID
func (c *LogtoManagementClient) GetUserByID(userID string) (*models.LogtoUser, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/users/%s", userID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("user not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user, status %d: %s", resp.StatusCode, string(body))
	}

	var user models.LogtoUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user: %w", err)
	}

	return &user, nil
}

// CreateUser creates a new account in Logto
func (c *LogtoManagementClient) CreateUser(request models.CreateUserRequest) (*models.LogtoUser, error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user request: %w", err)
	}

	resp, err := c.makeRequest("POST", "/users", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create user, status %d: %s", resp.StatusCode, string(body))
	}

	var user models.LogtoUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode created user: %w", err)
	}

	return &user, nil
}

// UpdateUser updates an existing user in Logto
func (c *LogtoManagementClient) UpdateUser(userID string, request models.UpdateUserRequest) (*models.LogtoUser, error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user update request: %w", err)
	}

	resp, err := c.makeRequest("PATCH", fmt.Sprintf("/users/%s", userID), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update user, status %d: %s", resp.StatusCode, string(body))
	}

	var user models.LogtoUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode updated user: %w", err)
	}

	return &user, nil
}

// DeleteUser deletes a user from Logto
func (c *LogtoManagementClient) DeleteUser(userID string) error {
	resp, err := c.makeRequest("DELETE", fmt.Sprintf("/users/%s", userID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete user, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// AssignUserToOrganization assigns a user to an organization without any roles
// Roles should be assigned separately using AssignOrganizationRolesToUser
func (c *LogtoManagementClient) AssignUserToOrganization(orgID, userID string) error {
	requestBody := map[string]interface{}{
		"userIds": []string{userID},
		// No organizationRoleIds - roles assigned separately
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal organization assignment request: %w", err)
	}

	resp, err := c.makeRequest("POST", fmt.Sprintf("/organizations/%s/users", orgID), bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to assign user to organization: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to assign user to organization, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// RemoveUserFromOrganization removes a user from an organization
func (c *LogtoManagementClient) RemoveUserFromOrganization(orgID, userID string) error {
	resp, err := c.makeRequest("DELETE", fmt.Sprintf("/organizations/%s/users/%s", orgID, userID), nil)
	if err != nil {
		return fmt.Errorf("failed to remove user from organization: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to remove user from organization, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// RemoveUserFromOrganizationRole removes a specific organization role from a user
func (c *LogtoManagementClient) RemoveUserFromOrganizationRole(orgID, userID, roleID string) error {
	resp, err := c.makeRequest("DELETE", fmt.Sprintf("/organizations/%s/users/%s/roles/%s", orgID, userID, roleID), nil)
	if err != nil {
		return fmt.Errorf("failed to remove user from organization role: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to remove user from organization role, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ResetUserPassword resets a user's password in Logto (admin function)
func (c *LogtoManagementClient) ResetUserPassword(userID string, password string) error {
	requestBody := map[string]interface{}{
		"password": password,
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal password update request: %w", err)
	}

	resp, err := c.makeRequest("PATCH", fmt.Sprintf("/users/%s/password", userID), bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update user password, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// VerifyUserPassword verifies a user's current password using Logto API
func (c *LogtoManagementClient) VerifyUserPassword(userID, password string) error {
	requestBody := map[string]interface{}{
		"password": password,
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal password verification request: %w", err)
	}

	resp, err := c.makeRequest("POST", fmt.Sprintf("/users/%s/password/verify", userID), bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to verify user password: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid current password")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to verify password, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateUserPassword updates a user's password using Logto API (user self-service)
func (c *LogtoManagementClient) UpdateUserPassword(userID, newPassword string) error {
	requestBody := map[string]interface{}{
		"password": newPassword,
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal password update request: %w", err)
	}

	resp, err := c.makeRequest("PATCH", fmt.Sprintf("/users/%s/password", userID), bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update user password, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
