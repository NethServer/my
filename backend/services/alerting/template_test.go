/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/nethesis/my/backend/models"
)

// smtpArgs returns the common SMTP args used across tests.
func smtpArgs() (string, int, string, string, string, bool) {
	return "smtp.example.com", 587, "user", "pass", "from@example.com", true
}

func boolPtr(b bool) *bool { return &b }

// isValidYAML checks that s is parseable YAML.
func isValidYAML(t *testing.T, s string) {
	t.Helper()
	var out interface{}
	require.NoError(t, yaml.Unmarshal([]byte(s), &out), "YAML must be valid")
}

// --- RenderConfig tests ---

func TestRenderConfig_NilCfg_BlackholeOnly(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	out, err := RenderConfig(host, port, user, pass, from, tls, "", "", nil)
	require.NoError(t, err)
	isValidYAML(t, out)

	assert.Contains(t, out, "receiver: 'blackhole'")
	assert.NotContains(t, out, "builtin-history")
	assert.NotContains(t, out, "system_key=")
	assert.NotContains(t, out, "severity=")
}

func TestRenderConfig_NilCfg_WithHistoryURL(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	out, err := RenderConfig(host, port, user, pass, from, tls, "http://history.example.com/hook", "", nil)
	require.NoError(t, err)
	isValidYAML(t, out)

	assert.Contains(t, out, "builtin-history")
	assert.Contains(t, out, "url: 'http://history.example.com/hook'")
	assert.Contains(t, out, "continue: true")
	// No user routes
	assert.NotContains(t, out, "system_key=")
	assert.NotContains(t, out, "severity=")
}

func TestRenderConfig_GlobalMailOnly(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	cfg := &models.AlertingConfig{
		MailEnabled:   true,
		MailAddresses: []string{"admin@example.com"},
	}
	out, err := RenderConfig(host, port, user, pass, from, tls, "", "", cfg)
	require.NoError(t, err)
	isValidYAML(t, out)

	assert.Contains(t, out, "receiver: 'global-receiver'")
	assert.Contains(t, out, "to: 'admin@example.com'")
	assert.NotContains(t, out, "webhook_configs")
}

func TestRenderConfig_GlobalWebhookOnly(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	cfg := &models.AlertingConfig{
		WebhookEnabled: true,
		WebhookReceivers: []models.WebhookReceiver{
			{Name: "slack", URL: "https://hooks.slack.com/abc"},
		},
	}
	out, err := RenderConfig(host, port, user, pass, from, tls, "", "", cfg)
	require.NoError(t, err)
	isValidYAML(t, out)

	assert.Contains(t, out, "receiver: 'global-receiver'")
	assert.Contains(t, out, "url: 'https://hooks.slack.com/abc'")
	assert.NotContains(t, out, "email_configs")
}

func TestRenderConfig_GlobalDisabled_BlackholeRoute(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	cfg := &models.AlertingConfig{
		MailEnabled:    false,
		WebhookEnabled: false,
	}
	out, err := RenderConfig(host, port, user, pass, from, tls, "", "", cfg)
	require.NoError(t, err)
	isValidYAML(t, out)

	// Global route must point to blackhole
	assert.Contains(t, out, "- receiver: 'blackhole'")
	// No named global-receiver
	assert.NotContains(t, out, "global-receiver")
}

func TestRenderConfig_HistoryAlwaysFires_EvenWhenDisabled(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	cfg := &models.AlertingConfig{
		MailEnabled:    false,
		WebhookEnabled: false,
	}
	out, err := RenderConfig(host, port, user, pass, from, tls, "http://history.example.com/hook", "", cfg)
	require.NoError(t, err)
	isValidYAML(t, out)

	assert.Contains(t, out, "builtin-history")
	assert.Contains(t, out, "continue: true")
}

