/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: GPL-2.0-only
*/

package methods

import (
	"fmt"
	"net/http"
	"time"

	"github.com/fatih/structs"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/nethesis/my/backend/logs"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services"
)

// CreateCustomer handles POST /api/customers - creates a new customer organization in Logto
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
	userOrgID, _ := c.Get("organization_id")

	// Create organization in Logto
	client := services.NewLogtoManagementClient()

	// Prepare custom data with hierarchy info and system metadata
	customData := map[string]interface{}{
		"type":      "customer",
		"createdBy": userOrgID,
		"createdAt": time.Now().Format(time.RFC3339),
	}

	// Add user's custom data to system custom data
	if request.CustomData != nil {
		for k, v := range request.CustomData {
			customData[k] = v
		}
	}

	// Use description from request or generate default
	description := request.Description
	if description == "" {
		description = fmt.Sprintf("Customer organization: %s", request.Name)
	}

	orgRequest := services.CreateOrganizationRequest{
		Name:          request.Name,
		Description:   description,
		CustomData:    customData,
		IsMfaRequired: request.IsMfaRequired,
	}

	// Create the organization in Logto
	org, err := client.CreateOrganization(orgRequest)
	if err != nil {
		logs.Logs.Printf("[ERROR][CUSTOMERS] Failed to create customer organization in Logto: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to create customer organization",
			Data:    err.Error(),
		}))
		return
	}

	// Assign Customer role as default JIT role
	customerRole, err := client.GetOrganizationRoleByName("Customer")
	if err != nil {
		logs.Logs.Printf("[ERROR][CUSTOMERS] Failed to find Customer role: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to configure customer role",
			Data:    err.Error(),
		}))
		return
	}

	if err := client.AssignOrganizationJitRoles(org.ID, []string{customerRole.ID}); err != nil {
		logs.Logs.Printf("[ERROR][CUSTOMERS] Failed to assign Customer JIT role: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to configure customer permissions",
			Data:    err.Error(),
		}))
		return
	}

	logs.Logs.Printf("[INFO][CUSTOMERS] Customer organization created in Logto: %s (ID: %s) by user %s", org.Name, org.ID, userID)

	// Return the created organization data
	customerResponse := gin.H{
		"id":            org.ID,
		"name":          org.Name,
		"description":   org.Description,
		"customData":    org.CustomData,
		"isMfaRequired": org.IsMfaRequired,
		"type":          "customer",
		"createdAt":     time.Now(),
	}

	c.JSON(http.StatusCreated, structs.Map(response.StatusOK{
		Code:    201,
		Message: "customer created successfully",
		Data:    customerResponse,
	}))
}

// GetCustomers handles GET /api/customers - retrieves all customers
func GetCustomers(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userOrgRole, _ := c.Get("org_role")
	userOrgID, _ := c.Get("organization_id")

	logs.Logs.Printf("[INFO][CUSTOMERS] Customers list requested by user %s (role: %s, org: %s)", userID, userOrgRole, userOrgID)

	// Get organizations with Customer role from Logto
	orgs, err := services.GetOrganizationsByRole("Customer")
	if err != nil {
		logs.Logs.Printf("[ERROR][CUSTOMERS] Failed to fetch customers from Logto: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to fetch customers",
			Data:    nil,
		}))
		return
	}

	// Apply visibility filtering
	filteredOrgs := services.FilterOrganizationsByVisibility(orgs, userOrgRole.(string), userOrgID.(string), "Customer")

	// Convert Logto organizations to customer format
	customers := make([]gin.H, 0, len(filteredOrgs))
	for _, org := range filteredOrgs {
		customer := gin.H{
			"id":            org.ID,
			"name":          org.Name,
			"description":   org.Description,
			"customData":    org.CustomData,
			"isMfaRequired": org.IsMfaRequired,
			"type":          "customer",
		}

		// Add branding if available
		if org.Branding != nil {
			customer["branding"] = gin.H{
				"logoUrl":     org.Branding.LogoUrl,
				"darkLogoUrl": org.Branding.DarkLogoUrl,
				"favicon":     org.Branding.Favicon,
				"darkFavicon": org.Branding.DarkFavicon,
			}
		}

		customers = append(customers, customer)
	}

	logs.Logs.Printf("[INFO][CUSTOMERS] Retrieved %d customers from Logto", len(customers))

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "customers retrieved successfully",
		Data:    gin.H{"customers": customers, "count": len(customers)},
	}))
}

