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
		params   alertFilter
		expected int
	}{
		{
			name:     "no filters returns all",
			params:   alertFilter{},
			expected: 3,
		},
		{
			name:     "filter by status active",
			params:   alertFilter{statuses: []string{"active"}},
			expected: 2,
		},
		{
			name:     "filter by status suppressed",
			params:   alertFilter{statuses: []string{"suppressed"}},
			expected: 1,
		},
		{
			name:     "filter by severity critical",
			params:   alertFilter{severities: []string{"critical"}},
			expected: 2,
		},
		{
			name:     "filter by severity warning",
			params:   alertFilter{severities: []string{"warning"}},
			expected: 1,
		},
		{
			name:     "filter by system_key SYS-001",
			params:   alertFilter{systemKeys: []string{"SYS-001"}},
			expected: 2,
		},
		{
			name:     "filter by system_key SYS-002",
			params:   alertFilter{systemKeys: []string{"SYS-002"}},
			expected: 1,
		},
		{
			name:     "filter by alertname DiskFull",
			params:   alertFilter{alertnames: []string{"DiskFull"}},
			expected: 1,
		},
		{
			name:     "combined filters: active + critical",
			params:   alertFilter{statuses: []string{"active"}, severities: []string{"critical"}},
			expected: 2,
		},
		{
			name:     "combined filters: active + warning",
			params:   alertFilter{statuses: []string{"active"}, severities: []string{"warning"}},
			expected: 0,
		},
		{
			name:     "combined filters: active + SYS-001 + critical",
			params:   alertFilter{statuses: []string{"active"}, systemKeys: []string{"SYS-001"}, severities: []string{"critical"}},
			expected: 2,
		},
		{
			name:     "multi-value severity (critical OR warning)",
			params:   alertFilter{severities: []string{"critical", "warning"}},
			expected: 3,
		},
		{
			name:     "multi-value alertname (DiskFull OR HighCPU)",
			params:   alertFilter{alertnames: []string{"DiskFull", "HighCPU"}},
			expected: 2,
		},
		{
			name:     "multi-value status (active OR suppressed)",
			params:   alertFilter{statuses: []string{"active", "suppressed"}},
			expected: 3,
		},
		{
			name:     "multi-value AND single-value combo",
			params:   alertFilter{severities: []string{"critical", "warning"}, statuses: []string{"active"}},
			expected: 2,
		},
		{
			name:     "non-existent status",
			params:   alertFilter{statuses: []string{"unknown"}},
			expected: 0,
		},
		{
			name:     "non-existent system_key",
			params:   alertFilter{systemKeys: []string{"SYS-999"}},
			expected: 0,
		},
		{
			name:     "non-existent alertname",
			params:   alertFilter{alertnames: []string{"DoesNotExist"}},
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

	// Filter by severity — alerts without the label must be excluded
	// to prevent silent leakage of unrelated alerts.
	result := filterAlerts(alerts, alertFilter{severities: []string{"critical"}})
	assert.Equal(t, 0, len(result))

	// Filter by system_key — alerts without the label must be excluded.
	result = filterAlerts(alerts, alertFilter{systemKeys: []string{"SYS-001"}})
	assert.Equal(t, 0, len(result))

	// Filter by alertname — second alert has no labels at all, must be excluded.
	result = filterAlerts(alerts, alertFilter{alertnames: []string{"NoSeverity"}})
	assert.Equal(t, 1, len(result))

	// Filter by state — both have "active" state, so both match.
	result = filterAlerts(alerts, alertFilter{statuses: []string{"active"}})
	assert.Equal(t, 2, len(result))
}

func TestFilterAlerts_EmptyInput(t *testing.T) {
	var empty []map[string]interface{}
	result := filterAlerts(empty, alertFilter{statuses: []string{"active"}})
	assert.Equal(t, 0, len(result))

	result = filterAlerts(nil, alertFilter{})
	assert.Nil(t, result)
}

func TestFindSystemAlertByFingerprint(t *testing.T) {
	alerts := []models.ActiveAlert{
		{
			Fingerprint: "alert-1",
			Labels: map[string]string{
				"alertname":  "LinkFailed",
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
	assert.Equal(t, "LinkFailed", result.Labels["alertname"])

	assert.Nil(t, findSystemAlertByFingerprint(alerts, "alert-1", "system-2"))
	assert.Nil(t, findSystemAlertByFingerprint(alerts, "missing", "system-1"))
}

func TestBuildSystemAlertSilenceRequest(t *testing.T) {
	now := time.Date(2026, time.April, 14, 10, 0, 0, 0, time.UTC)
	alert := &models.ActiveAlert{
		Labels: map[string]string{
			"system_key":      "forged-system-key",
			"alertname":       "LinkFailed",
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
		time.Time{},
	)

	assert.Equal(t, "2026-04-14T10:00:00Z", req.StartsAt)
	assert.Equal(t, "2026-04-14T11:00:00Z", req.EndsAt)
	assert.Equal(t, "admin@example.com", req.CreatedBy)
	assert.Equal(t, "silenced from my", req.Comment)
	assert.Equal(t, []models.AlertmanagerMatcher{
		{Name: "alertname", Value: "LinkFailed", IsRegex: false},
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
			"alertname": "LinkFailed",
		},
	}

	req := buildSystemAlertSilenceRequest(alert, "system-1-key", "admin@example.com", "manual silence", 30, now, time.Time{})

	assert.Equal(t, []models.AlertmanagerMatcher{
		{Name: "alertname", Value: "LinkFailed", IsRegex: false},
		{Name: "system_key", Value: "system-1-key", IsRegex: false},
	}, req.Matchers)
	assert.Equal(t, "manual silence", req.Comment)
	assert.Equal(t, "2026-04-14T10:30:00Z", req.EndsAt)
}

func TestSilenceBelongsToSystem(t *testing.T) {
	tests := []struct {
		name      string
		silence   *models.AlertmanagerSilence
		systemKey string
		expected  bool
	}{
		{
			name: "exact system key matcher",
			silence: &models.AlertmanagerSilence{
				Matchers: []models.AlertmanagerMatcher{
					{Name: "system_key", Value: "system-1", IsRegex: false},
				},
			},
			systemKey: "system-1",
			expected:  true,
		},
		{
			name: "regex matcher is rejected",
			silence: &models.AlertmanagerSilence{
				Matchers: []models.AlertmanagerMatcher{
					{Name: "system_key", Value: "system-1", IsRegex: true},
				},
			},
			systemKey: "system-1",
			expected:  false,
		},
		{
			name: "different system key",
			silence: &models.AlertmanagerSilence{
				Matchers: []models.AlertmanagerMatcher{
					{Name: "system_key", Value: "system-2", IsRegex: false},
				},
			},
			systemKey: "system-1",
			expected:  false,
		},
		{
			name: "missing system key matcher",
			silence: &models.AlertmanagerSilence{
				Matchers: []models.AlertmanagerMatcher{
					{Name: "alertname", Value: "LinkFailed", IsRegex: false},
				},
			},
			systemKey: "system-1",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, silenceBelongsToSystem(tt.silence, tt.systemKey))
		})
	}
}

func TestSystemKeyFromSilence(t *testing.T) {
	tests := []struct {
		name     string
		silence  *models.AlertmanagerSilence
		expected string
	}{
		{
			name:     "nil silence returns empty",
			silence:  nil,
			expected: "",
		},
		{
			name: "exact system_key matcher returns value",
			silence: &models.AlertmanagerSilence{
				Matchers: []models.AlertmanagerMatcher{
					{Name: "alertname", Value: "DiskFull", IsRegex: false},
					{Name: "system_key", Value: "SYS-001", IsRegex: false},
				},
			},
			expected: "SYS-001",
		},
		{
			name: "regex system_key matcher is ignored",
			silence: &models.AlertmanagerSilence{
				Matchers: []models.AlertmanagerMatcher{
					{Name: "system_key", Value: "SYS-.+", IsRegex: true},
				},
			},
			expected: "",
		},
		{
			name: "no system_key matcher returns empty",
			silence: &models.AlertmanagerSilence{
				Matchers: []models.AlertmanagerMatcher{
					{Name: "alertname", Value: "DiskFull", IsRegex: false},
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, systemKeyFromSilence(tt.silence))
		})
	}
}

func TestStatusOf(t *testing.T) {
	tests := []struct {
		name     string
		alert    map[string]interface{}
		expected string
	}{
		{
			name: "active state",
			alert: map[string]interface{}{
				"status": map[string]interface{}{"state": "active"},
			},
			expected: "active",
		},
		{
			name: "suppressed state",
			alert: map[string]interface{}{
				"status": map[string]interface{}{"state": "suppressed"},
			},
			expected: "suppressed",
		},
		{
			name:     "missing status returns empty",
			alert:    map[string]interface{}{"labels": map[string]interface{}{"alertname": "X"}},
			expected: "",
		},
		{
			name: "missing state field returns empty",
			alert: map[string]interface{}{
				"status": map[string]interface{}{"silencedBy": []interface{}{}},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, statusOf(tt.alert))
		})
	}
}

func TestSortAlertsListByStatus(t *testing.T) {
	alerts := []map[string]interface{}{
		{
			"fingerprint": "fp-z",
			"status":      map[string]interface{}{"state": "suppressed"},
		},
		{
			"fingerprint": "fp-a",
			"status":      map[string]interface{}{"state": "active"},
		},
		{
			"fingerprint": "fp-b",
			"status":      map[string]interface{}{"state": "active"},
		},
	}

	sortAlertsList(alerts, "status", "asc")

	// asc: active before suppressed (alphabetical on state string).
	// Tiebreaker is fingerprint asc and is stable regardless of direction,
	// so the two "active" rows stay in fp-a, fp-b order.
	assert.Equal(t, "fp-a", alerts[0]["fingerprint"])
	assert.Equal(t, "fp-b", alerts[1]["fingerprint"])
	assert.Equal(t, "fp-z", alerts[2]["fingerprint"])

	sortAlertsList(alerts, "status", "desc")
	assert.Equal(t, "fp-z", alerts[0]["fingerprint"])
	// Tiebreaker fingerprint asc stays in place even on desc primary.
	assert.Equal(t, "fp-a", alerts[1]["fingerprint"])
	assert.Equal(t, "fp-b", alerts[2]["fingerprint"])
}

func TestAlertsListAllowedSortByIncludesStatus(t *testing.T) {
	// The active-alerts list and the history list must share at least
	// severity/alertname/starts_at/status so the UI can offer a single
	// "Sort by" dropdown that works on both tabs.
	for _, col := range []string{"starts_at", "severity", "alertname", "status"} {
		assert.Truef(t, alertsListAllowedSortBy[col], "expected %q in alertsListAllowedSortBy", col)
	}
}

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{"valid https with public ip", "https://93.184.216.34/alert", false, ""},
		{"valid http with public ip", "http://93.184.216.34/alert", false, ""},
		{"reject ftp scheme", "ftp://example.com/file", true, "must use http or https"},
		{"reject gopher", "gopher://internal:6379", true, "must use http or https"},
		{"reject credentials in url", "https://user:pass@example.com/hook", true, "must not contain credentials"},
		{"reject loopback ip", "https://127.0.0.1/hook", true, "not allowed"},
		{"reject private 10.x", "https://10.0.0.1/hook", true, "not allowed"},
		{"reject private 192.168.x", "https://192.168.1.1/hook", true, "not allowed"},
		{"reject private 172.16.x", "https://172.16.0.1/hook", true, "not allowed"},
		{"reject link-local", "https://169.254.169.254/latest", true, "not allowed"},
		{"reject unspecified", "https://0.0.0.0/hook", true, "not allowed"},
		{"reject ipv6 loopback", "https://[::1]/hook", true, "not allowed"},
		{"reject ipv6 mapped loopback", "https://[::ffff:127.0.0.1]/hook", true, "not publicly routable"},
		{"reject ipv6 mapped private", "https://[::ffff:10.0.0.1]/hook", true, "not publicly routable"},
		{"reject empty host", "https:///path", true, "missing a host"},
		{"reject localhost string", "https://localhost/hook", true, "not allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := validateWebhookURL(tt.url)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.NotEmpty(t, code)
			} else {
				assert.NoError(t, err)
				assert.Empty(t, code)
			}
		})
	}
}

