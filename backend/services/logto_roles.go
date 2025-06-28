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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// GetUserRoles fetches user roles from Logto Management API
func (c *LogtoManagementClient) GetUserRoles(userID string) ([]LogtoRole, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/users/%s/roles", userID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user roles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user roles, status %d: %s", resp.StatusCode, string(body))
	}

	var roles []LogtoRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode user roles: %w", err)
	}

	return roles, nil
}

// GetRoleScopes fetches scopes for a role
func (c *LogtoManagementClient) GetRoleScopes(roleID string) ([]LogtoScope, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/roles/%s/scopes", roleID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch role scopes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch role scopes, status %d: %s", resp.StatusCode, string(body))
	}

	var scopes []LogtoScope
	if err := json.NewDecoder(resp.Body).Decode(&scopes); err != nil {
		return nil, fmt.Errorf("failed to decode role scopes: %w", err)
	}

	return scopes, nil
}

// GetUserOrganizationRoles fetches user's roles in an organization
func (c *LogtoManagementClient) GetUserOrganizationRoles(orgID, userID string) ([]LogtoOrganizationRole, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organizations/%s/users/%s/roles", orgID, userID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user organization roles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user organization roles, status %d: %s", resp.StatusCode, string(body))
	}

	var roles []LogtoOrganizationRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode user organization roles: %w", err)
	}

	return roles, nil
}

// GetOrganizationRoleScopes fetches scopes for an organization role
func (c *LogtoManagementClient) GetOrganizationRoleScopes(roleID string) ([]LogtoScope, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organization-roles/%s/scopes", roleID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization role scopes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organization role scopes, status %d: %s", resp.StatusCode, string(body))
	}

	var scopes []LogtoScope
	if err := json.NewDecoder(resp.Body).Decode(&scopes); err != nil {
		return nil, fmt.Errorf("failed to decode organization role scopes: %w", err)
	}

	return scopes, nil
}

// GetRoleByName finds a role by name
func (c *LogtoManagementClient) GetRoleByName(roleName string) (*LogtoRole, error) {
	resp, err := c.makeRequest("GET", "/roles", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch roles, status %d: %s", resp.StatusCode, string(body))
	}

	var roles []LogtoRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode roles: %w", err)
	}

	for _, role := range roles {
		if role.Name == roleName {
			return &role, nil
		}
	}

	return nil, fmt.Errorf("role '%s' not found", roleName)
}

// GetOrganizationRoleByName finds an organization role by name
func (c *LogtoManagementClient) GetOrganizationRoleByName(roleName string) (*LogtoOrganizationRole, error) {
	resp, err := c.makeRequest("GET", "/organization-roles", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization roles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organization roles, status %d: %s", resp.StatusCode, string(body))
	}

	var roles []LogtoOrganizationRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode organization roles: %w", err)
	}

	for _, role := range roles {
		if role.Name == roleName {
			return &role, nil
		}
	}

	return nil, fmt.Errorf("organization role '%s' not found", roleName)
}

