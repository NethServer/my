/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
	local "github.com/nethesis/my/backend/services/local"
)

// Asset validation constants
const (
	maxLogoSize       = 2 * 1024 * 1024 // 2MB
	maxFaviconSize    = 512 * 1024      // 512KB
	maxBackgroundSize = 5 * 1024 * 1024 // 5MB
	maxProductName    = 100
)

var allowedLogoMimes = map[string]bool{
	"image/png":     true,
	"image/svg+xml": true,
	"image/webp":    true,
}

var allowedFaviconMimes = map[string]bool{
	"image/png":                true,
	"image/x-icon":             true,
	"image/vnd.microsoft.icon": true,
	"image/svg+xml":            true,
}

var allowedBackgroundMimes = map[string]bool{
	"image/png":     true,
	"image/jpeg":    true,
	"image/webp":    true,
	"image/svg+xml": true,
}

// assetConfig holds validation rules per asset type
type assetConfig struct {
	maxSize      int64
	allowedMimes map[string]bool
}

var assetConfigs = map[string]assetConfig{
	"logo_light_rect":   {maxSize: maxLogoSize, allowedMimes: allowedLogoMimes},
	"logo_dark_rect":    {maxSize: maxLogoSize, allowedMimes: allowedLogoMimes},
	"logo_light_square": {maxSize: maxLogoSize, allowedMimes: allowedLogoMimes},
	"logo_dark_square":  {maxSize: maxLogoSize, allowedMimes: allowedLogoMimes},
	"favicon":           {maxSize: maxFaviconSize, allowedMimes: allowedFaviconMimes},
	"background_image":  {maxSize: maxBackgroundSize, allowedMimes: allowedBackgroundMimes},
}

// GetRebrandingProducts returns all rebrandable products
func GetRebrandingProducts(c *gin.Context) {
	service := local.NewRebrandingService()
	products, err := service.ListProducts()
	if err != nil {
		logger.Error().Err(err).Msg("failed to list rebrandable products")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to list rebrandable products", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("rebrandable products retrieved successfully", gin.H{
		"products": products,
	}))
}

