/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/alerting"
)

// alertFiltersCacheTTL bounds how stale a cached /filters/alerts response can
// be. The endpoint is dropdown fodder for the alerts views, so a few seconds
// of staleness is acceptable in exchange for cutting Mimir fan-out + DB cost
// to ~zero on repeat hits.
const alertFiltersCacheTTL = 15 * time.Second

type alertFiltersCacheEntry struct {
	payload gin.H
	expires time.Time
}

type alertFiltersInflight struct {
	done    chan struct{}
	payload gin.H
}

var (
	alertFiltersCacheMu sync.Mutex
	alertFiltersCache   = map[string]alertFiltersCacheEntry{}
	alertFiltersFlights = map[string]*alertFiltersInflight{}
)

// alertFiltersCacheKey produces a stable cache key from the orgIDs slice.
func alertFiltersCacheKey(orgIDs []string) string {
	sorted := append([]string(nil), orgIDs...)
	sort.Strings(sorted)
	return strings.Join(sorted, ",")
}

type alertFilterSystem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Key  string `json:"key"`
}

type alertFilterOrganization struct {
	LogtoID string `json:"logto_id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
}

// GetAlertFilters handles GET /api/filters/alerts - aggregated filters endpoint
// for the alerts views. Returns the systems, alert names, severities and
// organizations that have at least one alert in the caller's scope (active in
// Mimir OR present in alert_history), so the UI dropdowns only offer values
// that will yield results.
//
// Scope follows the same rules as the other alerts endpoints (resolveOrgScope):
// organization_id omitted = caller's full hierarchy; one/more organization_id =
// those tenants (validated, Owner exempt); customer pinned to own org.
//
// The response is cached per-scope for alertFiltersCacheTTL with singleflight
// coalescing on concurrent misses, mirroring /api/alerts/totals.
func GetAlertFilters(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	orgIDs, ok := resolveOrgScope(c, user)
	if !ok {
		return
	}

	// `alerts` is a static catalog: every alert a system can raise, regardless
	// of what has been received so far. It is NOT scoped to the caller's data.
	catalog := AlertCatalog()

	// Empty scope → no data-driven rows anywhere; still return the static
	// alert catalog so the dropdown is usable. Not cached — trivial path.
	if len(orgIDs) == 0 {
		c.JSON(http.StatusOK, response.OK("alert filters retrieved successfully", gin.H{
			"systems":       []alertFilterSystem{},
			"alerts":        catalog,
			"severities":    []string{},
			"organizations": []alertFilterOrganization{},
			"warnings":      []string{},
		}))
		return
	}

	cacheKey := alertFiltersCacheKey(orgIDs)

	// Cache hit → serve immediately.
	alertFiltersCacheMu.Lock()
	if entry, ok := alertFiltersCache[cacheKey]; ok && time.Now().Before(entry.expires) {
		alertFiltersCacheMu.Unlock()
		c.JSON(http.StatusOK, response.OK("alert filters retrieved successfully", entry.payload))
		return
	}

	// Coalesce concurrent misses on the same key: only one goroutine computes,
	// the others wait on its result. This protects Mimir + DB from a poll storm.
	if flight, ok := alertFiltersFlights[cacheKey]; ok {
		alertFiltersCacheMu.Unlock()
		<-flight.done
		c.JSON(http.StatusOK, response.OK("alert filters retrieved successfully", flight.payload))
		return
	}
	flight := &alertFiltersInflight{done: make(chan struct{})}
	alertFiltersFlights[cacheKey] = flight
	alertFiltersCacheMu.Unlock()

	payload, err := computeAlertFilters(c.Request.Context(), orgIDs, catalog)
	if err != nil {
		alertFiltersCacheMu.Lock()
		delete(alertFiltersFlights, cacheKey)
		alertFiltersCacheMu.Unlock()
		close(flight.done)
		logger.Error().Err(err).Str("user_id", user.ID).Msg("Failed in alert filters")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve alert filters", nil))
		return
	}

	alertFiltersCacheMu.Lock()
	alertFiltersCache[cacheKey] = alertFiltersCacheEntry{
		payload: payload,
		expires: time.Now().Add(alertFiltersCacheTTL),
	}
	flight.payload = payload
	delete(alertFiltersFlights, cacheKey)
	alertFiltersCacheMu.Unlock()
	close(flight.done)

	c.JSON(http.StatusOK, response.OK("alert filters retrieved successfully", payload))
}

// computeAlertFilters runs the four parallel collectors (history systems,
// history severities, history org IDs, active alerts from Mimir), then merges
// and dedupes. Returns the full payload (catalog included) or the first DB
// error. Mimir per-tenant failures are surfaced as warnings, not errors — the
// rest of the result is still returned.
func computeAlertFilters(parent context.Context, orgIDs []string, catalog []AlertCatalogEntry) (gin.H, error) {
	// Shared IN (...) placeholder list + args for organization_id scoping.
	placeholders := make([]string, len(orgIDs))
	args := make([]interface{}, len(orgIDs))
	for i, id := range orgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	inClause := strings.Join(placeholders, ",")

	var (
		historySystems    []alertFilterSystem
		historySeverities []string
		historyOrgIDs     []string

		activeSystems    map[string]alertFilterSystem
		activeSeverities map[string]struct{}
		activeOrgIDs     map[string]struct{}
		mimirWarnings    []string

		errHistSys, errHistSev, errHistOrg error
		wg                                 sync.WaitGroup
	)

	wg.Add(4)

	// Systems with at least one resolved alert in scope.
	go func() {
		defer wg.Done()

		query := fmt.Sprintf(`
			SELECT DISTINCT s.id, s.name, COALESCE(s.type, '') AS type, s.system_key
			FROM alert_history ah
			INNER JOIN systems s ON s.system_key = ah.system_key
			WHERE s.deleted_at IS NULL
				AND ah.organization_id IN (%s)
		`, inClause)

		rows, err := database.DB.Query(query, args...)
		if err != nil {
			errHistSys = fmt.Errorf("failed to retrieve system filters: %w", err)
			return
		}
		defer func() { _ = rows.Close() }()

		historySystems = make([]alertFilterSystem, 0)
		for rows.Next() {
			var s alertFilterSystem
			if err := rows.Scan(&s.ID, &s.Name, &s.Type, &s.Key); err != nil {
				continue
			}
			historySystems = append(historySystems, s)
		}
	}()

	// Distinct severities from resolved alerts.
	go func() {
		defer wg.Done()

		query := fmt.Sprintf(`
			SELECT DISTINCT severity
			FROM alert_history
			WHERE severity IS NOT NULL
				AND severity != ''
				AND organization_id IN (%s)
		`, inClause)

		rows, err := database.DB.Query(query, args...)
		if err != nil {
			errHistSev = fmt.Errorf("failed to retrieve severity filters: %w", err)
			return
		}
		defer func() { _ = rows.Close() }()

		historySeverities = make([]string, 0)
		for rows.Next() {
			var sev string
			if err := rows.Scan(&sev); err != nil {
				continue
			}
			historySeverities = append(historySeverities, sev)
		}
	}()

	// Organization IDs with at least one resolved alert in scope. Names/types
	// are resolved later in a single unified_organizations lookup that also
	// covers organizations seen only in active alerts.
	go func() {
		defer wg.Done()

		query := fmt.Sprintf(`
			SELECT DISTINCT organization_id
			FROM alert_history
			WHERE organization_id IN (%s)
		`, inClause)

		rows, err := database.DB.Query(query, args...)
		if err != nil {
			errHistOrg = fmt.Errorf("failed to retrieve organization filters: %w", err)
			return
		}
		defer func() { _ = rows.Close() }()

		historyOrgIDs = make([]string, 0)
		for rows.Next() {
			var o string
			if err := rows.Scan(&o); err != nil {
				continue
			}
			historyOrgIDs = append(historyOrgIDs, o)
		}
	}()

	// Active alerts from Mimir. Per-tenant failures collected as warnings;
	// successful tenants still contribute their distinct values.
	go func() {
		defer wg.Done()
		activeSystems, activeSeverities, activeOrgIDs, mimirWarnings = fanOutActiveAlertsForFilters(parent, orgIDs)
	}()

	wg.Wait()

	for _, e := range []error{errHistSys, errHistSev, errHistOrg} {
		if e != nil {
			return nil, e
		}
	}

	// Merge systems by system_key. History rows win on collision (they were
	// joined against the systems table, so name/type reflect the current DB
	// state rather than whatever was stamped on the alert at ingest time).
	sysByKey := make(map[string]alertFilterSystem, len(historySystems)+len(activeSystems))
	for k, s := range activeSystems {
		sysByKey[k] = s
	}
	for _, s := range historySystems {
		sysByKey[s.Key] = s
	}
	systems := make([]alertFilterSystem, 0, len(sysByKey))
	for _, s := range sysByKey {
		systems = append(systems, s)
	}
	sort.Slice(systems, func(i, j int) bool { return systems[i].Name < systems[j].Name })

	// Merge severities.
	sevSet := make(map[string]struct{}, len(historySeverities)+len(activeSeverities))
	for _, s := range historySeverities {
		sevSet[s] = struct{}{}
	}
	for s := range activeSeverities {
		sevSet[s] = struct{}{}
	}
	severities := make([]string, 0, len(sevSet))
	for s := range sevSet {
		severities = append(severities, s)
	}
	sort.Strings(severities)

	// Merge organization IDs and resolve names/types in a single lookup.
	orgSet := make(map[string]struct{}, len(historyOrgIDs)+len(activeOrgIDs))
	for _, o := range historyOrgIDs {
		orgSet[o] = struct{}{}
	}
	for o := range activeOrgIDs {
		orgSet[o] = struct{}{}
	}
	organizations, err := resolveOrganizationFilters(orgSet)
	if err != nil {
		return nil, err
	}

	if mimirWarnings == nil {
		mimirWarnings = []string{}
	}

	return gin.H{
		"systems":       systems,
		"alerts":        catalog,
		"severities":    severities,
		"organizations": organizations,
		"warnings":      mimirWarnings,
	}, nil
}

// resolveOrganizationFilters looks up name/type for the given organization
// logto_ids in a single query against unified_organizations. Returns an empty
// (non-nil) slice when the set is empty.
func resolveOrganizationFilters(orgSet map[string]struct{}) ([]alertFilterOrganization, error) {
	organizations := make([]alertFilterOrganization, 0, len(orgSet))
	if len(orgSet) == 0 {
		return organizations, nil
	}

	ids := make([]string, 0, len(orgSet))
	for id := range orgSet {
		ids = append(ids, id)
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT logto_id, name, org_type
		FROM unified_organizations
		WHERE logto_id IN (%s)
		ORDER BY name ASC
	`, strings.Join(placeholders, ","))

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve organization filters: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var o alertFilterOrganization
		if err := rows.Scan(&o.LogtoID, &o.Name, &o.Type); err != nil {
			continue
		}
		organizations = append(organizations, o)
	}
	return organizations, nil
}

