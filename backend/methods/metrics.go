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
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services"
)

// GetMetrics returns comprehensive system metrics
func GetMetrics(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	metricsService := services.NewMetricsService()
	metrics, err := metricsService.GetSystemMetrics(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Str("remote_addr", c.ClientIP()).
			Msg("Failed to collect system metrics")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to collect metrics", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	logger.Info().
		Str("remote_addr", c.ClientIP()).
		Msg("System metrics collected successfully")

	c.JSON(http.StatusOK, response.OK("System metrics retrieved successfully", metrics))
}

// GetHealth returns basic health status
func GetHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	metricsService := services.NewMetricsService()
	health := metricsService.GetHealthStatus(ctx)

	// Determine HTTP status based on health
	status := http.StatusOK
	if healthStatus, ok := health["status"].(string); ok && healthStatus != "healthy" {
		status = http.StatusServiceUnavailable
	}

	logger.Info().
		Str("remote_addr", c.ClientIP()).
		Str("health_status", health["status"].(string)).
		Msg("Health check performed")

	if status == http.StatusOK {
		c.JSON(status, response.OK("Health check completed", health))
	} else {
		c.JSON(status, response.Error(status, "Health check completed", health))
	}
}

// GetDatabaseMetrics returns detailed database metrics
func GetDatabaseMetrics(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	metricsService := services.NewMetricsService()
	metrics, err := metricsService.GetSystemMetrics(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Str("remote_addr", c.ClientIP()).
			Msg("Failed to collect database metrics")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to collect database metrics", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	c.JSON(http.StatusOK, response.OK("Database metrics retrieved successfully", map[string]interface{}{
		"database": metrics.Database,
		"collect":  metrics.Collect,
	}))
}

// GetWorkerMetrics returns worker status and performance metrics
func GetWorkerMetrics(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	metricsService := services.NewMetricsService()
	metrics, err := metricsService.GetSystemMetrics(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Str("remote_addr", c.ClientIP()).
			Msg("Failed to collect worker metrics")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to collect worker metrics", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	c.JSON(http.StatusOK, response.OK("Worker metrics retrieved successfully", map[string]interface{}{
		"workers": metrics.Workers,
		"queues":  metrics.Queues,
	}))
}

// GetSystemsStatus returns status of all systems and recent activity
func GetSystemsStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	metricsService := services.NewMetricsService()
	metrics, err := metricsService.GetSystemMetrics(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Str("remote_addr", c.ClientIP()).
			Msg("Failed to collect systems status")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to collect systems status", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	c.JSON(http.StatusOK, response.OK("Systems status retrieved successfully", map[string]interface{}{
		"systems": metrics.Systems,
		"collect": map[string]interface{}{
			"records_last_24h":      metrics.Collect.RecordsLast24h,
			"diffs_last_24h":        metrics.Collect.DiffsLast24h,
			"pending_notifications": metrics.Collect.PendingNotifications,
		},
	}))
}
