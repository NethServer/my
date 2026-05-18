/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
)

// GetAlertFilters handles GET /api/filters/alerts - aggregated filters endpoint
// for the alerts views. Returns the systems, alert names, severities and
// organizations that actually appear in alert_history within the caller's
// scope, so the UI dropdowns only offer values that yield results.
//
// Scope follows the same rules as the other alerts endpoints (resolveOrgScope):
// organization_id omitted = caller's full hierarchy; one/more organization_id =
// those tenants (validated, Owner exempt); customer pinned to own org.
// Single auth + scope resolution, parallel data fetching.
func GetAlertFilters(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	orgIDs, ok := resolveOrgScope(c, user)
	if !ok {
		return
	}

	type SystemFilter struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
		Key  string `json:"key"`
	}
	type OrganizationFilter struct {
		LogtoID string `json:"logto_id"`
		Name    string `json:"name"`
		Type    string `json:"type"`
	}

	// `alerts` is a static catalog: every alert a system can raise, regardless
	// of what has been received so far. It is NOT scoped to the caller's data.
	alerts := AlertCatalog()

	// Empty scope → no data-driven rows anywhere; still return the static
	// alert catalog so the dropdown is usable.
	if len(orgIDs) == 0 {
		c.JSON(http.StatusOK, response.OK("alert filters retrieved successfully", gin.H{
			"systems":       []SystemFilter{},
			"alerts":        alerts,
			"severities":    []string{},
			"organizations": []OrganizationFilter{},
		}))
		return
	}

	// Shared IN (...) placeholder list + args for organization_id scoping.
	placeholders := make([]string, len(orgIDs))
	args := make([]interface{}, len(orgIDs))
	for i, id := range orgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	inClause := strings.Join(placeholders, ",")

	var (
		systems       []SystemFilter
		severities    []string
		organizations []OrganizationFilter

		errSystems, errSeverities, errOrgs error
		wg                                 sync.WaitGroup
	)

	wg.Add(3)

	// Systems with at least one alert in scope.
	go func() {
		defer wg.Done()

		query := fmt.Sprintf(`
			SELECT DISTINCT s.id, s.name, COALESCE(s.type, '') AS type, s.system_key
			FROM alert_history ah
			INNER JOIN systems s ON s.system_key = ah.system_key
			WHERE s.deleted_at IS NULL
				AND ah.organization_id IN (%s)
			ORDER BY s.name ASC
		`, inClause)

		rows, err := database.DB.Query(query, args...)
		if err != nil {
			errSystems = fmt.Errorf("failed to retrieve system filters: %w", err)
			return
		}
		defer func() { _ = rows.Close() }()

		systems = make([]SystemFilter, 0)
		for rows.Next() {
			var s SystemFilter
			if err := rows.Scan(&s.ID, &s.Name, &s.Type, &s.Key); err != nil {
				continue
			}
			systems = append(systems, s)
		}
	}()

	// Distinct severities.
	go func() {
		defer wg.Done()

		query := fmt.Sprintf(`
			SELECT DISTINCT severity
			FROM alert_history
			WHERE severity IS NOT NULL
				AND severity != ''
				AND organization_id IN (%s)
			ORDER BY severity ASC
		`, inClause)

		rows, err := database.DB.Query(query, args...)
		if err != nil {
			errSeverities = fmt.Errorf("failed to retrieve severity filters: %w", err)
			return
		}
		defer func() { _ = rows.Close() }()

		severities = make([]string, 0)
		for rows.Next() {
			var sev string
			if err := rows.Scan(&sev); err != nil {
				continue
			}
			severities = append(severities, sev)
		}
	}()

	// Organizations with at least one alert in scope.
	go func() {
		defer wg.Done()

		query := fmt.Sprintf(`
			SELECT DISTINCT uo.logto_id, uo.name, uo.org_type
			FROM alert_history ah
			INNER JOIN unified_organizations uo ON uo.logto_id = ah.organization_id
			WHERE ah.organization_id IN (%s)
			ORDER BY uo.name ASC
		`, inClause)

		rows, err := database.DB.Query(query, args...)
		if err != nil {
			errOrgs = fmt.Errorf("failed to retrieve organization filters: %w", err)
			return
		}
		defer func() { _ = rows.Close() }()

		organizations = make([]OrganizationFilter, 0)
		for rows.Next() {
			var o OrganizationFilter
			if err := rows.Scan(&o.LogtoID, &o.Name, &o.Type); err != nil {
				continue
			}
			organizations = append(organizations, o)
		}
	}()

	wg.Wait()

	for _, e := range []error{errSystems, errSeverities, errOrgs} {
		if e != nil {
			logger.Error().Err(e).Str("user_id", user.ID).Msg("Failed in alert filters")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve alert filters", nil))
			return
		}
	}

	c.JSON(http.StatusOK, response.OK("alert filters retrieved successfully", gin.H{
		"systems":       systems,
		"alerts":        alerts,
		"severities":    severities,
		"organizations": organizations,
	}))
}
