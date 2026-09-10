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

	"github.com/nethesis/my/backend/helpers"
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

// UserRateLimit throttles per authenticated user rather than per IP: an office
// behind one NAT is many dashboards, while one user with many tabs open is
// still one dashboard. Requests carrying no user identity fall back to the
// client IP so an unauthenticated path is still bounded.
func UserRateLimit(rate float64, burst int) gin.HandlerFunc {
	var mu sync.Mutex
	users := make(map[string]*rateLimiterEntry)

	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			now := time.Now()
			for id, entry := range users {
				if now.Sub(entry.lastCheck) > 10*time.Minute {
					delete(users, id)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		key, _, _, _ := helpers.GetUserContextExtended(c)
		if key == "" {
			key = c.ClientIP()
		}

		mu.Lock()
		entry, exists := users[key]
		now := time.Now()

		if !exists {
			entry = &rateLimiterEntry{tokens: float64(burst), lastCheck: now}
			users[key] = entry
		}

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
