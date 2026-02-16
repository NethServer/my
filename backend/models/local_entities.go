/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

import (
	"time"
)

// LocalDistributor represents a distributor stored in local database
type LocalDistributor struct {
	ID             string                 `json:"id" db:"id"`
	LogtoID        *string                `json:"logto_id" db:"logto_id"`
	Name           string                 `json:"name" db:"name"`
	Description    string                 `json:"description" db:"description"`
	CustomData     map[string]interface{} `json:"custom_data" db:"custom_data"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
	LogtoSyncedAt  *time.Time             `json:"logto_synced_at" db:"logto_synced_at"`
	LogtoSyncError *string                `json:"logto_sync_error" db:"logto_sync_error"`
	DeletedAt      *time.Time             `json:"deleted_at" db:"deleted_at"`
	SuspendedAt    *time.Time             `json:"suspended_at" db:"suspended_at"`

	// Rebranding info (populated by handler)
	RebrandingEnabled bool    `json:"rebranding_enabled"`
	RebrandingOrgID   *string `json:"rebranding_org_id,omitempty"`

	// Inline stats (populated by List queries only, omitted in other responses)
	SystemsCount      *int `json:"systems_count,omitempty"`
	ResellersCount    *int `json:"resellers_count,omitempty"`
	CustomersCount    *int `json:"customers_count,omitempty"`
	ApplicationsCount *int `json:"applications_count,omitempty"`
}

// LocalReseller represents a reseller stored in local database
type LocalReseller struct {
	ID               string                 `json:"id" db:"id"`
	LogtoID          *string                `json:"logto_id" db:"logto_id"`
	Name             string                 `json:"name" db:"name"`
	Description      string                 `json:"description" db:"description"`
	CustomData       map[string]interface{} `json:"custom_data" db:"custom_data"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
	LogtoSyncedAt    *time.Time             `json:"logto_synced_at" db:"logto_synced_at"`
	LogtoSyncError   *string                `json:"logto_sync_error" db:"logto_sync_error"`
	DeletedAt        *time.Time             `json:"deleted_at" db:"deleted_at"`
	SuspendedAt      *time.Time             `json:"suspended_at" db:"suspended_at"`
	SuspendedByOrgID *string                `json:"suspended_by_org_id" db:"suspended_by_org_id"`

	// Rebranding info (populated by handler)
	RebrandingEnabled bool    `json:"rebranding_enabled"`
	RebrandingOrgID   *string `json:"rebranding_org_id,omitempty"`

	// Inline stats (populated by List queries only, omitted in other responses)
	SystemsCount      *int `json:"systems_count,omitempty"`
	CustomersCount    *int `json:"customers_count,omitempty"`
	ApplicationsCount *int `json:"applications_count,omitempty"`
}

// LocalCustomer represents a customer stored in local database
type LocalCustomer struct {
	ID               string                 `json:"id" db:"id"`
	LogtoID          *string                `json:"logto_id" db:"logto_id"`
	Name             string                 `json:"name" db:"name"`
	Description      string                 `json:"description" db:"description"`
	CustomData       map[string]interface{} `json:"custom_data" db:"custom_data"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
	LogtoSyncedAt    *time.Time             `json:"logto_synced_at" db:"logto_synced_at"`
	LogtoSyncError   *string                `json:"logto_sync_error" db:"logto_sync_error"`
	DeletedAt        *time.Time             `json:"deleted_at" db:"deleted_at"`
	SuspendedAt      *time.Time             `json:"suspended_at" db:"suspended_at"`
	SuspendedByOrgID *string                `json:"suspended_by_org_id" db:"suspended_by_org_id"`

	// Rebranding info (populated by handler)
	RebrandingEnabled bool    `json:"rebranding_enabled"`
	RebrandingOrgID   *string `json:"rebranding_org_id,omitempty"`

	// Inline stats (populated by List queries only, omitted in other responses)
	SystemsCount      *int `json:"systems_count,omitempty"`
	ApplicationsCount *int `json:"applications_count,omitempty"`
}

// CustomerFilters represents filters for customer queries
type CustomerFilters struct {
	Search string `json:"search,omitempty"` // general search term
	Status string `json:"status,omitempty"` // enabled, blocked, or empty for all
}

// DistributorFilters represents filters for distributor queries
type DistributorFilters struct {
	Search string `json:"search,omitempty"` // general search term
	Status string `json:"status,omitempty"` // enabled, blocked, or empty for all
}

// ResellerFilters represents filters for reseller queries
type ResellerFilters struct {
	Search string `json:"search,omitempty"` // general search term
	Status string `json:"status,omitempty"` // enabled, blocked, or empty for all
}

// UserOrganization represents organization info in user responses
type UserOrganization struct {
	ID      string `json:"id"`
	LogtoID string `json:"logto_id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
}