func TestRenderConfig_SeverityOverride_DisablesWarning(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	cfg := &models.AlertingConfig{
		MailEnabled:   true,
		MailAddresses: []string{"global@example.com"},
		Severities: []models.SeverityOverride{
			{
				Severity:    "warning",
				MailEnabled: boolPtr(false),
			},
		},
	}
	out, err := RenderConfig(host, port, user, pass, from, tls, "", "", cfg)
	require.NoError(t, err)
	isValidYAML(t, out)

	// Warning severity route must point to blackhole
	lines := strings.Split(out, "\n")
	inWarningBlock := false
	for _, line := range lines {
		if strings.Contains(line, `severity="warning"`) {
			inWarningBlock = true
		}
		if inWarningBlock && strings.Contains(line, "receiver:") {
			assert.Contains(t, line, "blackhole", "warning severity should route to blackhole")
			break
		}
	}
	assert.Contains(t, out, `severity="warning"`)
}

func TestRenderConfig_SeverityOverride_CustomAddresses(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	cfg := &models.AlertingConfig{
		MailEnabled:   true,
		MailAddresses: []string{"global@example.com"},
		Severities: []models.SeverityOverride{
			{
				Severity:      "critical",
				MailEnabled:   boolPtr(true),
				MailAddresses: []string{"oncall@example.com"},
			},
		},
	}
	out, err := RenderConfig(host, port, user, pass, from, tls, "", "", cfg)
	require.NoError(t, err)
	isValidYAML(t, out)

	assert.Contains(t, out, "severity-critical-receiver")
	assert.Contains(t, out, "oncall@example.com")
	// Global address must not appear in the critical receiver
	idx := strings.Index(out, "severity-critical-receiver")
	afterCritical := out[idx:]
	nextReceiver := strings.Index(afterCritical[1:], "- name:")
	if nextReceiver > 0 {
		criticalSection := afterCritical[:nextReceiver]
		assert.NotContains(t, criticalSection, "global@example.com")
	}
}

func TestRenderConfig_SeverityOverride_InheritsGlobalAddresses(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	cfg := &models.AlertingConfig{
		MailEnabled:   true,
		MailAddresses: []string{"global@example.com"},
		Severities: []models.SeverityOverride{
			{
				Severity:    "critical",
				MailEnabled: boolPtr(true),
				// No MailAddresses → should inherit global
			},
		},
	}
	out, err := RenderConfig(host, port, user, pass, from, tls, "", "", cfg)
	require.NoError(t, err)
	isValidYAML(t, out)

	assert.Contains(t, out, "severity-critical-receiver")
	assert.Contains(t, out, "global@example.com")
}

func TestRenderConfig_SystemOverride(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	cfg := &models.AlertingConfig{
		MailEnabled:   true,
		MailAddresses: []string{"global@example.com"},
		Systems: []models.SystemOverride{
			{
				SystemKey:     "ns8-prod",
				MailEnabled:   boolPtr(true),
				MailAddresses: []string{"ops@example.com"},
			},
		},
	}
	out, err := RenderConfig(host, port, user, pass, from, tls, "", "", cfg)
	require.NoError(t, err)
	isValidYAML(t, out)

	assert.Contains(t, out, `system_key="ns8-prod"`)
	assert.Contains(t, out, "system-ns8-prod-receiver")
	assert.Contains(t, out, "ops@example.com")
}

func TestRenderConfig_SystemOverride_Disabled(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	cfg := &models.AlertingConfig{
		MailEnabled:   true,
		MailAddresses: []string{"global@example.com"},
		Systems: []models.SystemOverride{
			{
				SystemKey:   "ns8-silent",
				MailEnabled: boolPtr(false),
			},
		},
	}
	out, err := RenderConfig(host, port, user, pass, from, tls, "", "", cfg)
	require.NoError(t, err)
	isValidYAML(t, out)

	assert.Contains(t, out, `system_key="ns8-silent"`)
	// System route must point to blackhole
	lines := strings.Split(out, "\n")
	inSystemBlock := false
	for _, line := range lines {
		if strings.Contains(line, `system_key="ns8-silent"`) {
			inSystemBlock = true
		}
		if inSystemBlock && strings.Contains(line, "receiver:") {
			assert.Contains(t, line, "blackhole")
			break
		}
	}
}

