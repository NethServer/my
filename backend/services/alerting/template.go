/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"

	"github.com/nethesis/my/backend/models"
)

// yamlEscape sanitizes a string for safe inclusion in single-quoted YAML values.
// It strips newlines/carriage returns and doubles single quotes.
func yamlEscape(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "'", "''")
	return s
}

var validSeverityKey = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// buildSeverityMatchers turns a list of severity narrowing values (e.g.
// `["critical","warning"]`) into Alertmanager matcher expressions appended to
// a system-scoped route. Empty list → no extra matcher (all severities).
// Single value → exact match (`severity="critical"`). Multiple values →
// regex alternation (`severity=~"critical|warning"`). Each value is
// validated against validSeverityKey; invalid characters drop the entry to
// avoid YAML injection (the SeverityOverride bind tag limits to oneof).
func buildSeverityMatchers(severities []string) []string {
	if len(severities) == 0 {
		return nil
	}
	clean := make([]string, 0, len(severities))
	for _, s := range severities {
		if validSeverityKey.MatchString(s) {
			clean = append(clean, s)
		}
	}
	if len(clean) == 0 {
		return nil
	}
	if len(clean) == 1 {
		return []string{`severity="` + clean[0] + `"`}
	}
	return []string{`severity=~"` + strings.Join(clean, "|") + `"`}
}

// routeEntry represents a single child route in the Alertmanager routing tree.
//
// MatcherKey/MatcherValue cover the primary scope (system_key or severity);
// empty MatcherKey means the global fallback route.
//
// ExtraMatchers carries additional fully-formed matcher expressions to AND
// alongside the primary one — used for system-scoped overrides that further
// narrow on severity (e.g. a system override targeting only `critical` alerts
// emits `system_key="X"` PLUS `severity="critical"`). Each entry is already
// YAML-escaped; the template renders them verbatim under `matchers:`.
type routeEntry struct {
	MatcherKey    string // "system_key" or "severity"; empty = global fallback
	MatcherValue  string
	ExtraMatchers []string
	ReceiverName  string // "blackhole" when notifications are disabled
}

// telegramEntry represents a single Telegram notification target inside a receiver.
type telegramEntry struct {
	BotToken string
	ChatID   int64
}

// receiverEntry represents a named Alertmanager receiver.
type receiverEntry struct {
	Name      string
	Emails    []string
	Webhooks  []string
	Telegrams []telegramEntry
}

// templateData holds all pre-computed values injected into the YAML template.
type templateData struct {
	SmtpSmarthost       string
	SmtpFrom            string
	SmtpAuthUsername    string
	SmtpAuthPassword    string
	SmtpRequireTLS      bool
	HistoryWebhookURL   string
	HistoryWebhookToken string
	Routes              []routeEntry
	Receivers           []receiverEntry
	// EmailTemplateLang is set when custom email templates are configured ("en" or "it").
	// An empty value means Alertmanager's built-in default templates are used.
	EmailTemplateLang string
}

