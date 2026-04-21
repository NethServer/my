/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/models"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}
var ErrSilenceNotFound = errors.New("silence not found")

// maxMimirResponseSize caps how much data we read from Mimir responses.
// Prevents memory exhaustion if a tenant has a very large number of alerts,
// silences, or rules.
const maxMimirResponseSize = 10 << 20 // 10 MB

var smtpSensitiveFields = regexp.MustCompile(`(?m)^(\s*smtp_(?:smarthost|auth_username|auth_password):\s*).*$`)
var bearerTokenField = regexp.MustCompile(`(?m)^(\s*credentials:\s*).*$`)
var telegramTokenField = regexp.MustCompile(`(?m)^(\s*bot_token:\s*).*$`)

// RedactSensitiveConfig replaces sensitive SMTP and bearer token fields in an alertmanager config
// YAML string with a redaction placeholder before returning to clients.
func RedactSensitiveConfig(config string) string {
	config = smtpSensitiveFields.ReplaceAllString(config, "${1}'[REDACTED]'")
	config = bearerTokenField.ReplaceAllString(config, "${1}'[REDACTED]'")
	config = telegramTokenField.ReplaceAllString(config, "${1}'[REDACTED]'")
	return config
}

// wrapForMimir wraps a raw Alertmanager YAML config in the Mimir multi-tenant
// format expected by POST /api/v1/alerts. Optional templateFiles are embedded
// as template_files entries so Alertmanager can load them.
func wrapForMimir(yamlConfig string, templateFiles map[string]string) string {
	var sb strings.Builder
	sb.WriteString("alertmanager_config: |\n")
	for _, line := range strings.Split(yamlConfig, "\n") {
		sb.WriteString("    ")
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	if len(templateFiles) > 0 {
		sb.WriteString("\ntemplate_files:\n")
		// Sort keys for deterministic output.
		keys := make([]string, 0, len(templateFiles))
		for k := range templateFiles {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, name := range keys {
			content := templateFiles[name]
			sb.WriteString("  ")
			sb.WriteString(name)
			sb.WriteString(": |\n")
			for _, line := range strings.Split(content, "\n") {
				sb.WriteString("    ")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}
	}
	return sb.String()
}

// PushConfig uploads an alertmanager YAML configuration for the given tenant.
// templateFiles optionally embeds custom Alertmanager template files (filename → content)
// so that email_configs can reference named templates via html:/text:/Subject: fields.
func PushConfig(orgID, yamlConfig string, templateFiles map[string]string) error {
	url := configuration.Config.MimirURL + "/api/v1/alerts"

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(wrapForMimir(yamlConfig, templateFiles)))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/yaml")
	req.Header.Set("X-Scope-OrgID", orgID)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("pushing config to mimir: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxMimirResponseSize))
		return fmt.Errorf("mimir returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetAlerts fetches active alerts for the given tenant from Mimir.
func GetAlerts(orgID string) ([]byte, error) {
	url := configuration.Config.MimirURL + "/alertmanager/api/v2/alerts"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-Scope-OrgID", orgID)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching alerts from mimir: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMimirResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mimir returned %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// GetConfig fetches the current alertmanager configuration for the given tenant.
// If Mimir returns 404 (no config has ever been pushed for this tenant), an
// empty body is returned with no error — callers should treat this as "no
// config set" rather than a hard failure.
func GetConfig(orgID string) ([]byte, error) {
	url := configuration.Config.MimirURL + "/api/v1/alerts"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-Scope-OrgID", orgID)
	req.Header.Set("Accept", "application/yaml")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching config from mimir: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMimirResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mimir returned %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// CreateSilence creates an Alertmanager silence for the given tenant.
func CreateSilence(orgID string, silence *models.AlertmanagerSilenceRequest) (*models.AlertmanagerSilenceResponse, error) {
	url := configuration.Config.MimirURL + "/alertmanager/api/v2/silences"

	payload, err := json.Marshal(silence)
	if err != nil {
		return nil, fmt.Errorf("marshalling silence payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Scope-OrgID", orgID)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("creating silence in mimir: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMimirResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mimir returned %d: %s", resp.StatusCode, string(body))
	}

	if len(body) == 0 {
		return &models.AlertmanagerSilenceResponse{}, nil
	}

	var silenceResponse models.AlertmanagerSilenceResponse
	if err := json.Unmarshal(body, &silenceResponse); err != nil {
		return nil, fmt.Errorf("decoding silence response: %w", err)
	}

	return &silenceResponse, nil
}

// GetSilences fetches all silences for the given tenant from Mimir.
func GetSilences(orgID string) ([]models.AlertmanagerSilence, error) {
	url := configuration.Config.MimirURL + "/alertmanager/api/v2/silences"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Scope-OrgID", orgID)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching silences from mimir: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMimirResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mimir returned %d: %s", resp.StatusCode, string(body))
	}

	var silences []models.AlertmanagerSilence
	if err := json.Unmarshal(body, &silences); err != nil {
		return nil, fmt.Errorf("decoding silences: %w", err)
	}

	return silences, nil
}

// GetSilence fetches a specific Alertmanager silence for the given tenant.
func GetSilence(orgID, silenceID string) (*models.AlertmanagerSilence, error) {
	url := configuration.Config.MimirURL + "/alertmanager/api/v2/silence/" + url.PathEscape(silenceID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Scope-OrgID", orgID)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching silence from mimir: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMimirResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: %s", ErrSilenceNotFound, silenceID)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mimir returned %d: %s", resp.StatusCode, string(body))
	}

	var silence models.AlertmanagerSilence
	if err := json.Unmarshal(body, &silence); err != nil {
		return nil, fmt.Errorf("decoding silence response: %w", err)
	}

	return &silence, nil
}

// DeleteSilence deletes a specific Alertmanager silence for the given tenant.
func DeleteSilence(orgID, silenceID string) error {
	url := configuration.Config.MimirURL + "/alertmanager/api/v2/silence/" + url.PathEscape(silenceID)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Scope-OrgID", orgID)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("deleting silence in mimir: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMimirResponseSize))
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: %s", ErrSilenceNotFound, silenceID)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mimir returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
