/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
)

const (
	MimirAlertsPath    = "/alertmanager/api/v2/alerts"
	LinkFailedAlert    = "LinkFailed"
	ManagedByLabel     = "managed_by"
	ManagedByCollect   = "my-collect"
	listAlertsBodySize = 4 << 20
)

var MimirHTTPClient = &http.Client{Timeout: 30 * time.Second}

// SystemAlertMetadata contains the system and organization data injected into alerts.
type SystemAlertMetadata struct {
	SystemID         string
	OrganizationID   string
	SystemKey        string
	SystemName       string
	SystemFQDN       string
	SystemIPv4       string
	OrganizationName string
	OrganizationVAT  string
	OrganizationType string
}

// SystemAlertContext contains the resolved tenant and authoritative labels for an alert.
type SystemAlertContext struct {
	OrganizationID string
	SystemID       string
	SystemKey      string
	Labels         map[string]string
}

// LookupSystemAlertContext resolves the server-side labels for a system.
func LookupSystemAlertContext(ctx context.Context, db *sql.DB, systemID string) (*SystemAlertContext, error) {
	var (
		metadata         SystemAlertMetadata
		systemFQDN       sql.NullString
		systemIPv4       sql.NullString
		organizationName sql.NullString
		organizationVAT  sql.NullString
		organizationType sql.NullString
	)

	err := db.QueryRowContext(ctx, `
		SELECT s.id::text,
		       s.organization_id,
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
	`, systemID).Scan(
		&metadata.SystemID,
		&metadata.OrganizationID,
		&metadata.SystemKey,
		&metadata.SystemName,
		&systemFQDN,
		&systemIPv4,
		&organizationName,
		&organizationVAT,
		&organizationType,
	)
	if err != nil {
		return nil, err
	}

	metadata.SystemFQDN = nullStringValue(systemFQDN)
	metadata.SystemIPv4 = nullStringValue(systemIPv4)
	metadata.OrganizationName = nullStringValue(organizationName)
	metadata.OrganizationVAT = nullStringValue(organizationVAT)
	metadata.OrganizationType = nullStringValue(organizationType)

	return BuildSystemAlertContext(metadata), nil
}

// BuildSystemAlertContext converts metadata into the authoritative label set used by alerts.
func BuildSystemAlertContext(metadata SystemAlertMetadata) *SystemAlertContext {
	labels := map[string]string{
		"system_id":  metadata.SystemID,
		"system_key": metadata.SystemKey,
	}

	if metadata.SystemName != "" {
		labels["system_name"] = metadata.SystemName
	}
	if metadata.SystemFQDN != "" {
		labels["system_fqdn"] = metadata.SystemFQDN
	}
	if metadata.SystemIPv4 != "" {
		labels["system_ipv4"] = metadata.SystemIPv4
	}
	if metadata.OrganizationName != "" {
		labels["organization_name"] = metadata.OrganizationName
	}
	if metadata.OrganizationVAT != "" {
		labels["organization_vat"] = metadata.OrganizationVAT
	}
	if metadata.OrganizationType != "" {
		labels["organization_type"] = metadata.OrganizationType
	}

	return &SystemAlertContext{
		OrganizationID: metadata.OrganizationID,
		SystemID:       metadata.SystemID,
		SystemKey:      metadata.SystemKey,
		Labels:         labels,
	}
}

// EnrichAlertPayload injects authoritative server-side labels and renders annotation templates.
func EnrichAlertPayload(body []byte, systemContext *SystemAlertContext) []byte {
	if systemContext == nil || len(systemContext.Labels) == 0 {
		return body
	}

	body = InjectLabels(body, systemContext.Labels)
	return ProcessAnnotationTemplates(body)
}

// EnrichAlerts applies the same enrichment flow used by ProxyMimir to structured alert payloads.
func EnrichAlerts(alerts []models.AlertmanagerPostAlert, systemContext *SystemAlertContext) ([]models.AlertmanagerPostAlert, error) {
	if len(alerts) == 0 {
		return []models.AlertmanagerPostAlert{}, nil
	}

	body, err := json.Marshal(alerts)
	if err != nil {
		return nil, fmt.Errorf("marshal alert payload: %w", err)
	}

	body = EnrichAlertPayload(body, systemContext)

	var enriched []models.AlertmanagerPostAlert
	if err := json.Unmarshal(body, &enriched); err != nil {
		return nil, fmt.Errorf("unmarshal enriched alert payload: %w", err)
	}

	return enriched, nil
}

// InjectLabels adds the given labels to each alert in the payload. The client-provided
// system_key label is always replaced with the authoritative system value.
func InjectLabels(body []byte, toInject map[string]string) []byte {
	if len(toInject) == 0 {
		return body
	}

	var alerts []map[string]interface{}
	if err := json.Unmarshal(body, &alerts); err != nil {
		return body
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

// ProcessAnnotationTemplates renders Go text/template expressions inside annotations
// using the alert labels as the template data source.
func ProcessAnnotationTemplates(body []byte) []byte {
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
				logger.Warn().Err(err).Str("annotation", key).Str("value", annotationStr).Msg("alerting: failed to parse annotation template")
				continue
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, labels); err != nil {
				logger.Warn().Err(err).Str("annotation", key).Msg("alerting: failed to execute annotation template")
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

// ListAlerts fetches active alerts for a tenant, optionally applying Alertmanager filters.
func ListAlerts(orgID string, filters ...string) ([]models.AlertmanagerAlert, error) {
	targetURL := fmt.Sprintf("%s%s", configuration.Config.MimirURL, MimirAlertsPath)
	if len(filters) > 0 {
		query := url.Values{}
		for _, filter := range filters {
			query.Add("filter", filter)
		}
		targetURL += "?" + query.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create alert list request: %w", err)
	}
	req.Header.Set("X-Scope-OrgID", orgID)
	req.Header.Set("Accept", "application/json")

	resp, err := MimirHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list alerts from mimir: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error().Err(err).Msg("alerting: failed to close alert list response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return nil, fmt.Errorf("list alerts returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, listAlertsBodySize))
	if err != nil {
		return nil, fmt.Errorf("read alert list response: %w", err)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return []models.AlertmanagerAlert{}, nil
	}

	var alerts []models.AlertmanagerAlert
	if err := json.Unmarshal(body, &alerts); err != nil {
		return nil, fmt.Errorf("unmarshal alert list response: %w", err)
	}
	if alerts == nil {
		alerts = []models.AlertmanagerAlert{}
	}

	return alerts, nil
}

// PostAlerts sends a batch of alerts to Alertmanager for the given tenant.
func PostAlerts(orgID string, alerts []models.AlertmanagerPostAlert) error {
	if len(alerts) == 0 {
		return nil
	}

	body, err := json.Marshal(alerts)
	if err != nil {
		return fmt.Errorf("marshal alerts: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", configuration.Config.MimirURL, MimirAlertsPath), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create alert post request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Scope-OrgID", orgID)

	resp, err := MimirHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("post alerts to mimir: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error().Err(err).Msg("alerting: failed to close alert post response body")
		}
	}()

	if resp.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("post alerts returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return nil
}

func nullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
