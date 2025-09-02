/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

import "time"

type User struct {
	ID               string   `json:"id" structs:"id"`                               // Local database ID
	LogtoID          *string  `json:"logto_id,omitempty" structs:"logto_id"`         // Logto ID for reference
	Username         string   `json:"username" structs:"username"`                   // Username from Logto
	Email            string   `json:"email" structs:"email"`                         // Email from Logto
	Name             string   `json:"name" structs:"name"`                           // Display name from Logto
	Phone            *string  `json:"phone" structs:"phone"`                         // Phone number from Logto
	UserRoles        []string `json:"user_roles" structs:"user_roles"`               // Technical capabilities (Admin, Support)
	UserRoleIDs      []string `json:"user_role_ids" structs:"user_role_ids"`         // Role IDs for technical capabilities
	UserPermissions  []string `json:"user_permissions" structs:"user_permissions"`   // Permissions derived from user roles
	OrgRole          string   `json:"org_role" structs:"org_role"`                   // Business hierarchy role (Owner, Distributor, Reseller, Customer)
	OrgRoleID        string   `json:"org_role_id" structs:"org_role_id"`             // Organization role ID
	OrgPermissions   []string `json:"org_permissions" structs:"org_permissions"`     // Permissions derived from organization role
	OrganizationID   string   `json:"organization_id" structs:"organization_id"`     // Which organization the user belongs to
	OrganizationName string   `json:"organization_name" structs:"organization_name"` // Organization name for display
}

// ChangePasswordRequest represents a request to change the current user's password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangeInfoRequest represents a request to change the current user's personal information
type ChangeInfoRequest struct {
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone,omitempty"`
}

// ImpersonationConsent represents a user's consent to be impersonated
type ImpersonationConsent struct {
	ID                 string    `json:"id" db:"id"`
	UserID             string    `json:"user_id" db:"user_id"`
	ExpiresAt          time.Time `json:"expires_at" db:"expires_at"`
	MaxDurationMinutes int       `json:"max_duration_minutes" db:"max_duration_minutes"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	Active             bool      `json:"active" db:"active"`
}

// EnableConsentRequest represents a request to enable impersonation consent
type EnableConsentRequest struct {
	DurationHours int `json:"duration_hours" binding:"required,min=1,max=168"` // Max 1 week
}

// ImpersonationAuditEntry represents an action performed during impersonation
type ImpersonationAuditEntry struct {
	ID                   string    `json:"id" db:"id"`
	SessionID            string    `json:"session_id" db:"session_id"`
	ImpersonatorUserID   string    `json:"impersonator_user_id" db:"impersonator_user_id"`
	ImpersonatedUserID   string    `json:"impersonated_user_id" db:"impersonated_user_id"`
	ActionType           string    `json:"action_type" db:"action_type"`         // "api_call", "session_start", "session_end"
	APIEndpoint          *string   `json:"api_endpoint" db:"api_endpoint"`       // Only for api_call actions
	HTTPMethod           *string   `json:"http_method" db:"http_method"`         // Only for api_call actions
	RequestData          *string   `json:"request_data" db:"request_data"`       // Only for api_call actions
	ResponseStatus       *int      `json:"response_status" db:"response_status"` // Only for api_call actions
	Timestamp            time.Time `json:"timestamp" db:"timestamp"`
	ImpersonatorUsername string    `json:"impersonator_username" db:"impersonator_username"`
	ImpersonatedUsername string    `json:"impersonated_username" db:"impersonated_username"`
}
