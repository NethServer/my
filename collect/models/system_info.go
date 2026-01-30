/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

import "time"

// SystemInfo represents the system information returned by the info endpoint
type SystemInfo struct {
	SystemID     string        `json:"system_id"`
	SystemKey    string        `json:"system_key"`
	Name         string        `json:"name"`
	Type         *string       `json:"type"`
	FQDN         *string       `json:"fqdn"`
	Status       string        `json:"status"`
	Suspended    bool          `json:"suspended"`
	SuspendedAt  *time.Time    `json:"suspended_at"`
	Deleted      bool          `json:"deleted"`
	DeletedAt    *time.Time    `json:"deleted_at"`
	Registered   bool          `json:"registered"`
	RegisteredAt *time.Time    `json:"registered_at"`
	CreatedAt    time.Time     `json:"created_at"`
	Organization SystemInfoOrg `json:"organization"`
}

// SystemInfoOrg represents the organization information for a system
type SystemInfoOrg struct {
	ID          string     `json:"id"`
	LogtoID     string     `json:"logto_id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Suspended   bool       `json:"suspended"`
	SuspendedAt *time.Time `json:"suspended_at"`
}
