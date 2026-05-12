/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/alerting"
	"github.com/nethesis/my/backend/services/local"
)

const defaultSystemAlertSilenceDurationMinutes = 60
const defaultSystemAlertSilenceComment = "silenced from my"

// webhookHostDenylist rejects URLs whose host resolves to loopback, link-local,
// cloud metadata service, RFC1918 private ranges, or unspecified addresses.
// See validateWebhookURL.
var webhookHostDenylist = []string{
	"127.", "0.", "::1", "localhost",
	"169.254.", "fe80:", "fe80::",
	"10.", "172.16.", "172.17.", "172.18.", "172.19.",
	"172.20.", "172.21.", "172.22.", "172.23.", "172.24.",
	"172.25.", "172.26.", "172.27.", "172.28.", "172.29.",
	"172.30.", "172.31.", "192.168.",
	"fc00:", "fd00:",
}

// resolveOrgID extracts and validates the target organization ID for alerting operations.
//   - Customer: always pinned to their own organization.
//   - Distributor/Reseller: must pass organization_id, validated via IsOrganizationInHierarchy.
//   - Owner: organization_id is optional; an empty result means "all tenants" and is
//     only meaningful for aggregate endpoints (totals, trend). Endpoints that talk to
//     Mimir per-tenant must reject an empty result themselves.
func resolveOrgID(c *gin.Context, user *models.User) (string, bool) {
	orgID := c.Query("organization_id")
	orgRole := strings.ToLower(user.OrgRole)

	if orgRole == "customer" {
		return user.OrganizationID, true
	}

	if orgRole == "owner" {
		// Owner may omit organization_id to operate across all tenants.
		return orgID, true
	}

	if orgID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("organization_id query parameter is required", nil))
		return "", false
	}

	// Validate hierarchical access to the target organization
	userService := local.NewUserService()
	if !userService.IsOrganizationInHierarchy(orgRole, user.OrganizationID, orgID) {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied: organization not in your hierarchy", nil))
		return "", false
	}

	return orgID, true
}

// requireOrgID rejects callers that omitted organization_id when the endpoint
// cannot operate without one (e.g., Mimir per-tenant queries). Returns true if
// the request is allowed to proceed.
func requireOrgID(c *gin.Context, orgID string) bool {
	if orgID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("organization_id query parameter is required", nil))
		return false
	}
	return true
}

// resolveOrgScope extracts the list of organization IDs the caller is operating on.
// Used by aggregate endpoints (e.g., /totals) where omitting organization_id means
// "aggregate across the caller's full hierarchy" rather than "specific tenant".
//
// Modes:
//
//  1. organization_id omitted      → caller's full hierarchy (incl. self).
//  2. organization_id=X            → single tenant X.
//  3. organization_id=X&organization_id=Y (multi)
//     → union of {X, Y, ...}. Each must be in the
//     caller's hierarchy (Owner exempt).
//  4. + include=descendants        → expand each org_id to itself + its sub-tree
//     (deduplicated). Useful to mix-and-match drill-downs across siblings.
//
// Customer is always pinned to their own organization regardless of params.
//
// Returns false on auth/validation failure (response already written).
func resolveOrgScope(c *gin.Context, user *models.User) ([]string, bool) {
	orgIDsParam := c.QueryArray("organization_id")
	includeDescendants := c.Query("include") == "descendants"
	orgRole := strings.ToLower(user.OrgRole)

	if orgRole == "customer" {
		return []string{user.OrganizationID}, true
	}

	userService := local.NewUserService()

	// No org_id passed → caller's full hierarchy.
	if len(orgIDsParam) == 0 {
		orgIDs, err := userService.GetHierarchicalOrganizationIDs(orgRole, user.OrganizationID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve organization hierarchy: "+err.Error(), nil))
			return nil, false
		}
		return orgIDs, true
	}

	// One or more org_ids: validate each, optionally expand each with descendants,
	// dedupe to keep the fan-out minimal when sub-trees overlap.
	result := make([]string, 0, len(orgIDsParam))
	seen := make(map[string]struct{}, len(orgIDsParam))
	for _, oid := range orgIDsParam {
		if oid == "" {
			continue
		}
		if orgRole != "owner" && !userService.IsOrganizationInHierarchy(orgRole, user.OrganizationID, oid) {
			c.JSON(http.StatusForbidden, response.Forbidden("access denied: organization not in your hierarchy", nil))
			return nil, false
		}
		if includeDescendants {
			targetType := userService.GetOrganizationType(oid)
			expanded, err := userService.GetHierarchicalOrganizationIDs(targetType, oid)
			if err != nil {
				c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve descendants: "+err.Error(), nil))
				return nil, false
			}
			for _, e := range expanded {
				if _, ok := seen[e]; !ok {
					seen[e] = struct{}{}
					result = append(result, e)
				}
			}
		} else {
			if _, ok := seen[oid]; !ok {
				seen[oid] = struct{}{}
				result = append(result, oid)
			}
		}
	}
	return result, true
}

// configPropagationFanoutTimeout caps how long a POST /alerts/config call
// will wait for re-rendering+pushing the effective config across descendant
// tenants. Tuned so a Owner save with hundreds of tenants completes in
// reasonable time without holding the request open too long.
const configPropagationFanoutTimeout = 30 * time.Second

// configPropagationFanoutConcurrency limits simultaneous in-flight Mimir
// pushes to avoid opening hundreds of sockets when an Owner saves.
const configPropagationFanoutConcurrency = 10

// alertLayerMutexes guards per-organization layer save+propagate operations.
// Two parallel POSTs/DELETEs for the same org would otherwise race at the
// Mimir push step: the DB upsert atomically last-write-wins, but the two
// fan-outs run concurrently and the slower-arriving push can land AFTER
// the faster one — leaving Mimir with stale state while the DB holds the
// newer layer. We serialise per-org to make save+propagate a critical
// section. Different orgs are independent (no global lock).
//
// Single-process scope. If/when the backend is deployed multi-instance,
// swap this for a Postgres advisory lock keyed on the same org_id.
var alertLayerMutexes sync.Map // map[string]*sync.Mutex

func acquireOrgLayerLock(orgID string) func() {
	mu, _ := alertLayerMutexes.LoadOrStore(orgID, &sync.Mutex{})
	m := mu.(*sync.Mutex)
	m.Lock()
	return m.Unlock
}

// snapshotLayerForAudit produces a JSON-serialisable, secret-redacted snapshot
// of a layer record suitable for inclusion in audit log details. The unredacted
// layer is on disk in alert_config_layers; the audit log only needs to record
// what changed, not the secrets themselves.
func snapshotLayerForAudit(rec *entities.AlertConfigLayerRecord) map[string]interface{} {
	if rec == nil {
		return nil
	}
	cfg := alerting.RedactLayerForAudit(rec.Config)
	return map[string]interface{}{
		"organization_id":    rec.OrganizationID,
		"config":             cfg,
		"updated_by_user_id": rec.UpdatedByUserID,
		"updated_by_name":    rec.UpdatedByName,
		"updated_at":         rec.UpdatedAt,
	}
}

// snapshotLayerBodyForAudit captures the inbound layer body (post-Normalize)
// before persistence, so the audit "after" reflects what we intend to write.
// Same redaction policy as snapshotLayerForAudit.
func snapshotLayerBodyForAudit(orgID string, layer models.AlertingConfigLayer) map[string]interface{} {
	cfg := alerting.RedactLayerForAudit(layer)
	return map[string]interface{}{
		"organization_id": orgID,
		"config":          cfg,
	}
}

