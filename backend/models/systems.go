/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: GPL-2.0-only
*/

package models

import "time"

// System represents a managed system in the infrastructure
type System struct {
	ID        string            `json:"id" structs:"id"`
	Name      string            `json:"name" structs:"name"`
	Type      string            `json:"type" structs:"type"`     // linux, windows, etc.
	Status    string            `json:"status" structs:"status"` // online, offline, maintenance
	IPAddress string            `json:"ip_address" structs:"ip_address"`
	Version   string            `json:"version" structs:"version"`
	LastSeen  time.Time         `json:"last_seen" structs:"last_seen"`
	Metadata  map[string]string `json:"metadata" structs:"metadata"`
	CreatedAt time.Time         `json:"created_at" structs:"created_at"`
	UpdatedAt time.Time         `json:"updated_at" structs:"updated_at"`
	CreatedBy string            `json:"created_by" structs:"created_by"`
}

// CreateSystemRequest represents the request payload for creating a new system
type CreateSystemRequest struct {
	Name      string            `json:"name" binding:"required" structs:"name"`
	Type      string            `json:"type" binding:"required" structs:"type"`
	IPAddress string            `json:"ip_address" binding:"required" structs:"ip_address"`
	Version   string            `json:"version" structs:"version"`
	Metadata  map[string]string `json:"metadata" structs:"metadata"`
}

// UpdateSystemRequest represents the request payload for updating an existing system
type UpdateSystemRequest struct {
	Name      string            `json:"name" structs:"name"`
	Type      string            `json:"type" structs:"type"`
	Status    string            `json:"status" structs:"status"`
	IPAddress string            `json:"ip_address" structs:"ip_address"`
	Version   string            `json:"version" structs:"version"`
	Metadata  map[string]string `json:"metadata" structs:"metadata"`
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