// AssignUserRoles assigns roles to a user
func (c *LogtoManagementClient) AssignUserRoles(userID string, roleIDs []string) error {
	requestBody := map[string]interface{}{
		"roleIds": roleIDs,
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal role assignment request: %w", err)
	}

	resp, err := c.makeRequest("POST", fmt.Sprintf("/users/%s/roles", userID), bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to assign user roles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to assign user roles, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// AssignOrganizationRolesToUser assigns specific organization roles to a user
// This is the correct API endpoint: POST /organizations/{orgId}/users/{userId}/roles
func (c *LogtoManagementClient) AssignOrganizationRolesToUser(orgID, userID string, roleIDs []string, roleNames []string) error {
	requestBody := map[string]interface{}{}

	// Add role IDs if provided
	if len(roleIDs) > 0 {
		requestBody["organizationRoleIds"] = roleIDs
	}

	// Add role names if provided
	if len(roleNames) > 0 {
		requestBody["organizationRoleNames"] = roleNames
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal organization role assignment request: %w", err)
	}

	resp, err := c.makeRequest("POST", fmt.Sprintf("/organizations/%s/users/%s/roles", orgID, userID), bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to assign organization roles to user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to assign organization roles to user, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// EnrichUserWithRolesAndPermissions fetches complete roles and permissions from Logto Management API
func EnrichUserWithRolesAndPermissions(userID string) (*models.User, error) {
	logger.ComponentLogger("logto").Debug().
		Str("operation", "enrich_user_start").
		Str("user_id", userID).
		Msg("Starting user enrichment")
	client := NewLogtoManagementClient()

	// Initialize user
	user := &models.User{
		ID:               userID,
		UserRoles:        []string{},
		UserPermissions:  []string{},
		OrgRole:          "",
		OrgPermissions:   []string{},
		OrganizationID:   "",
		OrganizationName: "",
	}

	// Fetch user roles (technical capabilities)
	logger.ComponentLogger("logto").Debug().
		Str("operation", "fetch_user_roles").
		Str("user_id", userID).
		Msg("Fetching user roles")
	userRoles, err := client.GetUserRoles(userID)
	if err != nil {
		logger.ComponentLogger("logto").Warn().
			Err(err).
			Str("operation", "fetch_user_roles").
			Str("user_id", userID).
			Msg("Failed to fetch user roles")
	} else {
		logger.ComponentLogger("logto").Debug().
			Int("role_count", len(userRoles)).
			Str("operation", "fetch_user_roles").
			Str("user_id", userID).
			Msg("Found user roles")
		// Extract role names
		for _, role := range userRoles {
			user.UserRoles = append(user.UserRoles, role.Name)
		}

		// Fetch permissions for each user role
		for _, role := range userRoles {
			scopes, err := client.GetRoleScopes(role.ID)
			if err != nil {
				logger.ComponentLogger("logto").Warn().
					Err(err).
					Str("operation", "fetch_role_scopes").
					Str("role_id", role.ID).
					Msg("Failed to fetch role scopes")
				continue
			}
			for _, scope := range scopes {
				user.UserPermissions = append(user.UserPermissions, scope.Name)
			}
		}
	}

	// Fetch user organizations
	logger.ComponentLogger("logto").Debug().
		Str("operation", "fetch_user_orgs").
		Str("user_id", userID).
		Msg("Fetching user organizations")
	orgs, err := client.GetUserOrganizations(userID)
	if err != nil {
		logger.ComponentLogger("logto").Warn().
			Err(err).
			Str("operation", "fetch_user_orgs").
			Str("user_id", userID).
			Msg("Failed to fetch user organizations")
	} else {
		logger.ComponentLogger("logto").Debug().
			Int("org_count", len(orgs)).
			Str("operation", "fetch_user_orgs").
			Str("user_id", userID).
			Msg("Found user organizations")
		if len(orgs) > 0 {
			// Use first organization as primary
			primaryOrg := orgs[0]
			user.OrganizationID = primaryOrg.ID
			user.OrganizationName = primaryOrg.Name

			// Fetch user's roles in this organization
			orgRoles, err := client.GetUserOrganizationRoles(primaryOrg.ID, userID)
			if err != nil {
				logger.ComponentLogger("logto").Warn().
					Err(err).
					Str("operation", "fetch_org_roles").
					Str("user_id", userID).
					Str("org_id", primaryOrg.ID).
					Msg("Failed to fetch organization roles")
			} else if len(orgRoles) > 0 {
				// Use first organization role as primary
				primaryOrgRole := orgRoles[0]
				user.OrgRole = primaryOrgRole.Name

				// Fetch permissions for organization role
				orgScopes, err := client.GetOrganizationRoleScopes(primaryOrgRole.ID)
				if err != nil {
					logger.ComponentLogger("logto").Warn().
						Err(err).
						Str("operation", "fetch_org_role_scopes").
						Str("role_id", primaryOrgRole.ID).
						Msg("Failed to fetch organization role scopes")
				} else {
					for _, scope := range orgScopes {
						user.OrgPermissions = append(user.OrgPermissions, scope.Name)
					}
				}
			}
		}
	}

	// Remove duplicates from permissions
	user.UserPermissions = removeDuplicates(user.UserPermissions)
	user.OrgPermissions = removeDuplicates(user.OrgPermissions)

	logger.ComponentLogger("logto").Info().
		Str("operation", "enrich_user_complete").
		Str("user_id", userID).
		Int("user_roles_count", len(user.UserRoles)).
		Int("user_permissions_count", len(user.UserPermissions)).
		Str("org_role", user.OrgRole).
		Int("org_permissions_count", len(user.OrgPermissions)).
		Msg("User enrichment completed")

	return user, nil
}

// removeDuplicates removes duplicate strings from a slice
func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	return result
}
