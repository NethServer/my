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

// NonceData stores the SSH auth nonce information in Redis
type NonceData struct {
	SystemKey string    `json:"system_key"`
	CreatedAt time.Time `json:"created_at"`
}

// AuthResult stores the SSH auth result from the backend in Redis
type AuthResult struct {
	UserID           string `json:"user_id"`
	Username         string `json:"username"`
	UserEmail        string `json:"user_email"`
	OrganizationName string `json:"organization_name"`
	SessionID        string `json:"session_id"`
	SystemID         string `json:"system_id"`
	SystemName       string `json:"system_name"`
	SystemType       string `json:"system_type"`
	NodeID           string `json:"node_id,omitempty"`
}
