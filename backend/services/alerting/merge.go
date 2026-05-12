/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"strconv"
	"strings"

	"github.com/nethesis/my/backend/models"
)

// MergeLayers combines a sequence of organization-level config layers into
// the effective AlertingConfigLayer that the renderer turns into Mimir YAML.
//
// `layers` is ordered from least specific (Owner) to most specific (the
// tenant whose effective config we are computing). Order matters for dedup
// collisions (first occurrence wins for language/format) and for the
// "any layer says all severities" widening rule.
//
// Behaviour summary (security-critical: descendants can ADD, never REMOVE):
//   - bool channel toggles: OR — if any layer enables a channel, effective
//     is enabled. A nil or false in a deeper layer never disables what an
//     ancestor enabled.
//   - recipient lists: union with stable dedup. Dedup keys are
//     email→address, webhook→URL, telegram→(bot_token, chat_id).
//   - severities[] per recipient: union; if any contributing copy has
//     severities=[] ("all severities"), the merged copy widens back to [].
//   - language/format on a deduped email recipient: first-occurrence wins
//     (Owner intent is preserved; descendants cannot retitle ancestor mail).
//
// IMPORTANT: the merged result is server-internal. It feeds the Mimir YAML
// renderer and nothing else. /alerts/config never returns a merged view to
// any client — descendants only see their own layer.
func MergeLayers(layers []models.AlertingConfigLayer) models.AlertingConfigLayer {
	out := models.AlertingConfigLayer{
		EmailRecipients:    []models.EmailRecipient{},
		WebhookRecipients:  []models.WebhookRecipient{},
		TelegramRecipients: []models.TelegramRecipient{},
	}

	// OR accumulators for the three toggles; promoted to *bool at the end.
	emailEnabled := false
	webhookEnabled := false
	telegramEnabled := false

	// Index into out.* lists by dedup key so collisions can update the
	// existing entry's severities (union) without reordering.
	emailIdx := map[string]int{}
	webhookIdx := map[string]int{}
	telegramIdx := map[string]int{}

	for _, layer := range layers {
		if layer.Enabled.Email != nil && *layer.Enabled.Email {
			emailEnabled = true
		}
		if layer.Enabled.Webhook != nil && *layer.Enabled.Webhook {
			webhookEnabled = true
		}
		if layer.Enabled.Telegram != nil && *layer.Enabled.Telegram {
			telegramEnabled = true
		}

		for _, r := range layer.EmailRecipients {
			addr := strings.TrimSpace(r.Address)
			if addr == "" {
				continue
			}
			if i, seen := emailIdx[addr]; seen {
				out.EmailRecipients[i].Severities = unionSeverities(out.EmailRecipients[i].Severities, r.Severities)
				continue
			}
			emailIdx[addr] = len(out.EmailRecipients)
			out.EmailRecipients = append(out.EmailRecipients, models.EmailRecipient{
				Address:    addr,
				Severities: normalizeSeverities(r.Severities),
				Language:   r.Language,
				Format:     r.Format,
			})
		}
		for _, r := range layer.WebhookRecipients {
			url := strings.TrimSpace(r.URL)
			if url == "" {
				continue
			}
			if i, seen := webhookIdx[url]; seen {
				out.WebhookRecipients[i].Severities = unionSeverities(out.WebhookRecipients[i].Severities, r.Severities)
				continue
			}
			webhookIdx[url] = len(out.WebhookRecipients)
			out.WebhookRecipients = append(out.WebhookRecipients, models.WebhookRecipient{
				Name:       r.Name,
				URL:        url,
				Severities: normalizeSeverities(r.Severities),
			})
		}
		for _, r := range layer.TelegramRecipients {
			key := telegramKey(r)
			if i, seen := telegramIdx[key]; seen {
				out.TelegramRecipients[i].Severities = unionSeverities(out.TelegramRecipients[i].Severities, r.Severities)
				continue
			}
			telegramIdx[key] = len(out.TelegramRecipients)
			out.TelegramRecipients = append(out.TelegramRecipients, models.TelegramRecipient{
				BotToken:   r.BotToken,
				ChatID:     r.ChatID,
				Severities: normalizeSeverities(r.Severities),
			})
		}
	}

	out.Enabled = models.ChannelToggles{
		Email:    boolPtr(emailEnabled),
		Webhook:  boolPtr(webhookEnabled),
		Telegram: boolPtr(telegramEnabled),
	}
	return out
}

// NormalizeLayerForRole sanitises a layer about to be saved for a given org
// role so that descendants cannot encode subtractive settings.
//
// For any role except owner we drop *bool=&false on the three channel
// toggles. The user's intent ("disable email for my tenant") doesn't fit the
// additive model — only Owner can globally turn a channel off, and even
// then descendant layers may bring it back via OR. nil is the correct
// "no opinion" representation; we silently rewrite false → nil to keep the
// stored layer consistent with the contract.
func NormalizeLayerForRole(layer *models.AlertingConfigLayer, orgRole string) {
	if layer == nil {
		return
	}
	if strings.EqualFold(orgRole, "owner") {
		return
	}
	if layer.Enabled.Email != nil && !*layer.Enabled.Email {
		layer.Enabled.Email = nil
	}
	if layer.Enabled.Webhook != nil && !*layer.Enabled.Webhook {
		layer.Enabled.Webhook = nil
	}
	if layer.Enabled.Telegram != nil && !*layer.Enabled.Telegram {
		layer.Enabled.Telegram = nil
	}
}

// unionSeverities merges two severities slices with widening semantics:
// if either side encodes "all severities" (empty slice), the result is also
// empty (= all). Otherwise the union of the two sets is returned in the
// canonical order (critical, warning, info).
func unionSeverities(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	for _, v := range a {
		seen[v] = struct{}{}
	}
	for _, v := range b {
		seen[v] = struct{}{}
	}
	return canonicalSeverityOrder(seen)
}

// normalizeSeverities returns a copy of `s` in canonical order with duplicates
// dropped and unknown values stripped. Empty (or all-unknown) → empty slice,
// which the renderer interprets as "all severities".
func normalizeSeverities(s []string) []string {
	if len(s) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	for _, v := range s {
		seen[v] = struct{}{}
	}
	return canonicalSeverityOrder(seen)
}

func canonicalSeverityOrder(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for _, sev := range []string{"critical", "warning", "info"} {
		if _, ok := set[sev]; ok {
			out = append(out, sev)
		}
	}
	return out
}

func telegramKey(r models.TelegramRecipient) string {
	return r.BotToken + "|" + strconv.FormatInt(r.ChatID, 10)
}

func boolPtr(b bool) *bool { return &b }
