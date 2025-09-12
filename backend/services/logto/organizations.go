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

// =============================================================================
// PUBLIC METHODS
// =============================================================================

func (c *LogtoManagementClient) GetUserOrganizations(userID string) ([]models.LogtoOrganization, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/users/%s/organizations", userID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user organizations: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user organizations, status %d: %s", resp.StatusCode, string(body))
	}

	var orgs []models.LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return nil, fmt.Errorf("failed to decode user organizations: %w", err)
	}

	return orgs, nil
}

func (c *LogtoManagementClient) GetAllOrganizations() ([]models.LogtoOrganization, error) {
	resp, err := c.makeRequest("GET", "/organizations", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organizations: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organizations, status %d: %s", resp.StatusCode, string(body))
	}

	var orgs []models.LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return nil, fmt.Errorf("failed to decode organizations: %w", err)
	}

	return orgs, nil
}

func (c *LogtoManagementClient) GetOrganizationByID(orgID string) (*models.LogtoOrganization, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organizations/%s", orgID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("organization not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organization, status %d: %s", resp.StatusCode, string(body))
	}

	var org models.LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&org); err != nil {
		return nil, fmt.Errorf("failed to decode organization: %w", err)
	}

	return &org, nil
}

func (c *LogtoManagementClient) CreateOrganization(request models.CreateOrganizationRequest) (*models.LogtoOrganization, error) {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create organization request: %w", err)
	}

	resp, err := c.makeRequest("POST", "/organizations", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create organization, status %d: %s", resp.StatusCode, string(body))
	}

	var org models.LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&org); err != nil {
		return nil, fmt.Errorf("failed to decode created organization: %w", err)
	}

	return &org, nil
}

func (c *LogtoManagementClient) UpdateOrganization(orgID string, request models.UpdateOrganizationRequest) (*models.LogtoOrganization, error) {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update organization request: %w", err)
	}

	resp, err := c.makeRequest("PATCH", fmt.Sprintf("/organizations/%s", orgID), bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update organization, status %d: %s", resp.StatusCode, string(body))
	}

	var org models.LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&org); err != nil {
		return nil, fmt.Errorf("failed to decode updated organization: %w", err)
	}

	return &org, nil
}

func (c *LogtoManagementClient) DeleteOrganization(orgID string) error {
	resp, err := c.makeRequest("DELETE", fmt.Sprintf("/organizations/%s", orgID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete organization, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SetOrganizationJitRoles configures JIT roles for an organization
func (c *LogtoManagementClient) SetOrganizationJitRoles(orgID string, roleIDs []string) error {
	requestBody := models.CreateOrganizationJitRolesRequest{
		OrganizationRoleIds: roleIDs,
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal JIT roles request: %w", err)
	}

	resp, err := c.makeRequest("POST", fmt.Sprintf("/organizations/%s/jit/roles", orgID), bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to set organization JIT roles: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to set organization JIT roles, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