// ConfigureAlerts handles POST /api/alerts/config — writes the CALLER's
// alerting layer (one row per organization in alert_config_layers) and
// propagates the change by re-rendering and re-pushing the effective Mimir
// config for every tenant in the caller's hierarchy.
//
// Per the additive model, descendants can ADD recipients/severity rules but
// cannot disable channels enabled by ancestors: NormalizeLayerForRole strips
// any explicit *bool=&false from non-Owner layers before storage.
//
// Returns a `warnings[]` array listing per-tenant push failures (timeout,
// 5xx, etc.). The caller's layer is saved regardless of push outcome — Mimir
// can be reconciled by saving again.
func ConfigureAlerts(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	var req models.AlertingConfigLayer
	if err := c.ShouldBindJSON(&req); err != nil {
		// MaxBytesReader (registered via middleware.MaxBodySize on the route)
		// surfaces as a "http: request body too large" error here; map to 413
		// so the client can distinguish "too big" from "malformed".
		if strings.Contains(err.Error(), "request body too large") {
			c.JSON(http.StatusRequestEntityTooLarge, response.Error(http.StatusRequestEntityTooLarge, "request body exceeds the configured maximum", nil))
			return
		}
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body: "+err.Error(), nil))
		return
	}

	// Webhook URLs go through DNS-aware validation (denylist resolution,
	// loopback/private-range rejection). The model's stateless Validate
	// covers format/structure for everything else (email format, severity
	// enum, language, format).
	if err := validateWebhookRecipients(req.WebhookRecipients); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error(), nil))
		return
	}

	// Enforce additive-only contract: descendants cannot encode "disable
	// channel" by writing explicit false. Owner is exempt (top of the chain
	// can globally turn a channel off, though OR with descendants may bring
	// it back).
	alerting.NormalizeLayerForRole(&req, user.OrgRole)

	// Serialise save+propagate per-org: prevents two concurrent saves from
	// racing at the Mimir push step, where the slower-arriving fan-out can
	// land AFTER the faster one and leave Mimir with stale state while the
	// DB holds the newer layer.
	releaseLock := acquireOrgLayerLock(user.OrganizationID)
	defer releaseLock()

	// Persist the caller's layer. Capture the previous layer (if any) BEFORE
	// the upsert so the audit log records the actual diff that was applied.
	layerRepo := entities.NewLocalAlertConfigLayersRepository()
	prevLayer, prevErr := layerRepo.Get(user.OrganizationID)
	if prevErr != nil && !errors.Is(prevErr, entities.ErrAlertConfigLayerNotFound) {
		logger.Warn().Err(prevErr).Str("org_id", user.OrganizationID).Msg("failed to read previous layer for audit; continuing")
		prevLayer = nil
	}

	updatedBy := ""
	if user.LogtoID != nil {
		updatedBy = *user.LogtoID
	}
	if _, err := layerRepo.Upsert(user.OrganizationID, req, updatedBy, user.Name); err != nil {
		logger.Error().Err(err).Str("org_id", user.OrganizationID).Msg("failed to save alert config layer")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to save alert config: "+err.Error(), nil))
		return
	}

	// Propagate: the caller's save affects the effective config of all
	// descendants in their hierarchy (including self). Walk the descendant
	// list, fan-out re-render+push to Mimir for each.
	userService := local.NewUserService()
	descendants, err := userService.GetHierarchicalOrganizationIDs(user.OrgRole, user.OrganizationID)
	if err != nil {
		// Layer saved but couldn't enumerate descendants — return success on
		// save with a warning; reconciliation possible by re-saving.
		logger.Warn().Err(err).Str("org_id", user.OrganizationID).Msg("layer saved but hierarchy enumeration failed")
		auditDetails := map[string]interface{}{
			"before": snapshotLayerForAudit(prevLayer),
			"after":  snapshotLayerBodyForAudit(user.OrganizationID, req),
		}
		logger.LogBusinessOperationDetails(c, "alerts", "save_layer", "alert_config_layer", user.OrganizationID, true, nil, auditDetails)
		c.JSON(http.StatusOK, response.OK("alerting layer saved (propagation skipped)", gin.H{
			"warnings": []string{fmt.Sprintf("hierarchy: %s", err.Error())},
		}))
		return
	}

	warnings := propagateAlertingConfigToTenants(c.Request.Context(), descendants)
	auditDetails := map[string]interface{}{
		"before":               snapshotLayerForAudit(prevLayer),
		"after":                snapshotLayerBodyForAudit(user.OrganizationID, req),
		"affected_tenants":     len(descendants),
		"propagation_warnings": len(warnings),
	}
	logger.LogBusinessOperationDetails(c, "alerts", "save_layer", "alert_config_layer", user.OrganizationID, true, nil, auditDetails)
	c.JSON(http.StatusOK, response.OK("alerting configuration updated successfully", gin.H{
		"warnings":         warnings,
		"propagated_to":    len(descendants) - len(warnings),
		"affected_tenants": len(descendants),
	}))
}

// propagateAlertingConfigToTenants re-renders and re-pushes the effective
// Mimir config for each tenant in the list, with bounded concurrency and a
// global timeout. Per-tenant errors are collected as warnings (string
// `org <logto_id>: <error>`) and returned; non-erroring tenants are pushed
// successfully. Always returns a non-nil slice.
func propagateAlertingConfigToTenants(parent context.Context, tenants []string) []string {
	warnings := []string{}
	if len(tenants) == 0 {
		return warnings
	}
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	ctx, cancel := context.WithTimeout(parent, configPropagationFanoutTimeout)
	defer cancel()
	sem := make(chan struct{}, configPropagationFanoutConcurrency)

	for _, tenant := range tenants {
		wg.Add(1)
		go func(tenant string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				warnings = append(warnings, fmt.Sprintf("org %s: timed out waiting for slot", tenant))
				mu.Unlock()
				return
			}
			if err := alerting.RenderAndPushEffective(ctx, tenant); err != nil {
				logger.Warn().Err(err).Str("org_id", tenant).Msg("config propagation failed")
				mu.Lock()
				warnings = append(warnings, fmt.Sprintf("org %s: %s", tenant, err.Error()))
				mu.Unlock()
			}
		}(tenant)
	}
	wg.Wait()
	return warnings
}

