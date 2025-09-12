/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package cache

import (
	"strings"
	"sync"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/services/logto"
)

// RoleAccessControl holds access control information for a role
type RoleAccessControl struct {
	RequiredOrgRole  string // Required organization role (e.g., "owner", "distributor")
	HasAccessControl bool   // Whether this role has access control restrictions
}

// RoleNames provides in-memory access to role names and access control from Logto
type RoleNames struct {
	roles         map[string]string            // roleID -> roleName
	accessControl map[string]RoleAccessControl // roleID -> access control info
	mutex         sync.RWMutex
	loaded        bool
}

var (
	roleNames     *RoleNames
	roleNamesOnce sync.Once
)

// GetRoleNames returns a singleton instance of the role names store
func GetRoleNames() *RoleNames {
	roleNamesOnce.Do(func() {
		roleNames = &RoleNames{
			roles:         make(map[string]string),
			accessControl: make(map[string]RoleAccessControl),
		}
	})
	return roleNames
}

// LoadRoles fetches all roles from Logto and stores them in memory
// This should be called at server startup
func (r *RoleNames) LoadRoles() error {
	logger.ComponentLogger("roles").Info().
		Msg("Loading roles from Logto")

	client := logto.NewManagementClient()
	roles, err := client.GetAllRoles()
	if err != nil {
		logger.ComponentLogger("roles").Error().
			Err(err).
			Msg("Failed to load roles from Logto")
		return err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Clear existing data
	r.roles = make(map[string]string)
	r.accessControl = make(map[string]RoleAccessControl)

	// Populate with new data
	for _, role := range roles {
		r.roles[role.ID] = role.Name

		// Load access control information for this role
		accessControlInfo, err := r.loadRoleAccessControl(client, role.ID)
		if err != nil {
			logger.ComponentLogger("roles").Warn().
				Err(err).
				Str("role_id", role.ID).
				Str("role_name", role.Name).
				Msg("Failed to load access control for role, defaulting to no restrictions")
			// Default to no access control restrictions on error
			accessControlInfo = RoleAccessControl{HasAccessControl: false}
		}
		r.accessControl[role.ID] = accessControlInfo
	}

	r.loaded = true

	logger.ComponentLogger("roles").Info().
		Int("role_count", len(roles)).
		Msg("Roles loaded successfully")

	return nil
}

// GetNames returns the names for the given role IDs
func (r *RoleNames) GetNames(roleIDs []string) []string {
	if len(roleIDs) == 0 {
		return []string{}
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if !r.loaded {
		logger.ComponentLogger("roles").Warn().
			Msg("Roles not loaded yet, returning empty names")
		return []string{}
	}

	var names []string
	for _, roleID := range roleIDs {
		if roleName, exists := r.roles[roleID]; exists {
			names = append(names, roleName)
		} else {
			logger.ComponentLogger("roles").Warn().
				Str("role_id", roleID).
				Msg("Role ID not found")
		}
	}

	return names
}

// IsLoaded returns true if roles have been loaded
func (r *RoleNames) IsLoaded() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.loaded
}

// GetAccessControl returns the access control information for a role
func (r *RoleNames) GetAccessControl(roleID string) (RoleAccessControl, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if !r.loaded {
		return RoleAccessControl{}, false
	}

	accessControl, exists := r.accessControl[roleID]
	return accessControl, exists
}

// loadRoleAccessControl loads access control information for a specific role
func (r *RoleNames) loadRoleAccessControl(client *logto.LogtoManagementClient, roleID string) (RoleAccessControl, error) {
	// Get role scopes to check for access control restrictions
	roleScopes, err := client.GetRoleScopes(roleID)
	if err != nil {
		return RoleAccessControl{}, err
	}

	// Check if role has any access control scopes
	for _, scope := range roleScopes {
		if strings.HasSuffix(scope.Name, ":role-access-control") {
			// Extract required organization role from scope name
			// Format: "owner:role-access-control" -> "owner"
			requiredOrgRole := strings.TrimSuffix(scope.Name, ":role-access-control")
			return RoleAccessControl{
				RequiredOrgRole:  requiredOrgRole,
				HasAccessControl: true,
			}, nil
		}
	}

	// No access control scope found
	return RoleAccessControl{HasAccessControl: false}, nil
}
