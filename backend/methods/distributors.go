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
var distributorsStorage = make(map[string]*models.Distributor)

// CreateDistributor handles POST /api/distributors - creates a new distributor
func CreateDistributor(c *gin.Context) {
	var request models.CreateDistributorRequest
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

	distributorID := uuid.New().String()

	distributor := &models.Distributor{
		ID:          distributorID,
		Name:        request.Name,
		Email:       request.Email,
		CompanyName: request.CompanyName,
		Status:      "active",
		Region:      request.Region,
		Territory:   request.Territory,
		Metadata:    request.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   userID.(string),
	}

	distributorsStorage[distributorID] = distributor

	logs.Logs.Printf("[INFO][DISTRIBUTORS] Distributor created: %s by user %s", distributor.Name, userID)

	c.JSON(http.StatusCreated, structs.Map(response.StatusOK{
		Code:    201,
		Message: "distributor created successfully",
		Data:    distributor,
	}))
}

// GetDistributors handles GET /api/distributors - retrieves all distributors
func GetDistributors(c *gin.Context) {
	distributors := make([]*models.Distributor, 0, len(distributorsStorage))
	for _, distributor := range distributorsStorage {
		distributors = append(distributors, distributor)
	}

	userID, _ := c.Get("user_id")
	logs.Logs.Printf("[INFO][DISTRIBUTORS] Distributors list requested by user %s", userID)

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "distributors retrieved successfully",
		Data:    gin.H{"distributors": distributors, "count": len(distributors)},
	}))
}

// UpdateDistributor handles PUT /api/distributors/:id - updates an existing distributor
func UpdateDistributor(c *gin.Context) {
	distributorID := c.Param("id")
	if distributorID == "" {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "distributor ID required",
			Data:    nil,
		}))
		return
	}

	distributor, exists := distributorsStorage[distributorID]
	if !exists {
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "distributor not found",
			Data:    nil,
		}))
		return
	}

	var request models.UpdateDistributorRequest
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
		distributor.Name = request.Name
	}
	if request.Email != "" {
		distributor.Email = request.Email
	}
	if request.CompanyName != "" {
		distributor.CompanyName = request.CompanyName
	}
	if request.Status != "" {
		distributor.Status = request.Status
	}
	if request.Region != "" {
		distributor.Region = request.Region
	}
	if request.Territory != nil {
		distributor.Territory = request.Territory
	}
	if request.Metadata != nil {
		distributor.Metadata = request.Metadata
	}

	distributor.UpdatedAt = time.Now()

	userID, _ := c.Get("user_id")
	logs.Logs.Printf("[INFO][DISTRIBUTORS] Distributor updated: %s by user %s", distributor.Name, userID)

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "distributor updated successfully",
		Data:    distributor,
	}))
}

// DeleteDistributor handles DELETE /api/distributors/:id - deletes a distributor
func DeleteDistributor(c *gin.Context) {
	distributorID := c.Param("id")
	if distributorID == "" {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "distributor ID required",
			Data:    nil,
		}))
		return
	}

	distributor, exists := distributorsStorage[distributorID]
	if !exists {
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "distributor not found",
			Data:    nil,
		}))
		return
	}

	delete(distributorsStorage, distributorID)

	userID, _ := c.Get("user_id")
	logs.Logs.Printf("[INFO][DISTRIBUTORS] Distributor deleted: %s by user %s", distributor.Name, userID)

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "distributor deleted successfully",
		Data:    nil,
	}))
}

// InitDistributorsStorage initializes demo data
func InitDistributorsStorage() {
	distributor1 := &models.Distributor{
		ID:          "dist-001",
		Name:        "Global Tech Distribution",
		Email:       "contact@globaltechdist.com",
		CompanyName: "Global Tech Distribution Ltd",
		Status:      "active",
		Region:      "EMEA",
		Territory:   []string{"Italy", "Germany", "France", "Spain"},
		Metadata:    map[string]string{"tier": "platinum", "contract_type": "exclusive"},
		CreatedAt:   time.Now().Add(-90 * 24 * time.Hour),
		UpdatedAt:   time.Now(),
		CreatedBy:   "god-user",
	}

	distributorsStorage["dist-001"] = distributor1
	logs.Logs.Println("[INFO][DISTRIBUTORS] Demo distributors storage initialized")
}
