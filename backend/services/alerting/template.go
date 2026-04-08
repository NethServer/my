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

// routeEntry represents a single child route in the Alertmanager routing tree.
type routeEntry struct {
	MatcherKey   string // "system_key" or "severity"; empty = global fallback
	MatcherValue string
	ReceiverName string // "blackhole" when notifications are disabled
}

// receiverEntry represents a named Alertmanager receiver.
type receiverEntry struct {
	Name     string
	Emails   []string
	Webhooks []string
}

// templateData holds all pre-computed values injected into the YAML template.
type templateData struct {
	SmtpSmarthost     string
	SmtpFrom          string
	SmtpAuthUsername  string
	SmtpAuthPassword  string
	SmtpRequireTLS    bool
	HistoryWebhookURL string
	Routes            []routeEntry
	Receivers         []receiverEntry
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
{{- end }}
{{- if .EmailTemplateLang }}

templates:
  - 'firing_{{ .EmailTemplateLang }}.html'
  - 'resolved_{{ .EmailTemplateLang }}.html'
  - 'firing_{{ .EmailTemplateLang }}.txt'
  - 'resolved_{{ .EmailTemplateLang }}.txt'
  - '_dispatcher.tmpl'
{{- else }}

templates: []
{{- end }}
`

// effectiveSettings resolves mail/webhook settings for a given system_key and
// severity, applying override priority: system > severity > global.
// Returns (mailEnabled, webhookEnabled, emails, webhooks).
func effectiveSettings(cfg *models.AlertingConfig, systemKey, severity string) (bool, bool, []string, []string) {
	mailEnabled := cfg.MailEnabled
	webhookEnabled := cfg.WebhookEnabled
	emails := cfg.MailAddresses
	webhooks := make([]string, 0, len(cfg.WebhookReceivers))
	for _, w := range cfg.WebhookReceivers {
		webhooks = append(webhooks, w.URL)
	}

	// Check severity override first (lower priority than system)
	for _, sv := range cfg.Severities {
		if sv.Severity == severity {
			if sv.MailEnabled != nil {
				mailEnabled = *sv.MailEnabled
			}
			if sv.WebhookEnabled != nil {
				webhookEnabled = *sv.WebhookEnabled
			}
			if len(sv.MailAddresses) > 0 {
				emails = sv.MailAddresses
			}
			if len(sv.WebhookReceivers) > 0 {
				webhooks = make([]string, 0, len(sv.WebhookReceivers))
				for _, w := range sv.WebhookReceivers {
					webhooks = append(webhooks, w.URL)
				}
			}
			break
		}
	}

	// Check system override (highest priority)
	for _, sys := range cfg.Systems {
		if sys.SystemKey == systemKey {
			if sys.MailEnabled != nil {
				mailEnabled = *sys.MailEnabled
			}
			if sys.WebhookEnabled != nil {
				webhookEnabled = *sys.WebhookEnabled
			}
			if len(sys.MailAddresses) > 0 {
				emails = sys.MailAddresses
			}
			if len(sys.WebhookReceivers) > 0 {
				webhooks = make([]string, 0, len(sys.WebhookReceivers))
				for _, w := range sys.WebhookReceivers {
					webhooks = append(webhooks, w.URL)
				}
			}
			break
		}
	}

	return mailEnabled, webhookEnabled, emails, webhooks
}

// buildReceiver creates a receiverEntry with effective email and webhook lists.
// Returns nil if both mail and webhooks are disabled.
func buildReceiver(name string, mailEnabled, webhookEnabled bool, emails, webhooks []string) *receiverEntry {
	r := &receiverEntry{Name: name}
	if mailEnabled {
		r.Emails = emails
	}
	if webhookEnabled {
		r.Webhooks = webhooks
	}
	return r
}

// RenderConfig renders the Alertmanager YAML configuration from AlertingConfig
// and SMTP settings. If cfg is nil, it produces a blackhole-only config.
// historyWebhookURL is always included as a non-bypassable builtin receiver.
func RenderConfig(smtpHost string, smtpPort int, smtpUser, smtpPass, smtpFrom string, smtpTLS bool, historyWebhookURL string, cfg *models.AlertingConfig) (string, error) {
	smarthost := smtpHost
	if smtpPort > 0 {
		smarthost = smtpHost + ":" + strconv.Itoa(smtpPort)
	}

	data := templateData{
		SmtpSmarthost:     smarthost,
		SmtpFrom:          smtpFrom,
		SmtpAuthUsername:  smtpUser,
		SmtpAuthPassword:  smtpPass,
		SmtpRequireTLS:    smtpTLS,
		HistoryWebhookURL: historyWebhookURL,
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

		// Per-system routes
		for _, sys := range cfg.Systems {
			mailOn, webhookOn, emails, webhooks := effectiveSettings(cfg, sys.SystemKey, "")
			recvName := "system-" + sys.SystemKey + "-receiver"
			if !mailOn && !webhookOn {
				recvName = "blackhole"
			}
			data.Routes = append(data.Routes, routeEntry{
				MatcherKey:   "system_key",
				MatcherValue: sys.SystemKey,
				ReceiverName: recvName,
			})
			if recvName != "blackhole" {
				data.Receivers = append(data.Receivers, *buildReceiver(recvName, mailOn, webhookOn, emails, webhooks))
			}
		}

		// Per-severity routes
		for _, sv := range cfg.Severities {
			mailOn, webhookOn, emails, webhooks := effectiveSettings(cfg, "", sv.Severity)
			recvName := "severity-" + sv.Severity + "-receiver"
			if !mailOn && !webhookOn {
				recvName = "blackhole"
			}
			data.Routes = append(data.Routes, routeEntry{
				MatcherKey:   "severity",
				MatcherValue: sv.Severity,
				ReceiverName: recvName,
			})
			if recvName != "blackhole" {
				data.Receivers = append(data.Receivers, *buildReceiver(recvName, mailOn, webhookOn, emails, webhooks))
			}
		}

		// Global fallback route
		globalRecvName := "global-receiver"
		if !cfg.MailEnabled && !cfg.WebhookEnabled {
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
			data.Receivers = append(data.Receivers, *buildReceiver(globalRecvName, cfg.MailEnabled, cfg.WebhookEnabled, globalEmails, globalWebhooks))
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
type amReceiver struct {
	Name           string            `yaml:"name"`
	EmailConfigs   []amEmailConfig   `yaml:"email_configs"`
	WebhookConfigs []amWebhookConfig `yaml:"webhook_configs"`
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
		var emails []string
		var webhooks []WebhookEntry

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
			}
			mailEnabled = len(emails) > 0
			webhookEnabled = len(webhooks) > 0
		}

		switch matchKey {
		case "system_key":
			hasAnyConfig = true
			bMailEnabled := mailEnabled
			bWebhookEnabled := webhookEnabled
			override := models.SystemOverride{
				SystemKey:      matchValue,
				MailEnabled:    &bMailEnabled,
				WebhookEnabled: &bWebhookEnabled,
			}
			override.MailAddresses = append(override.MailAddresses, emails...)
			for _, w := range webhooks {
				override.WebhookReceivers = append(override.WebhookReceivers, models.WebhookReceiver{Name: w.Name, URL: w.URL})
			}
			cfg.Systems = append(cfg.Systems, override)

		case "severity":
			hasAnyConfig = true
			bMailEnabled := mailEnabled
			bWebhookEnabled := webhookEnabled
			override := models.SeverityOverride{
				Severity:       matchValue,
				MailEnabled:    &bMailEnabled,
				WebhookEnabled: &bWebhookEnabled,
			}
			override.MailAddresses = append(override.MailAddresses, emails...)
			for _, w := range webhooks {
				override.WebhookReceivers = append(override.WebhookReceivers, models.WebhookReceiver{Name: w.Name, URL: w.URL})
			}
			cfg.Severities = append(cfg.Severities, override)

		default:
			// Global fallback route
			hasAnyConfig = true
			cfg.MailEnabled = mailEnabled
			cfg.WebhookEnabled = webhookEnabled
			cfg.MailAddresses = append(cfg.MailAddresses, emails...)
			for _, w := range webhooks {
				cfg.WebhookReceivers = append(cfg.WebhookReceivers, models.WebhookReceiver{Name: w.Name, URL: w.URL})
			}
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
