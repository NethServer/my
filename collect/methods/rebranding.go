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
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/response"
)

// getSystemOrgID returns the organization_id for a system
func getSystemOrgID(systemID string) (string, error) {
	var orgID string
	err := database.DB.QueryRow(
		`SELECT organization_id FROM systems WHERE id = $1 AND deleted_at IS NULL`, systemID,
	).Scan(&orgID)
	return orgID, err
}

// rebrandingProduct represents a rebranded product for system consumption
type rebrandingProduct struct {
	ProductID   string            `json:"product_id"`
	ProductName *string           `json:"product_name"`
	Assets      map[string]string `json:"assets"`
}

// GetSystemRebranding returns rebranding data for the authenticated system
func GetSystemRebranding(c *gin.Context) {
	systemID, ok := getAuthenticatedSystemID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}

	// Get system's organization_id
	orgID, err := getSystemOrgID(systemID)
	if err != nil {
		logger.Error().Err(err).Str("system_id", systemID).Msg("failed to get system organization")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get system organization", nil))
		return
	}

	// Resolve rebranding through hierarchy
	resolvedOrgID, inheritedFrom, err := resolveRebrandingOrg(orgID)
	if err != nil {
		logger.Error().Err(err).Str("organization_id", orgID).Msg("failed to resolve rebranding")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve rebranding", nil))
		return
	}

	if resolvedOrgID == "" {
		c.JSON(http.StatusOK, response.OK("rebranding retrieved successfully", gin.H{
			"enabled":      false,
			"system":       []rebrandingProduct{},
			"applications": []rebrandingProduct{},
		}))
		return
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
		logger.Error().Err(err).Msg("failed to query rebranding assets")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to query rebranding assets", nil))
		return
	}
	defer func() { _ = rows.Close() }()

	systemProducts := []rebrandingProduct{}
	appProducts := []rebrandingProduct{}

	for rows.Next() {
		var productID string
		var productName *string
		var productType string
		var hasLLR, hasLDR, hasLLS, hasLDS, hasFav, hasBG bool

		if err := rows.Scan(&productID, &productName, &productType, &hasLLR, &hasLDR, &hasLLS, &hasLDS, &hasFav, &hasBG); err != nil {
			logger.Error().Err(err).Msg("failed to scan rebranding asset")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to scan rebranding asset", nil))
			return
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

		if len(assets) == 0 && productName == nil {
			continue
		}

		product := rebrandingProduct{
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

	c.JSON(http.StatusOK, response.OK("rebranding retrieved successfully", gin.H{
		"enabled":        true,
		"inherited_from": inheritedFrom,
		"system":         systemProducts,
		"applications":   appProducts,
	}))
}

// GetSystemRebrandingAsset serves a single rebranding asset binary for the authenticated system
func GetSystemRebrandingAsset(c *gin.Context) {
	systemID, ok := getAuthenticatedSystemID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}

	productID := c.Param("product_id")
	assetName := c.Param("asset")

	if productID == "" || assetName == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("product id and asset name are required", nil))
		return
	}

	// Validate asset name
	validAssets := map[string]bool{
		"logo_light_rect":   true,
		"logo_dark_rect":    true,
		"logo_light_square": true,
		"logo_dark_square":  true,
		"favicon":           true,
		"background_image":  true,
	}
	if !validAssets[assetName] {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid asset name", nil))
		return
	}

	// Get system's organization_id
	orgID, err := getSystemOrgID(systemID)
	if err != nil {
		logger.Error().Err(err).Str("system_id", systemID).Msg("failed to get system organization")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get system organization", nil))
		return
	}

	// Resolve through hierarchy
	resolvedOrgID, _, err := resolveRebrandingOrg(orgID)
	if err != nil || resolvedOrgID == "" {
		c.JSON(http.StatusNotFound, response.NotFound("rebranding not enabled", nil))
		return
	}

	// Get asset binary
	mimeField := assetName + "_mime"
	query := fmt.Sprintf(
		`SELECT %s, %s FROM rebranding_assets WHERE organization_id = $1 AND product_id = $2`,
		assetName, mimeField,
	)

	var data []byte
	var mime sql.NullString
	err = database.DB.QueryRow(query, resolvedOrgID, productID).Scan(&data, &mime)
	if err != nil || data == nil {
		c.JSON(http.StatusNotFound, response.NotFound("asset not found", nil))
		return
	}

	mimeType := "application/octet-stream"
	if mime.Valid {
		mimeType = mime.String
	}

	c.Data(http.StatusOK, mimeType, data)
}