// UserRole represents role info in user responses
type UserRole struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LocalUser represents a user stored in local database
type LocalUser struct {
	ID               string                 `json:"id" db:"id"`
	LogtoID          *string                `json:"logto_id" db:"logto_id"`
	Username         string                 `json:"username" db:"username"`
	Email            string                 `json:"email" db:"email"`
	Name             string                 `json:"name" db:"name"`
	Phone            *string                `json:"phone" db:"phone"`
	Organization     *UserOrganization      `json:"organization,omitempty"`
	Roles            []UserRole             `json:"roles,omitempty"`
	CustomData       map[string]interface{} `json:"custom_data" db:"custom_data"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
	LogtoSyncedAt    *time.Time             `json:"logto_synced_at" db:"logto_synced_at"`
	LatestLoginAt    *time.Time             `json:"latest_login_at" db:"latest_login_at"`
	DeletedAt        *time.Time             `json:"deleted_at" db:"deleted_at"`                   // Soft delete timestamp
	SuspendedAt      *time.Time             `json:"suspended_at" db:"suspended_at"`               // Suspension timestamp
	SuspendedByOrgID *string                `json:"suspended_by_org_id" db:"suspended_by_org_id"` // Organization that caused cascade suspension
	DeletedByOrgID   *string                `json:"deleted_by_org_id" db:"deleted_by_org_id"`     // Organization that caused cascade soft-deletion

	// Internal fields for database operations (not serialized to JSON)
	UserRoleIDs         []string `json:"-" db:"user_role_ids"`
	OrganizationID      *string  `json:"-" db:"organization_id"`
	OrganizationName    *string  `json:"-"`
	OrganizationLocalID *string  `json:"-"`
	OrganizationType    *string  `json:"-"`
}

// SystemTotals represents total counts and status for systems
type SystemTotals struct {
	Total          int `json:"total"`
	Active         int `json:"active"`
	Inactive       int `json:"inactive"`
	Unknown        int `json:"unknown"`
	TimeoutMinutes int `json:"timeout_minutes"`
}

// OrganizationStats represents statistics for an organization (users and systems count)
type OrganizationStats struct {
	UsersCount   int `json:"users_count"`
	SystemsCount int `json:"systems_count"`
}

// DistributorStats represents statistics for a distributor (includes resellers, customers, and applications)
type DistributorStats struct {
	UsersCount                 int `json:"users_count"`
	SystemsCount               int `json:"systems_count"`
	ResellersCount             int `json:"resellers_count"`
	CustomersCount             int `json:"customers_count"`
	ApplicationsCount          int `json:"applications_count"`           // direct applications
	ApplicationsHierarchyCount int `json:"applications_hierarchy_count"` // applications in hierarchy
}

// ResellerStats represents statistics for a reseller (includes customers and applications)
type ResellerStats struct {
	UsersCount                 int `json:"users_count"`
	SystemsCount               int `json:"systems_count"`
	CustomersCount             int `json:"customers_count"`
	ApplicationsCount          int `json:"applications_count"`           // direct applications
	ApplicationsHierarchyCount int `json:"applications_hierarchy_count"` // applications in hierarchy
}

// CustomerStats represents statistics for a customer (includes applications)
type CustomerStats struct {
	UsersCount        int `json:"users_count"`
	SystemsCount      int `json:"systems_count"`
	ApplicationsCount int `json:"applications_count"` // direct applications only (leaf node)
}

// Create requests
type CreateLocalDistributorRequest struct {
	Name        string                 `json:"name" validate:"required,min=1,max=255"`
	Description string                 `json:"description,omitempty"`
	CustomData  map[string]interface{} `json:"custom_data,omitempty"`
}

type CreateLocalResellerRequest struct {
	Name        string                 `json:"name" validate:"required,min=1,max=255"`
	Description string                 `json:"description,omitempty"`
	CustomData  map[string]interface{} `json:"custom_data,omitempty"`
}

type CreateLocalCustomerRequest struct {
	Name        string                 `json:"name" validate:"required,min=1,max=255"`
	Description string                 `json:"description,omitempty"`
	CustomData  map[string]interface{} `json:"custom_data,omitempty"`
}

// Update requests
type UpdateLocalDistributorRequest struct {
	Name        *string                 `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string                 `json:"description,omitempty"`
	CustomData  *map[string]interface{} `json:"custom_data,omitempty"`
}

