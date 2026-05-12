/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"strings"
	"testing"

	"github.com/nethesis/my/backend/models"
)

func renderForTest(t *testing.T, cfg *models.AlertingConfigLayer) string {
	t.Helper()
	out, err := RenderConfig(
		"smtp.example", 587, "u", "p", "from@example", true,
		"", "",
		cfg,
	)
	if err != nil {
		t.Fatalf("RenderConfig error: %v", err)
	}
	return out
}

func TestRenderConfig_NilCfg_BlackholeOnly(t *testing.T) {
	out := renderForTest(t, nil)
	if !strings.Contains(out, "receiver: 'blackhole'") {
		t.Errorf("expected blackhole receiver, got:\n%s", out)
	}
	// No per-severity routes when cfg is nil.
	if strings.Contains(out, "severity=") {
		t.Errorf("nil cfg should not emit severity matchers, got:\n%s", out)
	}
}

func TestRenderConfig_PerSeverityFanout(t *testing.T) {
	cfg := &models.AlertingConfigLayer{
		Enabled: models.ChannelToggles{Email: ptrTrue()},
		EmailRecipients: []models.EmailRecipient{
			{Address: "all@x.com", Severities: []string{}, Language: "en", Format: "html"},
			{Address: "crit@x.com", Severities: []string{"critical"}, Language: "en", Format: "html"},
		},
	}
	out := renderForTest(t, cfg)
	for _, sev := range []string{"critical", "warning", "info"} {
		if !strings.Contains(out, "severity=\""+sev+"\"") {
			t.Errorf("expected route matcher for severity=%s, got:\n%s", sev, out)
		}
	}
	// all@ goes into every severity bucket; crit@ only into critical.
	// Locate the *receiver definition* (not the route match) by the
	// `name: '…-receiver'` line — the first occurrence is inside `routes:`
	// which only references the name, not the email_configs.
	criticalIdx := strings.Index(out, "name: 'severity-critical-receiver'")
	warningIdx := strings.Index(out, "name: 'severity-warning-receiver'")
	infoIdx := strings.Index(out, "name: 'severity-info-receiver'")
	if criticalIdx < 0 {
		t.Fatalf("missing critical receiver definition; got:\n%s", out)
	}
	endCritical := len(out)
	if warningIdx > criticalIdx {
		endCritical = warningIdx
	} else if infoIdx > criticalIdx {
		endCritical = infoIdx
	}
	critBlock := out[criticalIdx:endCritical]
	if !strings.Contains(critBlock, "to: 'crit@x.com'") {
		t.Errorf("crit@ should be on critical receiver, got:\n%s", critBlock)
	}
	if !strings.Contains(critBlock, "to: 'all@x.com'") {
		t.Errorf("all@ (severities=[]) should be on critical receiver, got:\n%s", critBlock)
	}
}

func TestRenderConfig_FormatPlain_EmitsEmptyHTML(t *testing.T) {
	cfg := &models.AlertingConfigLayer{
		Enabled: models.ChannelToggles{Email: ptrTrue()},
		EmailRecipients: []models.EmailRecipient{
			{Address: "p@x.com", Severities: []string{"critical"}, Language: "en", Format: "plain"},
		},
	}
	out := renderForTest(t, cfg)
	critIdx := strings.Index(out, "to: 'p@x.com'")
	if critIdx < 0 {
		t.Fatalf("recipient not found, got:\n%s", out)
	}
	tail := out[critIdx : critIdx+400]
	// Plain format must emit `html: ''` explicitly — Alertmanager's default
	// HTML template overrides ours when html: is absent (see emailEntry doc).
	if !strings.Contains(tail, "html: ''") {
		t.Errorf("plain format must emit html: '' to suppress Alertmanager default, got:\n%s", tail)
	}
	if !strings.Contains(tail, "text: '{{ template \"alert_en.txt\"") {
		t.Errorf("plain format must still include text: dispatcher reference, got:\n%s", tail)
	}
}