// DisableAlerts handles DELETE /api/alerts/config — removes the CALLER's
// alerting layer entirely. The effective config of all descendant tenants
// is re-rendered as the merge of the remaining ancestor layers (so the
// caller's contribution disappears but ancestor recipients/severity rules
// are preserved). To completely silence a tenant's alerting, every layer
// in its chain must drop its contribution; alternatively the Owner can do
// it globally by removing their own layer.
func DisableAlerts(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Same critical section as ConfigureAlerts: serialise per-org so a
	// concurrent save+delete race cannot leave Mimir with stale state.
	releaseLock := acquireOrgLayerLock(user.OrganizationID)
	defer releaseLock()

	layerRepo := entities.NewLocalAlertConfigLayersRepository()

	// Capture pre-delete snapshot for audit so the log records what was
	// removed, not just "delete_layer".
	prevLayer, prevErr := layerRepo.Get(user.OrganizationID)
	if prevErr != nil && !errors.Is(prevErr, entities.ErrAlertConfigLayerNotFound) {
		logger.Warn().Err(prevErr).Str("org_id", user.OrganizationID).Msg("failed to read previous layer for audit; continuing")
		prevLayer = nil
	}

	if err := layerRepo.Delete(user.OrganizationID); err != nil {
		logger.Error().Err(err).Str("org_id", user.OrganizationID).Msg("failed to delete alert config layer")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to delete alert config: "+err.Error(), nil))
		return
	}

	userService := local.NewUserService()
	descendants, err := userService.GetHierarchicalOrganizationIDs(user.OrgRole, user.OrganizationID)
	if err != nil {
		logger.Warn().Err(err).Str("org_id", user.OrganizationID).Msg("layer deleted but hierarchy enumeration failed")
		auditDetails := map[string]interface{}{"before": snapshotLayerForAudit(prevLayer), "after": nil}
		logger.LogBusinessOperationDetails(c, "alerts", "delete_layer", "alert_config_layer", user.OrganizationID, true, nil, auditDetails)
		c.JSON(http.StatusOK, response.OK("alerting layer removed (propagation skipped)", gin.H{
			"warnings": []string{fmt.Sprintf("hierarchy: %s", err.Error())},
		}))
		return
	}

	warnings := propagateAlertingConfigToTenants(c.Request.Context(), descendants)
	auditDetails := map[string]interface{}{
		"before":               snapshotLayerForAudit(prevLayer),
		"after":                nil,
		"affected_tenants":     len(descendants),
		"propagation_warnings": len(warnings),
	}
	logger.LogBusinessOperationDetails(c, "alerts", "delete_layer", "alert_config_layer", user.OrganizationID, true, nil, auditDetails)
	c.JSON(http.StatusOK, response.OK("alerting layer removed successfully", gin.H{
		"warnings":         warnings,
		"propagated_to":    len(descendants) - len(warnings),
		"affected_tenants": len(descendants),
	}))
}

// alertsListDefaultPageSize matches the per-list default the project uses
// elsewhere when the helper's general 20 is too small for the typical UX
// (cf. systems.go which also overrides to 50). Capped at 100 by the helper.
const alertsListDefaultPageSize = 50

// alertsListAllowedSortBy enumerates the user-selectable sort columns for the
// active /api/alerts list. Default is starts_at desc (most recent first).
// severity is sorted by criticality rank (critical > warning > info > other),
// not lexicographically. Anything outside this set falls back to starts_at.
var alertsListAllowedSortBy = map[string]bool{
	"starts_at": true,
	"severity":  true,
	"alertname": true,
}

// severityRank maps severity labels to a comparable integer (higher = more
// severe). Unknown values get -1 so they sort below info.
var severityRank = map[string]int{
	"critical": 3,
	"warning":  2,
	"info":     1,
}

// GetAlerts handles GET /api/alerts
//
// Lists active alerts. Scope follows the same three modes as /alerts/totals:
//   - no organization_id    → caller's full hierarchy (cross-tenant fan-out)
//   - organization_id=X     → single tenant X
//   - organization_id=X & include=descendants → X plus its sub-tree
//
// Customer callers are always pinned to their own organization.
//
// Response contains the paginated slice plus a `pagination` object and a
// `warnings` array (always present, populated only when one or more tenants
// failed during fan-out — the rest of the result is still returned).
func GetAlerts(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	orgIDs, ok := resolveOrgScope(c, user)
	if !ok {
		return
	}

	page, pageSize := helpers.GetPaginationFromQuery(c)
	if c.Query("page_size") == "" {
		pageSize = alertsListDefaultPageSize
	}

	// helpers.GetSortingFromQuery defaults sort_direction to "asc"; for active
	// alerts the natural default is "what's firing now" first, so we override
	// to desc when the caller didn't pass an explicit direction.
	sortBy, sortDirection := helpers.GetSortingFromQuery(c)
	if !alertsListAllowedSortBy[sortBy] {
		sortBy = "starts_at"
	}
	if c.Query("sort_direction") == "" {
		sortDirection = "desc"
	}

	all, warnings := fanOutMimirAlerts(c.Request.Context(), orgIDs)

	all = filterAlerts(all, alertFilter{
		states:     c.QueryArray("state"),
		severities: c.QueryArray("severity"),
		systemKeys: c.QueryArray("system_key"),
		alertnames: c.QueryArray("alertname"),
	})

	// Sort with fingerprint as a stable tiebreaker so pagination doesn't
	// shift between requests when the primary key ties.
	sortAlertsList(all, sortBy, sortDirection)

	totalCount := len(all)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > totalCount {
		start = totalCount
	}
	if end > totalCount {
		end = totalCount
	}
	pageAlerts := all[start:end]
	if pageAlerts == nil {
		pageAlerts = []map[string]interface{}{}
	}

	if warnings == nil {
		warnings = []string{}
	}

	c.JSON(http.StatusOK, response.OK("alerts retrieved successfully", gin.H{
		"alerts":     pageAlerts,
		"pagination": helpers.BuildPaginationInfoWithSorting(page, pageSize, totalCount, sortBy, sortDirection),
		"warnings":   warnings,
	}))
}

// sortAlertsList orders an in-memory slice of Mimir alerts by the given column
// and direction. Always uses fingerprint as a stable secondary key so paging
// doesn't shuffle alerts that tie on the primary key.
func sortAlertsList(alerts []map[string]interface{}, sortBy, sortDirection string) {
	desc := sortDirection == "desc"
	sort.SliceStable(alerts, func(i, j int) bool {
		var primaryLess bool
		var primaryEqual bool
		switch sortBy {
		case "severity":
			si := severityRank[severityOf(alerts[i])]
			sj := severityRank[severityOf(alerts[j])]
			primaryEqual = si == sj
			primaryLess = si < sj
		case "alertname":
			ai := alertnameOf(alerts[i])
			aj := alertnameOf(alerts[j])
			primaryEqual = ai == aj
			primaryLess = ai < aj
		default: // starts_at
			si, _ := alerts[i]["startsAt"].(string)
			sj, _ := alerts[j]["startsAt"].(string)
			primaryEqual = si == sj
			primaryLess = si < sj
		}
		if !primaryEqual {
			if desc {
				return !primaryLess
			}
			return primaryLess
		}
		// Tiebreaker: fingerprint asc, deterministic regardless of direction.
		fi, _ := alerts[i]["fingerprint"].(string)
		fj, _ := alerts[j]["fingerprint"].(string)
		return fi < fj
	})
}

func severityOf(alert map[string]interface{}) string {
	labels, _ := alert["labels"].(map[string]interface{})
	s, _ := labels["severity"].(string)
	return s
}

func alertnameOf(alert map[string]interface{}) string {
	labels, _ := alert["labels"].(map[string]interface{})
	s, _ := labels["alertname"].(string)
	return s
}

// alertFingerprintPattern restricts the fingerprint path param to safe chars.
// Alertmanager fingerprints are 16-char lowercase hex but we allow a slightly
// looser charset to accommodate test fixtures and any future format change.
var alertFingerprintPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

// GetAlertActivity handles GET /api/alerts/:fingerprint/activity
// Returns the per-alert audit timeline (silence created/updated/removed) for
// the alert identified by fingerprint within the resolved tenant. Most recent
// first. Operator notes are stored as the comment of the silence the action
// produced, so the timeline is the source of truth for "what happened, when,
// by whom".
func GetAlertActivity(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}
	fp := c.Param("fingerprint")
	if !alertFingerprintPattern.MatchString(fp) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid fingerprint", nil))
		return
	}
	orgID, ok := resolveOrgID(c, user)
	if !ok {
		return
	}
	if !requireOrgID(c, orgID) {
		return
	}

	limit := 100
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			limit = n
			if limit > 500 {
				limit = 500
			}
		}
	}

	repo := entities.NewLocalAlertActivityRepository()
	entries, err := repo.ListByFingerprint(orgID, fp, limit)
	if err != nil {
		logger.Error().Err(err).Str("org_id", orgID).Str("fingerprint", fp).Msg("failed to list alert activity")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to list alert activity", nil))
		return
	}
	c.JSON(http.StatusOK, response.OK("alert activity retrieved successfully", gin.H{
		"events": entries,
	}))
}

