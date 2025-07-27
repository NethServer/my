/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
)

// PreventSelfModification prevents users from performing administrative actions on themselves
func PreventSelfModification() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get current user ID from JWT claims
		currentUserID, exists := c.Get("user_id")
		if !exists {
			logger.Error().
				Str("component", "self_modification_middleware").
				Str("operation", "user_check").
				Bool("success", false).
				Msg("user_id not found in JWT claims")
			c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
			c.Abort()
			return
		}

		// Get target user ID from URL parameter
		targetUserID := c.Param("id")
		if targetUserID == "" {
			logger.Error().
				Str("component", "self_modification_middleware").
				Str("operation", "param_check").
				Bool("success", false).
				Msg("user ID parameter not found in URL")
			c.JSON(http.StatusBadRequest, response.BadRequest("user ID parameter required", nil))
			c.Abort()
			return
		}

		// Prevent self-modification
		if currentUserID == targetUserID {
			logger.Warn().
				Str("component", "self_modification_middleware").
				Str("operation", "self_modification_blocked").
				Str("user_id", currentUserID.(string)).
				Str("target_id", targetUserID).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Bool("success", false).
				Msg("self-modification attempt blocked")
			c.JSON(http.StatusForbidden, response.Forbidden("cannot perform administrative actions on yourself", nil))
			c.Abort()
			return
		}

		logger.Debug().
			Str("component", "self_modification_middleware").
			Str("operation", "access_granted").
			Str("current_user_id", currentUserID.(string)).
			Str("target_user_id", targetUserID).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Bool("success", true).
			Msg("self-modification check passed")

		c.Next()
	}
}
