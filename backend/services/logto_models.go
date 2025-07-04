/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package services

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
