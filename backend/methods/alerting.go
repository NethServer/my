/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"context"
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
	"github.com/go-playground/validator/v10"

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
		// Translate go-playground/validator errors into the standard
		// validation_error envelope (type/key/message/value) so the UI can
		// render per-field messages instead of a stringly-typed blob.
		if _, ok := err.(validator.ValidationErrors); ok {
			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", buildAlertingValidationErrors(err.(validator.ValidationErrors))))
			return
		}
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body: "+err.Error(), nil))
		return
	}

	// Defense-in-depth: run the model's stateless Validate before the
	// network-aware webhook checks so we surface any structural issue
	// (severities, language, format, email shape) that slipped past
	// binding tags as a structured 400 instead of a generic 500 from
	// the entity layer's backstop call.
	if err := req.Validate(); err != nil {
		if fieldErr := asAlertingFieldError(err); fieldErr != nil {
			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", []response.ValidationError{
				{Key: fieldErr.Key, Message: fieldErr.Code, Value: fieldErr.Value},
			}))
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
		if fieldErr := asAlertingFieldError(err); fieldErr != nil {
			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", []response.ValidationError{
				{Key: fieldErr.Key, Message: fieldErr.Code, Value: fieldErr.Value},
			}))
			return
		}
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
		// The entity layer calls cfg.Validate() as a backstop. If something
		// here trips it (shouldn't, given req.Validate() already ran above),
		// still surface a 400 with the structured envelope rather than 500.
		if fieldErr := asAlertingFieldError(err); fieldErr != nil {
			logger.Warn().Err(err).Str("org_id", user.OrganizationID).Msg("entity-layer validation backstop tripped")
			c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", []response.ValidationError{
				{Key: fieldErr.Key, Message: fieldErr.Code, Value: fieldErr.Value},
			}))
			return
		}
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

	// Map each affected org to its reseller tenant and dedupe: one nested config
	// per reseller covers all its customers, so push once per distinct tenant.
	tenants := distinctTenants(descendants)
	warnings := propagateAlertingConfigToTenants(c.Request.Context(), tenants)
	auditDetails := map[string]interface{}{
		"before":               snapshotLayerForAudit(prevLayer),
		"after":                snapshotLayerBodyForAudit(user.OrganizationID, req),
		"affected_tenants":     len(tenants),
		"propagation_warnings": len(warnings),
	}
	logger.LogBusinessOperationDetails(c, "alerts", "save_layer", "alert_config_layer", user.OrganizationID, true, nil, auditDetails)
	c.JSON(http.StatusOK, response.OK("alerting configuration updated successfully", gin.H{
		"warnings":         warnings,
		"propagated_to":    len(tenants) - len(warnings),
		"affected_tenants": len(tenants),
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

	// Map each affected org to its reseller tenant and dedupe (see ConfigureAlerts).
	tenants := distinctTenants(descendants)
	warnings := propagateAlertingConfigToTenants(c.Request.Context(), tenants)
	auditDetails := map[string]interface{}{
		"before":               snapshotLayerForAudit(prevLayer),
		"after":                nil,
		"affected_tenants":     len(tenants),
		"propagation_warnings": len(warnings),
	}
	logger.LogBusinessOperationDetails(c, "alerts", "delete_layer", "alert_config_layer", user.OrganizationID, true, nil, auditDetails)
	c.JSON(http.StatusOK, response.OK("alerting layer removed successfully", gin.H{
		"warnings":         warnings,
		"propagated_to":    len(tenants) - len(warnings),
		"affected_tenants": len(tenants),
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
//
// The intersection with /api/alerts/history (starts_at, severity, alertname,
// status) is intentional: the UI can offer a single "Sort by" dropdown that
// works on both tabs without sending column names that one side would
// silently fall back to its default.
var alertsListAllowedSortBy = map[string]bool{
	"starts_at": true,
	"severity":  true,
	"alertname": true,
	"status":    true,
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

	// orgIDs is the authorized customer/org set (the security gate from
	// resolveOrgScope). Alerts for those orgs live in their reseller tenants, so
	// query the distinct tenants, then keep only alerts whose organization_id
	// label is in scope — the isolation boundary now that many customers share
	// one tenant.
	tenants := distinctTenants(orgIDs)
	all, warnings := fanOutMimirAlerts(c.Request.Context(), tenants)
	all = filterByOrgScope(all, orgIDs)

	all = filterAlerts(all, alertFilter{
		statuses:   c.QueryArray("status"),
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
	attachAlertAssignments(pageAlerts)

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
		case "status":
			ai := statusOf(alerts[i])
			aj := statusOf(alerts[j])
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

// statusOf returns the Alertmanager status.state of an active alert
// ("active", "suppressed", "unprocessed"). Named after the public query
// param (`?status=`) and the sort column (`sort_by=status`) rather than
// the underlying JSON field; the alert payload still nests it under
// `.status.state` because that's the upstream Alertmanager shape.
func statusOf(alert map[string]interface{}) string {
	status, _ := alert["status"].(map[string]interface{})
	s, _ := status["state"].(string)
	return s
}

// alertFingerprintPattern restricts the fingerprint path param to safe chars.
// Alertmanager fingerprints are 16-char lowercase hex but we allow a slightly
// looser charset to accommodate test fixtures and any future format change.
var alertFingerprintPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

// GetAlertActivity handles GET /api/alerts/activity/:fingerprint
// Returns the per-alert audit timeline (silence created/updated/removed) for
// the alert identified by fingerprint within the resolved tenant. Most recent
// first. Operator notes are stored as the comment of the silence the action
// produced, so the timeline is the source of truth for "what happened, when,
// by whom". Literal `activity` precedes `:fingerprint` so the route does not
// collide with /alerts/silences/{silence_id}.
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

// CreateAlertNote handles POST /api/alerts/notes
// Appends a free-form operator note ({ fingerprint, text }) to the alert's
// activity timeline as a note_added event. Notes are independent from silences
// and assignments; no Mimir lookup is performed so a note can also be attached
// to an alert that already resolved (the timeline outlives the alert).
func CreateAlertNote(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}
	orgID, ok := resolveOrgID(c, user)
	if !ok {
		return
	}
	if !requireOrgID(c, orgID) {
		return
	}

	var req models.CreateAlertNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
			return
		}
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body: "+err.Error(), nil))
		return
	}
	if !alertFingerprintPattern.MatchString(req.Fingerprint) {
		c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", []response.ValidationError{
			{Key: "fingerprint", Message: "invalid_format", Value: req.Fingerprint},
		}))
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", []response.ValidationError{
			{Key: "text", Message: "required", Value: req.Text},
		}))
		return
	}

	// Unlike the audit logging around silence operations, here the timeline
	// write IS the primary action, so a failure fails the request.
	repo := entities.NewLocalAlertActivityRepository()
	if err := repo.Log(orgID, req.Fingerprint, entities.AlertActivityNoteAdded,
		alertActivityActorID(user), alertActivityActorName(user), "",
		map[string]interface{}{"text": text}); err != nil {
		logger.Error().Err(err).Str("org_id", orgID).Str("fingerprint", req.Fingerprint).Msg("failed to add alert note")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to add alert note", nil))
		return
	}
	c.JSON(http.StatusOK, response.OK("alert note added successfully", nil))
}

