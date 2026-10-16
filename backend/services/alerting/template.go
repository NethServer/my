/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"bytes"
	"fmt"
	"sort"
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

// routeEntry represents a single child route in the Alertmanager routing tree.
// MatcherKey/Value is the primary match expression (severity="X"); empty
// MatcherKey means the catch-all fallback route.
type routeEntry struct {
	MatcherKey   string
	MatcherValue string
	ReceiverName string
}

// customerRoute is one customer's sub-tree inside a reseller tenant's routing:
// a parent route matched on organization_id="<OrgID>" wrapping per-severity
// child routes. Receiver names are namespaced by OrgID so customers sharing a
// tenant never collide.
type customerRoute struct {
	OrgID          string
	ScopedRoutes   []scopedRoute
	SeverityRoutes []routeEntry
}

// scopedRoute sends one severity of some systems to the recipients scoped to
// those systems. It runs before the severity routes with continue=true, so
// the unscoped recipients still get the alert.
type scopedRoute struct {
	Severity     string
	SystemRegex  string
	ReceiverName string
}

// CustomerConfig pairs a customer org id with its effective (merged) alerting
// layer. RenderConfig turns each into a nested route + namespaced receivers
// inside a single reseller-tenant config.
type CustomerConfig struct {
	OrgID string
	Layer *models.AlertingConfigLayer
}

// emailEntry is a single email destination with its own per-recipient
// template overrides driven by the recipient's language and format
// preferences. format="" or "html" emits our html template plus our text
// template (multipart, html primary, text fallback). format="plain" emits
// our text template plus `html: ”` — the empty html: is mandatory because
// Alertmanager otherwise falls back to its built-in HTML template, which
// would override ours with the generic "Sent by Alertmanager" body.
type emailEntry struct {
	To       string
	Language string // "en" or "it" (resolved; never empty)
	UseHTML  bool   // true → include our html template; false → emit html: ''
}

// webhookEntry is a single webhook destination as it appears inside a
// receiver's webhook_configs.
type webhookEntry struct {
	URL string
}

// telegramEntry is a single Telegram destination as it appears inside a
// receiver's telegram_configs. Telegram messages currently always render
// in English; extend with a per-recipient Language field when needed.
// MessageThreadID > 0 pins the message to one topic of a forum supergroup.
type telegramEntry struct {
	BotToken        string
	ChatID          int64
	MessageThreadID int64
}

// receiverEntry represents a named Alertmanager receiver.
type receiverEntry struct {
	Name      string
	Emails    []emailEntry
	Webhooks  []webhookEntry
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
	Customers           []customerRoute
	Receivers           []receiverEntry
	// HasEmailReceivers is true when at least one receiver carries email
	// destinations. The Alertmanager `templates:` block is emitted only
	// then — pushing template files for tenants with no email recipients
	// would just be dead weight in Mimir.
	HasEmailReceivers bool
}

