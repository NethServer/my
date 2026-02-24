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
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
)

// GetUserFilters handles GET /api/filters/users - aggregated filters endpoint
// Returns roles and organizations in a single response.
// Single auth check, parallel data fetching.
func GetUserFilters(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	userOrgID := user.OrganizationID
	userOrgRole := user.OrgRole

	type Organization struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	}

	var (
		roles         []models.Role
		organizations []Organization

		errRoles, errOrgs error
		wg                sync.WaitGroup
	)

	wg.Add(2)

	// Roles
	go func() {
		defer wg.Done()
		roles, errRoles = FetchFilteredRoles(user)
	}()

	// Organizations
	go func() {
		defer wg.Done()

		baseQuery := `
			WITH all_organizations AS (
				SELECT logto_id, name, 'distributor' AS type FROM distributors WHERE deleted_at IS NULL
				UNION
				SELECT logto_id, name, 'reseller' AS type FROM resellers WHERE deleted_at IS NULL
				UNION
				SELECT logto_id, name, 'customer' AS type FROM customers WHERE deleted_at IS NULL
			)
			SELECT DISTINCT
				o.logto_id AS id,
				o.name,
				o.type
			FROM users u
			INNER JOIN all_organizations o ON u.organization_id = o.logto_id
			WHERE u.deleted_at IS NULL
		`
		var args []interface{}
		query, args, _ := helpers.AppendOrgFilter(baseQuery, userOrgRole, userOrgID, "u.", args, 1)
		query += ` ORDER BY o.name ASC`

		rows, err := database.DB.Query(query, args...)
		if err != nil {
			errOrgs = fmt.Errorf("failed to retrieve organization filters: %w", err)
			return
		}
		defer func() { _ = rows.Close() }()

		organizations = make([]Organization, 0)
		for rows.Next() {
			var org Organization
			if err := rows.Scan(&org.ID, &org.Name, &org.Type); err != nil {
				continue
			}
			organizations = append(organizations, org)
		}
	}()

	wg.Wait()

	// Check for errors
	for _, e := range []error{errRoles, errOrgs} {
		if e != nil {
			logger.Error().Err(e).Str("user_id", user.ID).Msg("Failed in user filters")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve user filters", nil))
			return
		}
	}

	c.JSON(http.StatusOK, response.OK("user filters retrieved successfully", gin.H{
		"roles":         helpers.EnsureSlice(roles),
		"organizations": organizations,
	}))
}
