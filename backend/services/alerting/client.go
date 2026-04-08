/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/nethesis/my/backend/configuration"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

var smtpSensitiveFields = regexp.MustCompile(`(?m)^(\s*smtp_(?:smarthost|auth_username|auth_password):\s*).*$`)

// RedactSensitiveConfig replaces sensitive SMTP fields in an alertmanager config
// YAML string with a redaction placeholder before returning to clients.
func RedactSensitiveConfig(config string) string {
	return smtpSensitiveFields.ReplaceAllString(config, "${1}'[REDACTED]'")
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mimir returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteConfig removes the alertmanager configuration for the given tenant.
func DeleteConfig(orgID string) error {
	url := configuration.Config.MimirURL + "/api/v1/alerts"

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-Scope-OrgID", orgID)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("deleting config from mimir: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mimir returned %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// GetConfig fetches the current alertmanager configuration for the given tenant.
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mimir returned %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