// CreateAlertAssignment handles POST /api/alerts/assignment
// Self-assigns the active alert identified by { fingerprint } to the caller.
// If the alert is already assigned to someone else the row is replaced
// (takeover) — the "do you want to take over from X?" confirmation is
// frontend-only by design. There is no unassign endpoint: assignments are
// auto-released by collect when the resolved webhook for the fingerprint
// arrives, which is also why the alert must still be firing in Mimir here
// (assigning a resolved alert would create a row nothing ever cleans up).
func CreateAlertAssignment(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}
	orgID, ok := resolveOrgID(c, user)
	if !ok {
		return
	}
	if !requireOrgID(c, orgID) {
		return
	}

	var req models.CreateAlertAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
			return
		}
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body: "+err.Error(), nil))
		return
	}
	if !alertFingerprintPattern.MatchString(req.Fingerprint) {
		c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", []response.ValidationError{
			{Key: "fingerprint", Message: "invalid_format", Value: req.Fingerprint},
		}))
		return
	}

	if _, ok := resolveActiveAlertForOrg(c, orgID, req.Fingerprint); !ok {
		return
	}

	repo := entities.NewLocalAlertAssignmentRepository()
	previousName, assignment, err := repo.Upsert(orgID, req.Fingerprint,
		alertActivityActorID(user), alertActivityActorName(user),
		user.OrganizationID, user.OrganizationName)
	if err != nil {
		logger.Error().Err(err).Str("org_id", orgID).Str("fingerprint", req.Fingerprint).Msg("failed to assign alert")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to assign alert", nil))
		return
	}

	details := map[string]interface{}{}
	if previousName != "" {
		details["reassigned_from"] = previousName
	}
	logAlertActivity(c, orgID, req.Fingerprint, entities.AlertActivityAssigned, user, "", details)

	// Optional note taken together with the assignment ("taking this, checking
	// the firewall"). Recorded as its own note_added event so the timeline
	// renders it exactly like a note posted via /alerts/notes. Same best-effort
	// policy as the assigned event: the assignment is already persisted, so a
	// failed note write logs a warning instead of failing the request.
	if note := strings.TrimSpace(req.Note); note != "" {
		logAlertActivity(c, orgID, req.Fingerprint, entities.AlertActivityNoteAdded, user, "",
			map[string]interface{}{"text": note})
	}

	c.JSON(http.StatusOK, response.OK("alert assigned successfully", gin.H{
		"assignment": assignment,
	}))
}

// resolveActiveAlertForOrg looks up an active alert by fingerprint inside the
// Mimir tenant that owns orgID's alerting (TenantForOrg: customers live in
// their managing reseller/distributor tenant) and verifies the alert actually
// belongs to orgID via its organization_id label. Reseller tenants hold many
// customers' alerts together, so the label check is what stops a caller
// authorized for org A from acting on org B's alert in the same tenant.
//
// Returns (alert, true) on success. On any failure the response is already
// written and the caller must just return.
func resolveActiveAlertForOrg(c *gin.Context, orgID, fingerprint string) (*models.ActiveAlert, bool) {
	tenant, err := alerting.TenantForOrg(orgID)
	if err != nil || tenant == "" {
		tenant = orgID
	}
	body, err := alerting.GetAlerts(tenant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch alerts from mimir: "+err.Error(), nil))
		return nil, false
	}
	var alerts []models.ActiveAlert
	if err := json.Unmarshal(body, &alerts); err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to parse alerts from mimir: "+err.Error(), nil))
		return nil, false
	}
	for i := range alerts {
		if alerts[i].Fingerprint == fingerprint && alerts[i].Labels["organization_id"] == orgID {
			return &alerts[i], true
		}
	}
	c.JSON(http.StatusNotFound, response.NotFound("alert not found", nil))
	return nil, false
}

// alertActivityActorID returns the stable user identifier stored in
// alert_activity.actor_user_id and alert_assignments.assigned_user_id:
// the Logto id when available, the local id otherwise.
func alertActivityActorID(user *models.User) string {
	if user == nil {
		return ""
	}
	if user.LogtoID != nil && *user.LogtoID != "" {
		return *user.LogtoID
	}
	return user.ID
}

// alertActivityActorName returns the display name recorded next to the actor
// id, falling back to username/email so the timeline never renders an empty
// actor.
func alertActivityActorName(user *models.User) string {
	if user == nil {
		return ""
	}
	if user.Name != "" {
		return user.Name
	}
	if user.Username != "" {
		return user.Username
	}
	return user.Email
}