const alertmanagerTemplate = `global:
  resolve_timeout: 6h
  smtp_smarthost: '{{ yamlEscape .SmtpSmarthost }}'
  smtp_from: '{{ yamlEscape .SmtpFrom }}'
  smtp_auth_username: '{{ yamlEscape .SmtpAuthUsername }}'
  smtp_auth_password: '{{ yamlEscape .SmtpAuthPassword }}'
  smtp_require_tls: {{ if .SmtpRequireTLS }}true{{ else }}false{{ end }}

route:
  receiver: 'blackhole'
  group_by: ['organization_id', 'alertname', 'system_key']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 8760h
  routes:
{{- if .HistoryWebhookURL }}
    - receiver: 'builtin-history'
      continue: true
{{- end }}
{{- range .Customers }}
    - matchers:
        - organization_id="{{ yamlEscape .OrgID }}"
      receiver: 'blackhole'
      routes:
{{- range .ScopedRoutes }}
        - matchers:
            - severity="{{ yamlEscape .Severity }}"
            - system_id=~"{{ yamlEscape .SystemRegex }}"
          receiver: '{{ yamlEscape .ReceiverName }}'
          continue: true
{{- end }}
{{- range .SeverityRoutes }}
        - matchers:
            - {{ .MatcherKey }}="{{ yamlEscape .MatcherValue }}"
          receiver: '{{ yamlEscape .ReceiverName }}'
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
      - to: '{{ yamlEscape .To }}'
        send_resolved: true
{{- if .UseHTML }}
        html: '{{ "{{" }} template "alert_{{ .Language }}.html" . {{ "}}" }}'
{{- else }}
        html: ''
{{- end }}
        text: '{{ "{{" }} template "alert_{{ .Language }}.txt" . {{ "}}" }}'
        headers:
          Subject: '{{ "{{" }} template "alert_{{ .Language }}.subject" . {{ "}}" }}'
{{- end }}
{{- end }}
{{- if .Webhooks }}
    webhook_configs:
{{- range .Webhooks }}
      - url: '{{ yamlEscape .URL }}'
        send_resolved: true
{{- end }}
{{- end }}
{{- if .Telegrams }}
    telegram_configs:
{{- range .Telegrams }}
      - bot_token: '{{ yamlEscape .BotToken }}'
        chat_id: {{ .ChatID }}
{{- if gt .MessageThreadID 0 }}
        message_thread_id: {{ .MessageThreadID }}
{{- end }}
        send_resolved: true
        parse_mode: 'HTML'
        message: '{{ "{{" }} template "telegram_en.message" . {{ "}}" }}'
{{- end }}
{{- end }}
{{- end }}
{{- if .HasEmailReceivers }}

templates:
  - 'firing_en.html'
  - 'resolved_en.html'
  - 'firing_en.txt'
  - 'resolved_en.txt'
  - 'firing_it.html'
  - 'resolved_it.html'
  - 'firing_it.txt'
  - 'resolved_it.txt'
  - '_datetime.tmpl'
  - '_telegram.tmpl'
  - '_dispatcher.tmpl'
  - 'telegram_en.tmpl'
  - 'telegram_it.tmpl'
{{- else }}

templates: []
{{- end }}
`

// matchesSeverity reports whether a recipient with the given Severities[]
// configuration should receive alerts at the given severity. severities=[]
// means "all severities" — the recipient lands on every per-severity
// receiver.
func matchesSeverity(severities []string, target string) bool {
	if len(severities) == 0 {
		return true
	}
	for _, s := range severities {
		if s == target {
			return true
		}
	}
	return false
}

// resolveLanguage returns the language to render email templates with. An
// empty recipient.Language falls back to "en"; an unrecognised value falls
// back to "en" too (the model validator rejects unknown values before
// storage, this guards against any direct in-memory tampering).
func resolveLanguage(lang string) string {
	switch lang {
	case "it":
		return "it"
	default:
		return "en"
	}
}

// resolveUseHTML returns true when the rendered email_config should
// reference our html template. For format="plain" the renderer emits the
// literal `html: ”` instead, suppressing Alertmanager's default HTML
// fallback so only our text body is delivered.
func resolveUseHTML(format string) bool {
	return format != "plain"
}

// buildReceiver materialises a receiver entry for one severity bucket.
// Drops categories whose channel toggle is off at the global layer; drops
// the entire receiver (caller substitutes blackhole) when every list ends
// up empty.
func buildReceiver(
	name string,
	severity string,
	cfg *models.AlertingConfigLayer,
) receiverEntry {
	r := receiverEntry{Name: name}

	emailOn := cfg.Enabled.Email != nil && *cfg.Enabled.Email
	webhookOn := cfg.Enabled.Webhook != nil && *cfg.Enabled.Webhook
	telegramOn := cfg.Enabled.Telegram != nil && *cfg.Enabled.Telegram

	if emailOn {
		for _, rcp := range cfg.EmailRecipients {
			if !rcp.Scope.IsEmpty() || !matchesSeverity(rcp.Severities, severity) {
				continue
			}
			r.Emails = append(r.Emails, emailEntry{
				To:       rcp.Address,
				Language: resolveLanguage(rcp.Language),
				UseHTML:  resolveUseHTML(rcp.Format),
			})
		}
	}
	if webhookOn {
		for _, rcp := range cfg.WebhookRecipients {
			if !rcp.Scope.IsEmpty() || !matchesSeverity(rcp.Severities, severity) {
				continue
			}
			r.Webhooks = append(r.Webhooks, webhookEntry{URL: rcp.URL})
		}
	}
	if telegramOn {
		for _, rcp := range cfg.TelegramRecipients {
			if !rcp.Scope.IsEmpty() || !matchesSeverity(rcp.Severities, severity) {
				continue
			}
			r.Telegrams = append(r.Telegrams, telegramEntry{
				BotToken:        rcp.BotToken,
				ChatID:          rcp.ChatID,
				MessageThreadID: rcp.MessageThreadID,
			})
		}
	}
	return r
}

