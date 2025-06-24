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
	"fmt"
	"net/http"

	"github.com/nethesis/my/sync/internal/logger"
)

// LogtoOrganizationRole represents an organization role in Logto
type LogtoOrganizationRole struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// LogtoOrganizationScope represents an organization scope in Logto
type LogtoOrganizationScope struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GetOrganizationScopes retrieves all organization scopes
func (c *LogtoClient) GetOrganizationScopes() ([]LogtoOrganizationScope, error) {
	logger.Debug("Fetching organization scopes")

	resp, err := c.makeRequest("GET", "/api/organization-scopes", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization scopes: %w", err)
	}

	var scopes []LogtoOrganizationScope
	if err := c.handlePaginatedResponse(resp, &scopes); err != nil {
		return nil, fmt.Errorf("failed to parse organization scopes response: %w", err)
	}

	logger.Debug("Retrieved %d organization scopes", len(scopes))
	return scopes, nil
}

// CreateOrganizationScope creates a new organization scope
func (c *LogtoClient) CreateOrganizationScope(scope LogtoOrganizationScope) error {
	logger.Debug("Creating organization scope: %s", scope.Name)

	resp, err := c.makeRequest("POST", "/api/organization-scopes", scope)
	if err != nil {
		return fmt.Errorf("failed to create organization scope: %w", err)
	}

	return c.handleResponse(resp, http.StatusCreated, nil)
}

// UpdateOrganizationScope updates an existing organization scope
func (c *LogtoClient) UpdateOrganizationScope(scopeID string, scope LogtoOrganizationScope) error {
	logger.Debug("Updating organization scope: %s", scopeID)

	resp, err := c.makeRequest("PATCH", "/api/organization-scopes/"+scopeID, scope)
	if err != nil {
		return fmt.Errorf("failed to update organization scope: %w", err)
	}

	return c.handleResponse(resp, http.StatusOK, nil)
}

// DeleteOrganizationScope deletes an organization scope
func (c *LogtoClient) DeleteOrganizationScope(scopeID string) error {
	logger.Debug("Deleting organization scope: %s", scopeID)

	resp, err := c.makeRequest("DELETE", "/api/organization-scopes/"+scopeID, nil)
	if err != nil {
		return fmt.Errorf("failed to delete organization scope: %w", err)
	}

	return c.handleResponse(resp, http.StatusNoContent, nil)
}

// GetOrganizationRoles retrieves all organization roles
func (c *LogtoClient) GetOrganizationRoles() ([]LogtoOrganizationRole, error) {
	logger.Debug("Fetching organization roles")

	resp, err := c.makeRequest("GET", "/api/organization-roles", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization roles: %w", err)
	}

	var roles []LogtoOrganizationRole
	if err := c.handlePaginatedResponse(resp, &roles); err != nil {
		return nil, fmt.Errorf("failed to parse organization roles response: %w", err)
	}

	logger.Debug("Retrieved %d organization roles", len(roles))
	return roles, nil
}

// CreateOrganizationRole creates a new organization role
func (c *LogtoClient) CreateOrganizationRole(role LogtoOrganizationRole) error {
	logger.Debug("Creating organization role: %s", role.Name)

	resp, err := c.makeRequest("POST", "/api/organization-roles", role)
	if err != nil {
		return fmt.Errorf("failed to create organization role: %w", err)
	}

	return c.handleResponse(resp, http.StatusCreated, nil)
}

// UpdateOrganizationRole updates an existing organization role
func (c *LogtoClient) UpdateOrganizationRole(roleID string, role LogtoOrganizationRole) error {
	logger.Debug("Updating organization role: %s", roleID)

	resp, err := c.makeRequest("PATCH", "/api/organization-roles/"+roleID, role)
	if err != nil {
		return fmt.Errorf("failed to update organization role: %w", err)
	}

	return c.handleResponse(resp, http.StatusOK, nil)
}

// DeleteOrganizationRole deletes an organization role
func (c *LogtoClient) DeleteOrganizationRole(roleID string) error {
	logger.Debug("Deleting organization role: %s", roleID)

	resp, err := c.makeRequest("DELETE", "/api/organization-roles/"+roleID, nil)
	if err != nil {
		return fmt.Errorf("failed to delete organization role: %w", err)
	}

	return c.handleResponse(resp, http.StatusNoContent, nil)
}

// AssignScopeToOrganizationRole assigns a scope to an organization role
func (c *LogtoClient) AssignScopeToOrganizationRole(roleID, scopeID string) error {
	logger.Debug("Assigning scope %s to organization role %s", scopeID, roleID)

	payload := map[string][]string{
		"organizationScopeIds": {scopeID},
	}

	resp, err := c.makeRequest("POST", "/api/organization-roles/"+roleID+"/scopes", payload)
	if err != nil {
		return fmt.Errorf("failed to assign scope to organization role: %w", err)
	}

	return c.handleResponse(resp, http.StatusCreated, nil)
}

// GetOrganizationRoleScopes gets all scopes for an organization role
func (c *LogtoClient) GetOrganizationRoleScopes(roleID string) ([]LogtoOrganizationScope, error) {
	logger.Debug("Fetching scopes for organization role: %s", roleID)

	resp, err := c.makeRequest("GET", "/api/organization-roles/"+roleID+"/scopes", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization role scopes: %w", err)
	}

	var scopes []LogtoOrganizationScope
	if err := c.handlePaginatedResponse(resp, &scopes); err != nil {
		return nil, fmt.Errorf("failed to parse organization role scopes response: %w", err)
	}

	logger.Debug("Retrieved %d scopes for organization role %s", len(scopes), roleID)
	return scopes, nil
}

// RemoveScopeFromOrganizationRole removes a scope from an organization role
func (c *LogtoClient) RemoveScopeFromOrganizationRole(roleID, scopeID string) error {
	logger.Debug("Removing scope %s from organization role %s", scopeID, roleID)

	resp, err := c.makeRequest("DELETE", "/api/organization-roles/"+roleID+"/scopes/"+scopeID, nil)
	if err != nil {
		return fmt.Errorf("failed to remove scope from organization role: %w", err)
	}

	return c.handleResponse(resp, http.StatusNoContent, nil)
}
