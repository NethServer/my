/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services"
)

// CreateCustomer handles POST /api/customers - creates a new customer organization in Logto
func CreateCustomer(c *gin.Context) {
	var request models.CreateCustomerRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("request fields malformed", err.Error()))
		return
	}

	_, _ = c.Get("user_id") // user context verified by middleware
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
		logger.NewHTTPErrorLogger(c, "customers").LogError(err, "create_customer_organization", http.StatusInternalServerError, "Failed to create customer organization in Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create customer organization", err.Error()))
		return
	}

	// Assign Customer role as default JIT role
	customerRole, err := client.GetOrganizationRoleByName("Customer")
	if err != nil {
		logger.NewHTTPErrorLogger(c, "customers").LogError(err, "find_customer_role", http.StatusInternalServerError, "Failed to find Customer role")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to configure customer role", err.Error()))
		return
	}

	if err := client.AssignOrganizationJitRoles(org.ID, []string{customerRole.ID}); err != nil {
		logger.NewHTTPErrorLogger(c, "customers").LogError(err, "assign_customer_jit_role", http.StatusInternalServerError, "Failed to assign Customer JIT role")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to configure customer permissions", err.Error()))
		return
	}

	logger.LogBusinessOperation(c, "customers", "create", "customer", org.ID, true, nil)

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

	c.JSON(http.StatusCreated, response.Created("customer created successfully", customerResponse))
}

// GetCustomers handles GET /api/customers - retrieves all customers
func GetCustomers(c *gin.Context) {
	_, _ = c.Get("user_id")
	userOrgRole, _ := c.Get("org_role")
	userOrgID, _ := c.Get("organization_id")

	logger.RequestLogger(c, "customers").Info().
		Str("operation", "list_customers").
		Msg("Customers list requested")

	// Get organizations with Customer role from Logto
	orgs, err := services.GetOrganizationsByRole("Customer")
	if err != nil {
		logger.NewHTTPErrorLogger(c, "customers").LogError(err, "fetch_customers", http.StatusInternalServerError, "Failed to fetch customers from Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch customers", nil))
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

	logger.RequestLogger(c, "customers").Info().
		Int("customer_count", len(customers)).
		Str("operation", "fetch_customers_result").
		Msg("Retrieved customers from Logto")

	c.JSON(http.StatusOK, response.OK("customers retrieved successfully", gin.H{"customers": customers, "count": len(customers)}))
}

// UpdateCustomer handles PUT /api/customers/:id - updates an existing customer organization in Logto
func UpdateCustomer(c *gin.Context) {
	customerID := c.Param("id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("customer ID required", nil))
		return
	}

	var request models.UpdateCustomerRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("request fields malformed", err.Error()))
		return
	}

	_, _ = c.Get("user_id")
	userOrgID, _ := c.Get("organization_id")

	// Connect to Logto Management API
	client := services.NewLogtoManagementClient()

	// First, verify the organization exists and get current data
	currentOrg, err := client.GetOrganizationByID(customerID)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "customers").LogError(err, "fetch_customer_organization", http.StatusInternalServerError, "Failed to fetch customer organization")
		c.JSON(http.StatusNotFound, response.NotFound("customer not found", nil))
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
		logger.NewHTTPErrorLogger(c, "customers").LogError(err, "update_customer_organization", http.StatusInternalServerError, "Failed to update customer organization in Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to update customer organization", err.Error()))
		return
	}

	logger.LogBusinessOperation(c, "customers", "update", "customer", customerID, true, nil)

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

	c.JSON(http.StatusOK, response.OK("customer updated successfully", customerResponse))
}

// DeleteCustomer handles DELETE /api/customers/:id - deletes a customer organization from Logto
func DeleteCustomer(c *gin.Context) {
	customerID := c.Param("id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("customer ID required", nil))
		return
	}

	_, _ = c.Get("user_id")

	// Connect to Logto Management API
	client := services.NewLogtoManagementClient()

	// First, verify the organization exists and get its data for logging
	currentOrg, err := client.GetOrganizationByID(customerID)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "customers").LogError(err, "fetch_customer_for_deletion", http.StatusInternalServerError, "Failed to fetch customer organization for deletion")
		c.JSON(http.StatusNotFound, response.NotFound("customer not found", nil))
		return
	}

	// Delete the organization from Logto
	if err := client.DeleteOrganization(customerID); err != nil {
		logger.NewHTTPErrorLogger(c, "customers").LogError(err, "delete_customer_organization", http.StatusInternalServerError, "Failed to delete customer organization from Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to delete customer organization", err.Error()))
		return
	}

	logger.LogBusinessOperation(c, "customers", "delete", "customer", customerID, true, nil)

	c.JSON(http.StatusOK, response.OK("customer deleted successfully", gin.H{
		"id":        customerID,
		"name":      currentOrg.Name,
		"deletedAt": time.Now(),
	}))
}
