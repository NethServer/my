/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/response"
)

var mimirHTTPClient = &http.Client{Timeout: 30 * time.Second}

const (
	mimirAlertsPath   = "/alertmanager/api/v2/alerts"
	mimirSilencesPath = "/alertmanager/api/v2/silences"
)

// ProxyMimir forwards requests to Mimir on behalf of authenticated systems.
// BasicAuthMiddleware has already validated credentials and set "system_id" in the context.
// Each machine is scoped to its own alerts and silences (identified by system_key):
//   - GET /alerts and GET /silences: filter param injected to scope results to this system
//   - POST /silences: system_key matcher injected into the silence matchers
//   - GET /silences/:id and DELETE /silences/:id: ownership verified before forwarding
//
// X-Scope-OrgID is always injected server-side from the system's organization_id.
func ProxyMimir(c *gin.Context) {
	subPath := strings.TrimPrefix(c.Request.URL.Path, "/api/services/mimir")
	rawQuery := c.Request.URL.RawQuery
	method := c.Request.Method

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

	// Enforce request body size limit (same ceiling as the inventory endpoint).
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, configuration.Config.APIMaxRequestSize)

	// Buffer request body once so it can be replayed across retry attempts.
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			logger.Warn().Int64("limit", maxErr.Limit).Str("system_id", systemID).Msg("mimir proxy: request body exceeds size limit")
			c.JSON(http.StatusRequestEntityTooLarge, response.BadRequest("request body too large", nil))
		} else {
			logger.Error().Err(err).Msg("mimir proxy: failed to read request body")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("internal server error", nil))
		}
		return
	}

	// Per-machine scoping: restrict each system to its own alerts and silences.
	isSilenceByID := strings.HasPrefix(subPath, mimirSilencesPath+"/") && len(subPath) > len(mimirSilencesPath)+1
	switch {
	case method == http.MethodGet && (subPath == mimirAlertsPath || subPath == mimirSilencesPath):
		// Scope listing results to only this machine's data.
		rawQuery = appendSystemKeyFilter(rawQuery, systemKey)

	case method == http.MethodPost && subPath == mimirSilencesPath:
		// Ensure the silence targets only this machine's alerts.
		bodyBytes = injectSilenceMatcher(bodyBytes, systemKey)

	case isSilenceByID:
		// Verify the silence belongs to this machine before allowing GET or DELETE.
		owned, checkErr := fetchAndCheckSilenceOwnership(orgID, subPath, systemKey)
		if checkErr != nil {
			logger.Error().Err(checkErr).Str("system_id", systemID).Str("path", subPath).Msg("mimir proxy: silence ownership check failed")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("internal server error", nil))
			return
		}
		if !owned {
			logger.Warn().Str("system_id", systemID).Str("path", subPath).Msg("mimir proxy: access denied to silence not owned by system")
			c.JSON(http.StatusForbidden, response.Forbidden("access denied", nil))
			return
		}
	}

	// Inject server-side system context into POST alerts, always overriding
	// system_key with the authenticated system value.
	if method == http.MethodPost && subPath == mimirAlertsPath && len(bodyBytes) > 0 {
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
		bodyBytes = processAnnotationTemplates(bodyBytes, injected)
	}

	// Forward request to Mimir
	targetURL := fmt.Sprintf("%s%s", configuration.Config.MimirURL, subPath)
	if rawQuery != "" {
		targetURL += "?" + rawQuery
	}

	logger.Info().Str("target", targetURL).Str("org_id", orgID).Msg("mimir proxy: forwarding request")

	req, err := http.NewRequest(method, targetURL, bytes.NewReader(bodyBytes))
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

// appendSystemKeyFilter appends a Prometheus label matcher for system_key to the query string,
// scoping GET /alerts and GET /silences results to only this machine's data.
func appendSystemKeyFilter(rawQuery, systemKey string) string {
	filter := fmt.Sprintf(`system_key="%s"`, systemKey)
	encoded := url.QueryEscape(filter)
	if rawQuery == "" {
		return "filter=" + encoded
	}
	return rawQuery + "&filter=" + encoded
}

