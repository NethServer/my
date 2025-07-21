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

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
)

// GetStats handles GET /api/stats - returns cached system statistics
func GetStats(c *gin.Context) {
	logger.RequestLogger(c, "stats").Info().
		Str("operation", "get_stats").
		Msg("System statistics requested")

	// Get cached statistics from the singleton manager
	cacheManager := cache.GetStatsCacheManager()
	stats := cacheManager.GetStats()

	// Convert to response format
	statsResponse := gin.H{
		"distributors": stats.Distributors,
		"resellers":    stats.Resellers,
		"customers":    stats.Customers,
		"users":        stats.Users,
		"systems":      stats.Systems,
		"timestamp":    stats.LastUpdated.Format("2006-01-02T15:04:05Z07:00"),
		"isStale":      stats.IsStale,
	}

	// If no cached data exists (first run), return what we have
	// The background updater will populate the cache automatically
	if stats.LastUpdated.IsZero() {
		logger.RequestLogger(c, "stats").Info().
			Msg("No cached statistics available, returning empty stats")

		// Return empty stats, background updater will populate cache
		statsResponse = gin.H{
			"distributors": 0,
			"resellers":    0,
			"customers":    0,
			"users":        0,
			"systems": gin.H{
				"total":  0,
				"alive":  0,
				"dead":   0,
				"zombie": 0,
			},
			"timestamp": "",
			"isStale":   true,
		}
	}

	logger.RequestLogger(c, "stats").Info().
		Int("distributors", stats.Distributors).
		Int("resellers", stats.Resellers).
		Int("customers", stats.Customers).
		Int("users", stats.Users).
		Int("systems_total", stats.Systems.Total).
		Int("systems_alive", stats.Systems.Alive).
		Int("systems_dead", stats.Systems.Dead).
		Int("systems_zombie", stats.Systems.Zombie).
		Bool("is_stale", stats.IsStale).
		Msg("System statistics retrieved")

	c.JSON(http.StatusOK, response.OK("system statistics", statsResponse))
}
