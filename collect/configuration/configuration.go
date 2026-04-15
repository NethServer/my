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

	"github.com/nethesis/my/collect/logger"
)

type Configuration struct {
	ListenAddress string `json:"listen_address"`

	// Database configuration
	DatabaseURL      string `json:"database_url"`
	DatabaseMaxConns int    `json:"database_max_conns"`

	// Redis configuration
	RedisURL          string        `json:"redis_url"`
	RedisDB           int           `json:"redis_db"`
	RedisPassword     string        `json:"redis_password"`
	RedisMaxRetries   int           `json:"redis_max_retries"`
	RedisDialTimeout  time.Duration `json:"redis_dial_timeout"`
	RedisReadTimeout  time.Duration `json:"redis_read_timeout"`
	RedisWriteTimeout time.Duration `json:"redis_write_timeout"`
	RedisPoolSize     int           `json:"redis_pool_size"`
	RedisMinIdleConns int           `json:"redis_min_idle_conns"`
	RedisPoolTimeout  time.Duration `json:"redis_pool_timeout"`

	// Queue configuration
	QueueInventoryName    string        `json:"queue_inventory_name"`
	QueueProcessingName   string        `json:"queue_processing_name"`
	QueueNotificationName string        `json:"queue_notification_name"`
	QueueHeartbeatName    string        `json:"queue_heartbeat_name"`
	QueueBatchSize        int           `json:"queue_batch_size"`
	QueueRetryAttempts    int           `json:"queue_retry_attempts"`
	QueueRetryDelay       time.Duration `json:"queue_retry_delay"`

	// Worker configuration
	WorkerInventoryCount    int           `json:"worker_inventory_count"`
	WorkerProcessingCount   int           `json:"worker_processing_count"`
	WorkerNotificationCount int           `json:"worker_notification_count"`
	WorkerShutdownTimeout   time.Duration `json:"worker_shutdown_timeout"`
	WorkerHeartbeatInterval time.Duration `json:"worker_heartbeat_interval"`

	// Scalable worker configuration
	BatchProcessorSize      int           `json:"batch_processor_size"`
	BatchProcessorTimeout   time.Duration `json:"batch_processor_timeout"`
	BackpressureThreshold   float64       `json:"backpressure_threshold"`
	CircuitBreakerThreshold int           `json:"circuit_breaker_threshold"`
	CircuitBreakerTimeout   time.Duration `json:"circuit_breaker_timeout"`

	// Inventory processing configuration
	InventoryCleanupInterval time.Duration `json:"inventory_cleanup_interval"`
	InventoryDiffDepth       int           `json:"inventory_diff_depth"`

	// System authentication configuration
	SystemSecretMinLength int           `json:"system_secret_min_length"`
	SystemAuthCacheTTL    time.Duration `json:"system_auth_cache_ttl"`

	// API configuration
	APIMaxRequestSize int64         `json:"api_max_request_size"`
	APIRequestTimeout time.Duration `json:"api_request_timeout"`

	// Monitoring configuration
	HealthCheckInterval time.Duration `json:"health_check_interval"`

	// Notification configuration
	NotificationRetryAttempts int `json:"notification_retry_attempts"`

	// Heartbeat monitoring configuration
	HeartbeatTimeoutMinutes int `json:"heartbeat_timeout_minutes"`

	// Mimir configuration
	MimirURL string `json:"mimir_url"`

	// Alertmanager webhook authentication
	AlertmanagerWebhookSecret string `json:"alertmanager_webhook_secret"`

	// Backup storage — S3 client credentials used to reach the DigitalOcean
	// Spaces bucket that holds appliance configuration backups. The same
	// Spaces account also hosts the Mimir buckets; values for endpoint,
	// access key, and secret key are the shared S3 credentials.
	BackupS3Endpoint       string `json:"backup_s3_endpoint"`
	BackupS3Region         string `json:"backup_s3_region"`
	BackupS3Bucket         string `json:"backup_s3_bucket"`
	BackupS3AccessKey      string `json:"backup_s3_access_key"`
	BackupS3SecretKey      string `json:"backup_s3_secret_key"`
	BackupS3UsePathStyle   bool   `json:"backup_s3_use_path_style"`
	BackupMaxUploadSize    int64  `json:"backup_max_upload_size"`
	BackupMaxPerSystem     int    `json:"backup_max_per_system"`
	BackupMaxSizePerSystem int64  `json:"backup_max_size_per_system"`
	// Ingest rate limits (per system_id, Redis-backed). Legitimate
	// appliances upload on a daily timer, so these numbers are set
	// generously — their job is to block flood-style abuse, not to
	// shape normal traffic.
	BackupRateLimitPerMinute int `json:"backup_rate_limit_per_minute"`
	BackupRateLimitPerHour   int `json:"backup_rate_limit_per_hour"`
}

var Config = Configuration{}

