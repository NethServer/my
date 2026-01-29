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
		// Get current user's Logto ID from JWT claims
		currentUserLogtoID, exists := c.Get("user_logto_id")
		if !exists || currentUserLogtoID == nil {
			logger.Error().
				Str("component", "self_modification_middleware").
				Str("operation", "user_check").
				Bool("success", false).
				Msg("user_logto_id not found in JWT claims")
			c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
			c.Abort()
			return
		}

		// Get target user ID from URL parameter (logto_id)
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

		// Prevent self-modification by comparing Logto IDs
		logtoIDPtr := currentUserLogtoID.(*string)
		if logtoIDPtr != nil && *logtoIDPtr == targetUserID {
			logger.Warn().
				Str("component", "self_modification_middleware").
				Str("operation", "self_modification_blocked").
				Str("user_logto_id", *logtoIDPtr).
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
			Str("target_user_id", targetUserID).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Bool("success", true).
			Msg("self-modification check passed")

		c.Next()
	}
}

// DisableOnImpersonate disables endpoint access during impersonation
func DisableOnImpersonate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if user is in impersonation mode
		isImpersonated, exists := c.Get("is_impersonated")
		if !exists {
			logger.Error().
				Str("component", "impersonation_middleware").
				Str("operation", "impersonation_check").
				Bool("success", false).
				Msg("is_impersonated not found in context")
			c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
			c.Abort()
			return
		}

		// Block self-service operations during impersonation
		if isImpersonated.(bool) {
			// Get impersonator information for logging
			impersonatorID, _ := c.Get("impersonator_id")
			impersonatorUsername, _ := c.Get("impersonator_username")
			currentUserID, _ := c.Get("user_id")
			currentUsername, _ := c.Get("username")

			logger.Warn().
				Str("component", "impersonation_middleware").
				Str("operation", "self_service_blocked").
				Str("impersonator_id", impersonatorID.(string)).
				Str("impersonator_username", impersonatorUsername.(string)).
				Str("impersonated_user_id", currentUserID.(string)).
				Str("impersonated_username", currentUsername.(string)).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Bool("success", false).
				Msg("self-service operation blocked during impersonation")

			c.JSON(http.StatusForbidden, response.Forbidden("self-service operations are not allowed during impersonation", gin.H{
				"reason":  "impersonation_active",
				"message": "self-service operations like changing password or profile are disabled during impersonation for security reasons",
			}))
			c.Abort()
			return
		}

		logger.Debug().
			Str("component", "impersonation_middleware").
			Str("operation", "self_service_allowed").
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Bool("success", true).
			Msg("self-service operation allowed - not in impersonation mode")

		c.Next()
	}
}
