/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nethesis/my/backend/configuration"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// wrapForMimir wraps a raw Alertmanager YAML config in the Mimir multi-tenant
// format expected by POST /api/v1/alerts.
func wrapForMimir(yamlConfig string) string {
	var sb strings.Builder
	sb.WriteString("alertmanager_config: |\n")
	for _, line := range strings.Split(yamlConfig, "\n") {
		sb.WriteString("    ")
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return sb.String()
}

// PushConfig uploads an alertmanager YAML configuration for the given tenant.
func PushConfig(orgID, yamlConfig string) error {
	url := configuration.Config.MimirURL + "/api/v1/alerts"

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(wrapForMimir(yamlConfig)))
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
