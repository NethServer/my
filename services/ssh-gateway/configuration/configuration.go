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
	"time"

	"github.com/nethesis/my/services/ssh-gateway/logger"
)

// Configuration holds all service configuration
type Configuration struct {
	// SSH server
	SSHListenAddress string `json:"ssh_listen_address"`
	SSHHostKeyPath   string `json:"ssh_host_key_path"`

	// HTTP health check
	HTTPListenAddress string `json:"http_listen_address"`

	// Database
	DatabaseURL string `json:"database_url"`

	// Backend API
	BackendURL string `json:"backend_url"`

	// Support service
	SupportServiceURL     string `json:"support_service_url"`
	SupportInternalSecret string `json:"-"`

	// Redis configuration
	RedisURL      string `json:"redis_url"`
	RedisDB       int    `json:"redis_db"`
	RedisPassword string `json:"redis_password"`

	// Auth flow configuration
	AuthNonceTTL     time.Duration `json:"auth_nonce_ttl"`
	AuthPollInterval time.Duration `json:"auth_poll_interval"`
	AuthPollTimeout  time.Duration `json:"auth_poll_timeout"`
	AppURL           string        `json:"app_url"`
	KeyAuthTTL       time.Duration `json:"key_auth_ttl"`

	// Logging
	LogLevel  string `json:"log_level"`
	LogFormat string `json:"log_format"`
}

// Config is the global configuration instance
var Config = Configuration{}

// Init initializes configuration from environment variables
func Init() {
	// SSH server
	Config.SSHListenAddress = getStringWithDefault("SSH_LISTEN_ADDRESS", "0.0.0.0:2222")
	Config.SSHHostKeyPath = getStringWithDefault("SSH_HOST_KEY_PATH", "./ssh_host_key")

	// HTTP health check
	Config.HTTPListenAddress = getStringWithDefault("HTTP_LISTEN_ADDRESS", "127.0.0.1:8083")

	// Database
	if os.Getenv("DATABASE_URL") != "" {
		Config.DatabaseURL = os.Getenv("DATABASE_URL")
	} else {
		logger.LogConfigLoad("env", "DATABASE_URL", false, fmt.Errorf("DATABASE_URL variable is empty"))
	}

	// Backend API
	Config.BackendURL = getStringWithDefault("BACKEND_URL", "http://localhost:8080")

	// Support service
	Config.SupportServiceURL = getStringWithDefault("SUPPORT_SERVICE_URL", "http://localhost:8082")
	Config.SupportInternalSecret = os.Getenv("SUPPORT_INTERNAL_SECRET")
	if Config.SupportInternalSecret == "" {
		logger.Fatal().Msg("SUPPORT_INTERNAL_SECRET is required")
	}

	// Redis configuration
	Config.RedisURL = getStringWithDefault("REDIS_URL", "redis://localhost:6379")
	Config.RedisDB = parseIntWithDefault("REDIS_DB", 0)
	Config.RedisPassword = os.Getenv("REDIS_PASSWORD")

	// Auth flow configuration
	Config.AuthNonceTTL = parseDurationWithDefault("AUTH_NONCE_TTL", 5*time.Minute)
	Config.AuthPollInterval = parseDurationWithDefault("AUTH_POLL_INTERVAL", 500*time.Millisecond)
	Config.AuthPollTimeout = parseDurationWithDefault("AUTH_POLL_TIMEOUT", 5*time.Minute)
	Config.AppURL = getStringWithDefault("APP_URL", "https://my.localtest.me")
	Config.KeyAuthTTL = parseDurationWithDefault("KEY_AUTH_TTL", 4*time.Hour)

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

func getStringWithDefault(envVar string, defaultValue string) string {
	if envValue := os.Getenv(envVar); envValue != "" {
		return envValue
	}
	return defaultValue
}
