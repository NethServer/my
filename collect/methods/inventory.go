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
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/queue"
	"github.com/nethesis/my/collect/response"
)

// CollectInventory handles the POST /api/systems/inventory endpoint
func CollectInventory(c *gin.Context) {
	systemID, exists := c.Get("system_id")
	if !exists {
		logger.Error().Msg("System ID not found in context after authentication")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Authentication context error", nil))
		return
	}

	systemIDStr, ok := systemID.(string)
	if !ok {
		logger.Error().Msg("System ID is not a string")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Authentication context error", nil))
		return
	}

	// Check request size limit
	if c.Request.ContentLength > configuration.Config.APIMaxRequestSize {
		logger.Warn().
			Str("system_id", systemIDStr).
			Int64("content_length", c.Request.ContentLength).
			Int64("max_size", configuration.Config.APIMaxRequestSize).
			Msg("Request size exceeds limit")

		c.JSON(http.StatusRequestEntityTooLarge, response.BadRequest("Request too large", map[string]interface{}{
			"max_size_bytes": configuration.Config.APIMaxRequestSize,
			"received_bytes": c.Request.ContentLength,
		}))
		return
	}

	// Parse request body using the simplified request model
	var inventoryRequest models.InventorySubmissionRequest
	if err := c.ShouldBindJSON(&inventoryRequest); err != nil {
		logger.Warn().
			Err(err).
			Str("system_id", systemIDStr).
			Msg("Failed to parse inventory data")

		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid JSON payload", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Create full InventoryData with auto-populated fields
	now := time.Now()
	inventoryData := models.InventoryData{
		SystemID:  systemIDStr,
		Timestamp: now,
		Data:      inventoryRequest.Data,
	}

	// Validate inventory data
	if err := inventoryData.ValidateInventoryData(); err != nil {
		logger.Warn().
			Err(err).
			Str("system_id", systemIDStr).
			Msg("Inventory data validation failed")

		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid inventory data", map[string]interface{}{
			"validation_error": err.Error(),
		}))
		return
	}

	// Quick validation of JSON structure
	var testData interface{}
	if err := json.Unmarshal(inventoryData.Data, &testData); err != nil {
		logger.Warn().
			Err(err).
			Str("system_id", systemIDStr).
			Msg("Invalid JSON structure in inventory data")

		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid data structure", map[string]interface{}{
			"error": "Data field must contain valid JSON",
		}))
		return
	}

	// Enqueue for processing with detailed timing and aggressive timeout
	start := time.Now()
	queueManager := queue.NewQueueManager()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second) // Very short timeout
	defer cancel()

	if err := queueManager.EnqueueInventory(ctx, &inventoryData); err != nil {
		logger.Error().
			Err(err).
			Str("system_id", systemIDStr).
			Dur("enqueue_duration", time.Since(start)).
			Msg("Failed to enqueue inventory for processing")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to process inventory", map[string]interface{}{
			"error": "Processing queue unavailable",
		}))
		return
	}

	enqueueTime := time.Since(start)

	logger.Info().
		Str("system_id", systemIDStr).
		Time("timestamp", inventoryData.Timestamp).
		Int("data_size", len(inventoryData.Data)).
		Dur("enqueue_time", enqueueTime).
		Msg("Inventory data received and queued for processing")

	// Return success response immediately
	c.JSON(http.StatusAccepted, response.OK("Inventory received and queued for processing", map[string]interface{}{
		"system_id":    systemIDStr,
		"timestamp":    inventoryData.Timestamp,
		"data_size":    len(inventoryData.Data),
		"queue_status": "queued",
		"message":      "Your inventory data has been received and will be processed shortly",
	}))
}
