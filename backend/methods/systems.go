/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services"
)

// In-memory storage for demo purposes
// In production, this should be replaced with a proper database
var systemsStorage = make(map[string]*models.System)
var subscriptionsStorage = make(map[string]*models.SystemSubscription)

// CreateSystem handles POST /api/systems - creates a new system
func CreateSystem(c *gin.Context) {
	// Parse request body
	var request models.CreateSystemRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	// Get current user context
	userID, userOrgRole, userRole := helpers.GetUserContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Create systems service
	systemsService := services.NewSystemsService()

	// Validate organization access
	if err := systemsService.ValidateOrganizationAccess(userID, request.OrganizationID, userOrgRole, userRole); err != nil {
		logger.Warn().
			Str("user_id", userID).
			Str("organization_id", request.OrganizationID).
			Str("user_org_role", userOrgRole).
			Str("user_role", userRole).
			Err(err).
			Msg("Access denied for system creation")
		
		c.JSON(http.StatusForbidden, response.Forbidden("Access denied", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Create system with automatic secret generation
	result, err := systemsService.CreateSystem(&request, userID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Str("system_name", request.Name).
			Msg("Failed to create system")
		
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to create system", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Store system in legacy storage for backward compatibility
	systemsStorage[result.System.ID] = result.System

	// Log the action
	logger.LogBusinessOperation(c, "systems", "create", "system", result.System.ID, true, nil)

	// Return success response with system and credentials
	c.JSON(http.StatusCreated, response.Created("system created successfully", result))
}

// GetSystems handles GET /api/systems - retrieves all systems
func GetSystems(c *gin.Context) {
	// Get current user context
	userID, userOrgRole, userRole := helpers.GetUserContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Create systems service
	systemsService := services.NewSystemsService()

	// Get systems with proper filtering
	systems, err := systemsService.GetSystemsByOrganization(userID, userOrgRole, userRole)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to retrieve systems")
		
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to retrieve systems", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.RequestLogger(c, "systems").Info().
		Str("operation", "list_systems").
		Int("count", len(systems)).
		Msg("Systems list requested")

	// Return systems list
	c.JSON(http.StatusOK, response.OK("systems retrieved successfully", gin.H{"systems": systems, "count": len(systems)}))
}

// GetSystem handles GET /api/systems/:id - retrieves a single system
func GetSystem(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	// Get current user context
	userID, userOrgRole, userRole := helpers.GetUserContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Create systems service
	systemsService := services.NewSystemsService()

	// Get system with access validation
	system, err := systemsService.GetSystemByID(systemID, userID, userOrgRole, userRole)
	if err != nil {
		if err.Error() == "system not found" {
			c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
			return
		}
		
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Str("system_id", systemID).
			Msg("Failed to retrieve system")
		
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to retrieve system", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.RequestLogger(c, "systems").Info().
		Str("operation", "get_system").
		Str("system_id", systemID).
		Msg("System details requested")

	// Return system
	c.JSON(http.StatusOK, response.OK("system retrieved successfully", system))
}

// UpdateSystem handles PUT /api/systems/:id - updates an existing system
func UpdateSystem(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	// Get current user context
	userID, userOrgRole, userRole := helpers.GetUserContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Parse request body
	var request models.UpdateSystemRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	// Create systems service
	systemsService := services.NewSystemsService()

	// Update system with access validation
	system, err := systemsService.UpdateSystem(systemID, &request, userID, userOrgRole, userRole)
	if err != nil {
		if err.Error() == "system not found" {
			c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
			return
		}
		
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Str("system_id", systemID).
			Msg("Failed to update system")
		
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to update system", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "systems", "update", "system", systemID, true, nil)

	// Return updated system
	c.JSON(http.StatusOK, response.OK("system updated successfully", system))
}

// DeleteSystem handles DELETE /api/systems/:id - deletes a system
func DeleteSystem(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	// Get current user context
	userID, userOrgRole, userRole := helpers.GetUserContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Create systems service
	systemsService := services.NewSystemsService()

	// Delete system with access validation
	err := systemsService.DeleteSystem(systemID, userID, userOrgRole, userRole)
	if err != nil {
		if err.Error() == "system not found" {
			c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
			return
		}
		
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Str("system_id", systemID).
			Msg("Failed to delete system")
		
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to delete system", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Also delete related subscription if exists (legacy compatibility)
	delete(subscriptionsStorage, systemID)

	// Log the action
	logger.LogBusinessOperation(c, "systems", "delete", "system", systemID, true, nil)

	// Return success response
	c.JSON(http.StatusOK, response.OK("system deleted successfully", nil))
}

// InitSystemsStorage initializes some demo data for testing
func InitSystemsStorage() {
	// Create demo systems
	system1 := &models.System{
		ID:             "sys-001",
		Name:           "production-server-01",
		Type:           "linux",
		Status:         "online",
		IPAddress:      "192.168.1.10",
		Version:        "8.2.1",
		LastSeen:       time.Now(),
		Metadata:       map[string]string{"datacenter": "eu-west-1", "environment": "production"},
		OrganizationID: "org-distributor-001", // Example distributor organization
		CreatedAt:      time.Now().Add(-24 * time.Hour),
		UpdatedAt:      time.Now(),
		CreatedBy:      "admin",
	}

	system2 := &models.System{
		ID:             "sys-002",
		Name:           "test-server-01",
		Type:           "linux",
		Status:         "maintenance",
		IPAddress:      "192.168.1.11",
		Version:        "8.2.0",
		LastSeen:       time.Now().Add(-2 * time.Hour),
		Metadata:       map[string]string{"datacenter": "eu-west-1", "environment": "test"},
		OrganizationID: "org-customer-001", // Example customer organization
		CreatedAt:      time.Now().Add(-48 * time.Hour),
		UpdatedAt:      time.Now().Add(-1 * time.Hour),
		CreatedBy:      "admin",
	}

	systemsStorage["sys-001"] = system1
	systemsStorage["sys-002"] = system2

	// Create demo subscriptions
	subscription1 := &models.SystemSubscription{
		SystemID:   "sys-001",
		Plan:       "premium",
		Status:     "active",
		StartDate:  time.Now().Add(-30 * 24 * time.Hour),
		EndDate:    time.Now().Add(335 * 24 * time.Hour),
		Features:   []string{"monitoring", "backup", "support"},
		MaxUsers:   100,
		MaxStorage: 1024 * 1024 * 1024 * 1024, // 1TB
	}

	subscription2 := &models.SystemSubscription{
		SystemID:   "sys-002",
		Plan:       "basic",
		Status:     "active",
		StartDate:  time.Now().Add(-48 * 24 * time.Hour),
		EndDate:    time.Now().Add(317 * 24 * time.Hour),
		Features:   []string{"monitoring"},
		MaxUsers:   10,
		MaxStorage: 100 * 1024 * 1024 * 1024, // 100GB
	}

	subscriptionsStorage["sys-001"] = subscription1
	subscriptionsStorage["sys-002"] = subscription2

	logger.ComponentLogger("systems").Info().
		Str("operation", "init_storage").
		Msg("Demo systems storage initialized")
}

// RegenerateSystemSecret handles POST /api/systems/:id/regenerate-secret - regenerates system secret
func RegenerateSystemSecret(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	// Get current user context
	userID, userOrgRole, userRole := helpers.GetUserContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Create systems service
	systemsService := services.NewSystemsService()

	// Regenerate system secret
	systemSecret, err := systemsService.RegenerateSystemSecret(systemID, userID, userOrgRole, userRole)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Str("system_id", systemID).
			Msg("Failed to regenerate system secret")
		
		c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to regenerate system secret", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Log the action
	logger.LogBusinessOperation(c, "systems", "regenerate_secret", "system", systemID, true, nil)

	// Return new secret (only time it's visible)
	c.JSON(http.StatusOK, response.OK("system secret regenerated successfully", systemSecret))
}
