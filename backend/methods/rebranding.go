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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	local "github.com/nethesis/my/backend/services/local"
)

// Asset validation constants
const (
	maxLogoSize       = 2 * 1024 * 1024 // 2MB
	maxFaviconSize    = 512 * 1024      // 512KB
	maxBackgroundSize = 5 * 1024 * 1024 // 5MB
	maxProductName    = 100
	maxFilename       = 255

	maxAvailableOrganizations = 200
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
		"products": helpers.EnsureSlice(products),
	}))
}

// GetRebrandingOrganizations handles GET /api/rebranding/organizations - the
// organizations that have rebranding enabled, with the products each has branded.
func GetRebrandingOrganizations(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	page, pageSize, sortBy, sortDirection := helpers.GetPaginationAndSortingFromQuery(c)

	filters := local.RebrandingOrganizationFilters{
		Search:        c.Query("search"),
		Types:         c.QueryArray("type"),
		ProductIDs:    c.QueryArray("product"),
		Page:          page,
		PageSize:      pageSize,
		SortBy:        sortBy,
		SortDirection: sortDirection,
	}

	service := local.NewRebrandingService()
	organizations, totalCount, err := service.ListOrganizations(strings.ToLower(user.OrgRole), user.OrganizationID, filters)
	if err != nil {
		logger.Error().Err(err).Msg("failed to list rebranding organizations")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to list rebranding organizations", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("rebranding organizations retrieved successfully", gin.H{
		"organizations": helpers.EnsureSlice(organizations),
		"pagination":    helpers.BuildPaginationInfoWithSorting(page, pageSize, totalCount, sortBy, sortDirection),
	}))
}

// GetRebrandingSummary handles GET /api/rebranding/summary - the counters shown
// above the organizations list.
func GetRebrandingSummary(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	service := local.NewRebrandingService()
	summary, err := service.Summary(strings.ToLower(user.OrgRole), user.OrganizationID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to summarize rebranding organizations")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to summarize rebranding organizations", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("rebranding summary retrieved successfully", summary))
}

// GetAvailableRebrandingOrganizations handles GET /api/rebranding/organizations/available -
// the organizations that can still be added, for the picker.
func GetAvailableRebrandingOrganizations(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Clamped rather than rejected, the way helpers.GetPaginationFromQuery treats
	// page_size: an out-of-range limit on a picker is not worth an error.
	limit := 50
	if parsed, err := strconv.Atoi(c.Query("limit")); err == nil && parsed > 0 {
		limit = min(parsed, maxAvailableOrganizations)
	}

	service := local.NewRebrandingService()
	organizations, err := service.ListAvailableOrganizations(
		strings.ToLower(user.OrgRole), user.OrganizationID, c.Query("search"), limit)
	if err != nil {
		logger.Error().Err(err).Msg("failed to list organizations available for rebranding")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to list organizations available for rebranding", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("available organizations retrieved successfully", gin.H{
		"organizations": helpers.EnsureSlice(organizations),
	}))
}

// EnableRebranding enables rebranding for an organization (Owner only)
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

// EnableRebrandingBulk handles POST /api/rebranding/organizations - adds the
// whole selection of the picker in one call. Either every organization is
// added or none is, so a partial failure cannot leave the caller with half a
// selection enabled and no error to show.
func EnableRebrandingBulk(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	if strings.ToLower(user.OrgRole) != "owner" {
		c.JSON(http.StatusForbidden, response.Forbidden("only owner organization can enable rebranding", nil))
		return
	}

	var request models.EnableRebrandingBulkRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	service := local.NewRebrandingService()
	enabled, invalid, err := service.EnableRebrandingBulk(request.OrganizationIDs)
	if err != nil {
		logger.Error().Err(err).Msg("failed to enable rebranding for organizations")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to enable rebranding", nil))
		return
	}
	if len(invalid) > 0 {
		validationErrors := make([]response.ValidationError, 0, len(invalid))
		for _, id := range invalid {
			validationErrors = append(validationErrors, response.ValidationError{
				Key: "organization_ids", Message: "unknown", Value: id,
			})
		}
		c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", validationErrors))
		return
	}

	logger.LogBusinessOperation(c, "rebranding", "enable", "organizations",
		strings.Join(request.OrganizationIDs, ","), true, nil)
	c.JSON(http.StatusOK, response.OK("rebranding enabled successfully", gin.H{"enabled": enabled}))
}

