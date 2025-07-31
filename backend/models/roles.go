/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

// Role represents a user role from Logto
type Role struct {
	ID          string `json:"id" structs:"id"`
	Name        string `json:"name" structs:"name"`
	Description string `json:"description" structs:"description"`
}

// OrganizationRole represents an organization role from Logto
type OrganizationRole struct {
	ID          string `json:"id" structs:"id"`
	Name        string `json:"name" structs:"name"`
	Description string `json:"description" structs:"description"`
}

// RolesResponse represents the response for getting all roles
type RolesResponse struct {
	Roles []Role `json:"roles" structs:"roles"`
}

// OrganizationRolesResponse represents the response for getting all organization roles
type OrganizationRolesResponse struct {
	OrganizationRoles []OrganizationRole `json:"organization_roles" structs:"organization_roles"`
}
