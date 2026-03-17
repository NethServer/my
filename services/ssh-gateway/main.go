/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/nethesis/my/services/ssh-gateway/configuration"
	"github.com/nethesis/my/services/ssh-gateway/database"
	"github.com/nethesis/my/services/ssh-gateway/logger"
	"github.com/nethesis/my/services/ssh-gateway/pkg/version"
	"github.com/nethesis/my/services/ssh-gateway/ssh"
)

func main() {
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	err := godotenv.Load(envFile)

	loggerErr := logger.InitFromEnv("ssh-gateway")
	if loggerErr != nil {
		logger.Fatal().Err(loggerErr).Msg("Failed to initialize logger")
	}

	if err == nil {
		logger.Info().Str("component", "env").Str("operation", "config_load").
			Str("config_type", "environment").Str("env_file", envFile).Bool("success", true).
			Msg("environment configuration loaded")
	} else {
		logger.Warn().Str("component", "env").Str("operation", "config_load").
			Str("config_type", "environment").Str("env_file", envFile).Bool("success", false).
			Err(err).Msg("environment configuration not loaded (using system environment)")
	}

	configuration.Init()

	// Initialize Redis client
	opt, err := redis.ParseURL(configuration.Config.RedisURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to parse Redis URL")
	}
	opt.DB = configuration.Config.RedisDB
	if configuration.Config.RedisPassword != "" {
		opt.Password = configuration.Config.RedisPassword
	}
	opt.PoolSize = 10
	opt.MinIdleConns = 2
	opt.ConnMaxIdleTime = 5 * time.Minute
	opt.ConnMaxLifetime = 30 * time.Minute

	redisClient := redis.NewClient(opt)

	// Initialize database for access logs
	if err := database.Init(); err != nil {
		logger.Warn().Err(err).Msg("Database not available, SSH access logs will not be persisted")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	logger.Info().
		Str("redis_url", logger.SanitizeConnectionURL(configuration.Config.RedisURL)).
		Int("redis_db", opt.DB).
		Msg("Redis client initialized")

	// Create auth handler and SSH server
	authHandler := ssh.NewAuthHandler(redisClient)
	sshServer := ssh.NewServer(authHandler)

	// Start HTTP health check server
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"code":    200,
			"message": "service healthy",
			"data": map[string]interface{}{
				"service": "ssh-gateway",
				"status":  "healthy",
				"version": version.Get(),
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	})

	httpServer := &http.Server{
		Addr:    configuration.Config.HTTPListenAddress,
		Handler: mux,
	}

	// Start HTTP server in background
	go func() {
		logger.Info().
			Str("listen_address", configuration.Config.HTTPListenAddress).
			Msg("HTTP health check server starting")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	// Start SSH server in background
	go func() {
		logger.LogServiceStart("ssh-gateway", version.Version, configuration.Config.SSHListenAddress)
		if err := sshServer.ListenAndServe(); err != nil {
			logger.Fatal().Err(err).Msg("SSH server failed")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := sshServer.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("SSH server shutdown error")
	}

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	}

	if err := redisClient.Close(); err != nil {
		logger.Error().Err(err).Msg("Failed to close Redis connection")
	}

	if err := database.Close(); err != nil {
		logger.Error().Err(err).Msg("Failed to close database connection")
	}

	fmt.Println("") // Clean line after ^C
	logger.Info().Msg("Server exited")
}
