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

// CreateDistributor handles POST /api/distributors - creates a new distributor organization in Logto
func CreateDistributor(c *gin.Context) {
	var request models.CreateDistributorRequest
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
		"type":      "distributor",
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
		description = fmt.Sprintf("Distributor organization: %s", request.Name)
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
		logger.NewHTTPErrorLogger(c, "distributors").LogError(err, "create_distributor_organization", http.StatusInternalServerError, "Failed to create distributor organization in Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create distributor organization", err.Error()))
		return
	}

	// Assign Distributor role as default JIT role
	distributorRole, err := client.GetOrganizationRoleByName("Distributor")
	if err != nil {
		logger.NewHTTPErrorLogger(c, "distributors").LogError(err, "find_distributor_role", http.StatusInternalServerError, "Failed to find Distributor role")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to configure distributor role", err.Error()))
		return
	}

	if err := client.AssignOrganizationJitRoles(org.ID, []string{distributorRole.ID}); err != nil {
		logger.NewHTTPErrorLogger(c, "distributors").LogError(err, "assign_distributor_jit_role", http.StatusInternalServerError, "Failed to assign Distributor JIT role")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to configure distributor permissions", err.Error()))
		return
	}

	logger.LogBusinessOperation(c, "distributors", "create", "distributor", org.ID, true, nil)

	// Return the created organization data
	distributorResponse := gin.H{
		"id":            org.ID,
		"name":          org.Name,
		"description":   org.Description,
		"customData":    org.CustomData,
		"isMfaRequired": org.IsMfaRequired,
		"type":          "distributor",
		"createdAt":     time.Now(),
	}

	c.JSON(http.StatusCreated, response.Created("distributor created successfully", distributorResponse))
}

// GetDistributors handles GET /api/distributors - retrieves organizations with Distributor role from Logto
func GetDistributors(c *gin.Context) {
	_, _ = c.Get("user_id")
	userOrgRole, _ := c.Get("org_role")
	userOrgID, _ := c.Get("organization_id")

	logger.RequestLogger(c, "distributors").Info().
		Str("operation", "list_distributors").
		Msg("Distributors list requested")

	// Get organizations with Distributor role from Logto
	orgs, err := services.GetOrganizationsByRole("Distributor")
	if err != nil {
		logger.NewHTTPErrorLogger(c, "distributors").LogError(err, "fetch_distributors", http.StatusInternalServerError, "Failed to fetch distributors from Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch distributors", nil))
		return
	}

	// Apply visibility filtering
	filteredOrgs := services.FilterOrganizationsByVisibility(orgs, userOrgRole.(string), userOrgID.(string), "Distributor")

	// Convert Logto organizations to distributor format
	distributors := make([]gin.H, 0, len(filteredOrgs))
	for _, org := range filteredOrgs {
		distributor := gin.H{
			"id":            org.ID,
			"name":          org.Name,
			"description":   org.Description,
			"customData":    org.CustomData,
			"isMfaRequired": org.IsMfaRequired,
			"type":          "distributor",
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

	logger.RequestLogger(c, "distributors").Info().
		Int("distributor_count", len(distributors)).
		Str("operation", "fetch_distributors_result").
		Msg("Retrieved distributors from Logto")

	c.JSON(http.StatusOK, response.OK("distributors retrieved successfully", gin.H{"distributors": distributors, "count": len(distributors)}))
}

// UpdateDistributor handles PUT /api/distributors/:id - updates an existing distributor organization in Logto
func UpdateDistributor(c *gin.Context) {
	distributorID := c.Param("id")
	if distributorID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("distributor ID required", nil))
		return
	}

	var request models.UpdateDistributorRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("request fields malformed", err.Error()))
		return
	}

	_, _ = c.Get("user_id")
	userOrgID, _ := c.Get("organization_id")

	// Connect to Logto Management API
	client := services.NewLogtoManagementClient()

	// First, verify the organization exists and get current data
	currentOrg, err := client.GetOrganizationByID(distributorID)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "distributors").LogError(err, "fetch_distributor_organization", http.StatusInternalServerError, "Failed to fetch distributor organization")
		c.JSON(http.StatusNotFound, response.NotFound("distributor not found", nil))
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
	updatedOrg, err := client.UpdateOrganization(distributorID, updateRequest)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "distributors").LogError(err, "update_distributor_organization", http.StatusInternalServerError, "Failed to update distributor organization in Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to update distributor organization", err.Error()))
		return
	}

	logger.LogBusinessOperation(c, "distributors", "update", "distributor", distributorID, true, nil)

	// Return the updated organization data
	distributorResponse := gin.H{
		"id":            updatedOrg.ID,
		"name":          updatedOrg.Name,
		"description":   updatedOrg.Description,
		"customData":    updatedOrg.CustomData,
		"isMfaRequired": updatedOrg.IsMfaRequired,
		"type":          "distributor",
		"updatedAt":     time.Now(),
	}

	c.JSON(http.StatusOK, response.OK("distributor updated successfully", distributorResponse))
}

// DeleteDistributor handles DELETE /api/distributors/:id - deletes a distributor organization from Logto
func DeleteDistributor(c *gin.Context) {
	distributorID := c.Param("id")
	if distributorID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("distributor ID required", nil))
		return
	}

	_, _ = c.Get("user_id")

	// Connect to Logto Management API
	client := services.NewLogtoManagementClient()

	// First, verify the organization exists and get its data for logging
	currentOrg, err := client.GetOrganizationByID(distributorID)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "distributors").LogError(err, "fetch_distributor_for_deletion", http.StatusInternalServerError, "Failed to fetch distributor organization for deletion")
		c.JSON(http.StatusNotFound, response.NotFound("distributor not found", nil))
		return
	}

	// Delete the organization from Logto
	if err := client.DeleteOrganization(distributorID); err != nil {
		logger.NewHTTPErrorLogger(c, "distributors").LogError(err, "delete_distributor_organization", http.StatusInternalServerError, "Failed to delete distributor organization from Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to delete distributor organization", err.Error()))
		return
	}

	logger.LogBusinessOperation(c, "distributors", "delete", "distributor", distributorID, true, nil)

	c.JSON(http.StatusOK, response.OK("distributor deleted successfully", gin.H{
		"id":        distributorID,
		"name":      currentOrg.Name,
		"deletedAt": time.Now(),
	}))
}