func TestRenderConfig_FormatHTMLDefault_IncludesBoth(t *testing.T) {
	cfg := &models.AlertingConfigLayer{
		Enabled: models.ChannelToggles{Email: ptrTrue()},
		EmailRecipients: []models.EmailRecipient{
			{Address: "h@x.com", Severities: []string{"critical"}, Language: "it"},
		},
	}
	out := renderForTest(t, cfg)
	if !strings.Contains(out, "html: '{{ template \"alert_it.html\"") {
		t.Errorf("default format=html missing html: reference, got:\n%s", out)
	}
	if !strings.Contains(out, "text: '{{ template \"alert_it.txt\"") {
		t.Errorf("default format=html missing text: fallback, got:\n%s", out)
	}
}

func TestRenderConfig_EmailDisabledDropsEmailConfigs(t *testing.T) {
	cfg := &models.AlertingConfigLayer{
		Enabled: models.ChannelToggles{Email: ptrFalse(), Webhook: ptrTrue()},
		EmailRecipients: []models.EmailRecipient{
			{Address: "a@x.com", Severities: []string{"critical"}},
		},
		WebhookRecipients: []models.WebhookRecipient{
			{Name: "w", URL: "https://hooks.example/x", Severities: []string{"critical"}},
		},
	}
	out := renderForTest(t, cfg)
	if strings.Contains(out, "to: 'a@x.com'") {
		t.Errorf("channel toggle off must drop email_configs, got:\n%s", out)
	}
	if !strings.Contains(out, "url: 'https://hooks.example/x'") {
		t.Errorf("webhook toggle on must still emit webhook_configs, got:\n%s", out)
	}
}

func TestRenderConfig_TelegramReferencesEnTemplate(t *testing.T) {
	cfg := &models.AlertingConfigLayer{
		Enabled: models.ChannelToggles{Telegram: ptrTrue()},
		TelegramRecipients: []models.TelegramRecipient{
			{BotToken: "tok", ChatID: -100, Severities: []string{}},
		},
	}
	out := renderForTest(t, cfg)
	if !strings.Contains(out, "{{ template \"telegram_en.message\"") {
		t.Errorf("telegram message dispatcher should be english-only, got:\n%s", out)
	}
}

func TestRenderConfig_HistoryWebhook(t *testing.T) {
	out, err := RenderConfig(
		"smtp.example", 587, "u", "p", "from@example", true,
		"https://history.example/sink", "secret-bearer",
		nil,
	)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(out, "name: 'builtin-history'") {
		t.Errorf("history receiver should be emitted when URL set, got:\n%s", out)
	}
	if !strings.Contains(out, "credentials: 'secret-bearer'") {
		t.Errorf("bearer token should propagate to webhook_configs, got:\n%s", out)
	}
}

func TestBuildTemplateFiles_AllLanguagesPresent(t *testing.T) {
	files, err := BuildTemplateFiles("https://app.example")
	if err != nil {
		t.Fatalf("BuildTemplateFiles error: %v", err)
	}
	want := []string{
		"firing_en.html", "resolved_en.html", "firing_en.txt", "resolved_en.txt",
		"firing_it.html", "resolved_it.html", "firing_it.txt", "resolved_it.txt",
		"telegram_en.tmpl", "telegram_it.tmpl",
		"_dispatcher.tmpl",
	}
	for _, name := range want {
		if _, ok := files[name]; !ok {
			t.Errorf("missing template file %s", name)
		}
	}
}

func TestBuildTemplateFiles_DispatcherDefinesPerLanguage(t *testing.T) {
	files, err := BuildTemplateFiles("https://app.example")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	disp := files["_dispatcher.tmpl"]
	for _, name := range []string{
		`define "alert_en.html"`, `define "alert_en.txt"`, `define "alert_en.subject"`,
		`define "alert_it.html"`, `define "alert_it.txt"`, `define "alert_it.subject"`,
	} {
		if !strings.Contains(disp, name) {
			t.Errorf("dispatcher missing %q", name)
		}
	}
}

func TestYamlEscape_SingleQuote(t *testing.T) {
	if got := yamlEscape("a'b"); got != "a''b" {
		t.Errorf("yamlEscape doubled single quote: got %q want %q", got, "a''b")
	}
}

func TestYamlEscape_NewlineStripped(t *testing.T) {
	if got := yamlEscape("a\nb\rc"); got != "abc" {
		t.Errorf("yamlEscape stripped newlines: got %q want %q", got, "abc")
	}
}
