/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

type User struct {
	ID               string   `json:"id" structs:"id"`
	Username         string   `json:"username" structs:"username"`                   // Username from Logto
	Email            string   `json:"email" structs:"email"`                         // Email from Logto
	Name             string   `json:"name" structs:"name"`                           // Display name from Logto
	UserRoles        []string `json:"user_roles" structs:"user_roles"`               // Technical capabilities (Admin, Support)
	UserPermissions  []string `json:"user_permissions" structs:"user_permissions"`   // Permissions derived from user roles
	OrgRole          string   `json:"org_role" structs:"org_role"`                   // Business hierarchy role (God, Distributor, Reseller, Customer)
	OrgPermissions   []string `json:"org_permissions" structs:"org_permissions"`     // Permissions derived from organization role
	OrganizationID   string   `json:"organization_id" structs:"organization_id"`     // Which organization the user belongs to
	OrganizationName string   `json:"organization_name" structs:"organization_name"` // Organization name for display
}
