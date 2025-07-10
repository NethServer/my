/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package config

import (
	"fmt"
	"strings"
)

// Config represents the complete configuration structure
type Config struct {
	Metadata  Metadata  `yaml:"metadata" json:"metadata"`
	Hierarchy Hierarchy `yaml:"hierarchy" json:"hierarchy"`
}

// Metadata contains configuration metadata
type Metadata struct {
	Name        string `yaml:"name" json:"name"`
	Version     string `yaml:"version" json:"version"`
	Description string `yaml:"description" json:"description"`
}

// Hierarchy contains the RBAC hierarchy configuration
type Hierarchy struct {
	OrganizationRoles []Role        `yaml:"organization_roles" json:"organization_roles"`
	UserRoles         []Role        `yaml:"user_roles" json:"user_roles"`
	Resources         []Resource    `yaml:"resources" json:"resources"`
	ThirdPartyApps    []Application `yaml:"third_party_apps,omitempty" json:"third_party_apps,omitempty"`
}

// Role represents a role with permissions
type Role struct {
	ID          string       `yaml:"id" json:"id"`
	Name        string       `yaml:"name" json:"name"`
	Type        string       `yaml:"type" json:"type"`
	Priority    int          `yaml:"priority,omitempty" json:"priority,omitempty"`
	Permissions []Permission `yaml:"permissions" json:"permissions"`
}

// Permission represents a permission/scope
type Permission struct {
	ID   string `yaml:"id,omitempty" json:"id,omitempty"`
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
}

// Resource represents an API resource with actions
type Resource struct {
	Name    string   `yaml:"name" json:"name"`
	Actions []string `yaml:"actions" json:"actions"`
}

// Application represents a third-party application configuration
type Application struct {
	Name                   string         `yaml:"name" json:"name"`                                                               // FQDN of the application
	Description            string         `yaml:"description" json:"description"`                                                 // Description of the application
	DisplayName            string         `yaml:"display_name" json:"display_name"`                                               // Display name for branding
	Scopes                 []string       `yaml:"scopes,omitempty" json:"scopes,omitempty"`                                       // Custom scopes (optional)
	RedirectUris           []string       `yaml:"redirect_uris,omitempty" json:"redirect_uris,omitempty"`                         // Redirect URIs for OAuth flow
	PostLogoutRedirectUris []string       `yaml:"post_logout_redirect_uris,omitempty" json:"post_logout_redirect_uris,omitempty"` // Post logout redirect URIs
	AccessControl          *AccessControl `yaml:"access_control,omitempty" json:"access_control,omitempty"`                       // Access control configuration
}

// AccessControl defines which roles can access a third-party application
type AccessControl struct {
	OrganizationRoles []string `yaml:"organization_roles,omitempty" json:"organization_roles,omitempty"` // Organization roles that can access the app
	UserRoles         []string `yaml:"user_roles,omitempty" json:"user_roles,omitempty"`                 // User roles that can access the app
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}

	if c.Metadata.Version == "" {
		return fmt.Errorf("metadata.version is required")
	}

	// Validate organization roles
	orgRoleNames := make(map[string]bool)
	for _, role := range c.Hierarchy.OrganizationRoles {
		if err := c.validateRole(role, "organization"); err != nil {
			return fmt.Errorf("organization role validation failed: %w", err)
		}

		if orgRoleNames[role.ID] {
			return fmt.Errorf("duplicate organization role ID: %s", role.ID)
		}
		orgRoleNames[role.ID] = true
	}

	// Validate user roles
	userRoleNames := make(map[string]bool)
	for _, role := range c.Hierarchy.UserRoles {
		if err := c.validateRole(role, "user"); err != nil {
			return fmt.Errorf("user role validation failed: %w", err)
		}

		if userRoleNames[role.ID] {
			return fmt.Errorf("duplicate user role ID: %s", role.ID)
		}
		userRoleNames[role.ID] = true
	}

	// Validate resources
	resourceNames := make(map[string]bool)
	for _, resource := range c.Hierarchy.Resources {
		if err := c.validateResource(resource); err != nil {
			return fmt.Errorf("resource validation failed: %w", err)
		}

		if resourceNames[resource.Name] {
			return fmt.Errorf("duplicate resource name: %s", resource.Name)
		}
		resourceNames[resource.Name] = true
	}

	// Validate permission references
	if err := c.validatePermissionReferences(); err != nil {
		return fmt.Errorf("permission reference validation failed: %w", err)
	}

	// Validate third-party apps
	appNames := make(map[string]bool)
	for _, app := range c.Hierarchy.ThirdPartyApps {
		if err := c.validateApplication(app); err != nil {
			return fmt.Errorf("third-party app validation failed: %w", err)
		}

		if appNames[app.Name] {
			return fmt.Errorf("duplicate third-party app name: %s", app.Name)
		}
		appNames[app.Name] = true
	}

	return nil
}

func (c *Config) validateRole(role Role, roleType string) error {
	if role.ID == "" {
		return fmt.Errorf("role ID is required")
	}

	if role.Name == "" {
		return fmt.Errorf("role name is required for role %s", role.ID)
	}

	if role.Type != "" && role.Type != roleType && role.Type != "user" {
		return fmt.Errorf("invalid role type %s for role %s", role.Type, role.ID)
	}

	if role.Priority < 0 {
		return fmt.Errorf("role priority must be non-negative for role %s", role.ID)
	}

	// Validate permissions
	permissionIDs := make(map[string]bool)
	for _, perm := range role.Permissions {
		if perm.ID == "" {
			return fmt.Errorf("permission ID is required for role %s", role.ID)
		}

		if permissionIDs[perm.ID] {
			return fmt.Errorf("duplicate permission ID %s in role %s", perm.ID, role.ID)
		}
		permissionIDs[perm.ID] = true
	}

	return nil
}

