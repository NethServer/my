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

// APIKey is a personal API key a user issues for non-interactive integrations.
// The secret part of the token is never stored or returned; only the public
// part (for lookup and display) is kept in clear.
type APIKey struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	OrganizationID string     `json:"organization_id"`
	Name           string     `json:"name"`
	KeyPublic      string     `json:"key_public"`
	Mode           string     `json:"mode"` // read | write
	ExpiresAt      time.Time  `json:"expires_at"`
	LastUsedAt     *time.Time `json:"last_used_at"`
	LastUsedIP     *string    `json:"last_used_ip"`
	RevokedAt      *time.Time `json:"revoked_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// CreateAPIKeyRequest is the body for POST /me/api-keys.
type CreateAPIKeyRequest struct {
	Name          string `json:"name" binding:"required"`
	Mode          string `json:"mode" binding:"required,oneof=read write"`
	ExpiresInDays int    `json:"expires_in_days"` // optional; default 90, capped at 365
}

// CreateAPIKeyResponse returns the freshly minted key. The plaintext token is
// shown exactly once and never stored or returned again.
type CreateAPIKeyResponse struct {
	APIKey
	Token string `json:"token"` // myk_<public>.<secret> — shown once, store it now
}
