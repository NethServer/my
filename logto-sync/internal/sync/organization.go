/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package sync

import (
	"fmt"
	"strings"

	"github.com/nethesis/my/logto-sync/internal/client"
	"github.com/nethesis/my/logto-sync/internal/config"
	"github.com/nethesis/my/logto-sync/internal/logger"
)

// syncOrganizationScopes synchronizes organization scopes
func (e *Engine) syncOrganizationScopes(cfg *config.Config, result *Result) error {
	logger.Info("Syncing organization scopes...")

	if e.options.DryRun {
		logger.Info("DRY RUN: Would sync organization scopes")
		return nil
	}

	// Get all unique permissions from user roles only
	allPermissions := cfg.GetAllPermissions()

	// Get existing organization scopes from Logto
	existingScopes, err := e.client.GetOrganizationScopes()
	if err != nil {
		return fmt.Errorf("failed to get existing organization scopes: %w", err)
	}

	// Create map for quick lookup BY NAME
	existingScopeMap := make(map[string]client.LogtoOrganizationScope)
	for _, scope := range existingScopes {
		existingScopeMap[scope.Name] = scope
	}

	// Create or update organization scopes
	for permissionID, configPermission := range allPermissions {
		scopeName := permissionID
		name := configPermission.Name
		if name == "" {
			name = permissionID
		}

		if existingScope, exists := existingScopeMap[scopeName]; exists {
			// Update existing scope if needed
			newDescription := fmt.Sprintf("Organization scope: %s", name)
			if existingScope.Description != newDescription {
				logger.Info("Updating organization scope: %s", scopeName)
				updateScope := client.LogtoOrganizationScope{
					ID:          existingScope.ID,
					Name:        scopeName,
					Description: newDescription,
				}

				err := e.client.UpdateOrganizationScope(existingScope.ID, updateScope)
				e.addOperation(result, "organization-scope", "update", scopeName, "Updated organization scope", err)
				if err != nil {
					return fmt.Errorf("failed to update organization scope %s: %w", scopeName, err)
				}
				result.Summary.ScopesUpdated++
			}
		} else {
			// Create new organization scope
			logger.Info("Creating organization scope: %s", scopeName)
			newScope := client.LogtoOrganizationScope{
				Name:        scopeName,
				Description: fmt.Sprintf("Organization scope: %s", name),
			}

			err := e.client.CreateOrganizationScope(newScope)
			e.addOperation(result, "organization-scope", "create", scopeName, "Created organization scope", err)
			if err != nil {
				return fmt.Errorf("failed to create organization scope %s: %w", scopeName, err)
			}
			result.Summary.ScopesCreated++
		}
	}

	logger.Info("Organization scopes sync completed")
	return nil
}

// syncOrganizationRoles synchronizes organization roles
func (e *Engine) syncOrganizationRoles(cfg *config.Config, result *Result) error {
	logger.Info("Syncing organization roles...")

	if e.options.DryRun {
		logger.Info("DRY RUN: Would sync organization roles")
		return nil
	}

	// Filter only user type roles
	userRoles := cfg.GetUserTypeRoles(cfg.Hierarchy.OrganizationRoles)

	// Get existing organization roles from Logto
	existingRoles, err := e.client.GetOrganizationRoles()
	if err != nil {
		return fmt.Errorf("failed to get existing organization roles: %w", err)
	}

	// Create map for quick lookup BY NAME (case-insensitive)
	existingRoleMap := make(map[string]client.LogtoOrganizationRole)
	for _, role := range existingRoles {
		existingRoleMap[strings.ToLower(role.Name)] = role
	}

	// Create or update organization roles
	for _, configRole := range userRoles {
		roleName := configRole.Name
		roleNameLower := strings.ToLower(roleName)

		if existingRole, exists := existingRoleMap[roleNameLower]; exists {
			// Update existing role if needed
			newDescription := fmt.Sprintf("Organization role (Priority: %d)", configRole.Priority)
			if existingRole.Description != newDescription {
				logger.Info("Updating organization role: %s", roleName)
				updateRole := client.LogtoOrganizationRole{
					ID:          existingRole.ID,
					Name:        existingRole.Name,
					Description: newDescription,
				}

				err := e.client.UpdateOrganizationRole(existingRole.ID, updateRole)
				e.addOperation(result, "organization-role", "update", roleName, "Updated organization role", err)
				if err != nil {
					return fmt.Errorf("failed to update organization role %s: %w", roleName, err)
				}
				result.Summary.RolesUpdated++
			}
		} else {
			// Create new organization role
			logger.Info("Creating organization role: %s", roleName)
			newRole := client.LogtoOrganizationRole{
				Name:        roleName,
				Description: fmt.Sprintf("Organization role (Priority: %d)", configRole.Priority),
			}

			err := e.client.CreateOrganizationRole(newRole)
			e.addOperation(result, "organization-role", "create", roleName, "Created organization role", err)
			if err != nil {
				return fmt.Errorf("failed to create organization role %s: %w", roleName, err)
			}
			result.Summary.RolesCreated++
		}
	}

	logger.Info("Organization roles sync completed")
	return nil
}

