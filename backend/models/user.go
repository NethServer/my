/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

type User struct {
	ID                 string   `json:"id" structs:"id"`
	Username           string   `json:"username" structs:"username"`
	Email              string   `json:"email" structs:"email"`
	Roles              []string `json:"roles" structs:"roles"`                             // User roles (Support, Sales, etc.)
	Scopes             []string `json:"scopes" structs:"scopes"`                           // User scopes (create:systems, etc.)
	OrganizationRoles  []string `json:"organization_roles" structs:"organization_roles"`   // Organization roles (God, Distributor, Reseller, Customer)
	OrganizationScopes []string `json:"organization_scopes" structs:"organization_scopes"` // Organization scopes (create:reseller, manage:customer, etc.)
}