// attachAlertAssignments decorates a page of Mimir alerts with the current
// assignee under the `assigned_to` key ({user_id, user_name, assigned_at} or
// null), resolved with a single batch query per page. Best-effort: on DB
// failure every alert renders as unassigned rather than failing the list.
func attachAlertAssignments(alerts []map[string]interface{}) {
	if len(alerts) == 0 {
		return
	}
	orgIDs := make([]string, 0, len(alerts))
	fingerprints := make([]string, 0, len(alerts))
	for _, alert := range alerts {
		alert["assigned_to"] = nil
		labels, _ := alert["labels"].(map[string]interface{})
		org, _ := labels["organization_id"].(string)
		fp, _ := alert["fingerprint"].(string)
		if org == "" || fp == "" {
			continue
		}
		orgIDs = append(orgIDs, org)
		fingerprints = append(fingerprints, fp)
	}

	repo := entities.NewLocalAlertAssignmentRepository()
	assignments, err := repo.GetByFingerprints(orgIDs, fingerprints)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to load alert assignments for list (non-fatal)")
		return
	}
	if len(assignments) == 0 {
		return
	}
	for _, alert := range alerts {
		labels, _ := alert["labels"].(map[string]interface{})
		org, _ := labels["organization_id"].(string)
		fp, _ := alert["fingerprint"].(string)
		if a, ok := assignments[entities.AssignmentKey(org, fp)]; ok {
			alert["assigned_to"] = gin.H{
				"user_id":       a.AssignedUserID,
				"user_name":     a.AssignedUserName,
				"user_org_id":   a.AssignedUserOrgID,
				"user_org_name": a.AssignedUserOrgName,
				"assigned_at":   a.AssignedAt,
			}
		}
	}
}

// fanOutMimirAlerts fetches active alerts from Mimir for every tenant in scope
// concurrently, with bounded concurrency and a global timeout. Per-tenant
// failures (timeout, 5xx, parse error) are collected as warnings; the rest of
// the result is returned.
//
// System identity (id, key, name, type) is carried as labels on the alert
// itself (system_id, system_key, system_name, system_type), stamped at ingest
// time by collect. No per-request DB lookup is needed here.
func fanOutMimirAlerts(parent context.Context, orgIDs []string) ([]map[string]interface{}, []string) {
	var (
		all      []map[string]interface{}
		warnings []string
		mu       sync.Mutex
		wg       sync.WaitGroup
	)
	ctx, cancel := context.WithTimeout(parent, mimirFanoutTimeout)
	defer cancel()
	sem := make(chan struct{}, mimirFanoutConcurrency)

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

			mu.Lock()
			all = append(all, alerts...)
			mu.Unlock()
		}(orgID)
	}
	wg.Wait()
	return all, warnings
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

