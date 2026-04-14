/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"testing"
	"time"

	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterAlerts(t *testing.T) {
	alerts := []map[string]interface{}{
		{
			"labels": map[string]interface{}{
				"alertname":  "DiskFull",
				"severity":   "critical",
				"system_key": "SYS-001",
			},
			"status": map[string]interface{}{
				"state": "active",
			},
		},
		{
			"labels": map[string]interface{}{
				"alertname":  "HighCPU",
				"severity":   "warning",
				"system_key": "SYS-002",
			},
			"status": map[string]interface{}{
				"state": "suppressed",
			},
		},
		{
			"labels": map[string]interface{}{
				"alertname":  "LowMemory",
				"severity":   "critical",
				"system_key": "SYS-001",
			},
			"status": map[string]interface{}{
				"state": "active",
			},
		},
	}

	tests := []struct {
		name     string
		params   models.AlertQueryParams
		expected int
	}{
		{
			name:     "no filters returns all",
			params:   models.AlertQueryParams{},
			expected: 3,
		},
		{
			name:     "filter by state active",
			params:   models.AlertQueryParams{State: "active"},
			expected: 2,
		},
		{
			name:     "filter by state suppressed",
			params:   models.AlertQueryParams{State: "suppressed"},
			expected: 1,
		},
		{
			name:     "filter by severity critical",
			params:   models.AlertQueryParams{Severity: "critical"},
			expected: 2,
		},
		{
			name:     "filter by severity warning",
			params:   models.AlertQueryParams{Severity: "warning"},
			expected: 1,
		},
		{
			name:     "filter by system_key SYS-001",
			params:   models.AlertQueryParams{SystemKey: "SYS-001"},
			expected: 2,
		},
		{
			name:     "filter by system_key SYS-002",
			params:   models.AlertQueryParams{SystemKey: "SYS-002"},
			expected: 1,
		},
		{
			name:     "combined filters: active + critical",
			params:   models.AlertQueryParams{State: "active", Severity: "critical"},
			expected: 2,
		},
		{
			name:     "combined filters: active + warning",
			params:   models.AlertQueryParams{State: "active", Severity: "warning"},
			expected: 0,
		},
		{
			name:     "combined filters: active + SYS-001 + critical",
			params:   models.AlertQueryParams{State: "active", SystemKey: "SYS-001", Severity: "critical"},
			expected: 2,
		},
		{
			name:     "non-existent state",
			params:   models.AlertQueryParams{State: "unknown"},
			expected: 0,
		},
		{
			name:     "non-existent system_key",
			params:   models.AlertQueryParams{SystemKey: "SYS-999"},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterAlerts(alerts, tt.params)
			assert.Equal(t, tt.expected, len(result))
		})
	}
}

func TestFilterAlerts_MissingLabels(t *testing.T) {
	alerts := []map[string]interface{}{
		{
			"labels": map[string]interface{}{
				"alertname": "NoSeverity",
			},
			"status": map[string]interface{}{
				"state": "active",
			},
		},
		{
			// No labels at all
			"status": map[string]interface{}{
				"state": "active",
			},
		},
	}

	// Filter by severity — alerts without the label pass through (not excluded)
	result := filterAlerts(alerts, models.AlertQueryParams{Severity: "critical"})
	assert.Equal(t, 2, len(result))

	// Filter by state — both have "active" state
	result = filterAlerts(alerts, models.AlertQueryParams{State: "active"})
	assert.Equal(t, 2, len(result))
}

func TestFilterAlerts_EmptyInput(t *testing.T) {
	var empty []map[string]interface{}
	result := filterAlerts(empty, models.AlertQueryParams{State: "active"})
	assert.Equal(t, 0, len(result))

	result = filterAlerts(nil, models.AlertQueryParams{})
	assert.Nil(t, result)
}

func TestFindSystemAlertByFingerprint(t *testing.T) {
	alerts := []models.ActiveAlert{
		{
			Fingerprint: "alert-1",
			Labels: map[string]string{
				"alertname":  "HostDown",
				"system_key": "system-1",
			},
		},
		{
			Fingerprint: "alert-2",
			Labels: map[string]string{
				"alertname":  "DiskFull",
				"system_key": "system-2",
			},
		},
	}

	result := findSystemAlertByFingerprint(alerts, "alert-1", "system-1")
	require.NotNil(t, result)
	assert.Equal(t, "HostDown", result.Labels["alertname"])

	assert.Nil(t, findSystemAlertByFingerprint(alerts, "alert-1", "system-2"))
	assert.Nil(t, findSystemAlertByFingerprint(alerts, "missing", "system-1"))
}

func TestBuildSystemAlertSilenceRequest(t *testing.T) {
	now := time.Date(2026, time.April, 14, 10, 0, 0, 0, time.UTC)
	alert := &models.ActiveAlert{
		Labels: map[string]string{
			"system_key":      "forged-system-key",
			"alertname":       "HostDown",
			"severity":        "critical",
			"system_id":       "system-1",
			"organization_id": "org-1",
			"empty_label":     "",
		},
	}

	req := buildSystemAlertSilenceRequest(
		alert,
		"system-1-key",
		"admin@example.com",
		"  ",
		0,
		now,
	)

	assert.Equal(t, "2026-04-14T10:00:00Z", req.StartsAt)
	assert.Equal(t, "2026-04-14T11:00:00Z", req.EndsAt)
	assert.Equal(t, "admin@example.com", req.CreatedBy)
	assert.Equal(t, "silenced from my", req.Comment)
	assert.Equal(t, []models.AlertmanagerMatcher{
		{Name: "alertname", Value: "HostDown", IsRegex: false},
		{Name: "organization_id", Value: "org-1", IsRegex: false},
		{Name: "severity", Value: "critical", IsRegex: false},
		{Name: "system_id", Value: "system-1", IsRegex: false},
		{Name: "system_key", Value: "system-1-key", IsRegex: false},
	}, req.Matchers)
}

func TestBuildSystemAlertSilenceRequestAddsSystemKeyMatcher(t *testing.T) {
	now := time.Date(2026, time.April, 14, 10, 0, 0, 0, time.UTC)
	alert := &models.ActiveAlert{
		Labels: map[string]string{
			"alertname": "HostDown",
		},
	}

	req := buildSystemAlertSilenceRequest(alert, "system-1-key", "admin@example.com", "manual silence", 30, now)

	assert.Equal(t, []models.AlertmanagerMatcher{
		{Name: "alertname", Value: "HostDown", IsRegex: false},
		{Name: "system_key", Value: "system-1-key", IsRegex: false},
	}, req.Matchers)
	assert.Equal(t, "manual silence", req.Comment)
	assert.Equal(t, "2026-04-14T10:30:00Z", req.EndsAt)
}
