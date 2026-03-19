/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

import (
	"encoding/json"
	"time"
)

// SupportSession represents a support tunnel session
type SupportSession struct {
	ID             string           `json:"id"`
	SystemID       string           `json:"system_id"`
	NodeID         string           `json:"node_id,omitempty"`
	SessionToken   string           `json:"session_token,omitempty"`
	ReconnectToken string           `json:"reconnect_token,omitempty"`
	StartedAt      time.Time        `json:"started_at"`
	ExpiresAt      time.Time        `json:"expires_at"`
	Status         string           `json:"status"`
	ClosedAt       *time.Time       `json:"closed_at,omitempty"`
	ClosedBy       *string          `json:"closed_by,omitempty"`
	Diagnostics    *json.RawMessage `json:"diagnostics,omitempty"`
	DiagnosticsAt  *time.Time       `json:"diagnostics_at,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}