// DisableRebranding disables rebranding for an organization (Owner only).
// The organization's assets are deleted with it.
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

	if !canReadRebranding(c, orgID) {
		return
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

	if !canReadRebranding(c, orgID) {
		return
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

// SaveRebrandingConfig handles PUT /api/rebranding/:org_id/config - the whole
// configuration form in one multipart request: the products the branding
// applies to, the brand name, the assets being uploaded and the ones being
// emptied. Products left out of the selection lose their configuration.
func SaveRebrandingConfig(c *gin.Context) {
	orgID := c.Param("org_id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("organization id is required", nil))
		return
	}

	if !canWriteRebranding(c, orgID) {
		return
	}

	service := local.NewRebrandingService()
	if !requireRebrandingEnabled(c, service, orgID) {
		return
	}

	if !parseRebrandingForm(c) {
		return
	}

	products := formList(c, "products")
	if len(products) == 0 {
		c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", []response.ValidationError{
			{Key: "products", Message: "at_least_one_required"},
		}))
		return
	}

	brandName, ok := formBrandName(c, "brand_name")
	if !ok {
		return
	}

	uploads, ok := parseRebrandingUploads(c)
	if !ok {
		return
	}

	cfg := models.RebrandingConfig{
		Products:  products,
		BrandName: brandName,
		Uploads:   uploads,
		Clear:     formList(c, "clear"),
	}

	if err := service.SaveConfig(orgID, cfg); err != nil {
		writeRebrandingWriteError(c, err, orgID, strings.Join(products, ","))
		return
	}

	// What the confirmation reports: the organizations downstream that display
	// this branding because they have none of their own.
	applied, err := service.CountInheritingOrganizations(orgID)
	if err != nil {
		logger.Warn().Err(err).Str("organization_id", orgID).Msg("failed to count organizations inheriting rebranding")
	}

	logger.LogBusinessOperation(c, "rebranding", "save", "config", orgID, true, nil)
	c.JSON(http.StatusOK, response.OK("rebranding configuration saved successfully", gin.H{
		"applied_to_organizations": applied,
	}))
}

