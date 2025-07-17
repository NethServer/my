/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services"
)

// GetRoles returns all available user roles
func GetRoles(c *gin.Context) {
	client := services.NewLogtoManagementClient()

	// Fetch all roles from Logto
	logtoRoles, err := client.GetAllRoles()
	if err != nil {
		httpLogger := logger.NewHTTPErrorLogger(c, "roles")
		httpLogger.LogError(err, "get_roles", http.StatusInternalServerError, "Failed to fetch roles from Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to fetch roles", nil))
		return
	}

	// Filter out system roles and convert LogtoRole to Role model
	roles := make([]models.Role, 0)
	for _, logtoRole := range logtoRoles {
		// Skip system roles based on name patterns
		if isSystemRole(logtoRole.Name, logtoRole.Description) {
			continue
		}
		roles = append(roles, models.Role{
			ID:          logtoRole.ID,
			Name:        logtoRole.Name,
			Description: logtoRole.Description,
		})
	}

	logger.ComponentLogger("roles").Info().
		Str("operation", "get_roles").
		Int("role_count", len(roles)).
		Msg("Roles fetched successfully")

	c.JSON(http.StatusOK, response.OK("roles retrieved successfully", models.RolesResponse{
		Roles: roles,
	}))
}

// GetOrganizationRoles returns all available organization roles
func GetOrganizationRoles(c *gin.Context) {
	client := services.NewLogtoManagementClient()

	// Fetch all organization roles from Logto
	logtoOrgRoles, err := client.GetAllOrganizationRoles()
	if err != nil {
		httpLogger := logger.NewHTTPErrorLogger(c, "roles")
		httpLogger.LogError(err, "get_organization_roles", http.StatusInternalServerError, "Failed to fetch organization roles from Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to fetch organization roles", nil))
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
		"system",
		"admin",
		"machine-to-machine",
		"m2m",
		"default",
		"service",
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
