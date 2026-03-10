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
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/nethesis/my/services/support/configuration"
	"github.com/nethesis/my/services/support/database"
	"github.com/nethesis/my/services/support/logger"
	"github.com/nethesis/my/services/support/methods"
	"github.com/nethesis/my/services/support/middleware"
	"github.com/nethesis/my/services/support/pkg/version"
	"github.com/nethesis/my/services/support/queue"
	"github.com/nethesis/my/services/support/response"
	"github.com/nethesis/my/services/support/session"
	"github.com/nethesis/my/services/support/tunnel"
)

func main() {
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	err := godotenv.Load(envFile)

	loggerErr := logger.InitFromEnv("support")
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

	err = database.Init()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize database")
	}

	err = queue.Init()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize Redis")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize tunnel manager
	tunnelManager := tunnel.NewManager(configuration.Config.MaxTunnels, configuration.Config.MaxStreamsPerTunnel)
	methods.TunnelManager = tunnelManager

	// Set grace period callback: close session when grace period expires without reconnection
	tunnelManager.SetGraceCallback(func(systemID, sessionID string) {
		if err := session.CloseSession(sessionID, "disconnect"); err != nil {
			logger.Error().Err(err).
				Str("system_id", systemID).
				Str("session_id", sessionID).
				Msg("failed to close session after grace period expired")
		}
	})

	// Start session cleaner
	go session.StartCleaner(ctx, func(expiredSessionIDs []string) {
		for _, sessionID := range expiredSessionIDs {
			tunnelManager.CloseBySessionID(sessionID)
		}
	})

	// Start command listener for backend commands via Redis pub/sub
	go methods.StartCommandListener(ctx)

	// #12: Start auth cache invalidation listener
	go middleware.StartAuthCacheInvalidator(ctx)

	// Setup HTTP router (gin.New without default logger to avoid raw query params in logs)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(logger.GinLogger())
	router.Use(logger.SecurityMiddleware())
	router.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPathsRegexs([]string{"^/api/proxy", "^/api/terminal"})))

	if gin.Mode() == gin.DebugMode {
		corsConf := cors.DefaultConfig()
		corsConf.AllowHeaders = []string{"Authorization", "Content-Type", "Accept"}
		corsConf.AllowOrigins = []string{"http://localhost:*", "https://localhost:*", "http://127.0.0.1:*", "https://127.0.0.1:*"}
		corsConf.AllowOriginFunc = func(origin string) bool {
			return strings.HasPrefix(origin, "http://localhost") ||
				strings.HasPrefix(origin, "https://localhost") ||
				strings.HasPrefix(origin, "http://127.0.0.1") ||
				strings.HasPrefix(origin, "https://127.0.0.1")
		}
		router.Use(cors.New(corsConf))
	}

	api := router.Group("/api")

	// Health endpoint (no sensitive data — tunnel details require authenticated admin endpoints)
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, response.OK("service healthy", gin.H{
			"service": "support",
			"status":  "healthy",
			"version": version.Get(),
		}))
	})

	// Tunnel endpoint (WebSocket, requires system Basic Auth, rate-limited per IP + per system_key)
	api.GET("/tunnel", middleware.TunnelRateLimitMiddleware(), middleware.BasicAuthMiddleware(), middleware.SystemKeyRateLimitMiddleware(), methods.HandleTunnel)

	// Internal endpoints: require per-session token from backend (#3/#4)
	internal := api.Group("/")
	internal.Use(middleware.SessionTokenMiddleware(), middleware.SessionRateLimitMiddleware())

	internal.GET("/terminal/:session_id", methods.HandleTerminal)
	internal.GET("/proxy/:session_id/services", methods.ListServices)
	internal.Any("/proxy/:session_id/:service/*path", methods.HandleProxy)

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, response.NotFound("api not found", nil))
	})

	// Start HTTP server
	srv := &http.Server{
		Addr:    configuration.Config.ListenAddress,
		Handler: router,
	}

	go func() {
		logger.LogServiceStart("support", version.Version, configuration.Config.ListenAddress)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")

	// Close all tunnels
	tunnelManager.CloseAll()

	cancel()

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	if err := queue.Close(); err != nil {
		logger.Error().Err(err).Msg("Failed to close Redis connection")
	}

	if err := database.Close(); err != nil {
		logger.Error().Err(err).Msg("Failed to close database connection")
	}

	logger.Info().Msg("Server exited")
}
