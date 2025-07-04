/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package initcmd

import (
	"fmt"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/constants"
	"github.com/nethesis/my/sync/internal/logger"
)

// SyncBasicConfiguration synchronizes basic RBAC configuration
func SyncBasicConfiguration(client *client.LogtoClient, ownerUsername string) error {
	logger.Info("Synchronizing basic RBAC configuration...")

	// Create essential roles for owner user to work immediately
	if err := createEssentialRoles(client); err != nil {
		return fmt.Errorf("failed to create essential roles: %w", err)
	}

	// Create Owner organization
	if err := createOwnerOrganization(client); err != nil {
		return fmt.Errorf("failed to create Owner organization: %w", err)
	}

	// Assign roles and organization to owner user
	if err := assignRolesToOwnerUser(client, ownerUsername); err != nil {
		return fmt.Errorf("failed to assign roles to owner user: %w", err)
	}

	logger.Info("Basic RBAC configuration synchronized - owner user ready")
	logger.Info("Run 'sync sync' later to create complete RBAC configuration")
	return nil
}

func createEssentialRoles(client *client.LogtoClient) error {
	logger.Info("Creating essential roles...")

	// Create organization scopes first (from config.yml)
	if err := createEssentialOrgScopes(client); err != nil {
		return fmt.Errorf("failed to create essential organization scopes: %w", err)
	}

	// Create organization role "owner" (from config.yml)
	if err := createOrgRoleIfNotExists(client, constants.OwnerRoleID, constants.OwnerRoleName, constants.OwnerOrgDescription); err != nil {
		return fmt.Errorf("failed to create owner organization role: %w", err)
	}

	// Create user role "admin" (from config.yml)
	if err := createUserRoleIfNotExists(client, constants.AdminRoleID, constants.AdminRoleName, "Admin user role - full technical control including dangerous operations"); err != nil {
		return fmt.Errorf("failed to create admin user role: %w", err)
	}

	// Assign scopes to owner organization role
	if err := assignScopesToOwnerRole(client); err != nil {
		return fmt.Errorf("failed to assign scopes to owner role: %w", err)
	}

	logger.Info("Essential roles created successfully")
	return nil
}

func createOrgRoleIfNotExists(client *client.LogtoClient, roleID, roleName, description string) error {
	// Check if organization role already exists
	orgRoles, err := client.GetOrganizationRoles()
	if err != nil {
		return fmt.Errorf("failed to get existing organization roles: %w", err)
	}

	for _, role := range orgRoles {
		if role.Name == roleName {
			logger.Info("Organization role '%s' already exists, skipping creation", roleName)
			return nil
		}
	}

	// Create the organization role using the simple method
	orgRoleMap := map[string]interface{}{
		"name":        roleName,
		"description": description,
	}

	err = client.CreateOrganizationRoleSimple(orgRoleMap)
	if err != nil {
		return fmt.Errorf("failed to create organization role '%s': %w", roleName, err)
	}

	logger.Info("Created organization role: %s", roleName)
	return nil
}

func createUserRoleIfNotExists(client *client.LogtoClient, roleID, roleName, description string) error {
	// Check if user role already exists
	userRoles, err := client.GetRoles()
	if err != nil {
		return fmt.Errorf("failed to get existing user roles: %w", err)
	}

	for _, role := range userRoles {
		if role.Name == roleName {
			logger.Info("User role '%s' already exists, skipping creation", roleName)
			return nil
		}
	}

	// Create the user role using the simple method
	roleData := map[string]interface{}{
		"name":        roleName,
		"description": description,
	}

	err = client.CreateRoleSimple(roleData)
	if err != nil {
		return fmt.Errorf("failed to create user role '%s': %w", roleName, err)
	}

	logger.Info("Created user role: %s", roleName)
	return nil
}

