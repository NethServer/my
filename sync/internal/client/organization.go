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

// LogtoOrganization represents an organization in Logto
type LogtoOrganization struct {
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	CustomData  map[string]interface{} `json:"customData,omitempty"`
}

// GetOrganizations retrieves all organizations
func (c *LogtoClient) GetOrganizations() ([]LogtoOrganization, error) {
	logger.Debug("Fetching organizations")

	resp, err := c.makeRequest("GET", "/api/organizations", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}

	var organizations []LogtoOrganization
	if err := c.handlePaginatedResponse(resp, &organizations); err != nil {
		return nil, fmt.Errorf("failed to parse organizations response: %w", err)
	}

	logger.Debug("Retrieved %d organizations", len(organizations))
	return organizations, nil
}

// GetOrganizationByName searches for an organization by name using the q parameter
func (c *LogtoClient) GetOrganizationByName(name string) (*LogtoOrganization, error) {
	logger.Debug("Searching for organization with name: %s", name)

	// Use q parameter as per Logto API for organizations
	resp, err := c.makeRequest("GET", fmt.Sprintf("/api/organizations?q=%s", name), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search organizations: %w", err)
	}

	var organizations []LogtoOrganization
	if err := c.handlePaginatedResponse(resp, &organizations); err != nil {
		return nil, fmt.Errorf("failed to parse organizations response: %w", err)
	}

	logger.Debug("Retrieved %d organizations for search", len(organizations))

	// Find exact name match
	for _, org := range organizations {
		if org.Name == name {
			logger.Debug("Found organization: %s (ID: %s)", org.Name, org.ID)
			return &org, nil
		}
	}

	return nil, fmt.Errorf("organization with name '%s' not found", name)
}

// CreateOrganization creates a new organization
func (c *LogtoClient) CreateOrganization(org LogtoOrganization) (*LogtoOrganization, error) {
	logger.Debug("Creating organization: %s", org.Name)

	resp, err := c.makeRequest("POST", "/api/organizations", org)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	var result LogtoOrganization
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		if err := c.handleResponse(resp, resp.StatusCode, &result); err != nil {
			return nil, fmt.Errorf("failed to parse organization creation response: %w", err)
		}
		return &result, nil
	}

	return nil, c.handleResponse(resp, http.StatusCreated, nil)
}

// AddUserToOrganization adds a user to an organization
func (c *LogtoClient) AddUserToOrganization(organizationID, userID string) error {
	logger.Debug("Adding user %s to organization %s", userID, organizationID)

	data := map[string]interface{}{
		"userIds": []string{userID},
	}

	resp, err := c.makeRequest("POST", "/api/organizations/"+organizationID+"/users", data)
	if err != nil {
		return fmt.Errorf("failed to add user to organization: %w", err)
	}

	return c.handleCreationResponse(resp, nil)
}

// AssignOrganizationRoleToUser assigns an organization role to a user in an organization
func (c *LogtoClient) AssignOrganizationRoleToUser(organizationID, userID, organizationRoleID string) error {
	logger.Debug("Assigning organization role %s to user %s in organization %s", organizationRoleID, userID, organizationID)

	// Use the correct endpoint found through investigation
	data := map[string]interface{}{
		"organizationRoleIds": []string{organizationRoleID},
	}

	endpoint := "/api/organizations/" + organizationID + "/users/" + userID + "/roles"
	logger.Debug("Using endpoint: %s", endpoint)
	logger.Debug("Using payload: %+v", data)

	resp, err := c.makeRequest("POST", endpoint, data)
	if err != nil {
		return fmt.Errorf("failed to assign organization role to user: %w", err)
	}

	return c.handleCreationResponse(resp, nil)
}

// SetOrganizationJITRole sets the Just-in-Time organization role for an organization
func (c *LogtoClient) SetOrganizationJITRole(organizationID, organizationRoleID string) error {
	logger.Debug("Setting JIT organization role %s for organization %s", organizationRoleID, organizationID)

	data := map[string]interface{}{
		"organizationRoleIds": []string{organizationRoleID},
	}

	resp, err := c.makeRequest("POST", "/api/organizations/"+organizationID+"/jit/roles", data)
	if err != nil {
		return fmt.Errorf("failed to set JIT organization role: %w", err)
	}

	return c.handleCreationResponse(resp, nil)
}

// CreateOrganizationRoleSimple creates an organization role using a simple map structure
func (c *LogtoClient) CreateOrganizationRoleSimple(roleData map[string]interface{}) error {
	return c.createEntitySimple("/api/organization-roles", roleData, "organization role")
}

// CreateOrganizationSimple creates an organization using a simple map structure
func (c *LogtoClient) CreateOrganizationSimple(orgData map[string]interface{}) error {
	return c.createEntitySimple("/api/organizations", orgData, "organization")
}

// CreateOrganizationScopeSimple creates an organization scope using a simple map structure
func (c *LogtoClient) CreateOrganizationScopeSimple(scopeData map[string]interface{}) error {
	return c.createEntitySimple("/api/organization-scopes", scopeData, "organization scope")
}
