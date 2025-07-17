/*
 * Copyright (C) 2025 Nethesis S.r.l.
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

	"github.com/nethesis/my/backend/logger"
)

type Configuration struct {
	ListenAddress string `json:"listen_address"`
	// Database configuration
	DatabaseURL   string `json:"database_url"`
	TenantID      string `json:"tenant_id"`
	TenantDomain  string `json:"tenant_domain"`
	LogtoIssuer   string `json:"logto_issuer"`
	LogtoAudience string `json:"logto_audience"`
	JWKSEndpoint  string `json:"jwks_endpoint"`
	// JWT Custom token configuration
	JWTSecret            string `json:"jwt_secret"`
	JWTIssuer            string `json:"jwt_issuer"`
	JWTExpiration        string `json:"jwt_expiration"`
	JWTRefreshExpiration string `json:"jwt_refresh_expiration"`
	// Logto Management API configuration
	LogtoManagementClientID     string `json:"logto_management_client_id"`
	LogtoManagementClientSecret string `json:"logto_management_client_secret"`
	LogtoManagementBaseURL      string `json:"logto_management_base_url"`
	// Redis configuration
	RedisURL          string        `json:"redis_url"`
	RedisDB           int           `json:"redis_db"`
	RedisPassword     string        `json:"redis_password"`
	RedisMaxRetries   int           `json:"redis_max_retries"`
	RedisDialTimeout  time.Duration `json:"redis_dial_timeout"`
	RedisReadTimeout  time.Duration `json:"redis_read_timeout"`
	RedisWriteTimeout time.Duration `json:"redis_write_timeout"`
	// Cache TTL configuration
	StatsCacheTTL           time.Duration `json:"stats_cache_ttl"`
	StatsUpdateInterval     time.Duration `json:"stats_update_interval"`
	StatsStaleThreshold     time.Duration `json:"stats_stale_threshold"`
	JitRolesCacheTTL        time.Duration `json:"jit_roles_cache_ttl"`
	JitRolesCleanupInterval time.Duration `json:"jit_roles_cleanup_interval"`
	OrgUsersCacheTTL        time.Duration `json:"org_users_cache_ttl"`
	OrgUsersCleanupInterval time.Duration `json:"org_users_cleanup_interval"`
	JWKSCacheTTL            time.Duration `json:"jwks_cache_ttl"`
	// HTTP timeouts configuration
	JWKSHTTPTimeout       time.Duration `json:"jwks_http_timeout"`
	RedisOperationTimeout time.Duration `json:"redis_operation_timeout"`
	// API configuration
	DefaultPageSize int `json:"default_page_size"`
	// System types configuration
	SystemTypes []string `json:"system_types"`
}

var Config = Configuration{}

func Init() {
	if os.Getenv("LISTEN_ADDRESS") != "" {
		Config.ListenAddress = os.Getenv("LISTEN_ADDRESS")
	} else {
		Config.ListenAddress = "127.0.0.1:8080"
	}

	// Database configuration
	if os.Getenv("DATABASE_URL") != "" {
		Config.DatabaseURL = os.Getenv("DATABASE_URL")
	} else {
		logger.LogConfigLoad("env", "DATABASE_URL", false, fmt.Errorf("DATABASE_URL variable is empty"))
	}

	// Tenant ID configuration (required)
	if os.Getenv("TENANT_ID") != "" {
		Config.TenantID = os.Getenv("TENANT_ID")
		// Derive base URL from tenant ID
		Config.LogtoIssuer = fmt.Sprintf("https://%s.logto.app", Config.TenantID)
	} else {
		logger.LogConfigLoad("env", "TENANT_ID", false, fmt.Errorf("TENANT_ID variable is empty"))
	}

	// Tenant domain configuration (required for JWT issuer)
	if os.Getenv("TENANT_DOMAIN") != "" {
		Config.TenantDomain = os.Getenv("TENANT_DOMAIN")
	} else {
		logger.LogConfigLoad("env", "TENANT_DOMAIN", false, fmt.Errorf("TENANT_DOMAIN variable is empty"))
	}

	// LOGTO_AUDIENCE (auto-derived from TENANT_DOMAIN)
	Config.LogtoAudience = fmt.Sprintf("https://%s/api", Config.TenantDomain)

	// JWKS endpoint (auto-derived from LogtoIssuer)
	Config.JWKSEndpoint = Config.LogtoIssuer + "/oidc/jwks"

	// JWT custom token configuration
	if os.Getenv("JWT_SECRET") != "" {
		Config.JWTSecret = os.Getenv("JWT_SECRET")
	} else {
		logger.LogConfigLoad("env", "JWT_SECRET", false, fmt.Errorf("JWT_SECRET variable is empty"))
	}

	// JWT issuer (uses tenant domain)
	Config.JWTIssuer = Config.TenantDomain

	if os.Getenv("JWT_EXPIRATION") != "" {
		Config.JWTExpiration = os.Getenv("JWT_EXPIRATION")
	} else {
		Config.JWTExpiration = "24h" // Default: 24 hours
	}

	if os.Getenv("JWT_REFRESH_EXPIRATION") != "" {
		Config.JWTRefreshExpiration = os.Getenv("JWT_REFRESH_EXPIRATION")
	} else {
		Config.JWTRefreshExpiration = "168h" // Default: 7 days
	}

	// Logto Management API configuration
	if os.Getenv("BACKEND_APP_ID") != "" {
		Config.LogtoManagementClientID = os.Getenv("BACKEND_APP_ID")
	} else {
		logger.LogConfigLoad("env", "BACKEND_APP_ID", false, fmt.Errorf("BACKEND_APP_ID variable is empty"))
	}

	if os.Getenv("BACKEND_APP_SECRET") != "" {
		Config.LogtoManagementClientSecret = os.Getenv("BACKEND_APP_SECRET")
	} else {
		logger.LogConfigLoad("env", "BACKEND_APP_SECRET", false, fmt.Errorf("BACKEND_APP_SECRET variable is empty"))
	}

	// Logto Management API base URL (auto-derived from LogtoIssuer)
	Config.LogtoManagementBaseURL = Config.LogtoIssuer + "/api"

	// Redis configuration with defaults
	if os.Getenv("REDIS_URL") != "" {
		Config.RedisURL = os.Getenv("REDIS_URL")
	} else {
		Config.RedisURL = "redis://localhost:6379"
	}

	Config.RedisDB = parseIntWithDefault("REDIS_DB", 0)
	Config.RedisPassword = os.Getenv("REDIS_PASSWORD")
	Config.RedisMaxRetries = parseIntWithDefault("REDIS_MAX_RETRIES", 3)
	Config.RedisDialTimeout = parseDurationWithDefault("REDIS_DIAL_TIMEOUT", 5*time.Second)
	Config.RedisReadTimeout = parseDurationWithDefault("REDIS_READ_TIMEOUT", 3*time.Second)
	Config.RedisWriteTimeout = parseDurationWithDefault("REDIS_WRITE_TIMEOUT", 3*time.Second)

	// Cache TTL configuration with defaults
	Config.StatsCacheTTL = parseDurationWithDefault("STATS_CACHE_TTL", 10*time.Minute)
	Config.StatsUpdateInterval = parseDurationWithDefault("STATS_UPDATE_INTERVAL", 5*time.Minute)
	Config.StatsStaleThreshold = parseDurationWithDefault("STATS_STALE_THRESHOLD", 15*time.Minute)
	Config.JitRolesCacheTTL = parseDurationWithDefault("JIT_ROLES_CACHE_TTL", 5*time.Minute)
	Config.JitRolesCleanupInterval = parseDurationWithDefault("JIT_ROLES_CLEANUP_INTERVAL", 2*time.Minute)
	Config.OrgUsersCacheTTL = parseDurationWithDefault("ORG_USERS_CACHE_TTL", 3*time.Minute)
	Config.OrgUsersCleanupInterval = parseDurationWithDefault("ORG_USERS_CLEANUP_INTERVAL", 1*time.Minute)
	Config.JWKSCacheTTL = parseDurationWithDefault("JWKS_CACHE_TTL", 5*time.Minute)
	Config.JWKSHTTPTimeout = parseDurationWithDefault("JWKS_HTTP_TIMEOUT", 10*time.Second)
	Config.RedisOperationTimeout = parseDurationWithDefault("REDIS_OPERATION_TIMEOUT", 5*time.Second)
	Config.DefaultPageSize = parseIntWithDefault("DEFAULT_PAGE_SIZE", 100)

	// System types configuration
	if os.Getenv("SYSTEM_TYPES") != "" {
		Config.SystemTypes = parseStringSliceWithDefault("SYSTEM_TYPES", []string{"ns8", "nsec"})
	} else {
		Config.SystemTypes = []string{"ns8", "nsec"}
	}

	// Log successful configuration load
	logger.LogConfigLoad("env", "configuration", true, nil)
}

// parseDurationWithDefault parses a duration from environment variable or returns default
func parseDurationWithDefault(envVar string, defaultValue time.Duration) time.Duration {
	envValue := os.Getenv(envVar)
	if envValue == "" {
		return defaultValue
	}

	// Parse as duration string (e.g., "5m", "10s", "1h30m")
	if duration, err := time.ParseDuration(envValue); err == nil {
		return duration
	}

	// If parsing fails, log warning and use default
	logger.LogConfigLoad("env", envVar, false, fmt.Errorf("invalid duration format, using default %v", defaultValue))
	return defaultValue
}

// parseIntWithDefault parses an integer from environment variable or returns default
func parseIntWithDefault(envVar string, defaultValue int) int {
	envValue := os.Getenv(envVar)
	if envValue == "" {
		return defaultValue
	}

	if value, err := strconv.Atoi(envValue); err == nil {
		return value
	}

	// If parsing fails, log warning and use default
	logger.LogConfigLoad("env", envVar, false, fmt.Errorf("invalid integer format, using default %d", defaultValue))
	return defaultValue
}

// parseStringSliceWithDefault parses a comma-separated string from environment variable or returns default
func parseStringSliceWithDefault(envVar string, defaultValue []string) []string {
	envValue := os.Getenv(envVar)
	if envValue == "" {
		return defaultValue
	}

	// Split by comma and trim whitespace
	parts := strings.Split(envValue, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		logger.LogConfigLoad("env", envVar, false, fmt.Errorf("empty list provided, using default %v", defaultValue))
		return defaultValue
	}

	return result
}
