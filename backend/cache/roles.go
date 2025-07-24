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
	"sync"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/services/logto"
)

// RoleNames provides in-memory access to role names from Logto
type RoleNames struct {
	roles  map[string]string // roleID -> roleName
	mutex  sync.RWMutex
	loaded bool
}

var (
	roleNames     *RoleNames
	roleNamesOnce sync.Once
)

// GetRoleNames returns a singleton instance of the role names store
func GetRoleNames() *RoleNames {
	roleNamesOnce.Do(func() {
		roleNames = &RoleNames{
			roles: make(map[string]string),
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

	// Clear existing roles
	r.roles = make(map[string]string)

	// Populate with new data
	for _, role := range roles {
		r.roles[role.ID] = role.Name
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
