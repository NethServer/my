/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: GPL-2.0-only
*/

package methods

import (
	"net/http"
	"time"

	"github.com/fatih/structs"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"

	"github.com/nethesis/my/backend/logs"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
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
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "request fields malformed",
			Data:    err.Error(),
		}))
		return
	}

	// Get current user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		userID = "unknown"
	}

	// Generate unique ID for the system
	systemID := uuid.New().String()

	// Create new system
	system := &models.System{
		ID:        systemID,
		Name:      request.Name,
		Type:      request.Type,
		Status:    "offline", // Default status
		IPAddress: request.IPAddress,
		Version:   request.Version,
		LastSeen:  time.Now(),
		Metadata:  request.Metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: userID.(string),
	}

	// Store system (in production, save to database)
	systemsStorage[systemID] = system

	// Log the action
	logs.Logs.Printf("[INFO][SYSTEMS] System created: %s by user %s", system.Name, userID)

	// Return success response
	c.JSON(http.StatusCreated, structs.Map(response.StatusOK{
		Code:    201,
		Message: "system created successfully",
		Data:    system,
	}))
}

// GetSystems handles GET /api/systems - retrieves all systems
func GetSystems(c *gin.Context) {
	// Convert map to slice for response
	systems := make([]*models.System, 0, len(systemsStorage))
	for _, system := range systemsStorage {
		systems = append(systems, system)
	}

	// Log the action
	userID, _ := c.Get("user_id")
	logs.Logs.Printf("[INFO][SYSTEMS] Systems list requested by user %s", userID)

	// Return systems list
	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "systems retrieved successfully",
		Data:    gin.H{"systems": systems, "count": len(systems)},
	}))
}

// UpdateSystem handles PUT /api/systems/:id - updates an existing system
func UpdateSystem(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "system ID required",
			Data:    nil,
		}))
		return
	}

	// Check if system exists
	system, exists := systemsStorage[systemID]
	if !exists {
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "system not found",
			Data:    nil,
		}))
		return
	}

	// Parse request body
	var request models.UpdateSystemRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "request fields malformed",
			Data:    err.Error(),
		}))
		return
	}

	// Update system fields (only if provided in request)
	if request.Name != "" {
		system.Name = request.Name
	}
	if request.Type != "" {
		system.Type = request.Type
	}
	if request.Status != "" {
		system.Status = request.Status
	}
	if request.IPAddress != "" {
		system.IPAddress = request.IPAddress
	}
	if request.Version != "" {
		system.Version = request.Version
	}
	if request.Metadata != nil {
		system.Metadata = request.Metadata
	}

	// Update timestamp
	system.UpdatedAt = time.Now()

	// Log the action
	userID, _ := c.Get("user_id")
	logs.Logs.Printf("[INFO][SYSTEMS] System updated: %s by user %s", system.Name, userID)

	// Return updated system
	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "system updated successfully",
		Data:    system,
	}))
}

// DeleteSystem handles DELETE /api/systems/:id - deletes a system
func DeleteSystem(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "system ID required",
			Data:    nil,
		}))
		return
	}

	// Check if system exists
	system, exists := systemsStorage[systemID]
	if !exists {
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "system not found",
			Data:    nil,
		}))
		return
	}

	// Delete system from storage
	delete(systemsStorage, systemID)

	// Also delete related subscription if exists
	delete(subscriptionsStorage, systemID)

	// Log the action
	userID, _ := c.Get("user_id")
	logs.Logs.Printf("[INFO][SYSTEMS] System deleted: %s by user %s", system.Name, userID)

	// Return success response
	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "system deleted successfully",
		Data:    nil,
	}))
}

// GetSystemSubscriptions handles GET /api/systems/subscriptions - retrieves subscription info for all systems
func GetSystemSubscriptions(c *gin.Context) {
	// Convert map to slice for response
	subscriptions := make([]*models.SystemSubscription, 0, len(subscriptionsStorage))
	for _, subscription := range subscriptionsStorage {
		subscriptions = append(subscriptions, subscription)
	}

	// Log the action
	userID, _ := c.Get("user_id")
	logs.Logs.Printf("[INFO][SYSTEMS] Subscriptions list requested by user %s", userID)

	// Return subscriptions list
	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "system subscriptions retrieved successfully",
		Data:    gin.H{"subscriptions": subscriptions, "count": len(subscriptions)},
	}))
}

// RestartSystem handles POST /api/systems/:id/restart - restarts a system
func RestartSystem(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "system ID required",
			Data:    nil,
		}))
		return
	}

	// Check if system exists
	system, exists := systemsStorage[systemID]
	if !exists {
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "system not found",
			Data:    nil,
		}))
		return
	}

	// Parse optional request body
	var request models.SystemActionRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		// Ignore binding errors for optional payload
		request = models.SystemActionRequest{}
	}

	// Simulate restart action (in production, this would trigger actual restart)
	system.Status = "restarting"
	system.UpdatedAt = time.Now()

	// Log the action
	userID, _ := c.Get("user_id")
	logs.Logs.Printf("[INFO][SYSTEMS] System restart initiated: %s by user %s (force: %v)", system.Name, userID, request.Force)

	// Return success response
	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "system restart initiated",
		Data: gin.H{
			"system_id": systemID,
			"status":    "restarting",
			"force":     request.Force,
		},
	}))
}

// EnableSystem handles PUT /api/systems/:id/enable - enables a system
func EnableSystem(c *gin.Context) {
	// Get system ID from URL parameter
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "system ID required",
			Data:    nil,
		}))
		return
	}

	// Check if system exists
	system, exists := systemsStorage[systemID]
	if !exists {
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "system not found",
			Data:    nil,
		}))
		return
	}

	// Enable system
	system.Status = "online"
	system.UpdatedAt = time.Now()
	system.LastSeen = time.Now()

	// Log the action
	userID, _ := c.Get("user_id")
	logs.Logs.Printf("[INFO][SYSTEMS] System enabled: %s by user %s", system.Name, userID)

	// Return success response
	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "system enabled successfully",
		Data:    system,
	}))
}

// InitSystemsStorage initializes some demo data for testing
func InitSystemsStorage() {
	// Create demo systems
	system1 := &models.System{
		ID:        "sys-001",
		Name:      "production-server-01",
		Type:      "linux",
		Status:    "online",
		IPAddress: "192.168.1.10",
		Version:   "8.2.1",
		LastSeen:  time.Now(),
		Metadata:  map[string]string{"datacenter": "eu-west-1", "environment": "production"},
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
		CreatedBy: "admin",
	}

	system2 := &models.System{
		ID:        "sys-002",
		Name:      "test-server-01",
		Type:      "linux",
		Status:    "maintenance",
		IPAddress: "192.168.1.11",
		Version:   "8.2.0",
		LastSeen:  time.Now().Add(-2 * time.Hour),
		Metadata:  map[string]string{"datacenter": "eu-west-1", "environment": "test"},
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
		CreatedBy: "admin",
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

	logs.Logs.Println("[INFO][SYSTEMS] Demo systems storage initialized")
}