// GetEffectiveAlertingConfig handles GET /api/alerts/config/effective: privileged per-layer + merged config + Mimir YAML for any tenant (organization_id required; nonexistent id → empty config), secrets redacted.
func GetEffectiveAlertingConfig(c *gin.Context) {
	if _, ok := helpers.GetUserFromContext(c); !ok {
		return
	}

	orgID := c.Query("organization_id")
	if !requireOrgID(c, orgID) {
		return
	}

	report, err := alerting.BuildEffectiveConfigReport(orgID)
	if err != nil {
		if errors.Is(err, alerting.ErrChainTooDeep) {
			c.JSON(http.StatusUnprocessableEntity, response.UnprocessableEntity("organization ancestor chain is too deep or contains a cycle", nil))
			return
		}
		logger.Error().Err(err).Str("org_id", orgID).Msg("failed to build effective alerting config report")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to build effective alerting config", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("effective alerting configuration retrieved successfully", alerting.RedactEffectiveConfigReport(report)))
}

// mimirFanoutTimeout caps how long cross-tenant Mimir fan-outs (alerts list,
// silences list) will wait. /alerts/totals no longer fans out: collect's
// AlertsTotalsRefresher pre-aggregates per-org counts into
// alerts_totals_by_org and this endpoint reads them with a single SUM.
const mimirFanoutTimeout = 10 * time.Second

// mimirFanoutConcurrency caps simultaneous in-flight Mimir requests per
// fan-out. Must stay <= the alerting httpClient's MaxConnsPerHost so we don't
// starve waiting for a free connection slot.
const mimirFanoutConcurrency = 50

// alertsTotalsStaleThreshold flags the /totals response when the oldest
// refreshed row is older than this. Surfaces "collect refresher is lagging
// or down" to the caller without blocking the response.
const alertsTotalsStaleThreshold = 5 * time.Minute

// GetAlertsTotals handles GET /api/alerts/totals
//
// Returns active alert counts by severity (and muted count) plus the total
// history count. Active counts come from alerts_totals_by_org, maintained by
// collect's AlertsTotalsRefresher cron — the endpoint is a single SQL SUM
// regardless of scope size, so it answers in <50ms even for the owner-wide
// dashboard view.
//
// Scope resolution follows the same three modes as the rest of /api/alerts/*:
// no organization_id → caller's full hierarchy; organization_id=X → single
// tenant X; +include=descendants → expand each org_id to its sub-tree.
// resolveOrgScope is the sole authorization gate; the SUM query filters by
// the returned list so a user only sees totals for orgs they can access.
//
// `warnings[]` carries non-fatal degradation messages (DB error on either
// table, or "stale data" when the refresher is lagging) so the UI can
// surface them without erroring out.
func GetAlertsTotals(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	orgIDs, ok := resolveOrgScope(c, user)
	if !ok {
		return
	}

	ownerAllScope := strings.ToLower(user.OrgRole) == "owner" && len(c.QueryArray("organization_id")) == 0

	warnings := []string{}

	totalsRepo := entities.NewLocalAlertsTotalsByOrgRepository()
	sum, err := totalsRepo.SumByOrgIDs(orgIDs)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to sum alerts totals from alerts_totals_by_org")
		warnings = append(warnings, fmt.Sprintf("totals: %s", err.Error()))
	} else if !sum.OldestUpdate.IsZero() && time.Since(sum.OldestUpdate) > alertsTotalsStaleThreshold {
		warnings = append(warnings, fmt.Sprintf("totals: stale data, oldest refresh %s ago", time.Since(sum.OldestUpdate).Truncate(time.Second)))
	}

	// For an Owner with no organization_id filter, the IN(...) variant
	// degenerates to "all rows" but with hundreds of placeholders. A bare
	// COUNT(*) on the table is much faster: Postgres can use the index-only
	// path on the visible rows without iterating the IN list.
	historyRepo := entities.NewLocalAlertHistoryRepository()
	var historyTotal int
	var historyErr error
	if ownerAllScope {
		historyTotal, historyErr = historyRepo.GetAlertHistoryTotals("")
	} else {
		historyTotal, historyErr = historyRepo.GetAlertHistoryTotalsByOrgIDs(orgIDs)
	}
	if historyErr != nil {
		logger.Warn().Err(historyErr).Msg("failed to count alert history for totals")
		warnings = append(warnings, fmt.Sprintf("history: %s", historyErr.Error()))
	}

	c.JSON(http.StatusOK, response.OK("alert totals retrieved successfully", gin.H{
		"active":   sum.Active,
		"critical": sum.Critical,
		"warning":  sum.Warning,
		"info":     sum.Info,
		"muted":    sum.Muted,
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
	statuses   []string
	severities []string
	systemKeys []string
	alertnames []string
}

// filterAlerts applies optional multi-value query filters to the alerts list.
// An alert is excluded when a requested filter's target label/field is missing
// or does not match any of the requested values; this prevents silent leakage
// of unrelated alerts when the caller narrows the query.
func filterAlerts(alerts []map[string]interface{}, f alertFilter) []map[string]interface{} {
	if len(f.statuses) == 0 && len(f.severities) == 0 && len(f.systemKeys) == 0 && len(f.alertnames) == 0 {
		return alerts
	}

	filtered := make([]map[string]interface{}, 0, len(alerts))
	for _, alert := range alerts {
		if len(f.statuses) > 0 {
			status, ok := alert["status"].(map[string]interface{})
			if !ok {
				continue
			}
			state, ok := status["state"].(string)
			if !ok || !slices.Contains(f.statuses, state) {
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

// systemActivityOrgID returns the org the system belongs to (the customer),
// which is the namespace alert_activity and alert_assignments are keyed by:
// it matches the organization_id label stamped on the system's alerts, i.e.
// what the timeline is read back with. Not to be confused with
// getSystemAlertOrgID, which maps further to the Mimir tenant (the managing
// reseller) and must only be used for Mimir API calls.
func systemActivityOrgID(system *models.System) string {
	orgID := system.Organization.LogtoID
	if orgID == "" {
		orgID = system.Organization.ID
	}
	return orgID
}

func getSystemAlertOrgID(system *models.System) string {
	orgID := systemActivityOrgID(system)
	// The Mimir tenant is the managing reseller, not the customer org: systems
	// of many customers share one reseller tenant. Fall back to the owning org
	// on resolution failure so a transient DB error never breaks the alert path.
	tenant, err := alerting.TenantForOrg(orgID)
	if err != nil || tenant == "" {
		return orgID
	}
	return tenant
}

// distinctTenants maps each authorized customer/org id to its Mimir tenant
// (reseller) via alerting.TenantForOrg and deduplicates. On resolution error it
// falls back to the org id itself so the caller still sees its own tenant.
func distinctTenants(orgIDs []string) []string {
	seen := make(map[string]struct{}, len(orgIDs))
	tenants := make([]string, 0, len(orgIDs))
	for _, oid := range orgIDs {
		t, err := alerting.TenantForOrg(oid)
		if err != nil || t == "" {
			t = oid
		}
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			tenants = append(tenants, t)
		}
	}
	return tenants
}

// filterByOrgScope keeps only alerts whose organization_id label is in the
// authorized set. Reseller tenants hold many customers' alerts together, so
// this application-level filter is the isolation boundary — security-critical
// when a customer logs in directly and must see only its own systems.
func filterByOrgScope(alerts []map[string]interface{}, allowed []string) []map[string]interface{} {
	allow := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		allow[o] = struct{}{}
	}
	filtered := make([]map[string]interface{}, 0, len(alerts))
	for _, alert := range alerts {
		labels, _ := alert["labels"].(map[string]interface{})
		orgID, _ := labels["organization_id"].(string)
		if _, ok := allow[orgID]; ok {
			filtered = append(filtered, alert)
		}
	}
	return filtered
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

// parseFutureEndAt parses an RFC3339 end_at and asserts it is in the future.
// Returns (endsAtUTC, true) on success. On failure it writes a structured
// validation_error response and returns (zero, false); the caller must just
// return. An empty endAt is treated as "not set" (success with zero time) so
// create paths can fall back to duration_minutes.
func parseFutureEndAt(c *gin.Context, endAt string, now time.Time) (time.Time, bool) {
	if endAt == "" {
		return time.Time{}, true
	}
	parsed, err := time.Parse(time.RFC3339, endAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", []response.ValidationError{
			{Key: "end_at", Message: "invalid_format", Value: endAt},
		}))
		return time.Time{}, false
	}
	if !parsed.After(now) {
		c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", []response.ValidationError{
			{Key: "end_at", Message: "must_be_future", Value: endAt},
		}))
		return time.Time{}, false
	}
	return parsed.UTC(), true
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
// Returns active alerts from Mimir scoped to a single system, with the same
// filters, pagination, and sorting surface as the cross-system /api/alerts
// list. The system_key in the URL acts as a hard scope: the multi-value
// `system_key` query filter that /api/alerts accepts is not exposed here
// since it would either be redundant (same value) or rejected.
//
// Accepted query params (all optional):
//   - severity, alertname, status (multi-value, OR within, AND across)
//   - page, page_size (default 50, cap 100)
//   - sort_by (starts_at | severity | alertname | status), default starts_at
//   - sort_direction (asc | desc), default desc
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

	page, pageSize := helpers.GetPaginationFromQuery(c)
	if c.Query("page_size") == "" {
		pageSize = alertsListDefaultPageSize
	}

	sortBy, sortDirection := helpers.GetSortingFromQuery(c)
	if !alertsListAllowedSortBy[sortBy] {
		sortBy = "starts_at"
	}
	if c.Query("sort_direction") == "" {
		sortDirection = "desc"
	}

	body, err := alerting.GetAlerts(getSystemAlertOrgID(system))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch alerts from mimir: "+err.Error(), nil))
		return
	}

	var alerts []map[string]interface{}
	if err := json.Unmarshal(body, &alerts); err != nil {
		c.JSON(http.StatusOK, response.OK("alerts retrieved successfully", gin.H{
			"alerts":     []map[string]interface{}{},
			"pagination": helpers.BuildPaginationInfoWithSorting(page, pageSize, 0, sortBy, sortDirection),
		}))
		return
	}

	// Hard scope by system_key. System identity is carried as labels on the
	// alert (system_id, system_key, system_name, system_type) stamped at
	// ingest time by collect, so no DB join is needed.
	scoped := make([]map[string]interface{}, 0, len(alerts))
	for _, alert := range alerts {
		labels, _ := alert["labels"].(map[string]interface{})
		if sk, ok := labels["system_key"].(string); ok && sk == system.SystemKey {
			scoped = append(scoped, alert)
		}
	}

	filtered := filterAlerts(scoped, alertFilter{
		statuses:   c.QueryArray("status"),
		severities: c.QueryArray("severity"),
		alertnames: c.QueryArray("alertname"),
		// systemKeys intentionally omitted: the URL path is the source of truth.
	})

	sortAlertsList(filtered, sortBy, sortDirection)

	totalCount := len(filtered)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > totalCount {
		start = totalCount
	}
	if end > totalCount {
		end = totalCount
	}
	pageAlerts := filtered[start:end]
	if pageAlerts == nil {
		pageAlerts = []map[string]interface{}{}
	}
	attachAlertAssignments(pageAlerts)

	c.JSON(http.StatusOK, response.OK("alerts retrieved successfully", gin.H{
		"alerts":     pageAlerts,
		"pagination": helpers.BuildPaginationInfoWithSorting(page, pageSize, totalCount, sortBy, sortDirection),
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
		if _, ok := err.(validator.ValidationErrors); ok {
			c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
			return
		}
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
	endsAt, ok := parseFutureEndAt(c, req.EndAt, now)
	if !ok {
		return
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
	// of truth; activity is a denormalised UX convenience. Keyed by the
	// system's org (the alert's organization_id label), NOT the Mimir tenant,
	// or the event would never show up when the timeline is read back.
	logAlertActivity(c, systemActivityOrgID(system), req.Fingerprint, entities.AlertActivitySilenced, user, silenceResp.SilenceID, map[string]interface{}{
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
	// Activity is keyed by the system's org, not the Mimir tenant.
	activityOrg := systemActivityOrgID(system)
	activityRepo := entities.NewLocalAlertActivityRepository()
	fingerprint, _ := activityRepo.FindFingerprintBySilenceID(activityOrg, silenceID)
	logAlertActivity(c, activityOrg, fingerprint, entities.AlertActivityUnsilenced, user, silenceID, nil)

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
		if _, ok := err.(validator.ValidationErrors); ok {
			c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
			return
		}
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body: "+err.Error(), nil))
		return
	}

	now := time.Now().UTC()
	endsAt, ok := parseFutureEndAt(c, req.EndAt, now)
	if !ok {
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
	// update lands on the right alert's timeline. Activity is keyed by the
	// system's org, not the Mimir tenant.
	activityOrg := systemActivityOrgID(system)
	activityRepo := entities.NewLocalAlertActivityRepository()
	fingerprint, _ := activityRepo.FindFingerprintBySilenceID(activityOrg, silenceID)
	logAlertActivity(c, activityOrg, fingerprint, entities.AlertActivitySilenceUpdated, user, silenceResp.SilenceID, map[string]interface{}{
		"comment": normalizeAlertSilenceComment(req.Comment),
		"end_at":  req.EndAt,
	})

	c.JSON(http.StatusOK, response.OK("silence updated successfully", gin.H{
		"silence_id": silenceResp.SilenceID,
	}))
}

// systemKeyFromSilence returns the `system_key` matcher value of a silence,
// or "" if the silence has no exact (non-regex) system_key matcher. The
// cross-system silence endpoints use it to scope operations to silences that
// were created against a specific system (the only ones our UI ever creates)
// and to ignore generic Alertmanager silences that may exist for the tenant.
func systemKeyFromSilence(s *models.AlertmanagerSilence) string {
	if s == nil {
		return ""
	}
	for _, m := range s.Matchers {
		if m.Name == "system_key" && !m.IsRegex {
			return m.Value
		}
	}
	return ""
}

// orgIDFromSilence returns the `organization_id` matcher value of a silence, or
// "" if absent. In the reseller-tenant model a silence's owning customer is
// identified by this matcher (buildSystemAlertSilenceRequest always sets it),
// so cross-hierarchy reads can attribute and scope each silence to a customer.
func orgIDFromSilence(s *models.AlertmanagerSilence) string {
	if s == nil {
		return ""
	}
	for _, m := range s.Matchers {
		if m.Name == "organization_id" && !m.IsRegex {
			return m.Value
		}
	}
	return ""
}

// resolveAlertSilenceContext looks up an active alert by fingerprint inside
// a single tenant's Mimir and returns the alert plus the `system_key` label
// the cross-system silence handler needs to attach as a matcher. Used by
// POST /api/alerts/silences; not used by the per-system silence handlers
// (those already know the system_key from the URL path).
//
// Returns (alert, systemKey, true) on success. On any failure the response
// is already written and the caller must just return.
func resolveAlertSilenceContext(c *gin.Context, orgID, fingerprint string) (*models.ActiveAlert, string, bool) {
	body, err := alerting.GetAlerts(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch alerts from mimir: "+err.Error(), nil))
		return nil, "", false
	}
	var alerts []models.ActiveAlert
	if err := json.Unmarshal(body, &alerts); err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to parse alerts from mimir: "+err.Error(), nil))
		return nil, "", false
	}
	for i := range alerts {
		if alerts[i].Fingerprint != fingerprint {
			continue
		}
		systemKey := alerts[i].Labels["system_key"]
		if systemKey == "" {
			c.JSON(http.StatusBadRequest, response.BadRequest("alert is not system-scoped (missing system_key label)", nil))
			return nil, "", false
		}
		return &alerts[i], systemKey, true
	}
	c.JSON(http.StatusNotFound, response.NotFound("alert not found", nil))
	return nil, "", false
}

// CreateAlertSilence handles POST /api/alerts/silences
// Cross-system mute: body is { fingerprint, end_at, comment, duration_minutes? }.
// The tenant comes from ?organization_id= (mandatory for non-Owner) and the
// system_key matcher is resolved from the alert's labels in Mimir. The actual
// silence creation reuses the same buildSystemAlertSilenceRequest +
// alerting.CreateSilence path used by the per-system endpoint, so the silence
// object stored in Mimir is byte-identical regardless of which route created it.
func CreateAlertSilence(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}
	orgID, ok := resolveOrgID(c, user)
	if !ok {
		return
	}
	if !requireOrgID(c, orgID) {
		return
	}

	var req models.CreateSystemAlertSilenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
			return
		}
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body: "+err.Error(), nil))
		return
	}
	if !alertFingerprintPattern.MatchString(req.Fingerprint) {
		c.JSON(http.StatusBadRequest, response.ValidationFailed("validation failed", []response.ValidationError{
			{Key: "fingerprint", Message: "invalid_format", Value: req.Fingerprint},
		}))
		return
	}

	alert, systemKey, ok := resolveAlertSilenceContext(c, orgID, req.Fingerprint)
	if !ok {
		return
	}
	if len(alert.Status.SilencedBy) > 0 {
		c.JSON(http.StatusBadRequest, response.BadRequest("alert is already silenced", nil))
		return
	}

	now := time.Now().UTC()
	endsAt, ok := parseFutureEndAt(c, req.EndAt, now)
	if !ok {
		return
	}

	silenceReq := buildSystemAlertSilenceRequest(
		alert,
		systemKey,
		getAlertSilenceCreatedBy(user),
		req.Comment,
		req.DurationMinutes,
		now,
		endsAt,
	)

	silenceResp, err := alerting.CreateSilence(orgID, silenceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create silence in mimir: "+err.Error(), nil))
		return
	}

	logAlertActivity(c, orgID, req.Fingerprint, entities.AlertActivitySilenced, user, silenceResp.SilenceID, map[string]interface{}{
		"comment":          normalizeAlertSilenceComment(req.Comment),
		"duration_minutes": req.DurationMinutes,
		"end_at":           req.EndAt,
	})

	c.JSON(http.StatusOK, response.OK("alert silenced successfully", gin.H{
		"silence_id": silenceResp.SilenceID,
	}))
}

// alertSilenceWithOrg is the per-row payload returned by GET /api/alerts/silences.
// We extend AlertmanagerSilence with the originating organization_id (Mimir
// stores silences per-tenant, so this isn't on the silence object itself) and
// with the system_key extracted from the matchers, so the FE can render the
// "muted on system X" pill without re-parsing matchers.
type alertSilenceWithOrg struct {
	models.AlertmanagerSilence
	OrganizationID string `json:"organization_id"`
	SystemKey      string `json:"system_key"`
}

// GetAlertSilences handles GET /api/alerts/silences
// Cross-hierarchy list of active and pending silences. Scope follows the same
// three modes as /api/alerts/totals (no organization_id / single tenant /
// descendants). Only system-scoped silences (those with a `system_key` matcher)
// are returned; generic Alertmanager silences are filtered out because they
// don't belong to our domain model. Expired silences are also excluded.
//
// Per-tenant fan-out failures are non-fatal and surface in `warnings`.
//
// Optional filters: `system_key` (multi-value, exact match on matcher value).
func GetAlertSilences(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}
	orgIDs, ok := resolveOrgScope(c, user)
	if !ok {
		return
	}

	systemKeyFilter := c.QueryArray("system_key")

	// Silences for the authorized customer orgs live in their reseller tenants;
	// query the distinct tenants and attribute each silence back to its owning
	// customer via the organization_id matcher, dropping any outside scope.
	allow := make(map[string]struct{}, len(orgIDs))
	for _, o := range orgIDs {
		allow[o] = struct{}{}
	}
	tenants := distinctTenants(orgIDs)

	var (
		out      []alertSilenceWithOrg
		warnings []string
		mu       sync.Mutex
		wg       sync.WaitGroup
	)
	ctx, cancel := context.WithTimeout(c.Request.Context(), mimirFanoutTimeout)
	defer cancel()
	sem := make(chan struct{}, mimirFanoutConcurrency)

	for _, orgID := range tenants {
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

			silences, err := alerting.GetSilences(orgID)
			if err != nil {
				logger.Warn().Err(err).Str("org_id", orgID).Msg("failed to fetch silences from mimir for cross-system list")
				mu.Lock()
				warnings = append(warnings, fmt.Sprintf("org %s: %s", orgID, err.Error()))
				mu.Unlock()
				return
			}

			local := make([]alertSilenceWithOrg, 0, len(silences))
			for i := range silences {
				if silences[i].Status != nil && silences[i].Status.State == "expired" {
					continue
				}
				sk := systemKeyFromSilence(&silences[i])
				if sk == "" {
					continue
				}
				if len(systemKeyFilter) > 0 && !slices.Contains(systemKeyFilter, sk) {
					continue
				}
				// Attribute to the owning customer and drop out-of-scope silences:
				// the reseller tenant holds every customer's silences together.
				ownerOrg := orgIDFromSilence(&silences[i])
				if ownerOrg == "" {
					ownerOrg = orgID
				}
				if _, allowed := allow[ownerOrg]; !allowed {
					continue
				}
				local = append(local, alertSilenceWithOrg{
					AlertmanagerSilence: silences[i],
					OrganizationID:      ownerOrg,
					SystemKey:           sk,
				})
			}

			if len(local) == 0 {
				return
			}
			mu.Lock()
			out = append(out, local...)
			mu.Unlock()
		}(orgID)
	}
	wg.Wait()

	if out == nil {
		out = []alertSilenceWithOrg{}
	}
	if warnings == nil {
		warnings = []string{}
	}

	c.JSON(http.StatusOK, response.OK("silences retrieved successfully", gin.H{
		"silences": out,
		"warnings": warnings,
	}))
}

