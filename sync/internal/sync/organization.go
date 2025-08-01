/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package sync

import (
	"fmt"
	"strings"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/config"
	"github.com/nethesis/my/sync/internal/logger"
)

// isSystemOrganizationRole checks if an organization role is a system role that shouldn't be deleted
func isSystemOrganizationRole(role client.LogtoOrganizationRole) bool {
	// Preserve Logto system roles and any role that looks like a system role
	systemRoleNames := []string{
		"logto",
		"admin",
		"system",
		"default",
		"owner",
		"member",
	}

	roleName := strings.ToLower(role.Name)
	for _, systemName := range systemRoleNames {
		if strings.Contains(roleName, systemName) {
			return true
		}
	}

	// Preserve roles with system-like descriptions
	description := strings.ToLower(role.Description)
	if strings.Contains(description, "system") ||
		strings.Contains(description, "default") ||
		strings.Contains(description, "logto") {
		return true
	}

	return false
}

// isSystemOrganizationScope checks if an organization scope is a system scope that shouldn't be deleted
func isSystemOrganizationScope(scope client.LogtoOrganizationScope) bool {
	// Preserve Logto system scopes and any scope that looks like a system scope
	systemScopeNames := []string{
		"logto",
		"system",
		"default",
		"management",
		"api",
	}

	scopeName := strings.ToLower(scope.Name)
	for _, systemName := range systemScopeNames {
		if strings.Contains(scopeName, systemName) {
			return true
		}
	}

	// Preserve scopes with system-like descriptions but be more specific
	description := strings.ToLower(scope.Description)
	if strings.Contains(description, "logto") ||
		strings.Contains(description, "management") ||
		(strings.Contains(description, "system") && !strings.HasPrefix(description, "organization scope:")) {
		return true
	}

	return false
}

