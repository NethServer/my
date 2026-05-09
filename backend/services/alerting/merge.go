/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"strings"

	"github.com/nethesis/my/backend/models"
)

// MergeLayers combines a sequence of organization-level config layers into a
// single effective AlertingConfig ready to be rendered as Mimir YAML.
//
// `layers` MUST be ordered from least specific (Owner) to most specific
// (the tenant whose effective config we are computing). The order matters
// for `email_template_lang` which uses "deepest non-empty wins"; for all
// other fields the merge is order-independent (OR for bools, union for
// lists, by-key merge for severity/system overrides).
//
// Behaviour summary (security-critical: descendants can ADD, never REMOVE):
//   - bool channel toggles: OR — if any layer enables a channel, effective
//     is enabled. A nil or false in a deeper layer never disables what an
//     ancestor enabled.
//   - mail_addresses / webhook_receivers / telegram_receivers: union with
//     stable dedup (preserves first-occurrence order). Receivers are
//     deduped by their identifying key (URL for webhooks, bot+chat for
//     telegram).
//   - severities[]: merge by severity key, recursing on bools (OR) and
//     lists (union dedup).
//   - systems[]: merge by system_key, same recursion.
//   - email_template_lang: deepest non-empty wins. Per-tenant rendering
//     preference; descendants can override their own subtree without
//     affecting siblings.
func MergeLayers(layers []models.AlertingConfigLayer) models.AlertingConfig {
	out := models.AlertingConfig{
		MailAddresses:     []string{},
		WebhookReceivers:  []models.WebhookReceiver{},
		TelegramReceivers: []models.TelegramReceiver{},
		Severities:        []models.SeverityOverride{},
		Systems:           []models.SystemOverride{},
	}

	mailEnabled := false
	webhookEnabled := false
	telegramEnabled := false

	addedEmails := make(map[string]struct{})
	addedWebhooks := make(map[string]struct{})  // key: name + "|" + url
	addedTelegrams := make(map[string]struct{}) // key: bot_token + "|" + chat_id

	// severity name -> merged override accumulator
	sevAcc := make(map[string]*severityAccum)
	// system_key -> merged override accumulator
	sysAcc := make(map[string]*systemAccum)

	for _, layer := range layers {
		// Bool toggles: OR semantics.
		if layer.MailEnabled != nil && *layer.MailEnabled {
			mailEnabled = true
		}
		if layer.WebhookEnabled != nil && *layer.WebhookEnabled {
			webhookEnabled = true
		}
		if layer.TelegramEnabled != nil && *layer.TelegramEnabled {
			telegramEnabled = true
		}

		// Lists: union dedup.
		for _, e := range layer.MailAddresses {
			e = strings.TrimSpace(e)
			if e == "" {
				continue
			}
			if _, seen := addedEmails[e]; seen {
				continue
			}
			addedEmails[e] = struct{}{}
			out.MailAddresses = append(out.MailAddresses, e)
		}
		for _, w := range layer.WebhookReceivers {
			// Dedup by URL only: two layers contributing the same destination
			// URL with different display names should NOT both be delivered
			// (would produce duplicate webhook calls). The first occurrence
			// (Owner-most-specific in chain order) wins on the display name.
			if _, seen := addedWebhooks[w.URL]; seen {
				continue
			}
			addedWebhooks[w.URL] = struct{}{}
			out.WebhookReceivers = append(out.WebhookReceivers, w)
		}
		for _, t := range layer.TelegramReceivers {
			key := t.BotToken + "|" + formatChatID(t.ChatID)
			if _, seen := addedTelegrams[key]; seen {
				continue
			}
			addedTelegrams[key] = struct{}{}
			out.TelegramReceivers = append(out.TelegramReceivers, t)
		}

		// Severity / system overrides: per-key merge.
		for _, sv := range layer.Severities {
			acc, ok := sevAcc[sv.Severity]
			if !ok {
				acc = newSeverityAccum(sv.Severity)
				sevAcc[sv.Severity] = acc
			}
			acc.absorb(sv)
		}
		for _, sys := range layer.Systems {
			acc, ok := sysAcc[sys.SystemKey]
			if !ok {
				acc = newSystemAccum(sys.SystemKey)
				sysAcc[sys.SystemKey] = acc
			}
			acc.absorb(sys)
		}

		// email_template_lang: deepest non-empty wins. We iterate from least
		// to most specific, so we just overwrite whenever a non-empty value
		// is provided.
		if lang := strings.TrimSpace(layer.EmailTemplateLang); lang != "" {
			out.EmailTemplateLang = lang
		}
	}

	out.MailEnabled = mailEnabled
	out.WebhookEnabled = webhookEnabled
	out.TelegramEnabled = telegramEnabled

	// Materialise accumulators in stable order: severities by name (critical
	// before warning before info), systems by system_key alphabetical.
	for _, sev := range []string{"critical", "warning", "info"} {
		if acc := sevAcc[sev]; acc != nil {
			out.Severities = append(out.Severities, acc.toModel())
		}
	}
	// Any unexpected severity name still appears, after the standard ones.
	for k, acc := range sevAcc {
		if k == "critical" || k == "warning" || k == "info" {
			continue
		}
		out.Severities = append(out.Severities, acc.toModel())
	}
	for k := range sysAcc {
		out.Systems = append(out.Systems, sysAcc[k].toModel())
	}

	return out
}

