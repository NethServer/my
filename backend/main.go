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

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/methods"
	"github.com/nethesis/my/backend/middleware"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services"
)

func main() {
	// Load .env file if exists (optional, won't fail if missing)
	_ = godotenv.Load()

	// Init logger with zerolog
	err := logger.InitFromEnv("backend")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize logger")
	}

	// Init configuration
	configuration.Init()

	// Initialize database connection
	err = database.Init()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize database connection")
	}

	// Initialize Redis cache
	err = cache.InitRedis()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize Redis cache")
	}

	// Start background statistics updater
	cache.InitAndStartStatsCacheManager(services.NewLogtoManagementClient())

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

		// Standard CRUD operations - role-based (Admin and Support can manage systems)
		systemsGroup := customAuth.Group("/systems", middleware.RequireAnyUserRole("Admin", "Support"))
		{
			systemsGroup.GET("", methods.GetSystems)
			systemsGroup.GET("/:id", methods.GetSystem)
			systemsGroup.POST("", methods.CreateSystem)
			systemsGroup.PUT("/:id", methods.UpdateSystem)
			systemsGroup.DELETE("/:id", methods.DeleteSystem)
			systemsGroup.POST("/:id/regenerate-secret", methods.RegenerateSystemSecret) // Regenerate system secret

			// Inventory endpoints
			systemsGroup.GET("/:id/inventory", methods.GetSystemInventoryHistory)                      // Get paginated inventory history
			systemsGroup.GET("/:id/inventory/latest", methods.GetSystemLatestInventory)                // Get latest inventory
			systemsGroup.GET("/:id/inventory/changes", methods.GetSystemInventoryChanges)              // Get changes summary
			systemsGroup.GET("/:id/inventory/changes/latest", methods.GetSystemLatestInventoryChanges) // Get latest batch changes summary
			systemsGroup.GET("/:id/inventory/diffs", methods.GetSystemInventoryDiffs)                  // Get paginated diffs
			systemsGroup.GET("/:id/inventory/diffs/latest", methods.GetSystemLatestInventoryDiff)      // Get latest diff
		}

		// ===========================================
		// BUSINESS HIERARCHY - Organization role-based
		// Owner > Distributor > Reseller > Customer
		// ===========================================

		// Distributors - only Owner can manage distributors
		distributorsGroup := customAuth.Group("/distributors", middleware.RequireOrgRole("Owner"))
		{
			distributorsGroup.GET("", methods.GetDistributors)
			distributorsGroup.GET("/:id", methods.GetDistributor)
			distributorsGroup.POST("", methods.CreateDistributor)
			distributorsGroup.PUT("/:id", methods.UpdateDistributor)
			distributorsGroup.DELETE("/:id", methods.DeleteDistributor)
		}

		// Resellers - Owner and Distributors can manage resellers
		resellersGroup := customAuth.Group("/resellers", middleware.RequireAnyOrgRole("Owner", "Distributor"))
		{
			resellersGroup.GET("", methods.GetResellers)
			resellersGroup.GET("/:id", methods.GetReseller)
			resellersGroup.POST("", methods.CreateReseller)
			resellersGroup.PUT("/:id", methods.UpdateReseller)
			resellersGroup.DELETE("/:id", methods.DeleteReseller)
		}

		// Customers - Owner, Distributors and Resellers can manage customers
		customersGroup := customAuth.Group("/customers", middleware.RequireAnyOrgRole("Owner", "Distributor", "Reseller"))
		{
			customersGroup.GET("", methods.GetCustomers)
			customersGroup.GET("/:id", methods.GetCustomer)
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
			accountsGroup.GET("", methods.GetAccounts)                         // List accounts with organization filtering
			accountsGroup.GET("/:id", methods.GetAccount)                      // Get single account with hierarchical validation
			accountsGroup.POST("", methods.CreateAccount)                      // Create new account with hierarchical validation
			accountsGroup.PUT("/:id", methods.UpdateAccount)                   // Update existing account
			accountsGroup.PATCH("/:id/password", methods.ResetAccountPassword) // Reset account password
			accountsGroup.DELETE("/:id", methods.DeleteAccount)                // Delete account
		}

		// Roles endpoints - for role selection in account creation
		customAuth.GET("/roles", methods.GetRoles)
		customAuth.GET("/organization-roles", methods.GetOrganizationRoles)

		// Organizations endpoint - for organization selection in account creation
		customAuth.GET("/organizations", methods.GetOrganizations)

		// Applications endpoint - filtered third-party applications based on user access
		customAuth.GET("/applications", methods.GetApplications)

		// System statistics endpoint - require management permissions
		customAuth.GET("/stats", middleware.RequireOrgRole("Owner"), methods.GetStats)
	}

	// Handle missing endpoints
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, response.NotFound("API not found", nil))
	})

	// Run server
	logger.LogServiceStart("backend", "1.0.0", configuration.Config.ListenAddress)
	if err := router.Run(configuration.Config.ListenAddress); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start server")
	}
}