// fanOutActiveAlertsForFilters queries Mimir for active alerts in each tenant
// concurrently, with the same bounded concurrency and global timeout as the
// other alerts fan-outs. The trusted orgID is the per-tenant call target — we
// do NOT trust the `organization_id` label on the alert payload itself.
func fanOutActiveAlertsForFilters(parent context.Context, orgIDs []string) (
	map[string]alertFilterSystem,
	map[string]struct{},
	map[string]struct{},
	[]string,
) {
	systems := make(map[string]alertFilterSystem)
	severities := make(map[string]struct{})
	activeOrgIDs := make(map[string]struct{})
	var warnings []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(parent, alertsTotalsFanoutTimeout)
	defer cancel()
	sem := make(chan struct{}, alertsTotalsFanoutConcurrency)

	for _, orgID := range orgIDs {
		wg.Add(1)
		go func(orgID string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				warnings = append(warnings, fmt.Sprintf("org %s: timed out waiting for slot", orgID))
				mu.Unlock()
				return
			}

			body, err := alerting.GetAlertsCtx(ctx, orgID)
			if err != nil {
				logger.Warn().Err(err).Str("org_id", orgID).Msg("failed to fetch alerts from mimir for filters")
				mu.Lock()
				warnings = append(warnings, fmt.Sprintf("org %s: %s", orgID, err.Error()))
				mu.Unlock()
				return
			}

			var alerts []map[string]interface{}
			if err := json.Unmarshal(body, &alerts); err != nil {
				logger.Warn().Err(err).Str("org_id", orgID).Msg("failed to parse mimir alerts response for filters")
				mu.Lock()
				warnings = append(warnings, fmt.Sprintf("org %s: parse error", orgID))
				mu.Unlock()
				return
			}

			if len(alerts) == 0 {
				return
			}

			// Collect into local maps to keep the shared lock tight.
			localSystems := make(map[string]alertFilterSystem)
			localSeverities := make(map[string]struct{})
			for _, alert := range alerts {
				labels, _ := alert["labels"].(map[string]interface{})

				if sev, _ := labels["severity"].(string); sev != "" {
					localSeverities[sev] = struct{}{}
				}

				sysKey, _ := labels["system_key"].(string)
				if sysKey != "" {
					if _, exists := localSystems[sysKey]; !exists {
						sysID, _ := labels["system_id"].(string)
						sysName, _ := labels["system_name"].(string)
						sysType, _ := labels["system_type"].(string)
						localSystems[sysKey] = alertFilterSystem{
							ID:   sysID,
							Name: sysName,
							Type: sysType,
							Key:  sysKey,
						}
					}
				}
			}

			mu.Lock()
			for k, s := range localSystems {
				if _, exists := systems[k]; !exists {
					systems[k] = s
				}
			}
			for s := range localSeverities {
				severities[s] = struct{}{}
			}
			activeOrgIDs[orgID] = struct{}{}
			mu.Unlock()
		}(orgID)
	}
	wg.Wait()
	return systems, severities, activeOrgIDs, warnings
}
