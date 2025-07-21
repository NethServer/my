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
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
)

// GetSystemsStatus returns the liveness status of systems based on heartbeat data
func GetSystemsStatus(c *gin.Context) {
	// Get query parameter for timeout (default 15 minutes)
	timeoutStr := c.DefaultQuery("timeout", "15")
	timeoutMinutes, err := strconv.Atoi(timeoutStr)
	if err != nil || timeoutMinutes <= 0 {
		logger.Warn().
			Str("component", "heartbeat").
			Str("operation", "parse_timeout").
			Str("timeout", timeoutStr).
			Err(err).
			Msg("invalid timeout format, using default")
		timeoutMinutes = 15
	}

	// Calculate cutoff time for alive/dead determination
	timeout := time.Duration(timeoutMinutes) * time.Minute
	cutoff := time.Now().Add(-timeout)

	// Get all systems from the systems table and LEFT JOIN with heartbeats
	query := `
		SELECT s.id, h.last_heartbeat
		FROM systems s
		LEFT JOIN system_heartbeats h ON s.id = h.system_id
		ORDER BY h.last_heartbeat DESC NULLS LAST, s.id ASC`

	rows, err := database.DB.Query(query)
	if err != nil {
		logger.Error().
			Str("component", "heartbeat").
			Str("operation", "database_query").
			Err(err).
			Msg("failed to query system status")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to query system status", nil))
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Warn().
				Str("component", "heartbeat").
				Str("operation", "close_rows").
				Err(err).
				Msg("failed to close rows")
		}
	}()

	var systems []models.SystemStatus
	aliveCount := 0
	deadCount := 0
	zombieCount := 0

	for rows.Next() {
		var systemID string
		var lastHeartbeat *time.Time

		err := rows.Scan(&systemID, &lastHeartbeat)
		if err != nil {
			logger.Error().
				Str("component", "heartbeat").
				Str("operation", "scan_row").
				Err(err).
				Msg("failed to scan system status row")
			continue
		}

		var status string
		var minutesAgo *int

		if lastHeartbeat == nil {
			// System exists but never sent heartbeat - zombie
			status = "zombie"
			zombieCount++
		} else {
			// Calculate minutes since last heartbeat
			minutes := int(time.Since(*lastHeartbeat).Minutes())
			minutesAgo = &minutes

			if lastHeartbeat.After(cutoff) {
				// System communicated within timeout period - alive
				status = "alive"
				aliveCount++
			} else {
				// System communicated but too long ago - dead
				status = "dead"
				deadCount++
			}
		}

		systems = append(systems, models.SystemStatus{
			SystemID:      systemID,
			LastHeartbeat: lastHeartbeat,
			Status:        status,
			MinutesAgo:    minutesAgo,
		})
	}

	if err := rows.Err(); err != nil {
		logger.Error().
			Str("component", "heartbeat").
			Str("operation", "row_iteration").
			Err(err).
			Msg("error during row iteration")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to process system status", nil))
		return
	}

	// Create summary response (no systems array for efficiency)
	result := map[string]interface{}{
		"total_systems":   len(systems),
		"alive_systems":   aliveCount,
		"dead_systems":    deadCount,
		"zombie_systems":  zombieCount,
		"timeout_minutes": timeoutMinutes,
	}

	logger.Info().
		Str("component", "heartbeat").
		Str("operation", "status_query").
		Int("total_systems", len(systems)).
		Int("alive_systems", aliveCount).
		Int("dead_systems", deadCount).
		Int("zombie_systems", zombieCount).
		Int("timeout_minutes", timeoutMinutes).
		Msg("system status summary retrieved")

	c.JSON(http.StatusOK, response.OK("system status retrieved", result))
}
