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

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// =============================================================================
// PUBLIC METHODS
// =============================================================================

// GetUserRoles fetches user roles from Logto Management API
func (c *LogtoManagementClient) GetUserRoles(userID string) ([]models.LogtoRole, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/users/%s/roles", userID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user roles: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user roles, status %d: %s", resp.StatusCode, string(body))
	}

	var roles []models.LogtoRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode user roles: %w", err)
	}

	return roles, nil
}

// GetRoleScopes fetches scopes for a role
func (c *LogtoManagementClient) GetRoleScopes(roleID string) ([]models.LogtoScope, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/roles/%s/scopes", roleID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch role scopes: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch role scopes, status %d: %s", resp.StatusCode, string(body))
	}

	var scopes []models.LogtoScope
	if err := json.NewDecoder(resp.Body).Decode(&scopes); err != nil {
		return nil, fmt.Errorf("failed to decode role scopes: %w", err)
	}

	return scopes, nil
}

// GetUserOrganizationRoles fetches user's roles in an organization
func (c *LogtoManagementClient) GetUserOrganizationRoles(orgID, userID string) ([]models.LogtoOrganizationRole, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organizations/%s/users/%s/roles", orgID, userID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user organization roles: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user organization roles, status %d: %s", resp.StatusCode, string(body))
	}

	var roles []models.LogtoOrganizationRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode user organization roles: %w", err)
	}

	return roles, nil
}

// GetOrganizationRoleScopes fetches scopes for an organization role
func (c *LogtoManagementClient) GetOrganizationRoleScopes(roleID string) ([]models.LogtoScope, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organization-roles/%s/scopes", roleID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization role scopes: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organization role scopes, status %d: %s", resp.StatusCode, string(body))
	}

	var scopes []models.LogtoScope
	if err := json.NewDecoder(resp.Body).Decode(&scopes); err != nil {
		return nil, fmt.Errorf("failed to decode organization role scopes: %w", err)
	}

	return scopes, nil
}

// GetAllRoles fetches all roles from Logto Management API
func (c *LogtoManagementClient) GetAllRoles() ([]models.LogtoRole, error) {
	resp, err := c.makeRequest("GET", "/roles", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roles: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch roles, status %d: %s", resp.StatusCode, string(body))
	}

	var roles []models.LogtoRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode roles: %w", err)
	}

	return roles, nil
}

// GetRoleByName finds a role by name
func (c *LogtoManagementClient) GetRoleByName(roleName string) (*models.LogtoRole, error) {
	roles, err := c.GetAllRoles()
	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		if role.Name == roleName {
			return &role, nil
		}
	}

	return nil, fmt.Errorf("role '%s' not found", roleName)
}

// GetRoleByID finds a role by ID
func (c *LogtoManagementClient) GetRoleByID(roleID string) (*models.LogtoRole, error) {
	roles, err := c.GetAllRoles()
	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		if role.ID == roleID {
			return &role, nil
		}
	}

	return nil, fmt.Errorf("role with ID '%s' not found", roleID)
}

// GetAllOrganizationRoles fetches all organization roles from Logto Management API
func (c *LogtoManagementClient) GetAllOrganizationRoles() ([]models.LogtoOrganizationRole, error) {
	resp, err := c.makeRequest("GET", "/organization-roles", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization roles: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organization roles, status %d: %s", resp.StatusCode, string(body))
	}

	var roles []models.LogtoOrganizationRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode organization roles: %w", err)
	}

	return roles, nil
}

