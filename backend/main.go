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
	"os"

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
	"github.com/nethesis/my/backend/pkg/version"
	"github.com/nethesis/my/backend/response"
)

func main() {
	// Load .env file if exists (optional, won't fail if missing)
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	err := godotenv.Load(envFile)

	// Init logger with zerolog
	loggerErr := logger.InitFromEnv("backend")
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
		logger.Fatal().Err(err).Msg("Failed to initialize database connection")
	}

	// Initialize Redis cache
	err = cache.InitRedis()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize Redis cache")
	}

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

			// System totals endpoint
			systemsGroup.GET("/totals", methods.GetSystemsTotals) // Get systems totals with liveness status

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

		// Distributors - local-first approach with Logto sync
		distributorsGroup := customAuth.Group("/distributors", middleware.RequireOrgRole("Owner"))
		{
			distributorsGroup.POST("", methods.CreateDistributor)       // Create distributor (Owner only - validated in handler)
			distributorsGroup.GET("", methods.GetDistributors)          // List distributors with pagination
			distributorsGroup.GET("/:id", methods.GetDistributor)       // Get distributor (Owner only - validated in handler)
			distributorsGroup.PUT("/:id", methods.UpdateDistributor)    // Update distributor (Owner only - validated in handler)
			distributorsGroup.DELETE("/:id", methods.DeleteDistributor) // Delete distributor (Owner only - validated in handler)

			// Distributors totals endpoint - accessible based on hierarchy
			distributorsGroup.GET("/totals", methods.GetDistributorsTotals)
		}

		// Resellers - local-first approach with Logto sync
		resellersGroup := customAuth.Group("/resellers", middleware.RequireAnyOrgRole("Owner", "Distributors"))
		{
			resellersGroup.POST("", methods.CreateReseller)       // Create reseller (Owner/Distributor - validated in handler)
			resellersGroup.GET("", methods.GetResellers)          // List resellers with pagination
			resellersGroup.GET("/:id", methods.GetReseller)       // Get reseller (RBAC validated in handler)
			resellersGroup.PUT("/:id", methods.UpdateReseller)    // Update reseller (RBAC validated in handler)
			resellersGroup.DELETE("/:id", methods.DeleteReseller) // Delete reseller (RBAC validated in handler)

			// Resellers totals endpoint - accessible based on hierarchy
			resellersGroup.GET("/totals", methods.GetResellersTotals)
		}

		// Customers - local-first approach with Logto sync
		customersGroup := customAuth.Group("/customers", middleware.RequireAnyOrgRole("Owner", "Distributors", "Reseller"))
		{
			customersGroup.POST("", methods.CreateCustomer)       // Create customer (Owner/Distributor/Reseller - validated in handler)
			customersGroup.GET("", methods.GetCustomers)          // List customers with pagination
			customersGroup.GET("/:id", methods.GetCustomer)       // Get customer (RBAC validated in handler)
			customersGroup.PUT("/:id", methods.UpdateCustomer)    // Update customer (RBAC validated in handler)
			customersGroup.DELETE("/:id", methods.DeleteCustomer) // Delete customer (RBAC validated in handler)

			// Customers totals endpoint - accessible based on hierarchy
			customersGroup.GET("/totals", methods.GetCustomersTotals)
		}

		// ===========================================
		// USERS MANAGEMENT - Permission-based
		// ===========================================

		// Users - Basic authentication required, hierarchical validation in handlers
		usersGroup := customAuth.Group("/users", middleware.RequireUserRole("Admin"))
		{
			usersGroup.GET("", methods.GetUsers)                         // List users with organization filtering
			usersGroup.GET("/:id", methods.GetUser)                      // Get single user with hierarchical validation
			usersGroup.POST("", methods.CreateUser)                      // Create new user with hierarchical validation
			usersGroup.PUT("/:id", methods.UpdateUser)                   // Update existing user
			usersGroup.PATCH("/:id/password", methods.ResetUserPassword) // Reset user password
			usersGroup.DELETE("/:id", methods.DeleteUser)                // Delete user

			// Users totals endpoint - accessible based on hierarchy
			usersGroup.GET("/totals", methods.GetUsersTotals)
		}

		// Roles endpoints - for role selection in user creation
		customAuth.GET("/roles", methods.GetRoles)
		customAuth.GET("/organization-roles", methods.GetOrganizationRoles)

		// Organizations endpoint - for organization selection in user creation
		customAuth.GET("/organizations", methods.GetOrganizations)

		// Applications endpoint - filtered third-party applications based on user access
		customAuth.GET("/applications", methods.GetApplications)

	}

	// Handle missing endpoints
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, response.NotFound("API not found", nil))
	})

	// Run server
	logger.LogServiceStart("backend", version.Version, configuration.Config.ListenAddress)
	if err := router.Run(configuration.Config.ListenAddress); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start server")
	}
}