const alertmanagerTemplate = `global:
  resolve_timeout: 1h
  smtp_smarthost: '{{ yamlEscape .SmtpSmarthost }}'
  smtp_from: '{{ yamlEscape .SmtpFrom }}'
  smtp_auth_username: '{{ yamlEscape .SmtpAuthUsername }}'
  smtp_auth_password: '{{ yamlEscape .SmtpAuthPassword }}'
  smtp_require_tls: {{ if .SmtpRequireTLS }}true{{ else }}false{{ end }}

route:
  receiver: 'blackhole'
  group_by: ['alertname', 'system_key']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 12h
  routes:
{{- if .HistoryWebhookURL }}
    - receiver: 'builtin-history'
      continue: true
{{- end }}
{{- range .Routes }}
{{- if .MatcherKey }}
    - matchers:
        - {{ .MatcherKey }}="{{ yamlEscape .MatcherValue }}"
{{- range .ExtraMatchers }}
        - {{ . }}
{{- end }}
      receiver: '{{ yamlEscape .ReceiverName }}'
      continue: false
{{- else }}
    - receiver: '{{ yamlEscape .ReceiverName }}'
      continue: false
{{- end }}
{{- end }}

receivers:
  - name: 'blackhole'
{{- if .HistoryWebhookURL }}

  - name: 'builtin-history'
    webhook_configs:
      - url: '{{ yamlEscape .HistoryWebhookURL }}'
        send_resolved: true
{{- if .HistoryWebhookToken }}
        http_config:
          authorization:
            type: Bearer
            credentials: '{{ yamlEscape .HistoryWebhookToken }}'
{{- end }}
{{- end }}
{{- range .Receivers }}

  - name: '{{ yamlEscape .Name }}'
{{- if .Emails }}
    email_configs:
{{- range .Emails }}
      - to: '{{ yamlEscape . }}'
        send_resolved: true
{{- if $.EmailTemplateLang }}
        html: '{{ "{{" }} template "alert.html" . {{ "}}" }}'
        text: '{{ "{{" }} template "alert.txt" . {{ "}}" }}'
        headers:
          Subject: '{{ "{{" }} template "alert.subject" . {{ "}}" }}'
{{- end }}
{{- end }}
{{- end }}
{{- if .Webhooks }}
    webhook_configs:
{{- range .Webhooks }}
      - url: '{{ yamlEscape . }}'
        send_resolved: true
{{- end }}
{{- end }}
{{- if .Telegrams }}
    telegram_configs:
{{- range .Telegrams }}
      - bot_token: '{{ yamlEscape .BotToken }}'
        chat_id: {{ .ChatID }}
        send_resolved: true
        parse_mode: 'HTML'
        message: '{{ "{{" }} template "telegram.message" . {{ "}}" }}'
{{- end }}
{{- end }}
{{- end }}
{{- if .EmailTemplateLang }}

templates:
  - 'firing_{{ .EmailTemplateLang }}.html'
  - 'resolved_{{ .EmailTemplateLang }}.html'
  - 'firing_{{ .EmailTemplateLang }}.txt'
  - 'resolved_{{ .EmailTemplateLang }}.txt'
  - '_dispatcher.tmpl'
  - 'telegram_{{ .EmailTemplateLang }}.tmpl'
{{- else }}

templates: []
{{- end }}
`

