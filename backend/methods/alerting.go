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
	"github.com/nethesis/my/backend/helpers"
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

	var req models.AlertingConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body: "+err.Error(), nil))
		return
	}

	// Validate severity keys
	validSeverities := map[string]bool{"critical": true, "warning": true, "info": true}
	for key := range req {
		if !validSeverities[key] {
			c.JSON(http.StatusBadRequest, response.BadRequest("invalid severity level: "+key+". allowed: critical, warning, info", nil))
			return
		}
	}

	cfg := configuration.Config
	yamlConfig, err := alerting.RenderConfig(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom, cfg.SMTPTLS,
		req,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to render alertmanager config: "+err.Error(), nil))
		return
	}

	if err := alerting.PushConfig(orgID, yamlConfig); err != nil {
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
		nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to render blackhole config: "+err.Error(), nil))
		return
	}

	if err := alerting.PushConfig(orgID, yamlConfig); err != nil {
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

	c.JSON(http.StatusOK, response.OK("alerting configuration retrieved successfully", gin.H{
		"config": string(body),
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