// syncOrganizationScopes synchronizes organization scopes
func (e *Engine) syncOrganizationScopes(cfg *config.Config, result *Result) error {
	logger.Info("Syncing organization scopes...")

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

	// Create map for quick lookup of config scopes BY NAME
	configScopeMap := make(map[string]bool)
	for permissionID := range allPermissions {
		configScopeMap[permissionID] = true
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
				if e.options.DryRun {
					logger.Info("DRY RUN: Would update organization scope: %s", scopeName)
					e.addOperation(result, "organization-scope", "update", scopeName, "Would update organization scope", nil)
					result.Summary.ScopesUpdated++
				} else {
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
			}
		} else {
			// Create new organization scope
			if e.options.DryRun {
				logger.Info("DRY RUN: Would create organization scope: %s", scopeName)
				e.addOperation(result, "organization-scope", "create", scopeName, "Would create organization scope", nil)
				result.Summary.ScopesCreated++
			} else {
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
	}

	// Cleanup phase: remove organization scopes not in config (only if --cleanup flag is set)
	if e.options.Cleanup {
		for _, existingScope := range existingScopes {
			// Skip scopes that are defined in config
			if configScopeMap[existingScope.Name] {
				logger.Debug("Skipping organization scope defined in config: %s", existingScope.Name)
				continue
			}
			// Skip system/default scopes that shouldn't be deleted
			if isSystemOrganizationScope(existingScope) {
				logger.Debug("Skipping system organization scope: %s (description: %s)", existingScope.Name, existingScope.Description)
				continue
			}
			// Delete scope not in config
			if e.options.DryRun {
				logger.Info("DRY RUN: Would remove organization scope not in config: %s", existingScope.Name)
				e.addOperation(result, "organization-scope", "delete", existingScope.Name, "Would remove organization scope not in config", nil)
				result.Summary.ScopesDeleted++
			} else {
				logger.Info("Removing organization scope not in config: %s", existingScope.Name)
				err := e.client.DeleteOrganizationScope(existingScope.ID)
				e.addOperation(result, "organization-scope", "delete", existingScope.Name, "Removed organization scope not in config", err)
				if err != nil {
					logger.Warn("Failed to delete organization scope %s: %v", existingScope.Name, err)
				} else {
					result.Summary.ScopesDeleted++
				}
			}
		}
	}

	logger.Info("Organization scopes sync completed")
	return nil
}

// syncOrganizationRoles synchronizes organization roles
func (e *Engine) syncOrganizationRoles(cfg *config.Config, result *Result) error {
	logger.Info("Syncing organization roles...")

	// Filter only org type roles
	orgRoles := cfg.GetOrgTypeRoles(cfg.Hierarchy.OrganizationRoles)

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

	// Create map for quick lookup of config roles BY NAME (case-insensitive)
	configRoleMap := make(map[string]bool)
	for _, configRole := range orgRoles {
		configRoleMap[strings.ToLower(configRole.Name)] = true
	}

	// Create or update organization roles
	for _, configRole := range orgRoles {
		roleName := configRole.Name
		roleNameLower := strings.ToLower(roleName)

		if existingRole, exists := existingRoleMap[roleNameLower]; exists {
			// Update existing role if needed
			newDescription := fmt.Sprintf("Organization role: %s", roleName)
			if existingRole.Description != newDescription {
				if e.options.DryRun {
					logger.Info("DRY RUN: Would update organization role: %s", roleName)
					e.addOperation(result, "organization-role", "update", roleName, "Would update organization role", nil)
					result.Summary.RolesUpdated++
				} else {
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
			}
		} else {
			// Create new organization role
			if e.options.DryRun {
				logger.Info("DRY RUN: Would create organization role: %s", roleName)
				e.addOperation(result, "organization-role", "create", roleName, "Would create organization role", nil)
				result.Summary.RolesCreated++
			} else {
				logger.Info("Creating organization role: %s", roleName)
				newRole := client.LogtoOrganizationRole{
					Name:        roleName,
					Description: fmt.Sprintf("Organization role: %s", roleName),
				}

				err := e.client.CreateOrganizationRole(newRole)
				e.addOperation(result, "organization-role", "create", roleName, "Created organization role", err)
				if err != nil {
					return fmt.Errorf("failed to create organization role %s: %w", roleName, err)
				}
				result.Summary.RolesCreated++
			}
		}
	}

	// Cleanup phase: remove organization roles not in config (only if --cleanup flag is set)
	if e.options.Cleanup {
		for _, existingRole := range existingRoles {
			roleNameLower := strings.ToLower(existingRole.Name)
			// Skip roles that are defined in config
			if configRoleMap[roleNameLower] {
				continue
			}
			// Skip system/default roles that shouldn't be deleted
			if isSystemOrganizationRole(existingRole) {
				continue
			}
			// Delete role not in config
			if e.options.DryRun {
				logger.Info("DRY RUN: Would remove organization role not in config: %s", existingRole.Name)
				e.addOperation(result, "organization-role", "delete", existingRole.Name, "Would remove organization role not in config", nil)
				result.Summary.RolesDeleted++
			} else {
				logger.Info("Removing organization role not in config: %s", existingRole.Name)
				err := e.client.DeleteOrganizationRole(existingRole.ID)
				e.addOperation(result, "organization-role", "delete", existingRole.Name, "Removed organization role not in config", err)
				if err != nil {
					logger.Warn("Failed to delete organization role %s: %v", existingRole.Name, err)
				} else {
					result.Summary.RolesDeleted++
				}
			}
		}
	}

	logger.Info("Organization roles sync completed")
	return nil
}

// syncOrganizationRoleScopes synchronizes organization role scope assignments
func (e *Engine) syncOrganizationRoleScopes(cfg *config.Config, result *Result) error {
	logger.Info("Syncing organization role scopes...")

	// Filter only org type roles
	orgRoles := cfg.GetOrgTypeRoles(cfg.Hierarchy.OrganizationRoles)

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

	for _, configRole := range orgRoles {
		// Find the actual role ID by name
		roleID, exists := roleNameToID[strings.ToLower(configRole.Name)]
		if !exists {
			if e.options.DryRun {
				// In dry-run mode, simulate with a dummy ID for roles being created
				roleID = "dry-run-role-" + strings.ToLower(configRole.Name)
			} else {
				logger.Warn("Organization role %s not found, skipping scope assignment", configRole.Name)
				continue
			}
		}

		// Get current scopes for this role
		currentScopes, err := e.client.GetOrganizationRoleScopes(roleID)
		if err != nil {
			if e.options.DryRun && strings.Contains(roleID, "dry-run-role-") {
				// In dry-run mode, for simulated roles, assume no current scopes
				currentScopes = []client.LogtoOrganizationScope{}
			} else {
				return fmt.Errorf("failed to get scopes for organization role %s: %w", configRole.Name, err)
			}
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
				} else if e.options.DryRun {
					// In dry-run mode, simulate scope ID for scopes being created
					simulatedScopeID := "dry-run-scope-" + permission.ID
					expectedScopeMap[simulatedScopeID] = true
					// Also add to scopeNameToID mapping for reverse lookup
					scopeNameToID[permission.ID] = simulatedScopeID
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

				if e.options.DryRun {
					logger.Info("DRY RUN: Would assign scope %s to organization role %s", scopeName, configRole.Name)
					e.addOperation(result, "organization-role-scope", "assign",
						fmt.Sprintf("%s->%s", configRole.Name, scopeName),
						"Would assign scope to organization role", nil)
					result.Summary.PermissionsCreated++
				} else {
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
					if e.options.DryRun {
						logger.Info("DRY RUN: Would remove scope %s from organization role %s", scopeName, configRole.Name)
						e.addOperation(result, "organization-role-scope", "remove",
							fmt.Sprintf("%s->%s", configRole.Name, scopeName),
							"Would remove scope from organization role", nil)
						result.Summary.PermissionsDeleted++
					} else {
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
	}

	logger.Info("Organization role scopes sync completed")
	return nil
}