// RenderConfig renders the Alertmanager YAML configuration for one tenant
// from a merged AlertingConfigLayer. The renderer fans out per-severity
// routes (critical/warning/info), each pointing to a dedicated receiver
// whose lists are restricted to the recipients in scope for that severity.
//
// Recipients with severities=[] land on every per-severity receiver
// (they apply to "all severities"). Empty buckets are routed to blackhole.
//
// historyWebhookURL is always included as a non-bypassable builtin
// receiver attached at the top of the routes via continue=true so every
// alert is mirrored to the history sink regardless of user config.
func RenderConfig(
	smtpHost string,
	smtpPort int,
	smtpUser, smtpPass, smtpFrom string,
	smtpTLS bool,
	historyWebhookURL, historyWebhookToken string,
	customers []CustomerConfig,
) (string, error) {
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

	// One nested route per customer, matched on the organization_id label the
	// alerts carry. Within each customer, the same per-severity fan-out as
	// before, but with receiver names namespaced by org id. Unmatched
	// severities fall to the customer route's own receiver (blackhole).
	for _, cust := range customers {
		if cust.Layer == nil || cust.OrgID == "" {
			continue
		}
		cr := customerRoute{OrgID: cust.OrgID}
		hasReal := false
		for _, severity := range []string{"critical", "warning", "info"} {
			recv := buildReceiver(cust.OrgID+"-severity-"+severity+"-receiver", severity, cust.Layer)
			routes, scopedRecvs := buildScopedRoutes(cust.OrgID, severity, cust.Layer, recv)
			for _, sr := range scopedRecvs {
				hasReal = true
				if len(sr.Emails) > 0 {
					data.HasEmailReceivers = true
				}
				data.Receivers = append(data.Receivers, sr)
			}
			cr.ScopedRoutes = append(cr.ScopedRoutes, routes...)
			recvName := "blackhole"
			if len(recv.Emails) > 0 || len(recv.Webhooks) > 0 || len(recv.Telegrams) > 0 {
				recvName = recv.Name
				hasReal = true
				if len(recv.Emails) > 0 {
					data.HasEmailReceivers = true
				}
				data.Receivers = append(data.Receivers, recv)
			}
			cr.SeverityRoutes = append(cr.SeverityRoutes, routeEntry{
				MatcherKey:   "severity",
				MatcherValue: severity,
				ReceiverName: recvName,
			})
		}
		// A customer with no active recipient at any severity would only emit
		// an all-blackhole sub-tree; its alerts fall to the tenant root
		// receiver (blackhole) anyway, so skip it to keep the config lean.
		if !hasReal {
			continue
		}
		data.Customers = append(data.Customers, cr)
	}

	funcMap := template.FuncMap{"yamlEscape": yamlEscape}
	tmpl, err := template.New("alertmanager").Funcs(funcMap).Parse(alertmanagerTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering alertmanager template: %w", err)
	}
	return buf.String(), nil
}

// destination is one channel entry of a scoped recipient, keyed for dedup
// against the unscoped receiver and across scoped copies.
type destination struct {
	key      string
	email    *emailEntry
	webhook  *webhookEntry
	telegram *telegramEntry
}

func receiverKeys(r receiverEntry) map[string]struct{} {
	keys := map[string]struct{}{}
	for _, e := range r.Emails {
		keys["email|"+e.To] = struct{}{}
	}
	for _, w := range r.Webhooks {
		keys["webhook|"+w.URL] = struct{}{}
	}
	for _, t := range r.Telegrams {
		keys["telegram|"+t.BotToken+"|"+strconv.FormatInt(t.ChatID, 10)+"|"+strconv.FormatInt(t.MessageThreadID, 10)] = struct{}{}
	}
	return keys
}

