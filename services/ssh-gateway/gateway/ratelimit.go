/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package gateway

import (
	"sync"
	"time"
)

// rateLimiter allows a fixed number of events per key in a sliding window
type rateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	events map[string][]time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, events: make(map[string][]time.Time)}
}

func (r *rateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	recent := r.events[key][:0]
	for _, t := range r.events[key] {
		if now.Sub(t) < r.window {
			recent = append(recent, t)
		}
	}
	if len(recent) >= r.limit {
		r.events[key] = recent
		return false
	}
	r.events[key] = append(recent, now)
	// keep the map from growing with keys that went quiet
	if len(r.events) > 10000 {
		for k, ts := range r.events {
			if len(ts) == 0 || now.Sub(ts[len(ts)-1]) >= r.window {
				delete(r.events, k)
			}
		}
	}
	return true
}
