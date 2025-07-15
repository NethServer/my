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
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/queue"
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

	// System ID is automatically set from authentication context, no validation needed

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

	// Timestamp is automatically set to current server time, no validation needed

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

// GetInventoryStats returns statistics about inventory processing
func GetInventoryStats(c *gin.Context) {
	// This endpoint could be added to provide statistics

	queueManager := queue.NewQueueManager()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	stats, err := queueManager.GetQueueStats(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to get queue statistics")

		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to get statistics", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("Queue statistics", stats))
}

// ValidateInventoryFormat validates the expected format of inventory data
func ValidateInventoryFormat(c *gin.Context) {
	// This endpoint can be used by systems to validate their data format
	// before sending the actual inventory

	var inventoryData models.InventoryData
	if err := c.ShouldBindJSON(&inventoryData); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid JSON payload", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Validate inventory data
	if err := inventoryData.ValidateInventoryData(); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("Validation failed", map[string]interface{}{
			"validation_error": err.Error(),
		}))
		return
	}

	// Additional structure validation
	var testData interface{}
	if err := json.Unmarshal(inventoryData.Data, &testData); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid data structure", map[string]interface{}{
			"error": "Data field must contain valid JSON",
		}))
		return
	}

	c.JSON(http.StatusOK, response.OK("Inventory format is valid", map[string]interface{}{
		"system_id": inventoryData.SystemID,
		"timestamp": inventoryData.Timestamp,
		"data_size": len(inventoryData.Data),
		"status":    "valid",
	}))
}

// GetExpectedFormat returns the expected inventory data format
func GetExpectedFormat(c *gin.Context) {
	expectedFormat := map[string]interface{}{
		"data": map[string]interface{}{
			"description": "object (required) - Complete system inventory data",
			"required_fields": map[string]string{
				"os":         "Operating system information",
				"networking": "Network configuration and status",
				"processors": "CPU information",
				"memory":     "Memory usage statistics",
			},
			"optional_fields": map[string]string{
				"dmi":           "Hardware DMI information",
				"features":      "System features and services",
				"esmithdb":      "System configuration database",
				"rpms":          "Installed packages",
				"kernel":        "Kernel information",
				"timezone":      "System timezone",
				"public_ip":     "Public IP address",
				"virtual":       "Virtualization status",
				"mountpoints":   "Filesystem mount points",
				"system_uptime": "System uptime information",
			},
		},
		"example": map[string]interface{}{
			"data": map[string]interface{}{
				"os": map[string]interface{}{
					"name":   "NethSec",
					"type":   "nethsecurity",
					"family": "OpenWRT",
					"release": map[string]interface{}{
						"full":  "8-24.10.0-ns.1.6.0-5-g0524860a0",
						"major": 7,
					},
				},
				"networking": map[string]interface{}{
					"fqdn": "fw.nethesis.it",
				},
				"processors": map[string]interface{}{
					"count":  "4",
					"models": []string{"Intel(R) Core(TM) i5-4570S CPU @ 2.90GHz"},
				},
				"memory": map[string]interface{}{
					"system": map[string]interface{}{
						"total_bytes":     7352455168,
						"used_bytes":      579198976,
						"available_bytes": 7352455168,
					},
				},
			},
		},
		"authentication": map[string]interface{}{
			"method":   "HTTP Basic Authentication",
			"username": "system_id (automatically populated from authentication)",
			"password": "system_secret (provided during system registration)",
			"note":     "system_id and timestamp are automatically set by the server",
		},
		"response": map[string]interface{}{
			"success": map[string]interface{}{
				"status_code": 202,
				"message":     "Inventory received and queued for processing",
			},
			"error_codes": map[string]interface{}{
				"400": "Invalid JSON payload or validation error",
				"401": "Authentication required or invalid credentials",
				"403": "System ID mismatch",
				"413": "Request entity too large",
				"500": "Internal server error",
			},
		},
		"limits": map[string]interface{}{
			"max_request_size": fmt.Sprintf("%d bytes (%.1f MB)",
				configuration.Config.APIMaxRequestSize,
				float64(configuration.Config.APIMaxRequestSize)/(1024*1024)),
			"timestamp_tolerance": "24 hours in the past, 5 minutes in the future",
			"processing_timeout":  configuration.Config.APIRequestTimeout.String(),
		},
	}

	c.JSON(http.StatusOK, response.OK("Expected inventory format", expectedFormat))
}