func createEssentialOrgScopes(client *client.LogtoClient) error {
	logger.Info("Creating essential organization scopes...")

	// Organization scopes from config.yml for owner role
	scopes := []struct {
		name        string
		description string
	}{
		{"create:distributors", "Create distributor organizations"},
		{"manage:distributors", "Manage distributor organizations"},
		{"create:resellers", "Create reseller organizations"},
		{"manage:resellers", "Manage reseller organizations"},
		{"create:customers", "Create customer organizations"},
		{"manage:customers", "Manage customer organizations"},
	}

	for _, scope := range scopes {
		if err := createOrgScopeIfNotExists(client, scope.name, scope.description); err != nil {
			return fmt.Errorf("failed to create scope %s: %w", scope.name, err)
		}
	}

	logger.Info("Essential organization scopes created successfully")
	return nil
}

func createOrgScopeIfNotExists(client *client.LogtoClient, scopeName, description string) error {
	// Check if organization scope already exists
	scopes, err := client.GetOrganizationScopes()
	if err != nil {
		return fmt.Errorf("failed to get existing organization scopes: %w", err)
	}

	for _, scope := range scopes {
		if scope.Name == scopeName {
			logger.Info("Organization scope '%s' already exists, skipping creation", scopeName)
			return nil
		}
	}

	// Create the organization scope using the simple method
	scopeData := map[string]interface{}{
		"name":        scopeName,
		"description": description,
	}

	err = client.CreateOrganizationScopeSimple(scopeData)
	if err != nil {
		return fmt.Errorf("failed to create organization scope '%s': %w", scopeName, err)
	}

	logger.Info("Created organization scope: %s", scopeName)
	return nil
}

func assignScopesToOwnerRole(client *client.LogtoClient) error {
	logger.Info("Assigning scopes to Owner organization role...")

	// Get Owner organization role ID
	orgRoles, err := client.GetOrganizationRoles()
	if err != nil {
		return fmt.Errorf("failed to get organization roles: %w", err)
	}

	var ownerRoleID string
	for _, role := range orgRoles {
		if role.Name == "Owner" {
			ownerRoleID = role.ID
			break
		}
	}

	if ownerRoleID == "" {
		return fmt.Errorf("owner organization role not found")
	}

	// Get organization scopes to assign
	orgScopes, err := client.GetOrganizationScopes()
	if err != nil {
		return fmt.Errorf("failed to get organization scopes: %w", err)
	}

	// Assign all owner-related scopes
	ownerScopeNames := []string{"create:distributors", "manage:distributors", "create:resellers", "manage:resellers", "create:customers", "manage:customers"}

	for _, scopeName := range ownerScopeNames {
		var scopeID string
		for _, scope := range orgScopes {
			if scope.Name == scopeName {
				scopeID = scope.ID
				break
			}
		}

		if scopeID != "" {
			if err := client.AssignScopeToOrganizationRole(ownerRoleID, scopeID); err != nil {
				logger.Warn("Failed to assign scope %s to Owner role (may already be assigned): %v", scopeName, err)
			} else {
				logger.Info("Assigned scope %s to Owner organization role", scopeName)
			}
		}
	}

	logger.Info("Scope assignment to Owner role completed")
	return nil
}

func createOwnerOrganization(client *client.LogtoClient) error {
	logger.Info("Creating Owner organization...")

	// Check if Owner organization already exists
	organizations, err := client.GetOrganizations()
	if err != nil {
		return fmt.Errorf("failed to get existing organizations: %w", err)
	}

	var ownerOrgID string
	var organizationExists bool
	for _, org := range organizations {
		if org.Name == "Owner" {
			ownerOrgID = org.ID
			organizationExists = true
			logger.Info("Organization 'Owner' already exists, configuring default role")
			break
		}
	}

	// Create organization if it doesn't exist
	if !organizationExists {
		// Create the Owner organization using the simple method
		ownerOrgMap := map[string]interface{}{
			"name":        "Owner",
			"description": "Owner organization - complete control over commercial hierarchy",
		}

		err = client.CreateOrganizationSimple(ownerOrgMap)
		if err != nil {
			return fmt.Errorf("failed to create Owner organization: %w", err)
		}

		// Get the created organization ID
		orgs, err := client.GetOrganizations()
		if err != nil {
			return fmt.Errorf("failed to get organizations after creation: %w", err)
		}

		for _, org := range orgs {
			if org.Name == "Owner" {
				ownerOrgID = org.ID
				break
			}
		}

		logger.Info("Created Owner organization")
	}

	// Set JIT organization role for the Owner organization
	if err := setOwnerOrganizationDefaultRole(client, ownerOrgID); err != nil {
		return fmt.Errorf("failed to set JIT role for Owner organization: %w", err)
	}

	return nil
}

