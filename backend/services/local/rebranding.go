/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"database/sql"
	"fmt"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// RebrandingService handles rebranding operations
type RebrandingService struct{}

// NewRebrandingService creates a new rebranding service
func NewRebrandingService() *RebrandingService {
	return &RebrandingService{}
}

// ListProducts returns all rebrandable products
func (s *RebrandingService) ListProducts() ([]models.RebrandableProduct, error) {
	query := `SELECT id, display_name, type, created_at FROM rebrandable_products ORDER BY type, display_name`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rebrandable products: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var products []models.RebrandableProduct
	for rows.Next() {
		var p models.RebrandableProduct
		if err := rows.Scan(&p.ID, &p.DisplayName, &p.Type, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan rebrandable product: %w", err)
		}
		products = append(products, p)
	}
	return products, nil
}

// EnableRebranding enables rebranding for an organization
func (s *RebrandingService) EnableRebranding(orgID, orgType string) error {
	query := `
		INSERT INTO rebranding_enabled (organization_id, organization_type)
		VALUES ($1, $2)
		ON CONFLICT (organization_id) DO UPDATE SET organization_type = $2, enabled_at = NOW()
	`
	_, err := database.DB.Exec(query, orgID, orgType)
	if err != nil {
		return fmt.Errorf("failed to enable rebranding: %w", err)
	}
	return nil
}

// DisableRebranding disables rebranding for an organization
func (s *RebrandingService) DisableRebranding(orgID string) error {
	query := `DELETE FROM rebranding_enabled WHERE organization_id = $1`
	_, err := database.DB.Exec(query, orgID)
	if err != nil {
		return fmt.Errorf("failed to disable rebranding: %w", err)
	}
	return nil
}

// IsRebrandingEnabled checks if rebranding is enabled for an organization
func (s *RebrandingService) IsRebrandingEnabled(orgID string) (bool, error) {
	query := `SELECT COUNT(*) FROM rebranding_enabled WHERE organization_id = $1`
	var count int
	err := database.DB.QueryRow(query, orgID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check rebranding status: %w", err)
	}
	return count > 0, nil
}

