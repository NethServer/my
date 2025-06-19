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

		// Admin only endpoints
		admin := protected.Group("/admin", middleware.RequireRole("admin"))
		{
			admin.GET("/users", func(c *gin.Context) {
				c.JSON(http.StatusOK, structs.Map(response.StatusOK{
					Code:    200,
					Message: "admin users list",
					Data:    []string{"user1", "user2"},
				}))
			})
		}

		// Endpoints that require specific scopes
		scoped := protected.Group("/data", middleware.RequireScope("read:data"))
		{
			scoped.GET("/sensitive", func(c *gin.Context) {
				c.JSON(http.StatusOK, structs.Map(response.StatusOK{
					Code:    200,
					Message: "sensitive data accessed",
					Data:    gin.H{"data": "very sensitive information"},
				}))
			})
		}
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
	router.Run(configuration.Config.ListenAddress)
}
