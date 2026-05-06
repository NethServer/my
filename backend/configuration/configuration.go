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
	"net/url"
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
	AppURL        string `json:"app_url"`
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
	RedisURL string `json:"redis_url"`
	RedisDB  int    `json:"redis_db"`
	// RedisDBShared is the Redis logical DB used for cross-service keys
	// (currently the per-org backup-usage counter). Default 1 to match
	// collect's default REDIS_DB.
	RedisDBShared     int           `json:"redis_db_shared"`
	RedisPassword     string        `json:"redis_password"`
	RedisMaxRetries   int           `json:"redis_max_retries"`
	RedisDialTimeout  time.Duration `json:"redis_dial_timeout"`
	RedisReadTimeout  time.Duration `json:"redis_read_timeout"`
	RedisWriteTimeout time.Duration `json:"redis_write_timeout"`
	// Cache TTL configuration
	JWKSCacheTTL time.Duration `json:"jwks_cache_ttl"`
	// Redis pool configuration
	RedisPoolSize     int           `json:"redis_pool_size"`
	RedisMinIdleConns int           `json:"redis_min_idle_conns"`
	RedisPoolTimeout  time.Duration `json:"redis_pool_timeout"`
	// HTTP timeouts configuration
	JWKSHTTPTimeout       time.Duration `json:"jwks_http_timeout"`
	RedisOperationTimeout time.Duration `json:"redis_operation_timeout"`
	// API configuration
	DefaultPageSize int `json:"default_page_size"`
	// System types configuration
	SystemTypes []string `json:"system_types"`
	// SMTP configuration for sending emails
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	SMTPFrom     string `json:"smtp_from"`
	SMTPFromName string `json:"smtp_from_name"`
	SMTPTLS      bool   `json:"smtp_tls"`
	// Mimir configuration
	MimirURL string `json:"mimir_url"`
	// Alerting configuration
	AlertingHistoryWebhookURL    string `json:"alerting_history_webhook_url"`
	AlertingHistoryWebhookSecret string `json:"alerting_history_webhook_secret"`

	// Cross-service plumbing.
	// AppEnv namespaces internal pub/sub channels (auth invalidation, etc.)
	// so a Redis instance shared between dev/qa/prod stops cross-pollinating.
	// Default "dev"; canonical values: dev, qa, prod.
	AppEnv string `json:"app_env"`
	// InternalHMACSecret signs internal pub/sub payloads so a network-adjacent
	// attacker with PUBLISH access cannot forge invalidations. Optional in dev
	// (empty disables the verification on the consumer side); required for
	// production-grade isolation.
	InternalHMACSecret string `json:"internal_hmac_secret"`

	// Backup storage — S3 client credentials used to read from the
	// DigitalOcean Spaces bucket that holds client configuration
	// backups. The same Spaces account also hosts the Mimir buckets;
	// values for endpoint, access key, and secret key are the shared
	// S3 credentials.
	S3Endpoint              string        `json:"s3_endpoint"`
	BackupS3PresignEndpoint string        `json:"backup_s3_presign_endpoint"`
	BackupS3Region          string        `json:"backup_s3_region"`
	BackupS3Bucket          string        `json:"backup_s3_bucket"`
	S3AccessKey             string        `json:"s3_access_key"`
	S3SecretKey             string        `json:"s3_secret_key"`
	BackupS3UsePathStyle    bool          `json:"backup_s3_use_path_style"`
	BackupPresignTTL        time.Duration `json:"backup_presign_ttl"`
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
	if os.Getenv("LOGTO_TENANT_ID") != "" {
		Config.TenantID = os.Getenv("LOGTO_TENANT_ID")
		// Derive base URL from tenant ID
		Config.LogtoIssuer = fmt.Sprintf("https://%s.logto.app", Config.TenantID)
	} else {
		logger.LogConfigLoad("env", "LOGTO_TENANT_ID", false, fmt.Errorf("LOGTO_TENANT_ID variable is empty"))
	}

	// Tenant domain configuration (required for JWT issuer)
	if os.Getenv("LOGTO_TENANT_DOMAIN") != "" {
		Config.TenantDomain = os.Getenv("LOGTO_TENANT_DOMAIN")
	} else {
		logger.LogConfigLoad("env", "LOGTO_TENANT_DOMAIN", false, fmt.Errorf("LOGTO_TENANT_DOMAIN variable is empty"))
	}

	// App URL configuration (required for app URL)
	if os.Getenv("APP_URL") != "" {
		Config.AppURL = os.Getenv("APP_URL")
	} else {
		logger.LogConfigLoad("env", "APP_URL", false, fmt.Errorf("APP_URL variable is empty"))
	}

	// LOGTO_AUDIENCE (auto-derived from LOGTO_TENANT_DOMAIN)
	Config.LogtoAudience = fmt.Sprintf("https://%s/api", Config.TenantDomain)

	// JWKS endpoint (auto-derived from LogtoIssuer)
	Config.JWKSEndpoint = Config.LogtoIssuer + "/oidc/jwks"

	// JWT custom token configuration
	if os.Getenv("JWT_SECRET") != "" {
		Config.JWTSecret = os.Getenv("JWT_SECRET")
		if len(Config.JWTSecret) < 32 {
			logger.ComponentLogger("env").Warn().
				Str("operation", "config_load").
				Str("config_type", "JWT_SECRET").
				Int("length", len(Config.JWTSecret)).
				Int("min_length", 32).
				Msg("JWT_SECRET should be at least 32 characters for security")
		}
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
	if os.Getenv("LOGTO_BACKEND_APP_ID") != "" {
		Config.LogtoManagementClientID = os.Getenv("LOGTO_BACKEND_APP_ID")
	} else {
		logger.LogConfigLoad("env", "LOGTO_BACKEND_APP_ID", false, fmt.Errorf("LOGTO_BACKEND_APP_ID variable is empty"))
	}

	if os.Getenv("LOGTO_BACKEND_APP_SECRET") != "" {
		Config.LogtoManagementClientSecret = os.Getenv("LOGTO_BACKEND_APP_SECRET")
	} else {
		logger.LogConfigLoad("env", "LOGTO_BACKEND_APP_SECRET", false, fmt.Errorf("LOGTO_BACKEND_APP_SECRET variable is empty"))
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
	Config.RedisDBShared = parseIntWithDefault("REDIS_DB_SHARED", 1)
	Config.RedisPassword = os.Getenv("REDIS_PASSWORD")
	Config.RedisMaxRetries = parseIntWithDefault("REDIS_MAX_RETRIES", 3)
	Config.RedisDialTimeout = parseDurationWithDefault("REDIS_DIAL_TIMEOUT", 5*time.Second)
	Config.RedisReadTimeout = parseDurationWithDefault("REDIS_READ_TIMEOUT", 3*time.Second)
	Config.RedisWriteTimeout = parseDurationWithDefault("REDIS_WRITE_TIMEOUT", 3*time.Second)

	// Redis pool configuration with defaults
	Config.RedisPoolSize = parseIntWithDefault("REDIS_POOL_SIZE", 50)
	Config.RedisMinIdleConns = parseIntWithDefault("REDIS_MIN_IDLE_CONNS", 10)
	Config.RedisPoolTimeout = parseDurationWithDefault("REDIS_POOL_TIMEOUT", 5*time.Second)

	// Cache TTL configuration with defaults
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

	// SMTP configuration
	Config.SMTPHost = os.Getenv("SMTP_HOST")
	Config.SMTPPort = parseIntWithDefault("SMTP_PORT", 587)
	Config.SMTPUsername = os.Getenv("SMTP_USERNAME")
	Config.SMTPPassword = os.Getenv("SMTP_PASSWORD")
	Config.SMTPFrom = os.Getenv("SMTP_FROM")
	Config.SMTPFromName = os.Getenv("SMTP_FROM_NAME")
	if Config.SMTPFromName == "" {
		Config.SMTPFromName = "My Nethesis"
	}
	Config.SMTPTLS = parseBoolWithDefault("SMTP_TLS", true)

	// Mimir configuration
	if mimirURL := os.Getenv("MIMIR_URL"); mimirURL != "" {
		Config.MimirURL = mimirURL
	} else {
		Config.MimirURL = "http://localhost:9009"
		logger.LogConfigLoad("env", "MIMIR_URL", true, fmt.Errorf("MIMIR_URL variable is empty, using default http://localhost:9009"))
	}

	// Alerting configuration — optional, empty means no built-in history webhook.
	// Backend renders the secret into the Alertmanager YAML pushed to Mimir, so
	// a short value would be cheaply brute-forced from any read of the Mimir
	// alertmanager-config bucket. Refuse to boot with a too-short secret.
	Config.AlertingHistoryWebhookURL = os.Getenv("ALERTING_HISTORY_WEBHOOK_URL")
	Config.AlertingHistoryWebhookSecret = os.Getenv("ALERTING_HISTORY_WEBHOOK_SECRET")
	if Config.AlertingHistoryWebhookSecret != "" && len(Config.AlertingHistoryWebhookSecret) < 32 {
		logger.Fatal().
			Int("length", len(Config.AlertingHistoryWebhookSecret)).
			Msg("ALERTING_HISTORY_WEBHOOK_SECRET must be at least 32 characters; refusing to start with a weak secret")
	}

	// Internal cross-service plumbing. APP_ENV scopes pub/sub channel names so
	// dev/qa/prod sharing a Redis instance never invalidate each other's caches.
	// INTERNAL_HMAC_SECRET, when set, signs the system_key payload on the
	// auth-invalidation channel; collect drops messages with a missing or
	// invalid HMAC. A short secret is refused (< 32 chars).
	if v := os.Getenv("APP_ENV"); v != "" {
		Config.AppEnv = v
	} else {
		Config.AppEnv = "dev"
	}
	Config.InternalHMACSecret = os.Getenv("INTERNAL_HMAC_SECRET")
	if Config.InternalHMACSecret != "" && len(Config.InternalHMACSecret) < 32 {
		logger.Fatal().
			Int("length", len(Config.InternalHMACSecret)).
			Msg("INTERNAL_HMAC_SECRET must be at least 32 characters; refusing to start with a weak secret")
	}

	// Backup storage — S3 client credentials (DigitalOcean Spaces)
	Config.S3Endpoint = validateBackupEndpoint("S3_ENDPOINT", os.Getenv("S3_ENDPOINT"))
	// Optional override used only when the API-facing endpoint differs
	// from the endpoint the user's browser can reach (typical for local
	// dev where backend runs inside a container and MinIO is exposed on
	// the host). Empty in production.
	Config.BackupS3PresignEndpoint = validateBackupEndpoint("BACKUP_S3_PRESIGN_ENDPOINT", os.Getenv("BACKUP_S3_PRESIGN_ENDPOINT"))
	if envRegion := os.Getenv("BACKUP_S3_REGION"); envRegion != "" {
		Config.BackupS3Region = envRegion
	} else {
		Config.BackupS3Region = "us-east-1"
	}
	Config.BackupS3Bucket = os.Getenv("BACKUP_S3_BUCKET")
	Config.S3AccessKey = os.Getenv("S3_ACCESS_KEY")
	Config.S3SecretKey = os.Getenv("S3_SECRET_KEY")
	Config.BackupS3UsePathStyle = parseBoolWithDefault("BACKUP_S3_USE_PATH_STYLE", false)
	// Cap the presigned URL lifetime at 15 minutes so a misconfigured
	// env can never mint long-lived bearer URLs to backup objects.
	ttl := parseDurationWithDefault("BACKUP_PRESIGN_TTL", 5*time.Minute)
	if ttl > 15*time.Minute {
		logger.LogConfigLoad("env", "BACKUP_PRESIGN_TTL", false, fmt.Errorf("value %s exceeds the 15m hard cap, clamping", ttl))
		ttl = 15 * time.Minute
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	Config.BackupPresignTTL = ttl

	// Log successful configuration load
	logger.LogConfigLoad("env", "configuration", true, nil)
}

// validateBackupEndpoint refuses HTTP endpoints unless the host is one
// of the well-known dev loopback names; misconfigured prod deployments
// would otherwise send signed S3 traffic in plaintext. Empty values are
// returned unchanged (the storage package surfaces a clearer error).
//
// A bare hostname (e.g. "ams3.digitaloceanspaces.com") is accepted and
// rewritten to "https://<host>" so the env var format is coherent with
// Mimir's S3 endpoint config (which takes a bare host) and the AWS SDK
// still receives a parseable URL for BaseEndpoint.
func validateBackupEndpoint(name, raw string) string {
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		logger.LogConfigLoad("env", name, false, fmt.Errorf("invalid URL %q", raw))
		return ""
	}
	// Reject userinfo (https://attacker:pw@host) — there is no legitimate
	// reason to embed credentials in a bucket endpoint URL.
	if u.User != nil {
		logger.LogConfigLoad("env", name, false, fmt.Errorf("userinfo in endpoint URL is not allowed"))
		return ""
	}
	// The endpoint must be the bucket service root, not a sub-path. The S3
	// SDK appends `/<bucket>/<key>` to whatever is given, so a non-root path
	// silently mis-targets traffic.
	if u.Path != "" && u.Path != "/" {
		logger.LogConfigLoad("env", name, false, fmt.Errorf("non-root path %q in endpoint URL is not allowed", u.Path))
		return ""
	}
	if u.Scheme == "https" {
		return raw
	}
	if u.Scheme == "http" {
		host := u.Hostname()
		// Allowlist exact loopback hosts only. Suffix-matching `.local` /
		// `.localtest.me` would otherwise admit `evil.local` /
		// `evil.localtest.me` shipped over plaintext.
		if host == "localhost" || host == "127.0.0.1" || host == "::1" ||
			host == "my.localtest.me" ||
			parseBoolWithDefault("BACKUP_S3_ALLOW_INSECURE", false) {
			return raw
		}
		logger.LogConfigLoad("env", name, false, fmt.Errorf("HTTP endpoint to non-loopback host %q rejected; set BACKUP_S3_ALLOW_INSECURE=true to override", host))
		return ""
	}
	logger.LogConfigLoad("env", name, false, fmt.Errorf("unsupported scheme %q", u.Scheme))
	return ""
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

// parseBoolWithDefault parses a boolean from environment variable or returns default
func parseBoolWithDefault(envVar string, defaultValue bool) bool {
	envValue := os.Getenv(envVar)
	if envValue == "" {
		return defaultValue
	}

	if value, err := strconv.ParseBool(envValue); err == nil {
		return value
	}

	// If parsing fails, log warning and use default
	logger.LogConfigLoad("env", envVar, false, fmt.Errorf("invalid boolean format, using default %t", defaultValue))
	return defaultValue
}
