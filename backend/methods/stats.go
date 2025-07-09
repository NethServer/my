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

	"github.com/nethesis/my/backend/background"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services"
)

// GetStats handles GET /api/stats - returns cached system statistics
func GetStats(c *gin.Context) {
	logger.RequestLogger(c, "stats").Info().
		Str("operation", "get_stats").
		Msg("System statistics requested")

	// Get cached statistics from the singleton manager
	cacheManager := background.GetStatsCacheManager()
	stats := cacheManager.GetStats()

	// Check if data is stale and trigger background update if needed
	if stats.IsStale {
		logger.RequestLogger(c, "stats").Info().
			Time("last_updated", stats.LastUpdated).
			Msg("Statistics are stale, triggering background update")

		// Trigger background update (non-blocking)
		cacheManager.TriggerUpdate()
	}

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

	// If no cached data exists (first run), calculate real-time
	if stats.LastUpdated.IsZero() {
		logger.RequestLogger(c, "stats").Info().
			Msg("No cached statistics available, calculating real-time")

		client := services.NewLogtoManagementClient()
		realTimeStats, err := background.CalculateStatsRealTime(client)
		if err != nil {
			logger.NewHTTPErrorLogger(c, "stats").LogError(err, "calculate_realtime_stats", http.StatusInternalServerError, "Failed to calculate real-time statistics")

			// Return cached stats even if stale, or empty stats if no cache
			c.JSON(http.StatusOK, response.OK("system statistics (cached/stale)", statsResponse))
			return
		}

		// Update cache with real-time data
		cacheManager.SetStats(realTimeStats)

		// Return fresh data
		statsResponse = gin.H{
			"distributors": realTimeStats.Distributors,
			"resellers":    realTimeStats.Resellers,
			"customers":    realTimeStats.Customers,
			"users":        realTimeStats.Users,
			"systems":      realTimeStats.Systems,
			"timestamp":    realTimeStats.LastUpdated.Format("2006-01-02T15:04:05Z07:00"),
			"isStale":      false,
		}
	}

	logger.RequestLogger(c, "stats").Info().
		Int("distributors", stats.Distributors).
		Int("resellers", stats.Resellers).
		Int("customers", stats.Customers).
		Int("users", stats.Users).
		Int("systems", stats.Systems).
		Bool("is_stale", stats.IsStale).
		Msg("System statistics retrieved")

	c.JSON(http.StatusOK, response.OK("system statistics", statsResponse))
}
