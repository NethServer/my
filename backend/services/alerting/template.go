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

// templateData holds all values injected into the alertmanager YAML template
type templateData struct {
	SmtpSmarthost    string
	SmtpFrom         string
	SmtpAuthUsername string
	SmtpAuthPassword string
	SmtpRequireTLS   bool
	Severities       map[string]models.SeverityConfig
}

const alertmanagerTemplate = `global:
  resolve_timeout: 5m
  smtp_smarthost: '{{ .SmtpSmarthost }}'
  smtp_from: '{{ .SmtpFrom }}'
  smtp_auth_username: '{{ .SmtpAuthUsername }}'
  smtp_auth_password: '{{ .SmtpAuthPassword }}'
  smtp_require_tls: {{ if .SmtpRequireTLS }}true{{ else }}false{{ end }}

route:
  receiver: 'blackhole'
  group_by: ['alertname', 'system_key']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 12h
{{- if .Severities }}

  routes:
{{- range $severity, $cfg := .Severities }}
    - matchers:
        - severity="{{ $severity }}"
{{- range $cfg.Exceptions }}
        - system_key!="{{ yamlEscape . }}"
{{- end }}
      receiver: '{{ yamlEscape $severity }}-receiver'
      continue: false
{{- end }}
{{- end }}

receivers:
  - name: 'blackhole'
{{- range $severity, $cfg := .Severities }}

  - name: '{{ yamlEscape $severity }}-receiver'
    email_configs:
{{- range $cfg.Emails }}
      - to: '{{ yamlEscape . }}'
        send_resolved: true
{{- end }}
{{- range $cfg.Webhooks }}
    webhook_configs:
      - url: '{{ yamlEscape .URL }}'
        send_resolved: true
{{- end }}
{{- end }}

templates: []
`

// RenderConfig renders the alertmanager YAML configuration from the request body
// and SMTP settings. If severities is nil, it produces a blackhole-only config.
func RenderConfig(smtpHost string, smtpPort int, smtpUser, smtpPass, smtpFrom string, smtpTLS bool, severities models.AlertingConfigRequest) (string, error) {
	for key := range severities {
		if !validSeverityKey.MatchString(key) {
			return "", fmt.Errorf("invalid severity key: %q", key)
		}
	}

	smarthost := smtpHost
	if smtpPort > 0 {
		smarthost = smtpHost + ":" + strconv.Itoa(smtpPort)
	}

	data := templateData{
		SmtpSmarthost:    smarthost,
		SmtpFrom:         smtpFrom,
		SmtpAuthUsername: smtpUser,
		SmtpAuthPassword: smtpPass,
		SmtpRequireTLS:   smtpTLS,
		Severities:       severities,
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
