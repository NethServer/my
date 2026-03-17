/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package ssh

import (
	"sync"
	"time"
)

// rateLimiter tracks connection counts per key within a rolling time window.
// Counts are reset entirely at each window boundary.
type rateLimiter struct {
	mu     sync.Mutex
	counts map[string]int
	window time.Duration
	limit  int
}

// newRateLimiter creates a rate limiter that allows at most limit connections
// per key within each time window. A background goroutine resets counts at
// every window interval.
func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		counts: make(map[string]int),
		window: window,
		limit:  limit,
	}
	go rl.cleanup()
	return rl
}

// Allow returns true if the key has not exceeded the rate limit in the
// current window. If allowed, the count for the key is incremented.
func (rl *rateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.counts[key] >= rl.limit {
		return false
	}
	rl.counts[key]++
	return true
}

// cleanup resets all counters at each window interval
func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		rl.counts = make(map[string]int)
		rl.mu.Unlock()
	}
}
