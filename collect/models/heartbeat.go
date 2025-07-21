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

// HeartbeatRequest represents the request payload for heartbeat endpoint
type HeartbeatRequest struct {
	SystemID string `json:"system_id" binding:"required"`
}

// HeartbeatResponse represents the response payload for heartbeat endpoint
type HeartbeatResponse struct {
	SystemID      string    `json:"system_id"`
	Acknowledged  bool      `json:"acknowledged"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
}
