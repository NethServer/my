/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package configuration

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nethesis/my/services/support/logger"
)

// Configuration holds all service configuration
type Configuration struct {
	ListenAddress string `json:"listen_address"`

	// Database configuration
	DatabaseURL string `json:"database_url"`

	// Redis configuration
	RedisURL      string `json:"redis_url"`
	RedisDB       int    `json:"redis_db"`
	RedisPassword string `json:"redis_password"`

	// System authentication configuration
	SystemAuthCacheTTL    time.Duration `json:"system_auth_cache_ttl"`
	SystemSecretMinLength int           `json:"system_secret_min_length"`

	// Session configuration
	SessionDefaultDuration time.Duration `json:"session_default_duration"`
	SessionCleanerInterval time.Duration `json:"session_cleaner_interval"`

	// Tunnel configuration
	TunnelGracePeriod    time.Duration `json:"tunnel_grace_period"`
	MaxTunnels           int           `json:"max_tunnels"`
	MaxSessionsPerSystem int           `json:"max_sessions_per_system"`
	MaxStreamsPerTunnel  int           `json:"max_streams_per_tunnel"`

	// Terminal configuration
	TerminalInactivityTimeout time.Duration `json:"terminal_inactivity_timeout"`
	TerminalMaxFrameSize      int           `json:"terminal_max_frame_size"`

	// Rate limiting configuration
	RateLimitTunnelPerIP   int           `json:"rate_limit_tunnel_per_ip"`
	RateLimitTunnelPerKey  int           `json:"rate_limit_tunnel_per_key"`
	RateLimitSessionPerID  int           `json:"rate_limit_session_per_id"`
	RateLimitAuthFailPerIP int           `json:"rate_limit_auth_fail_per_ip"`
	RateLimitWindow        time.Duration `json:"rate_limit_window"`

	// Proxies whose X-Forwarded-For is trusted for the client IP
	TrustedProxies []string `json:"trusted_proxies"`

	// Cross-service plumbing — must match backend's APP_ENV / INTERNAL_HMAC_SECRET
	AppEnv             string `json:"app_env"`
	InternalHMACSecret string `json:"-"`

	// Internal authentication (shared secret with backend)
	InternalSecret string `json:"-"`
}

// Config is the global configuration instance
var Config = Configuration{}

// Init initializes configuration from environment variables
func Init() {
	Config.ListenAddress = getStringWithDefault("LISTEN_ADDRESS", "127.0.0.1:8082")

	// Database configuration
	if os.Getenv("DATABASE_URL") != "" {
		Config.DatabaseURL = os.Getenv("DATABASE_URL")
	} else {
		logger.LogConfigLoad("env", "DATABASE_URL", false, fmt.Errorf("DATABASE_URL variable is empty"))
	}

	// Redis configuration
	Config.RedisURL = getStringWithDefault("REDIS_URL", "redis://localhost:6379")
	Config.RedisDB = parseIntWithDefault("REDIS_DB", 2)
	Config.RedisPassword = os.Getenv("REDIS_PASSWORD")

	// System authentication configuration
	Config.SystemAuthCacheTTL = parseDurationWithDefault("SYSTEM_AUTH_CACHE_TTL", 1*time.Hour)
	Config.SystemSecretMinLength = parseIntWithDefault("SYSTEM_SECRET_MIN_LENGTH", 32)

	// Session configuration
	Config.SessionDefaultDuration = parseDurationWithDefault("SESSION_DEFAULT_DURATION", 24*time.Hour)
	Config.SessionCleanerInterval = parseDurationWithDefault("SESSION_CLEANER_INTERVAL", 5*time.Minute)

	// Tunnel configuration
	Config.TunnelGracePeriod = parseDurationWithDefault("TUNNEL_GRACE_PERIOD", 30*time.Second)
	Config.MaxTunnels = parseIntWithDefault("MAX_TUNNELS", 1000)
	Config.MaxSessionsPerSystem = parseIntWithDefault("MAX_SESSIONS_PER_SYSTEM", 5)
	Config.MaxStreamsPerTunnel = parseIntWithDefault("MAX_STREAMS_PER_TUNNEL", 64)

	// Terminal configuration
	Config.TerminalInactivityTimeout = parseDurationWithDefault("TERMINAL_INACTIVITY_TIMEOUT", 30*time.Minute)
	Config.TerminalMaxFrameSize = parseIntWithDefault("TERMINAL_MAX_FRAME_SIZE", 65536)

	// Rate limiting configuration
	Config.RateLimitTunnelPerIP = parseIntWithDefault("RATE_LIMIT_TUNNEL_PER_IP", 10)
	Config.RateLimitTunnelPerKey = parseIntWithDefault("RATE_LIMIT_TUNNEL_PER_KEY", 5)
	Config.RateLimitSessionPerID = parseIntWithDefault("RATE_LIMIT_SESSION_PER_ID", 10000)
	Config.RateLimitAuthFailPerIP = parseIntWithDefault("RATE_LIMIT_AUTH_FAIL_PER_IP", 5)
	Config.RateLimitWindow = parseDurationWithDefault("RATE_LIMIT_WINDOW", 1*time.Minute)

	// Default: private ranges only (nginx and the Render proxy reach us over the private network)
	Config.TrustedProxies = parseListWithDefault("TRUSTED_PROXIES",
		[]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8", "::1/128", "fc00::/7"})

	Config.AppEnv = getStringWithDefault("APP_ENV", "dev")
	Config.InternalHMACSecret = os.Getenv("INTERNAL_HMAC_SECRET")
	if Config.InternalHMACSecret != "" && len(Config.InternalHMACSecret) < 32 {
		logger.Fatal().
			Int("length", len(Config.InternalHMACSecret)).
			Msg("INTERNAL_HMAC_SECRET must be at least 32 characters; refusing to start with a weak secret")
	}

	// Internal authentication (required: fail fast if empty)
	Config.InternalSecret = os.Getenv("SUPPORT_INTERNAL_SECRET")
	if Config.InternalSecret == "" {
		logger.Fatal().Msg("SUPPORT_INTERNAL_SECRET is required but not set")
	}

	logger.LogConfigLoad("env", "configuration", true, nil)
}

func parseDurationWithDefault(envVar string, defaultValue time.Duration) time.Duration {
	envValue := os.Getenv(envVar)
	if envValue == "" {
		return defaultValue
	}
	if duration, err := time.ParseDuration(envValue); err == nil {
		return duration
	}
	logger.LogConfigLoad("env", envVar, false, fmt.Errorf("invalid duration format, using default %v", defaultValue))
	return defaultValue
}

func parseIntWithDefault(envVar string, defaultValue int) int {
	envValue := os.Getenv(envVar)
	if envValue == "" {
		return defaultValue
	}
	if value, err := strconv.Atoi(envValue); err == nil {
		return value
	}
	logger.LogConfigLoad("env", envVar, false, fmt.Errorf("invalid integer format, using default %d", defaultValue))
	return defaultValue
}

func parseListWithDefault(envVar string, defaultValue []string) []string {
	envValue := os.Getenv(envVar)
	if envValue == "" {
		return defaultValue
	}
	var out []string
	for _, item := range strings.Split(envValue, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func getStringWithDefault(envVar string, defaultValue string) string {
	if envValue := os.Getenv(envVar); envValue != "" {
		return envValue
	}
	return defaultValue
}
