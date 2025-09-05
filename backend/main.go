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
	"github.com/nethesis/my/backend/methods/validators"
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

	// Initialize roles from Logto
	roleNames := cache.GetRoleNames()
	err = roleNames.LoadRoles()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to load roles from Logto - userRolesNames will be empty")
	}

	// Initialize domain validation from Logto
	domainValidation := cache.GetDomainValidation()
	err = domainValidation.LoadDomainValidation()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to load domain validation from Logto - will fallback to tenant ID")
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
		c.JSON(http.StatusOK, response.OK("service healthy", version.Get()))
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
	// Resource-based permissions: read:resource for GET, manage:resource for POST/PUT/PATCH/DELETE
	// ===========================================
	customAuth := api.Group("/", middleware.JWTAuthMiddleware())
	{
		// Authentication endpoints
		customAuth.POST("/auth/logout", methods.Logout)

		// User profile endpoints using custom JWT
		customAuth.GET("/me", methods.GetCurrentUser)
		customAuth.POST("/me/change-password", methods.ChangePassword)
		customAuth.POST("/me/change-info", methods.ChangeInfo)

		// Business operations
		// ===========================================
		// SYSTEMS - Hybrid approach
		// ===========================================

		// Standard CRUD operations - resource-based (read:systems for GET, manage:systems for POST/PUT/DELETE)
		systemsGroup := customAuth.Group("/systems", middleware.RequireResourcePermission("systems"))
		{
			systemsGroup.GET("", methods.GetSystems)
			systemsGroup.GET("/:id", methods.GetSystem)
			systemsGroup.POST("", methods.CreateSystem)
			systemsGroup.POST("/callback", methods.CreateSystemWithCallback) // Create system with callback
			systemsGroup.PUT("/:id", methods.UpdateSystem)
			systemsGroup.DELETE("/:id", methods.DeleteSystem)

			systemsGroup.POST("/:id/regenerate-secret", methods.RegenerateSystemSecret) // Regenerate system secret

			// Dangerous operations requiring specific permissions
			// systemsGroup.DELETE("/:id/destroy", middleware.RequirePermission("destroy:systems"), methods.DestroySystem) // Complete system destruction (destroy:systems required)

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
		// BUSINESS HIERARCHY - Permission-based with org_permissions
		// Owner > Distributor > Reseller > Customer
		// ===========================================

		// Distributors - resource-based permission validation (read:distributors for GET, manage:distributors for POST/PUT/DELETE)
		distributorsGroup := customAuth.Group("/distributors", middleware.RequireResourcePermission("distributors"))
		{
			distributorsGroup.POST("", methods.CreateDistributor)       // Create distributor (manage:distributors required)
			distributorsGroup.GET("", methods.GetDistributors)          // List distributors (read:distributors required)
			distributorsGroup.GET("/:id", methods.GetDistributor)       // Get distributor (read:distributors required)
			distributorsGroup.PUT("/:id", methods.UpdateDistributor)    // Update distributor (manage:distributors required)
			distributorsGroup.DELETE("/:id", methods.DeleteDistributor) // Delete distributor (manage:distributors required)

			// Distributors totals endpoint (read:distributors required)
			distributorsGroup.GET("/totals", methods.GetDistributorsTotals)
		}

		// Resellers - resource-based permission validation (read:resellers for GET, manage:resellers for POST/PUT/DELETE)
		resellersGroup := customAuth.Group("/resellers", middleware.RequireResourcePermission("resellers"))
		{
			resellersGroup.POST("", methods.CreateReseller)       // Create reseller (manage:resellers required)
			resellersGroup.GET("", methods.GetResellers)          // List resellers (read:resellers required)
			resellersGroup.GET("/:id", methods.GetReseller)       // Get reseller (read:resellers required)
			resellersGroup.PUT("/:id", methods.UpdateReseller)    // Update reseller (manage:resellers required)
			resellersGroup.DELETE("/:id", methods.DeleteReseller) // Delete reseller (manage:resellers required)

			// Resellers totals endpoint (read:resellers required)
			resellersGroup.GET("/totals", methods.GetResellersTotals)
		}

		// Customers - resource-based permission validation (read:customers for GET, manage:customers for POST/PUT/DELETE)
		customersGroup := customAuth.Group("/customers", middleware.RequireResourcePermission("customers"))
		{
			customersGroup.POST("", methods.CreateCustomer)       // Create customer (manage:customers required)
			customersGroup.GET("", methods.GetCustomers)          // List customers (read:customers required)
			customersGroup.GET("/:id", methods.GetCustomer)       // Get customer (read:customers required)
			customersGroup.PUT("/:id", methods.UpdateCustomer)    // Update customer (manage:customers required)
			customersGroup.DELETE("/:id", methods.DeleteCustomer) // Delete customer (manage:customers required)

			// Customers totals endpoint (read:customers required)
			customersGroup.GET("/totals", methods.GetCustomersTotals)
		}

		// ===========================================
		// USERS MANAGEMENT - Permission-based
		// ===========================================

		// Users - Resource-based permission validation (read:users for GET, manage:users for POST/PUT/PATCH/DELETE)
		usersGroup := customAuth.Group("/users", middleware.RequireResourcePermission("users"))
		{
			usersGroup.GET("", methods.GetUsers)                                                               // List users with organization filtering
			usersGroup.GET("/:id", methods.GetUser)                                                            // Get single user with hierarchical validation
			usersGroup.POST("", methods.CreateUser)                                                            // Create new user with hierarchical validation
			usersGroup.PUT("/:id", middleware.PreventSelfModification(), methods.UpdateUser)                   // Update existing user (prevent self-modification)
			usersGroup.PATCH("/:id/password", middleware.PreventSelfModification(), methods.ResetUserPassword) // Reset user password (prevent self-modification)
			usersGroup.PATCH("/:id/suspend", middleware.PreventSelfModification(), methods.SuspendUser)        // Suspend user (prevent self-modification)
			usersGroup.PATCH("/:id/reactivate", middleware.PreventSelfModification(), methods.ReactivateUser)  // Reactivate suspended user (prevent self-modification)
			usersGroup.DELETE("/:id", middleware.PreventSelfModification(), methods.DeleteUser)                // Delete user (prevent self-modification)

			// Users totals endpoint (read:users required)
			usersGroup.GET("/totals", methods.GetUsersTotals)
		}

		// Roles endpoints - for role selection in user creation
		customAuth.GET("/roles", methods.GetRoles)
		customAuth.GET("/organization-roles", methods.GetOrganizationRoles)

		// Organizations endpoint - for organization selection in user creation
		customAuth.GET("/organizations", methods.GetOrganizations)

		// Applications endpoint - filtered third-party applications based on user access
		customAuth.GET("/applications", methods.GetApplications)

		// Validators group - for validation endpoints
		validatorsGroup := customAuth.Group("/validators")
		{
			validatorsGroup.GET("/vat/:entity_type", validators.ValidateVAT)
		}

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
