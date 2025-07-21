/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/response"
)

// ReceiveHeartbeat handles system heartbeat requests - optimized for high throughput
func ReceiveHeartbeat(c *gin.Context) {
	var req models.HeartbeatRequest

	// Parse JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request format", nil))
		return
	}

	// Validate system_id
	if req.SystemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system_id is required", nil))
		return
	}

	// Get system_id from basic auth context (set by middleware)
	authSystemID, exists := c.Get("system_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}

	// Verify that the system_id in the request matches the authenticated system
	if req.SystemID != authSystemID.(string) {
		logger.Warn().
			Str("component", "heartbeat").
			Str("operation", "auth_mismatch").
			Str("requested_system_id", req.SystemID).
			Str("authenticated_system_id", authSystemID.(string)).
			Msg("system_id mismatch in heartbeat")
		c.JSON(http.StatusForbidden, response.Forbidden("system_id mismatch", nil))
		return
	}

	// Update heartbeat in database - optimized single query
	now := time.Now()
	err := updateSystemHeartbeat(req.SystemID, now)
	if err != nil {
		logger.Error().
			Str("component", "heartbeat").
			Str("operation", "database_update").
			Str("system_id", req.SystemID).
			Err(err).
			Msg("failed to update system heartbeat")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to update heartbeat", nil))
		return
	}

	// Minimal response for efficiency
	resp := models.HeartbeatResponse{
		SystemID:      req.SystemID,
		Acknowledged:  true,
		LastHeartbeat: now,
	}

	c.JSON(http.StatusOK, response.OK("heartbeat acknowledged", resp))
}

// updateSystemHeartbeat updates or inserts a system heartbeat record
// Optimized for high throughput with UPSERT
func updateSystemHeartbeat(systemID string, timestamp time.Time) error {
	query := `
		INSERT INTO system_heartbeats (system_id, last_heartbeat) 
		VALUES ($1, $2)
		ON CONFLICT (system_id) 
		DO UPDATE SET last_heartbeat = $2`

	_, err := database.DB.Exec(query, systemID, timestamp)
	return err
}
