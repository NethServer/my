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
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nethesis/my/services/ssh-gateway/logger"
)

type Configuration struct {
	// HTTPListenAddress serves the health check and the SSH-over-WebSocket endpoint
	HTTPListenAddress string
	SSHHostKeyPath    string

	// AppURL is where operators approve an access request
	AppURL string
	// BackendURL opens the shells, with the internal secret and a grant
	BackendURL            string
	SupportInternalSecret string

	RedisURL      string
	RedisDB       int
	RedisPassword string

	// AuthTimeout bounds the wait for the browser approval
	AuthTimeout time.Duration
	// KeyAuthTTL is how long an approved SSH key skips the browser
	KeyAuthTTL time.Duration
	// MaxRequestsPerMinute caps the access requests a client IP can open
	MaxRequestsPerMinute int
}

var Config = Configuration{}

func Init() {
	Config.HTTPListenAddress = env("HTTP_LISTEN_ADDRESS", "127.0.0.1:8083")
	Config.SSHHostKeyPath = env("SSH_HOST_KEY_PATH", "./ssh_host_key")
	Config.AppURL = strings.TrimRight(env("APP_URL", "https://my.localtest.me"), "/")
	Config.BackendURL = strings.TrimRight(env("BACKEND_URL", "http://localhost:8080"), "/")
	Config.SupportInternalSecret = os.Getenv("SUPPORT_INTERNAL_SECRET")
	if Config.SupportInternalSecret == "" {
		logger.Fatal().Msg("SUPPORT_INTERNAL_SECRET is required")
	}

	Config.RedisURL = env("REDIS_URL", "redis://localhost:6379")
	Config.RedisDB = envInt("REDIS_DB", 0)
	Config.RedisPassword = os.Getenv("REDIS_PASSWORD")

	Config.AuthTimeout = envDuration("AUTH_TIMEOUT", 5*time.Minute)
	Config.KeyAuthTTL = envDuration("KEY_AUTH_TTL", 4*time.Hour)
	Config.MaxRequestsPerMinute = envInt("MAX_REQUESTS_PER_MINUTE", 10)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v, err := time.ParseDuration(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return fallback
}