// fanOutMimirAlerts fetches active alerts from Mimir for every tenant in scope
// concurrently, with bounded concurrency and a global timeout. Per-tenant
// failures (timeout, 5xx, parse error) are collected as warnings; the rest of
// the result is returned.
//
// Each alert is enriched with a top-level `system` object containing the
// owning system's `name` and `type` (product, e.g. "nsec") looked up in the
// local `systems` table by (org_id, system_key). This saves the frontend a
// per-row round-trip to /systems just to render the table cell. If the lookup
// fails or the alert has no system_key, the field is simply omitted.
func fanOutMimirAlerts(parent context.Context, orgIDs []string) ([]map[string]interface{}, []string) {
	var (
		all      []map[string]interface{}
		warnings []string
		mu       sync.Mutex
		wg       sync.WaitGroup
	)
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
				logger.Warn().Err(err).Str("org_id", orgID).Msg("failed to fetch alerts from mimir for list")
				mu.Lock()
				warnings = append(warnings, fmt.Sprintf("org %s: %s", orgID, err.Error()))
				mu.Unlock()
				return
			}

			var alerts []map[string]interface{}
			if err := json.Unmarshal(body, &alerts); err != nil {
				logger.Warn().Err(err).Str("org_id", orgID).Msg("failed to parse mimir alerts response for list")
				mu.Lock()
				warnings = append(warnings, fmt.Sprintf("org %s: parse error", orgID))
				mu.Unlock()
				return
			}

			enrichAlertsWithSystemInfo(orgID, alerts)

			mu.Lock()
			all = append(all, alerts...)
			mu.Unlock()
		}(orgID)
	}
	wg.Wait()
	return all, warnings
}

// enrichAlertsWithSystemInfo decorates each alert with a `system` object
// (id, name, type, system_key) by issuing a single SELECT against the local
// systems table for the distinct system_key values it sees. Best-effort:
// a DB hiccup or an unmatched key just leaves the system field unset on
// that alert. The `id` is the local DB UUID — what the frontend uses to
// build the system-detail link (/systems/:id).
func enrichAlertsWithSystemInfo(orgID string, alerts []map[string]interface{}) {
	if len(alerts) == 0 {
		return
	}
	keys := make(map[string]struct{}, len(alerts))
	for _, a := range alerts {
		labels, _ := a["labels"].(map[string]interface{})
		if k, ok := labels["system_key"].(string); ok && k != "" {
			keys[k] = struct{}{}
		}
	}
	if len(keys) == 0 {
		return
	}
	keyList := make([]string, 0, len(keys))
	for k := range keys {
		keyList = append(keyList, k)
	}
	rows, err := database.DB.Query(
		`SELECT id, system_key, name, type FROM systems WHERE organization_id = $1 AND system_key = ANY($2)`,
		orgID, pq.Array(keyList),
	)
	if err != nil {
		logger.Warn().Err(err).Str("org_id", orgID).Msg("failed to lookup system info for alert enrichment")
		return
	}
	defer func() { _ = rows.Close() }()

	type sysInfo struct {
		ID   string
		Name string
		Type sql.NullString
	}
	infoBy := make(map[string]sysInfo, len(keyList))
	for rows.Next() {
		var id, k, n string
		var t sql.NullString
		if err := rows.Scan(&id, &k, &n, &t); err != nil {
			continue
		}
		infoBy[k] = sysInfo{ID: id, Name: n, Type: t}
	}

	for _, a := range alerts {
		labels, _ := a["labels"].(map[string]interface{})
		k, _ := labels["system_key"].(string)
		if k == "" {
			continue
		}
		info, ok := infoBy[k]
		if !ok {
			continue
		}
		s := map[string]interface{}{
			"id":         info.ID,
			"system_key": k,
			"name":       info.Name,
		}
		if info.Type.Valid {
			s["type"] = info.Type.String
		}
		a["system"] = s
	}
}

// GetAlertingConfig handles GET /api/alerts/config — returns the CALLER's
// own alerting layer. Returns an empty layer (with audit metadata absent)
// when the caller has never saved one; the frontend renders the empty-state
// form on top of it.
//
// Nothing else is exposed: no inherited ancestor layers, no merged
// effective view. Every organization sees only its own configuration,
// regardless of role. The merge happens server-side at render time and
// stays inside the backend.
func GetAlertingConfig(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	repo := entities.NewLocalAlertConfigLayersRepository()
	rec, err := repo.Get(user.OrganizationID)
	if err != nil && !errors.Is(err, entities.ErrAlertConfigLayerNotFound) {
		logger.Error().Err(err).Str("org_id", user.OrganizationID).Msg("failed to load alert config layer")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to load alert config", nil))
		return
	}

	if rec == nil {
		// First-time view: emit an empty layer body so the UI can render
		// without a null-check, plus null audit fields the UI uses to detect
		// the "never saved" state.
		c.JSON(http.StatusOK, response.OK("alerting layer retrieved successfully", gin.H{
			"enabled":             models.ChannelToggles{},
			"email_recipients":    []models.EmailRecipient{},
			"webhook_recipients":  []models.WebhookRecipient{},
			"telegram_recipients": []models.TelegramRecipient{},
			"updated_by_name":     nil,
			"updated_at":          nil,
		}))
		return
	}

	c.JSON(http.StatusOK, response.OK("alerting layer retrieved successfully", gin.H{
		"enabled":             rec.Config.Enabled,
		"email_recipients":    rec.Config.EmailRecipients,
		"webhook_recipients":  rec.Config.WebhookRecipients,
		"telegram_recipients": rec.Config.TelegramRecipients,
		"updated_by_name":     rec.UpdatedByName,
		"updated_at":          rec.UpdatedAt,
	}))
}

// alertsTotalsFanoutTimeout caps how long /totals will wait for Mimir to answer
// across the caller's whole hierarchy. Per-tenant calls that don't return in
// time are reported as warnings; their counts simply don't contribute. Tuned
// to keep dashboard latency bounded even when a tenant's Mimir slot is slow.
const alertsTotalsFanoutTimeout = 10 * time.Second

// alertsTotalsFanoutConcurrency caps simultaneous in-flight Mimir requests for
// the /totals fan-out. Prevents an Owner with hundreds of tenants from opening
// hundreds of sockets at once.
const alertsTotalsFanoutConcurrency = 10

