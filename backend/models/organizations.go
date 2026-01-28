/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

// OrganizationSummary represents a simplified organization for selection/assignment
type OrganizationSummary struct {
	ID          string `json:"id" structs:"id"`             // Database UUID
	LogtoID     string `json:"logto_id" structs:"logto_id"` // Logto organization ID
	Name        string `json:"name" structs:"name"`
	Description string `json:"description" structs:"description"`
	Type        string `json:"type" structs:"type"` // "owner", "distributor", "reseller", "customer"
}

// OrganizationsResponse represents the response for getting filtered organizations
type OrganizationsResponse struct {
	Organizations []OrganizationSummary `json:"organizations" structs:"organizations"`
}

// PaginatedOrganizationsResponse represents the response for getting filtered organizations with pagination
type PaginatedOrganizationsResponse struct {
	Organizations []OrganizationSummary `json:"organizations" structs:"organizations"`
	Pagination    PaginationInfo        `json:"pagination" structs:"pagination"`
}