// GetAlertSilence handles GET /api/alerts/silences/:silence_id
// Single-silence read across the caller's scope. The tenant is resolved from
// ?organization_id= (mandatory for non-Owner). Refuses to return silences
// that aren't system-scoped (no `system_key` matcher) — those don't belong to
// our domain and a generic 404 keeps the surface tight.
func GetAlertSilence(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}
	orgID, ok := resolveOrgID(c, user)
	if !ok {
		return
	}
	if !requireOrgID(c, orgID) {
		return
	}
	silenceID := c.Param("silence_id")
	if silenceID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("silence id required", nil))
		return
	}

	silence, err := alerting.GetSilence(orgID, silenceID)
	if errors.Is(err, alerting.ErrSilenceNotFound) {
		c.JSON(http.StatusNotFound, response.NotFound("silence not found", nil))
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch silence from mimir: "+err.Error(), nil))
		return
	}
	systemKey := systemKeyFromSilence(silence)
	if systemKey == "" {
		c.JSON(http.StatusNotFound, response.NotFound("silence not found", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("silence retrieved successfully", gin.H{
		"silence": alertSilenceWithOrg{
			AlertmanagerSilence: *silence,
			OrganizationID:      orgID,
			SystemKey:           systemKey,
		},
	}))
}

// UpdateAlertSilence handles PUT /api/alerts/silences/:silence_id
// Cross-system silence edit (change end time / comment). Mirrors
// UpdateSystemAlertSilence but discovers system_key from the silence matchers
// instead of taking it from the URL.
func UpdateAlertSilence(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}
	orgID, ok := resolveOrgID(c, user)
	if !ok {
		return
	}
	if !requireOrgID(c, orgID) {
		return
	}
	silenceID := c.Param("silence_id")
	if silenceID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("silence id required", nil))
		return
	}

	var req models.UpdateSystemAlertSilenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
			return
		}
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body: "+err.Error(), nil))
		return
	}

	now := time.Now().UTC()
	endsAt, ok := parseFutureEndAt(c, req.EndAt, now)
	if !ok {
		return
	}

	existing, err := alerting.GetSilence(orgID, silenceID)
	if errors.Is(err, alerting.ErrSilenceNotFound) {
		c.JSON(http.StatusNotFound, response.NotFound("silence not found", nil))
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch silence from mimir: "+err.Error(), nil))
		return
	}
	if systemKeyFromSilence(existing) == "" {
		c.JSON(http.StatusNotFound, response.NotFound("silence not found", nil))
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

// DeleteAlertSilence handles DELETE /api/alerts/silences/:silence_id
// Cross-system unmute. Same ownership rule as UpdateAlertSilence: only
// silences carrying a `system_key` matcher are addressable through this
// endpoint, so generic Alertmanager silences cannot be removed via the
// public API.
func DeleteAlertSilence(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}
	orgID, ok := resolveOrgID(c, user)
	if !ok {
		return
	}
	if !requireOrgID(c, orgID) {
		return
	}
	silenceID := c.Param("silence_id")
	if silenceID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("silence id required", nil))
		return
	}

	silence, err := alerting.GetSilence(orgID, silenceID)
	if errors.Is(err, alerting.ErrSilenceNotFound) {
		c.JSON(http.StatusNotFound, response.NotFound("silence not found", nil))
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch silence from mimir: "+err.Error(), nil))
		return
	}
	if systemKeyFromSilence(silence) == "" {
		c.JSON(http.StatusNotFound, response.NotFound("silence not found", nil))
		return
	}

	if err := alerting.DeleteSilence(orgID, silenceID); errors.Is(err, alerting.ErrSilenceNotFound) {
		c.JSON(http.StatusNotFound, response.NotFound("silence not found", nil))
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to delete silence in mimir: "+err.Error(), nil))
		return
	}

	activityRepo := entities.NewLocalAlertActivityRepository()
	fingerprint, _ := activityRepo.FindFingerprintBySilenceID(orgID, silenceID)
	logAlertActivity(c, orgID, fingerprint, entities.AlertActivityUnsilenced, user, silenceID, nil)

	c.JSON(http.StatusOK, response.OK("silence disabled successfully", nil))
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
	// Natural defaults for history are "by ingest time, most recent first".
	// The shared helpers default sort_by to "" and sort_direction to asc,
	// which (a) hide what the repo actually sorts on from the pagination
	// response and (b) would surface the oldest events first.
	if c.Query("sort_by") == "" {
		sortBy = "created_at"
	}
	if c.Query("sort_direction") == "" {
		sortDirection = "desc"
	}

	from, to, perr := parseDateRange(c)
	if perr != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(perr.Error(), nil))
		return
	}

	repo := entities.NewLocalAlertHistoryRepository()
	records, totalCount, err := repo.QueryAlertHistory(entities.AlertHistoryQuery{
		OrgIDs:        []string{system.Organization.LogtoID},
		SystemKeys:    []string{system.SystemKey},
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
	// See GetSystemAlertHistory for the rationale: default sort_by to
	// created_at so the pagination response reflects what the repo actually
	// sorts on, and flip the direction default to desc ("most recent first").
	if c.Query("sort_by") == "" {
		sortBy = "created_at"
	}
	if c.Query("sort_direction") == "" {
		sortDirection = "desc"
	}

	from, to, perr := parseDateRange(c)
	if perr != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(perr.Error(), nil))
		return
	}

	repo := entities.NewLocalAlertHistoryRepository()
	records, totalCount, err := repo.QueryAlertHistory(entities.AlertHistoryQuery{
		OrgIDs:        orgIDs,
		SystemKeys:    c.QueryArray("system_key"),
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

// asAlertingFieldError extracts a *models.AlertingFieldError from any error
// in the chain. Returns nil when the error is not a structured validation
// failure (so the caller can fall back to a generic 4xx/5xx response).
func asAlertingFieldError(err error) *models.AlertingFieldError {
	var fe *models.AlertingFieldError
	if errors.As(err, &fe) {
		return fe
	}
	return nil
}

// buildAlertingValidationErrors converts go-playground/validator errors into
// the standard ValidationError slice, preserving the full JSON path (e.g.
// `email_recipients.0.address`) so the UI can pin the failure to the exact
// input. Falls back to snake_case'd field name when the path can't be parsed.
func buildAlertingValidationErrors(verrs validator.ValidationErrors) []response.ValidationError {
	out := make([]response.ValidationError, 0, len(verrs))
	for _, ve := range verrs {
		key := alertingJSONPath(ve.Namespace())
		value := ""
		if v := ve.Value(); v != nil {
			value = fmt.Sprintf("%v", v)
		}
		out = append(out, response.ValidationError{
			Key:     key,
			Message: alertingValidatorTagToCode(ve.Tag()),
			Value:   value,
		})
	}
	return out
}

// alertingJSONPath turns a validator namespace like
// "AlertingConfigLayer.EmailRecipients[0].Address" into the JSON path
// "email_recipients.0.address". The first segment (the struct name) is
// dropped; array indices in brackets become dotted segments.
func alertingJSONPath(ns string) string {
	parts := strings.Split(ns, ".")
	if len(parts) == 0 {
		return ns
	}
	parts = parts[1:] // drop root struct name
	for i, p := range parts {
		// Split "EmailRecipients[0]" → "email_recipients.0"
		if idx := strings.Index(p, "["); idx >= 0 && strings.HasSuffix(p, "]") {
			field := alertingFieldToJSON(p[:idx])
			arrIdx := p[idx+1 : len(p)-1]
			parts[i] = field + "." + arrIdx
		} else {
			parts[i] = alertingFieldToJSON(p)
		}
	}
	return strings.Join(parts, ".")
}

// alertingFieldToJSON maps a PascalCase Go field name to the snake_case JSON
// name used in models.AlertingConfigLayer and its nested types. Covers every
// field on the layer surface; unknown names fall back to a generic snake_case
// conversion so future additions don't silently produce a wrong key.
func alertingFieldToJSON(field string) string {
	switch field {
	case "EmailRecipients":
		return "email_recipients"
	case "WebhookRecipients":
		return "webhook_recipients"
	case "TelegramRecipients":
		return "telegram_recipients"
	case "Address":
		return "address"
	case "Severities":
		return "severities"
	case "Language":
		return "language"
	case "Format":
		return "format"
	case "Name":
		return "name"
	case "URL":
		return "url"
	case "BotToken":
		return "bot_token"
	case "ChatID":
		return "chat_id"
	case "Enabled":
		return "enabled"
	case "Email":
		return "email"
	case "Webhook":
		return "webhook"
	case "Telegram":
		return "telegram"
	}
	var b strings.Builder
	for i, r := range field {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteRune('_')
		}
		if r >= 'A' && r <= 'Z' {
			r = r + ('a' - 'A')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// alertingValidatorTagToCode maps go-playground/validator tag names to the
// stable codes the UI consumes. "email" / "url" become "invalid_format" so
// the UI doesn't have to know the underlying validator; "oneof" becomes
// "invalid_value". Everything else (required, max, min, len, ...) passes
// through as-is to match what `ValidationBadRequestMultiple` produces for
// the rest of the backend.
func alertingValidatorTagToCode(tag string) string {
	switch tag {
	case "email", "url":
		return "invalid_format"
	case "oneof":
		return "invalid_value"
	}
	return tag
}

// validateWebhookRecipients enforces that every webhook URL is a plain http/https
// URL pointing to a publicly-routable host. This protects Mimir's Alertmanager
// (which dispatches alert payloads from inside the internal network) from being
// abused as a blind SSRF relay to loopback, metadata, or private-range hosts.
//
// On failure returns a *models.AlertingFieldError pinned to
// `webhook_recipients.<i>.url` so the handler can surface a structured 400.
func validateWebhookRecipients(recipients []models.WebhookRecipient) error {
	for i, r := range recipients {
		if code, err := validateWebhookURL(r.URL); err != nil {
			return &models.AlertingFieldError{
				Key:   fmt.Sprintf("webhook_recipients.%d.url", i),
				Code:  code,
				Value: r.URL,
			}
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

// validateWebhookURL returns (code, error). On success both are zero. On
// failure code is a stable machine token for the UI ("invalid_format",
// "invalid_scheme", "host_not_allowed", ...) and error preserves the human
// message for logs and tests.
func validateWebhookURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "invalid_format", fmt.Errorf("invalid webhook url: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "invalid_scheme", fmt.Errorf("webhook url must use http or https, got %q", u.Scheme)
	}
	if u.User != nil {
		return "credentials_not_allowed", fmt.Errorf("webhook url must not contain credentials")
	}
	host := u.Hostname()
	if host == "" {
		return "missing_host", fmt.Errorf("webhook url is missing a host")
	}

	hostLower := strings.ToLower(host)
	for _, prefix := range webhookHostDenylist {
		if hostLower == strings.TrimSuffix(prefix, ".") || strings.HasPrefix(hostLower, prefix) {
			return "host_not_allowed", fmt.Errorf("webhook url host %q is not allowed (loopback, metadata, or private network)", host)
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
			return "invalid_host_format", fmt.Errorf("webhook url host %q is not a canonical hostname or IP literal", host)
		}
		if !strings.ContainsAny(host, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			return "invalid_host_format", fmt.Errorf("webhook url host %q must be either an IP literal or a hostname with at least one letter", host)
		}
	}

	// IP literal path: reject loopback / private / link-local / multicast /
	// unspecified / CGNAT. Handles IPv6-mapped IPv4 via ip.To4() unmasking.
	if parsedIP != nil {
		if err := rejectNonPublicIP(parsedIP); err != nil {
			return "host_not_publicly_routable", fmt.Errorf("webhook url host %q: %w", host, err)
		}
		return "", nil
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
		return "dns_resolution_failed", fmt.Errorf("webhook url host %q: dns resolution failed: %w", host, err)
	}
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			return "host_not_publicly_routable", fmt.Errorf("webhook url host %q resolved to unparseable address %q", host, addr)
		}
		if err := rejectNonPublicIP(ip); err != nil {
			return "host_not_publicly_routable", fmt.Errorf("webhook url host %q resolved to non-public address %s: %w", host, addr, err)
		}
	}

	return "", nil
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
