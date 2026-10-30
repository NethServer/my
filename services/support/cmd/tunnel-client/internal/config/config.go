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
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"
)

// Defaults -- all overridable via CLI flags or environment variables
const (
	DefaultRedisAddr                = "127.0.0.1:6379"
	DefaultReconnectDelay           = 5 * time.Second
	DefaultMaxReconnect             = 5 * time.Minute
	DefaultDiscoveryInterval        = 5 * time.Minute
	DefaultYamuxKeepAlive           = 30 // seconds
	RedisPingTimeout                = 2 * time.Second
	DefaultDiagnosticsPluginTimeout = 10 * time.Second
	DefaultDiagnosticsTotalTimeout  = 30 * time.Second
	DefaultUsersDir                 = "/usr/share/my/users.d"
	DefaultUsersPluginTimeout       = 15 * time.Second
	DefaultUsersTotalTimeout        = 60 * time.Second
	DefaultUsersStateFile           = "/var/lib/my-support/users-state.json"
)

// ClientConfig holds the runtime configuration for the tunnel client
type ClientConfig struct {
	URL                      string
	Key                      string
	Secret                   string
	NodeID                   string
	RedisAddr                string
	StaticServices           string
	Exclude                  []string
	ReconnectDelay           time.Duration
	MaxReconnectDelay        time.Duration
	DiscoveryInterval        time.Duration
	TLSInsecure              bool
	Dev                      bool
	DiagnosticsDir           string
	DiagnosticsPluginTimeout time.Duration
	DiagnosticsTotalTimeout  time.Duration
	UsersDir                 string
	UsersPluginTimeout       time.Duration
	UsersTotalTimeout        time.Duration
	UsersStateFile           string
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

// ValidateTransport refuses anything but wss:// and refuses skipping TLS
// verification, unless dev mode is explicitly enabled.
func ValidateTransport(rawURL string, tlsInsecure, dev bool) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid URL %q: missing host", rawURL)
	}
	switch u.Scheme {
	case "wss":
	case "ws":
		if !dev {
			return fmt.Errorf("refusing plaintext ws:// URL (use wss://, or --dev for local development)")
		}
	default:
		return fmt.Errorf("unsupported URL scheme %q (use wss://)", u.Scheme)
	}
	if tlsInsecure && !dev {
		return fmt.Errorf("--tls-insecure requires --dev")
	}
	return nil
}

// IsTruthy reports whether an environment value enables a boolean option.
func IsTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
