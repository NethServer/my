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
type CreateDistributorRequest struct {
	Name        string            `json:"name" binding:"required" structs:"name"`
	Email       string            `json:"email" binding:"required,email" structs:"email"`
	CompanyName string            `json:"company_name" binding:"required" structs:"company_name"`
	Region      string            `json:"region" binding:"required" structs:"region"`
	Territory   []string          `json:"territory" structs:"territory"`
	Metadata    map[string]string `json:"metadata" structs:"metadata"`
}

// UpdateDistributorRequest represents the request payload for updating an existing distributor
type UpdateDistributorRequest struct {
	Name        string            `json:"name" structs:"name"`
	Email       string            `json:"email" structs:"email"`
	CompanyName string            `json:"company_name" structs:"company_name"`
	Status      string            `json:"status" structs:"status"`
	Region      string            `json:"region" structs:"region"`
	Territory   []string          `json:"territory" structs:"territory"`
	Metadata    map[string]string `json:"metadata" structs:"metadata"`
}
