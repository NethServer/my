/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"net/url"

	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/models"
)

// RedactedSecretPlaceholder is the literal returned in lieu of any sensitive
// value (Telegram bot tokens, webhook URL paths/queries) when serializing a
// layer to a downstream actor.
const RedactedSecretPlaceholder = "[REDACTED]"

// RedactLayerForDownstream returns a copy of `layer` with secret-bearing
// fields scrubbed for safe display to downstream tenants (or any caller that
// is NOT the org that owns the layer).
//
// What gets scrubbed and why:
//   - TelegramReceiver.BotToken: full bot impersonation if leaked. Replaced
//     with [REDACTED].
//   - WebhookReceiver.URL: path + query frequently carry bearer tokens
//     (Slack/Teams/Discord webhook URLs are bearer-equivalent). The host
//     is preserved so the UI can show "this webhook goes to hooks.slack.com"
//     without leaking the secret. Scheme + host kept; path/query/fragment
//     stripped.
//
// All other fields (recipient names, severity overrides, system_keys,
// email addresses) are preserved verbatim — they are intentionally visible
// to descendants for transparency ("you cannot remove this email; it was set
// by your distributor").
func RedactLayerForDownstream(layer models.AlertingConfigLayer) models.AlertingConfigLayer {
	out := layer
	out.TelegramReceivers = redactTelegramReceivers(layer.TelegramReceivers)
	out.WebhookReceivers = redactWebhookReceivers(layer.WebhookReceivers)
	out.Severities = redactSeverities(layer.Severities)
	out.Systems = redactSystems(layer.Systems)
	return out
}

// RedactRecordForDownstream returns a copy of `rec` with both the embedded
// layer redacted (see RedactLayerForDownstream) AND the audit metadata
// `UpdatedByUserID` stripped so descendants don't learn the upstream
// admin's logto user id. UpdatedByName is preserved (display name only —
// useful for the UI to show "set by Mario Rossi").
func RedactRecordForDownstream(rec entities.AlertConfigLayerRecord) entities.AlertConfigLayerRecord {
	out := rec
	out.UpdatedByUserID = nil
	out.Config = RedactLayerForDownstream(rec.Config)
	return out
}

// RedactRecordsForDownstream applies RedactRecordForDownstream to a slice.
func RedactRecordsForDownstream(records []entities.AlertConfigLayerRecord) []entities.AlertConfigLayerRecord {
	out := make([]entities.AlertConfigLayerRecord, len(records))
	for i, r := range records {
		out[i] = RedactRecordForDownstream(r)
	}
	return out
}

// RedactConfigForDownstream redacts a merged effective AlertingConfig before
// returning it to a downstream caller. Same masking rules as the layer
// variant; used for the response payload of GET /alerts/config/effective so
// the tokens and webhook URL secrets that flow into the rendered Mimir YAML
// (which is server-side only) never leak through the user-facing JSON.
func RedactConfigForDownstream(cfg models.AlertingConfig) models.AlertingConfig {
	out := cfg
	out.TelegramReceivers = redactTelegramReceivers(cfg.TelegramReceivers)
	out.WebhookReceivers = redactWebhookReceivers(cfg.WebhookReceivers)
	out.Severities = redactSeverities(cfg.Severities)
	out.Systems = redactSystems(cfg.Systems)
	return out
}

func redactTelegramReceivers(in []models.TelegramReceiver) []models.TelegramReceiver {
	if len(in) == 0 {
		return in
	}
	out := make([]models.TelegramReceiver, len(in))
	for i, t := range in {
		out[i] = models.TelegramReceiver{
			BotToken: RedactedSecretPlaceholder,
			ChatID:   t.ChatID,
		}
	}
	return out
}

func redactWebhookReceivers(in []models.WebhookReceiver) []models.WebhookReceiver {
	if len(in) == 0 {
		return in
	}
	out := make([]models.WebhookReceiver, len(in))
	for i, w := range in {
		out[i] = models.WebhookReceiver{
			Name: w.Name,
			URL:  maskWebhookURL(w.URL),
		}
	}
	return out
}

// maskWebhookURL keeps scheme + host + port (so the UI can show where the
// webhook goes) but strips path, query, and fragment which routinely carry
// bearer-equivalent secrets (e.g. Slack incoming webhook IDs).
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

func redactSeverities(in []models.SeverityOverride) []models.SeverityOverride {
	if len(in) == 0 {
		return in
	}
	out := make([]models.SeverityOverride, len(in))
	for i, s := range in {
		out[i] = s
		out[i].TelegramReceivers = redactTelegramReceivers(s.TelegramReceivers)
		out[i].WebhookReceivers = redactWebhookReceivers(s.WebhookReceivers)
	}
	return out
}

func redactSystems(in []models.SystemOverride) []models.SystemOverride {
	if len(in) == 0 {
		return in
	}
	out := make([]models.SystemOverride, len(in))
	for i, s := range in {
		out[i] = s
		out[i].TelegramReceivers = redactTelegramReceivers(s.TelegramReceivers)
		out[i].WebhookReceivers = redactWebhookReceivers(s.WebhookReceivers)
	}
	return out
}
