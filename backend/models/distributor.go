/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: GPL-2.0-only
*/

package models

import "time"

// Distributor represents a distributor in the hierarchy
type Distributor struct {
	ID          string            `json:"id" structs:"id"`
	Name        string            `json:"name" structs:"name"`
	Email       string            `json:"email" structs:"email"`
	CompanyName string            `json:"company_name" structs:"company_name"`
	Status      string            `json:"status" structs:"status"` // active, suspended, inactive
	Region      string            `json:"region" structs:"region"`
	Territory   []string          `json:"territory" structs:"territory"` // Countries/regions covered
	Metadata    map[string]string `json:"metadata" structs:"metadata"`
	CreatedAt   time.Time         `json:"created_at" structs:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" structs:"updated_at"`
	CreatedBy   string            `json:"created_by" structs:"created_by"`
}

// CreateDistributorRequest represents the request payload for creating a new distributor
// Aligned with Logto's CreateOrganization API: https://openapi.logto.io/dev/operation/operation-createorganization
type CreateDistributorRequest struct {
	Name          string                 `json:"name" binding:"required" structs:"name"`                   // Organization name (required by Logto)
	Description   string                 `json:"description" structs:"description"`                        // Organization description (optional)
	CustomData    map[string]interface{} `json:"customData" structs:"customData"`                         // Business metadata (email, region, etc.)
	IsMfaRequired bool                   `json:"isMfaRequired" structs:"isMfaRequired"`                   // MFA requirement (optional, defaults to false)
}

// UpdateDistributorRequest represents the request payload for updating an existing distributor
// Aligned with Logto's UpdateOrganization API
type UpdateDistributorRequest struct {
	Name          string                 `json:"name" structs:"name"`                       // Organization name
	Description   string                 `json:"description" structs:"description"`         // Organization description
	CustomData    map[string]interface{} `json:"customData" structs:"customData"`           // Business metadata
	IsMfaRequired *bool                  `json:"isMfaRequired" structs:"isMfaRequired"`     // MFA requirement (pointer for optional update)
}
