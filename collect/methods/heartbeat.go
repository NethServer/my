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
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/queue"
	"github.com/nethesis/my/collect/response"
)

var (
	heartbeatQueueManager     *queue.QueueManager
	heartbeatQueueManagerOnce sync.Once
)

// getHeartbeatQueueManager returns a singleton QueueManager for the heartbeat handler
func getHeartbeatQueueManager() *queue.QueueManager {
	heartbeatQueueManagerOnce.Do(func() {
		heartbeatQueueManager = queue.NewQueueManager()
	})
	return heartbeatQueueManager
}

// ReceiveHeartbeat handles system heartbeat requests - queues for async processing
// Body is optional and ignored - authentication via HTTP Basic Auth is sufficient
func ReceiveHeartbeat(c *gin.Context) {
	authSystemID, ok := getAuthenticatedSystemID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}

	authSystemKey, _ := c.Get("system_key")

	now := time.Now()

	// Enqueue heartbeat for async batch processing
	qm := getHeartbeatQueueManager()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	err := qm.EnqueueHeartbeat(ctx, &models.SystemHeartbeat{
		SystemID:      authSystemID,
		LastHeartbeat: now,
	})
	if err != nil {
		logger.Error().
			Str("component", "heartbeat").
			Str("operation", "enqueue").
			Str("system_key", authSystemKey.(string)).
			Str("system_id", authSystemID).
			Err(err).
			Msg("failed to enqueue heartbeat")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to process heartbeat", nil))
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
