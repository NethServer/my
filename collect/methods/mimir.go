/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/response"
)

var mimirHTTPClient = &http.Client{Timeout: 30 * time.Second}

// ProxyMimir forwards requests to Mimir on behalf of authenticated systems.
// BasicAuthMiddleware has already validated credentials and set "system_id" in the context.
// Route matching in main.go restricts access to /alertmanager/api/v2/alerts and
// /alertmanager/api/v2/silences; no further path checks are needed here.
// X-Scope-OrgID is always injected using the system's organization_id.
func ProxyMimir(c *gin.Context) {
	subPath := strings.TrimPrefix(c.Request.URL.Path, "/api/services/mimir")
	rawQuery := c.Request.URL.RawQuery

	// Resolve organization_id for X-Scope-OrgID injection
	systemID, ok := getAuthenticatedSystemID(c)
	if !ok {
		logger.Warn().Str("reason", "missing system_id in context").Msg("mimir proxy auth failed")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
		return
	}

	var (
		orgID      string
		systemKey  string
		systemName string
		systemFQDN sql.NullString
		systemIPv4 sql.NullString
		orgName    sql.NullString
		orgVAT     sql.NullString
		orgType    sql.NullString
	)
	err := database.DB.QueryRow(`
		SELECT s.organization_id,
		       s.system_key,
		       s.name,
		       s.fqdn,
		       s.ipv4_address::text,
		       COALESCE(d.name, r.name, c.name),
		       COALESCE(d.custom_data->>'vat', r.custom_data->>'vat', c.custom_data->>'vat'),
		       CASE
		           WHEN d.logto_id IS NOT NULL THEN 'distributor'
		           WHEN r.logto_id IS NOT NULL THEN 'reseller'
		           WHEN c.logto_id IS NOT NULL THEN 'customer'
		           ELSE NULL
		       END
		FROM systems s
		LEFT JOIN distributors d ON (s.organization_id = d.logto_id OR s.organization_id = d.id) AND d.deleted_at IS NULL
		LEFT JOIN resellers r ON (s.organization_id = r.logto_id OR s.organization_id = r.id) AND r.deleted_at IS NULL
		LEFT JOIN customers c ON (s.organization_id = c.logto_id OR s.organization_id = c.id) AND c.deleted_at IS NULL
		WHERE s.id = $1
	`, systemID).Scan(&orgID, &systemKey, &systemName, &systemFQDN, &systemIPv4, &orgName, &orgVAT, &orgType)

	if err == sql.ErrNoRows {
		logger.Warn().Str("system_id", systemID).Str("reason", "system not found").Msg("mimir proxy: system lookup failed")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
		return
	}
	if err != nil {
		logger.Error().Err(err).Str("system_id", systemID).Msg("mimir proxy: db query failed")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("internal server error", nil))
		return
	}

	// Buffer request body once so it can be replayed across retry attempts
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error().Err(err).Msg("mimir proxy: failed to read request body")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("internal server error", nil))
		return
	}

	// Inject server-side system context into POST alerts, always overriding
	// system_key with the authenticated system value.
	if c.Request.Method == http.MethodPost && strings.Contains(subPath, "/alerts") && len(bodyBytes) > 0 {
		injected := map[string]string{
			"system_id":  systemID,
			"system_key": systemKey,
		}
		if systemName != "" {
			injected["system_name"] = systemName
		}
		if systemFQDN.Valid && systemFQDN.String != "" {
			injected["system_fqdn"] = systemFQDN.String
		}
		if systemIPv4.Valid && systemIPv4.String != "" {
			injected["system_ipv4"] = systemIPv4.String
		}
		if orgName.Valid && orgName.String != "" {
			injected["organization_name"] = orgName.String
		}
		if orgVAT.Valid && orgVAT.String != "" {
			injected["organization_vat"] = orgVAT.String
		}
		if orgType.Valid && orgType.String != "" {
			injected["organization_type"] = orgType.String
		}
		bodyBytes = injectLabels(bodyBytes, injected)
	}

	// Forward request to Mimir
	targetURL := fmt.Sprintf("%s%s", configuration.Config.MimirURL, subPath)
	if rawQuery != "" {
		targetURL += "?" + rawQuery
	}

	logger.Info().Str("target", targetURL).Str("org_id", orgID).Msg("mimir proxy: forwarding request")

	req, err := http.NewRequest(c.Request.Method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		logger.Error().Err(err).Str("target", targetURL).Msg("mimir proxy: failed to create upstream request")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("internal server error", nil))
		return
	}

	for _, header := range []string{"Content-Type", "Content-Encoding", "Accept", "User-Agent"} {
		if val := c.GetHeader(header); val != "" {
			req.Header.Set(header, val)
		}
	}
	// Remove Accept-Encoding so Mimir sends plain JSON, not gzip
	req.Header.Del("Accept-Encoding")
	req.Header.Set("X-Scope-OrgID", orgID)

	resp, err := mimirHTTPClient.Do(req)
	if err != nil {
		logger.Error().Err(err).Str("target", targetURL).Msg("mimir proxy: network error")
		c.JSON(http.StatusBadGateway, response.InternalServerError("mimir is unavailable", nil))
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error().Err(err).Msg("mimir proxy: failed to close upstream response body")
		}
	}()

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		c.Header("Content-Type", ct)
	}
	c.Status(resp.StatusCode)
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		logger.Error().Err(err).Msg("mimir proxy: error streaming response body")
	}
}

// injectLabels adds the given labels to each alert in the payload. The
// client-provided system_key label is always replaced with the authenticated
// system value; other labels are added only when missing.
func injectLabels(body []byte, toInject map[string]string) []byte {
	if len(toInject) == 0 {
		return body
	}

	var alerts []map[string]interface{}
	if err := json.Unmarshal(body, &alerts); err != nil {
		return body // Not a valid alert array, pass through unchanged
	}

	modified := false
	for _, alert := range alerts {
		labels, ok := alert["labels"].(map[string]interface{})
		if !ok {
			labels = map[string]interface{}{}
			alert["labels"] = labels
		}
		for key, value := range toInject {
			if key == "system_key" {
				current, exists := labels[key]
				currentValue, isString := current.(string)
				if !exists || !isString || currentValue != value {
					labels[key] = value
					modified = true
				}
				continue
			}
			if _, exists := labels[key]; !exists {
				labels[key] = value
				modified = true
			}
		}
	}

	if !modified {
		return body
	}

	out, err := json.Marshal(alerts)
	if err != nil {
		return body
	}
	return out
}