// NormalizeLayerForRole sanitises a layer about to be saved for a given org
// role so that descendants cannot encode subtractive settings.
//
// Concretely: for any role except owner we drop *bool=&false on channel
// toggles. The user's intent ("disable email for my tenant") doesn't fit the
// additive model — only Owner can globally turn a channel off, and even then
// other layers may bring it back via OR. Setting to nil is the right
// "no opinion" representation; we silently rewrite false → nil to keep the
// stored layer consistent with the contract.
func NormalizeLayerForRole(layer *models.AlertingConfigLayer, orgRole string) {
	if layer == nil {
		return
	}
	if strings.EqualFold(orgRole, "owner") {
		return
	}
	if layer.MailEnabled != nil && !*layer.MailEnabled {
		layer.MailEnabled = nil
	}
	if layer.WebhookEnabled != nil && !*layer.WebhookEnabled {
		layer.WebhookEnabled = nil
	}
	if layer.TelegramEnabled != nil && !*layer.TelegramEnabled {
		layer.TelegramEnabled = nil
	}
	for i := range layer.Severities {
		nilFalse(&layer.Severities[i].MailEnabled)
		nilFalse(&layer.Severities[i].WebhookEnabled)
		nilFalse(&layer.Severities[i].TelegramEnabled)
	}
	for i := range layer.Systems {
		nilFalse(&layer.Systems[i].MailEnabled)
		nilFalse(&layer.Systems[i].WebhookEnabled)
		nilFalse(&layer.Systems[i].TelegramEnabled)
	}
}

func nilFalse(p **bool) {
	if *p != nil && !**p {
		*p = nil
	}
}

func formatChatID(id int64) string {
	// Avoid pulling strconv just for one call — we render a 64-bit int.
	// The exact formatting only matters for dedup uniqueness.
	return intToString(id)
}

func intToString(n int64) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	digits := make([]byte, 0, 20)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	if negative {
		digits = append(digits, '-')
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}

// severityAccum accumulates merged state for one severity key across layers.
// Bools follow tri-state precedence (any true → true; else any false → false;
// else nil) so an Owner can disable a channel for a specific severity even
// when descendants haven't expressed an opinion. Non-Owner layers cannot
// reach this accumulator with explicit false because NormalizeLayerForRole
// strips it at write time.
//
// Lists are appended with dedup (additive — descendants can only ADD).
type severityAccum struct {
	severity        string
	mailEnabled     *bool
	webhookEnabled  *bool
	telegramEnabled *bool

	emails    []string
	webhooks  []models.WebhookReceiver
	telegrams []models.TelegramReceiver

	seenEmail    map[string]struct{}
	seenWebhook  map[string]struct{}
	seenTelegram map[string]struct{}
}

func newSeverityAccum(severity string) *severityAccum {
	return &severityAccum{
		severity:     severity,
		seenEmail:    map[string]struct{}{},
		seenWebhook:  map[string]struct{}{},
		seenTelegram: map[string]struct{}{},
	}
}

