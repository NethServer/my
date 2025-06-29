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

// CreateReseller handles POST /api/resellers - creates a new reseller organization in Logto
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
	userOrgID, _ := c.Get("organization_id")

	// Create organization in Logto
	client := services.NewLogtoManagementClient()

	// Prepare custom data with hierarchy info and system metadata
	customData := map[string]interface{}{
		"type":      "reseller",
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
		description = fmt.Sprintf("Reseller organization: %s", request.Name)
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
		logs.Logs.Printf("[ERROR][RESELLERS] Failed to create reseller organization in Logto: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to create reseller organization",
			Data:    err.Error(),
		}))
		return
	}

	// Assign Reseller role as default JIT role
	resellerRole, err := client.GetOrganizationRoleByName("Reseller")
	if err != nil {
		logs.Logs.Printf("[ERROR][RESELLERS] Failed to find Reseller role: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to configure reseller role",
			Data:    err.Error(),
		}))
		return
	}

	if err := client.AssignOrganizationJitRoles(org.ID, []string{resellerRole.ID}); err != nil {
		logs.Logs.Printf("[ERROR][RESELLERS] Failed to assign Reseller JIT role: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to configure reseller permissions",
			Data:    err.Error(),
		}))
		return
	}

	logs.Logs.Printf("[INFO][RESELLERS] Reseller organization created in Logto: %s (ID: %s) by user %s", org.Name, org.ID, userID)

	// Return the created organization data
	resellerResponse := gin.H{
		"id":            org.ID,
		"name":          org.Name,
		"description":   org.Description,
		"customData":    org.CustomData,
		"isMfaRequired": org.IsMfaRequired,
		"type":          "reseller",
		"createdAt":     time.Now(),
	}

	c.JSON(http.StatusCreated, structs.Map(response.StatusOK{
		Code:    201,
		Message: "reseller created successfully",
		Data:    resellerResponse,
	}))
}

// GetResellers handles GET /api/resellers - retrieves all resellers
func GetResellers(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userOrgRole, _ := c.Get("org_role")
	userOrgID, _ := c.Get("organization_id")

	logs.Logs.Printf("[INFO][RESELLERS] Resellers list requested by user %s (role: %s, org: %s)", userID, userOrgRole, userOrgID)

	// Get organizations with Reseller role from Logto
	orgs, err := services.GetOrganizationsByRole("Reseller")
	if err != nil {
		logs.Logs.Printf("[ERROR][RESELLERS] Failed to fetch resellers from Logto: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to fetch resellers",
			Data:    nil,
		}))
		return
	}

	// Apply visibility filtering
	filteredOrgs := services.FilterOrganizationsByVisibility(orgs, userOrgRole.(string), userOrgID.(string), "Reseller")

	// Convert Logto organizations to reseller format
	resellers := make([]gin.H, 0, len(filteredOrgs))
	for _, org := range filteredOrgs {
		reseller := gin.H{
			"id":            org.ID,
			"name":          org.Name,
			"description":   org.Description,
			"customData":    org.CustomData,
			"isMfaRequired": org.IsMfaRequired,
			"type":          "reseller",
		}

		// Add branding if available
		if org.Branding != nil {
			reseller["branding"] = gin.H{
				"logoUrl":     org.Branding.LogoUrl,
				"darkLogoUrl": org.Branding.DarkLogoUrl,
				"favicon":     org.Branding.Favicon,
				"darkFavicon": org.Branding.DarkFavicon,
			}
		}

		resellers = append(resellers, reseller)
	}

	logs.Logs.Printf("[INFO][RESELLERS] Retrieved %d resellers from Logto", len(resellers))

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "resellers retrieved successfully",
		Data:    gin.H{"resellers": resellers, "count": len(resellers)},
	}))
}

// UpdateReseller handles PUT /api/resellers/:id - updates an existing reseller organization in Logto
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

	var request models.UpdateResellerRequest
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
	currentOrg, err := client.GetOrganizationByID(resellerID)
	if err != nil {
		logs.Logs.Printf("[ERROR][RESELLERS] Failed to fetch reseller organization: %v", err)
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "reseller not found",
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
	updatedOrg, err := client.UpdateOrganization(resellerID, updateRequest)
	if err != nil {
		logs.Logs.Printf("[ERROR][RESELLERS] Failed to update reseller organization in Logto: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to update reseller organization",
			Data:    err.Error(),
		}))
		return
	}

	logs.Logs.Printf("[INFO][RESELLERS] Reseller organization updated in Logto: %s (ID: %s) by user %s", updatedOrg.Name, updatedOrg.ID, userID)

	// Return the updated organization data
	resellerResponse := gin.H{
		"id":            updatedOrg.ID,
		"name":          updatedOrg.Name,
		"description":   updatedOrg.Description,
		"customData":    updatedOrg.CustomData,
		"isMfaRequired": updatedOrg.IsMfaRequired,
		"type":          "reseller",
		"updatedAt":     time.Now(),
	}

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "reseller updated successfully",
		Data:    resellerResponse,
	}))
}

// DeleteReseller handles DELETE /api/resellers/:id - deletes a reseller organization from Logto
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

	userID, _ := c.Get("user_id")

	// Connect to Logto Management API
	client := services.NewLogtoManagementClient()

	// First, verify the organization exists and get its data for logging
	currentOrg, err := client.GetOrganizationByID(resellerID)
	if err != nil {
		logs.Logs.Printf("[ERROR][RESELLERS] Failed to fetch reseller organization for deletion: %v", err)
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "reseller not found",
			Data:    nil,
		}))
		return
	}

	// Delete the organization from Logto
	if err := client.DeleteOrganization(resellerID); err != nil {
		logs.Logs.Printf("[ERROR][RESELLERS] Failed to delete reseller organization from Logto: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to delete reseller organization",
			Data:    err.Error(),
		}))
		return
	}

	logs.Logs.Printf("[INFO][RESELLERS] Reseller organization deleted from Logto: %s (ID: %s) by user %s", currentOrg.Name, resellerID, userID)

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "reseller deleted successfully",
		Data: gin.H{
			"id":        resellerID,
			"name":      currentOrg.Name,
			"deletedAt": time.Now(),
		},
	}))
}
