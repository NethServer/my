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
}

// LocalReseller represents a reseller stored in local database
type LocalReseller struct {
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
}

// LocalCustomer represents a customer stored in local database
type LocalCustomer struct {
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
}

// CustomerFilters represents filters for customer queries
type CustomerFilters struct {
	Search string `json:"search,omitempty"` // general search term
}

// DistributorFilters represents filters for distributor queries
type DistributorFilters struct {
	Search string `json:"search,omitempty"` // general search term
}

// ResellerFilters represents filters for reseller queries
type ResellerFilters struct {
	Search string `json:"search,omitempty"` // general search term
}

// UserOrganization represents organization info in user responses
type UserOrganization struct {
	ID      string `json:"id"`
	LogtoID string `json:"logto_id"`
	Name    string `json:"name"`
}

// UserRole represents role info in user responses
type UserRole struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LocalUser represents a user stored in local database
type LocalUser struct {
	ID            string                 `json:"id" db:"id"`
	LogtoID       *string                `json:"logto_id" db:"logto_id"`
	Username      string                 `json:"username" db:"username"`
	Email         string                 `json:"email" db:"email"`
	Name          string                 `json:"name" db:"name"`
	Phone         *string                `json:"phone" db:"phone"`
	Organization  *UserOrganization      `json:"organization,omitempty"`
	Roles         []UserRole             `json:"roles,omitempty"`
	CustomData    map[string]interface{} `json:"custom_data" db:"custom_data"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
	LogtoSyncedAt *time.Time             `json:"logto_synced_at" db:"logto_synced_at"`
	LatestLoginAt *time.Time             `json:"latest_login_at" db:"latest_login_at"`
	DeletedAt     *time.Time             `json:"deleted_at" db:"deleted_at"`     // Soft delete timestamp
	SuspendedAt   *time.Time             `json:"suspended_at" db:"suspended_at"` // Suspension timestamp

	// Internal fields for database operations (not serialized to JSON)
	UserRoleIDs         []string `json:"-" db:"user_role_ids"`
	OrganizationID      *string  `json:"-" db:"organization_id"`
	OrganizationName    *string  `json:"-"`
	OrganizationLocalID *string  `json:"-"`
}

// SystemTotals represents total counts and status for systems
type SystemTotals struct {
	Total          int `json:"total"`
	Alive          int `json:"alive"`
	Dead           int `json:"dead"`
	Zombie         int `json:"zombie"`
	TimeoutMinutes int `json:"timeout_minutes"`
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

// Active returns true if the distributor is not deleted
func (d *LocalDistributor) Active() bool {
	return d.DeletedAt == nil
}

// IsDeleted returns true if the distributor is soft-deleted
func (d *LocalDistributor) IsDeleted() bool {
	return d.DeletedAt != nil
}

// Active returns true if the reseller is not deleted
func (r *LocalReseller) Active() bool {
	return r.DeletedAt == nil
}

// IsDeleted returns true if the reseller is soft-deleted
func (r *LocalReseller) IsDeleted() bool {
	return r.DeletedAt != nil
}

// Active returns true if the customer is not deleted
func (c *LocalCustomer) Active() bool {
	return c.DeletedAt == nil
}

// IsDeleted returns true if the customer is soft-deleted
func (c *LocalCustomer) IsDeleted() bool {
	return c.DeletedAt != nil
}
