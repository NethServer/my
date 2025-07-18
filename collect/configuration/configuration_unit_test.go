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
	"os"
	"testing"
	"time"

	"github.com/nethesis/my/collect/logger"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	// Initialize logger for testing
	_ = logger.InitFromEnv("test")

	// Save original env vars
	originalEnvVars := make(map[string]string)
	envVars := []string{
		"LISTEN_ADDRESS", "DATABASE_URL", "DATABASE_MAX_CONNS",
		"REDIS_URL", "REDIS_DB", "REDIS_PASSWORD", "REDIS_MAX_RETRIES",
		"REDIS_DIAL_TIMEOUT", "REDIS_READ_TIMEOUT", "REDIS_WRITE_TIMEOUT",
		"QUEUE_INVENTORY_NAME", "QUEUE_PROCESSING_NAME", "QUEUE_NOTIFICATION_NAME",
		"QUEUE_BATCH_SIZE", "QUEUE_RETRY_ATTEMPTS", "QUEUE_RETRY_DELAY",
		"WORKER_INVENTORY_COUNT", "WORKER_PROCESSING_COUNT", "WORKER_NOTIFICATION_COUNT",
		"WORKER_SHUTDOWN_TIMEOUT", "WORKER_HEARTBEAT_INTERVAL",
		"BATCH_PROCESSOR_SIZE", "BATCH_PROCESSOR_TIMEOUT", "BACKPRESSURE_THRESHOLD",
		"CIRCUIT_BREAKER_THRESHOLD", "CIRCUIT_BREAKER_TIMEOUT",
		"INVENTORY_MAX_AGE", "INVENTORY_CLEANUP_INTERVAL", "INVENTORY_DIFF_DEPTH",
		"SYSTEM_SECRET_MIN_LENGTH", "SYSTEM_AUTH_CACHE_TTL",
		"API_MAX_REQUEST_SIZE", "API_REQUEST_TIMEOUT",
		"HEALTH_CHECK_INTERVAL", "NOTIFICATION_RETRY_ATTEMPTS",
	}

	for _, envVar := range envVars {
		if val := os.Getenv(envVar); val != "" {
			originalEnvVars[envVar] = val
		}
		_ = os.Unsetenv(envVar)
	}

	// Restore original env vars after test
	defer func() {
		for _, envVar := range envVars {
			_ = os.Unsetenv(envVar)
			if val, exists := originalEnvVars[envVar]; exists {
				_ = os.Setenv(envVar, val)
			}
		}
	}()

	// Set DATABASE_URL to avoid logger.Fatal call
	_ = os.Setenv("DATABASE_URL", "postgres://localhost:5432/testdb")
	defer func() { _ = os.Unsetenv("DATABASE_URL") }()

	// Test with default values
	Init()

	assert.Equal(t, "127.0.0.1:8081", Config.ListenAddress)
	assert.Equal(t, "redis://localhost:6379", Config.RedisURL)
	assert.Equal(t, 1, Config.RedisDB)
	assert.Equal(t, 10, Config.DatabaseMaxConns)
	assert.Equal(t, 3, Config.RedisMaxRetries)
	assert.Equal(t, 5*time.Second, Config.RedisDialTimeout)
	assert.Equal(t, 3*time.Second, Config.RedisReadTimeout)
	assert.Equal(t, 3*time.Second, Config.RedisWriteTimeout)
	assert.Equal(t, "collect:inventory", Config.QueueInventoryName)
	assert.Equal(t, "collect:processing", Config.QueueProcessingName)
	assert.Equal(t, "collect:notifications", Config.QueueNotificationName)
	assert.Equal(t, 10, Config.QueueBatchSize)
	assert.Equal(t, 3, Config.QueueRetryAttempts)
	assert.Equal(t, 5*time.Second, Config.QueueRetryDelay)
	assert.Equal(t, 5, Config.WorkerInventoryCount)
	assert.Equal(t, 3, Config.WorkerProcessingCount)
	assert.Equal(t, 2, Config.WorkerNotificationCount)
	assert.Equal(t, 30*time.Second, Config.WorkerShutdownTimeout)
	assert.Equal(t, 10*time.Second, Config.WorkerHeartbeatInterval)
	assert.Equal(t, 100, Config.BatchProcessorSize)
	assert.Equal(t, 5*time.Second, Config.BatchProcessorTimeout)
	assert.Equal(t, 0.8, Config.BackpressureThreshold)
	assert.Equal(t, 5, Config.CircuitBreakerThreshold)
	assert.Equal(t, 30*time.Second, Config.CircuitBreakerTimeout)
	assert.Equal(t, 90*24*time.Hour, Config.InventoryMaxAge)
	assert.Equal(t, 6*time.Hour, Config.InventoryCleanupInterval)
	assert.Equal(t, 10, Config.InventoryDiffDepth)
	assert.Equal(t, 32, Config.SystemSecretMinLength)
	assert.Equal(t, 5*time.Minute, Config.SystemAuthCacheTTL)
	assert.Equal(t, int64(10*1024*1024), Config.APIMaxRequestSize)
	assert.Equal(t, 30*time.Second, Config.APIRequestTimeout)
	assert.Equal(t, 30*time.Second, Config.HealthCheckInterval)
	assert.Equal(t, 3, Config.NotificationRetryAttempts)
}