// syncOrganizationRoleScopes synchronizes organization role scope assignments
func (e *Engine) syncOrganizationRoleScopes(cfg *config.Config, result *Result) error {
	logger.Info("Syncing organization role scopes...")

	if e.options.DryRun {
		logger.Info("DRY RUN: Would sync organization role scopes")
		return nil
	}

	// Filter only user type roles
	userRoles := cfg.GetUserTypeRoles(cfg.Hierarchy.OrganizationRoles)

	// Get all existing scopes to create name->ID mapping
	existingScopes, err := e.client.GetOrganizationScopes()
	if err != nil {
		return fmt.Errorf("failed to get existing organization scopes: %w", err)
	}

	scopeNameToID := make(map[string]string)
	for _, scope := range existingScopes {
		scopeNameToID[scope.Name] = scope.ID
	}

	// Get existing roles to create name->ID mapping
	existingRoles, err := e.client.GetOrganizationRoles()
	if err != nil {
		return fmt.Errorf("failed to get existing organization roles: %w", err)
	}

	roleNameToID := make(map[string]string)
	for _, role := range existingRoles {
		roleNameToID[strings.ToLower(role.Name)] = role.ID
	}

	for _, configRole := range userRoles {
		// Find the actual role ID by name
		roleID, exists := roleNameToID[strings.ToLower(configRole.Name)]
		if !exists {
			logger.Warn("Organization role %s not found, skipping scope assignment", configRole.Name)
			continue
		}

		// Get current scopes for this role
		currentScopes, err := e.client.GetOrganizationRoleScopes(roleID)
		if err != nil {
			return fmt.Errorf("failed to get scopes for organization role %s: %w", configRole.Name, err)
		}

		// Create map of current scope IDs
		currentScopeMap := make(map[string]bool)
		for _, scope := range currentScopes {
			currentScopeMap[scope.ID] = true
		}

		// Create map of expected scope IDs
		expectedScopeMap := make(map[string]bool)
		for _, permission := range configRole.Permissions {
			if permission.ID != "" {
				if scopeID, exists := scopeNameToID[permission.ID]; exists {
					expectedScopeMap[scopeID] = true
				} else {
					logger.Warn("Scope %s not found, skipping assignment to role %s", permission.ID, configRole.Name)
				}
			}
		}

		// Add missing scopes
		for scopeID := range expectedScopeMap {
			if !currentScopeMap[scopeID] {
				scopeName := ""
				for name, id := range scopeNameToID {
					if id == scopeID {
						scopeName = name
						break
					}
				}

				logger.Info("Assigning scope %s to organization role %s", scopeName, configRole.Name)
				err := e.client.AssignScopeToOrganizationRole(roleID, scopeID)
				e.addOperation(result, "organization-role-scope", "assign",
					fmt.Sprintf("%s->%s", configRole.Name, scopeName),
					"Assigned scope to organization role", err)
				if err != nil {
					return fmt.Errorf("failed to assign scope %s to organization role %s: %w", scopeID, configRole.Name, err)
				}
				result.Summary.PermissionsCreated++
			}
		}

		// Remove scopes that are no longer needed (only managed ones)
		for scopeID := range currentScopeMap {
			if !expectedScopeMap[scopeID] {
				// Check if this scope is one we manage
				scopeName := ""
				for name, id := range scopeNameToID {
					if id == scopeID {
						scopeName = name
						break
					}
				}

				// Only remove if it's a scope we created (not system scopes)
				if scopeName != "" && !strings.Contains(strings.ToLower(scopeName), "management") {
					logger.Info("Removing scope %s from organization role %s", scopeName, configRole.Name)
					err := e.client.RemoveScopeFromOrganizationRole(roleID, scopeID)
					e.addOperation(result, "organization-role-scope", "remove",
						fmt.Sprintf("%s->%s", configRole.Name, scopeName),
						"Removed scope from organization role", err)
					if err != nil {
						logger.Warn("Failed to remove scope %s from organization role %s: %v", scopeID, configRole.Name, err)
					} else {
						result.Summary.PermissionsDeleted++
					}
				}
			}
		}
	}

	logger.Info("Organization role scopes sync completed")
	return nil
}