// UpdateCustomer handles PUT /api/customers/:id - updates an existing customer organization in Logto
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

	var request models.UpdateCustomerRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "request fields malformed",
			Data:    err.Error(),
		}))
		return
	}

	userID, _ := c.Get("user_id")
	userOrgID, _ := c.Get("organization_id")

	// Connect to Logto Management API
	client := services.NewLogtoManagementClient()

	// First, verify the organization exists and get current data
	currentOrg, err := client.GetOrganizationByID(customerID)
	if err != nil {
		logs.Logs.Printf("[ERROR][CUSTOMERS] Failed to fetch customer organization: %v", err)
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "customer not found",
			Data:    nil,
		}))
		return
	}

	// Prepare update request with only changed fields
	updateRequest := services.UpdateOrganizationRequest{}

	// Update name if provided
	if request.Name != "" {
		updateRequest.Name = &request.Name
	}

	// Update description if provided
	if request.Description != "" {
		updateRequest.Description = &request.Description
	}

	// Update MFA requirement if provided
	if request.IsMfaRequired != nil {
		updateRequest.IsMfaRequired = request.IsMfaRequired
	}

	// Merge custom data with existing data
	if currentOrg.CustomData != nil || request.CustomData != nil {
		updateRequest.CustomData = make(map[string]interface{})

		// Copy existing custom data
		if currentOrg.CustomData != nil {
			for k, v := range currentOrg.CustomData {
				updateRequest.CustomData[k] = v
			}
		}

		// Update with new custom data values
		if request.CustomData != nil {
			for k, v := range request.CustomData {
				updateRequest.CustomData[k] = v
			}
		}

		// Update modification tracking
		updateRequest.CustomData["updatedBy"] = userOrgID
		updateRequest.CustomData["updatedAt"] = time.Now().Format(time.RFC3339)
	}

	// Update the organization in Logto
	updatedOrg, err := client.UpdateOrganization(customerID, updateRequest)
	if err != nil {
		logs.Logs.Printf("[ERROR][CUSTOMERS] Failed to update customer organization in Logto: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to update customer organization",
			Data:    err.Error(),
		}))
		return
	}

	logs.Logs.Printf("[INFO][CUSTOMERS] Customer organization updated in Logto: %s (ID: %s) by user %s", updatedOrg.Name, updatedOrg.ID, userID)

	// Return the updated organization data
	customerResponse := gin.H{
		"id":            updatedOrg.ID,
		"name":          updatedOrg.Name,
		"description":   updatedOrg.Description,
		"customData":    updatedOrg.CustomData,
		"isMfaRequired": updatedOrg.IsMfaRequired,
		"type":          "customer",
		"updatedAt":     time.Now(),
	}

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "customer updated successfully",
		Data:    customerResponse,
	}))
}

// DeleteCustomer handles DELETE /api/customers/:id - deletes a customer organization from Logto
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

	userID, _ := c.Get("user_id")

	// Connect to Logto Management API
	client := services.NewLogtoManagementClient()

	// First, verify the organization exists and get its data for logging
	currentOrg, err := client.GetOrganizationByID(customerID)
	if err != nil {
		logs.Logs.Printf("[ERROR][CUSTOMERS] Failed to fetch customer organization for deletion: %v", err)
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "customer not found",
			Data:    nil,
		}))
		return
	}

	// Delete the organization from Logto
	if err := client.DeleteOrganization(customerID); err != nil {
		logs.Logs.Printf("[ERROR][CUSTOMERS] Failed to delete customer organization from Logto: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to delete customer organization",
			Data:    err.Error(),
		}))
		return
	}

	logs.Logs.Printf("[INFO][CUSTOMERS] Customer organization deleted from Logto: %s (ID: %s) by user %s", currentOrg.Name, customerID, userID)

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "customer deleted successfully",
		Data: gin.H{
			"id":        customerID,
			"name":      currentOrg.Name,
			"deletedAt": time.Now(),
		},
	}))
}
