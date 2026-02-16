/*
 * Copyright (C) 2025 Nethesis S.r.l.
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
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/cron"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/methods"
	"github.com/nethesis/my/collect/middleware"
	"github.com/nethesis/my/collect/pkg/version"
	"github.com/nethesis/my/collect/queue"
	"github.com/nethesis/my/collect/response"
	"github.com/nethesis/my/collect/workers"
)

func main() {
	// Load .env file if exists (optional, won't fail if missing)
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	err := godotenv.Load(envFile)

	// Init logger with zerolog
	loggerErr := logger.InitFromEnv("collect")
	if loggerErr != nil {
		logger.Fatal().Err(loggerErr).Msg("Failed to initialize logger")
	}

	// Log which environment file was loaded
	if err == nil {
		logger.Info().
			Str("component", "env").
			Str("operation", "config_load").
			Str("config_type", "environment").
			Str("env_file", envFile).
			Bool("success", true).
			Msg("environment configuration loaded")
	} else {
		logger.Warn().
			Str("component", "env").
			Str("operation", "config_load").
			Str("config_type", "environment").
			Str("env_file", envFile).
			Bool("success", false).
			Err(err).
			Msg("environment configuration not loaded (using system environment)")
	}

	// Init configuration
	configuration.Init()

	// Initialize database connection
	err = database.Init()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize database")
	}

	// Initialize Redis queue
	err = queue.Init()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize Redis queue")
	}

	// Start scalable worker manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workerManager := workers.NewManager()
	if err := workerManager.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start worker manager")
	}

	// Start heartbeat monitor cron job
	heartbeatMonitor := cron.NewHeartbeatMonitor()
	go heartbeatMonitor.Start(ctx)

	// Init router
	router := gin.Default()

	// Add request logging middleware
	router.Use(logger.GinLogger())

	// Add security monitoring middleware
	router.Use(logger.SecurityMiddleware())

	// Add compression
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	// CORS configuration in debug mode
	if gin.Mode() == gin.DebugMode {
		corsConf := cors.DefaultConfig()
		corsConf.AllowHeaders = []string{"Authorization", "Content-Type", "Accept"}
		corsConf.AllowAllOrigins = true
		router.Use(cors.New(corsConf))
	}

	// Define API group
	api := router.Group("/api")

	// Health check endpoint with detailed metrics
	api.GET("/health", func(c *gin.Context) {
		dbStats := database.GetStats()
		if redisStats := queue.GetStats(); redisStats != nil {
			dbStats["redis"] = redisStats
		}

		healthData := map[string]interface{}{
			"service":  "collect",
			"status":   "healthy",
			"workers":  workerManager.GetStatus(),
			"database": dbStats,
			"version":  version.Get(),
		}

		if !workerManager.IsHealthy() {
			c.JSON(http.StatusServiceUnavailable, response.Error(http.StatusServiceUnavailable, "service unhealthy", healthData))
			return
		}

		c.JSON(http.StatusOK, response.OK("service healthy", healthData))
	})

	// ===========================================
	// INVENTORY COLLECTION ENDPOINTS
	// ===========================================

	// System inventory collection with HTTP Basic authentication
	systemsGroup := api.Group("/systems", middleware.BasicAuthMiddleware())
	{
		systemsGroup.GET("/info", methods.GetSystemInfo)
		systemsGroup.POST("/inventory", methods.CollectInventory)
		systemsGroup.POST("/heartbeat", methods.ReceiveHeartbeat)

		// Rebranding endpoints (system fetches its own rebranding config)
		systemsGroup.GET("/rebranding", methods.GetSystemRebranding)
		systemsGroup.GET("/rebranding/:product_id/:asset", methods.GetSystemRebrandingAsset)
	}

	// Handle missing endpoints
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, response.NotFound("api not found", nil))
	})

	// Setup graceful shutdown
	srv := &http.Server{
		Addr:    configuration.Config.ListenAddress,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.LogServiceStart("collect", version.Version, configuration.Config.ListenAddress)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")

	// Stop worker manager gracefully
	if err := workerManager.Stop(); err != nil {
		logger.Error().Err(err).Msg("Failed to stop worker manager gracefully")
	}

	// Cancel context for any remaining operations
	cancel()

	// Shutdown HTTP server
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	// Close Redis connection
	if err := queue.Close(); err != nil {
		logger.Error().Err(err).Msg("Failed to close Redis connection")
	}

	// Close database connection
	if err := database.Close(); err != nil {
		logger.Error().Err(err).Msg("Failed to close database connection")
	}

	logger.Info().Msg("Server exited")
}
