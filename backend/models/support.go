/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import "time"

// SupportSession represents a support tunnel session
type SupportSession struct {
	ID           string     `json:"id"`
	SystemID     string     `json:"system_id"`
	NodeID       *string    `json:"node_id,omitempty"`
	SessionToken string     `json:"session_token,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
	Status       string     `json:"status"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
	ClosedBy     *string    `json:"closed_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Joined system info (populated in list/detail queries)
	SystemName   string        `json:"system_name,omitempty"`
	SystemType   *string       `json:"system_type,omitempty"`
	SystemKey    string        `json:"system_key,omitempty"`
	Organization *Organization `json:"organization,omitempty"`
}

// SupportAccessLog represents an operator's access to a support session
type SupportAccessLog struct {
	ID             string     `json:"id"`
	SessionID      string     `json:"session_id"`
	OperatorID     string     `json:"operator_id"`
	OperatorName   *string    `json:"operator_name,omitempty"`
	AccessType     string     `json:"access_type"`
	ConnectedAt    time.Time  `json:"connected_at"`
	DisconnectedAt *time.Time `json:"disconnected_at,omitempty"`
	Metadata       *string    `json:"metadata,omitempty"`
}

// SystemSessionGroup represents a system with its aggregated support session info
type SystemSessionGroup struct {
	SystemID     string        `json:"system_id"`
	SystemName   string        `json:"system_name"`
	SystemType   *string       `json:"system_type,omitempty"`
	SystemKey    string        `json:"system_key"`
	Organization *Organization `json:"organization,omitempty"`
	StartedAt    time.Time     `json:"started_at"`
	ExpiresAt    time.Time     `json:"expires_at"`
	Status       string        `json:"status"`
	SessionCount int           `json:"session_count"`
	NodeCount    int           `json:"node_count"`
	Sessions     []SessionRef  `json:"sessions"`
}

// SessionRef is a lightweight reference to an individual session within a group
type SessionRef struct {
	ID        string    `json:"id"`
	NodeID    *string   `json:"node_id,omitempty"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ExtendSessionRequest represents a request to extend a session
type ExtendSessionRequest struct {
	Hours int `json:"hours" binding:"required,min=1,max=168"`
}

// AddSessionServiceItem describes a single static service to add to a tunnel
type AddSessionServiceItem struct {
	Name   string `json:"name" binding:"required"`
	Target string `json:"target" binding:"required"`
	Label  string `json:"label"`
	TLS    bool   `json:"tls"`
}

// AddSessionServicesRequest represents a request to dynamically add static services
type AddSessionServicesRequest struct {
	Services []AddSessionServiceItem `json:"services" binding:"required,min=1,max=10"`
}
