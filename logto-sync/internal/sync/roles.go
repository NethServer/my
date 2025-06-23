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

// isSystemUserRole checks if a role is a system role that shouldn't be deleted
func isSystemUserRole(role client.LogtoRole) bool {
	// Preserve Logto system roles and any role that looks like a system role
	systemRoleNames := []string{
		"logto",
		"admin",
		"machine-to-machine",
		"system",
		"default",
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

// syncUserRoles synchronizes user roles
func (e *Engine) syncUserRoles(cfg *config.Config, result *Result) error {
	logger.Info("Syncing user roles...")

	if e.options.DryRun {
		logger.Info("DRY RUN: Would sync user roles")
		return nil
	}

	// Filter only user type roles
	userRoles := cfg.GetUserTypeRoles(cfg.Hierarchy.UserRoles)

	// Get existing user roles from Logto
	existingRoles, err := e.client.GetRoles()
	if err != nil {
		return fmt.Errorf("failed to get existing user roles: %w", err)
	}

	// Create map for quick lookup BY NAME (case-insensitive)
	existingRoleMap := make(map[string]client.LogtoRole)
	for _, role := range existingRoles {
		existingRoleMap[strings.ToLower(role.Name)] = role
	}

	// Create map for quick lookup of config roles BY NAME (case-insensitive)
	configRoleMap := make(map[string]bool)
	for _, configRole := range userRoles {
		configRoleMap[strings.ToLower(configRole.Name)] = true
	}

	// Create or update user roles
	for _, configRole := range userRoles {
		roleName := configRole.Name
		roleNameLower := strings.ToLower(roleName)

		if existingRole, exists := existingRoleMap[roleNameLower]; exists {
			// Update existing role if needed
			newDescription := fmt.Sprintf("User role (Priority: %d)", configRole.Priority)
			if existingRole.Description != newDescription {
				logger.Info("Updating user role: %s", roleName)
				updateRole := client.LogtoRole{
					ID:          existingRole.ID,
					Name:        existingRole.Name,
					Description: newDescription,
				}

				err := e.client.UpdateRole(existingRole.ID, updateRole)
				e.addOperation(result, "user-role", "update", roleName, "Updated user role", err)
				if err != nil {
					return fmt.Errorf("failed to update user role %s: %w", roleName, err)
				}
				result.Summary.RolesUpdated++
			}
		} else {
			// Create new user role
			logger.Info("Creating user role: %s", roleName)
			newRole := client.LogtoRole{
				Name:        roleName,
				Description: fmt.Sprintf("User role (Priority: %d)", configRole.Priority),
			}

			err := e.client.CreateRole(newRole)
			e.addOperation(result, "user-role", "create", roleName, "Created user role", err)
			if err != nil {
				return fmt.Errorf("failed to create user role %s: %w", roleName, err)
			}
			result.Summary.RolesCreated++
		}
	}

	// Cleanup phase: remove user roles not in config (only if --cleanup flag is set)
	if e.options.Cleanup {
		for _, existingRole := range existingRoles {
			roleNameLower := strings.ToLower(existingRole.Name)
			// Skip roles that are defined in config
			if configRoleMap[roleNameLower] {
				continue
			}
			// Skip system/default roles that shouldn't be deleted
			if isSystemUserRole(existingRole) {
				continue
			}
			// Delete role not in config
			logger.Info("Removing user role not in config: %s", existingRole.Name)
			err := e.client.DeleteRole(existingRole.ID)
			e.addOperation(result, "user-role", "delete", existingRole.Name, "Removed user role not in config", err)
			if err != nil {
				logger.Warn("Failed to delete user role %s: %v", existingRole.Name, err)
			} else {
				result.Summary.RolesDeleted++
			}
		}
	}

	logger.Info("User roles sync completed")
	return nil
}

// syncUserRolePermissions synchronizes user role permissions
func (e *Engine) syncUserRolePermissions(cfg *config.Config, result *Result) error {
	logger.Info("Syncing user role permissions...")

	if e.options.DryRun {
		logger.Info("DRY RUN: Would sync user role permissions")
		return nil
	}

	// Filter only user type roles
	userRoles := cfg.GetUserTypeRoles(cfg.Hierarchy.UserRoles)

	// Get existing roles to create name->ID mapping
	existingRoles, err := e.client.GetRoles()
	if err != nil {
		return fmt.Errorf("failed to get existing user roles: %w", err)
	}

	roleNameToID := make(map[string]string)
	for _, role := range existingRoles {
		roleNameToID[strings.ToLower(role.Name)] = role.ID
	}

	// Get existing resources to create name->ID mapping
	existingResources, err := e.client.GetResources()
	if err != nil {
		return fmt.Errorf("failed to get existing resources: %w", err)
	}

	resourceNameToID := make(map[string]string)
	for _, resource := range existingResources {
		resourceNameToID[resource.Name] = resource.ID
	}

	// Build global scope mapping from all resources
	allScopeNameToID := make(map[string]string)
	for _, configResource := range cfg.Hierarchy.Resources {
		resourceID, exists := resourceNameToID[configResource.Name]
		if !exists {
			logger.Warn("Resource %s not found, skipping scope mappings", configResource.Name)
			continue
		}

		scopes, err := e.client.GetScopes(resourceID)
		if err != nil {
			return fmt.Errorf("failed to get scopes for resource %s: %w", configResource.Name, err)
		}

		for _, scope := range scopes {
			allScopeNameToID[scope.Name] = scope.ID
		}
	}

	// Process each user role
	for _, configRole := range userRoles {
		roleID, exists := roleNameToID[strings.ToLower(configRole.Name)]
		if !exists {
			logger.Warn("User role %s not found, skipping permission assignment", configRole.Name)
			continue
		}

		// Get current permissions (scopes) for this role
		currentPermissions, err := e.client.GetRolePermissions(roleID)
		if err != nil {
			return fmt.Errorf("failed to get permissions for role %s: %w", configRole.Name, err)
		}

		// Create map of current permission IDs
		currentPermissionMap := make(map[string]bool)
		for _, permission := range currentPermissions {
			currentPermissionMap[permission.ID] = true
		}

		// Build expected permission IDs from config
		expectedPermissionIDs := []string{}
		expectedPermissionMap := make(map[string]bool)

		for _, permission := range configRole.Permissions {
			if permission.ID != "" {
				if scopeID, exists := allScopeNameToID[permission.ID]; exists {
					expectedPermissionIDs = append(expectedPermissionIDs, scopeID)
					expectedPermissionMap[scopeID] = true
				} else {
					logger.Warn("Permission/scope %s not found for role %s", permission.ID, configRole.Name)
				}
			}
		}

		// Find permissions to add and remove
		permissionsToAdd := []string{}
		permissionsToRemove := []string{}

		for scopeID := range expectedPermissionMap {
			if !currentPermissionMap[scopeID] {
				permissionsToAdd = append(permissionsToAdd, scopeID)
			}
		}

		for scopeID := range currentPermissionMap {
			if !expectedPermissionMap[scopeID] {
				// Only remove permissions we manage (not system ones)
				scopeName := ""
				for name, id := range allScopeNameToID {
					if id == scopeID {
						scopeName = name
						break
					}
				}
				// Remove only if it's not a system permission
				if scopeName != "" && !strings.Contains(strings.ToLower(scopeName), "management") {
					permissionsToRemove = append(permissionsToRemove, scopeID)
				}
			}
		}

		// Add missing permissions
		if len(permissionsToAdd) > 0 {
			logger.Info("Assigning %d permissions to role %s", len(permissionsToAdd), configRole.Name)
			err := e.client.AssignPermissionsToRole(roleID, permissionsToAdd)
			e.addOperation(result, "user-role-permission", "assign",
				fmt.Sprintf("%s (%d permissions)", configRole.Name, len(permissionsToAdd)),
				"Assigned permissions to user role", err)
			if err != nil {
				return fmt.Errorf("failed to assign permissions to role %s: %w", configRole.Name, err)
			}
			result.Summary.PermissionsCreated += len(permissionsToAdd)
		}

		// Remove unwanted permissions
		if len(permissionsToRemove) > 0 {
			logger.Info("Removing %d permissions from role %s", len(permissionsToRemove), configRole.Name)
			err := e.client.RemovePermissionsFromRole(roleID, permissionsToRemove)
			e.addOperation(result, "user-role-permission", "remove",
				fmt.Sprintf("%s (%d permissions)", configRole.Name, len(permissionsToRemove)),
				"Removed permissions from user role", err)
			if err != nil {
				logger.Warn("Failed to remove permissions from role %s: %v", configRole.Name, err)
			} else {
				result.Summary.PermissionsDeleted += len(permissionsToRemove)
			}
		}
	}

	logger.Info("User role permissions sync completed")
	return nil
}
