/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/configuration"
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

// resolveOrgID extracts and validates the target organization ID for alerting operations.
// Owner/Distributor/Reseller must pass organization_id query param; Customer uses their own.
// Validates hierarchical access via IsOrganizationInHierarchy.
func resolveOrgID(c *gin.Context, user *models.User) (string, bool) {
	orgID := c.Query("organization_id")
	orgRole := strings.ToLower(user.OrgRole)

	if orgRole == "customer" {
		return user.OrganizationID, true
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

// ConfigureAlerts handles POST /api/alerts/config
func ConfigureAlerts(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	orgID, ok := resolveOrgID(c, user)
	if !ok {
		return
	}

	var req models.AlertingConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body: "+err.Error(), nil))
		return
	}

	// Validate severity values
	validSeverities := map[string]bool{"critical": true, "warning": true, "info": true}
	for _, severity := range req.Severities {
		if !validSeverities[severity.Severity] {
			c.JSON(http.StatusBadRequest, response.BadRequest("invalid severity level: "+severity.Severity+". allowed: critical, warning, info", nil))
			return
		}
	}

	// Validate email template language
	if req.EmailTemplateLang != "" {
		valid := false
		for _, lang := range alerting.ValidTemplateLangs {
			if req.EmailTemplateLang == lang {
				valid = true
				break
			}
		}
		if !valid {
			c.JSON(http.StatusBadRequest, response.BadRequest("invalid email_template_lang: allowed values are "+strings.Join(alerting.ValidTemplateLangs, ", "), nil))
			return
		}
	}

	cfg := configuration.Config
	yamlConfig, err := alerting.RenderConfig(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom, cfg.SMTPTLS,
		cfg.AlertingHistoryWebhookURL, cfg.AlertingHistoryWebhookToken,
		&req,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to render alertmanager config: "+err.Error(), nil))
		return
	}

	templateFiles, err := alerting.BuildTemplateFiles(req.EmailTemplateLang, cfg.AppURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to load alert email templates: "+err.Error(), nil))
		return
	}

	if err := alerting.PushConfig(orgID, yamlConfig, templateFiles); err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to push config to mimir: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("alerting configuration updated successfully", nil))
}

// DisableAlerts handles DELETE /api/alerts/config
func DisableAlerts(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	orgID, ok := resolveOrgID(c, user)
	if !ok {
		return
	}

	cfg := configuration.Config
	yamlConfig, err := alerting.RenderConfig(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom, cfg.SMTPTLS,
		cfg.AlertingHistoryWebhookURL, cfg.AlertingHistoryWebhookToken,
		nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to render blackhole config: "+err.Error(), nil))
		return
	}

	if err := alerting.PushConfig(orgID, yamlConfig, nil); err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to push config to mimir: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("all alerts disabled successfully", nil))
}

// GetAlerts handles GET /api/alerts
func GetAlerts(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	orgID, ok := resolveOrgID(c, user)
	if !ok {
		return
	}

	body, err := alerting.GetAlerts(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch alerts from mimir: "+err.Error(), nil))
		return
	}

	// Parse alerts for optional filtering
	var alerts []map[string]interface{}
	if err := json.Unmarshal(body, &alerts); err != nil {
		// Return raw response if parsing fails
		c.Data(http.StatusOK, "application/json", body)
		return
	}

	var params models.AlertQueryParams
	if err := c.ShouldBindQuery(&params); err == nil {
		alerts = filterAlerts(alerts, params)
	}

	c.JSON(http.StatusOK, response.OK("alerts retrieved successfully", gin.H{
		"alerts": alerts,
	}))
}