// effectiveSettings resolves mail/webhook/telegram settings for a given
// (system_key, severity) tuple. Hybrid semantics so the layered config model
// stays additive on recipients while preserving the historical capability to
// gate channels per scope:
//
//   - LISTS (mail_addresses, webhook/telegram receivers): UNION across global
//
//   - matching severity + matching system overrides, deduped. A descendant
//     adding a recipient at the system level extends the list; it does NOT
//     replace the upstream global recipients (the pre-Opzione 2 REPLACE
//     behaviour silently blinded ancestors and is no longer used).
//
//   - BOOL toggles (mail_enabled, webhook_enabled, telegram_enabled):
//     "more specific wins". A severity or system override with explicit
//     true/false replaces the global value for that scope. Allows Owner to
//     express "no email for severity=info" via `severities[info].mail_enabled=false`.
//     Non-Owner layers cannot reach this code with `false` because
//     NormalizeLayerForRole strips it at write time.
//
// Returns (mailEnabled, webhookEnabled, telegramEnabled, emails, webhooks, telegrams).
func effectiveSettings(cfg *models.AlertingConfig, systemKey, severity string) (bool, bool, bool, []string, []string, []telegramEntry) {
	mailEnabled := cfg.MailEnabled
	webhookEnabled := cfg.WebhookEnabled
	telegramEnabled := cfg.TelegramEnabled

	seenEmail := make(map[string]struct{}, len(cfg.MailAddresses))
	seenWebhook := make(map[string]struct{}, len(cfg.WebhookReceivers))
	seenTelegram := make(map[string]struct{}, len(cfg.TelegramReceivers))

	emails := make([]string, 0, len(cfg.MailAddresses))
	for _, e := range cfg.MailAddresses {
		if _, ok := seenEmail[e]; ok {
			continue
		}
		seenEmail[e] = struct{}{}
		emails = append(emails, e)
	}
	webhooks := make([]string, 0, len(cfg.WebhookReceivers))
	for _, w := range cfg.WebhookReceivers {
		if _, ok := seenWebhook[w.URL]; ok {
			continue
		}
		seenWebhook[w.URL] = struct{}{}
		webhooks = append(webhooks, w.URL)
	}
	telegrams := make([]telegramEntry, 0, len(cfg.TelegramReceivers))
	for _, tg := range cfg.TelegramReceivers {
		k := telegramKey(tg.BotToken, tg.ChatID)
		if _, ok := seenTelegram[k]; ok {
			continue
		}
		seenTelegram[k] = struct{}{}
		telegrams = append(telegrams, telegramEntry{BotToken: tg.BotToken, ChatID: tg.ChatID})
	}

	// Severity override: bools replace, lists union.
	for _, sv := range cfg.Severities {
		if sv.Severity == severity {
			if sv.MailEnabled != nil {
				mailEnabled = *sv.MailEnabled
			}
			if sv.WebhookEnabled != nil {
				webhookEnabled = *sv.WebhookEnabled
			}
			if sv.TelegramEnabled != nil {
				telegramEnabled = *sv.TelegramEnabled
			}
			for _, e := range sv.MailAddresses {
				if _, ok := seenEmail[e]; ok {
					continue
				}
				seenEmail[e] = struct{}{}
				emails = append(emails, e)
			}
			for _, w := range sv.WebhookReceivers {
				if _, ok := seenWebhook[w.URL]; ok {
					continue
				}
				seenWebhook[w.URL] = struct{}{}
				webhooks = append(webhooks, w.URL)
			}
			for _, tg := range sv.TelegramReceivers {
				k := telegramKey(tg.BotToken, tg.ChatID)
				if _, ok := seenTelegram[k]; ok {
					continue
				}
				seenTelegram[k] = struct{}{}
				telegrams = append(telegrams, telegramEntry{BotToken: tg.BotToken, ChatID: tg.ChatID})
			}
			break
		}
	}

	// System override: bools replace, lists union (cumulative on top of severity).
	for _, sys := range cfg.Systems {
		if sys.SystemKey == systemKey {
			if sys.MailEnabled != nil {
				mailEnabled = *sys.MailEnabled
			}
			if sys.WebhookEnabled != nil {
				webhookEnabled = *sys.WebhookEnabled
			}
			if sys.TelegramEnabled != nil {
				telegramEnabled = *sys.TelegramEnabled
			}
			for _, e := range sys.MailAddresses {
				if _, ok := seenEmail[e]; ok {
					continue
				}
				seenEmail[e] = struct{}{}
				emails = append(emails, e)
			}
			for _, w := range sys.WebhookReceivers {
				if _, ok := seenWebhook[w.URL]; ok {
					continue
				}
				seenWebhook[w.URL] = struct{}{}
				webhooks = append(webhooks, w.URL)
			}
			for _, tg := range sys.TelegramReceivers {
				k := telegramKey(tg.BotToken, tg.ChatID)
				if _, ok := seenTelegram[k]; ok {
					continue
				}
				seenTelegram[k] = struct{}{}
				telegrams = append(telegrams, telegramEntry{BotToken: tg.BotToken, ChatID: tg.ChatID})
			}
			break
		}
	}

	return mailEnabled, webhookEnabled, telegramEnabled, emails, webhooks, telegrams
}

func telegramKey(botToken string, chatID int64) string {
	return botToken + "|" + intToString(chatID)
}

