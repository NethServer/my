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
	"regexp"
	"strings"
	"time"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
)

const (
	MimirAlertsPath  = "/alertmanager/api/v2/alerts"
	LinkFailedAlert  = "LinkFailed"
	ManagedByLabel   = "managed_by"
	ManagedByCollect = "my-collect"
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
// Every identity label is included even when the DB value is empty: InjectLabels
// will strip the client-supplied value for empty keys, preventing a compromised
// system from spoofing labels when the server has no value.
func BuildSystemAlertContext(metadata SystemAlertMetadata) *SystemAlertContext {
	labels := map[string]string{
		"system_id":         metadata.SystemID,
		"system_key":        metadata.SystemKey,
		"system_name":       metadata.SystemName,
		"system_fqdn":       metadata.SystemFQDN,
		"system_ipv4":       metadata.SystemIPv4,
		"organization_name": metadata.OrganizationName,
		"organization_vat":  metadata.OrganizationVAT,
		"organization_type": metadata.OrganizationType,
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

// InjectLabels authoritatively sets the given labels on each alert in the
// payload. Every key in toInject is server-controlled: a non-empty value
// replaces whatever the client sent; an empty value strips the client-supplied
// value entirely. Identity labels (system/organization) must never be
// attacker-controlled, otherwise a compromised system could spoof alerts for
// other systems within the same tenant.
func InjectLabels(body []byte, toInject map[string]string) []byte {
	if len(toInject) == 0 {
		return body
	}

	var alerts []map[string]interface{}
	if err := json.Unmarshal(body, &alerts); err != nil {
		return body
	}

	for _, alert := range alerts {
		labels, ok := alert["labels"].(map[string]interface{})
		if !ok {
			labels = map[string]interface{}{}
			alert["labels"] = labels
		}
		for key, value := range toInject {
			if value == "" {
				delete(labels, key)
			} else {
				labels[key] = value
			}
		}
	}

	out, err := json.Marshal(alerts)
	if err != nil {
		return body
	}
	return out
}

// simplePlaceholderRe matches only simple {{.label_name}} placeholders.
// It intentionally rejects any Go template directive (range, call, index, if,
// with, printf, etc.) so that attacker-controlled annotations cannot enumerate
// server-injected labels or cause resource exhaustion.
var simplePlaceholderRe = regexp.MustCompile(`\{\{\s*\.([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)

// ProcessAnnotationTemplates performs safe string substitution of {{.label}}
// placeholders in alert annotations using the alert's own labels as values.
// Only simple {{.variable_name}} placeholders are expanded; all other Go
// template constructs (range, call, index, printf, etc.) are left as-is.
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

			rendered := simplePlaceholderRe.ReplaceAllStringFunc(annotationStr, func(match string) string {
				sub := simplePlaceholderRe.FindStringSubmatch(match)
				if len(sub) < 2 {
					return match
				}
				labelName := sub[1]
				if v, exists := labels[labelName]; exists {
					if s, ok := v.(string); ok {
						return s
					}
				}
				return match
			})

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
	logger.Debug().Int("status_code", resp.StatusCode).Int("alerts_posted", len(alerts)).Msg("successfully posted alerts to mimir")

	return nil
}

func nullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
