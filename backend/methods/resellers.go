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
var resellersStorage = make(map[string]*models.Reseller)

// CreateReseller handles POST /api/resellers - creates a new reseller
func CreateReseller(c *gin.Context) {
	var request models.CreateResellerRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "request fields malformed",
			Data:    err.Error(),
		}))
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		userID = "unknown"
	}

	resellerID := uuid.New().String()

	reseller := &models.Reseller{
		ID:          resellerID,
		Name:        request.Name,
		Email:       request.Email,
		CompanyName: request.CompanyName,
		Status:      "active",
		Region:      request.Region,
		Metadata:    request.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   userID.(string),
	}

	resellersStorage[resellerID] = reseller

	logs.Logs.Printf("[INFO][RESELLERS] Reseller created: %s by user %s", reseller.Name, userID)

	c.JSON(http.StatusCreated, structs.Map(response.StatusOK{
		Code:    201,
		Message: "reseller created successfully",
		Data:    reseller,
	}))
}

// GetResellers handles GET /api/resellers - retrieves all resellers
func GetResellers(c *gin.Context) {
	resellers := make([]*models.Reseller, 0, len(resellersStorage))
	for _, reseller := range resellersStorage {
		resellers = append(resellers, reseller)
	}

	userID, _ := c.Get("user_id")
	logs.Logs.Printf("[INFO][RESELLERS] Resellers list requested by user %s", userID)

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "resellers retrieved successfully",
		Data:    gin.H{"resellers": resellers, "count": len(resellers)},
	}))
}

// UpdateReseller handles PUT /api/resellers/:id - updates an existing reseller
func UpdateReseller(c *gin.Context) {
	resellerID := c.Param("id")
	if resellerID == "" {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "reseller ID required",
			Data:    nil,
		}))
		return
	}

	reseller, exists := resellersStorage[resellerID]
	if !exists {
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "reseller not found",
			Data:    nil,
		}))
		return
	}

	var request models.UpdateResellerRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "request fields malformed",
			Data:    err.Error(),
		}))
		return
	}

	// Update fields if provided
	if request.Name != "" {
		reseller.Name = request.Name
	}
	if request.Email != "" {
		reseller.Email = request.Email
	}
	if request.CompanyName != "" {
		reseller.CompanyName = request.CompanyName
	}
	if request.Status != "" {
		reseller.Status = request.Status
	}
	if request.Region != "" {
		reseller.Region = request.Region
	}
	if request.Metadata != nil {
		reseller.Metadata = request.Metadata
	}

	reseller.UpdatedAt = time.Now()

	userID, _ := c.Get("user_id")
	logs.Logs.Printf("[INFO][RESELLERS] Reseller updated: %s by user %s", reseller.Name, userID)

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "reseller updated successfully",
		Data:    reseller,
	}))
}

// DeleteReseller handles DELETE /api/resellers/:id - deletes a reseller
func DeleteReseller(c *gin.Context) {
	resellerID := c.Param("id")
	if resellerID == "" {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "reseller ID required",
			Data:    nil,
		}))
		return
	}

	reseller, exists := resellersStorage[resellerID]
	if !exists {
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "reseller not found",
			Data:    nil,
		}))
		return
	}

	delete(resellersStorage, resellerID)

	userID, _ := c.Get("user_id")
	logs.Logs.Printf("[INFO][RESELLERS] Reseller deleted: %s by user %s", reseller.Name, userID)

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "reseller deleted successfully",
		Data:    nil,
	}))
}

// InitResellersStorage initializes demo data
func InitResellersStorage() {
	reseller1 := &models.Reseller{
		ID:          "res-001",
		Name:        "Mario Rossi",
		Email:       "mario.rossi@techsolutions.it",
		CompanyName: "Tech Solutions SRL",
		Status:      "active",
		Region:      "EU-South",
		Metadata:    map[string]string{"country": "Italy", "tier": "premium"},
		CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:   time.Now(),
		CreatedBy:   "distributor-001",
	}

	reseller2 := &models.Reseller{
		ID:          "res-002",
		Name:        "Laura Bianchi",
		Email:       "l.bianchi@netpro.de",
		CompanyName: "NetPro GmbH",
		Status:      "active",
		Region:      "EU-Central",
		Metadata:    map[string]string{"country": "Germany", "tier": "standard"},
		CreatedAt:   time.Now().Add(-15 * 24 * time.Hour),
		UpdatedAt:   time.Now().Add(-2 * 24 * time.Hour),
		CreatedBy:   "distributor-001",
	}

	resellersStorage["res-001"] = reseller1
	resellersStorage["res-002"] = reseller2

	logs.Logs.Println("[INFO][RESELLERS] Demo resellers storage initialized")
}
