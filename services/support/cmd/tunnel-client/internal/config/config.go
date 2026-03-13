/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package config

import (
	"log"
	"os"
	"strings"
	"time"
)

// Defaults -- all overridable via CLI flags or environment variables
const (
	DefaultRedisAddr         = "127.0.0.1:6379"
	DefaultReconnectDelay    = 5 * time.Second
	DefaultMaxReconnect      = 5 * time.Minute
	DefaultDiscoveryInterval = 5 * time.Minute
	DefaultYamuxKeepAlive    = 30 // seconds
	RedisPingTimeout         = 2 * time.Second
)

// ClientConfig holds the runtime configuration for the tunnel client
type ClientConfig struct {
	URL               string
	Key               string
	Secret            string
	NodeID            string
	RedisAddr         string
	StaticServices    string
	Exclude           []string
	ReconnectDelay    time.Duration
	MaxReconnectDelay time.Duration
	DiscoveryInterval time.Duration
	TLSInsecure       bool
}

// ParseExcludePatterns parses a comma-separated string of glob patterns.
// Returns nil if the input is empty.
func ParseExcludePatterns(raw string) []string {
	if raw == "" {
		log.Println("No exclusion patterns configured (set EXCLUDE_PATTERNS or --exclude to filter services)")
		return nil
	}

	var patterns []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			patterns = append(patterns, p)
		}
	}
	log.Printf("Excluding %d service patterns: %v", len(patterns), patterns)
	return patterns
}

// EnvWithDefault returns the value of the environment variable named by key,
// or defaultValue if the variable is not set or empty.
func EnvWithDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// ParseDurationDefault parses a duration string, returning d if s is empty or invalid.
func ParseDurationDefault(s string, d time.Duration) time.Duration {
	if s == "" {
		return d
	}
	if v, err := time.ParseDuration(s); err == nil {
		return v
	}
	return d
}
