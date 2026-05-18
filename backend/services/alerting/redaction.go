/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"net/url"

	"github.com/nethesis/my/backend/models"
)

// RedactedSecretPlaceholder is the literal returned in place of any sensitive
// value (telegram bot token, webhook URL path/query carrying a bearer secret)
// in audit-log snapshots. Audit rows are queried by admins; the unredacted
// values live in alert_config_layers and are read only by the renderer.
const RedactedSecretPlaceholder = "[REDACTED]"

// RedactLayerForAudit returns a copy of `layer` with secrets scrubbed, used by audit snapshots and the effective-config inspection response.
//
// Specifically:
//   - telegram_recipients[].bot_token → "[REDACTED]"
//   - webhook_recipients[].url        → scheme://host/[REDACTED] (path/query stripped)
//
// Email addresses are NOT scrubbed: they're already user-typed PII the
// admin is authorised to see, and they double as the dedup key.
func RedactLayerForAudit(layer models.AlertingConfigLayer) models.AlertingConfigLayer {
	out := layer
	if len(layer.WebhookRecipients) > 0 {
		out.WebhookRecipients = make([]models.WebhookRecipient, len(layer.WebhookRecipients))
		for i, w := range layer.WebhookRecipients {
			out.WebhookRecipients[i] = models.WebhookRecipient{
				Name:       w.Name,
				URL:        maskWebhookURL(w.URL),
				Severities: w.Severities,
			}
		}
	}
	if len(layer.TelegramRecipients) > 0 {
		out.TelegramRecipients = make([]models.TelegramRecipient, len(layer.TelegramRecipients))
		for i, t := range layer.TelegramRecipients {
			out.TelegramRecipients[i] = models.TelegramRecipient{
				BotToken:   RedactedSecretPlaceholder,
				ChatID:     t.ChatID,
				Severities: t.Severities,
			}
		}
	}
	return out
}

// RedactEffectiveConfigReport returns an API-safe copy: layers via RedactLayerForAudit, YAML via RedactSensitiveConfig; original untouched.
func RedactEffectiveConfigReport(r EffectiveConfigReport) EffectiveConfigReport {
	out := r
	out.Chain = make([]EffectiveLayerContribution, len(r.Chain))
	for i, c := range r.Chain {
		c.Layer = RedactLayerForAudit(c.Layer)
		out.Chain[i] = c
	}
	out.Effective = RedactLayerForAudit(r.Effective)
	out.YAML = RedactSensitiveConfig(r.YAML)
	return out
}

// maskWebhookURL keeps scheme + host + port (so the audit log records where
// the webhook went) but strips path, query, and fragment which routinely
// carry bearer-equivalent secrets (e.g. Slack incoming webhook IDs).
func maskWebhookURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return RedactedSecretPlaceholder
	}
	masked := url.URL{Scheme: u.Scheme, Host: u.Host}
	if u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return masked.String() + "/" + RedactedSecretPlaceholder
	}
	return masked.String()
}