// GetAlertingConfig handles GET /api/alerts/config
// By default returns structured JSON parsed from Mimir YAML.
// Use ?format=yaml to get the raw (redacted) YAML.
func GetAlertingConfig(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	orgID, ok := resolveOrgID(c, user)
	if !ok {
		return
	}

	body, err := alerting.GetConfig(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch alerting config from mimir: "+err.Error(), nil))
		return
	}

	// No config exists for this tenant (Mimir 404 or empty body). Return null
	// so the frontend can show the "no configuration found" empty state.
	if len(body) == 0 {
		c.JSON(http.StatusOK, response.OK("alerting configuration retrieved successfully", gin.H{
			"config": nil,
		}))
		return
	}

	if c.Query("format") == "yaml" {
		c.JSON(http.StatusOK, response.OK("alerting configuration retrieved successfully", gin.H{
			"config": alerting.RedactSensitiveConfig(string(body)),
		}))
		return
	}

	cfg, err := alerting.ParseConfig(string(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to parse alerting config: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("alerting configuration retrieved successfully", gin.H{
		"config": cfg,
	}))
}

// GetAlertsTotals handles GET /api/alerts/totals
// Returns active alert counts by severity (from Mimir) and total history count (from DB).
func GetAlertsTotals(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	orgID, ok := resolveOrgID(c, user)
	if !ok {
		return
	}

	result := gin.H{
		"active":   0,
		"critical": 0,
		"warning":  0,
		"info":     0,
		"history":  0,
	}

	// Fetch active alerts from Mimir
	body, err := alerting.GetAlerts(orgID)
	if err != nil {
		logger.Warn().Err(err).Str("org_id", orgID).Msg("failed to fetch alerts from mimir for totals")
	} else {
		var alerts []map[string]interface{}
		if err := json.Unmarshal(body, &alerts); err == nil {
			result["active"] = len(alerts)
			for _, alert := range alerts {
				labels, _ := alert["labels"].(map[string]interface{})
				if sev, ok := labels["severity"].(string); ok {
					switch sev {
					case "critical":
						result["critical"] = result["critical"].(int) + 1
					case "warning":
						result["warning"] = result["warning"].(int) + 1
					case "info":
						result["info"] = result["info"].(int) + 1
					}
				}
			}
		}
	}

	// Fetch history total from DB
	repo := entities.NewLocalAlertHistoryRepository()
	historyTotal, err := repo.GetAlertHistoryTotals(strings.ToLower(user.OrgRole), user.OrganizationID)
	if err != nil {
		logger.Warn().Err(err).Str("org_id", orgID).Msg("failed to count alert history for totals")
	} else {
		result["history"] = historyTotal
	}

	c.JSON(http.StatusOK, response.OK("alert totals retrieved successfully", result))
}

// GetAlertsTrend handles GET /api/alerts/trend
// Returns trend data for resolved alerts over a specified period.
func GetAlertsTrend(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
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
	trend, err := repo.GetAlertHistoryTrend(period, strings.ToLower(user.OrgRole), user.OrganizationID)
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

// filterAlerts applies optional query filters to the alerts list
func filterAlerts(alerts []map[string]interface{}, params models.AlertQueryParams) []map[string]interface{} {
	if params.State == "" && params.Severity == "" && params.SystemKey == "" {
		return alerts
	}

	filtered := make([]map[string]interface{}, 0, len(alerts))
	for _, alert := range alerts {
		if params.State != "" {
			if status, ok := alert["status"].(map[string]interface{}); ok {
				if state, ok := status["state"].(string); ok && state != params.State {
					continue
				}
			}
		}

		labels, _ := alert["labels"].(map[string]interface{})

		if params.Severity != "" {
			if sev, ok := labels["severity"].(string); ok && sev != params.Severity {
				continue
			}
		}

		if params.SystemKey != "" {
			if sk, ok := labels["system_key"].(string); ok && sk != params.SystemKey {
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
) *models.AlertmanagerSilenceRequest {
	if durationMinutes <= 0 {
		durationMinutes = defaultSystemAlertSilenceDurationMinutes
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
	seen := make(map[string]struct{}, len(labelNames))
	for _, name := range labelNames {
		if _, found := seen[name]; found {
			continue
		}
		seen[name] = struct{}{}

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
		EndsAt:    now.Add(time.Duration(durationMinutes) * time.Minute).Format(time.RFC3339),
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

	// Filter alerts by this system's key
	filtered := make([]map[string]interface{}, 0, len(alerts))
	for _, alert := range alerts {
		labels, _ := alert["labels"].(map[string]interface{})
		if sk, ok := labels["system_key"].(string); ok && sk == system.SystemKey {
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

	silenceReq := buildSystemAlertSilenceRequest(
		alert,
		system.SystemKey,
		getAlertSilenceCreatedBy(user),
		req.Comment,
		req.DurationMinutes,
		time.Now().UTC(),
	)

	silenceResp, err := alerting.CreateSilence(getSystemAlertOrgID(system), silenceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create silence in mimir: "+err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("alert silenced successfully", gin.H{
		"silence_id": silenceResp.SilenceID,
	}))
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

	c.JSON(http.StatusOK, response.OK("silence disabled successfully", nil))
}

// GetSystemAlertHistory handles GET /api/systems/:id/alerts/history
// Returns paginated resolved/inactive alert history for a system.
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

	repo := entities.NewLocalAlertHistoryRepository()
	records, totalCount, err := repo.GetAlertHistoryBySystemKey(system.SystemKey, page, pageSize, sortBy, sortDirection)
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
