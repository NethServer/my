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
	// PUBLIC SYSTEM REGISTRATION ENDPOINT
	// ===========================================
	api.POST("/systems/register", methods.RegisterSystem)

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

	// Apply impersonation audit middleware to all routes EXCEPT the impersonation management routes
	customAuthWithAudit := customAuth.Group("/", middleware.ImpersonationAuditMiddleware())

	{
		// Authentication endpoints
		customAuth.POST("/auth/logout", middleware.DisableOnImpersonate(), methods.Logout)

		// Consent-based impersonation endpoints (no audit to avoid recursion)
		impersonateGroup := customAuth.Group("/impersonate")
		{
			// Consent management endpoints
			impersonateGroup.POST("/consent", methods.EnableImpersonationConsent)
			impersonateGroup.DELETE("/consent", methods.DisableImpersonationConsent)
			impersonateGroup.GET("/consent", methods.GetImpersonationConsentStatus)

			// Impersonation endpoints
			impersonateGroup.GET("/status", methods.GetImpersonationStatus)
			impersonateGroup.POST("", middleware.RequirePermission("impersonate:users"), methods.ImpersonateUserWithConsent) // Only users with impersonate:users permission
			impersonateGroup.DELETE("", methods.ExitImpersonationWithAudit)

			// Session management endpoints
			impersonateGroup.GET("/sessions", methods.GetImpersonationSessions)
			impersonateGroup.GET("/sessions/:session_id", methods.GetImpersonationSession)
			impersonateGroup.GET("/sessions/:session_id/audit", methods.GetSessionAudit)
		}

		// User profile endpoints using custom JWT (with audit for impersonation)
		customAuthWithAudit.GET("/me", methods.GetCurrentUser)
		customAuthWithAudit.POST("/me/change-password", middleware.DisableOnImpersonate(), methods.ChangePassword)
		customAuthWithAudit.POST("/me/change-info", middleware.DisableOnImpersonate(), methods.ChangeInfo)

		// ===========================================
		// SYSTEMS - resource-based permission validation (read:systems for GET, manage:systems for POST/PUT/DELETE)
		// ===========================================
		systemsGroup := customAuthWithAudit.Group("/systems", middleware.RequireResourcePermission("systems"))
		{
			// CRUD operations
			systemsGroup.POST("", methods.CreateSystem)                     // Create system (manage:systems required)
			systemsGroup.GET("", methods.GetSystems)                        // List systems (read:systems required)
			systemsGroup.GET("/:id", methods.GetSystem)                     // Get system (read:systems required)
			systemsGroup.PUT("/:id", methods.UpdateSystem)                  // Update system (manage:systems required)
			systemsGroup.DELETE("/:id", methods.DeleteSystem)               // Soft-delete system (manage:systems required)
			systemsGroup.PATCH("/:id/restore", methods.RestoreSystem)       // Restore soft-deleted system (manage:systems required)
			systemsGroup.PATCH("/:id/suspend", methods.SuspendSystem)       // Suspend system (manage:systems required)
			systemsGroup.PATCH("/:id/reactivate", methods.ReactivateSystem) // Reactivate suspended system (manage:systems required)

			// Systems totals and trend endpoints (read:systems required)
			systemsGroup.GET("/totals", methods.GetSystemsTotals)
			systemsGroup.GET("/trend", methods.GetSystemsTrend)

			// System actions
			systemsGroup.POST("/:id/regenerate-secret", methods.RegenerateSystemSecret) // Regenerate system secret

			// Export endpoint
			systemsGroup.GET("/export", methods.ExportSystems) // Export systems to CSV or PDF with applied filters

			// Inventory endpoints
			systemsGroup.GET("/:id/inventory", methods.GetSystemInventoryHistory)                      // Get paginated inventory history
			systemsGroup.GET("/:id/inventory/latest", methods.GetSystemLatestInventory)                // Get latest inventory
			systemsGroup.GET("/:id/inventory/changes", methods.GetSystemInventoryChanges)              // Get changes summary
			systemsGroup.GET("/:id/inventory/changes/latest", methods.GetSystemLatestInventoryChanges) // Get latest batch changes summary
			systemsGroup.GET("/:id/inventory/diffs", methods.GetSystemInventoryDiffs)                  // Get paginated diffs
			systemsGroup.GET("/:id/inventory/diffs/latest", methods.GetSystemLatestInventoryDiff)      // Get latest diff
		}

		// ===========================================
		// FILTERS - For UI dropdowns
		// ===========================================
		filtersGroup := customAuthWithAudit.Group("/filters")
		{
			// Systems filters (read:systems required)
			systemsFiltersGroup := filtersGroup.Group("/systems", middleware.RequireResourcePermission("systems"))
			{
				systemsFiltersGroup.GET("/products", methods.GetFilterProducts)           // Get unique product types
				systemsFiltersGroup.GET("/created-by", methods.GetFilterCreatedBy)        // Get users who created systems
				systemsFiltersGroup.GET("/versions", methods.GetFilterVersions)           // Get unique versions
				systemsFiltersGroup.GET("/organizations", methods.GetFilterOrganizations) // Get organizations with systems
			}

			// Applications filters (read:applications required)
			appsFiltersGroup := filtersGroup.Group("/applications", middleware.RequireResourcePermission("applications"))
			{
				appsFiltersGroup.GET("/types", methods.GetApplicationTypes)                 // Get available application types
				appsFiltersGroup.GET("/versions", methods.GetApplicationVersions)           // Get available versions
				appsFiltersGroup.GET("/systems", methods.GetApplicationSystems)             // Get available systems
				appsFiltersGroup.GET("/organizations", methods.GetApplicationOrganizations) // Get available organizations for assignment
			}

			// Users filters (read:users required)
			usersFiltersGroup := filtersGroup.Group("/users", middleware.RequireResourcePermission("users"))
			{
				usersFiltersGroup.GET("/roles", methods.GetRoles)                            // Get available user roles
				usersFiltersGroup.GET("/organizations", methods.GetFilterUsersOrganizations) // Get organizations for user filtering
			}
		}

		// ===========================================
		// BUSINESS HIERARCHY - Permission-based with org_permissions
		// Owner > Distributor > Reseller > Customer
		// ===========================================

		// Distributors - resource-based permission validation (read:distributors for GET, manage:distributors for POST/PUT/DELETE)
		distributorsGroup := customAuthWithAudit.Group("/distributors", middleware.RequireResourcePermission("distributors"))
		{
			distributorsGroup.POST("", methods.CreateDistributor)       // Create distributor (manage:distributors required)
			distributorsGroup.GET("", methods.GetDistributors)          // List distributors (read:distributors required)
			distributorsGroup.GET("/:id", methods.GetDistributor)       // Get distributor (read:distributors required)
			distributorsGroup.PUT("/:id", methods.UpdateDistributor)    // Update distributor (manage:distributors required)
			distributorsGroup.DELETE("/:id", methods.DeleteDistributor) // Delete distributor (manage:distributors required)

			// Distributors totals and trend endpoints (read:distributors required)
			distributorsGroup.GET("/totals", methods.GetDistributorsTotals)
			distributorsGroup.GET("/trend", methods.GetDistributorsTrend)

			// Stats endpoint (users and systems count)
			distributorsGroup.GET("/:id/stats", methods.GetDistributorStats)

			// Suspend and reactivate endpoints (cascade to users)
			distributorsGroup.PATCH("/:id/suspend", methods.SuspendDistributor)       // Suspend distributor and all its users
			distributorsGroup.PATCH("/:id/reactivate", methods.ReactivateDistributor) // Reactivate distributor and cascade-suspended users

			// Export endpoint
			distributorsGroup.GET("/export", methods.ExportDistributors) // Export distributors to CSV or PDF with applied filters
		}

		// Resellers - resource-based permission validation (read:resellers for GET, manage:resellers for POST/PUT/DELETE)
		resellersGroup := customAuthWithAudit.Group("/resellers", middleware.RequireResourcePermission("resellers"))
		{
			resellersGroup.POST("", methods.CreateReseller)       // Create reseller (manage:resellers required)
			resellersGroup.GET("", methods.GetResellers)          // List resellers (read:resellers required)
			resellersGroup.GET("/:id", methods.GetReseller)       // Get reseller (read:resellers required)
			resellersGroup.PUT("/:id", methods.UpdateReseller)    // Update reseller (manage:resellers required)
			resellersGroup.DELETE("/:id", methods.DeleteReseller) // Delete reseller (manage:resellers required)

			// Resellers totals and trend endpoints (read:resellers required)
			resellersGroup.GET("/totals", methods.GetResellersTotals)
			resellersGroup.GET("/trend", methods.GetResellersTrend)

			// Stats endpoint (users and systems count)
			resellersGroup.GET("/:id/stats", methods.GetResellerStats)

			// Suspend and reactivate endpoints (cascade to users)
			resellersGroup.PATCH("/:id/suspend", methods.SuspendReseller)       // Suspend reseller and all its users
			resellersGroup.PATCH("/:id/reactivate", methods.ReactivateReseller) // Reactivate reseller and cascade-suspended users

			// Export endpoint
			resellersGroup.GET("/export", methods.ExportResellers) // Export resellers to CSV or PDF with applied filters
		}

		// Customers - resource-based permission validation (read:customers for GET, manage:customers for POST/PUT/DELETE)
		customersGroup := customAuthWithAudit.Group("/customers", middleware.RequireResourcePermission("customers"))
		{
			customersGroup.POST("", methods.CreateCustomer)       // Create customer (manage:customers required)
			customersGroup.GET("", methods.GetCustomers)          // List customers (read:customers required)
			customersGroup.GET("/:id", methods.GetCustomer)       // Get customer (read:customers required)
			customersGroup.PUT("/:id", methods.UpdateCustomer)    // Update customer (manage:customers required)
			customersGroup.DELETE("/:id", methods.DeleteCustomer) // Delete customer (manage:customers required)

			// Customers totals and trend endpoints (read:customers required)
			customersGroup.GET("/totals", methods.GetCustomersTotals)
			customersGroup.GET("/trend", methods.GetCustomersTrend)

			// Stats endpoint (users and systems count)
			customersGroup.GET("/:id/stats", methods.GetCustomerStats)

			// Suspend and reactivate endpoints (cascade to users)
			customersGroup.PATCH("/:id/suspend", methods.SuspendCustomer)       // Suspend customer and all its users
			customersGroup.PATCH("/:id/reactivate", methods.ReactivateCustomer) // Reactivate customer and cascade-suspended users

			// Export endpoint
			customersGroup.GET("/export", methods.ExportCustomers) // Export customers to CSV or PDF with applied filters
		}

		// ===========================================
		// USERS - resource-based permission validation (read:users for GET, manage:users for POST/PUT/PATCH/DELETE)
		// ===========================================
		usersGroup := customAuthWithAudit.Group("/users", middleware.RequireResourcePermission("users"))
		{
			// CRUD operations
			usersGroup.POST("", methods.CreateUser)                                             // Create user (manage:users required)
			usersGroup.GET("", methods.GetUsers)                                                // List users (read:users required)
			usersGroup.GET("/:id", methods.GetUser)                                             // Get user (read:users required)
			usersGroup.PUT("/:id", middleware.PreventSelfModification(), methods.UpdateUser)    // Update user (manage:users required, prevent self-modification)
			usersGroup.DELETE("/:id", middleware.PreventSelfModification(), methods.DeleteUser) // Delete user (manage:users required, prevent self-modification)

			// Users totals and trend endpoints (read:users required)
			usersGroup.GET("/totals", methods.GetUsersTotals)
			usersGroup.GET("/trend", methods.GetUsersTrend)

			// User actions (manage:users required, prevent self-modification)
			usersGroup.PATCH("/:id/password", middleware.PreventSelfModification(), methods.ResetUserPassword) // Reset user password
			usersGroup.PATCH("/:id/suspend", middleware.PreventSelfModification(), methods.SuspendUser)        // Suspend user
			usersGroup.PATCH("/:id/reactivate", middleware.PreventSelfModification(), methods.ReactivateUser)  // Reactivate suspended user

			// Export endpoint
			usersGroup.GET("/export", methods.ExportUsers) // Export users to CSV or PDF with applied filters
		}

		// ===========================================
		// APPLICATIONS - resource-based permission validation (read:applications for GET, manage:applications for POST/PUT/PATCH/DELETE)
		// ===========================================
		appsGroup := customAuthWithAudit.Group("/applications", middleware.RequireResourcePermission("applications"))
		{
			// CRUD operations
			appsGroup.GET("", methods.GetApplications)          // List applications (read:applications required)
			appsGroup.GET("/:id", methods.GetApplication)       // Get application (read:applications required)
			appsGroup.PUT("/:id", methods.UpdateApplication)    // Update application (manage:applications required)
			appsGroup.DELETE("/:id", methods.DeleteApplication) // Soft-delete application (manage:applications required)

			// Applications totals and trend endpoints (read:applications required)
			appsGroup.GET("/totals", methods.GetApplicationTotals)
			appsGroup.GET("/trend", methods.GetApplicationsTrend)

			// Application actions (manage:applications required)
			appsGroup.PATCH("/:id/assign", methods.AssignApplicationOrganization)     // Assign organization to application
			appsGroup.PATCH("/:id/unassign", methods.UnassignApplicationOrganization) // Remove organization from application
		}

		// ===========================================
		// REBRANDING - per-product rebranding management
		// ===========================================
		rebrandingGroup := customAuthWithAudit.Group("/rebranding")
		{
			// Rebrandable products list (any authenticated user)
			rebrandingGroup.GET("/products", methods.GetRebrandingProducts)

			// Enable/disable rebranding (Owner only, checked in handler)
			rebrandingGroup.PATCH("/:org_id/enable", methods.EnableRebranding)
			rebrandingGroup.PATCH("/:org_id/disable", methods.DisableRebranding)

			// Rebranding status and products for an organization
			rebrandingGroup.GET("/:org_id/status", methods.GetRebrandingStatus)
			rebrandingGroup.GET("/:org_id/products", methods.GetRebrandingOrgProducts)

			// Asset management
			rebrandingGroup.PUT("/:org_id/products/:product_id", methods.UploadRebrandingAssets)
			rebrandingGroup.DELETE("/:org_id/products/:product_id", methods.DeleteRebrandingProduct)
			rebrandingGroup.DELETE("/:org_id/products/:product_id/:asset", methods.DeleteRebrandingAsset)
			rebrandingGroup.GET("/:org_id/products/:product_id/:asset", methods.GetRebrandingAsset)
		}

		// ===========================================
		// METADATA - roles, organizations, third-party apps
		// ===========================================
		customAuthWithAudit.GET("/roles", methods.GetRoles)                                     // Get available user roles
		customAuthWithAudit.GET("/organization-roles", methods.GetOrganizationRoles)            // Get available organization roles
		customAuthWithAudit.GET("/organizations", methods.GetOrganizations)                     // Get organizations for user assignment
		customAuthWithAudit.GET("/third-party-applications", methods.GetThirdPartyApplications) // Get third-party applications filtered by user access

		// ===========================================
		// VALIDATORS - validation endpoints
		// ===========================================
		validatorsGroup := customAuth.Group("/validators")
		{
			validatorsGroup.GET("/vat/:entity_type", validators.ValidateVAT) // Validate VAT number for entity type
		}

	}

	// Handle missing endpoints
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, response.NotFound("api not found", nil))
	})

	// Run server
	logger.LogServiceStart("backend", version.Version, configuration.Config.ListenAddress)
	if err := router.Run(configuration.Config.ListenAddress); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start server")
	}
}