func Init() {
	if os.Getenv("LISTEN_ADDRESS") != "" {
		Config.ListenAddress = os.Getenv("LISTEN_ADDRESS")
	} else {
		Config.ListenAddress = "127.0.0.1:8081"
	}

	// Database configuration
	if os.Getenv("DATABASE_URL") != "" {
		Config.DatabaseURL = os.Getenv("DATABASE_URL")
	} else {
		logger.LogConfigLoad("env", "DATABASE_URL", false, fmt.Errorf("DATABASE_URL variable is empty"))
	}

	Config.DatabaseMaxConns = parseIntWithDefault("DATABASE_MAX_CONNS", 10)

	// Redis configuration with defaults
	if os.Getenv("REDIS_URL") != "" {
		Config.RedisURL = os.Getenv("REDIS_URL")
	} else {
		Config.RedisURL = "redis://localhost:6379"
	}

	Config.RedisDB = parseIntWithDefault("REDIS_DB", 1) // Use DB 1 to separate from backend
	Config.RedisPassword = os.Getenv("REDIS_PASSWORD")
	Config.RedisMaxRetries = parseIntWithDefault("REDIS_MAX_RETRIES", 3)
	Config.RedisDialTimeout = parseDurationWithDefault("REDIS_DIAL_TIMEOUT", 5*time.Second)
	Config.RedisReadTimeout = parseDurationWithDefault("REDIS_READ_TIMEOUT", 3*time.Second)
	Config.RedisWriteTimeout = parseDurationWithDefault("REDIS_WRITE_TIMEOUT", 3*time.Second)
	Config.RedisPoolSize = parseIntWithDefault("REDIS_POOL_SIZE", 50)
	Config.RedisMinIdleConns = parseIntWithDefault("REDIS_MIN_IDLE_CONNS", 10)
	Config.RedisPoolTimeout = parseDurationWithDefault("REDIS_POOL_TIMEOUT", 10*time.Second)

	// Queue configuration
	Config.QueueInventoryName = getStringWithDefault("QUEUE_INVENTORY_NAME", "collect:inventory")
	Config.QueueProcessingName = getStringWithDefault("QUEUE_PROCESSING_NAME", "collect:processing")
	Config.QueueNotificationName = getStringWithDefault("QUEUE_NOTIFICATION_NAME", "collect:notifications")
	Config.QueueHeartbeatName = getStringWithDefault("QUEUE_HEARTBEAT_NAME", "collect:heartbeat")
	Config.QueueBatchSize = parseIntWithDefault("QUEUE_BATCH_SIZE", 10)
	Config.QueueRetryAttempts = parseIntWithDefault("QUEUE_RETRY_ATTEMPTS", 3)
	Config.QueueRetryDelay = parseDurationWithDefault("QUEUE_RETRY_DELAY", 5*time.Second)

	// Worker configuration
	Config.WorkerInventoryCount = parseIntWithDefault("WORKER_INVENTORY_COUNT", 5)
	Config.WorkerProcessingCount = parseIntWithDefault("WORKER_PROCESSING_COUNT", 3)
	Config.WorkerNotificationCount = parseIntWithDefault("WORKER_NOTIFICATION_COUNT", 2)
	Config.WorkerShutdownTimeout = parseDurationWithDefault("WORKER_SHUTDOWN_TIMEOUT", 30*time.Second)
	Config.WorkerHeartbeatInterval = parseDurationWithDefault("WORKER_HEARTBEAT_INTERVAL", 10*time.Second)

	// Scalable worker configuration
	Config.BatchProcessorSize = parseIntWithDefault("BATCH_PROCESSOR_SIZE", 100)
	Config.BatchProcessorTimeout = parseDurationWithDefault("BATCH_PROCESSOR_TIMEOUT", 5*time.Second)
	Config.BackpressureThreshold = parseFloatWithDefault("BACKPRESSURE_THRESHOLD", 0.8)
	Config.CircuitBreakerThreshold = parseIntWithDefault("CIRCUIT_BREAKER_THRESHOLD", 5)
	Config.CircuitBreakerTimeout = parseDurationWithDefault("CIRCUIT_BREAKER_TIMEOUT", 30*time.Second)

	// Inventory processing configuration
	Config.InventoryCleanupInterval = parseDurationWithDefault("INVENTORY_CLEANUP_INTERVAL", 6*time.Hour)
	Config.InventoryDiffDepth = parseIntWithDefault("INVENTORY_DIFF_DEPTH", 10) // Max diff levels

	// System authentication configuration
	Config.SystemSecretMinLength = parseIntWithDefault("SYSTEM_SECRET_MIN_LENGTH", 32)
	// Cache positive auth results for 10 minutes by default. Shorter
	// than a day so that credential rotations and system deletions
	// done on the backend propagate to collect within that window
	// without a dedicated invalidation bus. Each cache miss falls
	// through to Redis and then to Postgres.
	Config.SystemAuthCacheTTL = parseDurationWithDefault("SYSTEM_AUTH_CACHE_TTL", 10*time.Minute)

	// API configuration
	Config.APIMaxRequestSize = parseInt64WithDefault("API_MAX_REQUEST_SIZE", 10*1024*1024) // 10MB
	Config.APIRequestTimeout = parseDurationWithDefault("API_REQUEST_TIMEOUT", 30*time.Second)

	// Monitoring configuration
	Config.HealthCheckInterval = parseDurationWithDefault("HEALTH_CHECK_INTERVAL", 30*time.Second)

	// Notification configuration
	Config.NotificationRetryAttempts = parseIntWithDefault("NOTIFICATION_RETRY_ATTEMPTS", 3)

	// Heartbeat monitoring configuration
	Config.HeartbeatTimeoutMinutes = parseIntWithDefault("HEARTBEAT_TIMEOUT_MINUTES", 10)

	// Mimir configuration
	if mimirURL := os.Getenv("MIMIR_URL"); mimirURL != "" {
		Config.MimirURL = mimirURL
	} else {
		Config.MimirURL = "http://localhost:9009"
	}

	// Alerting history webhook authentication
	Config.AlertmanagerWebhookSecret = os.Getenv("ALERTING_HISTORY_WEBHOOK_SECRET")

	// Backup storage — S3 client credentials (DigitalOcean Spaces)
	Config.BackupS3Endpoint = validateBackupEndpoint("BACKUP_S3_ENDPOINT", os.Getenv("BACKUP_S3_ENDPOINT"))
	Config.BackupS3Region = getStringWithDefault("BACKUP_S3_REGION", "us-east-1")
	Config.BackupS3Bucket = os.Getenv("BACKUP_S3_BUCKET")
	Config.BackupS3AccessKey = os.Getenv("BACKUP_S3_ACCESS_KEY")
	Config.BackupS3SecretKey = os.Getenv("BACKUP_S3_SECRET_KEY")
	Config.BackupS3UsePathStyle = os.Getenv("BACKUP_S3_USE_PATH_STYLE") == "true"
	Config.BackupMaxUploadSize = parseInt64WithDefault("BACKUP_MAX_UPLOAD_SIZE", 2*1024*1024*1024)
	Config.BackupMaxPerSystem = parseIntWithDefault("BACKUP_MAX_PER_SYSTEM", 10)
	Config.BackupMaxSizePerSystem = parseInt64WithDefault("BACKUP_MAX_SIZE_PER_SYSTEM", 500*1024*1024)
	Config.BackupRateLimitPerMinute = parseIntWithDefault("BACKUP_RATE_LIMIT_PER_MINUTE", 6)
	Config.BackupRateLimitPerHour = parseIntWithDefault("BACKUP_RATE_LIMIT_PER_HOUR", 60)

	// Log successful configuration load
	logger.LogConfigLoad("env", "configuration", true, nil)
}

