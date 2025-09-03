/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package constants

// HTTP and API constants
const (
	DefaultHTTPTimeout = 30   // seconds
	DefaultTokenTTL    = 3600 // seconds
)

// API Endpoints
const (
	EndpointApplications       = "/api/applications"
	EndpointUsers              = "/api/users"
	EndpointOrganizations      = "/api/organizations"
	EndpointOrganizationRoles  = "/api/organization-roles"
	EndpointOrganizationScopes = "/api/organization-scopes"
	EndpointRoles              = "/api/roles"
	EndpointDomains            = "/api/domains"
	EndpointResources          = "/api/resources"
	EndpointScopes             = "/api/scopes"
)

// Entity Names and Default Values
const (
	// Application Names
	BackendAppName  = "backend"
	FrontendAppName = "frontend"

	// Organization Names
	OwnerOrgName        = "Owner"
	OwnerOrgDescription = "Owner organization - complete control over commercial hierarchy"

	// Role Names
	AdminRoleName      = "Admin"
	AdminRoleID        = "admin"
	SuperAdminRoleName = "Super Admin"
	SuperAdminRoleID   = "super"
	OwnerRoleName      = "Owner"
	OwnerRoleID        = "owner"

	// Application Types
	AppTypeSPA = "SPA"
	AppTypeM2M = "MachineToMachine"

	// Default Password Settings
	DefaultPasswordLength = 16

	// Organization Scopes
	ScopeCreateDistributors = "create:distributors"
	ScopeManageDistributors = "manage:distributors"
	ScopeCreateResellers    = "create:resellers"
	ScopeManageResellers    = "manage:resellers"
	ScopeCreateCustomers    = "create:customers"
	ScopeManageCustomers    = "manage:customers"
)
