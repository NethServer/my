/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: GPL-2.0-only
*/

package models

import "time"

// Reseller represents a reseller in the hierarchy
type Reseller struct {
	ID          string            `json:"id" structs:"id"`
	Name        string            `json:"name" structs:"name"`
	Email       string            `json:"email" structs:"email"`
	CompanyName string            `json:"company_name" structs:"company_name"`
	Status      string            `json:"status" structs:"status"` // active, suspended, inactive
	Region      string            `json:"region" structs:"region"`
	Metadata    map[string]string `json:"metadata" structs:"metadata"`
	CreatedAt   time.Time         `json:"created_at" structs:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" structs:"updated_at"`
	CreatedBy   string            `json:"created_by" structs:"created_by"`
}

// CreateResellerRequest represents the request payload for creating a new reseller
// Aligned with Logto's CreateOrganization API: https://openapi.logto.io/dev/operation/operation-createorganization
type CreateResellerRequest struct {
	Name          string                 `json:"name" binding:"required" structs:"name"`                   // Organization name (required by Logto)
	Description   string                 `json:"description" structs:"description"`                        // Organization description (optional)
	CustomData    map[string]interface{} `json:"customData" structs:"customData"`                         // Business metadata (email, region, etc.)
	IsMfaRequired bool                   `json:"isMfaRequired" structs:"isMfaRequired"`                   // MFA requirement (optional, defaults to false)
}

// UpdateResellerRequest represents the request payload for updating an existing reseller
// Aligned with Logto's UpdateOrganization API
type UpdateResellerRequest struct {
	Name          string                 `json:"name" structs:"name"`                       // Organization name
	Description   string                 `json:"description" structs:"description"`         // Organization description
	CustomData    map[string]interface{} `json:"customData" structs:"customData"`           // Business metadata
	IsMfaRequired *bool                  `json:"isMfaRequired" structs:"isMfaRequired"`     // MFA requirement (pointer for optional update)
}
