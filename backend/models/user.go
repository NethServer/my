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
	"strings"
	"time"
)

// OwnerOrgRole is the organization role of the Owner organization: every
// member (the break-glass owner user and the Staff users) holds it. It grants
// global reach (read/manage) over the whole hierarchy; destructive authority
// lives on the "Owner" USER role, never on the organization.
const OwnerOrgRole = "owner"

// Technical (user) roles reserved to the Owner organization's members.
const (
	// OwnerUserRole carries the destroy:* permissions of the technical
	// resources (systems, users) and the authority to manage the Owner
	// organization's own membership (break-glass tier).
	OwnerUserRole = "Owner"
	// StaffUserRole is the non-destructive counterpart for the Nethesis
	// cross-cutting employees.
	StaffUserRole = "Staff"
)

// IsGlobalOrgRole reports whether orgRole grants global reach across the whole
// hierarchy (every distributor/reseller/customer): the organization role of
// the Owner organization. Reach only — destructive authority stays gated on
// the destroy:* permissions of the caller's user role.
func IsGlobalOrgRole(orgRole string) bool {
	return strings.EqualFold(orgRole, OwnerOrgRole)
}

// IsPartnerOrgType reports whether orgType is one of the partner organization
// types (distributor/reseller/customer). The gates protecting the Owner
// organization's membership use it fail-closed: anything else — "owner", an
// empty string from a lookup error, an unknown value — is treated as the Owner
// organization and requires the Owner user role.
func IsPartnerOrgType(orgType string) bool {
	switch strings.ToLower(orgType) {
	case "distributor", "reseller", "customer":
		return true
	}
	return false
}

// HasOwnerUserRole reports whether any of the technical role names is "Owner":
// the tier that manages the Owner organization's own membership and holds the
// destroy:* permissions of the technical resources.
func HasOwnerUserRole(roleNames []string) bool {
	for _, name := range roleNames {
		if strings.EqualFold(name, OwnerUserRole) {
			return true
		}
	}
	return false
}

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
	HasAvatar        bool     `json:"has_avatar" structs:"has_avatar"`               // Whether a profile picture is stored, so clients can skip the avatar request
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
	Active             bool      `json:"-" db:"active"`
}

// EnableConsentRequest represents a request to enable impersonation consent
type EnableConsentRequest struct {
	DurationHours int `json:"duration_hours" binding:"min=0,max=168"` // Max 1 week, defaults to 1 hour if 0 or not provided
}

// ImpersonationAuditEntry represents an action performed during impersonation
type ImpersonationAuditEntry struct {
	ID                   string    `json:"id" db:"id"`
	SessionID            string    `json:"session_id" db:"session_id"`
	ImpersonatorUserID   string    `json:"impersonator_user_id" db:"impersonator_user_id"`
	ImpersonatedUserID   string    `json:"impersonated_user_id" db:"impersonated_user_id"`
	ActionType           string    `json:"action_type" db:"action_type"`                   // "api_call", "session_start", "session_end"
	APIEndpoint          *string   `json:"api_endpoint" db:"api_endpoint"`                 // Only for api_call actions
	HTTPMethod           *string   `json:"http_method" db:"http_method"`                   // Only for api_call actions
	RequestData          *string   `json:"request_data" db:"request_data"`                 // Only for api_call actions
	ResponseStatus       *int      `json:"response_status" db:"response_status"`           // Only for api_call actions
	ResponseStatusText   *string   `json:"response_status_text" db:"response_status_text"` // Only for api_call actions
	Timestamp            time.Time `json:"timestamp" db:"timestamp"`
	ImpersonatorUsername string    `json:"impersonator_username" db:"impersonator_username"`
	ImpersonatedUsername string    `json:"impersonated_username" db:"impersonated_username"`
	ImpersonatorName     string    `json:"impersonator_name" db:"impersonator_name"`
	ImpersonatedName     string    `json:"impersonated_name" db:"impersonated_name"`
}

// ImpersonationSession represents a summary of an impersonation session
type ImpersonationSession struct {
	SessionID            string     `json:"session_id" db:"session_id"`
	ImpersonatorUserID   string     `json:"impersonator_user_id" db:"impersonator_user_id"`
	ImpersonatedUserID   string     `json:"impersonated_user_id" db:"impersonated_user_id"`
	ImpersonatorUsername string     `json:"impersonator_username" db:"impersonator_username"`
	ImpersonatedUsername string     `json:"impersonated_username" db:"impersonated_username"`
	ImpersonatorName     string     `json:"impersonator_name" db:"impersonator_name"`
	ImpersonatedName     string     `json:"impersonated_name" db:"impersonated_name"`
	StartTime            time.Time  `json:"start_time" db:"start_time"`
	EndTime              *time.Time `json:"end_time" db:"end_time"`
	Duration             *int       `json:"duration_minutes" db:"duration_minutes"` // Duration in minutes, null if still active
	ActionCount          int        `json:"action_count" db:"action_count"`         // Number of actions performed in this session
	Status               string     `json:"status"`                                 // "active", "completed"
}