// GetAlertsTotals handles GET /api/alerts/totals
// Returns active alert counts by severity (from Mimir) and total history count (from DB).
//
// Without organization_id the totals are aggregated across the caller's full
// hierarchy (one Mimir call per tenant, fanned out with bounded concurrency
// and a global timeout). With organization_id the call is scoped to that
// single tenant.
func GetAlertsTotals(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	orgIDs, ok := resolveOrgScope(c, user)
	if !ok {
		return
	}

	var (
		active, critical, warning, info, muted int
		warnings                               []string
		mu                                     sync.Mutex
		wg                                     sync.WaitGroup
	)

	ctx, cancel := context.WithTimeout(c.Request.Context(), alertsTotalsFanoutTimeout)
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
				logger.Warn().Err(err).Str("org_id", orgID).Msg("failed to fetch alerts from mimir for totals")
				mu.Lock()
				warnings = append(warnings, fmt.Sprintf("org %s: %s", orgID, err.Error()))
				mu.Unlock()
				return
			}

			var alerts []map[string]interface{}
			if err := json.Unmarshal(body, &alerts); err != nil {
				logger.Warn().Err(err).Str("org_id", orgID).Msg("failed to parse mimir alerts response for totals")
				mu.Lock()
				warnings = append(warnings, fmt.Sprintf("org %s: parse error", orgID))
				mu.Unlock()
				return
			}

			var localCritical, localWarning, localInfo, localMuted int
			for _, alert := range alerts {
				labels, _ := alert["labels"].(map[string]interface{})
				switch sev, _ := labels["severity"].(string); sev {
				case "critical":
					localCritical++
				case "warning":
					localWarning++
				case "info":
					localInfo++
				}
				// An alert is muted when Alertmanager has at least one
				// active silence matching it (status.silencedBy non-empty).
				if status, ok := alert["status"].(map[string]interface{}); ok {
					if sb, ok := status["silencedBy"].([]interface{}); ok && len(sb) > 0 {
						localMuted++
					}
				}
			}

			mu.Lock()
			active += len(alerts)
			critical += localCritical
			warning += localWarning
			info += localInfo
			muted += localMuted
			mu.Unlock()
		}(orgID)
	}
	wg.Wait()

	repo := entities.NewLocalAlertHistoryRepository()
	historyTotal, err := repo.GetAlertHistoryTotalsByOrgIDs(orgIDs)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to count alert history for totals")
		warnings = append(warnings, fmt.Sprintf("history: %s", err.Error()))
	}

	if warnings == nil {
		warnings = []string{}
	}

	c.JSON(http.StatusOK, response.OK("alert totals retrieved successfully", gin.H{
		"active":   active,
		"critical": critical,
		"warning":  warning,
		"info":     info,
		"muted":    muted,
		"history":  historyTotal,
		"warnings": warnings,
	}))
}

// GetAlertsTrend handles GET /api/alerts/trend
// Returns trend data for resolved alerts over a specified period, scoped to
// the caller's hierarchy (no organization_id), a specific tenant, or a
// sub-tree (organization_id=X&include=descendants). Mirrors the scope rules
// of /api/alerts/totals.
func GetAlertsTrend(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	orgIDs, ok := resolveOrgScope(c, user)
	if !ok {
		return
	}

	periodStr := c.DefaultQuery("period", "7")
	period, err := strconv.Atoi(periodStr)
	if err != nil || (period != 7 && period != 30 && period != 180 && period != 365) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid period parameter (supported: 7, 30, 180, 365)", nil))
		return
	}

	repo := entities.NewLocalAlertHistoryRepository()
	trend, err := repo.GetAlertHistoryTrendByOrgIDs(period, orgIDs)
	if err != nil {
		logger.Error().Err(err).Int("period", period).Msg("failed to retrieve alerts trend")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve alerts trend", nil))
		return
	}

	if trend.DataPoints == nil {
		trend.DataPoints = []models.TrendDataPoint{}
	}

	c.JSON(http.StatusOK, response.OK("alerts trend retrieved successfully", trend))
}

// alertFilter holds the multi-value filters parsed from /api/alerts query
// params. Internal to this package — not exposed in the public models, since
// it's a binding detail of one handler. Multiple values within a single field
// are matched as OR; different fields AND together.
type alertFilter struct {
	states     []string
	severities []string
	systemKeys []string
	alertnames []string
}

// filterAlerts applies optional multi-value query filters to the alerts list.
// An alert is excluded when a requested filter's target label/field is missing
// or does not match any of the requested values; this prevents silent leakage
// of unrelated alerts when the caller narrows the query.
func filterAlerts(alerts []map[string]interface{}, f alertFilter) []map[string]interface{} {
	if len(f.states) == 0 && len(f.severities) == 0 && len(f.systemKeys) == 0 && len(f.alertnames) == 0 {
		return alerts
	}

	filtered := make([]map[string]interface{}, 0, len(alerts))
	for _, alert := range alerts {
		if len(f.states) > 0 {
			status, ok := alert["status"].(map[string]interface{})
			if !ok {
				continue
			}
			state, ok := status["state"].(string)
			if !ok || !slices.Contains(f.states, state) {
				continue
			}
		}

		labels, _ := alert["labels"].(map[string]interface{})

		if len(f.severities) > 0 {
			sev, ok := labels["severity"].(string)
			if !ok || !slices.Contains(f.severities, sev) {
				continue
			}
		}

		if len(f.systemKeys) > 0 {
			sk, ok := labels["system_key"].(string)
			if !ok || !slices.Contains(f.systemKeys, sk) {
				continue
			}
		}

		if len(f.alertnames) > 0 {
			an, ok := labels["alertname"].(string)
			if !ok || !slices.Contains(f.alertnames, an) {
				continue
			}
		}

		filtered = append(filtered, alert)
	}

	return filtered
}

func getSystemAlertOrgID(system *models.System) string {
	if system.Organization.LogtoID != "" {
		return system.Organization.LogtoID
	}
	return system.Organization.ID
}

func findSystemAlertByFingerprint(alerts []models.ActiveAlert, fingerprint, systemKey string) *models.ActiveAlert {
	for i := range alerts {
		if alerts[i].Fingerprint != fingerprint {
			continue
		}
		if alerts[i].Labels["system_key"] != systemKey {
			continue
		}
		return &alerts[i]
	}
	return nil
}

func normalizeAlertSilenceComment(comment string) string {
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return defaultSystemAlertSilenceComment
	}
	return comment
}

func getAlertSilenceCreatedBy(user *models.User) string {
	if user == nil {
		return "my"
	}
	if user.Username != "" {
		return user.Username
	}
	if user.Email != "" {
		return user.Email
	}
	if user.Name != "" {
		return user.Name
	}
	if user.ID != "" {
		return user.ID
	}
	return "my"
}

func buildSystemAlertSilenceRequest(
	alert *models.ActiveAlert,
	systemKey, createdBy, comment string,
	durationMinutes int,
	now time.Time,
	endsAt time.Time, // if non-zero, overrides durationMinutes
) *models.AlertmanagerSilenceRequest {
	if endsAt.IsZero() {
		if durationMinutes <= 0 {
			durationMinutes = defaultSystemAlertSilenceDurationMinutes
		}
		endsAt = now.Add(time.Duration(durationMinutes) * time.Minute)
	}

	labelNames := make([]string, 0, len(alert.Labels)+1)
	for name, value := range alert.Labels {
		if strings.TrimSpace(value) == "" {
			continue
		}
		labelNames = append(labelNames, name)
	}
	if systemKey != "" {
		if _, found := alert.Labels["system_key"]; !found {
			labelNames = append(labelNames, "system_key")
		}
	}

	sort.Strings(labelNames)

	matchers := make([]models.AlertmanagerMatcher, 0, len(labelNames))
	for _, name := range labelNames {
		value := alert.Labels[name]
		if name == "system_key" {
			value = systemKey
		}
		if strings.TrimSpace(value) == "" {
			continue
		}

		matchers = append(matchers, models.AlertmanagerMatcher{
			Name:    name,
			Value:   value,
			IsRegex: false,
		})
	}

	return &models.AlertmanagerSilenceRequest{
		Matchers:  matchers,
		StartsAt:  now.Format(time.RFC3339),
		EndsAt:    endsAt.Format(time.RFC3339),
		Comment:   normalizeAlertSilenceComment(comment),
		CreatedBy: createdBy,
	}
}