func (a *severityAccum) absorb(o models.SeverityOverride) {
	a.mailEnabled = mergeTristate(a.mailEnabled, o.MailEnabled)
	a.webhookEnabled = mergeTristate(a.webhookEnabled, o.WebhookEnabled)
	a.telegramEnabled = mergeTristate(a.telegramEnabled, o.TelegramEnabled)
	for _, e := range o.MailAddresses {
		if e = strings.TrimSpace(e); e == "" {
			continue
		}
		if _, ok := a.seenEmail[e]; ok {
			continue
		}
		a.seenEmail[e] = struct{}{}
		a.emails = append(a.emails, e)
	}
	for _, w := range o.WebhookReceivers {
		// Dedup by URL only (see global merge above for rationale).
		if _, ok := a.seenWebhook[w.URL]; ok {
			continue
		}
		a.seenWebhook[w.URL] = struct{}{}
		a.webhooks = append(a.webhooks, w)
	}
	for _, t := range o.TelegramReceivers {
		k := t.BotToken + "|" + intToString(t.ChatID)
		if _, ok := a.seenTelegram[k]; ok {
			continue
		}
		a.seenTelegram[k] = struct{}{}
		a.telegrams = append(a.telegrams, t)
	}
}

func (a *severityAccum) toModel() models.SeverityOverride {
	return models.SeverityOverride{
		Severity:          a.severity,
		MailEnabled:       a.mailEnabled,
		WebhookEnabled:    a.webhookEnabled,
		TelegramEnabled:   a.telegramEnabled,
		MailAddresses:     a.emails,
		WebhookReceivers:  a.webhooks,
		TelegramReceivers: a.telegrams,
	}
}

// mergeTristate combines an accumulated *bool with an incoming layer value.
// Precedence: any explicit true wins absolutely; else explicit false wins
// over nil; else nil. This lets Owner express "disable channel for this
// severity/system" while still allowing descendants to override with true
// (additive enable).
func mergeTristate(acc, in *bool) *bool {
	if in == nil {
		return acc
	}
	if *in {
		t := true
		return &t
	}
	if acc == nil {
		f := false
		return &f
	}
	return acc
}

// systemAccum mirrors severityAccum for system_key overrides; same tri-state
// precedence on bools and additive list semantics.
type systemAccum struct {
	systemKey       string
	mailEnabled     *bool
	webhookEnabled  *bool
	telegramEnabled *bool

	emails    []string
	webhooks  []models.WebhookReceiver
	telegrams []models.TelegramReceiver

	seenEmail    map[string]struct{}
	seenWebhook  map[string]struct{}
	seenTelegram map[string]struct{}
}

func newSystemAccum(systemKey string) *systemAccum {
	return &systemAccum{
		systemKey:    systemKey,
		seenEmail:    map[string]struct{}{},
		seenWebhook:  map[string]struct{}{},
		seenTelegram: map[string]struct{}{},
	}
}

func (a *systemAccum) absorb(o models.SystemOverride) {
	a.mailEnabled = mergeTristate(a.mailEnabled, o.MailEnabled)
	a.webhookEnabled = mergeTristate(a.webhookEnabled, o.WebhookEnabled)
	a.telegramEnabled = mergeTristate(a.telegramEnabled, o.TelegramEnabled)
	for _, e := range o.MailAddresses {
		if e = strings.TrimSpace(e); e == "" {
			continue
		}
		if _, ok := a.seenEmail[e]; ok {
			continue
		}
		a.seenEmail[e] = struct{}{}
		a.emails = append(a.emails, e)
	}
	for _, w := range o.WebhookReceivers {
		// Dedup by URL only (see global merge above for rationale).
		if _, ok := a.seenWebhook[w.URL]; ok {
			continue
		}
		a.seenWebhook[w.URL] = struct{}{}
		a.webhooks = append(a.webhooks, w)
	}
	for _, t := range o.TelegramReceivers {
		k := t.BotToken + "|" + intToString(t.ChatID)
		if _, ok := a.seenTelegram[k]; ok {
			continue
		}
		a.seenTelegram[k] = struct{}{}
		a.telegrams = append(a.telegrams, t)
	}
}

func (a *systemAccum) toModel() models.SystemOverride {
	return models.SystemOverride{
		SystemKey:         a.systemKey,
		MailEnabled:       a.mailEnabled,
		WebhookEnabled:    a.webhookEnabled,
		TelegramEnabled:   a.telegramEnabled,
		MailAddresses:     a.emails,
		WebhookReceivers:  a.webhooks,
		TelegramReceivers: a.telegrams,
	}
}