func (c *Config) validateResource(resource Resource) error {
	if resource.Name == "" {
		return fmt.Errorf("resource name is required")
	}

	if len(resource.Actions) == 0 {
		return fmt.Errorf("resource %s must have at least one action", resource.Name)
	}

	// Validate actions
	actionMap := make(map[string]bool)
	for _, action := range resource.Actions {
		if action == "" {
			return fmt.Errorf("empty action in resource %s", resource.Name)
		}

		if actionMap[action] {
			return fmt.Errorf("duplicate action %s in resource %s", action, resource.Name)
		}
		actionMap[action] = true
	}

	return nil
}

func (c *Config) validatePermissionReferences() error {
	// Build map of valid permissions from resources
	validPermissions := make(map[string]bool)

	for _, resource := range c.Hierarchy.Resources {
		for _, action := range resource.Actions {
			permissionID := fmt.Sprintf("%s:%s", action, resource.Name)
			validPermissions[permissionID] = true
		}
	}

	// Check organization roles
	for _, role := range c.Hierarchy.OrganizationRoles {
		for _, perm := range role.Permissions {
			if !validPermissions[perm.ID] && !c.isSystemPermission(perm.ID) {
				return fmt.Errorf("invalid permission reference %s in organization role %s", perm.ID, role.ID)
			}
		}
	}

	// Check user roles
	for _, role := range c.Hierarchy.UserRoles {
		for _, perm := range role.Permissions {
			if !validPermissions[perm.ID] && !c.isSystemPermission(perm.ID) {
				return fmt.Errorf("invalid permission reference %s in user role %s", perm.ID, role.ID)
			}
		}
	}

	return nil
}

func (c *Config) isSystemPermission(permissionID string) bool {
	// Allow certain system permissions that might not be defined in resources
	systemPatterns := []string{
		"admin:",
		"manage:",
		"view:",
		"create:",
		"read:",
		"update:",
		"delete:",
		"destroy:",
		"audit:",
		"backup:",
	}

	for _, pattern := range systemPatterns {
		if strings.HasPrefix(permissionID, pattern) {
			return true
		}
	}

	return false
}

func (c *Config) validateApplication(app Application) error {
	if app.Name == "" {
		return fmt.Errorf("application name is required")
	}

	if app.Description == "" {
		return fmt.Errorf("application description is required for app %s", app.Name)
	}

	if app.DisplayName == "" {
		return fmt.Errorf("application display_name is required for app %s", app.Name)
	}

	// Validate scopes if provided
	if len(app.Scopes) > 0 {
		scopeMap := make(map[string]bool)
		for _, scope := range app.Scopes {
			if scope == "" {
				return fmt.Errorf("empty scope in application %s", app.Name)
			}

			if scopeMap[scope] {
				return fmt.Errorf("duplicate scope %s in application %s", scope, app.Name)
			}
			scopeMap[scope] = true
		}
	}

	// Validate access control if provided
	if app.AccessControl != nil {
		if err := c.validateAccessControl(*app.AccessControl, app.Name); err != nil {
			return fmt.Errorf("access control validation failed for app %s: %w", app.Name, err)
		}
	}

	return nil
}

func (c *Config) validateAccessControl(accessControl AccessControl, appName string) error {
	// Build map of valid organization roles
	validOrgRoles := make(map[string]bool)
	for _, role := range c.Hierarchy.OrganizationRoles {
		validOrgRoles[role.ID] = true
	}

	// Build map of valid user roles
	validUserRoles := make(map[string]bool)
	for _, role := range c.Hierarchy.UserRoles {
		validUserRoles[role.ID] = true
	}

	// Validate organization roles
	for _, roleID := range accessControl.OrganizationRoles {
		if roleID == "" {
			return fmt.Errorf("empty organization role in application %s", appName)
		}
		if !validOrgRoles[roleID] {
			return fmt.Errorf("invalid organization role %s in application %s", roleID, appName)
		}
	}

	// Validate user roles
	for _, roleID := range accessControl.UserRoles {
		if roleID == "" {
			return fmt.Errorf("empty user role in application %s", appName)
		}
		if !validUserRoles[roleID] {
			return fmt.Errorf("invalid user role %s in application %s", roleID, appName)
		}
	}

	return nil
}

// GetDefaultScopes returns the default scopes for third-party applications
func (c *Config) GetDefaultScopes() []string {
	return []string{
		"profile",
		"email",
		"roles",
		"urn:logto:scope:organizations",
		"urn:logto:scope:organization_roles",
	}
}

// GetUserTypeRoles returns only roles with type "user" or empty type
func (c *Config) GetUserTypeRoles(roles []Role) []Role {
	var userRoles []Role
	for _, role := range roles {
		if role.Type == "user" || role.Type == "" {
			userRoles = append(userRoles, role)
		}
	}
	return userRoles
}

// GetAllPermissions returns all unique permissions from both organization roles and user roles
func (c *Config) GetAllPermissions() map[string]Permission {
	allPermissions := make(map[string]Permission)

	// Get permissions from organization roles
	organizationRoles := c.GetUserTypeRoles(c.Hierarchy.OrganizationRoles)
	for _, role := range organizationRoles {
		for _, permission := range role.Permissions {
			if permission.ID != "" {
				allPermissions[permission.ID] = permission
			}
		}
	}

	// Get permissions from user roles
	userRoles := c.GetUserTypeRoles(c.Hierarchy.UserRoles)
	for _, role := range userRoles {
		for _, permission := range role.Permissions {
			if permission.ID != "" {
				allPermissions[permission.ID] = permission
			}
		}
	}

	return allPermissions
}