func silenceBelongsToSystem(silence *models.AlertmanagerSilence, systemKey string) bool {
	if silence == nil || systemKey == "" {
		return false
	}

	for _, matcher := range silence.Matchers {
		if matcher.Name == "system_key" && matcher.Value == systemKey && !matcher.IsRegex {
			return true
		}
	}

	return false
}

// GetSystemAlerts handles GET /api/systems/:id/alerts
// Returns active alerts from Mimir for a specific system, filtered by system_key.
func GetSystemAlerts(c *gin.Context) {
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system id required", nil))
		return
	}

	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	systemsService := local.NewSystemsService()
	system, err := systemsService.GetSystem(systemID, userOrgRole, userOrgID)
	if helpers.HandleAccessError(c, err, "system", systemID) {
		return
	}

	body, err := alerting.GetAlerts(getSystemAlertOrgID(system))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch alerts from mimir: "+err.Error(), nil))
		return
	}

	var alerts []map[string]interface{}
	if err := json.Unmarshal(body, &alerts); err != nil {
		c.JSON(http.StatusOK, response.OK("alerts retrieved successfully", gin.H{
			"alerts": []interface{}{},
		}))
		return
	}

	// Filter alerts by this system's key and decorate each with the same
	// `system` enrichment shape as /api/alerts. The system was already loaded
	// at the start of the handler so no extra query is needed.
	sysInfo := map[string]interface{}{
		"id":         system.ID,
		"system_key": system.SystemKey,
		"name":       system.Name,
	}
	if system.Type != nil {
		sysInfo["type"] = *system.Type
	}
	filtered := make([]map[string]interface{}, 0, len(alerts))
	for _, alert := range alerts {
		labels, _ := alert["labels"].(map[string]interface{})
		if sk, ok := labels["system_key"].(string); ok && sk == system.SystemKey {
			alert["system"] = sysInfo
			filtered = append(filtered, alert)
		}
	}

	c.JSON(http.StatusOK, response.OK("alerts retrieved successfully", gin.H{
		"alerts": filtered,
	}))
}

// CreateSystemAlertSilence handles POST /api/systems/:id/alerts/silences
// Creates a silence in Alertmanager for a specific active system alert.
func CreateSystemAlertSilence(c *gin.Context) {
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system id required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	systemsService := local.NewSystemsService()
	system, err := systemsService.GetSystem(systemID, user.OrgRole, user.OrganizationID)
	if helpers.HandleAccessError(c, err, "system", systemID) {
		return
	}
	if system.SystemKey == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system is not registered", nil))
		return
	}

	var req models.CreateSystemAlertSilenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body: "+err.Error(), nil))
		return
	}

	body, err := alerting.GetAlerts(getSystemAlertOrgID(system))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch alerts from mimir: "+err.Error(), nil))
		return
	}

	var alerts []models.ActiveAlert
	if err := json.Unmarshal(body, &alerts); err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to parse alerts from mimir: "+err.Error(), nil))
		return
	}

	alert := findSystemAlertByFingerprint(alerts, req.Fingerprint, system.SystemKey)
	if alert == nil {
		c.JSON(http.StatusNotFound, response.NotFound("alert not found", nil))
		return
	}
	if len(alert.Status.SilencedBy) > 0 {
		c.JSON(http.StatusBadRequest, response.BadRequest("alert is already silenced", nil))
		return
	}

	now := time.Now().UTC()
	var endsAt time.Time
	if req.EndAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.EndAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.BadRequest("invalid end_at: must be RFC3339 datetime", nil))
			return
		}
		if !parsed.After(now) {
			c.JSON(http.StatusBadRequest, response.BadRequest("invalid end_at: must be in the future", nil))
			return
		}
		endsAt = parsed.UTC()
	}

	silenceReq := buildSystemAlertSilenceRequest(
		alert,
		system.SystemKey,
		getAlertSilenceCreatedBy(user),
		req.Comment,
		req.DurationMinutes,
		now,
		endsAt,
	)

	silenceResp, err := alerting.CreateSilence(getSystemAlertOrgID(system), silenceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create silence in mimir: "+err.Error(), nil))
		return
	}

	// Best-effort activity log: a write failure must not break the user's
	// silence creation, so we only log a warning. The silence is the source
	// of truth; activity is a denormalised UX convenience.
	logAlertActivity(c, getSystemAlertOrgID(system), req.Fingerprint, entities.AlertActivitySilenced, user, silenceResp.SilenceID, map[string]interface{}{
		"comment":          normalizeAlertSilenceComment(req.Comment),
		"duration_minutes": req.DurationMinutes,
		"end_at":           req.EndAt,
	})

	c.JSON(http.StatusOK, response.OK("alert silenced successfully", gin.H{
		"silence_id": silenceResp.SilenceID,
	}))
}

// logAlertActivity writes one row to alert_activity. Fails open: a DB error
// is logged at warn level but does not surface to the caller, since the
// activity timeline is auxiliary to the primary action. Empty fingerprint
// silently no-ops because the row would be unreachable from any alert detail
// view.
func logAlertActivity(c *gin.Context, orgID, fingerprint, action string, user *models.User, silenceID string, details map[string]interface{}) {
	if fingerprint == "" {
		return
	}
	actorUserID := ""
	if user != nil && user.LogtoID != nil {
		actorUserID = *user.LogtoID
	}
	actorName := ""
	if user != nil {
		actorName = user.Name
	}
	repo := entities.NewLocalAlertActivityRepository()
	if err := repo.Log(orgID, fingerprint, action, actorUserID, actorName, silenceID, details); err != nil {
		logger.Warn().Err(err).
			Str("org_id", orgID).
			Str("fingerprint", fingerprint).
			Str("action", action).
			Msg("failed to write alert activity (non-fatal)")
	}
}

// DeleteSystemAlertSilence handles DELETE /api/systems/:id/alerts/silences/:silence_id
// Deletes a system-scoped silence in Alertmanager after validating its ownership.
func DeleteSystemAlertSilence(c *gin.Context) {
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system id required", nil))
		return
	}

	silenceID := c.Param("silence_id")
	if silenceID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("silence id required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	systemsService := local.NewSystemsService()
	system, err := systemsService.GetSystem(systemID, user.OrgRole, user.OrganizationID)
	if helpers.HandleAccessError(c, err, "system", systemID) {
		return
	}
	if system.SystemKey == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system is not registered", nil))
		return
	}

	orgID := getSystemAlertOrgID(system)
	silence, err := alerting.GetSilence(orgID, silenceID)
	if errors.Is(err, alerting.ErrSilenceNotFound) {
		c.JSON(http.StatusNotFound, response.NotFound("silence not found", nil))
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch silence from mimir: "+err.Error(), nil))
		return
	}
	if !silenceBelongsToSystem(silence, system.SystemKey) {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to silence", nil))
		return
	}

	if err := alerting.DeleteSilence(orgID, silenceID); errors.Is(err, alerting.ErrSilenceNotFound) {
		c.JSON(http.StatusNotFound, response.NotFound("silence not found", nil))
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to delete silence in mimir: "+err.Error(), nil))
		return
	}

	// Resolve the fingerprint of the alert this silence was originally tied to
	// so the unsilence event lands on the right timeline. If we can't find it
	// (e.g. silence pre-dates activity tracking), skip the activity write.
	activityRepo := entities.NewLocalAlertActivityRepository()
	fingerprint, _ := activityRepo.FindFingerprintBySilenceID(orgID, silenceID)
	logAlertActivity(c, orgID, fingerprint, entities.AlertActivityUnsilenced, user, silenceID, nil)

	c.JSON(http.StatusOK, response.OK("silence disabled successfully", nil))
}

