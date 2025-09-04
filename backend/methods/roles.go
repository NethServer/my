/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/logto"
)

// GetRoles returns all available user roles filtered by access control
func GetRoles(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	client := logto.NewManagementClient()

	// Fetch all roles from Logto
	logtoRoles, err := client.GetAllRoles()
	if err != nil {
		httpLogger := logger.NewHTTPErrorLogger(c, "roles")
		httpLogger.LogError(err, "get_roles", http.StatusInternalServerError, "Failed to fetch roles from Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch roles", nil))
		return
	}

	// Get role cache for access control information
	roleCache := cache.GetRoleNames()

	// Filter out system roles and apply access control filtering
	roles := make([]models.Role, 0)
	for _, logtoRole := range logtoRoles {
		// Skip system roles based on name patterns
		if isSystemRole(logtoRole.Name, logtoRole.Description) {
			logger.ComponentLogger("roles").Debug().
				Str("role_name", logtoRole.Name).
				Str("role_description", logtoRole.Description).
				Msg("Role filtered as system role")
			continue
		}

		// Check if user can access this role based on cached access control
		canAccess := canUserAccessRoleCached(roleCache, logtoRole.ID, user)

		logger.ComponentLogger("roles").Debug().
			Str("role_name", logtoRole.Name).
			Str("role_id", logtoRole.ID).
			Bool("can_access", canAccess).
			Msg("Role access check result")

		if canAccess {
			roles = append(roles, models.Role{
				ID:          logtoRole.ID,
				Name:        logtoRole.Name,
				Description: logtoRole.Description,
			})
		}
	}

	logger.ComponentLogger("roles").Info().
		Str("operation", "get_roles").
		Str("user_id", user.ID).
		Str("user_org_role", user.OrgRole).
		Int("total_roles", len(logtoRoles)).
		Int("accessible_roles", len(roles)).
		Msg("Roles fetched and filtered successfully")

	c.JSON(http.StatusOK, response.OK("roles retrieved successfully", models.RolesResponse{
		Roles: roles,
	}))
}

// GetOrganizationRoles returns all available organization roles
func GetOrganizationRoles(c *gin.Context) {
	client := logto.NewManagementClient()

	// Fetch all organization roles from Logto
	logtoOrgRoles, err := client.GetAllOrganizationRoles()
	if err != nil {
		httpLogger := logger.NewHTTPErrorLogger(c, "roles")
		httpLogger.LogError(err, "get_organization_roles", http.StatusInternalServerError, "Failed to fetch organization roles from Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch organization roles", nil))
		return
	}

	// Convert LogtoOrganizationRole to OrganizationRole model
	orgRoles := make([]models.OrganizationRole, len(logtoOrgRoles))
	for i, logtoOrgRole := range logtoOrgRoles {
		orgRoles[i] = models.OrganizationRole{
			ID:          logtoOrgRole.ID,
			Name:        logtoOrgRole.Name,
			Description: logtoOrgRole.Description,
		}
	}

	logger.ComponentLogger("roles").Info().
		Str("operation", "get_organization_roles").
		Int("org_role_count", len(orgRoles)).
		Msg("Organization roles fetched successfully")

	c.JSON(http.StatusOK, response.OK("organization roles retrieved successfully", models.OrganizationRolesResponse{
		OrganizationRoles: orgRoles,
	}))
}

// isSystemRole checks if a role is a system role that should be filtered out
func isSystemRole(name, description string) bool {
	// Convert to lowercase for case-insensitive comparison
	nameLower := strings.ToLower(name)
	descLower := strings.ToLower(description)

	// System role name patterns
	systemPatterns := []string{
		"logto",
		"management api",
		"machine-to-machine",
		"m2m",
		"default",
	}

	// Check name patterns
	for _, pattern := range systemPatterns {
		if strings.Contains(nameLower, pattern) {
			return true
		}
	}

	// Check description patterns
	for _, pattern := range systemPatterns {
		if strings.Contains(descLower, pattern) {
			return true
		}
	}

	return false
}

// canUserAccessRoleCached checks if a user can access/assign a specific role using cached access control data
func canUserAccessRoleCached(roleCache *cache.RoleNames, roleID string, user *models.User) bool {
	// Get access control information from cache
	accessControl, exists := roleCache.GetAccessControl(roleID)
	if !exists {
		// If role not found in cache, deny access for security (fail closed)
		logger.ComponentLogger("roles").Warn().
			Str("role_id", roleID).
			Msg("Role access control not found in cache, denying access for security")
		return false
	}

	logger.ComponentLogger("roles").Debug().
		Str("role_id", roleID).
		Bool("has_access_control", accessControl.HasAccessControl).
		Str("required_org_role", accessControl.RequiredOrgRole).
		Str("user_org_role", user.OrgRole).
		Msg("Access control check details")

	// If role has no access control restrictions, everyone can access it
	if !accessControl.HasAccessControl {
		return true
	}

	// Check if user's organization role has sufficient privileges
	hasPermission := HasOrgRolePermission(user.OrgRole, accessControl.RequiredOrgRole)

	logger.ComponentLogger("roles").Debug().
		Str("role_id", roleID).
		Str("user_org_role", user.OrgRole).
		Str("required_org_role", accessControl.RequiredOrgRole).
		Bool("has_permission", hasPermission).
		Msg("Organization role permission check result")

	return hasPermission
}

// HasOrgRolePermission checks if userOrgRole has permission to access requiredOrgRole
// Following the business hierarchy: Owner > Distributor > Reseller > Customer
func HasOrgRolePermission(userOrgRole, requiredOrgRole string) bool {
	// Define hierarchy levels (case-insensitive)
	orgRoleLevels := map[string]int{
		"owner":       1,
		"distributor": 2,
		"reseller":    3,
		"customer":    4,
	}

	// Normalize to lowercase for case-insensitive comparison
	userLevel, userExists := orgRoleLevels[strings.ToLower(userOrgRole)]
	requiredLevel, requiredExists := orgRoleLevels[strings.ToLower(requiredOrgRole)]

	// If either role is not recognized, deny access for security
	if !userExists || !requiredExists {
		return false
	}

	// User can access if their level is equal or higher (lower number = higher privilege)
	return userLevel <= requiredLevel
}

// HasPermission checks if a permission exists in either user permissions or org permissions arrays
func HasPermission(userPermissions, orgPermissions []string, permission string) bool {
	// Check user permissions (from user roles)
	for _, p := range userPermissions {
		if p == permission {
			return true
		}
	}
	// Check organization permissions (from organization role)
	for _, p := range orgPermissions {
		if p == permission {
			return true
		}
	}
	return false
}
