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
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
)

// GetFilterProducts returns the list of unique product types for filtering
// Respects RBAC hierarchy - users only see products from systems they can access
func GetFilterProducts(c *gin.Context) {
	// Get current user context for hierarchical filtering
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Build query with RBAC filtering
	query := `
		SELECT DISTINCT type
		FROM systems
		WHERE deleted_at IS NULL
			AND type IS NOT NULL
			AND type != ''
	`

	// Apply RBAC filtering based on user role
	var args []interface{}
	query, args, _ = helpers.AppendOrgFilter(query, userOrgRole, userOrgID, "", args, 1)

	query += ` ORDER BY type ASC`

	// Execute query
	var err error
	rows, err := database.DB.Query(query, args...)

	if err != nil {
		logger.Error().
			Str("component", "filters").
			Str("operation", "get_products").
			Err(err).
			Msg("failed to retrieve product filters")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve product filters", nil))
		return
	}
	defer func() {
		_ = rows.Close()
	}()

	// Collect unique products
	var products []string
	for rows.Next() {
		var product string
		if err := rows.Scan(&product); err != nil {
			logger.Error().
				Str("component", "filters").
				Str("operation", "scan_products").
				Err(err).
				Msg("failed to scan product")
			continue
		}
		products = append(products, product)
	}

	result := map[string]interface{}{
		"products": products,
	}

	logger.Info().
		Str("component", "filters").
		Str("operation", "products_filters").
		Str("user_org_id", userOrgID).
		Str("user_org_role", userOrgRole).
		Int("count", len(products)).
		Msg("product filters retrieved")

	c.JSON(http.StatusOK, response.OK("product filters retrieved successfully", result))
}

// GetFilterCreatedBy returns the list of users who created systems for filtering
// Respects RBAC hierarchy - users only see creators from systems they can access
func GetFilterCreatedBy(c *gin.Context) {
	// Get current user context for hierarchical filtering
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Build query with RBAC filtering
	query := `
		SELECT DISTINCT
			created_by->>'user_id' as user_id,
			created_by->>'name' as name,
			created_by->>'email' as email
		FROM systems
		WHERE deleted_at IS NULL
			AND created_by IS NOT NULL
			AND created_by->>'user_id' IS NOT NULL
			AND created_by->>'name' IS NOT NULL
			AND created_by->>'user_id' != ''
			AND created_by->>'name' != ''
	`

	// Apply RBAC filtering based on user role
	var args []interface{}
	query, args, _ = helpers.AppendOrgFilter(query, userOrgRole, userOrgID, "", args, 1)

	query += ` ORDER BY name ASC`

	// Execute query
	var err error
	rows, err := database.DB.Query(query, args...)

	if err != nil {
		logger.Error().
			Str("component", "filters").
			Str("operation", "get_created_by").
			Err(err).
			Msg("failed to retrieve created-by filters")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve created-by filters", nil))
		return
	}
	defer func() {
		_ = rows.Close()
	}()

	// Collect unique creators
	type Creator struct {
		UserID string `json:"user_id"`
		Name   string `json:"name"`
		Email  string `json:"email"`
	}
	var creators []Creator

	for rows.Next() {
		var userID, name, email *string
		if err := rows.Scan(&userID, &name, &email); err != nil {
			logger.Error().
				Str("component", "filters").
				Str("operation", "scan_created_by").
				Err(err).
				Msg("failed to scan creator")
			continue
		}

		// Skip if required fields are NULL
		if userID != nil && name != nil {
			emailValue := ""
			if email != nil {
				emailValue = *email
			}
			creators = append(creators, Creator{
				UserID: *userID,
				Name:   *name,
				Email:  emailValue,
			})
		}
	}

	result := map[string]interface{}{
		"created_by": creators,
	}

	logger.Info().
		Str("component", "filters").
		Str("operation", "created_by_filters").
		Str("user_org_id", userOrgID).
		Str("user_org_role", userOrgRole).
		Int("count", len(creators)).
		Msg("created-by filters retrieved")

	c.JSON(http.StatusOK, response.OK("created-by filters retrieved successfully", result))
}

