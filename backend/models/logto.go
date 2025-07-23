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

// LogtoManagementTokenResponse represents the response from Logto Management API token endpoint
type LogtoManagementTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// LogtoUserInfo represents the user info returned by Logto
type LogtoUserInfo struct {
	Sub              string   `json:"sub"`
	Username         string   `json:"username"`
	Email            string   `json:"email"`
	Name             string   `json:"name"`
	Roles            []string `json:"roles"`
	OrganizationId   string   `json:"organization_id"`
	OrganizationName string   `json:"organization_name"`
	// Add other fields as needed
}

// LogtoRole represents a role from Logto Management API
type LogtoRole struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// LogtoScope represents a scope/permission from Logto Management API
type LogtoScope struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ResourceID  string `json:"resourceId"`
}

// LogtoOrganization represents an organization from Logto Management API
type LogtoOrganization struct {
	ID            string                     `json:"id"`
	Name          string                     `json:"name"`
	Description   string                     `json:"description"`
	CustomData    map[string]interface{}     `json:"customData"`
	IsMfaRequired bool                       `json:"isMfaRequired"`
	Branding      *LogtoOrganizationBranding `json:"branding"`
}

// LogtoOrganizationRole represents an organization role from Logto Management API
type LogtoOrganizationRole struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	IsDefault   bool   `json:"isDefault"`
}

// LogtoUser represents a user from Logto Management API
type LogtoUser struct {
	ID            string                 `json:"id"`
	Username      string                 `json:"username"`
	PrimaryEmail  string                 `json:"primaryEmail"`
	PrimaryPhone  string                 `json:"primaryPhone"`
	Name          string                 `json:"name"`
	Avatar        string                 `json:"avatar"`
	CustomData    map[string]interface{} `json:"customData"`
	Identities    map[string]interface{} `json:"identities"`
	LastSignInAt  *int64                 `json:"lastSignInAt"`
	CreatedAt     int64                  `json:"createdAt"`
	UpdatedAt     int64                  `json:"updatedAt"`
	Profile       map[string]interface{} `json:"profile"`
	ApplicationId string                 `json:"applicationId"`
	IsSuspended   bool                   `json:"isSuspended"`
	HasPassword   bool                   `json:"hasPassword"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	Page       int  `json:"page"`
	PageSize   int  `json:"page_size"`
	TotalCount int  `json:"total_count"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
	NextPage   *int `json:"next_page,omitempty"`
	PrevPage   *int `json:"prev_page,omitempty"`
}

// OrganizationFilters represents filters for organization queries
type OrganizationFilters struct {
	Name        string `json:"name,omitempty"`        // exact match
	Description string `json:"description,omitempty"` // from customData.type
	Type        string `json:"type,omitempty"`        // from customData.type
	CreatedBy   string `json:"created_by,omitempty"`  // from customData.createdBy
	Search      string `json:"search,omitempty"`      // general search term
}

// PaginatedOrganizations represents a paginated response of organizations
type PaginatedOrganizations struct {
	Data       []LogtoOrganization `json:"data"`
	Pagination PaginationInfo      `json:"pagination"`
}

// JitRolesResult represents the result of a parallel JIT roles fetch
type JitRolesResult struct {
	OrgID string                  `json:"org_id"`
	Roles []LogtoOrganizationRole `json:"roles"`
	Error error                   `json:"error,omitempty"`
}

// PaginatedUsers represents a paginated response of users
type PaginatedUsers struct {
	Data       []LogtoUser    `json:"data"`
	Pagination PaginationInfo `json:"pagination"`
}

// UserFilters represents filters for user queries
type UserFilters struct {
	Search         string `json:"search,omitempty"`          // general search term
	OrganizationID string `json:"organization_id,omitempty"` // filter by organization
	Role           string `json:"role,omitempty"`            // filter by user role
	Username       string `json:"username,omitempty"`        // filter by username
	Email          string `json:"email,omitempty"`           // filter by email
}

// OrgUsersResult represents the result of a parallel organization users fetch
type OrgUsersResult struct {
	OrgID string      `json:"org_id"`
	Users []LogtoUser `json:"users"`
	Error error       `json:"error,omitempty"`
}

// UsersCache represents cached user list
type UsersCache struct {
	Users     []LogtoUser `json:"users"`
	CachedAt  time.Time   `json:"cached_at"`
	ExpiresAt time.Time   `json:"expires_at"`
}

// LogtoOrganizationBranding represents organization branding settings
type LogtoOrganizationBranding struct {
	LogoUrl     string `json:"logoUrl"`
	DarkLogoUrl string `json:"darkLogoUrl"`
	Favicon     string `json:"favicon"`
	DarkFavicon string `json:"darkFavicon"`
}

// Request structures for API calls

// CreateOrganizationRequest represents a request to create an organization
type CreateOrganizationRequest struct {
	Name          string                     `json:"name" binding:"required"`
	Description   string                     `json:"description,omitempty"`
	CustomData    map[string]interface{}     `json:"customData,omitempty"`
	IsMfaRequired bool                       `json:"isMfaRequired"`
	Branding      *LogtoOrganizationBranding `json:"branding,omitempty"`
}

// UpdateOrganizationRequest represents a request to update an organization
type UpdateOrganizationRequest struct {
	Name          *string                `json:"name,omitempty"`
	Description   *string                `json:"description,omitempty"`
	CustomData    map[string]interface{} `json:"customData,omitempty"`
	IsMfaRequired *bool                  `json:"isMfaRequired,omitempty"`
}

// CreateOrganizationJitRolesRequest represents a request to configure JIT roles for an organization
type CreateOrganizationJitRolesRequest struct {
	OrganizationRoleIds []string `json:"organizationRoleIds"`
}

// CreateUserRequest represents a request to create a user
type CreateUserRequest struct {
	Username     string                 `json:"username" binding:"required"`
	Password     string                 `json:"password" binding:"required"`
	Name         string                 `json:"name" binding:"required"`
	PrimaryEmail string                 `json:"primaryEmail"`
	PrimaryPhone string                 `json:"primaryPhone,omitempty"`
	Avatar       *string                `json:"avatar,omitempty"`
	CustomData   map[string]interface{} `json:"customData,omitempty"`
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	Username     *string                `json:"username,omitempty"`
	Name         *string                `json:"name,omitempty"`
	PrimaryEmail *string                `json:"primaryEmail,omitempty"`
	PrimaryPhone *string                `json:"primaryPhone,omitempty"`
	Avatar       *string                `json:"avatar,omitempty"`
	CustomData   map[string]interface{} `json:"customData,omitempty"`
	IsSuspended  *bool                  `json:"isSuspended,omitempty"`
}
