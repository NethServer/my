/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"encoding/json"
	"net/http"
	"strings"

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

// resolveOrgID extracts the target organization ID.
// Owner/Distributor/Reseller must pass organization_id query param.
// Customer uses their own organization from JWT.
func resolveOrgID(c *gin.Context, user *models.User) (string, bool) {
	orgID := c.Query("organization_id")
	orgRole := strings.ToLower(user.OrgRole)

	if orgRole == "customer" {
		// Customer always uses their own organization
		return user.OrganizationID, true
	}

	// Owner, Distributor, Reseller must provide organization_id
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

// ConfigureAlerts handles POST /api/alerting/config
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

	templateFiles, err := alerting.BuildTemplateFiles(req.EmailTemplateLang)
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

// DisableAlerts handles DELETE /api/alerting/config
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

// GetAlerts handles GET /api/alerting/alerts
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

// GetAlertingConfig handles GET /api/alerting/config
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

// GetSystemAlertHistory handles GET /api/systems/:id/alerting/history
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