// injectSilenceMatcher ensures the silence body contains an exact system_key matcher for
// this machine, overwriting any client-supplied system_key matcher.
func injectSilenceMatcher(body []byte, systemKey string) []byte {
	if len(body) == 0 {
		return body
	}
	var silence map[string]interface{}
	if err := json.Unmarshal(body, &silence); err != nil {
		return body
	}

	matchers, _ := silence["matchers"].([]interface{})
	filtered := make([]interface{}, 0, len(matchers)+1)
	for _, m := range matchers {
		mm, ok := m.(map[string]interface{})
		if !ok {
			filtered = append(filtered, m)
			continue
		}
		if name, _ := mm["name"].(string); name == "system_key" {
			continue // replaced below
		}
		filtered = append(filtered, m)
	}
	filtered = append(filtered, map[string]interface{}{
		"name":    "system_key",
		"value":   systemKey,
		"isRegex": false,
		"isEqual": true,
	})
	silence["matchers"] = filtered

	out, err := json.Marshal(silence)
	if err != nil {
		return body
	}
	return out
}

// fetchAndCheckSilenceOwnership fetches a silence from Mimir and returns true only if it
// contains an exact non-regex system_key matcher matching the given systemKey.
func fetchAndCheckSilenceOwnership(orgID, subPath, systemKey string) (bool, error) {
	targetURL := fmt.Sprintf("%s%s", configuration.Config.MimirURL, subPath)
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return false, fmt.Errorf("building ownership-check request: %w", err)
	}
	req.Header.Set("X-Scope-OrgID", orgID)
	req.Header.Set("Accept", "application/json")

	resp, err := mimirHTTPClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("fetching silence for ownership check: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected upstream status %d during ownership check", resp.StatusCode)
	}

	// Limit the ownership-check response to 1 MB; individual silences are small JSON objects.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return false, fmt.Errorf("reading silence response: %w", err)
	}
	return silenceHasSystemKeyMatcher(body, systemKey), nil
}

// silenceHasSystemKeyMatcher returns true if the silence JSON contains an exact
// (non-regex, isEqual=true) matcher for the given systemKey.
func silenceHasSystemKeyMatcher(silenceBody []byte, systemKey string) bool {
	var silence struct {
		Matchers []struct {
			Name    string `json:"name"`
			Value   string `json:"value"`
			IsRegex bool   `json:"isRegex"`
			IsEqual *bool  `json:"isEqual"`
		} `json:"matchers"`
	}
	if err := json.Unmarshal(silenceBody, &silence); err != nil {
		return false
	}
	for _, m := range silence.Matchers {
		if m.Name == "system_key" && m.Value == systemKey && !m.IsRegex {
			// isEqual defaults to true when absent
			return m.IsEqual == nil || *m.IsEqual
		}
	}
	return false
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

// processAnnotationTemplates applies Go text/template processing to annotations in alerts.
// It uses alert labels as template data. For example, an annotation value like
// "severity={{.severity}}" will have {{.severity}} replaced with the value of the
// severity label. If template processing fails, the annotation is left unchanged.
func processAnnotationTemplates(body []byte, _ map[string]string) []byte {
	var alerts []map[string]interface{}
	if err := json.Unmarshal(body, &alerts); err != nil {
		return body
	}

	modified := false
	for _, alert := range alerts {
		annotations, ok := alert["annotations"].(map[string]interface{})
		if !ok || len(annotations) == 0 {
			continue
		}

		labels, ok := alert["labels"].(map[string]interface{})
		if !ok {
			labels = map[string]interface{}{}
		}

		for key, val := range annotations {
			annotationStr, ok := val.(string)
			if !ok || !strings.Contains(annotationStr, "{{") {
				continue
			}

			tmpl, err := template.New(key).Parse(annotationStr)
			if err != nil {
				logger.Warn().Err(err).Str("annotation", key).Str("value", annotationStr).Msg("mimir proxy: failed to parse annotation template")
				continue
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, labels); err != nil {
				logger.Warn().Err(err).Str("annotation", key).Msg("mimir proxy: failed to execute annotation template")
				continue
			}

			rendered := buf.String()
			if rendered != annotationStr {
				annotations[key] = rendered
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
