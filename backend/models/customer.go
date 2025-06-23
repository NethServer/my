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
type CreateCustomerRequest struct {
	Name        string            `json:"name" binding:"required" structs:"name"`
	Email       string            `json:"email" binding:"required,email" structs:"email"`
	CompanyName string            `json:"company_name" binding:"required" structs:"company_name"`
	Tier        string            `json:"tier" binding:"required" structs:"tier"`
	ResellerID  string            `json:"reseller_id" structs:"reseller_id"`
	Metadata    map[string]string `json:"metadata" structs:"metadata"`
}

// UpdateCustomerRequest represents the request payload for updating an existing customer
type UpdateCustomerRequest struct {
	Name        string            `json:"name" structs:"name"`
	Email       string            `json:"email" structs:"email"`
	CompanyName string            `json:"company_name" structs:"company_name"`
	Status      string            `json:"status" structs:"status"`
	Tier        string            `json:"tier" structs:"tier"`
	ResellerID  string            `json:"reseller_id" structs:"reseller_id"`
	Metadata    map[string]string `json:"metadata" structs:"metadata"`
}