// resolveRebrandingOrg walks up the hierarchy to find the first org with rebranding enabled.
// Uses a recursive CTE to traverse the organization hierarchy in a single DB round-trip.
func resolveRebrandingOrg(orgID string) (string, *string, error) {
	query := `
		WITH RECURSIVE org_hierarchy AS (
			-- Start with the given org
			SELECT $1::text AS org_id, 0 AS depth,
				CASE
					WHEN EXISTS (SELECT 1 FROM customers WHERE logto_id = $1 AND deleted_at IS NULL) THEN 'customer'
					WHEN EXISTS (SELECT 1 FROM resellers WHERE logto_id = $1 AND deleted_at IS NULL) THEN 'reseller'
					WHEN EXISTS (SELECT 1 FROM distributors WHERE logto_id = $1 AND deleted_at IS NULL) THEN 'distributor'
					ELSE 'owner'
				END AS org_type

			UNION ALL

			-- Walk up: customer -> reseller/distributor, reseller -> distributor
			SELECT
				COALESCE(
					(SELECT custom_data->>'createdBy' FROM customers WHERE logto_id = oh.org_id AND deleted_at IS NULL),
					(SELECT custom_data->>'createdBy' FROM resellers WHERE logto_id = oh.org_id AND deleted_at IS NULL)
				) AS org_id,
				oh.depth + 1,
				CASE
					WHEN EXISTS (SELECT 1 FROM resellers WHERE logto_id = COALESCE(
						(SELECT custom_data->>'createdBy' FROM customers WHERE logto_id = oh.org_id AND deleted_at IS NULL),
						(SELECT custom_data->>'createdBy' FROM resellers WHERE logto_id = oh.org_id AND deleted_at IS NULL)
					) AND deleted_at IS NULL) THEN 'reseller'
					WHEN EXISTS (SELECT 1 FROM distributors WHERE logto_id = COALESCE(
						(SELECT custom_data->>'createdBy' FROM customers WHERE logto_id = oh.org_id AND deleted_at IS NULL),
						(SELECT custom_data->>'createdBy' FROM resellers WHERE logto_id = oh.org_id AND deleted_at IS NULL)
					) AND deleted_at IS NULL) THEN 'distributor'
					ELSE 'owner'
				END AS org_type
			FROM org_hierarchy oh
			WHERE oh.depth < 3
				AND COALESCE(
					(SELECT custom_data->>'createdBy' FROM customers WHERE logto_id = oh.org_id AND deleted_at IS NULL),
					(SELECT custom_data->>'createdBy' FROM resellers WHERE logto_id = oh.org_id AND deleted_at IS NULL)
				) IS NOT NULL
		)
		SELECT oh.org_id, oh.depth, oh.org_type
		FROM org_hierarchy oh
		JOIN rebranding_enabled re ON re.organization_id = oh.org_id
		ORDER BY oh.depth ASC
		LIMIT 1
	`

	var resolvedOrgID string
	var depth int
	var orgType string

	err := database.DB.QueryRow(query, orgID).Scan(&resolvedOrgID, &depth, &orgType)
	if err == sql.ErrNoRows {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve rebranding org: %w", err)
	}

	// depth == 0 means the org itself has rebranding (no inheritance)
	if depth == 0 {
		return resolvedOrgID, nil, nil
	}

	label := orgType + ":" + resolvedOrgID
	return resolvedOrgID, &label, nil
}
