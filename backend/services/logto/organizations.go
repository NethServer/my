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

	return decodeSliceResponse[models.LogtoOrganization](resp, []int{http.StatusOK}, "fetch user organizations")
}

func (c *LogtoManagementClient) GetAllOrganizations() ([]models.LogtoOrganization, error) {
	resp, err := c.makeRequest("GET", "/organizations", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organizations: %w", err)
	}

	return decodeSliceResponse[models.LogtoOrganization](resp, []int{http.StatusOK}, "fetch organizations")
}

func (c *LogtoManagementClient) GetOrganizationByID(orgID string) (*models.LogtoOrganization, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organizations/%s", orgID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("organization not found")
	}

	return decodeResponse[models.LogtoOrganization](resp, []int{http.StatusOK}, "fetch organization")
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

	return decodeResponse[models.LogtoOrganization](resp, []int{http.StatusCreated}, "create organization")
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

	return decodeResponse[models.LogtoOrganization](resp, []int{http.StatusOK}, "update organization")
}

func (c *LogtoManagementClient) DeleteOrganization(orgID string) error {
	resp, err := c.makeRequest("DELETE", fmt.Sprintf("/organizations/%s", orgID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	return checkStatus(resp, []int{http.StatusNoContent}, "delete organization")
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

	return checkStatus(resp, []int{http.StatusCreated}, "set organization JIT roles")
}