func setOwnerOrganizationDefaultRole(client *client.LogtoClient, ownerOrgID string) error {
	logger.Info("Setting JIT organization role for Owner organization...")

	// Get Owner organization role ID
	orgRoles, err := client.GetOrganizationRoles()
	if err != nil {
		return fmt.Errorf("failed to get organization roles: %w", err)
	}

	var ownerRoleID string
	for _, role := range orgRoles {
		if role.Name == "Owner" {
			ownerRoleID = role.ID
			break
		}
	}

	if ownerRoleID == "" {
		return fmt.Errorf("owner organization role not found")
	}

	// Set the JIT organization role (Just-in-Time provisioning)
	if err := client.SetOrganizationJITRole(ownerOrgID, ownerRoleID); err != nil {
		logger.Warn("Failed to set JIT organization role (may already be set): %v", err)
	} else {
		logger.Info("Set Owner as JIT organization role for Owner organization")
	}

	return nil
}

func assignRolesToOwnerUser(client *client.LogtoClient, ownerUsername string) error {
	logger.Info("Assigning roles and organization to owner user...")

	// Get owner user ID
	users, err := client.GetUsers()
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	ownerUserID, found := client.FindEntityID(users, "username", ownerUsername)
	if !found {
		return fmt.Errorf("owner user not found")
	}

	// Get Owner organization ID
	organizations, err := client.GetOrganizations()
	if err != nil {
		return fmt.Errorf("failed to get organizations: %w", err)
	}

	var ownerOrgID string
	for _, org := range organizations {
		if org.Name == "Owner" {
			ownerOrgID = org.ID
			break
		}
	}

	if ownerOrgID == "" {
		return fmt.Errorf("owner organization not found")
	}

	// Get user roles to assign (admin)
	userRoles, err := client.GetRoles()
	if err != nil {
		return fmt.Errorf("failed to get user roles: %w", err)
	}

	var adminRoleID string
	for _, role := range userRoles {
		if role.Name == "Admin" {
			adminRoleID = role.ID
			break
		}
	}

	// Get organization roles to assign (owner)
	orgRoles, err := client.GetOrganizationRoles()
	if err != nil {
		return fmt.Errorf("failed to get organization roles: %w", err)
	}

	var ownerOrgRoleID string
	for _, role := range orgRoles {
		if role.Name == "Owner" {
			ownerOrgRoleID = role.ID
			break
		}
	}

	// Assign Admin user role
	if adminRoleID != "" {
		if err := client.AssignRoleToUser(ownerUserID, adminRoleID); err != nil {
			logger.Warn("Failed to assign Admin user role (may already be assigned): %v", err)
		} else {
			logger.Info("Assigned Admin user role to owner user")
		}
	}

	// Add user to Owner organization
	if err := client.AddUserToOrganization(ownerOrgID, ownerUserID); err != nil {
		logger.Warn("Failed to add user to Owner organization (may already be member): %v", err)
	} else {
		logger.Info("Added owner user to Owner organization")
	}

	// Assign Owner organization role to user in organization
	if ownerOrgRoleID != "" {
		logger.Info("Attempting to assign Owner organization role (ID: %s) to user (ID: %s) in organization (ID: %s)", ownerOrgRoleID, ownerUserID, ownerOrgID)
		if err := client.AssignOrganizationRoleToUser(ownerOrgID, ownerUserID, ownerOrgRoleID); err != nil {
			logger.Warn("Failed to assign Owner organization role (may already be assigned): %v", err)
		} else {
			logger.Info("Assigned Owner organization role to owner user")
		}
	} else {
		logger.Warn("Owner organization role ID not found - unable to assign role")
	}

	logger.Info("Role and organization assignment completed")
	return nil
}
