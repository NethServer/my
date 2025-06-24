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

// LogtoResource represents a resource in Logto
type LogtoResource struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Indicator      string `json:"indicator"`
	IsDefault      bool   `json:"isDefault"`
	AccessTokenTTL int    `json:"accessTokenTtl"`
}

// GetResources retrieves all resources
func (c *LogtoClient) GetResources() ([]LogtoResource, error) {
	logger.Debug("Fetching resources")

	resp, err := c.makeRequest("GET", "/api/resources", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources: %w", err)
	}

	var resources []LogtoResource
	if err := c.handlePaginatedResponse(resp, &resources); err != nil {
		return nil, fmt.Errorf("failed to parse resources response: %w", err)
	}

	logger.Debug("Retrieved %d resources", len(resources))
	return resources, nil
}

// CreateResource creates a new resource
func (c *LogtoClient) CreateResource(resource LogtoResource) error {
	logger.Debug("Creating resource: %s", resource.Name)

	resp, err := c.makeRequest("POST", "/api/resources", resource)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	return c.handleResponse(resp, http.StatusCreated, nil)
}

// UpdateResource updates an existing resource
func (c *LogtoClient) UpdateResource(resourceID string, resource LogtoResource) error {
	logger.Debug("Updating resource: %s", resourceID)

	resp, err := c.makeRequest("PATCH", "/api/resources/"+resourceID, resource)
	if err != nil {
		return fmt.Errorf("failed to update resource: %w", err)
	}

	return c.handleResponse(resp, http.StatusOK, nil)
}

// DeleteResource deletes a resource
func (c *LogtoClient) DeleteResource(resourceID string) error {
	logger.Debug("Deleting resource: %s", resourceID)

	resp, err := c.makeRequest("DELETE", "/api/resources/"+resourceID, nil)
	if err != nil {
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	return c.handleResponse(resp, http.StatusNoContent, nil)
}

// GetScopes gets all scopes for a resource
func (c *LogtoClient) GetScopes(resourceID string) ([]LogtoScope, error) {
	logger.Debug("Fetching scopes for resource: %s", resourceID)

	resp, err := c.makeRequest("GET", "/api/resources/"+resourceID+"/scopes", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get scopes: %w", err)
	}

	var scopes []LogtoScope
	if err := c.handlePaginatedResponse(resp, &scopes); err != nil {
		return nil, fmt.Errorf("failed to parse scopes response: %w", err)
	}

	logger.Debug("Retrieved %d scopes for resource %s", len(scopes), resourceID)
	return scopes, nil
}

// CreateScope creates a new scope for a resource
func (c *LogtoClient) CreateScope(resourceID string, scope LogtoScope) error {
	logger.Debug("Creating scope %s for resource %s", scope.Name, resourceID)

	resp, err := c.makeRequest("POST", "/api/resources/"+resourceID+"/scopes", scope)
	if err != nil {
		return fmt.Errorf("failed to create scope: %w", err)
	}

	return c.handleResponse(resp, http.StatusCreated, nil)
}

// UpdateScope updates an existing scope
func (c *LogtoClient) UpdateScope(resourceID, scopeID string, scope LogtoScope) error {
	logger.Debug("Updating scope %s for resource %s", scopeID, resourceID)

	resp, err := c.makeRequest("PATCH", "/api/resources/"+resourceID+"/scopes/"+scopeID, scope)
	if err != nil {
		return fmt.Errorf("failed to update scope: %w", err)
	}

	return c.handleResponse(resp, http.StatusOK, nil)
}

// DeleteScope deletes a scope from a resource
func (c *LogtoClient) DeleteScope(resourceID, scopeID string) error {
	logger.Debug("Deleting scope %s from resource %s", scopeID, resourceID)

	resp, err := c.makeRequest("DELETE", "/api/resources/"+resourceID+"/scopes/"+scopeID, nil)
	if err != nil {
		return fmt.Errorf("failed to delete scope: %w", err)
	}

	return c.handleResponse(resp, http.StatusNoContent, nil)
}
