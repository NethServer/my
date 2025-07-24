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
	Active         bool                   `json:"active" db:"active"`
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
	Active         bool                   `json:"active" db:"active"`
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
	Active         bool                   `json:"active" db:"active"`
}

// LocalUser represents a user stored in local database
type LocalUser struct {
	ID               string                 `json:"id" db:"id"`
	LogtoID          *string                `json:"logto_id" db:"logto_id"`
	Username         string                 `json:"username" db:"username"`
	Email            string                 `json:"email" db:"email"`
	Name             string                 `json:"name" db:"name"`
	Phone            *string                `json:"phone" db:"phone"`
	UserRoleIDs      []string               `json:"userRoleIds" db:"user_role_ids"`
	OrganizationID   *string                `json:"organizationId" db:"organization_id"`
	OrganizationName *string                `json:"organizationName,omitempty"`
	CustomData       map[string]interface{} `json:"customData" db:"custom_data"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
	LogtoSyncedAt    *time.Time             `json:"logto_synced_at" db:"logto_synced_at"`
	Active           bool                   `json:"active" db:"active"`
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
	Password       string                 `json:"password" validate:"required,min=8"`
	UserRoleIDs    []string               `json:"userRoleIds,omitempty"`
	OrganizationID *string                `json:"organizationId,omitempty"`
	CustomData     map[string]interface{} `json:"customData,omitempty"`
}

type UpdateLocalUserRequest struct {
	Username       *string                 `json:"username,omitempty" validate:"omitempty,min=1,max=255"`
	Email          *string                 `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Name           *string                 `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Phone          *string                 `json:"phone,omitempty"`
	UserRoleIDs    *[]string               `json:"userRoleIds,omitempty"`
	OrganizationID *string                 `json:"organizationId,omitempty"`
	CustomData     *map[string]interface{} `json:"customData,omitempty"`
}
