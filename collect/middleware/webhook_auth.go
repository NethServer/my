/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/response"
)

// WebhookAuthMiddleware validates the Bearer token in the Authorization header
// against ALERTING_HISTORY_WEBHOOK_SECRET. If the secret is not configured, the
// endpoint rejects all requests to prevent accidental open access.
func WebhookAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := configuration.Config.AlertmanagerWebhookSecret
		if secret == "" {
			logger.Warn().
				Str("client_ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("webhook auth: ALERTING_HISTORY_WEBHOOK_SECRET not configured")
			c.JSON(http.StatusServiceUnavailable, response.Error(http.StatusServiceUnavailable, "webhook authentication not configured", nil))
			c.Abort()
			return
		}

		auth := c.GetHeader("Authorization")
		token, found := strings.CutPrefix(auth, "Bearer ")
		if !found || subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
			logger.Warn().
				Str("client_ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("webhook auth: invalid or missing bearer token")
			c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
			c.Abort()
			return
		}

		c.Next()
	}
}
