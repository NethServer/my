/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services"
)

// CreateReseller handles POST /api/resellers - creates a new reseller organization in Logto
func CreateReseller(c *gin.Context) {
	var request models.CreateResellerRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequest(err))
		return
	}

	_, _ = c.Get("user_id") // user context verified by middleware
	userOrgID, _ := c.Get("organization_id")

	// Create organization in Logto
	client := services.NewLogtoManagementClient()

	// Check if organization name is unique
	isUnique, err := client.CheckOrganizationNameUniqueness(request.Name)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "resellers").LogError(err, "check_name_uniqueness", http.StatusInternalServerError, "Failed to check organization name uniqueness")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to validate organization name", err.Error()))
		return
	}
	if !isUnique {
		c.JSON(http.StatusConflict, response.Conflict("organization name already exists", gin.H{"name": request.Name}))
		return
	}

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
		logger.NewHTTPErrorLogger(c, "resellers").LogError(err, "create_reseller_organization", http.StatusInternalServerError, "Failed to create reseller organization in Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create reseller organization", err.Error()))
		return
	}

	// Assign Reseller role as default JIT role
	resellerRole, err := client.GetOrganizationRoleByName("Reseller")
	if err != nil {
		logger.NewHTTPErrorLogger(c, "resellers").LogError(err, "find_reseller_role", http.StatusInternalServerError, "Failed to find Reseller role")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to configure reseller role", err.Error()))
		return
	}

	if err := client.AssignOrganizationJitRoles(org.ID, []string{resellerRole.ID}); err != nil {
		logger.NewHTTPErrorLogger(c, "resellers").LogError(err, "assign_reseller_jit_role", http.StatusInternalServerError, "Failed to assign Reseller JIT role")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to configure reseller permissions", err.Error()))
		return
	}

	logger.LogBusinessOperation(c, "resellers", "create", "reseller", org.ID, true, nil)

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

	c.JSON(http.StatusCreated, response.Created("reseller created successfully", resellerResponse))
}

// GetReseller handles GET /api/resellers/:id - retrieves a single reseller organization from Logto
func GetReseller(c *gin.Context) {
	resellerID := c.Param("id")
	if resellerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("reseller ID required", nil))
		return
	}

	_, _ = c.Get("user_id")
	userOrgRole, _ := c.Get("org_role")
	userOrgID, _ := c.Get("organization_id")

	logger.RequestLogger(c, "resellers").Info().
		Str("operation", "get_reseller").
		Str("reseller_id", resellerID).
		Msg("Single reseller requested")

	// Connect to Logto Management API
	client := services.NewLogtoManagementClient()

	// Get the specific organization
	org, err := client.GetOrganizationByID(resellerID)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "resellers").LogError(err, "fetch_reseller", http.StatusInternalServerError, "Failed to fetch reseller from Logto")
		c.JSON(http.StatusNotFound, response.NotFound("reseller not found", nil))
		return
	}

	// Verify this is actually a reseller organization
	if org.CustomData == nil || org.CustomData["type"] != "reseller" {
		c.JSON(http.StatusNotFound, response.NotFound("reseller not found", nil))
		return
	}

	// Apply visibility filtering - ensure user can see this reseller
	orgs := []services.LogtoOrganization{*org}
	filteredOrgs := services.FilterOrganizationsByVisibility(orgs, userOrgRole.(string), userOrgID.(string), "Reseller")

	if len(filteredOrgs) == 0 {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to this reseller", nil))
		return
	}

	// Convert to reseller format
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

	logger.RequestLogger(c, "resellers").Info().
		Str("operation", "get_reseller_result").
		Str("reseller_id", resellerID).
		Msg("Retrieved reseller from Logto")

	c.JSON(http.StatusOK, response.OK("reseller retrieved successfully", reseller))
}

