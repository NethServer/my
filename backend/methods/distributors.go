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


// CreateDistributor handles POST /api/distributors - creates a new distributor organization in Logto
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
	userOrgID, _ := c.Get("organization_id")

	// Create organization in Logto
	client := services.NewLogtoManagementClient()
	
	// Prepare custom data with hierarchy info
	customData := map[string]interface{}{
		"type":        "distributor",
		"email":       request.Email,
		"companyName": request.CompanyName,
		"region":      request.Region,
		"territory":   request.Territory,
		"createdBy":   userOrgID,
		"createdAt":   time.Now().Format(time.RFC3339),
	}
	
	// Add request metadata to custom data
	if request.Metadata != nil {
		for k, v := range request.Metadata {
			customData[k] = v
		}
	}

	orgRequest := services.CreateOrganizationRequest{
		Name:        request.Name,
		Description: fmt.Sprintf("Distributor organization for %s (%s)", request.CompanyName, request.Region),
		CustomData:  customData,
		IsMfaRequired: false,
	}

	// Create the organization in Logto
	org, err := client.CreateOrganization(orgRequest)
	if err != nil {
		logs.Logs.Printf("[ERROR][DISTRIBUTORS] Failed to create distributor organization in Logto: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to create distributor organization",
			Data:    err.Error(),
		}))
		return
	}

	// Assign Distributor role as default JIT role
	distributorRole, err := client.GetOrganizationRoleByName("Distributor")
	if err != nil {
		logs.Logs.Printf("[ERROR][DISTRIBUTORS] Failed to find Distributor role: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to configure distributor role",
			Data:    err.Error(),
		}))
		return
	}

	if err := client.AssignOrganizationJitRoles(org.ID, []string{distributorRole.ID}); err != nil {
		logs.Logs.Printf("[ERROR][DISTRIBUTORS] Failed to assign Distributor JIT role: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to configure distributor permissions",
			Data:    err.Error(),
		}))
		return
	}

	logs.Logs.Printf("[INFO][DISTRIBUTORS] Distributor organization created in Logto: %s (ID: %s) by user %s", org.Name, org.ID, userID)

	// Return the created organization data
	distributorResponse := gin.H{
		"id":           org.ID,
		"name":         org.Name,
		"description":  org.Description,
		"customData":   org.CustomData,
		"isMfaRequired": org.IsMfaRequired,
		"type":         "distributor",
		"createdAt":    time.Now(),
	}

	c.JSON(http.StatusCreated, structs.Map(response.StatusOK{
		Code:    201,
		Message: "distributor created successfully",
		Data:    distributorResponse,
	}))
}

// GetDistributors handles GET /api/distributors - retrieves organizations with Distributor role from Logto
func GetDistributors(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userOrgRole, _ := c.Get("org_role")
	userOrgID, _ := c.Get("organization_id")
	
	logs.Logs.Printf("[INFO][DISTRIBUTORS] Distributors list requested by user %s (role: %s, org: %s)", userID, userOrgRole, userOrgID)

	// Get organizations with Distributor role from Logto
	orgs, err := services.GetOrganizationsByRole("Distributor")
	if err != nil {
		logs.Logs.Printf("[ERROR][DISTRIBUTORS] Failed to fetch distributors from Logto: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to fetch distributors",
			Data:    nil,
		}))
		return
	}

	// Apply visibility filtering
	filteredOrgs := services.FilterOrganizationsByVisibility(orgs, userOrgRole.(string), userOrgID.(string), "Distributor")

	// Convert Logto organizations to distributor format
	distributors := make([]gin.H, 0, len(filteredOrgs))
	for _, org := range filteredOrgs {
		distributor := gin.H{
			"id":           org.ID,
			"name":         org.Name,
			"description":  org.Description,
			"customData":   org.CustomData,
			"isMfaRequired": org.IsMfaRequired,
			"type":         "distributor",
		}
		
		// Add branding if available
		if org.Branding != nil {
			distributor["branding"] = gin.H{
				"logoUrl":     org.Branding.LogoUrl,
				"darkLogoUrl": org.Branding.DarkLogoUrl,
				"favicon":     org.Branding.Favicon,
				"darkFavicon": org.Branding.DarkFavicon,
			}
		}
		
		distributors = append(distributors, distributor)
	}

	logs.Logs.Printf("[INFO][DISTRIBUTORS] Retrieved %d distributors from Logto", len(distributors))

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "distributors retrieved successfully",
		Data:    gin.H{"distributors": distributors, "count": len(distributors)},
	}))
}

