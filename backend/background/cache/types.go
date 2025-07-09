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
	"time"

	"github.com/nethesis/my/backend/models"
)

// Cache data structures

// JitRolesCache represents cached JIT organization roles
type JitRolesCache struct {
	Roles     []models.LogtoOrganizationRole `json:"roles"`
	CachedAt  time.Time                      `json:"cachedAt"`
	ExpiresAt time.Time                      `json:"expiresAt"`
}

// OrgUsersCache represents cached organization users
type OrgUsersCache struct {
	Users     []models.LogtoUser `json:"users"`
	CachedAt  time.Time          `json:"cachedAt"`
	ExpiresAt time.Time          `json:"expiresAt"`
}

// CacheManager aggregates all cache managers for centralized access
type CacheManager struct {
	JitRoles *JitRolesCacheManager
	OrgUsers *OrgUsersCacheManager
}

// GetCacheManager returns a centralized cache manager instance
func GetCacheManager() *CacheManager {
	return &CacheManager{
		JitRoles: GetJitRolesCacheManager(),
		OrgUsers: GetOrgUsersCacheManager(),
	}
}