// GetOrgStatus returns the full rebranding status for an organization
func (s *RebrandingService) GetOrgStatus(orgID string) (*models.RebrandingOrgStatus, error) {
	enabled, err := s.IsRebrandingEnabled(orgID)
	if err != nil {
		return nil, err
	}

	// Get all rebrandable products
	products, err := s.ListProducts()
	if err != nil {
		return nil, err
	}

	// Get assets for this org
	assetsQuery := `
		SELECT ra.product_id, ra.product_name,
			ra.logo_light_rect IS NOT NULL AS has_logo_light_rect,
			ra.logo_dark_rect IS NOT NULL AS has_logo_dark_rect,
			ra.logo_light_square IS NOT NULL AS has_logo_light_square,
			ra.logo_dark_square IS NOT NULL AS has_logo_dark_square,
			ra.favicon IS NOT NULL AS has_favicon,
			ra.background_image IS NOT NULL AS has_background_image
		FROM rebranding_assets ra
		WHERE ra.organization_id = $1
	`
	rows, err := database.DB.Query(assetsQuery, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query rebranding assets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Build a map of product_id -> asset info
	type assetInfo struct {
		productName *string
		assets      []string
	}
	assetMap := make(map[string]assetInfo)

	for rows.Next() {
		var productID string
		var productName *string
		var hasLLR, hasLDR, hasLLS, hasLDS, hasFav, hasBG bool

		if err := rows.Scan(&productID, &productName, &hasLLR, &hasLDR, &hasLLS, &hasLDS, &hasFav, &hasBG); err != nil {
			return nil, fmt.Errorf("failed to scan rebranding asset: %w", err)
		}

		var assets []string
		if hasLLR {
			assets = append(assets, "logo_light_rect")
		}
		if hasLDR {
			assets = append(assets, "logo_dark_rect")
		}
		if hasLLS {
			assets = append(assets, "logo_light_square")
		}
		if hasLDS {
			assets = append(assets, "logo_dark_square")
		}
		if hasFav {
			assets = append(assets, "favicon")
		}
		if hasBG {
			assets = append(assets, "background_image")
		}

		assetMap[productID] = assetInfo{productName: productName, assets: assets}
	}

	// Build response
	var productStatuses []models.RebrandingProductStatus
	for _, p := range products {
		ps := models.RebrandingProductStatus{
			ProductID:          p.ID,
			ProductDisplayName: p.DisplayName,
			ProductType:        p.Type,
			Assets:             []string{},
		}
		if info, ok := assetMap[p.ID]; ok {
			ps.ProductName = info.productName
			if info.assets != nil {
				ps.Assets = info.assets
			}
		}
		productStatuses = append(productStatuses, ps)
	}

	return &models.RebrandingOrgStatus{
		Enabled:  enabled,
		Products: productStatuses,
	}, nil
}

// UpsertAssets creates or updates rebranding assets for an organization+product
// Fields map contains field names to their binary data and mime type
func (s *RebrandingService) UpsertAssets(orgID, productID string, productName *string, fields map[string][]byte, mimeTypes map[string]string) error {
	// Verify product exists
	var exists bool
	err := database.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM rebrandable_products WHERE id = $1)`, productID).Scan(&exists)
	if err != nil || !exists {
		return fmt.Errorf("product not found: %s", productID)
	}

	// Build dynamic upsert
	query := `
		INSERT INTO rebranding_assets (organization_id, product_id, product_name,
			logo_light_rect, logo_dark_rect, logo_light_square, logo_dark_square, favicon, background_image,
			logo_light_rect_mime, logo_dark_rect_mime, logo_light_square_mime, logo_dark_square_mime, favicon_mime, background_image_mime)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (organization_id, product_id) DO UPDATE SET
			product_name = COALESCE($3, rebranding_assets.product_name),
			logo_light_rect = COALESCE($4, rebranding_assets.logo_light_rect),
			logo_dark_rect = COALESCE($5, rebranding_assets.logo_dark_rect),
			logo_light_square = COALESCE($6, rebranding_assets.logo_light_square),
			logo_dark_square = COALESCE($7, rebranding_assets.logo_dark_square),
			favicon = COALESCE($8, rebranding_assets.favicon),
			background_image = COALESCE($9, rebranding_assets.background_image),
			logo_light_rect_mime = COALESCE($10, rebranding_assets.logo_light_rect_mime),
			logo_dark_rect_mime = COALESCE($11, rebranding_assets.logo_dark_rect_mime),
			logo_light_square_mime = COALESCE($12, rebranding_assets.logo_light_square_mime),
			logo_dark_square_mime = COALESCE($13, rebranding_assets.logo_dark_square_mime),
			favicon_mime = COALESCE($14, rebranding_assets.favicon_mime),
			background_image_mime = COALESCE($15, rebranding_assets.background_image_mime),
			updated_at = NOW()
	`

	_, err = database.DB.Exec(query, orgID, productID, productName,
		nilIfEmpty(fields["logo_light_rect"]), nilIfEmpty(fields["logo_dark_rect"]),
		nilIfEmpty(fields["logo_light_square"]), nilIfEmpty(fields["logo_dark_square"]),
		nilIfEmpty(fields["favicon"]), nilIfEmpty(fields["background_image"]),
		nilIfEmptyStr(mimeTypes["logo_light_rect"]), nilIfEmptyStr(mimeTypes["logo_dark_rect"]),
		nilIfEmptyStr(mimeTypes["logo_light_square"]), nilIfEmptyStr(mimeTypes["logo_dark_square"]),
		nilIfEmptyStr(mimeTypes["favicon"]), nilIfEmptyStr(mimeTypes["background_image"]),
	)
	if err != nil {
		return fmt.Errorf("failed to upsert rebranding assets: %w", err)
	}
	return nil
}

// DeleteProductAssets deletes all rebranding assets for an organization+product
func (s *RebrandingService) DeleteProductAssets(orgID, productID string) error {
	query := `DELETE FROM rebranding_assets WHERE organization_id = $1 AND product_id = $2`
	result, err := database.DB.Exec(query, orgID, productID)
	if err != nil {
		return fmt.Errorf("failed to delete rebranding assets: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no rebranding assets found for organization %s product %s", orgID, productID)
	}
	return nil
}

// DeleteSingleAsset removes a single asset field for an organization+product
func (s *RebrandingService) DeleteSingleAsset(orgID, productID, assetName string) error {
	validAssets := map[string]bool{
		"logo_light_rect":   true,
		"logo_dark_rect":    true,
		"logo_light_square": true,
		"logo_dark_square":  true,
		"favicon":           true,
		"background_image":  true,
	}
	if !validAssets[assetName] {
		return fmt.Errorf("invalid asset name: %s", assetName)
	}

	mimeField := assetName + "_mime"
	query := fmt.Sprintf(
		`UPDATE rebranding_assets SET %s = NULL, %s = NULL, updated_at = NOW() WHERE organization_id = $1 AND product_id = $2`,
		assetName, mimeField,
	)
	result, err := database.DB.Exec(query, orgID, productID)
	if err != nil {
		return fmt.Errorf("failed to delete asset %s: %w", assetName, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no rebranding assets found for organization %s product %s", orgID, productID)
	}
	return nil
}

// GetAssetBinary retrieves a single asset binary and mime type
func (s *RebrandingService) GetAssetBinary(orgID, productID, assetName string) ([]byte, string, error) {
	validAssets := map[string]bool{
		"logo_light_rect":   true,
		"logo_dark_rect":    true,
		"logo_light_square": true,
		"logo_dark_square":  true,
		"favicon":           true,
		"background_image":  true,
	}
	if !validAssets[assetName] {
		return nil, "", fmt.Errorf("invalid asset name: %s", assetName)
	}

	mimeField := assetName + "_mime"
	query := fmt.Sprintf(
		`SELECT %s, %s FROM rebranding_assets WHERE organization_id = $1 AND product_id = $2`,
		assetName, mimeField,
	)

	var data []byte
	var mime sql.NullString
	err := database.DB.QueryRow(query, orgID, productID).Scan(&data, &mime)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", fmt.Errorf("asset not found")
		}
		return nil, "", fmt.Errorf("failed to get asset: %w", err)
	}

	if data == nil {
		return nil, "", fmt.Errorf("asset not found")
	}

	mimeType := "application/octet-stream"
	if mime.Valid {
		mimeType = mime.String
	}

	return data, mimeType, nil
}

// GetSystemRebranding returns rebranding data for a system, resolving hierarchy inheritance
func (s *RebrandingService) GetSystemRebranding(systemID string) (*models.SystemRebrandingResponse, error) {
	// Get system's organization_id
	var orgID string
	err := database.DB.QueryRow(`SELECT organization_id FROM systems WHERE id = $1 AND deleted_at IS NULL`, systemID).Scan(&orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get system organization: %w", err)
	}

	// Try to resolve rebranding through hierarchy
	resolvedOrgID, inheritedFrom, err := s.resolveRebrandingOrg(orgID)
	if err != nil {
		return nil, err
	}

	if resolvedOrgID == "" {
		return &models.SystemRebrandingResponse{
			Enabled:      false,
			System:       []models.SystemRebrandingProduct{},
			Applications: []models.SystemRebrandingProduct{},
		}, nil
	}

	// Get all assets for the resolved org
	query := `
		SELECT ra.product_id, ra.product_name, rp.type,
			ra.logo_light_rect IS NOT NULL AS has_logo_light_rect,
			ra.logo_dark_rect IS NOT NULL AS has_logo_dark_rect,
			ra.logo_light_square IS NOT NULL AS has_logo_light_square,
			ra.logo_dark_square IS NOT NULL AS has_logo_dark_square,
			ra.favicon IS NOT NULL AS has_favicon,
			ra.background_image IS NOT NULL AS has_background_image
		FROM rebranding_assets ra
		JOIN rebrandable_products rp ON rp.id = ra.product_id
		WHERE ra.organization_id = $1
	`
	rows, err := database.DB.Query(query, resolvedOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query rebranding assets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var systemProducts []models.SystemRebrandingProduct
	var appProducts []models.SystemRebrandingProduct

	for rows.Next() {
		var productID string
		var productName *string
		var productType string
		var hasLLR, hasLDR, hasLLS, hasLDS, hasFav, hasBG bool

		if err := rows.Scan(&productID, &productName, &productType, &hasLLR, &hasLDR, &hasLLS, &hasLDS, &hasFav, &hasBG); err != nil {
			return nil, fmt.Errorf("failed to scan rebranding asset: %w", err)
		}

		assets := make(map[string]string)
		if hasLLR {
			assets["logo_light_rect"] = fmt.Sprintf("/api/systems/rebranding/%s/logo_light_rect", productID)
		}
		if hasLDR {
			assets["logo_dark_rect"] = fmt.Sprintf("/api/systems/rebranding/%s/logo_dark_rect", productID)
		}
		if hasLLS {
			assets["logo_light_square"] = fmt.Sprintf("/api/systems/rebranding/%s/logo_light_square", productID)
		}
		if hasLDS {
			assets["logo_dark_square"] = fmt.Sprintf("/api/systems/rebranding/%s/logo_dark_square", productID)
		}
		if hasFav {
			assets["favicon"] = fmt.Sprintf("/api/systems/rebranding/%s/favicon", productID)
		}
		if hasBG {
			assets["background_image"] = fmt.Sprintf("/api/systems/rebranding/%s/background_image", productID)
		}

		// Only include products that have at least one asset or a product name
		if len(assets) == 0 && productName == nil {
			continue
		}

		product := models.SystemRebrandingProduct{
			ProductID:   productID,
			ProductName: productName,
			Assets:      assets,
		}

		if productType == "system" {
			systemProducts = append(systemProducts, product)
		} else {
			appProducts = append(appProducts, product)
		}
	}

	if systemProducts == nil {
		systemProducts = []models.SystemRebrandingProduct{}
	}
	if appProducts == nil {
		appProducts = []models.SystemRebrandingProduct{}
	}

	return &models.SystemRebrandingResponse{
		Enabled:       true,
		InheritedFrom: inheritedFrom,
		System:        systemProducts,
		Applications:  appProducts,
	}, nil
}

// GetSystemAssetBinary retrieves a rebranding asset for a system, resolving hierarchy
func (s *RebrandingService) GetSystemAssetBinary(systemID, productID, assetName string) ([]byte, string, error) {
	// Get system's organization_id
	var orgID string
	err := database.DB.QueryRow(`SELECT organization_id FROM systems WHERE id = $1 AND deleted_at IS NULL`, systemID).Scan(&orgID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get system organization: %w", err)
	}

	// Resolve through hierarchy
	resolvedOrgID, _, err := s.resolveRebrandingOrg(orgID)
	if err != nil {
		return nil, "", err
	}
	if resolvedOrgID == "" {
		return nil, "", fmt.Errorf("rebranding not enabled for this system's organization")
	}

	return s.GetAssetBinary(resolvedOrgID, productID, assetName)
}

// ResolveRebranding checks if rebranding is active for an organization (directly or inherited)
// and returns the organization ID that provides the rebranding assets
func (s *RebrandingService) ResolveRebranding(orgID string) (bool, string, error) {
	resolvedOrgID, _, err := s.resolveRebrandingOrg(orgID)
	if err != nil {
		return false, "", err
	}
	if resolvedOrgID == "" {
		return false, "", nil
	}
	return true, resolvedOrgID, nil
}

// resolveRebrandingOrg walks up the hierarchy to find the first org with rebranding enabled
// Returns (resolved_org_id, inherited_from_label, error)
func (s *RebrandingService) resolveRebrandingOrg(orgID string) (string, *string, error) {
	// Check if this org has rebranding enabled
	enabled, err := s.IsRebrandingEnabled(orgID)
	if err != nil {
		return "", nil, err
	}
	if enabled {
		return orgID, nil, nil // Own rebranding, no inheritance
	}

	// Determine org type and walk up hierarchy
	// Check if org is a customer
	var parentOrgID string
	err = database.DB.QueryRow(
		`SELECT custom_data->>'createdBy' FROM customers WHERE logto_id = $1 AND deleted_at IS NULL`, orgID,
	).Scan(&parentOrgID)
	if err == nil && parentOrgID != "" {
		// It's a customer - check parent (reseller or distributor)
		enabled, err = s.IsRebrandingEnabled(parentOrgID)
		if err != nil {
			return "", nil, err
		}
		if enabled {
			label := s.getOrgLabel(parentOrgID)
			return parentOrgID, &label, nil
		}

		// Check if parent is a reseller - go up to its distributor
		var grandparentOrgID string
		err = database.DB.QueryRow(
			`SELECT custom_data->>'createdBy' FROM resellers WHERE logto_id = $1 AND deleted_at IS NULL`, parentOrgID,
		).Scan(&grandparentOrgID)
		if err == nil && grandparentOrgID != "" {
			enabled, err = s.IsRebrandingEnabled(grandparentOrgID)
			if err != nil {
				return "", nil, err
			}
			if enabled {
				label := s.getOrgLabel(grandparentOrgID)
				return grandparentOrgID, &label, nil
			}
		}
		return "", nil, nil
	}

	// Check if org is a reseller
	err = database.DB.QueryRow(
		`SELECT custom_data->>'createdBy' FROM resellers WHERE logto_id = $1 AND deleted_at IS NULL`, orgID,
	).Scan(&parentOrgID)
	if err == nil && parentOrgID != "" {
		// It's a reseller - check parent distributor
		enabled, err = s.IsRebrandingEnabled(parentOrgID)
		if err != nil {
			return "", nil, err
		}
		if enabled {
			label := s.getOrgLabel(parentOrgID)
			return parentOrgID, &label, nil
		}
	}

	return "", nil, nil
}

// getOrgLabel returns a label like "distributor:org_id" for the inherited_from field
func (s *RebrandingService) getOrgLabel(orgID string) string {
	// Check type
	var count int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM distributors WHERE logto_id = $1 AND deleted_at IS NULL`, orgID).Scan(&count)
	if err == nil && count > 0 {
		return "distributor:" + orgID
	}
	err = database.DB.QueryRow(`SELECT COUNT(*) FROM resellers WHERE logto_id = $1 AND deleted_at IS NULL`, orgID).Scan(&count)
	if err == nil && count > 0 {
		return "reseller:" + orgID
	}
	return "unknown:" + orgID
}

// helper functions
func nilIfEmpty(data []byte) interface{} {
	if len(data) == 0 {
		return nil
	}
	return data
}

func nilIfEmptyStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
