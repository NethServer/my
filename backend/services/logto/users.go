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
	"errors"
	"fmt"
	"net/http"

	"github.com/nethesis/my/backend/models"
)

// =============================================================================
// PUBLIC METHODS
// =============================================================================

// GetUserByID fetches a specific user by ID
func (c *LogtoManagementClient) GetUserByID(userID string) (*models.LogtoUser, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/users/%s", userID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("user not found")
	}

	return decodeResponse[models.LogtoUser](resp, []int{http.StatusOK}, "fetch user")
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

	return decodeResponse[models.LogtoUser](resp, []int{http.StatusCreated, http.StatusOK}, "create user")
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

	return decodeResponse[models.LogtoUser](resp, []int{http.StatusOK}, "update user")
}

// DeleteUser deletes a user from Logto
func (c *LogtoManagementClient) DeleteUser(userID string) error {
	resp, err := c.makeRequest("DELETE", fmt.Sprintf("/users/%s", userID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return checkStatus(resp, []int{http.StatusNoContent, http.StatusOK}, "delete user")
}

// AssignUserToOrganization assigns a user to an organization without any roles
// Roles should be assigned separately using AssignOrganizationRolesToUser
func (c *LogtoManagementClient) AssignUserToOrganization(orgID, userID string) error {
	requestBody := map[string]interface{}{
		"userIds": []string{userID},
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal organization assignment request: %w", err)
	}

	resp, err := c.makeRequest("POST", fmt.Sprintf("/organizations/%s/users", orgID), bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to assign user to organization: %w", err)
	}

	return checkStatus(resp, []int{http.StatusCreated, http.StatusOK}, "assign user to organization")
}

// RemoveUserFromOrganization removes a user from an organization
func (c *LogtoManagementClient) RemoveUserFromOrganization(orgID, userID string) error {
	resp, err := c.makeRequest("DELETE", fmt.Sprintf("/organizations/%s/users/%s", orgID, userID), nil)
	if err != nil {
		return fmt.Errorf("failed to remove user from organization: %w", err)
	}

	return checkStatus(resp, []int{http.StatusNoContent, http.StatusOK}, "remove user from organization")
}

// RemoveUserFromOrganizationRole removes a specific organization role from a user
func (c *LogtoManagementClient) RemoveUserFromOrganizationRole(orgID, userID, roleID string) error {
	resp, err := c.makeRequest("DELETE", fmt.Sprintf("/organizations/%s/users/%s/roles/%s", orgID, userID, roleID), nil)
	if err != nil {
		return fmt.Errorf("failed to remove user from organization role: %w", err)
	}

	return checkStatus(resp, []int{http.StatusNoContent, http.StatusOK}, "remove user from organization role")
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

	return checkStatus(resp, []int{http.StatusOK, http.StatusNoContent}, "update user password")
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

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_ = resp.Body.Close()
		return fmt.Errorf("invalid current password")
	}

	return checkStatus(resp, []int{http.StatusOK, http.StatusNoContent}, "verify password")
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

	return checkStatus(resp, []int{http.StatusOK, http.StatusNoContent}, "update user password")
}

// SuspendUser suspends a user in Logto
func (c *LogtoManagementClient) SuspendUser(userID string) error {
	requestBody := map[string]interface{}{
		"isSuspended": true,
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal suspend user request: %w", err)
	}

	resp, err := c.makeRequest("PATCH", fmt.Sprintf("/users/%s/is-suspended", userID), bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to suspend user: %w", err)
	}

	return checkStatus(resp, []int{http.StatusOK, http.StatusNoContent}, "suspend user")
}

// ReactivateUser reactivates a suspended user in Logto
func (c *LogtoManagementClient) ReactivateUser(userID string) error {
	requestBody := map[string]interface{}{
		"isSuspended": false,
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal reactivate user request: %w", err)
	}

	resp, err := c.makeRequest("PATCH", fmt.Sprintf("/users/%s/is-suspended", userID), bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to reactivate user: %w", err)
	}

	return checkStatus(resp, []int{http.StatusOK, http.StatusNoContent}, "reactivate user")
}

// ErrVerificationCodeInvalid is returned when Logto rejects the code presented
// for an email address: wrong code, expired code, or no code ever sent.
var ErrVerificationCodeInvalid = errors.New("verification code invalid or expired")

// SendEmailVerificationCode asks Logto to email a one-time code to the address.
// Logto stores the code against the address (not against a user), so the same
// address must be presented again to VerifyEmailCode. The tenant's email
// connector does the delivery.
func (c *LogtoManagementClient) SendEmailVerificationCode(email string) error {
	reqBody, err := json.Marshal(map[string]string{"email": email})
	if err != nil {
		return fmt.Errorf("failed to marshal verification code request: %w", err)
	}

	resp, err := c.makeRequest("POST", "/verification-codes", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to send verification code: %w", err)
	}

	return checkStatus(resp, []int{http.StatusOK, http.StatusNoContent}, "send verification code")
}

// VerifyEmailCode checks a one-time code Logto emailed to the address. Logto
// answers 400 for a wrong or expired code, which is mapped to
// ErrVerificationCodeInvalid so the caller can tell "try again" from an outage.
func (c *LogtoManagementClient) VerifyEmailCode(email, code string) error {
	reqBody, err := json.Marshal(map[string]string{"email": email, "verificationCode": code})
	if err != nil {
		return fmt.Errorf("failed to marshal verification request: %w", err)
	}

	resp, err := c.makeRequest("POST", "/verification-codes/verify", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to verify code: %w", err)
	}

	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound {
		_ = resp.Body.Close()
		return ErrVerificationCodeInvalid
	}

	return checkStatus(resp, []int{http.StatusOK, http.StatusNoContent}, "verify code")
}
