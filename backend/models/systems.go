/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import "time"

// Organization represents an organization with its type
type Organization struct {
	ID      string `json:"id" structs:"id"`             // Database UUID
	LogtoID string `json:"logto_id" structs:"logto_id"` // Logto organization ID
	Name    string `json:"name" structs:"name"`
	Type    string `json:"type" structs:"type"` // owner, distributor, reseller, customer
}

// SystemCreator represents the user who created the system
type SystemCreator struct {
	UserID           string `json:"user_id" structs:"user_id"`
	Username         string `json:"username" structs:"username"`
	Name             string `json:"name" structs:"name"`
	Email            string `json:"email" structs:"email"`
	OrganizationID   string `json:"organization_id" structs:"organization_id"`
	OrganizationName string `json:"organization_name" structs:"organization_name"`
}

// System represents a managed system in the infrastructure
type System struct {
	ID               string            `json:"id" structs:"id"`
	Name             string            `json:"name" structs:"name"`
	Type             *string           `json:"type" structs:"type"`     // ns8, nsec, etc. - nullable until first inventory
	Status           string            `json:"status" structs:"status"` // unknown (default), online, offline, deleted
	FQDN             string            `json:"fqdn" structs:"fqdn"`
	IPv4Address      string            `json:"ipv4_address" structs:"ipv4_address"`
	IPv6Address      string            `json:"ipv6_address" structs:"ipv6_address"`
	Version          string            `json:"version" structs:"version"`
	CustomData       map[string]string `json:"custom_data" structs:"custom_data"`
	Organization     Organization      `json:"organization" structs:"organization"` // Organization details with type
	SystemKey        string            `json:"system_key" structs:"system_key"`
	SystemSecret     string            `json:"system_secret,omitempty" structs:"system_secret"` // Returned during creation and regeneration
	Notes            string            `json:"notes" structs:"notes"`
	CreatedAt        time.Time         `json:"created_at" structs:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at" structs:"updated_at"`
	DeletedAt        *time.Time        `json:"deleted_at" structs:"deleted_at"`                   // Soft delete timestamp
	RegisteredAt     *time.Time        `json:"registered_at" structs:"registered_at"`             // Registration timestamp
	SuspendedAt      *time.Time        `json:"suspended_at" structs:"suspended_at"`               // Suspension timestamp
	SuspendedByOrgID *string           `json:"suspended_by_org_id" structs:"suspended_by_org_id"` // Organization that caused cascade suspension
	CreatedBy        SystemCreator     `json:"created_by" structs:"created_by"`
	// Heartbeat status fields
	HeartbeatStatus  string     `json:"heartbeat_status,omitempty"`  // active, inactive, unknown
	LastHeartbeat    *time.Time `json:"last_heartbeat,omitempty"`    // Last heartbeat timestamp
	HeartbeatMinutes *int       `json:"heartbeat_minutes,omitempty"` // Minutes since last heartbeat
}

// IsSuspended returns true if the system is suspended
func (s *System) IsSuspended() bool {
	return s.SuspendedAt != nil
}

// CreateSystemRequest represents the request payload for creating a new system
type CreateSystemRequest struct {
	Name           string            `json:"name" binding:"required" structs:"name"`
	OrganizationID string            `json:"organization_id" binding:"required" structs:"organization_id"`
	CustomData     map[string]string `json:"custom_data" structs:"custom_data"`
	Notes          string            `json:"notes" structs:"notes"`
}

// UpdateSystemRequest represents the request payload for updating an existing system
type UpdateSystemRequest struct {
	Name           string            `json:"name" structs:"name"`
	OrganizationID string            `json:"organization_id" structs:"organization_id"`
	CustomData     map[string]string `json:"custom_data" structs:"custom_data"`
	Notes          string            `json:"notes" structs:"notes"`
}

// RegisterSystemRequest represents the request payload for registering a system
type RegisterSystemRequest struct {
	SystemSecret string `json:"system_secret" binding:"required" structs:"system_secret"`
}

// RegisterSystemResponse represents the response for successful system registration
type RegisterSystemResponse struct {
	SystemKey    string    `json:"system_key" structs:"system_key"`
	RegisteredAt time.Time `json:"registered_at" structs:"registered_at"`
	Message      string    `json:"message" structs:"message"`
}

// TrendDataPoint represents a single data point in a trend chart
type TrendDataPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// TrendResponse represents the trend data for a given period
type TrendResponse struct {
	Period          int              `json:"period"`
	PeriodLabel     string           `json:"period_label"`
	CurrentTotal    int              `json:"current_total"`
	PreviousTotal   int              `json:"previous_total"`
	Delta           int              `json:"delta"`
	DeltaPercentage float64          `json:"delta_percentage"`
	Trend           string           `json:"trend"` // "up", "down", "stable"
	DataPoints      []TrendDataPoint `json:"data_points"`
}
