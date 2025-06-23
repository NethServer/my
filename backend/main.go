/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package main

import (
	"net/http"

	"github.com/fatih/structs"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logs"
	"github.com/nethesis/my/backend/methods"
	"github.com/nethesis/my/backend/middleware"
	"github.com/nethesis/my/backend/response"
)

func main() {
	// Load .env file if exists (optional, won't fail if missing)
	_ = godotenv.Load()

	// Init logs
	logs.Init("logto_backend")

	// Init configuration
	configuration.Init()

	// Initialize demo data for all entities
	methods.InitSystemsStorage()
	methods.InitDistributorsStorage()
	methods.InitResellersStorage()
	methods.InitCustomersStorage()

	// Init router
	router := gin.Default()

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
		c.JSON(http.StatusOK, structs.Map(response.StatusOK{
			Code:    200,
			Message: "service healthy",
			Data:    nil,
		}))
	})

	// Protected routes
	protected := api.Group("/", middleware.LogtoAuthMiddleware())
	{
		// User profile
		protected.GET("/profile", methods.GetProfile)

		// Protected resource
		protected.GET("/protected", methods.GetProtectedResource)

		// ===========================================
		// SYSTEMS - Hybrid approach
		// ===========================================

		// Standard CRUD operations - role-based (clean & simple)
		systemsGroup := protected.Group("/systems", middleware.AutoRoleRBAC("Support"))
		{
			systemsGroup.POST("", methods.CreateSystem)
			systemsGroup.GET("", methods.GetSystems)
			systemsGroup.PUT("/:id", methods.UpdateSystem)
			systemsGroup.DELETE("/:id", methods.DeleteSystem)
			systemsGroup.GET("/subscriptions", methods.GetSystemSubscriptions)
		}

		// Special/Sensitive operations - scope-based (granular control)
		systemsSpecial := protected.Group("/systems")
		{
			// System management operations - require specific permissions
			systemsSpecial.POST("/:id/restart", middleware.RequireScope("manage:systems"), methods.RestartSystem)
			systemsSpecial.PUT("/:id/enable", middleware.RequireScope("manage:systems"), methods.EnableSystem)

			// Dangerous operations - require admin permissions
			systemsSpecial.POST("/:id/factory-reset", middleware.RequireScope("admin:systems"), methods.FactoryResetSystem)
			systemsSpecial.DELETE("/:id/destroy", middleware.RequireScope("destroy:systems"), methods.DestroySystem)

			// Audit operations - require audit permissions
			systemsSpecial.GET("/:id/logs", middleware.RequireScope("audit:systems"), methods.GetSystemLogs)
			systemsSpecial.GET("/audit", middleware.RequireScope("audit:systems"), methods.GetSystemsAudit)

			// Backup operations - require backup permissions
			systemsSpecial.POST("/:id/backup", middleware.RequireScope("backup:systems"), methods.BackupSystem)
			systemsSpecial.POST("/:id/restore", middleware.RequireScope("backup:systems"), methods.RestoreSystem)
		}

		// ===========================================
		// HIERARCHY - Organization role-based
		// ===========================================

		// Distributors - only God can manage
		distributorsGroup := protected.Group("/distributors", middleware.AutoOrganizationRoleRBAC("God"))
		{
			distributorsGroup.POST("", methods.CreateDistributor)
			distributorsGroup.GET("", methods.GetDistributors)
			distributorsGroup.PUT("/:id", methods.UpdateDistributor)
			distributorsGroup.DELETE("/:id", methods.DeleteDistributor)
		}

		// Resellers - God and Distributors can manage
		resellersGroup := protected.Group("/resellers", middleware.RequireAnyOrganizationRole("God", "Distributor"))
		{
			resellersGroup.POST("", methods.CreateReseller)
			resellersGroup.GET("", methods.GetResellers)
			resellersGroup.PUT("/:id", methods.UpdateReseller)
			resellersGroup.DELETE("/:id", methods.DeleteReseller)
		}

		// Customers - God, Distributors, and Resellers can manage
		customersGroup := protected.Group("/customers", middleware.RequireAnyOrganizationRole("God", "Distributor", "Reseller"))
		{
			customersGroup.POST("", methods.CreateCustomer)
			customersGroup.GET("", methods.GetCustomers)
			customersGroup.PUT("/:id", methods.UpdateCustomer)
			customersGroup.DELETE("/:id", methods.DeleteCustomer)
		}

		// Quick stats endpoint - management roles only
		protected.GET("/stats", middleware.RequireAnyOrganizationRole("God", "Distributor"), func(c *gin.Context) {
			c.JSON(http.StatusOK, structs.Map(response.StatusOK{
				Code:    200,
				Message: "system statistics",
				Data: gin.H{
					"distributors": 1,
					"resellers":    2,
					"customers":    2,
					"systems":      2,
					"timestamp":    "2025-01-20T10:30:00Z",
				},
			}))
		})
	}

	// Handle missing endpoints
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "API not found",
			Data:    nil,
		}))
	})

	// Run server
	logs.Logs.Printf("[INFO][MAIN] Starting server on %s", configuration.Config.ListenAddress)
	router.Run(configuration.Config.ListenAddress)
}
