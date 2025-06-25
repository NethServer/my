/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: GPL-2.0-only
*/

package models

import "time"

// Customer represents a customer in the hierarchy
type Customer struct {
	ID          string            `json:"id" structs:"id"`
	Name        string            `json:"name" structs:"name"`
	Email       string            `json:"email" structs:"email"`
	CompanyName string            `json:"company_name" structs:"company_name"`
	Status      string            `json:"status" structs:"status"` // active, suspended, inactive
	Tier        string            `json:"tier" structs:"tier"`     // basic, premium, enterprise
	ResellerID  string            `json:"reseller_id" structs:"reseller_id"`
	Metadata    map[string]string `json:"metadata" structs:"metadata"`
	CreatedAt   time.Time         `json:"created_at" structs:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" structs:"updated_at"`
	CreatedBy   string            `json:"created_by" structs:"created_by"`
}

// CreateCustomerRequest represents the request payload for creating a new customer
// Aligned with Logto's CreateOrganization API: https://openapi.logto.io/dev/operation/operation-createorganization
type CreateCustomerRequest struct {
	Name          string                 `json:"name" binding:"required" structs:"name"`                   // Organization name (required by Logto)
	Description   string                 `json:"description" structs:"description"`                        // Organization description (optional)
	CustomData    map[string]interface{} `json:"customData" structs:"customData"`                         // Business metadata (email, tier, etc.)
	IsMfaRequired bool                   `json:"isMfaRequired" structs:"isMfaRequired"`                   // MFA requirement (optional, defaults to false)
}

// UpdateCustomerRequest represents the request payload for updating an existing customer
// Aligned with Logto's UpdateOrganization API
type UpdateCustomerRequest struct {
	Name          string                 `json:"name" structs:"name"`                       // Organization name
	Description   string                 `json:"description" structs:"description"`         // Organization description
	CustomData    map[string]interface{} `json:"customData" structs:"customData"`           // Business metadata
	IsMfaRequired *bool                  `json:"isMfaRequired" structs:"isMfaRequired"`     // MFA requirement (pointer for optional update)
}
