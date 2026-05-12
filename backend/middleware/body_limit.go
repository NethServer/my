/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/response"
)

// MaxBodySize wraps the request body in http.MaxBytesReader so that handlers
// downstream (typically c.ShouldBindJSON) cannot allocate more than `limit`
// bytes when decoding the body. Use on routes whose payload should never
// exceed a known cap, to prevent memory-exhaustion DoS via crafted oversized
// JSON.
//
// On overflow the handler returns 413 Payload Too Large; subsequent
// middlewares and handlers are skipped via c.Abort.
func MaxBodySize(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

// PayloadTooLarge writes a structured 413 response and aborts. Helper for
// handlers that catch http.MaxBytesReader errors during binding.
func PayloadTooLarge(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, response.Error(http.StatusRequestEntityTooLarge, msg, nil))
}
