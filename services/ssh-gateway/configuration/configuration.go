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
	// SSHHostKey is the host key itself (PEM), so it survives redeploys on
	// hosts without a persistent disk; otherwise it lives at SSHHostKeyPath
	SSHHostKey     string
	SSHHostKeyPath string

	// BackendURL holds every piece of state: access requests, approved keys,
	// grants and the shell bridge. The gateway has no store of its own.
	BackendURL string
	// SSHGatewaySecret authenticates the gateway on the backend's internal
	// routes; it opens nothing else
	SSHGatewaySecret string

	// AuthTimeout bounds the wait for the browser approval
	AuthTimeout time.Duration
	// HandshakeTimeout bounds a connection until its authentication starts
	HandshakeTimeout time.Duration
	// MaxRequestsPerMinute caps the access requests a client IP can open
	MaxRequestsPerMinute int
	// MaxConnsPerIP and MaxPendingConns cap connections not yet authenticated
	MaxConnsPerIP   int
	MaxPendingConns int
}

var Config = Configuration{}

func Init() {
	Config.HTTPListenAddress = env("HTTP_LISTEN_ADDRESS", "127.0.0.1:8083")
	// a PEM pasted on one line keeps its newlines as \n
	Config.SSHHostKey = strings.ReplaceAll(os.Getenv("SSH_HOST_KEY"), `\n`, "\n")
	Config.SSHHostKeyPath = env("SSH_HOST_KEY_PATH", "./ssh_host_key")
	// on Render the disk does not survive a deploy: a generated key would
	// change at every release and teach operators to ignore the warning
	if Config.SSHHostKey == "" && os.Getenv("RENDER") != "" {
		logger.Fatal().Msg("SSH_HOST_KEY is required on Render")
	}

	Config.BackendURL = strings.TrimRight(env("BACKEND_URL", "http://localhost:8080"), "/")
	Config.SSHGatewaySecret = os.Getenv("SSH_GATEWAY_SECRET")
	if Config.SSHGatewaySecret == "" {
		logger.Fatal().Msg("SSH_GATEWAY_SECRET is required")
	}

	Config.AuthTimeout = envDuration("AUTH_TIMEOUT", 5*time.Minute)
	Config.HandshakeTimeout = envDuration("HANDSHAKE_TIMEOUT", 30*time.Second)
	Config.MaxRequestsPerMinute = envInt("MAX_REQUESTS_PER_MINUTE", 10)
	Config.MaxConnsPerIP = envInt("MAX_CONNS_PER_IP", 10)
	Config.MaxPendingConns = envInt("MAX_PENDING_CONNS", 200)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
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