// GetSystemAlertSilences handles GET /api/systems/:id/alerts/silences
// Returns all active and pending silences in Alertmanager that are scoped to this system.
func GetSystemAlertSilences(c *gin.Context) {
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system id required", nil))
		return
	}

	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	systemsService := local.NewSystemsService()
	system, err := systemsService.GetSystem(systemID, userOrgRole, userOrgID)
	if helpers.HandleAccessError(c, err, "system", systemID) {
		return
	}

	silences, err := alerting.GetSilences(getSystemAlertOrgID(system))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch silences from mimir: "+err.Error(), nil))
		return
	}

	filtered := make([]models.AlertmanagerSilence, 0, len(silences))
	for _, s := range silences {
		if !silenceBelongsToSystem(&s, system.SystemKey) {
			continue
		}
		// Exclude expired silences from the list
		if s.Status != nil && s.Status.State == "expired" {
			continue
		}
		filtered = append(filtered, s)
	}

	c.JSON(http.StatusOK, response.OK("silences retrieved successfully", gin.H{
		"silences": filtered,
	}))
}

// GetSystemAlertSilence handles GET /api/systems/:id/alerts/silences/:silence_id
// Returns a single silence after validating it belongs to this system.
func GetSystemAlertSilence(c *gin.Context) {
	systemID := c.Param("id")
	silenceID := c.Param("silence_id")
	if systemID == "" || silenceID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system id and silence id required", nil))
		return
	}

	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	systemsService := local.NewSystemsService()
	system, err := systemsService.GetSystem(systemID, userOrgRole, userOrgID)
	if helpers.HandleAccessError(c, err, "system", systemID) {
		return
	}

	orgID := getSystemAlertOrgID(system)
	silence, err := alerting.GetSilence(orgID, silenceID)
	if errors.Is(err, alerting.ErrSilenceNotFound) {
		c.JSON(http.StatusNotFound, response.NotFound("silence not found", nil))
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch silence from mimir: "+err.Error(), nil))
		return
	}
	if !silenceBelongsToSystem(silence, system.SystemKey) {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to silence", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("silence retrieved successfully", gin.H{
		"silence": silence,
	}))
}

// UpdateSystemAlertSilence handles PUT /api/systems/:id/alerts/silences/:silence_id
// Updates the end time and/or comment of an existing silence.
// Preserves the original matchers and start time; only end time and comment change.
func UpdateSystemAlertSilence(c *gin.Context) {
	systemID := c.Param("id")
	silenceID := c.Param("silence_id")
	if systemID == "" || silenceID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system id and silence id required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	systemsService := local.NewSystemsService()
	system, err := systemsService.GetSystem(systemID, user.OrgRole, user.OrganizationID)
	if helpers.HandleAccessError(c, err, "system", systemID) {
		return
	}
	if system.SystemKey == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system is not registered", nil))
		return
	}

	var req models.UpdateSystemAlertSilenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body: "+err.Error(), nil))
		return
	}

	now := time.Now().UTC()
	endsAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid end_at: must be RFC3339 datetime", nil))
		return
	}
	if !endsAt.After(now) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid end_at: must be in the future", nil))
		return
	}

	orgID := getSystemAlertOrgID(system)
	existing, err := alerting.GetSilence(orgID, silenceID)
	if errors.Is(err, alerting.ErrSilenceNotFound) {
		c.JSON(http.StatusNotFound, response.NotFound("silence not found", nil))
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch silence from mimir: "+err.Error(), nil))
		return
	}
	if !silenceBelongsToSystem(existing, system.SystemKey) {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to silence", nil))
		return
	}

	updateReq := &models.AlertmanagerSilenceRequest{
		ID:        existing.ID,
		Matchers:  existing.Matchers,
		StartsAt:  existing.StartsAt,
		EndsAt:    endsAt.UTC().Format(time.RFC3339),
		Comment:   normalizeAlertSilenceComment(req.Comment),
		CreatedBy: existing.CreatedBy,
	}

	silenceResp, err := alerting.CreateSilence(orgID, updateReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to update silence in mimir: "+err.Error(), nil))
		return
	}

	// Resolve fingerprint from the original silence creation event so the
	// update lands on the right alert's timeline.
	activityRepo := entities.NewLocalAlertActivityRepository()
	fingerprint, _ := activityRepo.FindFingerprintBySilenceID(orgID, silenceID)
	logAlertActivity(c, orgID, fingerprint, entities.AlertActivitySilenceUpdated, user, silenceResp.SilenceID, map[string]interface{}{
		"comment": normalizeAlertSilenceComment(req.Comment),
		"end_at":  req.EndAt,
	})

	c.JSON(http.StatusOK, response.OK("silence updated successfully", gin.H{
		"silence_id": silenceResp.SilenceID,
	}))
}

// GetSystemAlertHistory handles GET /api/systems/:id/alerts/history
// Returns paginated resolved/inactive alert history for a system, with
// optional date range (?from_date=, ?to_date=, RFC3339) and multi-value
// label filters (alertname, severity, status).
func GetSystemAlertHistory(c *gin.Context) {
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system id required", nil))
		return
	}

	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Validate system access and retrieve system_key.
	systemsService := local.NewSystemsService()
	system, err := systemsService.GetSystem(systemID, userOrgRole, userOrgID)
	if helpers.HandleAccessError(c, err, "system", systemID) {
		return
	}

	page, pageSize, sortBy, sortDirection := helpers.GetPaginationAndSortingFromQuery(c)

	from, to, perr := parseDateRange(c)
	if perr != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(perr.Error(), nil))
		return
	}

	repo := entities.NewLocalAlertHistoryRepository()
	records, totalCount, err := repo.QueryAlertHistory(entities.AlertHistoryQuery{
		OrgIDs:        []string{system.Organization.LogtoID},
		SystemKey:     system.SystemKey,
		Alertnames:    c.QueryArray("alertname"),
		Severities:    c.QueryArray("severity"),
		Statuses:      c.QueryArray("status"),
		From:          from,
		To:            to,
		Page:          page,
		PageSize:      pageSize,
		SortBy:        sortBy,
		SortDirection: sortDirection,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Str("system_id", systemID).
			Str("system_key", system.SystemKey).
			Msg("failed to retrieve alert history")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve alert history", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("alert history retrieved successfully", gin.H{
		"alerts":     records,
		"pagination": helpers.BuildPaginationInfoWithSorting(page, pageSize, totalCount, sortBy, sortDirection),
	}))
}

// GetAlertsHistory handles GET /api/alerts/history
// Returns paginated resolved alert history scoped to the caller's hierarchy
// (no organization_id), a single tenant (organization_id=X), or a sub-tree
// (organization_id=X&include=descendants). Mirrors the scope rules of
// /api/alerts/totals. Supports date range and multi-value label filters.
func GetAlertsHistory(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	orgIDs, ok := resolveOrgScope(c, user)
	if !ok {
		return
	}

	page, pageSize, sortBy, sortDirection := helpers.GetPaginationAndSortingFromQuery(c)

	from, to, perr := parseDateRange(c)
	if perr != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(perr.Error(), nil))
		return
	}

	repo := entities.NewLocalAlertHistoryRepository()
	records, totalCount, err := repo.QueryAlertHistory(entities.AlertHistoryQuery{
		OrgIDs:        orgIDs,
		Alertnames:    c.QueryArray("alertname"),
		Severities:    c.QueryArray("severity"),
		Statuses:      c.QueryArray("status"),
		From:          from,
		To:            to,
		Page:          page,
		PageSize:      pageSize,
		SortBy:        sortBy,
		SortDirection: sortDirection,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve alert history (org-level)")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve alert history", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("alert history retrieved successfully", gin.H{
		"alerts":     records,
		"pagination": helpers.BuildPaginationInfoWithSorting(page, pageSize, totalCount, sortBy, sortDirection),
	}))
}

