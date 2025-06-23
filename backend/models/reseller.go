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
type CreateResellerRequest struct {
	Name        string            `json:"name" binding:"required" structs:"name"`
	Email       string            `json:"email" binding:"required,email" structs:"email"`
	CompanyName string            `json:"company_name" binding:"required" structs:"company_name"`
	Region      string            `json:"region" binding:"required" structs:"region"`
	Metadata    map[string]string `json:"metadata" structs:"metadata"`
}

// UpdateResellerRequest represents the request payload for updating an existing reseller
type UpdateResellerRequest struct {
	Name        string            `json:"name" structs:"name"`
	Email       string            `json:"email" structs:"email"`
	CompanyName string            `json:"company_name" structs:"company_name"`
	Status      string            `json:"status" structs:"status"`
	Region      string            `json:"region" structs:"region"`
	Metadata    map[string]string `json:"metadata" structs:"metadata"`
}