// GetResellers handles GET /api/resellers - retrieves all resellers
func GetResellers(c *gin.Context) {
	_, _ = c.Get("user_id")
	userOrgRole, _ := c.Get("org_role")
	userOrgID, _ := c.Get("organization_id")

	// Parse query parameters
	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		if parsedPage, err := strconv.Atoi(p); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if parsedPageSize, err := strconv.Atoi(ps); err == nil && parsedPageSize > 0 && parsedPageSize <= 100 {
			pageSize = parsedPageSize
		}
	}

	// Parse filters
	filters := services.OrganizationFilters{
		Name:        c.Query("name"),
		Description: c.Query("description"),
		Type:        "reseller", // Fixed for resellers
		CreatedBy:   c.Query("created_by"),
		Search:      c.Query("search"),
	}

	logger.RequestLogger(c, "resellers").Info().
		Str("operation", "list_resellers").
		Int("page", page).
		Int("page_size", pageSize).
		Str("search", filters.Search).
		Msg("Resellers list requested")

	// Get organizations with Reseller role from Logto (paginated)
	result, err := services.GetOrganizationsByRolePaginated("Reseller", page, pageSize, filters)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "resellers").LogError(err, "fetch_resellers", http.StatusInternalServerError, "Failed to fetch resellers from Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch resellers", nil))
		return
	}

	// Apply visibility filtering
	filteredOrgs := services.FilterOrganizationsByVisibility(result.Data, userOrgRole.(string), userOrgID.(string), "Reseller")

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

	// Update pagination info with filtered count
	paginationInfo := result.Pagination
	paginationInfo.TotalCount = len(filteredOrgs)
	paginationInfo.TotalPages = (paginationInfo.TotalCount + pageSize - 1) / pageSize

	logger.RequestLogger(c, "resellers").Info().
		Int("reseller_count", len(resellers)).
		Int("total_count", paginationInfo.TotalCount).
		Int("page", page).
		Str("operation", "fetch_resellers_result").
		Msg("Retrieved resellers from Logto")

	c.JSON(http.StatusOK, response.OK("resellers retrieved successfully", gin.H{
		"resellers":  resellers,
		"pagination": paginationInfo,
	}))
}

// UpdateReseller handles PUT /api/resellers/:id - updates an existing reseller organization in Logto
func UpdateReseller(c *gin.Context) {
	resellerID := c.Param("id")
	if resellerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("reseller ID required", nil))
		return
	}

	var request models.UpdateResellerRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequest(err))
		return
	}

	_, _ = c.Get("user_id")
	userOrgID, _ := c.Get("organization_id")

	// Connect to Logto Management API
	client := services.NewLogtoManagementClient()

	// First, verify the organization exists and get current data
	currentOrg, err := client.GetOrganizationByID(resellerID)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "resellers").LogError(err, "fetch_reseller_organization", http.StatusInternalServerError, "Failed to fetch reseller organization")
		c.JSON(http.StatusNotFound, response.NotFound("reseller not found", nil))
		return
	}

	// Prepare update request with only changed fields
	updateRequest := services.UpdateOrganizationRequest{}

	// Update name if provided
	if request.Name != "" {
		// Check if new name is unique (if different from current)
		if request.Name != currentOrg.Name {
			isUnique, err := client.CheckOrganizationNameUniqueness(request.Name)
			if err != nil {
				logger.NewHTTPErrorLogger(c, "resellers").LogError(err, "check_name_uniqueness_update", http.StatusInternalServerError, "Failed to check organization name uniqueness for update")
				c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to validate organization name", err.Error()))
				return
			}
			if !isUnique {
				c.JSON(http.StatusConflict, response.Conflict("organization name already exists", gin.H{"name": request.Name}))
				return
			}
		}
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
		logger.NewHTTPErrorLogger(c, "resellers").LogError(err, "update_reseller_organization", http.StatusInternalServerError, "Failed to update reseller organization in Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to update reseller organization", err.Error()))
		return
	}

	logger.LogBusinessOperation(c, "resellers", "update", "reseller", resellerID, true, nil)

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

	c.JSON(http.StatusOK, response.OK("reseller updated successfully", resellerResponse))
}

// DeleteReseller handles DELETE /api/resellers/:id - deletes a reseller organization from Logto
func DeleteReseller(c *gin.Context) {
	resellerID := c.Param("id")
	if resellerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("reseller ID required", nil))
		return
	}

	_, _ = c.Get("user_id")

	// Connect to Logto Management API
	client := services.NewLogtoManagementClient()

	// First, verify the organization exists and get its data for logging
	currentOrg, err := client.GetOrganizationByID(resellerID)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "resellers").LogError(err, "fetch_reseller_for_deletion", http.StatusInternalServerError, "Failed to fetch reseller organization for deletion")
		c.JSON(http.StatusNotFound, response.NotFound("reseller not found", nil))
		return
	}

	// Delete the organization from Logto
	if err := client.DeleteOrganization(resellerID); err != nil {
		logger.NewHTTPErrorLogger(c, "resellers").LogError(err, "delete_reseller_organization", http.StatusInternalServerError, "Failed to delete reseller organization from Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to delete reseller organization", err.Error()))
		return
	}

	logger.LogBusinessOperation(c, "resellers", "delete", "reseller", resellerID, true, nil)

	c.JSON(http.StatusOK, response.OK("reseller deleted successfully", gin.H{
		"id":        resellerID,
		"name":      currentOrg.Name,
		"deletedAt": time.Now(),
	}))
}
