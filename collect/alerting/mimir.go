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
	SystemType       string
	SystemFQDN       string
	SystemIPv4       string
	OrganizationName string
	OrganizationVAT  string
	OrganizationType string
	// ResellerOrgID is the Mimir tenant that owns this system's alerting: the
	// managing reseller/distributor for a customer-owned system, else the
	// owning org itself. Resolved server-side; never trusted from the client.
	ResellerOrgID string
}

// SystemAlertContext contains the resolved tenant and authoritative labels for an alert.
type SystemAlertContext struct {
	OrganizationID string
	ResellerOrgID  string
	SystemID       string
	SystemKey      string
	Labels         map[string]string
}

// LookupSystemAlertContext resolves the server-side labels for a system.
func LookupSystemAlertContext(ctx context.Context, db *sql.DB, systemID string) (*SystemAlertContext, error) {
	var (
		metadata         SystemAlertMetadata
		systemType       sql.NullString
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
		       s.type,
		       s.fqdn,
		       s.ipv4_address::text,
		       COALESCE(d.name, r.name, c.name),
		       COALESCE(d.custom_data->>'vat', r.custom_data->>'vat', c.custom_data->>'vat'),
		       CASE
		           WHEN d.logto_id IS NOT NULL THEN 'distributor'
		           WHEN r.logto_id IS NOT NULL THEN 'reseller'
		           WHEN c.logto_id IS NOT NULL THEN 'customer'
		           ELSE NULL
		       END,
		       COALESCE(NULLIF(c.custom_data->>'createdBy', ''), s.organization_id)
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
		&systemType,
		&systemFQDN,
		&systemIPv4,
		&organizationName,
		&organizationVAT,
		&organizationType,
		&metadata.ResellerOrgID,
	)
	if err != nil {
		return nil, err
	}

	metadata.SystemType = nullStringValue(systemType)
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
		"system_type":       metadata.SystemType,
		"system_fqdn":       metadata.SystemFQDN,
		"system_ipv4":       metadata.SystemIPv4,
		"organization_id":   metadata.OrganizationID,
		"organization_name": metadata.OrganizationName,
		"organization_vat":  metadata.OrganizationVAT,
		"organization_type": metadata.OrganizationType,
	}

	return &SystemAlertContext{
		OrganizationID: metadata.OrganizationID,
		ResellerOrgID:  metadata.ResellerOrgID,
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

// maxLazyInitAttempts bounds retries for the 406 "Not initializing the
// Alertmanager" window: after a tenant's config is first pushed, Mimir
// instantiates that tenant's Alertmanager lazily (~config poll interval), and
// POSTs during that window return 406. We retry with backoff instead of
// dropping the alert.
const maxLazyInitAttempts = 4

// DoWithLazyInitRetry issues the request produced by newReq, retrying with
// exponential backoff on a 406 ("not initializing the Alertmanager") or a
// transient network error. Any other response — including other >=300 codes —
// is returned to the caller unchanged. newReq is called once per attempt so the
// request body can be replayed.
func DoWithLazyInitRetry(newReq func() (*http.Request, error)) (*http.Response, error) {
	backoff := time.Second
	var lastErr error
	for attempt := 0; attempt < maxLazyInitAttempts; attempt++ {
		req, err := newReq()
		if err != nil {
			return nil, err
		}
		resp, err := MimirHTTPClient.Do(req)
		switch {
		case err != nil:
			lastErr = err
		case resp.StatusCode == http.StatusNotAcceptable:
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("mimir tenant alertmanager not initialized (406)")
		default:
			return resp, nil
		}
		if attempt < maxLazyInitAttempts-1 {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	return nil, lastErr
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

	resp, err := DoWithLazyInitRetry(func() (*http.Request, error) {
		req, reqErr := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", configuration.Config.MimirURL, MimirAlertsPath), bytes.NewReader(body))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Scope-OrgID", orgID)
		return req, nil
	})
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

// BuildResolvedLinkFailedAlert builds a resolved (EndsAt in the past) LinkFailed
// alert for a recovered system. It reuses the exact firing label set — the same
// three base labels plus EnrichAlerts(systemContext) — so Alertmanager recomputes
// the identical fingerprint and clears the firing alert instead of opening a new
// one. Annotations are omitted: they don't affect the fingerprint.
func BuildResolvedLinkFailedAlert(systemContext *SystemAlertContext) (models.AlertmanagerPostAlert, error) {
	now := time.Now().UTC()
	enriched, err := EnrichAlerts([]models.AlertmanagerPostAlert{
		{
			Labels: map[string]string{
				"alertname":    LinkFailedAlert,
				"severity":     "critical",
				ManagedByLabel: ManagedByCollect,
			},
			StartsAt: now.Add(-time.Minute),
			EndsAt:   now,
		},
	}, systemContext)
	if err != nil {
		return models.AlertmanagerPostAlert{}, err
	}
	return enriched[0], nil
}

func nullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