// UploadRebrandingAssets handles multipart upload of rebranding assets for an org+product
func UploadRebrandingAssets(c *gin.Context) {
	orgID := c.Param("org_id")
	productID := c.Param("product_id")

	if orgID == "" || productID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("organization id and product id are required", nil))
		return
	}

	if !canWriteRebranding(c, orgID) {
		return
	}

	service := local.NewRebrandingService()
	if !requireRebrandingEnabled(c, service, orgID) {
		return
	}

	if !parseRebrandingForm(c) {
		return
	}

	uploads, ok := parseRebrandingUploads(c)
	if !ok {
		return
	}

	productName, ok := formBrandName(c, "product_name")
	if !ok {
		return
	}

	clear := formList(c, "clear")

	if len(uploads) == 0 && productName == nil && len(clear) == 0 {
		c.JSON(http.StatusBadRequest, response.BadRequest("at least one asset or product_name is required", nil))
		return
	}

	if err := service.UpsertAssets(orgID, productID, productName, uploads, clear); err != nil {
		writeRebrandingWriteError(c, err, orgID, productID)
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

	if !canWriteRebranding(c, orgID) {
		return
	}

	service := local.NewRebrandingService()
	if err := service.DeleteProductAssets(orgID, productID); err != nil {
		// Nothing to delete is a 404, not a server failure: answering 500 there
		// hid the real outcome and made an authorization probe unreadable.
		if errors.Is(err, local.ErrRebrandingAssetsNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("rebranding product not found", nil))
			return
		}
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

	if !canWriteRebranding(c, orgID) {
		return
	}

	service := local.NewRebrandingService()
	if err := service.DeleteSingleAsset(orgID, productID, assetName); err != nil {
		// The mirror image of the bug above: this collapsed every failure into
		// 404, so a storage outage was reported as "asset not found".
		switch {
		case errors.Is(err, local.ErrInvalidRebrandingAsset):
			c.JSON(http.StatusBadRequest, response.BadRequest("invalid asset name", nil))
		case errors.Is(err, local.ErrRebrandingAssetsNotFound):
			c.JSON(http.StatusNotFound, response.NotFound("asset not found", nil))
		default:
			logger.Error().Err(err).Str("organization_id", orgID).Str("product_id", productID).Str("asset", assetName).Msg("failed to delete rebranding asset")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to delete rebranding asset", nil))
		}
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

	if !canReadRebranding(c, orgID) {
		return
	}

	serveRebrandingAsset(c, orgID, productID, assetName, "private, max-age=300, must-revalidate")
}

// GetPublicRebrandingAsset serves an asset with no token, so a page can point an
// <img> straight at it. Branding is what a partner shows on its own login
// screens, so the binary is not a secret; the endpoint is rate-limited and
// serves nothing but the image.
func GetPublicRebrandingAsset(c *gin.Context) {
	orgID := c.Param("org_id")
	productID := c.Param("product_id")
	assetName := c.Param("asset")

	if orgID == "" || productID == "" || assetName == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("organization id, product id, and asset name are required", nil))
		return
	}

	serveRebrandingAsset(c, orgID, productID, assetName, "public, max-age=300, must-revalidate")
}

func serveRebrandingAsset(c *gin.Context, orgID, productID, assetName, cacheControl string) {
	service := local.NewRebrandingService()
	data, mimeType, updatedAt, err := service.GetAssetBinary(orgID, productID, assetName)
	if err != nil {
		if errors.Is(err, local.ErrInvalidRebrandingAsset) {
			c.JSON(http.StatusBadRequest, response.BadRequest("invalid asset name", nil))
			return
		}
		c.JSON(http.StatusNotFound, response.NotFound("asset not found", nil))
		return
	}

	etag := fmt.Sprintf(`"%d-%d"`, updatedAt.UnixNano(), len(data))
	c.Header("ETag", etag)
	c.Header("Last-Modified", updatedAt.UTC().Format(http.TimeFormat))
	c.Header("Cache-Control", cacheControl)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", "inline")
	// An uploaded SVG is markup, and SVG is the recommended format here. Served
	// from the API origin it would be a place to run script against that origin,
	// so the response is stripped of every capability an image does not need.
	c.Header("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")

	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}

	c.Data(http.StatusOK, mimeType, data)
}

// canReadRebranding allows the owner everywhere and everyone else within their
// own hierarchy. It answers 403 itself when access is denied.
func canReadRebranding(c *gin.Context, orgID string) bool {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return false
	}

	if strings.ToLower(user.OrgRole) == "owner" {
		return true
	}

	userService := local.NewUserService()
	if userService.IsOrganizationInHierarchy(strings.ToLower(user.OrgRole), user.OrganizationID, orgID) {
		return true
	}

	// The hierarchy check only looks downwards, but branding is inherited
	// upwards: a customer displays the branding of the organization above it and
	// has to be able to read those assets. Only that one organization, though —
	// the one the caller actually inherits from.
	if enabled, resolved := resolveRebranding(user.OrganizationID); enabled && resolved != nil && *resolved == orgID {
		return true
	}

	c.JSON(http.StatusForbidden, response.Forbidden("access denied", nil))
	return false
}

// canWriteRebranding allows the owner everywhere and everyone else on their own
// organization only. A partner configures its own branding and the
// organizations below inherit it; writing into one of them directly would
// override a configuration its own administrators own.
func canWriteRebranding(c *gin.Context, orgID string) bool {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return false
	}

	if strings.ToLower(user.OrgRole) == "owner" || user.OrganizationID == orgID {
		return true
	}

	c.JSON(http.StatusForbidden, response.Forbidden("rebranding can only be configured for your own organization", nil))
	return false
}

