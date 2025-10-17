/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import "time"

// SystemCreator represents the user who created the system
type SystemCreator struct {
	UserID           string `json:"user_id" structs:"user_id"`
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
	Status           *string           `json:"status" structs:"status"` // online, offline, maintenance - nullable until first inventory
	FQDN             string            `json:"fqdn" structs:"fqdn"`
	IPv4Address      string            `json:"ipv4_address" structs:"ipv4_address"`
	IPv6Address      string            `json:"ipv6_address" structs:"ipv6_address"`
	Version          string            `json:"version" structs:"version"`
	CustomData       map[string]string `json:"custom_data" structs:"custom_data"`
	OrganizationID   string            `json:"organization_id" structs:"organization_id"`
	OrganizationName string            `json:"organization_name" structs:"organization_name"`
	SystemKey        string            `json:"system_key" structs:"system_key"`
	SystemSecret     string            `json:"system_secret,omitempty" structs:"system_secret"` // Returned during creation and regeneration
	Notes            string            `json:"notes" structs:"notes"`
	CreatedAt        time.Time         `json:"created_at" structs:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at" structs:"updated_at"`
	DeletedAt        *time.Time        `json:"deleted_at" structs:"deleted_at"` // Soft delete timestamp
	CreatedBy        SystemCreator     `json:"created_by" structs:"created_by"`
	// Heartbeat status fields
	HeartbeatStatus  string     `json:"heartbeat_status,omitempty"`  // alive, dead, zombie
	LastHeartbeat    *time.Time `json:"last_heartbeat,omitempty"`    // Last heartbeat timestamp
	HeartbeatMinutes *int       `json:"heartbeat_minutes,omitempty"` // Minutes since last heartbeat
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
