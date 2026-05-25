/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
)

// GetSystemFilters handles GET /api/filters/systems - aggregated filters endpoint
// Returns products, created_by, and versions in a single response.
// Single auth check, parallel data fetching.
//
// Organizations dropdown is populated by /api/organizations, which supports
// search + pagination and scales to tenants with thousands of rows.
func GetSystemFilters(c *gin.Context) {
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	type Creator struct {
		UserID string `json:"user_id"`
		Name   string `json:"name"`
		Email  string `json:"email"`
	}

	type ProductVersions struct {
		Product  string   `json:"product"`
		Versions []string `json:"versions"`
	}

	var (
		products []string
		creators []Creator
		versions []ProductVersions

		errProducts, errCreators, errVersions error
		wg                                    sync.WaitGroup
	)

	wg.Add(3)

	// Products
	go func() {
		defer wg.Done()

		query := `
			SELECT DISTINCT type
			FROM systems
			WHERE deleted_at IS NULL
				AND type IS NOT NULL
				AND type != ''
		`
		var args []interface{}
		query, args, _ = helpers.AppendOrgFilter(query, userOrgRole, userOrgID, "", args, 1)
		query += ` ORDER BY type ASC`

		rows, err := database.DB.Query(query, args...)
		if err != nil {
			errProducts = fmt.Errorf("failed to retrieve product filters: %w", err)
			return
		}
		defer func() { _ = rows.Close() }()

		products = make([]string, 0)
		for rows.Next() {
			var product string
			if err := rows.Scan(&product); err != nil {
				continue
			}
			products = append(products, product)
		}
	}()

	// Created-by
	go func() {
		defer wg.Done()

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
		var args []interface{}
		query, args, _ = helpers.AppendOrgFilter(query, userOrgRole, userOrgID, "", args, 1)
		query += ` ORDER BY name ASC`

		rows, err := database.DB.Query(query, args...)
		if err != nil {
			errCreators = fmt.Errorf("failed to retrieve created-by filters: %w", err)
			return
		}
		defer func() { _ = rows.Close() }()

		creators = make([]Creator, 0)
		for rows.Next() {
			var uid, name, email *string
			if err := rows.Scan(&uid, &name, &email); err != nil {
				continue
			}
			if uid != nil && name != nil {
				emailValue := ""
				if email != nil {
					emailValue = *email
				}
				creators = append(creators, Creator{
					UserID: *uid,
					Name:   *name,
					Email:  emailValue,
				})
			}
		}
	}()

	// Versions
	go func() {
		defer wg.Done()

		query := `
			SELECT DISTINCT type, version
			FROM systems
			WHERE deleted_at IS NULL
				AND type IS NOT NULL
				AND type != ''
				AND version IS NOT NULL
				AND version != ''
		`
		var args []interface{}
		query, args, _ = helpers.AppendOrgFilter(query, userOrgRole, userOrgID, "", args, 1)
		query += ` ORDER BY type ASC, version DESC`

		rows, err := database.DB.Query(query, args...)
		if err != nil {
			errVersions = fmt.Errorf("failed to retrieve version filters: %w", err)
			return
		}
		defer func() { _ = rows.Close() }()

		versionsByProduct := make(map[string][]string)
		for rows.Next() {
			var productType, version string
			if err := rows.Scan(&productType, &version); err != nil {
				continue
			}
			prefixedVersion := fmt.Sprintf("%s:%s", productType, version)
			versionsByProduct[productType] = append(versionsByProduct[productType], prefixedVersion)
		}

		versions = make([]ProductVersions, 0)
		for product, vers := range versionsByProduct {
			versions = append(versions, ProductVersions{
				Product:  product,
				Versions: vers,
			})
		}
	}()

	wg.Wait()

	for _, e := range []error{errProducts, errCreators, errVersions} {
		if e != nil {
			logger.Error().Err(e).Str("user_id", userID).Msg("Failed in system filters")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve system filters", nil))
			return
		}
	}

	c.JSON(http.StatusOK, response.OK("system filters retrieved successfully", gin.H{
		"products":   products,
		"created_by": creators,
		"versions":   versions,
	}))
}
