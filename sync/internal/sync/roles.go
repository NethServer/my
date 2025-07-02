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
				if e.options.DryRun {
					logger.Info("DRY RUN: Would update user role: %s", roleName)
					e.addOperation(result, "user-role", "update", roleName, "Would update user role", nil)
					result.Summary.RolesUpdated++
				} else {
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
			}
		} else {
			// Create new user role
			if e.options.DryRun {
				logger.Info("DRY RUN: Would create user role: %s", roleName)
				e.addOperation(result, "user-role", "create", roleName, "Would create user role", nil)
				result.Summary.RolesCreated++
			} else {
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
			if e.options.DryRun {
				logger.Info("DRY RUN: Would remove user role not in config: %s", existingRole.Name)
				e.addOperation(result, "user-role", "delete", existingRole.Name, "Would remove user role not in config", nil)
				result.Summary.RolesDeleted++
			} else {
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
	}

	logger.Info("User roles sync completed")
	return nil
}

// UserRolePermissionMappings holds all necessary mappings for user role permission sync
type UserRolePermissionMappings struct {
	RoleNameToID     map[string]string
	ResourceNameToID map[string]string
	ScopeMapping     *ScopeMapping
}

// buildUserRolePermissionMappings builds all necessary mappings for user role permission synchronization
func (e *Engine) buildUserRolePermissionMappings(cfg *config.Config) (*UserRolePermissionMappings, error) {
	var roleNameToID map[string]string
	var resourceNameToID map[string]string
	var scopeMapping *ScopeMapping

	if e.options.DryRun {
		// In dry-run mode, get real existing entities first
		existingRoles, err := e.client.GetRoles()
		if err != nil {
			return nil, fmt.Errorf("failed to get existing user roles: %w", err)
		}

		existingResources, err := e.client.GetResources()
		if err != nil {
			return nil, fmt.Errorf("failed to get existing resources: %w", err)
		}

		// Start with real mappings
		roleNameToID = CreateRoleNameToIDMapping(existingRoles)
		resourceNameToID = CreateResourceNameToIDMapping(existingResources)

		// Add simulated mappings for roles that don't exist
		for _, role := range cfg.Hierarchy.UserRoles {
			roleNameLower := strings.ToLower(role.Name)
			if _, exists := roleNameToID[roleNameLower]; !exists {
				roleNameToID[roleNameLower] = "dry-run-role-" + roleNameLower
			}
		}

		// Add simulated mappings for resources that don't exist
		for _, resource := range cfg.Hierarchy.Resources {
			if _, exists := resourceNameToID[resource.Name]; !exists {
				resourceNameToID[resource.Name] = "dry-run-resource-" + resource.Name
			}
		}

		// Build scope mapping for dry-run (mix of real and simulated)
		scopeMapping, err = BuildGlobalScopeMapping(e.client, cfg, resourceNameToID)
		if err != nil {
			return nil, err
		}
	} else {
		// Get existing roles
		existingRoles, err := e.client.GetRoles()
		if err != nil {
			return nil, fmt.Errorf("failed to get existing user roles: %w", err)
		}

		// Get existing resources
		existingResources, err := e.client.GetResources()
		if err != nil {
			return nil, fmt.Errorf("failed to get existing resources: %w", err)
		}

		// Create mappings using utility functions
		roleNameToID = CreateRoleNameToIDMapping(existingRoles)
		resourceNameToID = CreateResourceNameToIDMapping(existingResources)

		// Build global scope mapping
		scopeMapping, err = BuildGlobalScopeMapping(e.client, cfg, resourceNameToID)
		if err != nil {
			return nil, err
		}
	}

	return &UserRolePermissionMappings{
		RoleNameToID:     roleNameToID,
		ResourceNameToID: resourceNameToID,
		ScopeMapping:     scopeMapping,
	}, nil
}

// syncUserRolePermissions synchronizes user role permissions
func (e *Engine) syncUserRolePermissions(cfg *config.Config, result *Result) error {
	logger.Info("Syncing user role permissions...")

	// Filter only user type roles
	userRoles := cfg.GetUserTypeRoles(cfg.Hierarchy.UserRoles)

	// Build necessary mappings
	mappings, err := e.buildUserRolePermissionMappings(cfg)
	if err != nil {
		return err
	}

	// Process each user role
	for _, configRole := range userRoles {
		if err := e.syncSingleUserRolePermissions(configRole, mappings, result); err != nil {
			return err
		}
	}

	logger.Info("User role permissions sync completed")
	return nil
}

// syncSingleUserRolePermissions synchronizes permissions for a single user role
func (e *Engine) syncSingleUserRolePermissions(configRole config.Role, mappings *UserRolePermissionMappings, result *Result) error {
	roleID, exists := mappings.RoleNameToID[strings.ToLower(configRole.Name)]
	isDryRunSimulated := false
	if !exists {
		if e.options.DryRun {
			// In dry-run mode, simulate with a dummy ID for roles being created
			roleID = "dry-run-role-" + strings.ToLower(configRole.Name)
			isDryRunSimulated = true
		} else {
			syncLogger := logger.ComponentLogger("sync")
			syncLogger.Warn().
				Str("role", configRole.Name).
				Str("type", "user").
				Msg("Role not found, skipping permission assignment")
			return nil
		}
	}

	// Get current permissions
	var currentPermissions []client.LogtoScope
	if isDryRunSimulated {
		// For simulated roles in dry-run, assume no current permissions
		currentPermissions = []client.LogtoScope{}
	} else {
		var err error
		currentPermissions, err = e.client.GetRolePermissions(roleID)
		if err != nil {
			return fmt.Errorf("failed to get permissions for role %s: %w", configRole.Name, err)
		}
	}

	// Build current and expected permission sets
	currentPermissionIDs := make([]string, 0, len(currentPermissions))
	for _, permission := range currentPermissions {
		currentPermissionIDs = append(currentPermissionIDs, permission.ID)
	}

	expectedPermissionIDs := e.buildExpectedPermissionIDs(configRole, mappings.ScopeMapping)

	// Calculate differences using utility function
	permissionNames := e.convertIDsToNames(currentPermissionIDs, expectedPermissionIDs, mappings.ScopeMapping)
	diff := CalculatePermissionDiff(permissionNames.Current, permissionNames.Expected)

	// Apply changes
	return e.applyUserRolePermissionChanges(roleID, configRole.Name, diff, mappings.ScopeMapping, result)
}

// PermissionNameSets holds current and expected permission names
type PermissionNameSets struct {
	Current  []string
	Expected []string
}

// convertIDsToNames converts permission IDs to names for easier comparison
func (e *Engine) convertIDsToNames(currentIDs, expectedIDs []string, scopeMapping *ScopeMapping) *PermissionNameSets {
	current := make([]string, 0, len(currentIDs))
	for _, id := range currentIDs {
		if name, exists := scopeMapping.IDToName[id]; exists {
			current = append(current, name)
		}
	}

	expected := make([]string, 0, len(expectedIDs))
	for _, id := range expectedIDs {
		if name, exists := scopeMapping.IDToName[id]; exists {
			expected = append(expected, name)
		}
	}

	return &PermissionNameSets{
		Current:  current,
		Expected: expected,
	}
}

// buildExpectedPermissionIDs builds list of expected permission IDs from config
func (e *Engine) buildExpectedPermissionIDs(configRole config.Role, scopeMapping *ScopeMapping) []string {
	expectedIDs := make([]string, 0, len(configRole.Permissions))

	for _, permission := range configRole.Permissions {
		if permission.ID != "" {
			if scopeID, exists := scopeMapping.NameToID[permission.ID]; exists {
				expectedIDs = append(expectedIDs, scopeID)
			} else {
				logger.Warn("Permission/scope %s not found for role %s", permission.ID, configRole.Name)
			}
		}
	}

	return expectedIDs
}

// applyUserRolePermissionChanges applies permission changes to a user role
func (e *Engine) applyUserRolePermissionChanges(roleID, roleName string, diff *PermissionDiff, scopeMapping *ScopeMapping, result *Result) error {
	// Add missing permissions
	if len(diff.ToAdd) > 0 {
		permissionIDs := make([]string, 0, len(diff.ToAdd))
		for _, permName := range diff.ToAdd {
			if scopeID, exists := scopeMapping.NameToID[permName]; exists {
				permissionIDs = append(permissionIDs, scopeID)
			}
		}

		if len(permissionIDs) > 0 {
			if e.options.DryRun {
				logger.Info("DRY RUN: Would assign %d permissions to role %s", len(permissionIDs), roleName)
				e.addOperation(result, "user-role-permission", "assign",
					fmt.Sprintf("%s (%d permissions)", roleName, len(permissionIDs)),
					"Would assign permissions to user role", nil)
				result.Summary.PermissionsCreated += len(permissionIDs)
			} else {
				logger.Info("Assigning %d permissions to role %s", len(permissionIDs), roleName)
				err := e.client.AssignPermissionsToRole(roleID, permissionIDs)
				e.addOperation(result, "user-role-permission", "assign",
					fmt.Sprintf("%s (%d permissions)", roleName, len(permissionIDs)),
					"Assigned permissions to user role", err)
				if err != nil {
					return fmt.Errorf("failed to assign permissions to role %s: %w", roleName, err)
				}
				result.Summary.PermissionsCreated += len(permissionIDs)
			}
		}
	}

	// Remove unwanted permissions
	if len(diff.ToRemove) > 0 {
		permissionIDs := make([]string, 0, len(diff.ToRemove))
		for _, permName := range diff.ToRemove {
			if scopeID, exists := scopeMapping.NameToID[permName]; exists {
				permissionIDs = append(permissionIDs, scopeID)
			}
		}

		if len(permissionIDs) > 0 {
			if e.options.DryRun {
				logger.Info("DRY RUN: Would remove %d permissions from role %s", len(permissionIDs), roleName)
				e.addOperation(result, "user-role-permission", "remove",
					fmt.Sprintf("%s (%d permissions)", roleName, len(permissionIDs)),
					"Would remove permissions from user role", nil)
				result.Summary.PermissionsDeleted += len(permissionIDs)
			} else {
				logger.Info("Removing %d permissions from role %s", len(permissionIDs), roleName)
				err := e.client.RemovePermissionsFromRole(roleID, permissionIDs)
				e.addOperation(result, "user-role-permission", "remove",
					fmt.Sprintf("%s (%d permissions)", roleName, len(permissionIDs)),
					"Removed permissions from user role", err)
				if err != nil {
					logger.Warn("Failed to remove permissions from role %s: %v", roleName, err)
				} else {
					result.Summary.PermissionsDeleted += len(permissionIDs)
				}
			}
		}
	}

	return nil
}