// validateBackupEndpoint refuses HTTP endpoints unless the host is a
// well-known dev loopback name; misconfigured prod deployments would
// otherwise send signed S3 traffic in plaintext. Empty values are
// returned unchanged (the storage package surfaces a clearer error).
func validateBackupEndpoint(name, raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		logger.LogConfigLoad("env", name, false, fmt.Errorf("invalid URL %q", raw))
		return ""
	}
	if u.Scheme == "https" {
		return raw
	}
	if u.Scheme == "http" {
		host := u.Hostname()
		if host == "localhost" || host == "127.0.0.1" || host == "::1" ||
			strings.HasSuffix(host, ".local") ||
			strings.HasSuffix(host, ".localtest.me") ||
			os.Getenv("BACKUP_S3_ALLOW_INSECURE") == "true" {
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

	if duration, err := time.ParseDuration(envValue); err == nil {
		return duration
	}

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

	logger.LogConfigLoad("env", envVar, false, fmt.Errorf("invalid integer format, using default %d", defaultValue))
	return defaultValue
}

// parseInt64WithDefault parses an int64 from environment variable or returns default
func parseInt64WithDefault(envVar string, defaultValue int64) int64 {
	envValue := os.Getenv(envVar)
	if envValue == "" {
		return defaultValue
	}

	if value, err := strconv.ParseInt(envValue, 10, 64); err == nil {
		return value
	}

	logger.LogConfigLoad("env", envVar, false, fmt.Errorf("invalid int64 format, using default %d", defaultValue))
	return defaultValue
}

// parseFloatWithDefault parses a float64 from environment variable or returns default
func parseFloatWithDefault(envVar string, defaultValue float64) float64 {
	envValue := os.Getenv(envVar)
	if envValue == "" {
		return defaultValue
	}

	if value, err := strconv.ParseFloat(envValue, 64); err == nil {
		return value
	}

	logger.LogConfigLoad("env", envVar, false, fmt.Errorf("invalid float format, using default %f", defaultValue))
	return defaultValue
}

// getStringWithDefault gets a string from environment variable or returns default
func getStringWithDefault(envVar string, defaultValue string) string {
	if envValue := os.Getenv(envVar); envValue != "" {
		return envValue
	}
	return defaultValue
}
