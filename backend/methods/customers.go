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
var customersStorage = make(map[string]*models.Customer)

// CreateCustomer handles POST /api/customers - creates a new customer
func CreateCustomer(c *gin.Context) {
	var request models.CreateCustomerRequest
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

	customerID := uuid.New().String()

	customer := &models.Customer{
		ID:          customerID,
		Name:        request.Name,
		Email:       request.Email,
		CompanyName: request.CompanyName,
		Status:      "active",
		Tier:        request.Tier,
		ResellerID:  request.ResellerID,
		Metadata:    request.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   userID.(string),
	}

	customersStorage[customerID] = customer

	logs.Logs.Printf("[INFO][CUSTOMERS] Customer created: %s by user %s", customer.Name, userID)

	c.JSON(http.StatusCreated, structs.Map(response.StatusOK{
		Code:    201,
		Message: "customer created successfully",
		Data:    customer,
	}))
}

// GetCustomers handles GET /api/customers - retrieves all customers
func GetCustomers(c *gin.Context) {
	customers := make([]*models.Customer, 0, len(customersStorage))
	for _, customer := range customersStorage {
		customers = append(customers, customer)
	}

	userID, _ := c.Get("user_id")
	logs.Logs.Printf("[INFO][CUSTOMERS] Customers list requested by user %s", userID)

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "customers retrieved successfully",
		Data:    gin.H{"customers": customers, "count": len(customers)},
	}))
}

// UpdateCustomer handles PUT /api/customers/:id - updates an existing customer
func UpdateCustomer(c *gin.Context) {
	customerID := c.Param("id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "customer ID required",
			Data:    nil,
		}))
		return
	}

	customer, exists := customersStorage[customerID]
	if !exists {
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "customer not found",
			Data:    nil,
		}))
		return
	}

	var request models.UpdateCustomerRequest
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
		customer.Name = request.Name
	}
	if request.Email != "" {
		customer.Email = request.Email
	}
	if request.CompanyName != "" {
		customer.CompanyName = request.CompanyName
	}
	if request.Status != "" {
		customer.Status = request.Status
	}
	if request.Tier != "" {
		customer.Tier = request.Tier
	}
	if request.ResellerID != "" {
		customer.ResellerID = request.ResellerID
	}
	if request.Metadata != nil {
		customer.Metadata = request.Metadata
	}

	customer.UpdatedAt = time.Now()

	userID, _ := c.Get("user_id")
	logs.Logs.Printf("[INFO][CUSTOMERS] Customer updated: %s by user %s", customer.Name, userID)

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "customer updated successfully",
		Data:    customer,
	}))
}

// DeleteCustomer handles DELETE /api/customers/:id - deletes a customer
func DeleteCustomer(c *gin.Context) {
	customerID := c.Param("id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "customer ID required",
			Data:    nil,
		}))
		return
	}

	customer, exists := customersStorage[customerID]
	if !exists {
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "customer not found",
			Data:    nil,
		}))
		return
	}

	delete(customersStorage, customerID)

	userID, _ := c.Get("user_id")
	logs.Logs.Printf("[INFO][CUSTOMERS] Customer deleted: %s by user %s", customer.Name, userID)

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "customer deleted successfully",
		Data:    nil,
	}))
}

// InitCustomersStorage initializes demo data
func InitCustomersStorage() {
	customer1 := &models.Customer{
		ID:          "cust-001",
		Name:        "Acme Corporation",
		Email:       "admin@acmecorp.com",
		CompanyName: "Acme Corporation Ltd",
		Status:      "active",
		Tier:        "premium",
		ResellerID:  "res-001",
		Metadata:    map[string]string{"industry": "manufacturing", "employees": "500"},
		CreatedAt:   time.Now().Add(-15 * 24 * time.Hour),
		UpdatedAt:   time.Now(),
		CreatedBy:   "res-001",
	}

	customer2 := &models.Customer{
		ID:          "cust-002",
		Name:        "StartupTech Inc",
		Email:       "contact@startuptech.io",
		CompanyName: "StartupTech Inc",
		Status:      "active",
		Tier:        "basic",
		ResellerID:  "res-002",
		Metadata:    map[string]string{"industry": "technology", "employees": "25"},
		CreatedAt:   time.Now().Add(-7 * 24 * time.Hour),
		UpdatedAt:   time.Now(),
		CreatedBy:   "res-002",
	}

	customersStorage["cust-001"] = customer1
	customersStorage["cust-002"] = customer2
	logs.Logs.Println("[INFO][CUSTOMERS] Demo customers storage initialized")
}