type UpdateLocalResellerRequest struct {
	Name        *string                 `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string                 `json:"description,omitempty"`
	CustomData  *map[string]interface{} `json:"custom_data,omitempty"`
}

type UpdateLocalCustomerRequest struct {
	Name        *string                 `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string                 `json:"description,omitempty"`
	CustomData  *map[string]interface{} `json:"custom_data,omitempty"`
}

// User CRUD requests
type CreateLocalUserRequest struct {
	Username       string                 `json:"username,omitempty" validate:"omitempty,min=1,max=255"`
	Email          string                 `json:"email" validate:"required,email,max=255"`
	Name           string                 `json:"name" validate:"required,min=1,max=255"`
	Phone          *string                `json:"phone,omitempty"`
	UserRoleIDs    []string               `json:"user_role_ids,omitempty"`
	OrganizationID *string                `json:"organization_id,omitempty"`
	CustomData     map[string]interface{} `json:"custom_data,omitempty"`
}

type UpdateLocalUserRequest struct {
	Username       *string                 `json:"username,omitempty" validate:"omitempty,min=1,max=255"`
	Email          *string                 `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Name           *string                 `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Phone          *string                 `json:"phone,omitempty"`
	UserRoleIDs    *[]string               `json:"user_role_ids,omitempty"`
	OrganizationID *string                 `json:"organization_id,omitempty"`
	CustomData     *map[string]interface{} `json:"custom_data,omitempty"`
}

// Suspension/reactivation requests
type SuspendUserRequest struct {
	Reason *string `json:"reason,omitempty"`
}

type ReactivateUserRequest struct {
	Reason *string `json:"reason,omitempty"`
}

// Active returns true if the user is not deleted and not suspended
func (u *LocalUser) Active() bool {
	return u.DeletedAt == nil && u.SuspendedAt == nil
}

// IsDeleted returns true if the user is soft-deleted
func (u *LocalUser) IsDeleted() bool {
	return u.DeletedAt != nil
}

// IsSuspended returns true if the user is suspended
func (u *LocalUser) IsSuspended() bool {
	return u.SuspendedAt != nil
}

// Active returns true if the distributor is not deleted and not suspended
func (d *LocalDistributor) Active() bool {
	return d.DeletedAt == nil && d.SuspendedAt == nil
}

// IsDeleted returns true if the distributor is soft-deleted
func (d *LocalDistributor) IsDeleted() bool {
	return d.DeletedAt != nil
}

// IsSuspended returns true if the distributor is suspended
func (d *LocalDistributor) IsSuspended() bool {
	return d.SuspendedAt != nil
}

// Active returns true if the reseller is not deleted and not suspended
func (r *LocalReseller) Active() bool {
	return r.DeletedAt == nil && r.SuspendedAt == nil
}

// IsDeleted returns true if the reseller is soft-deleted
func (r *LocalReseller) IsDeleted() bool {
	return r.DeletedAt != nil
}

// IsSuspended returns true if the reseller is suspended
func (r *LocalReseller) IsSuspended() bool {
	return r.SuspendedAt != nil
}

// Active returns true if the customer is not deleted and not suspended
func (c *LocalCustomer) Active() bool {
	return c.DeletedAt == nil && c.SuspendedAt == nil
}

// IsDeleted returns true if the customer is soft-deleted
func (c *LocalCustomer) IsDeleted() bool {
	return c.DeletedAt != nil
}

// IsSuspended returns true if the customer is suspended
func (c *LocalCustomer) IsSuspended() bool {
	return c.SuspendedAt != nil
}

// VATValidationResponse represents a VAT validation response
type VATValidationResponse struct {
	Exists bool `json:"exists"`
}