// TestAlertingFieldErrorPaths covers the paths that ConfigureAlerts uses to
// translate validation failures into the standard validation_error envelope:
// the model's Validate() returns *AlertingFieldError with a JSON-shaped key,
// and validateWebhookRecipients does the same for the SSRF check.
func TestAlertingFieldErrorPaths(t *testing.T) {
	t.Run("invalid email at index 2 produces structured error", func(t *testing.T) {
		cfg := models.AlertingConfigLayer{
			EmailRecipients: []models.EmailRecipient{
				{Address: "ok1@test.org"},
				{Address: "ok2@test.org"},
				{Address: "wwwwwww"},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		fe := asAlertingFieldError(err)
		require.NotNil(t, fe)
		assert.Equal(t, "email_recipients.2.address", fe.Key)
		assert.Equal(t, "invalid_format", fe.Code)
		assert.Equal(t, "wwwwwww", fe.Value)
	})

	t.Run("invalid webhook url produces structured error", func(t *testing.T) {
		err := validateWebhookRecipients([]models.WebhookRecipient{
			{Name: "ok", URL: "https://example.com/hook"},
			{Name: "broken", URL: "asdf"},
		})
		require.Error(t, err)
		fe := asAlertingFieldError(err)
		require.NotNil(t, fe)
		assert.Equal(t, "webhook_recipients.1.url", fe.Key)
		assert.Equal(t, "invalid_scheme", fe.Code)
		assert.Equal(t, "asdf", fe.Value)
	})

	t.Run("namespace to JSON path with array index", func(t *testing.T) {
		assert.Equal(t,
			"email_recipients.0.address",
			alertingJSONPath("AlertingConfigLayer.EmailRecipients[0].Address"),
		)
		assert.Equal(t,
			"webhook_recipients.3.url",
			alertingJSONPath("AlertingConfigLayer.WebhookRecipients[3].URL"),
		)
		assert.Equal(t,
			"telegram_recipients.0.bot_token",
			alertingJSONPath("AlertingConfigLayer.TelegramRecipients[0].BotToken"),
		)
	})
}