func TestRenderConfig_InvalidSeverityKey(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	cfg := &models.AlertingConfig{
		MailEnabled:   true,
		MailAddresses: []string{"a@b.com"},
		Severities: []models.SeverityOverride{
			{Severity: "bad severity!"},
		},
	}
	_, err := RenderConfig(host, port, user, pass, from, tls, "", "", cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid severity key")
}

func TestRenderConfig_SmtpCredentialsRedacted(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	out, err := RenderConfig(host, port, user, pass, from, tls, "", "", nil)
	require.NoError(t, err)

	// SMTP creds appear in raw output; RedactSensitiveConfig removes them
	assert.Contains(t, out, "smtp_auth_username: 'user'")
	assert.Contains(t, out, "smtp_auth_password: 'pass'")
}

func TestRedactSensitiveConfig_BearerToken(t *testing.T) {
	input := `global:
  smtp_smarthost: 'smtp.example.com'
  smtp_auth_username: 'testuser'
  smtp_auth_password: 'testpass'

receivers:
  - name: 'builtin-history'
    webhook_configs:
      - url: 'http://example.com/webhook'
        http_config:
          authorization:
            type: Bearer
            credentials: 'secret-token-12345'`

	output := RedactSensitiveConfig(input)

	// Bearer token should be redacted
	assert.NotContains(t, output, "secret-token-12345")
	assert.Contains(t, output, "credentials: '[REDACTED]'")

	// SMTP credentials should also be redacted
	assert.NotContains(t, output, "testpass")
	assert.Contains(t, output, "smtp_auth_password: '[REDACTED]'")
}

// --- ParseConfig tests ---

func TestParseConfig_BlackholeOnly_ReturnsNil(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	yamlStr, err := RenderConfig(host, port, user, pass, from, tls, "", "", nil)
	require.NoError(t, err)

	cfg, err := ParseConfig(yamlStr)
	require.NoError(t, err)
	assert.Nil(t, cfg, "blackhole-only config should parse to nil")
}

func TestParseConfig_HistoryOnly_ReturnsNil(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	yamlStr, err := RenderConfig(host, port, user, pass, from, tls, "http://history.example.com/hook", "", nil)
	require.NoError(t, err)

	cfg, err := ParseConfig(yamlStr)
	require.NoError(t, err)
	assert.Nil(t, cfg, "history-only config should parse to nil")
}

func TestParseConfig_GlobalMail_Roundtrip(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	original := &models.AlertingConfig{
		MailEnabled:   true,
		MailAddresses: []string{"admin@example.com"},
	}
	yamlStr, err := RenderConfig(host, port, user, pass, from, tls, "", "", original)
	require.NoError(t, err)

	parsed, err := ParseConfig(yamlStr)
	require.NoError(t, err)
	require.NotNil(t, parsed)

	assert.True(t, parsed.MailEnabled)
	assert.False(t, parsed.WebhookEnabled)
	assert.Equal(t, []string{"admin@example.com"}, parsed.MailAddresses)
}

func TestParseConfig_GlobalWebhook_Roundtrip(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	original := &models.AlertingConfig{
		WebhookEnabled: true,
		WebhookReceivers: []models.WebhookReceiver{
			{Name: "slack", URL: "https://hooks.slack.com/abc"},
		},
	}
	yamlStr, err := RenderConfig(host, port, user, pass, from, tls, "", "", original)
	require.NoError(t, err)

	parsed, err := ParseConfig(yamlStr)
	require.NoError(t, err)
	require.NotNil(t, parsed)

	assert.False(t, parsed.MailEnabled)
	assert.True(t, parsed.WebhookEnabled)
	require.Len(t, parsed.WebhookReceivers, 1)
	assert.Equal(t, "https://hooks.slack.com/abc", parsed.WebhookReceivers[0].URL)
}

func TestParseConfig_SeverityOverride_Roundtrip(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	original := &models.AlertingConfig{
		MailEnabled:   true,
		MailAddresses: []string{"global@example.com"},
		Severities: []models.SeverityOverride{
			{
				Severity:      "critical",
				MailEnabled:   boolPtr(true),
				MailAddresses: []string{"oncall@example.com"},
			},
		},
	}
	yamlStr, err := RenderConfig(host, port, user, pass, from, tls, "", "", original)
	require.NoError(t, err)

	parsed, err := ParseConfig(yamlStr)
	require.NoError(t, err)
	require.NotNil(t, parsed)

	require.Len(t, parsed.Severities, 1)
	assert.Equal(t, "critical", parsed.Severities[0].Severity)
	require.NotNil(t, parsed.Severities[0].MailEnabled)
	assert.True(t, *parsed.Severities[0].MailEnabled)
	assert.Equal(t, []string{"oncall@example.com"}, parsed.Severities[0].MailAddresses)
}

func TestParseConfig_SystemOverride_Roundtrip(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	original := &models.AlertingConfig{
		MailEnabled:   true,
		MailAddresses: []string{"global@example.com"},
		Systems: []models.SystemOverride{
			{
				SystemKey:     "ns8-prod",
				MailEnabled:   boolPtr(true),
				MailAddresses: []string{"ops@example.com"},
			},
		},
	}
	yamlStr, err := RenderConfig(host, port, user, pass, from, tls, "", "", original)
	require.NoError(t, err)

	parsed, err := ParseConfig(yamlStr)
	require.NoError(t, err)
	require.NotNil(t, parsed)

	require.Len(t, parsed.Systems, 1)
	assert.Equal(t, "ns8-prod", parsed.Systems[0].SystemKey)
	require.NotNil(t, parsed.Systems[0].MailEnabled)
	assert.True(t, *parsed.Systems[0].MailEnabled)
	assert.Equal(t, []string{"ops@example.com"}, parsed.Systems[0].MailAddresses)
}

func TestParseConfig_MimirWrapperFormat(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	inner, err := RenderConfig(host, port, user, pass, from, tls, "", "", &models.AlertingConfig{
		MailEnabled:   true,
		MailAddresses: []string{"a@b.com"},
	})
	require.NoError(t, err)

	// Simulate Mimir wrapper format
	wrapped := "alertmanager_config: |\n"
	for _, line := range strings.Split(inner, "\n") {
		wrapped += "    " + line + "\n"
	}

	cfg, err := ParseConfig(wrapped)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.True(t, cfg.MailEnabled)
	assert.Equal(t, []string{"a@b.com"}, cfg.MailAddresses)
}

func TestParseConfig_GlobalDisabled_NotNil(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	cfg := &models.AlertingConfig{
		MailEnabled:    false,
		WebhookEnabled: false,
	}
	yamlStr, err := RenderConfig(host, port, user, pass, from, tls, "", "", cfg)
	require.NoError(t, err)

	parsed, err := ParseConfig(yamlStr)
	require.NoError(t, err)
	// Globally disabled but explicitly configured → non-nil
	require.NotNil(t, parsed)
	assert.False(t, parsed.MailEnabled)
	assert.False(t, parsed.WebhookEnabled)
}

func TestParseConfig_DisabledSeverity_BlackholeRoute(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	cfg := &models.AlertingConfig{
		MailEnabled:   true,
		MailAddresses: []string{"global@example.com"},
		Severities: []models.SeverityOverride{
			{
				Severity:    "warning",
				MailEnabled: boolPtr(false),
			},
		},
	}
	yamlStr, err := RenderConfig(host, port, user, pass, from, tls, "", "", cfg)
	require.NoError(t, err)

	parsed, err := ParseConfig(yamlStr)
	require.NoError(t, err)
	require.NotNil(t, parsed)

	require.Len(t, parsed.Severities, 1)
	assert.Equal(t, "warning", parsed.Severities[0].Severity)
	require.NotNil(t, parsed.Severities[0].MailEnabled)
	assert.False(t, *parsed.Severities[0].MailEnabled)
}

func TestParseConfig_InvalidYAML(t *testing.T) {
	_, err := ParseConfig("not: valid: yaml: ::::")
	require.Error(t, err)
}

// --- yamlEscape unit tests ---

func TestYamlEscape_SingleQuote(t *testing.T) {
	assert.Equal(t, "it''s", yamlEscape("it's"))
}

func TestYamlEscape_Newlines(t *testing.T) {
	assert.Equal(t, "noline", yamlEscape("no\nline"))
	assert.Equal(t, "noline", yamlEscape("no\rline"))
}

func TestYamlEscape_Empty(t *testing.T) {
	assert.Equal(t, "", yamlEscape(""))
}

// --- parseMatcherValue unit tests ---

func TestParseMatcherValue_DoubleQuoted(t *testing.T) {
	k, v := parseMatcherValue(`system_key="ns8-prod"`)
	assert.Equal(t, "system_key", k)
	assert.Equal(t, "ns8-prod", v)
}

func TestParseMatcherValue_Unquoted(t *testing.T) {
	k, v := parseMatcherValue("severity=critical")
	assert.Equal(t, "severity", k)
	assert.Equal(t, "critical", v)
}

func TestParseMatcherValue_NoEquals(t *testing.T) {
	k, v := parseMatcherValue("invalid")
	assert.Empty(t, k)
	assert.Empty(t, v)
}

// --- EmailTemplateLang / BuildTemplateFiles tests ---

func TestRenderConfig_DefaultsToEnglishTemplates(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	cfg := &models.AlertingConfig{
		MailEnabled:   true,
		MailAddresses: []string{"admin@example.com"},
		// EmailTemplateLang not set → should default to "en"
	}
	out, err := RenderConfig(host, port, user, pass, from, tls, "", "", cfg)
	require.NoError(t, err)
	isValidYAML(t, out)

	assert.Contains(t, out, "firing_en.html")
	assert.Contains(t, out, "resolved_en.html")
	assert.Contains(t, out, "firing_en.txt")
	assert.Contains(t, out, "resolved_en.txt")
	assert.Contains(t, out, "_dispatcher.tmpl")
	assert.Contains(t, out, `template "alert.html"`)
	assert.Contains(t, out, `template "alert.txt"`)
	assert.Contains(t, out, `template "alert.subject"`)
}

func TestRenderConfig_ItalianTemplates(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	cfg := &models.AlertingConfig{
		MailEnabled:       true,
		MailAddresses:     []string{"admin@example.com"},
		EmailTemplateLang: "it",
	}
	out, err := RenderConfig(host, port, user, pass, from, tls, "", "", cfg)
	require.NoError(t, err)
	isValidYAML(t, out)

	assert.Contains(t, out, "firing_it.html")
	assert.Contains(t, out, "resolved_it.html")
	assert.Contains(t, out, "firing_it.txt")
	assert.Contains(t, out, "resolved_it.txt")
	assert.NotContains(t, out, "firing_en.html")
}

func TestRenderConfig_NilCfg_NoTemplateSection(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	out, err := RenderConfig(host, port, user, pass, from, tls, "", "", nil)
	require.NoError(t, err)

	// nil cfg → blackhole-only, no email templates needed
	assert.Contains(t, out, "templates: []")
	assert.NotContains(t, out, "firing_en.html")
}

func TestRenderConfig_WebhookOnly_NoHtmlField(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	cfg := &models.AlertingConfig{
		WebhookEnabled: true,
		WebhookReceivers: []models.WebhookReceiver{
			{Name: "slack", URL: "https://hooks.slack.com/abc"},
		},
	}
	out, err := RenderConfig(host, port, user, pass, from, tls, "", "", cfg)
	require.NoError(t, err)
	isValidYAML(t, out)

	// No email_configs → no html: field in output
	assert.NotContains(t, out, `html:`)
}

func TestBuildTemplateFiles_English(t *testing.T) {
	files, err := BuildTemplateFiles("en", "https://my.nethesis.it")
	require.NoError(t, err)

	expected := []string{
		"firing_en.html",
		"resolved_en.html",
		"firing_en.txt",
		"resolved_en.txt",
		"_dispatcher.tmpl",
	}
	for _, name := range expected {
		content, ok := files[name]
		require.True(t, ok, "missing template file: %s", name)
		assert.NotEmpty(t, content, "template file %s is empty", name)
	}

	// Dispatcher must route to en templates
	assert.Contains(t, files["_dispatcher.tmpl"], `firing_en.html`)
	assert.Contains(t, files["_dispatcher.tmpl"], `resolved_en.html`)
	assert.Contains(t, files["_dispatcher.tmpl"], `firing_en.txt`)
	assert.Contains(t, files["_dispatcher.tmpl"], `resolved_en.txt`)
	assert.Contains(t, files["_dispatcher.tmpl"], `firing_en.subject`)
	assert.Contains(t, files["_dispatcher.tmpl"], `resolved_en.subject`)
}

func TestBuildTemplateFiles_Italian(t *testing.T) {
	files, err := BuildTemplateFiles("it", "https://my.nethesis.it")
	require.NoError(t, err)

	assert.Contains(t, files, "firing_it.html")
	assert.Contains(t, files, "resolved_it.html")
	assert.Contains(t, files["_dispatcher.tmpl"], `firing_it.html`)
}

func TestBuildTemplateFiles_EmptyLang_DefaultsToEnglish(t *testing.T) {
	files, err := BuildTemplateFiles("", "https://my.nethesis.it")
	require.NoError(t, err)
	assert.Contains(t, files, "firing_en.html")
}

func TestBuildTemplateFiles_InvalidLang_ReturnsError(t *testing.T) {
	_, err := BuildTemplateFiles("zz", "https://my.nethesis.it")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "zz")
}

