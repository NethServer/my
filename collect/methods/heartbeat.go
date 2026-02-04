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
// Body is optional and ignored - authentication via HTTP Basic Auth is sufficient
func ReceiveHeartbeat(c *gin.Context) {
	authSystemID, ok := getAuthenticatedSystemID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}

	authSystemKey, _ := c.Get("system_key")

	// Update heartbeat in database - optimized single query (using internal system_id)
	now := time.Now()
	err := updateSystemHeartbeat(authSystemID, now)
	if err != nil {
		logger.Error().
			Str("component", "heartbeat").
			Str("operation", "database_update").
			Str("system_key", authSystemKey.(string)).
			Str("system_id", authSystemID).
			Err(err).
			Msg("failed to update system heartbeat")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to update heartbeat", nil))
		return
	}

	// Minimal response for efficiency
	resp := models.HeartbeatResponse{
		SystemKey:     authSystemKey.(string),
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