// EnableRebranding enables rebranding for an organization (Owner + Admin only)
func EnableRebranding(c *gin.Context) {
	orgID := c.Param("org_id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("organization id is required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Only Owner org role can enable rebranding
	if strings.ToLower(user.OrgRole) != "owner" {
		c.JSON(http.StatusForbidden, response.Forbidden("only owner organization can enable rebranding", nil))
		return
	}

	// Determine org type
	userService := local.NewUserService()
	orgType := userService.GetOrganizationType(orgID)
	if orgType == "owner" {
		c.JSON(http.StatusBadRequest, response.BadRequest("cannot enable rebranding for owner organization", nil))
		return
	}

	service := local.NewRebrandingService()
	if err := service.EnableRebranding(orgID, orgType); err != nil {
		logger.Error().Err(err).Str("organization_id", orgID).Msg("failed to enable rebranding")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to enable rebranding", nil))
		return
	}

	logger.LogBusinessOperation(c, "rebranding", "enable", "organization", orgID, true, nil)
	c.JSON(http.StatusOK, response.OK("rebranding enabled successfully", nil))
}

// DisableRebranding disables rebranding for an organization (Owner + Admin only)
func DisableRebranding(c *gin.Context) {
	orgID := c.Param("org_id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("organization id is required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	if strings.ToLower(user.OrgRole) != "owner" {
		c.JSON(http.StatusForbidden, response.Forbidden("only owner organization can disable rebranding", nil))
		return
	}

	service := local.NewRebrandingService()
	if err := service.DisableRebranding(orgID); err != nil {
		logger.Error().Err(err).Str("organization_id", orgID).Msg("failed to disable rebranding")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to disable rebranding", nil))
		return
	}

	logger.LogBusinessOperation(c, "rebranding", "disable", "organization", orgID, true, nil)
	c.JSON(http.StatusOK, response.OK("rebranding disabled successfully", nil))
}

// GetRebrandingStatus returns the rebranding status for an organization
func GetRebrandingStatus(c *gin.Context) {
	orgID := c.Param("org_id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("organization id is required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Check access: owner can see all, others can see their own org
	if strings.ToLower(user.OrgRole) != "owner" {
		userService := local.NewUserService()
		if !userService.IsOrganizationInHierarchy(strings.ToLower(user.OrgRole), user.OrganizationID, orgID) {
			c.JSON(http.StatusForbidden, response.Forbidden("access denied", nil))
			return
		}
	}

	service := local.NewRebrandingService()
	status, err := service.GetOrgStatus(orgID)
	if err != nil {
		logger.Error().Err(err).Str("organization_id", orgID).Msg("failed to get rebranding status")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get rebranding status", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("rebranding status retrieved successfully", status))
}

// GetRebrandingOrgProducts returns rebranding products configuration for an organization
func GetRebrandingOrgProducts(c *gin.Context) {
	orgID := c.Param("org_id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("organization id is required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Check access
	if strings.ToLower(user.OrgRole) != "owner" {
		userService := local.NewUserService()
		if !userService.IsOrganizationInHierarchy(strings.ToLower(user.OrgRole), user.OrganizationID, orgID) {
			c.JSON(http.StatusForbidden, response.Forbidden("access denied", nil))
			return
		}
	}

	service := local.NewRebrandingService()
	status, err := service.GetOrgStatus(orgID)
	if err != nil {
		logger.Error().Err(err).Str("organization_id", orgID).Msg("failed to get rebranding products")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get rebranding products", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("rebranding products retrieved successfully", status))
}

// UploadRebrandingAssets handles multipart upload of rebranding assets for an org+product
func UploadRebrandingAssets(c *gin.Context) {
	orgID := c.Param("org_id")
	productID := c.Param("product_id")

	if orgID == "" || productID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("organization id and product id are required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Check access: owner can upload for any org, others for their own org only
	userOrgRole := strings.ToLower(user.OrgRole)
	if userOrgRole != "owner" {
		userService := local.NewUserService()
		if !userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, orgID) {
			c.JSON(http.StatusForbidden, response.Forbidden("access denied", nil))
			return
		}
	}

	// Check rebranding is enabled for this org
	service := local.NewRebrandingService()
	enabled, err := service.IsRebrandingEnabled(orgID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to check rebranding status")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to check rebranding status", nil))
		return
	}
	if !enabled {
		c.JSON(http.StatusForbidden, response.Forbidden("rebranding is not enabled for this organization", nil))
		return
	}

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(maxBackgroundSize + maxLogoSize*4 + maxFaviconSize); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid multipart form", nil))
		return
	}

	fields := make(map[string][]byte)
	mimeTypes := make(map[string]string)

	// Process each asset field
	for fieldName, config := range assetConfigs {
		file, header, err := c.Request.FormFile(fieldName)
		if err != nil {
			continue // Field not provided, skip
		}
		defer func() { _ = file.Close() }()

		// Validate size
		if header.Size > config.maxSize {
			c.JSON(http.StatusBadRequest, response.BadRequest(
				fieldName+" exceeds maximum size",
				gin.H{"field": fieldName, "max_size": config.maxSize, "actual_size": header.Size},
			))
			return
		}

		// Validate MIME type
		contentType := header.Header.Get("Content-Type")
		if !config.allowedMimes[contentType] {
			c.JSON(http.StatusBadRequest, response.BadRequest(
				fieldName+" has invalid content type",
				gin.H{"field": fieldName, "content_type": contentType},
			))
			return
		}

		// Read file data
		data, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to read "+fieldName, nil))
			return
		}

		fields[fieldName] = data
		mimeTypes[fieldName] = contentType
	}

	// Get product_name from form
	var productName *string
	if pn := c.Request.FormValue("product_name"); pn != "" {
		if len(pn) > maxProductName {
			c.JSON(http.StatusBadRequest, response.BadRequest("product name exceeds maximum length", gin.H{"max_length": maxProductName}))
			return
		}
		productName = &pn
	}

	// Must have at least one field
	if len(fields) == 0 && productName == nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("at least one asset or product_name is required", nil))
		return
	}

	if err := service.UpsertAssets(orgID, productID, productName, fields, mimeTypes); err != nil {
		logger.Error().Err(err).Str("organization_id", orgID).Str("product_id", productID).Msg("failed to upload rebranding assets")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to upload rebranding assets", nil))
		return
	}

	logger.LogBusinessOperation(c, "rebranding", "upload", "assets", orgID+"/"+productID, true, nil)
	c.JSON(http.StatusOK, response.OK("rebranding assets uploaded successfully", nil))
}