func TestTemplateFiles_DefineNamedTemplates(t *testing.T) {
	files, err := BuildTemplateFiles("en", "https://my.nethesis.it")
	require.NoError(t, err)

	assert.Contains(t, files["firing_en.html"], `define "firing_en.html"`)
	assert.Contains(t, files["firing_en.html"], `define "firing_en.subject"`)
	assert.Contains(t, files["resolved_en.html"], `define "resolved_en.html"`)
	assert.Contains(t, files["resolved_en.html"], `define "resolved_en.subject"`)
	assert.Contains(t, files["firing_en.txt"], `define "firing_en.txt"`)
	assert.Contains(t, files["resolved_en.txt"], `define "resolved_en.txt"`)
}

func TestParseConfig_EmailTemplateLang_English(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	original := &models.AlertingConfig{
		MailEnabled:       true,
		MailAddresses:     []string{"a@b.com"},
		EmailTemplateLang: "en",
	}
	yamlStr, err := RenderConfig(host, port, user, pass, from, tls, "", "", original)
	require.NoError(t, err)

	parsed, err := ParseConfig(yamlStr)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, "en", parsed.EmailTemplateLang)
}

func TestParseConfig_EmailTemplateLang_Italian(t *testing.T) {
	host, port, user, pass, from, tls := smtpArgs()
	original := &models.AlertingConfig{
		MailEnabled:       true,
		MailAddresses:     []string{"a@b.com"},
		EmailTemplateLang: "it",
	}
	yamlStr, err := RenderConfig(host, port, user, pass, from, tls, "", "", original)
	require.NoError(t, err)

	parsed, err := ParseConfig(yamlStr)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, "it", parsed.EmailTemplateLang)
}

func TestWrapForMimirWithTemplates(t *testing.T) {
	templateFiles := map[string]string{
		"firing_en.html":   "{{ define \"firing_en.html\" }}test{{ end }}",
		"_dispatcher.tmpl": "{{ define \"alert.html\" }}x{{ end }}",
	}
	out := wrapForMimir("route:\n  receiver: blackhole", templateFiles)

	assert.Contains(t, out, "alertmanager_config: |")
	assert.Contains(t, out, "template_files:")
	assert.Contains(t, out, "firing_en.html:")
	assert.Contains(t, out, "_dispatcher.tmpl:")
}

func TestWrapForMimirWithoutTemplates(t *testing.T) {
	out := wrapForMimir("route:\n  receiver: blackhole", nil)

	assert.Contains(t, out, "alertmanager_config: |")
	assert.NotContains(t, out, "template_files:")
}
