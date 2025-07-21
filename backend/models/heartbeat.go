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

// SystemHeartbeat represents a system heartbeat record
type SystemHeartbeat struct {
	SystemID      string    `json:"system_id" db:"system_id"`
	LastHeartbeat time.Time `json:"last_heartbeat" db:"last_heartbeat"`
}

// SystemStatus represents the liveness status of a system
type SystemStatus struct {
	SystemID      string     `json:"system_id"`
	LastHeartbeat *time.Time `json:"last_heartbeat,omitempty"`
	Status        string     `json:"status"` // "alive", "dead", "zombie"
	MinutesAgo    *int       `json:"minutes_ago,omitempty"`
}

// SystemsStatusSummary represents the overall status summary
type SystemsStatusSummary struct {
	TotalSystems   int            `json:"total_systems"`
	AliveSystems   int            `json:"alive_systems"`
	DeadSystems    int            `json:"dead_systems"`
	ZombieSystems  int            `json:"zombie_systems"`
	TimeoutMinutes int            `json:"timeout_minutes"`
	Systems        []SystemStatus `json:"systems"`
}
