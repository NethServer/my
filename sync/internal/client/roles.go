/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package client

import (
	"fmt"
	"net/http"

	"github.com/nethesis/my/sync/internal/logger"
)

// LogtoRole represents a user role in Logto
type LogtoRole struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// LogtoScope represents a scope/permission in Logto
type LogtoScope struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GetRoles retrieves all user roles
func (c *LogtoClient) GetRoles() ([]LogtoRole, error) {
	logger.Debug("Fetching user roles")

	resp, err := c.makeRequest("GET", "/api/roles", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	var roles []LogtoRole
	if err := c.handlePaginatedResponse(resp, &roles); err != nil {
		return nil, fmt.Errorf("failed to parse user roles response: %w", err)
	}

	logger.Debug("Retrieved %d user roles", len(roles))
	return roles, nil
}

// CreateRole creates a new user role
func (c *LogtoClient) CreateRole(role LogtoRole) error {
	logger.Debug("Creating user role: %s", role.Name)

	resp, err := c.makeRequest("POST", "/api/roles", role)
	if err != nil {
		return fmt.Errorf("failed to create user role: %w", err)
	}

	// Logto sometimes returns 200 instead of 201 for creation
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return c.handleResponse(resp, http.StatusCreated, nil)
	}
	_ = resp.Body.Close()
	return nil
}

// UpdateRole updates an existing user role
func (c *LogtoClient) UpdateRole(roleID string, role LogtoRole) error {
	logger.Debug("Updating user role: %s", roleID)

	resp, err := c.makeRequest("PATCH", "/api/roles/"+roleID, role)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	return c.handleResponse(resp, http.StatusOK, nil)
}

// DeleteRole deletes a user role
func (c *LogtoClient) DeleteRole(roleID string) error {
	logger.Debug("Deleting user role: %s", roleID)

	resp, err := c.makeRequest("DELETE", "/api/roles/"+roleID, nil)
	if err != nil {
		return fmt.Errorf("failed to delete user role: %w", err)
	}

	return c.handleResponse(resp, http.StatusNoContent, nil)
}

// GetRolePermissions gets all permissions (scopes) for a user role
func (c *LogtoClient) GetRolePermissions(roleID string) ([]LogtoScope, error) {
	logger.Debug("Fetching permissions for user role: %s", roleID)

	resp, err := c.makeRequest("GET", "/api/roles/"+roleID+"/scopes", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}

	var scopes []LogtoScope
	if err := c.handlePaginatedResponse(resp, &scopes); err != nil {
		return nil, fmt.Errorf("failed to parse role permissions response: %w", err)
	}

	logger.Debug("Retrieved %d permissions for user role %s", len(scopes), roleID)
	return scopes, nil
}

// AssignPermissionsToRole assigns multiple permissions to a role
func (c *LogtoClient) AssignPermissionsToRole(roleID string, scopeIDs []string) error {
	logger.Debug("Assigning %d permissions to user role %s", len(scopeIDs), roleID)

	payload := map[string][]string{
		"scopeIds": scopeIDs,
	}

	resp, err := c.makeRequest("POST", "/api/roles/"+roleID+"/scopes", payload)
	if err != nil {
		return fmt.Errorf("failed to assign permissions to role: %w", err)
	}

	return c.handleResponse(resp, http.StatusCreated, nil)
}

// RemovePermissionsFromRole removes multiple permissions from a role
func (c *LogtoClient) RemovePermissionsFromRole(roleID string, scopeIDs []string) error {
	logger.Debug("Removing %d permissions from user role %s", len(scopeIDs), roleID)

	// Remove each scope individually as Logto doesn't support bulk removal
	for _, scopeID := range scopeIDs {
		resp, err := c.makeRequest("DELETE", "/api/roles/"+roleID+"/scopes/"+scopeID, nil)
		if err != nil {
			return fmt.Errorf("failed to remove permission %s from role: %w", scopeID, err)
		}

		if err := c.handleResponse(resp, http.StatusNoContent, nil); err != nil {
			return fmt.Errorf("failed to remove permission %s from role: %w", scopeID, err)
		}
	}

	return nil
}

// AssignRoleToUser assigns a role to a user
func (c *LogtoClient) AssignRoleToUser(userID, roleID string) error {
	logger.Debug("Assigning role %s to user %s", roleID, userID)

	data := map[string]interface{}{
		"roleIds": []string{roleID},
	}

	resp, err := c.makeRequest("POST", "/api/users/"+userID+"/roles", data)
	if err != nil {
		return fmt.Errorf("failed to assign role to user: %w", err)
	}

	// Role assignment typically returns 201 or 200
	return c.handleCreationResponse(resp, nil)
}

// CreateRoleSimple creates a role using a simple map structure
func (c *LogtoClient) CreateRoleSimple(roleData map[string]interface{}) error {
	return c.createEntitySimple("/api/roles", roleData, "role")
}
