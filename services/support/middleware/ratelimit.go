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

	"github.com/nethesis/my/services/support/logger"
	"github.com/nethesis/my/services/support/response"
)

// NOTE: Rate limiters are in-process (not distributed). If the support service
// is scaled to multiple instances behind a load balancer, rate limits multiply
// by N instances. The support service is designed to run as a single instance
// (stateful tunnel management in memory), so this is acceptable. If horizontal
// scaling is needed, migrate to Redis-based rate limiting with INCR+EXPIRE.

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateLimitEntry
	limit   int
	window  time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		entries: make(map[string]*rateLimitEntry),
		limit:   limit,
		window:  window,
	}
	// Background cleanup of expired entries every window period
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for range ticker.C {
			rl.mu.Lock()
			now := time.Now()
			for key, entry := range rl.entries {
				if now.After(entry.resetAt) {
					delete(rl.entries, key)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.entries[key]
	if !exists || now.After(entry.resetAt) {
		rl.entries[key] = &rateLimitEntry{
			count:   1,
			resetAt: now.Add(rl.window),
		}
		return true
	}

	entry.count++
	return entry.count <= rl.limit
}

// tunnelIPRateLimiter limits tunnel connection attempts per IP
var tunnelIPRateLimiter = newRateLimiter(10, 1*time.Minute)

// tunnelKeyRateLimiter limits tunnel connection attempts per system_key (#14)
var tunnelKeyRateLimiter = newRateLimiter(5, 1*time.Minute)

// sessionRateLimiter limits requests per session ID on internal endpoints
var sessionRateLimiter = newRateLimiter(100, 1*time.Minute)

// TunnelRateLimitMiddleware limits the rate of tunnel connection attempts
// per client IP (10/min) and per system_key (5/min, checked after auth).
func TunnelRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if !tunnelIPRateLimiter.allow(clientIP) {
			logger.Warn().
				Str("client_ip", clientIP).
				Str("path", c.Request.URL.Path).
				Msg("tunnel IP rate limit exceeded")
			c.JSON(http.StatusTooManyRequests, response.Error(http.StatusTooManyRequests, "too many connection attempts", nil))
			c.Abort()
			return
		}
		c.Next()
	}
}

// SessionRateLimitMiddleware limits the rate of requests per session ID on internal endpoints.
func SessionRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("session_id")
		if sessionID == "" {
			c.Next()
			return
		}
		if !sessionRateLimiter.allow(sessionID) {
			logger.Warn().
				Str("session_id", sessionID).
				Str("client_ip", c.ClientIP()).
				Msg("session rate limit exceeded")
			c.JSON(http.StatusTooManyRequests, response.Error(http.StatusTooManyRequests, "too many requests for this session", nil))
			c.Abort()
			return
		}
		c.Next()
	}
}

// SystemKeyRateLimitMiddleware checks the per-system_key rate limit.
// Runs after BasicAuthMiddleware so that system_key is available in the context.
func SystemKeyRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		systemKey, exists := c.Get("system_key")
		if !exists {
			c.Next()
			return
		}
		key := systemKey.(string)
		if !tunnelKeyRateLimiter.allow(key) {
			logger.Warn().
				Str("system_key", key).
				Str("client_ip", c.ClientIP()).
				Msg("tunnel system_key rate limit exceeded")
			c.JSON(http.StatusTooManyRequests, response.Error(http.StatusTooManyRequests, "too many connection attempts for this system", nil))
			c.Abort()
			return
		}
		c.Next()
	}
}