// GetOrganizationRoleByName finds an organization role by name
func (c *LogtoManagementClient) GetOrganizationRoleByName(roleName string) (*models.LogtoOrganizationRole, error) {
	roles, err := c.GetAllOrganizationRoles()
	if err != nil {
		return nil, err
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to assign user roles, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// RemoveUserRoles removes roles from a user
func (c *LogtoManagementClient) RemoveUserRoles(userID string, roleIDs []string) error {
	// Remove roles one by one as per Logto API specification
	for _, roleID := range roleIDs {
		resp, err := c.makeRequest("DELETE", fmt.Sprintf("/users/%s/roles/%s", userID, roleID), nil)
		if err != nil {
			return fmt.Errorf("failed to remove user role %s: %w", roleID, err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to remove user role %s, status %d: %s", roleID, resp.StatusCode, string(body))
		}
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to assign organization roles to user, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// EnrichUserWithRolesAndPermissions fetches complete roles and permissions from Logto Management API
// Optimized version with parallel API calls for improved performance
func EnrichUserWithRolesAndPermissions(userID string) (*models.User, error) {
	logger.ComponentLogger("logto").Debug().
		Str("operation", "enrich_user_start").
		Str("user_id", userID).
		Msg("Starting user enrichment")
	client := NewManagementClient()

	// Initialize user
	user := &models.User{
		ID:               userID,
		UserRoles:        []string{},
		UserRoleIDs:      []string{},
		UserPermissions:  []string{},
		OrgRole:          "",
		OrgRoleID:        "",
		OrgPermissions:   []string{},
		OrganizationID:   "",
		OrganizationName: "",
	}

	// Step 1: Parallel fetch of user roles and user organizations
	type userRolesResult struct {
		roles []models.LogtoRole
		err   error
	}
	type userOrgsResult struct {
		orgs []models.LogtoOrganization
		err  error
	}

	userRolesCh := make(chan userRolesResult, 1)
	userOrgsCh := make(chan userOrgsResult, 1)

	// Fetch user roles in parallel
	go func() {
		logger.ComponentLogger("logto").Debug().
			Str("operation", "fetch_user_roles").
			Str("user_id", userID).
			Msg("Fetching user roles")
		roles, err := client.GetUserRoles(userID)
		userRolesCh <- userRolesResult{roles: roles, err: err}
	}()

	// Fetch user organizations in parallel
	go func() {
		logger.ComponentLogger("logto").Debug().
			Str("operation", "fetch_user_orgs").
			Str("user_id", userID).
			Msg("Fetching user organizations")
		orgs, err := client.GetUserOrganizations(userID)
		userOrgsCh <- userOrgsResult{orgs: orgs, err: err}
	}()

	// Wait for both results
	userRolesRes := <-userRolesCh
	userOrgsRes := <-userOrgsCh

	// Process user roles
	var userRoles []models.LogtoRole
	if userRolesRes.err != nil {
		logger.ComponentLogger("logto").Warn().
			Err(userRolesRes.err).
			Str("operation", "fetch_user_roles").
			Str("user_id", userID).
			Msg("Failed to fetch user roles")
	} else {
		userRoles = userRolesRes.roles
		logger.ComponentLogger("logto").Debug().
			Int("role_count", len(userRoles)).
			Str("operation", "fetch_user_roles").
			Str("user_id", userID).
			Msg("Found user roles")
		// Extract role names and IDs
		for _, role := range userRoles {
			user.UserRoles = append(user.UserRoles, role.Name)
			user.UserRoleIDs = append(user.UserRoleIDs, role.ID)
			logger.ComponentLogger("logto").Debug().
				Str("operation", "extract_user_role").
				Str("role_name", role.Name).
				Str("role_id", role.ID).
				Msg("Extracted user role")
		}
	}

	// Process user organizations
	var orgs []models.LogtoOrganization
	if userOrgsRes.err != nil {
		logger.ComponentLogger("logto").Warn().
			Err(userOrgsRes.err).
			Str("operation", "fetch_user_orgs").
			Str("user_id", userID).
			Msg("Failed to fetch user organizations")
	} else {
		orgs = userOrgsRes.orgs
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
		}
	}

	// Step 2: Parallel fetch of role scopes and organization roles
	type roleScopesResult struct {
		roleID string
		scopes []models.LogtoScope
		err    error
	}
	type orgRolesResult struct {
		roles []models.LogtoOrganizationRole
		err   error
	}

	// Create channels for parallel processing
	roleScopesCh := make(chan roleScopesResult, len(userRoles))
	orgRolesCh := make(chan orgRolesResult, 1)

	// Fetch scopes for all user roles in parallel
	for _, role := range userRoles {
		go func(roleID string) {
			scopes, err := client.GetRoleScopes(roleID)
			roleScopesCh <- roleScopesResult{roleID: roleID, scopes: scopes, err: err}
		}(role.ID)
	}

	// Fetch organization roles in parallel (if we have organizations)
	var orgRolesWaitCount int
	if len(orgs) > 0 {
		orgRolesWaitCount = 1
		go func() {
			orgRoles, err := client.GetUserOrganizationRoles(orgs[0].ID, userID)
			orgRolesCh <- orgRolesResult{roles: orgRoles, err: err}
		}()
	}

	// Collect role scopes results
	for i := 0; i < len(userRoles); i++ {
		result := <-roleScopesCh
		if result.err != nil {
			logger.ComponentLogger("logto").Warn().
				Err(result.err).
				Str("operation", "fetch_role_scopes").
				Str("role_id", result.roleID).
				Msg("Failed to fetch role scopes")
			continue
		}
		for _, scope := range result.scopes {
			user.UserPermissions = append(user.UserPermissions, scope.Name)
		}
	}

	// Collect organization roles result (if fetched)
	var orgRoles []models.LogtoOrganizationRole
	if orgRolesWaitCount > 0 {
		orgRolesRes := <-orgRolesCh
		if orgRolesRes.err != nil {
			logger.ComponentLogger("logto").Warn().
				Err(orgRolesRes.err).
				Str("operation", "fetch_org_roles").
				Str("user_id", userID).
				Str("org_id", orgs[0].ID).
				Msg("Failed to fetch organization roles")
		} else {
			orgRoles = orgRolesRes.roles
		}
	}

	// Step 3: Process organization roles and fetch their scopes
	if len(orgRoles) > 0 {
		// Use first organization role as primary
		primaryOrgRole := orgRoles[0]
		user.OrgRole = primaryOrgRole.Name
		user.OrgRoleID = primaryOrgRole.ID
		logger.ComponentLogger("logto").Debug().
			Str("operation", "extract_org_role").
			Str("org_role_name", primaryOrgRole.Name).
			Str("org_role_id", primaryOrgRole.ID).
			Msg("Extracted organization role")

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

	// Remove duplicates from permissions
	user.UserPermissions = removeDuplicates(user.UserPermissions)
	user.OrgPermissions = removeDuplicates(user.OrgPermissions)

	// Filter out manage permissions for reader roles
	user.OrgPermissions = filterManagePermissionsForReader(user.OrgPermissions, user.UserRoles)

	logger.ComponentLogger("logto").Info().
		Str("operation", "enrich_user_complete").
		Str("user_id", userID).
		Int("user_roles_count", len(user.UserRoles)).
		Strs("user_role_ids", user.UserRoleIDs).
		Int("user_permissions_count", len(user.UserPermissions)).
		Str("org_role", user.OrgRole).
		Str("org_role_id", user.OrgRoleID).
		Int("org_permissions_count", len(user.OrgPermissions)).
		Msg("User enrichment completed")

	return user, nil
}

// =============================================================================
// PRIVATE METHODS
// =============================================================================

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

// filterManagePermissionsForReader removes manage permissions for users with reader role
func filterManagePermissionsForReader(permissions []string, userRoles []string) []string {
	// Check if user has reader role (case-insensitive)
	hasReaderRole := false
	for _, role := range userRoles {
		if role == "reader" || role == "Reader" {
			hasReaderRole = true
			break
		}
	}

	// If user doesn't have reader role, return permissions unchanged
	if !hasReaderRole {
		return permissions
	}

	// Filter out manage permissions for distributors, resellers, customers
	var filteredPermissions []string
	for _, permission := range permissions {
		// Skip manage permissions for distributors, resellers, and customers
		if permission == "manage:distributors" ||
			permission == "manage:resellers" ||
			permission == "manage:customers" {
			continue
		}
		filteredPermissions = append(filteredPermissions, permission)
	}

	return filteredPermissions
}
