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
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/methods"
	"github.com/nethesis/my/backend/middleware"
	"github.com/nethesis/my/backend/response"
)

func main() {
	// Load .env file if exists (optional, won't fail if missing)
	_ = godotenv.Load()

	// Init logger with zerolog
	err := logger.InitFromEnv("nethesis-backend")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize logger")
	}

	// Init configuration
	configuration.Init()

	// Initialize demo data for systems (still using in-memory storage)
	methods.InitSystemsStorage()

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

	// Health check endpoint
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, response.OK("service healthy", nil))
	})

	// ===========================================
	// PUBLIC AUTH ENDPOINTS
	// ===========================================
	api.POST("/auth/exchange", methods.ExchangeToken)
	api.POST("/auth/refresh", methods.RefreshToken)

	// ===========================================
	// STANDARD OAUTH2/OIDC ROUTES (for third-party apps)
	// Uses Logto tokens directly - standard compliance
	// ===========================================
	standardAuth := api.Group("/", middleware.LogtoAuthMiddleware())
	{
		// User business data endpoints (OAuth2/OIDC standard flow)
		userGroup := standardAuth.Group("/user")
		{
			userGroup.GET("/permissions", methods.GetUserPermissions)
			userGroup.GET("/profile", methods.GetUserProfile)
		}
	}

	// ===========================================
	// CUSTOM JWT ROUTES (for resilient apps)
	// Uses our enriched JWT - works offline when Logto is down
	// ===========================================
	customAuth := api.Group("/", middleware.JWTAuthMiddleware())
	{
		// Auth endpoint using custom JWT
		customAuth.GET("/auth/me", methods.GetCurrentUser)

		// Business operations
		// ===========================================
		// SYSTEMS - Hybrid approach
		// ===========================================

		// Standard CRUD operations - role-based (Support can manage systems)
		systemsGroup := customAuth.Group("/systems", middleware.RequireUserRole("Support"))
		{
			systemsGroup.GET("", methods.GetSystems)
			systemsGroup.POST("", methods.CreateSystem)
			systemsGroup.PUT("/:id", methods.UpdateSystem)
			systemsGroup.DELETE("/:id", middleware.RequirePermission("admin:systems"), methods.DeleteSystem) // Admin-only
			systemsGroup.GET("/subscriptions", methods.GetSystemSubscriptions)
		}

		// Special/Sensitive operations - permission-based
		systemsSpecial := customAuth.Group("/systems")
		{
			// System management operations
			systemsSpecial.POST("/:id/restart", middleware.RequirePermission("manage:systems"), methods.RestartSystem)
			systemsSpecial.PUT("/:id/enable", middleware.RequirePermission("manage:systems"), methods.EnableSystem)

			// Dangerous operations - require admin permissions
			systemsSpecial.POST("/:id/factory-reset", middleware.RequirePermission("admin:systems"), methods.FactoryResetSystem)
			systemsSpecial.DELETE("/:id/destroy", middleware.RequirePermission("destroy:systems"), methods.DestroySystem)

			// Log viewing operations
			systemsSpecial.GET("/:id/logs", middleware.RequirePermission("manage:systems"), methods.GetSystemLogs)
			systemsSpecial.GET("/audit", middleware.RequirePermission("manage:systems"), methods.GetSystemsAudit)

			// System backup operations - require admin permissions
			systemsSpecial.POST("/:id/backup", middleware.RequirePermission("admin:systems"), methods.BackupSystem)
			systemsSpecial.POST("/:id/restore", middleware.RequirePermission("admin:systems"), methods.RestoreSystem)
		}

		// ===========================================
		// BUSINESS HIERARCHY - Organization role-based
		// God > Distributor > Reseller > Customer
		// ===========================================

		// Distributors - only God can manage distributors
		distributorsGroup := customAuth.Group("/distributors", middleware.RequireOrgRole("God"))
		{
			distributorsGroup.GET("", methods.GetDistributors)
			distributorsGroup.POST("", methods.CreateDistributor)
			distributorsGroup.PUT("/:id", methods.UpdateDistributor)
			distributorsGroup.DELETE("/:id", methods.DeleteDistributor)
		}

		// Resellers - God and Distributors can manage resellers
		resellersGroup := customAuth.Group("/resellers", middleware.RequireAnyOrgRole("God", "Distributor"))
		{
			resellersGroup.GET("", methods.GetResellers)
			resellersGroup.POST("", methods.CreateReseller)
			resellersGroup.PUT("/:id", methods.UpdateReseller)
			resellersGroup.DELETE("/:id", methods.DeleteReseller)
		}

		// Customers - God, Distributors and Resellers can manage customers
		customersGroup := customAuth.Group("/customers", middleware.RequireAnyOrgRole("God", "Distributor", "Reseller"))
		{
			customersGroup.GET("", methods.GetCustomers)
			customersGroup.POST("", methods.CreateCustomer)
			customersGroup.PUT("/:id", methods.UpdateCustomer)
			customersGroup.DELETE("/:id", methods.DeleteCustomer)
		}

		// ===========================================
		// ACCOUNTS MANAGEMENT - Permission-based
		// ===========================================

		// Accounts - Basic authentication required, hierarchical validation in handlers
		accountsGroup := customAuth.Group("/accounts")
		{
			accountsGroup.GET("", methods.GetAccounts)          // List accounts with organization filtering
			accountsGroup.POST("", methods.CreateAccount)       // Create new account with hierarchical validation
			accountsGroup.PUT("/:id", methods.UpdateAccount)    // Update existing account
			accountsGroup.DELETE("/:id", methods.DeleteAccount) // Delete account
		}

		// Quick stats endpoint - require management permissions
		customAuth.GET("/stats", middleware.RequirePermission("manage:distributors"), func(c *gin.Context) {
			c.JSON(http.StatusOK, response.OK("system statistics", gin.H{
				"distributors": 1,
				"resellers":    2,
				"customers":    2,
				"systems":      2,
				"timestamp":    "2025-01-20T10:30:00Z",
			}))
		})
	}

	// Handle missing endpoints
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, response.NotFound("API not found", nil))
	})

	// Run server
	logger.LogServiceStart("nethesis-backend", "1.0.0", configuration.Config.ListenAddress)
	if err := router.Run(configuration.Config.ListenAddress); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start server")
	}
}