// DeleteRebrandingProduct deletes all rebranding assets for an org+product
func DeleteRebrandingProduct(c *gin.Context) {
	orgID := c.Param("org_id")
	productID := c.Param("product_id")

	if orgID == "" || productID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("organization id and product id are required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	userOrgRole := strings.ToLower(user.OrgRole)
	if userOrgRole != "owner" {
		userService := local.NewUserService()
		if !userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, orgID) {
			c.JSON(http.StatusForbidden, response.Forbidden("access denied", nil))
			return
		}
	}

	service := local.NewRebrandingService()
	if err := service.DeleteProductAssets(orgID, productID); err != nil {
		logger.Error().Err(err).Str("organization_id", orgID).Str("product_id", productID).Msg("failed to delete rebranding product")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to delete rebranding product", nil))
		return
	}

	logger.LogBusinessOperation(c, "rebranding", "delete", "product", orgID+"/"+productID, true, nil)
	c.JSON(http.StatusOK, response.OK("rebranding product deleted successfully", nil))
}

// DeleteRebrandingAsset deletes a single asset for an org+product
func DeleteRebrandingAsset(c *gin.Context) {
	orgID := c.Param("org_id")
	productID := c.Param("product_id")
	assetName := c.Param("asset")

	if orgID == "" || productID == "" || assetName == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("organization id, product id, and asset name are required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	userOrgRole := strings.ToLower(user.OrgRole)
	if userOrgRole != "owner" {
		userService := local.NewUserService()
		if !userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, orgID) {
			c.JSON(http.StatusForbidden, response.Forbidden("access denied", nil))
			return
		}
	}

	service := local.NewRebrandingService()
	if err := service.DeleteSingleAsset(orgID, productID, assetName); err != nil {
		logger.Error().Err(err).Str("organization_id", orgID).Str("product_id", productID).Str("asset", assetName).Msg("failed to delete rebranding asset")
		c.JSON(http.StatusNotFound, response.NotFound("asset not found", nil))
		return
	}

	logger.LogBusinessOperation(c, "rebranding", "delete_asset", "asset", orgID+"/"+productID+"/"+assetName, true, nil)
	c.JSON(http.StatusOK, response.OK("rebranding asset deleted successfully", nil))
}

// GetRebrandingAsset serves a single asset binary for an org+product (authenticated via JWT)
func GetRebrandingAsset(c *gin.Context) {
	orgID := c.Param("org_id")
	productID := c.Param("product_id")
	assetName := c.Param("asset")

	if orgID == "" || productID == "" || assetName == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("organization id, product id, and asset name are required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Check access
	userOrgRole := strings.ToLower(user.OrgRole)
	if userOrgRole != "owner" {
		userService := local.NewUserService()
		if !userService.IsOrganizationInHierarchy(userOrgRole, user.OrganizationID, orgID) {
			c.JSON(http.StatusForbidden, response.Forbidden("access denied", nil))
			return
		}
	}

	service := local.NewRebrandingService()
	data, mimeType, err := service.GetAssetBinary(orgID, productID, assetName)
	if err != nil {
		c.JSON(http.StatusNotFound, response.NotFound("asset not found", nil))
		return
	}

	c.Data(http.StatusOK, mimeType, data)
}

// resolveRebranding resolves rebranding status for a given organization ID.
// Returns (enabled, rebrandingOrgID). If rebranding is not enabled or an error occurs,
// returns (false, nil).
func resolveRebranding(orgID string) (bool, *string) {
	rebrandingService := local.NewRebrandingService()
	enabled, resolvedOrgID, err := rebrandingService.ResolveRebranding(orgID)
	if err == nil && enabled {
		return true, &resolvedOrgID
	}
	return false, nil
}