// GetFilterVersions returns system versions grouped by product type for filtering
// Respects RBAC hierarchy - users only see versions from systems they can access
func GetFilterVersions(c *gin.Context) {
	// Get current user context for hierarchical filtering
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Build query with RBAC filtering - get type and version together
	query := `
		SELECT DISTINCT type, version
		FROM systems
		WHERE deleted_at IS NULL
			AND type IS NOT NULL
			AND type != ''
			AND version IS NOT NULL
			AND version != ''
	`

	// Apply RBAC filtering based on user role
	var args []interface{}
	query, args, _ = helpers.AppendOrgFilter(query, userOrgRole, userOrgID, "", args, 1)

	query += ` ORDER BY type ASC, version DESC`

	// Execute query
	var err error
	rows, err := database.DB.Query(query, args...)

	if err != nil {
		logger.Error().
			Str("component", "filters").
			Str("operation", "get_versions").
			Err(err).
			Msg("failed to retrieve version filters")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve version filters", nil))
		return
	}
	defer func() {
		_ = rows.Close()
	}()

	// Collect versions grouped by product type
	type ProductVersions struct {
		Product  string   `json:"product"`
		Versions []string `json:"versions"`
	}

	// Use a map to group versions by product
	versionsByProduct := make(map[string][]string)

	for rows.Next() {
		var productType, version string
		if err := rows.Scan(&productType, &version); err != nil {
			logger.Error().
				Str("component", "filters").
				Str("operation", "scan_versions").
				Err(err).
				Msg("failed to scan version")
			continue
		}

		// Create prefixed version string (product:version) to avoid ambiguity
		// when the same version number exists for multiple products
		prefixedVersion := fmt.Sprintf("%s:%s", productType, version)

		// Group prefixed versions by product type
		versionsByProduct[productType] = append(versionsByProduct[productType], prefixedVersion)
	}

	// Convert map to array of ProductVersions
	var groupedVersions []ProductVersions
	for product, versions := range versionsByProduct {
		groupedVersions = append(groupedVersions, ProductVersions{
			Product:  product,
			Versions: versions,
		})
	}

	result := map[string]interface{}{
		"versions": groupedVersions,
	}

	logger.Info().
		Str("component", "filters").
		Str("operation", "versions_filters").
		Str("user_org_id", userOrgID).
		Str("user_org_role", userOrgRole).
		Int("product_count", len(groupedVersions)).
		Msg("version filters retrieved")

	c.JSON(http.StatusOK, response.OK("version filters retrieved successfully", result))
}

// GetFilterOrganizations returns the list of organizations with systems for filtering
// Respects RBAC hierarchy - users only see organizations they can access
func GetFilterOrganizations(c *gin.Context) {
	// Get current user context for hierarchical filtering
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Build query with RBAC filtering
	// The database schema uses separate tables: distributors, resellers, customers
	// We need to UNION them to get all organizations
	// Note: systems.organization_id references the UUID id field, not logto_id
	baseQuery := `
		WITH all_organizations AS (
			SELECT id, logto_id, name FROM distributors WHERE deleted_at IS NULL
			UNION
			SELECT id, logto_id, name FROM resellers WHERE deleted_at IS NULL
			UNION
			SELECT id, logto_id, name FROM customers WHERE deleted_at IS NULL
		)
		SELECT DISTINCT
			o.logto_id AS id,
			o.name
		FROM systems s
		INNER JOIN all_organizations o ON s.organization_id = o.logto_id
		WHERE s.deleted_at IS NULL
	`

	// Apply RBAC filtering based on user role
	query := baseQuery
	var args []interface{}
	query, args, _ = helpers.AppendOrgFilter(query, userOrgRole, userOrgID, "s.", args, 1)

	query += ` ORDER BY o.name ASC`

	// Execute query
	var err error
	rows, err := database.DB.Query(query, args...)

	if err != nil {
		logger.Error().
			Str("component", "filters").
			Str("operation", "get_organizations").
			Err(err).
			Msg("failed to retrieve organization filters")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve organization filters", nil))
		return
	}
	defer func() {
		_ = rows.Close()
	}()

	// Collect unique organizations
	type Organization struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	var organizations []Organization

	for rows.Next() {
		var org Organization
		if err := rows.Scan(&org.ID, &org.Name); err != nil {
			logger.Error().
				Str("component", "filters").
				Str("operation", "scan_organizations").
				Err(err).
				Msg("failed to scan organization")
			continue
		}
		organizations = append(organizations, org)
	}

	result := map[string]interface{}{
		"organizations": organizations,
	}

	logger.Info().
		Str("component", "filters").
		Str("operation", "organizations_filters").
		Str("user_org_id", userOrgID).
		Str("user_org_role", userOrgRole).
		Int("count", len(organizations)).
		Msg("organization filters retrieved")

	c.JSON(http.StatusOK, response.OK("organization filters retrieved successfully", result))
}