// buildReceiver creates a receiverEntry with effective email, webhook, and telegram lists.
func buildReceiver(name string, mailEnabled, webhookEnabled, telegramEnabled bool, emails, webhooks []string, telegrams []telegramEntry) *receiverEntry {
	r := &receiverEntry{Name: name}
	if mailEnabled {
		r.Emails = emails
	}
	if webhookEnabled {
		r.Webhooks = webhooks
	}
	if telegramEnabled {
		r.Telegrams = telegrams
	}
	return r
}

// RenderConfig renders the Alertmanager YAML configuration from AlertingConfig
// and SMTP settings. If cfg is nil, it produces a blackhole-only config.
// historyWebhookURL is always included as a non-bypassable builtin receiver.
// historyWebhookToken is the Bearer token for the history webhook (optional).
func RenderConfig(smtpHost string, smtpPort int, smtpUser, smtpPass, smtpFrom string, smtpTLS bool, historyWebhookURL, historyWebhookToken string, cfg *models.AlertingConfig) (string, error) {
	smarthost := smtpHost
	if smtpPort > 0 {
		smarthost = smtpHost + ":" + strconv.Itoa(smtpPort)
	}

	data := templateData{
		SmtpSmarthost:       smarthost,
		SmtpFrom:            smtpFrom,
		SmtpAuthUsername:    smtpUser,
		SmtpAuthPassword:    smtpPass,
		SmtpRequireTLS:      smtpTLS,
		HistoryWebhookURL:   historyWebhookURL,
		HistoryWebhookToken: historyWebhookToken,
	}

	if cfg != nil {
		// Validate severity keys
		for _, sv := range cfg.Severities {
			if !validSeverityKey.MatchString(sv.Severity) {
				return "", fmt.Errorf("invalid severity key: %q", sv.Severity)
			}
		}

		// Set email template language (default to "en" when mail is used)
		lang := cfg.EmailTemplateLang
		if lang == "" {
			lang = "en"
		}
		data.EmailTemplateLang = lang

		// Per-system routes. Severities[] narrows the route to specific
		// severity levels by appending an extra matcher; absent (= "all
		// severities") emits a single matcher on system_key only.
		for _, sys := range cfg.Systems {
			mailOn, webhookOn, telegramOn, emails, webhooks, telegrams := effectiveSettings(cfg, sys.SystemKey, "")
			recvName := "system-" + sys.SystemKey + "-receiver"
			if !mailOn && !webhookOn && !telegramOn {
				recvName = "blackhole"
			}
			extra := buildSeverityMatchers(sys.Severities)
			data.Routes = append(data.Routes, routeEntry{
				MatcherKey:    "system_key",
				MatcherValue:  sys.SystemKey,
				ExtraMatchers: extra,
				ReceiverName:  recvName,
			})
			if recvName != "blackhole" {
				data.Receivers = append(data.Receivers, *buildReceiver(recvName, mailOn, webhookOn, telegramOn, emails, webhooks, telegrams))
			}
		}

		// Per-severity routes
		for _, sv := range cfg.Severities {
			mailOn, webhookOn, telegramOn, emails, webhooks, telegrams := effectiveSettings(cfg, "", sv.Severity)
			recvName := "severity-" + sv.Severity + "-receiver"
			if !mailOn && !webhookOn && !telegramOn {
				recvName = "blackhole"
			}
			data.Routes = append(data.Routes, routeEntry{
				MatcherKey:   "severity",
				MatcherValue: sv.Severity,
				ReceiverName: recvName,
			})
			if recvName != "blackhole" {
				data.Receivers = append(data.Receivers, *buildReceiver(recvName, mailOn, webhookOn, telegramOn, emails, webhooks, telegrams))
			}
		}

		// Global fallback route
		globalRecvName := "global-receiver"
		if !cfg.MailEnabled && !cfg.WebhookEnabled && !cfg.TelegramEnabled {
			globalRecvName = "blackhole"
		}
		data.Routes = append(data.Routes, routeEntry{
			MatcherKey:   "",
			ReceiverName: globalRecvName,
		})
		if globalRecvName != "blackhole" {
			globalEmails := cfg.MailAddresses
			globalWebhooks := make([]string, 0, len(cfg.WebhookReceivers))
			for _, w := range cfg.WebhookReceivers {
				globalWebhooks = append(globalWebhooks, w.URL)
			}
			globalTelegrams := make([]telegramEntry, 0, len(cfg.TelegramReceivers))
			for _, tg := range cfg.TelegramReceivers {
				globalTelegrams = append(globalTelegrams, telegramEntry{BotToken: tg.BotToken, ChatID: tg.ChatID})
			}
			data.Receivers = append(data.Receivers, *buildReceiver(globalRecvName, cfg.MailEnabled, cfg.WebhookEnabled, cfg.TelegramEnabled, globalEmails, globalWebhooks, globalTelegrams))
		}
	}

	funcMap := template.FuncMap{"yamlEscape": yamlEscape}
	tmpl, err := template.New("alertmanager").Funcs(funcMap).Parse(alertmanagerTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// --- YAML parsing structs (used only by ParseConfig) ---

type amEmailConfig struct {
	To string `yaml:"to"`
}
type amWebhookConfig struct {
	URL string `yaml:"url"`
}
type amTelegramConfig struct {
	BotToken string `yaml:"bot_token"`
	ChatID   int64  `yaml:"chat_id"`
}
type amReceiver struct {
	Name            string             `yaml:"name"`
	EmailConfigs    []amEmailConfig    `yaml:"email_configs"`
	WebhookConfigs  []amWebhookConfig  `yaml:"webhook_configs"`
	TelegramConfigs []amTelegramConfig `yaml:"telegram_configs"`
}
type amRoute struct {
	Receiver string    `yaml:"receiver"`
	Continue bool      `yaml:"continue"`
	Matchers []string  `yaml:"matchers"`
	Routes   []amRoute `yaml:"routes"`
}
type amConfig struct {
	Route     amRoute      `yaml:"route"`
	Receivers []amReceiver `yaml:"receivers"`
	Templates []string     `yaml:"templates"`
}

// parseMatcherValue extracts the value from a matcher string like `key="value"`.
func parseMatcherValue(matcher string) (key, value string) {
	// Supports both key="value" and key=value
	idx := strings.Index(matcher, "=")
	if idx < 0 {
		return "", ""
	}
	key = strings.TrimSpace(matcher[:idx])
	value = strings.Trim(strings.TrimSpace(matcher[idx+1:]), `"'`)
	return key, value
}

// ParseConfig parses an Alertmanager YAML configuration (as stored in Mimir)
// back into an AlertingConfig struct. Returns nil if the config is blackhole-only.
func ParseConfig(yamlStr string) (*models.AlertingConfig, error) {
	// Mimir wraps the config under alertmanager_config key
	var wrapper struct {
		AlertmanagerConfig string `yaml:"alertmanager_config"`
	}
	if err := yaml.Unmarshal([]byte(yamlStr), &wrapper); err == nil && wrapper.AlertmanagerConfig != "" {
		yamlStr = wrapper.AlertmanagerConfig
	}

	var am amConfig
	if err := yaml.Unmarshal([]byte(yamlStr), &am); err != nil {
		return nil, fmt.Errorf("parsing alertmanager config: %w", err)
	}

	// Build receiver lookup: name -> amReceiver
	receiverMap := make(map[string]amReceiver, len(am.Receivers))
	for _, r := range am.Receivers {
		receiverMap[r.Name] = r
	}

	cfg := &models.AlertingConfig{}
	hasAnyConfig := false

	for _, route := range am.Route.Routes {
		recv := route.Receiver

		// Skip builtin-history (internal, not user-configurable)
		if recv == "builtin-history" {
			continue
		}

		// Determine what this route matches
		var matchKey, matchValue string
		for _, m := range route.Matchers {
			k, v := parseMatcherValue(m)
			if k == "system_key" || k == "severity" {
				matchKey = k
				matchValue = v
				break
			}
		}

		mailEnabled := recv != "blackhole"
		webhookEnabled := recv != "blackhole"
		telegramEnabled := recv != "blackhole"
		var emails []string
		var webhooks []WebhookEntry
		var telegramReceivers []models.TelegramReceiver

		if recv != "blackhole" {
			r, ok := receiverMap[recv]
			if ok {
				for _, ec := range r.EmailConfigs {
					emails = append(emails, ec.To)
				}
				for _, wc := range r.WebhookConfigs {
					// Infer name from receiver name (best effort)
					name := strings.TrimSuffix(recv, "-receiver")
					webhooks = append(webhooks, WebhookEntry{Name: name, URL: wc.URL})
				}
				for _, tc := range r.TelegramConfigs {
					telegramReceivers = append(telegramReceivers, models.TelegramReceiver{BotToken: tc.BotToken, ChatID: tc.ChatID})
				}
			}
			mailEnabled = len(emails) > 0
			webhookEnabled = len(webhooks) > 0
			telegramEnabled = len(telegramReceivers) > 0
		}

		switch matchKey {
		case "system_key":
			hasAnyConfig = true
			bMailEnabled := mailEnabled
			bWebhookEnabled := webhookEnabled
			bTelegramEnabled := telegramEnabled
			override := models.SystemOverride{
				SystemKey:       matchValue,
				MailEnabled:     &bMailEnabled,
				WebhookEnabled:  &bWebhookEnabled,
				TelegramEnabled: &bTelegramEnabled,
			}
			override.MailAddresses = append(override.MailAddresses, emails...)
			for _, w := range webhooks {
				override.WebhookReceivers = append(override.WebhookReceivers, models.WebhookReceiver{Name: w.Name, URL: w.URL})
			}
			override.TelegramReceivers = append(override.TelegramReceivers, telegramReceivers...)
			cfg.Systems = append(cfg.Systems, override)

		case "severity":
			hasAnyConfig = true
			bMailEnabled := mailEnabled
			bWebhookEnabled := webhookEnabled
			bTelegramEnabled := telegramEnabled
			override := models.SeverityOverride{
				Severity:        matchValue,
				MailEnabled:     &bMailEnabled,
				WebhookEnabled:  &bWebhookEnabled,
				TelegramEnabled: &bTelegramEnabled,
			}
			override.MailAddresses = append(override.MailAddresses, emails...)
			for _, w := range webhooks {
				override.WebhookReceivers = append(override.WebhookReceivers, models.WebhookReceiver{Name: w.Name, URL: w.URL})
			}
			override.TelegramReceivers = append(override.TelegramReceivers, telegramReceivers...)
			cfg.Severities = append(cfg.Severities, override)

		default:
			// Global fallback route
			hasAnyConfig = true
			cfg.MailEnabled = mailEnabled
			cfg.WebhookEnabled = webhookEnabled
			cfg.TelegramEnabled = telegramEnabled
			cfg.MailAddresses = append(cfg.MailAddresses, emails...)
			for _, w := range webhooks {
				cfg.WebhookReceivers = append(cfg.WebhookReceivers, models.WebhookReceiver{Name: w.Name, URL: w.URL})
			}
			cfg.TelegramReceivers = append(cfg.TelegramReceivers, telegramReceivers...)
		}
	}

	if !hasAnyConfig {
		return nil, nil
	}

	// Detect email template language from the templates list.
	for _, t := range am.Templates {
		if strings.Contains(t, "_en.") {
			cfg.EmailTemplateLang = "en"
			break
		} else if strings.Contains(t, "_it.") {
			cfg.EmailTemplateLang = "it"
			break
		}
	}

	return cfg, nil
}

// WebhookEntry is a temporary struct used during YAML parsing.
type WebhookEntry struct {
	Name string
	URL  string
}