func TestInitWithEnvironmentVariables(t *testing.T) {
	// Initialize logger for testing
	_ = logger.InitFromEnv("test")

	// Set environment variables
	_ = os.Setenv("LISTEN_ADDRESS", "0.0.0.0:9000")
	_ = os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/testdb")
	_ = os.Setenv("DATABASE_MAX_CONNS", "20")
	_ = os.Setenv("REDIS_URL", "redis://localhost:6380")
	_ = os.Setenv("REDIS_DB", "2")
	_ = os.Setenv("REDIS_PASSWORD", "testpass")
	_ = os.Setenv("REDIS_MAX_RETRIES", "5")
	_ = os.Setenv("REDIS_DIAL_TIMEOUT", "10s")
	_ = os.Setenv("REDIS_READ_TIMEOUT", "5s")
	_ = os.Setenv("REDIS_WRITE_TIMEOUT", "5s")
	_ = os.Setenv("QUEUE_INVENTORY_NAME", "test:inventory")
	_ = os.Setenv("QUEUE_PROCESSING_NAME", "test:processing")
	_ = os.Setenv("QUEUE_NOTIFICATION_NAME", "test:notifications")
	_ = os.Setenv("QUEUE_BATCH_SIZE", "20")
	_ = os.Setenv("QUEUE_RETRY_ATTEMPTS", "5")
	_ = os.Setenv("QUEUE_RETRY_DELAY", "10s")
	_ = os.Setenv("WORKER_INVENTORY_COUNT", "10")
	_ = os.Setenv("WORKER_PROCESSING_COUNT", "6")
	_ = os.Setenv("WORKER_NOTIFICATION_COUNT", "4")
	_ = os.Setenv("WORKER_SHUTDOWN_TIMEOUT", "60s")
	_ = os.Setenv("WORKER_HEARTBEAT_INTERVAL", "20s")
	_ = os.Setenv("BATCH_PROCESSOR_SIZE", "200")
	_ = os.Setenv("BATCH_PROCESSOR_TIMEOUT", "10s")
	_ = os.Setenv("BACKPRESSURE_THRESHOLD", "0.9")
	_ = os.Setenv("CIRCUIT_BREAKER_THRESHOLD", "10")
	_ = os.Setenv("CIRCUIT_BREAKER_TIMEOUT", "60s")
	_ = os.Setenv("INVENTORY_MAX_AGE", "168h")
	_ = os.Setenv("INVENTORY_CLEANUP_INTERVAL", "12h")
	_ = os.Setenv("INVENTORY_DIFF_DEPTH", "20")
	_ = os.Setenv("SYSTEM_SECRET_MIN_LENGTH", "64")
	_ = os.Setenv("SYSTEM_AUTH_CACHE_TTL", "10m")
	_ = os.Setenv("API_MAX_REQUEST_SIZE", "20971520")
	_ = os.Setenv("API_REQUEST_TIMEOUT", "60s")
	_ = os.Setenv("HEALTH_CHECK_INTERVAL", "60s")
	_ = os.Setenv("NOTIFICATION_RETRY_ATTEMPTS", "5")

	defer func() {
		envVars := []string{
			"LISTEN_ADDRESS", "DATABASE_URL", "DATABASE_MAX_CONNS",
			"REDIS_URL", "REDIS_DB", "REDIS_PASSWORD", "REDIS_MAX_RETRIES",
			"REDIS_DIAL_TIMEOUT", "REDIS_READ_TIMEOUT", "REDIS_WRITE_TIMEOUT",
			"QUEUE_INVENTORY_NAME", "QUEUE_PROCESSING_NAME", "QUEUE_NOTIFICATION_NAME",
			"QUEUE_BATCH_SIZE", "QUEUE_RETRY_ATTEMPTS", "QUEUE_RETRY_DELAY",
			"WORKER_INVENTORY_COUNT", "WORKER_PROCESSING_COUNT", "WORKER_NOTIFICATION_COUNT",
			"WORKER_SHUTDOWN_TIMEOUT", "WORKER_HEARTBEAT_INTERVAL",
			"BATCH_PROCESSOR_SIZE", "BATCH_PROCESSOR_TIMEOUT", "BACKPRESSURE_THRESHOLD",
			"CIRCUIT_BREAKER_THRESHOLD", "CIRCUIT_BREAKER_TIMEOUT",
			"INVENTORY_MAX_AGE", "INVENTORY_CLEANUP_INTERVAL", "INVENTORY_DIFF_DEPTH",
			"SYSTEM_SECRET_MIN_LENGTH", "SYSTEM_AUTH_CACHE_TTL",
			"API_MAX_REQUEST_SIZE", "API_REQUEST_TIMEOUT",
			"HEALTH_CHECK_INTERVAL", "NOTIFICATION_RETRY_ATTEMPTS",
		}
		for _, envVar := range envVars {
			_ = os.Unsetenv(envVar)
		}
	}()

	Init()

	assert.Equal(t, "0.0.0.0:9000", Config.ListenAddress)
	assert.Equal(t, "postgres://test:test@localhost:5432/testdb", Config.DatabaseURL)
	assert.Equal(t, 20, Config.DatabaseMaxConns)
	assert.Equal(t, "redis://localhost:6380", Config.RedisURL)
	assert.Equal(t, 2, Config.RedisDB)
	assert.Equal(t, "testpass", Config.RedisPassword)
	assert.Equal(t, 5, Config.RedisMaxRetries)
	assert.Equal(t, 10*time.Second, Config.RedisDialTimeout)
	assert.Equal(t, 5*time.Second, Config.RedisReadTimeout)
	assert.Equal(t, 5*time.Second, Config.RedisWriteTimeout)
	assert.Equal(t, "test:inventory", Config.QueueInventoryName)
	assert.Equal(t, "test:processing", Config.QueueProcessingName)
	assert.Equal(t, "test:notifications", Config.QueueNotificationName)
	assert.Equal(t, 20, Config.QueueBatchSize)
	assert.Equal(t, 5, Config.QueueRetryAttempts)
	assert.Equal(t, 10*time.Second, Config.QueueRetryDelay)
	assert.Equal(t, 10, Config.WorkerInventoryCount)
	assert.Equal(t, 6, Config.WorkerProcessingCount)
	assert.Equal(t, 4, Config.WorkerNotificationCount)
	assert.Equal(t, 60*time.Second, Config.WorkerShutdownTimeout)
	assert.Equal(t, 20*time.Second, Config.WorkerHeartbeatInterval)
	assert.Equal(t, 200, Config.BatchProcessorSize)
	assert.Equal(t, 10*time.Second, Config.BatchProcessorTimeout)
	assert.Equal(t, 0.9, Config.BackpressureThreshold)
	assert.Equal(t, 10, Config.CircuitBreakerThreshold)
	assert.Equal(t, 60*time.Second, Config.CircuitBreakerTimeout)
	assert.Equal(t, 168*time.Hour, Config.InventoryMaxAge)
	assert.Equal(t, 12*time.Hour, Config.InventoryCleanupInterval)
	assert.Equal(t, 20, Config.InventoryDiffDepth)
	assert.Equal(t, 64, Config.SystemSecretMinLength)
	assert.Equal(t, 10*time.Minute, Config.SystemAuthCacheTTL)
	assert.Equal(t, int64(20*1024*1024), Config.APIMaxRequestSize)
	assert.Equal(t, 60*time.Second, Config.APIRequestTimeout)
	assert.Equal(t, 60*time.Second, Config.HealthCheckInterval)
	assert.Equal(t, 5, Config.NotificationRetryAttempts)
}

func TestParseDurationWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		envVar       string
		envValue     string
		defaultValue time.Duration
		expected     time.Duration
	}{
		{
			name:         "valid duration",
			envVar:       "TEST_DURATION",
			envValue:     "5s",
			defaultValue: 1 * time.Second,
			expected:     5 * time.Second,
		},
		{
			name:         "empty env var",
			envVar:       "TEST_DURATION",
			envValue:     "",
			defaultValue: 1 * time.Second,
			expected:     1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				_ = os.Setenv(tt.envVar, tt.envValue)
			} else {
				_ = os.Unsetenv(tt.envVar)
			}
			defer func() { _ = os.Unsetenv(tt.envVar) }()

			result := parseDurationWithDefault(tt.envVar, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseIntWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		envVar       string
		envValue     string
		defaultValue int
		expected     int
	}{
		{
			name:         "valid integer",
			envVar:       "TEST_INT",
			envValue:     "42",
			defaultValue: 10,
			expected:     42,
		},
		{
			name:         "empty env var",
			envVar:       "TEST_INT",
			envValue:     "",
			defaultValue: 10,
			expected:     10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				_ = os.Setenv(tt.envVar, tt.envValue)
			} else {
				_ = os.Unsetenv(tt.envVar)
			}
			defer func() { _ = os.Unsetenv(tt.envVar) }()

			result := parseIntWithDefault(tt.envVar, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseInt64WithDefault(t *testing.T) {
	tests := []struct {
		name         string
		envVar       string
		envValue     string
		defaultValue int64
		expected     int64
	}{
		{
			name:         "valid int64",
			envVar:       "TEST_INT64",
			envValue:     "9223372036854775807",
			defaultValue: 100,
			expected:     9223372036854775807,
		},
		{
			name:         "empty env var",
			envVar:       "TEST_INT64",
			envValue:     "",
			defaultValue: 100,
			expected:     100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				_ = os.Setenv(tt.envVar, tt.envValue)
			} else {
				_ = os.Unsetenv(tt.envVar)
			}
			defer func() { _ = os.Unsetenv(tt.envVar) }()

			result := parseInt64WithDefault(tt.envVar, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseFloatWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		envVar       string
		envValue     string
		defaultValue float64
		expected     float64
	}{
		{
			name:         "valid float",
			envVar:       "TEST_FLOAT",
			envValue:     "3.14",
			defaultValue: 1.0,
			expected:     3.14,
		},
		{
			name:         "empty env var",
			envVar:       "TEST_FLOAT",
			envValue:     "",
			defaultValue: 1.0,
			expected:     1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				_ = os.Setenv(tt.envVar, tt.envValue)
			} else {
				_ = os.Unsetenv(tt.envVar)
			}
			defer func() { _ = os.Unsetenv(tt.envVar) }()

			result := parseFloatWithDefault(tt.envVar, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStringWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		envVar       string
		envValue     string
		defaultValue string
		expected     string
	}{
		{
			name:         "valid string",
			envVar:       "TEST_STRING",
			envValue:     "hello",
			defaultValue: "default",
			expected:     "hello",
		},
		{
			name:         "empty env var",
			envVar:       "TEST_STRING",
			envValue:     "",
			defaultValue: "default",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				_ = os.Setenv(tt.envVar, tt.envValue)
			} else {
				_ = os.Unsetenv(tt.envVar)
			}
			defer func() { _ = os.Unsetenv(tt.envVar) }()

			result := getStringWithDefault(tt.envVar, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigurationStruct(t *testing.T) {
	config := Configuration{
		ListenAddress:    "127.0.0.1:8080",
		DatabaseURL:      "postgres://localhost:5432/testdb",
		DatabaseMaxConns: 20,
		RedisURL:         "redis://localhost:6379",
		RedisDB:          1,
		RedisPassword:    "password",
	}

	assert.Equal(t, "127.0.0.1:8080", config.ListenAddress)
	assert.Equal(t, "postgres://localhost:5432/testdb", config.DatabaseURL)
	assert.Equal(t, 20, config.DatabaseMaxConns)
	assert.Equal(t, "redis://localhost:6379", config.RedisURL)
	assert.Equal(t, 1, config.RedisDB)
	assert.Equal(t, "password", config.RedisPassword)
}