// GetFilterUsersOrganizations returns the list of organizations for filtering users
// Respects RBAC hierarchy - users only see organizations they can access
// Unlike GetFilterOrganizations (for systems), this returns all accessible organizations
// regardless of whether they have systems associated
func GetFilterUsersOrganizations(c *gin.Context) {
	// Get current user context for hierarchical filtering
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Build query with RBAC filtering
	// Returns all organizations the user can see based on their role in the hierarchy
	userOrgRoleLower := strings.ToLower(userOrgRole)

	var query string
	var args []interface{}

	switch userOrgRoleLower {
	case "owner":
		// Owner sees all organizations
		query = `
			SELECT logto_id AS id, name, 'distributor' AS type FROM distributors WHERE deleted_at IS NULL
			UNION
			SELECT logto_id AS id, name, 'reseller' AS type FROM resellers WHERE deleted_at IS NULL
			UNION
			SELECT logto_id AS id, name, 'customer' AS type FROM customers WHERE deleted_at IS NULL
			ORDER BY name ASC
		`
	case "distributor":
		// Distributor sees their org + resellers + customers
		query = `
			SELECT logto_id AS id, name, 'distributor' AS type FROM distributors WHERE deleted_at IS NULL AND logto_id = $1
			UNION
			SELECT logto_id AS id, name, 'reseller' AS type FROM resellers WHERE deleted_at IS NULL
			UNION
			SELECT logto_id AS id, name, 'customer' AS type FROM customers WHERE deleted_at IS NULL
			ORDER BY name ASC
		`
		args = append(args, userOrgID)
	case "reseller":
		// Reseller sees their org + customers
		query = `
			SELECT logto_id AS id, name, 'reseller' AS type FROM resellers WHERE deleted_at IS NULL AND logto_id = $1
			UNION
			SELECT logto_id AS id, name, 'customer' AS type FROM customers WHERE deleted_at IS NULL
			ORDER BY name ASC
		`
		args = append(args, userOrgID)
	default:
		// Customer or unknown role - only their organization
		query = `
			SELECT logto_id AS id, name, 'customer' AS type FROM customers WHERE deleted_at IS NULL AND logto_id = $1
			ORDER BY name ASC
		`
		args = append(args, userOrgID)
	}

	// Execute query
	var rows *sql.Rows
	var err error

	if len(args) > 0 {
		rows, err = database.DB.Query(query, args...)
	} else {
		rows, err = database.DB.Query(query)
	}

	if err != nil {
		logger.Error().
			Str("component", "filters").
			Str("operation", "get_users_organizations").
			Err(err).
			Msg("failed to retrieve users organization filters")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve organization filters", nil))
		return
	}
	defer func() {
		_ = rows.Close()
	}()

	// Collect organizations
	type Organization struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	}
	var organizations []Organization

	for rows.Next() {
		var org Organization
		if err := rows.Scan(&org.ID, &org.Name, &org.Type); err != nil {
			logger.Error().
				Str("component", "filters").
				Str("operation", "scan_users_organizations").
				Err(err).
				Msg("failed to scan organization")
			continue
		}
		organizations = append(organizations, org)
	}

	result := map[string]interface{}{
		"organizations": organizations,
	}

	logger.Info().
		Str("component", "filters").
		Str("operation", "users_organizations_filters").
		Str("user_org_id", userOrgID).
		Str("user_org_role", userOrgRole).
		Int("count", len(organizations)).
		Msg("users organization filters retrieved")

	c.JSON(http.StatusOK, response.OK("organization filters retrieved successfully", result))
}