// UpdateDistributor handles PUT /api/distributors/:id - updates an existing distributor organization in Logto
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

	var request models.UpdateDistributorRequest
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
	currentOrg, err := client.GetOrganizationByID(distributorID)
	if err != nil {
		logs.Logs.Printf("[ERROR][DISTRIBUTORS] Failed to fetch distributor organization: %v", err)
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "distributor not found",
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
	
	// Update description based on company name if provided
	if request.CompanyName != "" {
		description := fmt.Sprintf("Distributor organization for %s (%s)", request.CompanyName, request.Region)
		if request.Region == "" && currentOrg.CustomData != nil {
			if region, ok := currentOrg.CustomData["region"].(string); ok {
				description = fmt.Sprintf("Distributor organization for %s (%s)", request.CompanyName, region)
			}
		}
		updateRequest.Description = &description
	}

	// Merge custom data with existing data
	if currentOrg.CustomData != nil {
		updateRequest.CustomData = make(map[string]interface{})
		// Copy existing custom data
		for k, v := range currentOrg.CustomData {
			updateRequest.CustomData[k] = v
		}
		
		// Update with new values
		if request.Email != "" {
			updateRequest.CustomData["email"] = request.Email
		}
		if request.CompanyName != "" {
			updateRequest.CustomData["companyName"] = request.CompanyName
		}
		if request.Region != "" {
			updateRequest.CustomData["region"] = request.Region
		}
		if request.Territory != nil {
			updateRequest.CustomData["territory"] = request.Territory
		}
		if request.Metadata != nil {
			for k, v := range request.Metadata {
				updateRequest.CustomData[k] = v
			}
		}
		
		// Update modification tracking
		updateRequest.CustomData["updatedBy"] = userOrgID
		updateRequest.CustomData["updatedAt"] = time.Now().Format(time.RFC3339)
	}

	// Update the organization in Logto
	updatedOrg, err := client.UpdateOrganization(distributorID, updateRequest)
	if err != nil {
		logs.Logs.Printf("[ERROR][DISTRIBUTORS] Failed to update distributor organization in Logto: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to update distributor organization",
			Data:    err.Error(),
		}))
		return
	}

	logs.Logs.Printf("[INFO][DISTRIBUTORS] Distributor organization updated in Logto: %s (ID: %s) by user %s", updatedOrg.Name, updatedOrg.ID, userID)

	// Return the updated organization data
	distributorResponse := gin.H{
		"id":           updatedOrg.ID,
		"name":         updatedOrg.Name,
		"description":  updatedOrg.Description,
		"customData":   updatedOrg.CustomData,
		"isMfaRequired": updatedOrg.IsMfaRequired,
		"type":         "distributor",
		"updatedAt":    time.Now(),
	}

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "distributor updated successfully",
		Data:    distributorResponse,
	}))
}

// DeleteDistributor handles DELETE /api/distributors/:id - deletes a distributor organization from Logto
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

	userID, _ := c.Get("user_id")

	// Connect to Logto Management API
	client := services.NewLogtoManagementClient()

	// First, verify the organization exists and get its data for logging
	currentOrg, err := client.GetOrganizationByID(distributorID)
	if err != nil {
		logs.Logs.Printf("[ERROR][DISTRIBUTORS] Failed to fetch distributor organization for deletion: %v", err)
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "distributor not found",
			Data:    nil,
		}))
		return
	}

	// Delete the organization from Logto
	if err := client.DeleteOrganization(distributorID); err != nil {
		logs.Logs.Printf("[ERROR][DISTRIBUTORS] Failed to delete distributor organization from Logto: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to delete distributor organization",
			Data:    err.Error(),
		}))
		return
	}

	logs.Logs.Printf("[INFO][DISTRIBUTORS] Distributor organization deleted from Logto: %s (ID: %s) by user %s", currentOrg.Name, distributorID, userID)

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "distributor deleted successfully",
		Data:    gin.H{
			"id":        distributorID,
			"name":      currentOrg.Name,
			"deletedAt": time.Now(),
		},
	}))
}

