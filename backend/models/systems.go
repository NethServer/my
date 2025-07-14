/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import "time"

// SystemCreator represents the user who created the system
type SystemCreator struct {
	UserID           string `json:"user_id" structs:"user_id"`
	UserName         string `json:"user_name" structs:"user_name"`
	OrganizationID   string `json:"organization_id" structs:"organization_id"`
	OrganizationName string `json:"organization_name" structs:"organization_name"`
}

// System represents a managed system in the infrastructure
type System struct {
	ID          string            `json:"id" structs:"id"`
	Name        string            `json:"name" structs:"name"`
	Type        string            `json:"type" structs:"type"`     // ns8, nsec, etc.
	Status      string            `json:"status" structs:"status"` // online, offline, maintenance
	FQDN        string            `json:"fqdn" structs:"fqdn"`
	IPv4Address string            `json:"ipv4_address" structs:"ipv4_address"`
	IPv6Address string            `json:"ipv6_address" structs:"ipv6_address"`
	Version     string            `json:"version" structs:"version"`
	LastSeen    time.Time         `json:"last_seen" structs:"last_seen"`
	CustomData  map[string]string `json:"custom_data" structs:"custom_data"`
	CustomerID  string            `json:"customer_id" structs:"customer_id"`
	Secret      string            `json:"secret,omitempty" structs:"secret"`           // Only returned during creation
	SecretHash  string            `json:"-" structs:"secret_hash"`                     // Stored in DB, never returned
	SecretHint  string            `json:"secret_hint,omitempty" structs:"secret_hint"` // Last 4 chars for identification
	CreatedAt   time.Time         `json:"created_at" structs:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" structs:"updated_at"`
	CreatedBy   SystemCreator     `json:"created_by" structs:"created_by"`
}

// CreateSystemRequest represents the request payload for creating a new system
type CreateSystemRequest struct {
	Name       string            `json:"name" binding:"required" structs:"name"`
	Type       string            `json:"type" binding:"required" structs:"type"`
	CustomerID string            `json:"customer_id" binding:"required" structs:"customer_id"`
	CustomData map[string]string `json:"custom_data" structs:"custom_data"`
}

// UpdateSystemRequest represents the request payload for updating an existing system
type UpdateSystemRequest struct {
	Name       string            `json:"name" structs:"name"`
	Type       string            `json:"type" structs:"type"`
	CustomerID string            `json:"customer_id" structs:"customer_id"`
	CustomData map[string]string `json:"custom_data" structs:"custom_data"`
}

// SystemSubscription represents subscription information for a system
type SystemSubscription struct {
	SystemID   string    `json:"system_id" structs:"system_id"`
	Plan       string    `json:"plan" structs:"plan"`
	Status     string    `json:"status" structs:"status"` // active, expired, suspended
	StartDate  time.Time `json:"start_date" structs:"start_date"`
	EndDate    time.Time `json:"end_date" structs:"end_date"`
	Features   []string  `json:"features" structs:"features"`
	MaxUsers   int       `json:"max_users" structs:"max_users"`
	MaxStorage int64     `json:"max_storage" structs:"max_storage"` // in bytes
}

// SystemActionRequest represents request for system actions like restart, enable, etc.
type SystemActionRequest struct {
	Force   bool              `json:"force" structs:"force"`
	Options map[string]string `json:"options" structs:"options"`
}
