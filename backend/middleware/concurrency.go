/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
)

// exportConcurrency is how many exports one backend instance runs at once.
// Exports read a whole collection in one query; on 2026-09-23 four of them
// queued by retries pinned the 0.1-CPU production Postgres for eight minutes
// and every other request waited behind them. Two lets a second user export
// while one is in flight; a third gets 429 and the UI can retry.
const exportConcurrency = 2

// exportRetryAfter is what a rejected export is told to wait before retrying.
const exportRetryAfter = 5 * time.Second

var (
	exportLimiterOnce sync.Once
	exportLimiter     gin.HandlerFunc
)

// LimitExports returns the process-wide limiter shared by every export route,
// so the cap applies across collections (a systems export and a customers
// export compete for the same database).
func LimitExports() gin.HandlerFunc {
	exportLimiterOnce.Do(func() {
		exportLimiter = LimitConcurrency("export", exportConcurrency, exportRetryAfter)
	})
	return exportLimiter
}

// LimitConcurrency admits at most max requests through this middleware at the
// same time and answers 429 with a Retry-After header to the others. It never
// queues: a caller that cannot start now is told so immediately, which is
// what keeps a burst of retries from stacking up on the database.
func LimitConcurrency(name string, max int, retryAfter time.Duration) gin.HandlerFunc {
	slots := make(chan struct{}, max)
	retryAfterSeconds := strconv.Itoa(int(retryAfter / time.Second))
	return func(c *gin.Context) {
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
			c.Next()
		default:
			logger.Warn().
				Str("limiter", name).
				Int("max_concurrent", max).
				Str("path", c.FullPath()).
				Msg("concurrency limit reached, rejecting request")
			c.Header("Retry-After", retryAfterSeconds)
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				response.TooManyRequests("too many concurrent "+name+" requests, retry shortly", gin.H{
					"retry_after_seconds": retryAfterSeconds,
				}))
		}
	}
}
