/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import "time"

// CreateAccountRequest represents the request payload for creating a new account
type CreateAccountRequest struct {
	Username       string                 `json:"username" binding:"required" structs:"username"`
	Email          string                 `json:"email" binding:"required,email" structs:"email"`
	Name           string                 `json:"name" binding:"required" structs:"name"`
	Phone          string                 `json:"phone" structs:"phone"`
	Password       string                 `json:"password" binding:"required,min=8" structs:"password"`
	UserRoleID     string                 `json:"userRoleId" binding:"required" structs:"userRoleId"`         // Role ID instead of name for security
	OrganizationID string                 `json:"organizationId" binding:"required" structs:"organizationId"` // Which organization they belong to
	Avatar         string                 `json:"avatar" structs:"avatar"`
	CustomData     map[string]interface{} `json:"customData" structs:"customData"`
}

// UpdateAccountRequest represents the request payload for updating an existing account
type UpdateAccountRequest struct {
	Username       string                 `json:"username" structs:"username"`
	Email          string                 `json:"email" structs:"email"`
	Name           string                 `json:"name" structs:"name"`
	Phone          string                 `json:"phone" structs:"phone"`
	UserRoleID     string                 `json:"userRoleId" structs:"userRoleId"`         // Role ID instead of name for security
	OrganizationID string                 `json:"organizationId" structs:"organizationId"` // Which organization they belong to
	Avatar         string                 `json:"avatar" structs:"avatar"`
	CustomData     map[string]interface{} `json:"customData" structs:"customData"`
}

// AccountResponse represents the response format for account data
type AccountResponse struct {
	ID               string                 `json:"id" structs:"id"`
	Username         string                 `json:"username" structs:"username"`
	Email            string                 `json:"email" structs:"email"`
	Name             string                 `json:"name" structs:"name"`
	Phone            string                 `json:"phone" structs:"phone"`
	Avatar           string                 `json:"avatar" structs:"avatar"`
	UserRoleID       string                 `json:"userRoleId" structs:"userRoleId"`
	OrganizationID   string                 `json:"organizationId" structs:"organizationId"`
	OrganizationName string                 `json:"organizationName" structs:"organizationName"`
	OrganizationRole string                 `json:"organizationRole" structs:"organizationRole"`
	IsSuspended      bool                   `json:"isSuspended" structs:"isSuspended"`
	LastSignInAt     *time.Time             `json:"lastSignInAt" structs:"lastSignInAt"`
	CreatedAt        time.Time              `json:"createdAt" structs:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt" structs:"updatedAt"`
	CustomData       map[string]interface{} `json:"customData" structs:"customData"`
}