// buildScopedRoutes renders the system-scoped recipients of one severity.
// Each system gets the scoped destinations that are not already on the
// unscoped receiver; systems sharing the same destinations share one route
// matched on system_id=~"a|b".
func buildScopedRoutes(orgID, severity string, cfg *models.AlertingConfigLayer, unscoped receiverEntry) ([]scopedRoute, []receiverEntry) {
	emailOn := cfg.Enabled.Email != nil && *cfg.Enabled.Email
	webhookOn := cfg.Enabled.Webhook != nil && *cfg.Enabled.Webhook
	telegramOn := cfg.Enabled.Telegram != nil && *cfg.Enabled.Telegram
	already := receiverKeys(unscoped)

	dests := map[string]destination{}
	perSystem := map[string]map[string]struct{}{}
	add := func(d destination, scope *models.RecipientScope) {
		if _, dup := already[d.key]; dup {
			return
		}
		if _, ok := dests[d.key]; !ok {
			dests[d.key] = d
		}
		for _, sys := range scope.SystemIDs {
			if perSystem[sys] == nil {
				perSystem[sys] = map[string]struct{}{}
			}
			perSystem[sys][d.key] = struct{}{}
		}
	}
	if emailOn {
		for _, rcp := range cfg.EmailRecipients {
			if rcp.Scope.IsEmpty() || !matchesSeverity(rcp.Severities, severity) {
				continue
			}
			add(destination{key: "email|" + rcp.Address, email: &emailEntry{
				To:       rcp.Address,
				Language: resolveLanguage(rcp.Language),
				UseHTML:  resolveUseHTML(rcp.Format),
			}}, rcp.Scope)
		}
	}
	if webhookOn {
		for _, rcp := range cfg.WebhookRecipients {
			if rcp.Scope.IsEmpty() || !matchesSeverity(rcp.Severities, severity) {
				continue
			}
			add(destination{key: "webhook|" + rcp.URL, webhook: &webhookEntry{URL: rcp.URL}}, rcp.Scope)
		}
	}
	if telegramOn {
		for _, rcp := range cfg.TelegramRecipients {
			if rcp.Scope.IsEmpty() || !matchesSeverity(rcp.Severities, severity) {
				continue
			}
			add(destination{
				key: "telegram|" + rcp.BotToken + "|" + strconv.FormatInt(rcp.ChatID, 10) + "|" + strconv.FormatInt(rcp.MessageThreadID, 10),
				telegram: &telegramEntry{
					BotToken:        rcp.BotToken,
					ChatID:          rcp.ChatID,
					MessageThreadID: rcp.MessageThreadID,
				},
			}, rcp.Scope)
		}
	}
	if len(perSystem) == 0 {
		return nil, nil
	}

	// Group systems by their destination set, in a stable order.
	type group struct {
		keys    []string
		systems []string
	}
	groups := map[string]*group{}
	order := []string{}
	systems := make([]string, 0, len(perSystem))
	for sys := range perSystem {
		systems = append(systems, sys)
	}
	sort.Strings(systems)
	for _, sys := range systems {
		keys := make([]string, 0, len(perSystem[sys]))
		for k := range perSystem[sys] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		sig := strings.Join(keys, "\n")
		g, ok := groups[sig]
		if !ok {
			g = &group{keys: keys}
			groups[sig] = g
			order = append(order, sig)
		}
		g.systems = append(g.systems, sys)
	}

	routes := make([]scopedRoute, 0, len(order))
	receivers := make([]receiverEntry, 0, len(order))
	for i, sig := range order {
		g := groups[sig]
		r := receiverEntry{Name: fmt.Sprintf("%s-severity-%s-scoped-%d-receiver", orgID, severity, i+1)}
		for _, k := range g.keys {
			d := dests[k]
			switch {
			case d.email != nil:
				r.Emails = append(r.Emails, *d.email)
			case d.webhook != nil:
				r.Webhooks = append(r.Webhooks, *d.webhook)
			case d.telegram != nil:
				r.Telegrams = append(r.Telegrams, *d.telegram)
			}
		}
		receivers = append(receivers, r)
		routes = append(routes, scopedRoute{
			Severity:     severity,
			SystemRegex:  strings.Join(g.systems, "|"),
			ReceiverName: r.Name,
		})
	}
	return routes, receivers
}