func requireRebrandingEnabled(c *gin.Context, service *local.RebrandingService, orgID string) bool {
	enabled, err := service.IsRebrandingEnabled(orgID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to check rebranding status")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to check rebranding status", nil))
		return false
	}
	if !enabled {
		c.JSON(http.StatusForbidden, response.Forbidden("rebranding is not enabled for this organization", nil))
		return false
	}
	return true
}

func parseRebrandingForm(c *gin.Context) bool {
	if err := c.Request.ParseMultipartForm(maxBackgroundSize + maxLogoSize*4 + maxFaviconSize); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid multipart form", nil))
		return false
	}
	return true
}

// parseRebrandingUploads reads the asset fields of the form, validating size and
// content type. It answers the request itself on the first invalid asset.
func parseRebrandingUploads(c *gin.Context) (map[string]models.RebrandingUpload, bool) {
	uploads := make(map[string]models.RebrandingUpload)

	for fieldName, config := range assetConfigs {
		file, header, err := c.Request.FormFile(fieldName)
		if err != nil {
			continue // Field not provided, skip
		}
		defer func() { _ = file.Close() }()

		if header.Size > config.maxSize {
			c.JSON(http.StatusBadRequest, response.BadRequest(
				fieldName+" exceeds maximum size",
				gin.H{"field": fieldName, "max_size": config.maxSize, "actual_size": header.Size},
			))
			return nil, false
		}

		contentType := header.Header.Get("Content-Type")
		if !config.allowedMimes[contentType] {
			c.JSON(http.StatusBadRequest, response.BadRequest(
				fieldName+" has invalid content type",
				gin.H{"field": fieldName, "content_type": contentType},
			))
			return nil, false
		}

		data, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to read "+fieldName, nil))
			return nil, false
		}

		filename := header.Filename
		if len(filename) > maxFilename {
			filename = filename[:maxFilename]
		}

		uploads[fieldName] = models.RebrandingUpload{Data: data, MimeType: contentType, Filename: filename}
	}

	return uploads, true
}

// formList reads a repeatable form field that also accepts a comma-separated
// list, so "products=nethvoice,nsec" and two products fields mean the same.
func formList(c *gin.Context, field string) []string {
	values := make([]string, 0)
	if c.Request.MultipartForm == nil {
		return values
	}
	for _, raw := range c.Request.MultipartForm.Value[field] {
		for _, item := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				values = append(values, trimmed)
			}
		}
	}
	return values
}

func formBrandName(c *gin.Context, field string) (*string, bool) {
	value := c.Request.FormValue(field)
	if value == "" {
		return nil, true
	}
	if len(value) > maxProductName {
		// "max" is the code the binding validator emits for a max=N tag, so the
		// frontend handles one code whether the limit is checked here or there.
		c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", []response.ValidationError{
			{Key: field, Message: "max", Value: value},
		}))
		return nil, false
	}
	return &value, true
}

func writeRebrandingWriteError(c *gin.Context, err error, orgID, productIDs string) {
	// An unknown product id or asset name is a field the caller got wrong, and
	// the caller needs to know which one: answer in the validation shape, with
	// the value that was rejected.
	var fieldErr *local.RebrandingFieldError
	if errors.As(err, &fieldErr) {
		c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", []response.ValidationError{
			{Key: fieldErr.Field, Message: "unknown", Value: fieldErr.Value},
		}))
		return
	}

	if errors.Is(err, local.ErrNoRebrandingProducts) {
		c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", []response.ValidationError{
			{Key: "products", Message: "at_least_one_required"},
		}))
		return
	}

	logger.Error().Err(err).Str("organization_id", orgID).Str("product_id", productIDs).
		Msg("failed to save rebranding configuration")
	c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to save rebranding configuration", nil))
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
