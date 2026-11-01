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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	gossh "golang.org/x/crypto/ssh"

	"github.com/nethesis/my/services/ssh-gateway/configuration"
	"github.com/nethesis/my/services/ssh-gateway/gateway"
	"github.com/nethesis/my/services/ssh-gateway/logger"
	"github.com/nethesis/my/services/ssh-gateway/pkg/version"
)

func main() {
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	envErr := godotenv.Load(envFile)

	if err := logger.InitFromEnv("ssh-gateway"); err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize logger")
	}
	if envErr != nil {
		logger.Warn().Err(envErr).Str("env_file", envFile).Msg("environment file not loaded, using the process environment")
	}

	configuration.Init()

	opt, err := redis.ParseURL(configuration.Config.RedisURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("invalid REDIS_URL")
	}
	opt.DB = configuration.Config.RedisDB
	if configuration.Config.RedisPassword != "" {
		opt.Password = configuration.Config.RedisPassword
	}
	rdb := redis.NewClient(opt)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		logger.Fatal().Err(err).Msg("cannot connect to Redis")
	}

	hostKey, err := gateway.LoadOrGenerateHostKey(configuration.Config.SSHHostKey, configuration.Config.SSHHostKeyPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot load the SSH host key")
	}
	// operators compare it with the documented one at their first connection
	logger.Info().Str("fingerprint", gossh.FingerprintSHA256(hostKey.PublicKey())).Msg("ssh host key loaded")

	server := gateway.NewServer(gateway.NewAuthenticator(rdb), hostKey, configuration.Config.AuthTimeout)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    200,
			"message": "service healthy",
			"data":    map[string]interface{}{"service": "ssh-gateway", "version": version.Get()},
		})
	})
	mux.Handle("GET /api/ssh", server)

	httpServer := &http.Server{
		Addr:              configuration.Config.HTTPListenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		logger.LogServiceStart("ssh-gateway", version.Version, configuration.Config.HTTPListenAddress)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
	_ = rdb.Close()
	logger.Info().Msg("server exited")
}
