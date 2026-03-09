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
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/response"
)

type rateLimiterEntry struct {
	tokens    float64
	lastCheck time.Time
}

// RateLimit returns a middleware that limits requests per IP using a token bucket algorithm.
// rate is tokens added per second, burst is the maximum tokens (bucket capacity).
func RateLimit(rate float64, burst int) gin.HandlerFunc {
	var mu sync.Mutex
	clients := make(map[string]*rateLimiterEntry)

	// Background cleanup of stale entries every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			now := time.Now()
			for ip, entry := range clients {
				if now.Sub(entry.lastCheck) > 10*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		entry, exists := clients[ip]
		now := time.Now()

		if !exists {
			entry = &rateLimiterEntry{
				tokens:    float64(burst),
				lastCheck: now,
			}
			clients[ip] = entry
		}

		// Refill tokens based on elapsed time
		elapsed := now.Sub(entry.lastCheck).Seconds()
		entry.tokens += elapsed * rate
		if entry.tokens > float64(burst) {
			entry.tokens = float64(burst)
		}
		entry.lastCheck = now

		if entry.tokens < 1 {
			mu.Unlock()
			c.JSON(http.StatusTooManyRequests, response.TooManyRequests("rate limit exceeded", nil))
			c.Abort()
			return
		}

		entry.tokens--
		mu.Unlock()

		c.Next()
	}
}
