/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package services

import (
	"time"
)

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

// PaginatedOrganizations represents a paginated response of organizations
type PaginatedOrganizations struct {
	Data       []LogtoOrganization `json:"data"`
	Pagination PaginationInfo      `json:"pagination"`
}

// OrganizationFilters represents filters for organization queries
type OrganizationFilters struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`       // from customData.type
	CreatedBy   string `json:"created_by,omitempty"` // from customData.createdBy
	Search      string `json:"search,omitempty"`     // general search term
}

// JitRolesCache represents cached JIT roles for an organization
type JitRolesCache struct {
	Roles     []LogtoOrganizationRole `json:"roles"`
	CachedAt  time.Time               `json:"cached_at"`
	ExpiresAt time.Time               `json:"expires_at"`
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

// OrgUsersCache represents cached organization users
type OrgUsersCache struct {
	Users     []LogtoUser `json:"users"`
	CachedAt  time.Time   `json:"cached_at"`
	ExpiresAt time.Time   `json:"expires_at"`
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

// LogtoOrganizationRole represents an organization role from Logto Management API
type LogtoOrganizationRole struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LogtoManagementTokenResponse represents the token response from Logto Management API
type LogtoManagementTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// LogtoUser represents a user/account from Logto Management API
// Note: In our system these are called "accounts" to distinguish from the current logged-in user
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
	IsSuspended   bool                   `json:"isSuspended"`
	HasPassword   bool                   `json:"hasPassword"`
	ApplicationId string                 `json:"applicationId"`
	CreatedAt     int64                  `json:"createdAt"`
	UpdatedAt     int64                  `json:"updatedAt"`
}

// CreateOrganizationRequest represents the request to create a new organization in Logto
type CreateOrganizationRequest struct {
	Name          string                     `json:"name"`
	Description   string                     `json:"description"`
	CustomData    map[string]interface{}     `json:"customData"`
	IsMfaRequired bool                       `json:"isMfaRequired"`
	Branding      *LogtoOrganizationBranding `json:"branding,omitempty"`
}

// UpdateOrganizationRequest represents the request to update an organization in Logto
type UpdateOrganizationRequest struct {
	Name          *string                    `json:"name,omitempty"`
	Description   *string                    `json:"description,omitempty"`
	CustomData    map[string]interface{}     `json:"customData,omitempty"`
	IsMfaRequired *bool                      `json:"isMfaRequired,omitempty"`
	Branding      *LogtoOrganizationBranding `json:"branding,omitempty"`
}

// CreateUserRequest represents the request to create a new account in Logto
type CreateUserRequest struct {
	Username     string                 `json:"username,omitempty"`
	PrimaryEmail string                 `json:"primaryEmail,omitempty"`
	PrimaryPhone *string                `json:"primaryPhone,omitempty"`
	Name         string                 `json:"name,omitempty"`
	Avatar       *string                `json:"avatar,omitempty"`
	CustomData   map[string]interface{} `json:"customData,omitempty"`
	Password     string                 `json:"password,omitempty"`
}

// UpdateUserRequest represents the request to update an account in Logto
type UpdateUserRequest struct {
	Username     *string                `json:"username,omitempty"`
	PrimaryEmail *string                `json:"primaryEmail,omitempty"`
	PrimaryPhone *string                `json:"primaryPhone,omitempty"`
	Name         *string                `json:"name,omitempty"`
	Avatar       *string                `json:"avatar,omitempty"`
	CustomData   map[string]interface{} `json:"customData,omitempty"`
}