// alertsStatsDefaultTopN is the default and maximum cap for top-N grouped
// breakdowns returned by /api/alerts/stats. The cap protects the response
// size when an org has thousands of distinct alertnames or system_keys.
const alertsStatsDefaultTopN = 10
const alertsStatsMaxTopN = 50

// GetAlertsStats handles GET /api/alerts/stats
// Returns aggregate statistics over alert_history for the caller's scope:
// total, by_severity buckets, top-N alertname / system_key, plus MTTR and
// MTBF approximations. Honors the same scope rules as /alerts/totals
// (no organization_id / single tenant / descendants drill-down) and accepts
// an optional date range (from_date / to_date, RFC3339).
func GetAlertsStats(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	orgIDs, ok := resolveOrgScope(c, user)
	if !ok {
		return
	}

	from, to, perr := parseDateRange(c)
	if perr != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(perr.Error(), nil))
		return
	}

	topN := alertsStatsDefaultTopN
	if s := c.Query("top"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			topN = n
			if topN > alertsStatsMaxTopN {
				topN = alertsStatsMaxTopN
			}
		}
	}

	repo := entities.NewLocalAlertHistoryRepository()
	stats, err := repo.GetAlertStats(entities.AlertStatsQuery{
		OrgIDs: orgIDs,
		From:   from,
		To:     to,
		TopN:   topN,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve alert stats")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve alert stats", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("alert stats retrieved successfully", stats))
}

// parseDateRange reads optional from_date / to_date query params and parses
// them as RFC3339 timestamps. Returns nil values when the param is missing,
// or a 400-friendly error when the param is present but malformed.
func parseDateRange(c *gin.Context) (*time.Time, *time.Time, error) {
	var from, to *time.Time
	if s := c.Query("from_date"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid from_date: must be RFC3339 (e.g. 2026-05-01T00:00:00Z)")
		}
		from = &t
	}
	if s := c.Query("to_date"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid to_date: must be RFC3339 (e.g. 2026-05-08T00:00:00Z)")
		}
		to = &t
	}
	if from != nil && to != nil && !to.After(*from) {
		return nil, nil, fmt.Errorf("to_date must be after from_date")
	}
	return from, to, nil
}

// validateWebhookRecipients enforces that every webhook URL is a plain http/https
// URL pointing to a publicly-routable host. This protects Mimir's Alertmanager
// (which dispatches alert payloads from inside the internal network) from being
// abused as a blind SSRF relay to loopback, metadata, or private-range hosts.
func validateWebhookRecipients(recipients []models.WebhookRecipient) error {
	for _, r := range recipients {
		if err := validateWebhookURL(r.URL); err != nil {
			return fmt.Errorf("webhook recipient %q: %w", r.Name, err)
		}
	}
	return nil
}

// fqdnPattern restricts webhook hostnames to the canonical RFC1035 form when
// they aren't valid IP literals. Rejecting non-canonical forms (decimal IPs
// like "2130706433", octal "0177.0.0.1", hex "0x7f.0.0.1") closes the door
// on libc-dependent address parsing where some resolvers (notably glibc)
// would interpret them as 127.0.0.1, while our denylist keys on string
// prefixes that miss those encodings.
var fqdnPattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*\.?$`)

// cgnatRange is the carrier-grade NAT range (RFC6598). Not covered by
// net.IP.IsPrivate() and a real bypass risk on cloud and ISP networks.
var cgnatRange = &net.IPNet{IP: net.IPv4(100, 64, 0, 0).To4(), Mask: net.CIDRMask(10, 32)}

func validateWebhookURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid webhook url: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("webhook url must use http or https, got %q", u.Scheme)
	}
	if u.User != nil {
		return fmt.Errorf("webhook url must not contain credentials")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("webhook url is missing a host")
	}

	hostLower := strings.ToLower(host)
	for _, prefix := range webhookHostDenylist {
		if hostLower == strings.TrimSuffix(prefix, ".") || strings.HasPrefix(hostLower, prefix) {
			return fmt.Errorf("webhook url host %q is not allowed (loopback, metadata, or private network)", host)
		}
	}

	// Strict input shape. Either:
	//   1) a valid IP literal (v4 or v6), or
	//   2) a canonical FQDN (RFC1035 — alpha-prefixed labels, optional trailing dot)
	//      AND containing at least one alphabetic character.
	// The "at least one letter" rule rejects all-digit hosts that some resolvers
	// (notably macOS BSD libc) interpret as alternative IP encodings. Concrete
	// case caught: "0177.0.0.1" — net.ParseIP returns nil, fqdnPattern matches,
	// but BSD getaddrinfo strips the leading zero and resolves to "177.0.0.1"
	// (a public IP unrelated to the user's intent of 127.0.0.1). Without the
	// letter requirement, the URL would be saved with an unexpected destination.
	parsedIP := net.ParseIP(host)
	if parsedIP == nil {
		if !fqdnPattern.MatchString(host) {
			return fmt.Errorf("webhook url host %q is not a canonical hostname or IP literal", host)
		}
		if !strings.ContainsAny(host, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			return fmt.Errorf("webhook url host %q must be either an IP literal or a hostname with at least one letter", host)
		}
	}

	// IP literal path: reject loopback / private / link-local / multicast /
	// unspecified / CGNAT. Handles IPv6-mapped IPv4 via ip.To4() unmasking.
	if parsedIP != nil {
		if err := rejectNonPublicIP(parsedIP); err != nil {
			return fmt.Errorf("webhook url host %q: %w", host, err)
		}
		return nil
	}

	// FQDN path: resolve and reject if ANY returned address is non-public.
	// NOTE on DNS rebinding: this validation is point-in-time; Mimir's
	// Alertmanager re-resolves the hostname when delivering the webhook.
	// A short-TTL record can resolve to a public IP here and to a private
	// one at delivery. Authoritative mitigation requires either pinning the
	// resolved IP into the URL pushed to Mimir (breaks legitimate cloud-LB
	// hosts whose IPs rotate) or running the egress through a proxy that
	// re-validates per-request. Today we accept the residual risk and rely
	// on Mimir-side network ACLs to backstop egress to private ranges.
	addrs, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("webhook url host %q: dns resolution failed: %w", host, err)
	}
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			return fmt.Errorf("webhook url host %q resolved to unparseable address %q", host, addr)
		}
		if err := rejectNonPublicIP(ip); err != nil {
			return fmt.Errorf("webhook url host %q resolved to non-public address %s: %w", host, addr, err)
		}
	}

	return nil
}

// rejectNonPublicIP returns an error if the IP is loopback, private, link-local,
// multicast, unspecified, or in the carrier-grade NAT range (RFC6598). For
// IPv6-mapped IPv4 addresses (::ffff:A.B.C.D), the underlying IPv4 is checked.
func rejectNonPublicIP(ip net.IP) error {
	// Unmask IPv6-mapped IPv4 so checks work on the real address.
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() {
		return fmt.Errorf("address is not publicly routable")
	}
	// IsPrivate covers RFC1918 only — not CGNAT (100.64.0.0/10).
	if v4 := ip.To4(); v4 != nil && cgnatRange.Contains(v4) {
		return fmt.Errorf("address is in the carrier-grade NAT range (100.64.0.0/10)")
	}
	return nil
}
