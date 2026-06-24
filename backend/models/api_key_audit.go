/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

import "time"

// API key audit event types.
const (
	APIKeyEventCreated     = "created"
	APIKeyEventRevoked     = "revoked"
	APIKeyEventAuthFailed  = "auth_failed"
	APIKeyEventRateLimited = "rate_limited"
)

// Reasons attached to an auth_failed event.
const (
	APIKeyReasonRevoked       = "revoked"
	APIKeyReasonExpired       = "expired"
	APIKeyReasonUserInactive  = "user_inactive"
	APIKeyReasonInvalidSecret = "invalid_secret"
)

// APIKeyAuditRecord is the input for recording an audit event. Empty fields are
// stored as NULL.
type APIKeyAuditRecord struct {
	APIKeyID       string
	UserID         string
	OrganizationID string
	Event          string
	Reason         string
	KeyName        string
	KeyMode        string
	IP             string
	Method         string
	Path           string
}

// APIKeyAuditEntry is a stored audit row, returned by the read API.
type APIKeyAuditEntry struct {
	ID             string    `json:"id"`
	APIKeyID       *string   `json:"api_key_id"`
	UserID         *string   `json:"user_id"`
	OrganizationID *string   `json:"organization_id"`
	Event          string    `json:"event"`
	Reason         *string   `json:"reason"`
	KeyName        *string   `json:"key_name"`
	KeyMode        *string   `json:"key_mode"`
	IP             *string   `json:"ip"`
	Method         *string   `json:"method"`
	Path           *string   `json:"path"`
	CreatedAt      time.Time `json:"created_at"`
}
