/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
)

// systemKeyValidationPattern restricts SystemOverride.SystemKey to characters
// that are safe to embed verbatim in an Alertmanager matcher (rendered inside
// a double-quoted YAML scalar). Mirrors the regex enforced at the API
// boundary; duplicated here so any direct write through the storage layer
// (admin tools, future endpoints, migrations) gets the same protection.
var systemKeyValidationPattern = regexp.MustCompile(`^[A-Za-z0-9_:.\-]+$`)

// validSeverities is the canonical Alertmanager severity set we route on.
var validSeverities = map[string]struct{}{
	"critical": {},
	"warning":  {},
	"info":     {},
}

// WebhookReceiver represents a generic webhook receiver with name and URL
type WebhookReceiver struct {
	Name string `json:"name" binding:"required,max=100"`
	URL  string `json:"url" binding:"required,url,max=2048"`
}

// TelegramReceiver represents a Telegram bot notification target.
// BotToken is the secret token obtained from @BotFather (capped at 256 bytes
// to bound storage and YAML size; real Telegram tokens are ~46 chars).
// ChatID is the numeric identifier of the target chat (user, group, or channel).
type TelegramReceiver struct {
	BotToken string `json:"bot_token" binding:"required,max=256"`
	ChatID   int64  `json:"chat_id" binding:"required"`
}

// SeverityOverride defines mail/webhook/telegram settings for a specific severity level
type SeverityOverride struct {
	Severity          string             `json:"severity" binding:"required,oneof=critical warning info"`
	MailEnabled       *bool              `json:"mail_enabled"`
	WebhookEnabled    *bool              `json:"webhook_enabled"`
	TelegramEnabled   *bool              `json:"telegram_enabled"`
	MailAddresses     []string           `json:"mail_addresses,omitempty" binding:"max=50,dive,email,max=320"`
	WebhookReceivers  []WebhookReceiver  `json:"webhook_receivers,omitempty" binding:"max=20"`
	TelegramReceivers []TelegramReceiver `json:"telegram_receivers,omitempty" binding:"max=20"`
}

// SystemOverride defines mail/webhook/telegram settings for a specific system_key.
// SystemKey is bounded at 128 bytes to keep matcher rendering predictable;
// the regex enforced in the handler restricts the character set further.
type SystemOverride struct {
	SystemKey         string             `json:"system_key" binding:"required,max=128"`
	MailEnabled       *bool              `json:"mail_enabled"`
	WebhookEnabled    *bool              `json:"webhook_enabled"`
	TelegramEnabled   *bool              `json:"telegram_enabled"`
	MailAddresses     []string           `json:"mail_addresses,omitempty" binding:"max=50,dive,email,max=320"`
	WebhookReceivers  []WebhookReceiver  `json:"webhook_receivers,omitempty" binding:"max=20"`
	TelegramReceivers []TelegramReceiver `json:"telegram_receivers,omitempty" binding:"max=20"`
}

// AlertingConfigLayer is the per-organization layer used by the hierarchical
// alerting config model. Each org has one layer in alert_config_layers; the
// effective Mimir-bound config for a tenant is the merge of all layers
// walking up from the tenant to the Owner.
//
// Bool fields are *bool to keep "not set at this layer" distinguishable
// from "explicitly disabled". Merge rules (see services/alerting.MergeLayers):
//   - bool channel toggles: OR (additive enable; descendants cannot disable
//     a channel enabled by an ancestor). Normalised at write time so
//     non-Owner layers cannot store an explicit false.
//   - list fields (mail_addresses, webhook/telegram receivers, severities,
//     systems): union with dedup. Per-key merge for severity/system overrides.
//   - email_template_lang: deepest non-empty wins (per-tenant rendering
//     preference; descendants can override their own subtree).
type AlertingConfigLayer struct {
	MailEnabled       *bool              `json:"mail_enabled,omitempty"`
	WebhookEnabled    *bool              `json:"webhook_enabled,omitempty"`
	TelegramEnabled   *bool              `json:"telegram_enabled,omitempty"`
	MailAddresses     []string           `json:"mail_addresses,omitempty" binding:"max=50,dive,email,max=320"`
	WebhookReceivers  []WebhookReceiver  `json:"webhook_receivers,omitempty" binding:"max=20"`
	TelegramReceivers []TelegramReceiver `json:"telegram_receivers,omitempty" binding:"max=20"`
	Severities        []SeverityOverride `json:"severities,omitempty" binding:"max=10"`
	Systems           []SystemOverride   `json:"systems,omitempty" binding:"max=500"`
	EmailTemplateLang string             `json:"email_template_lang,omitempty" binding:"max=8"`
}

// Validate runs stateless format/structure checks that must hold for every
// write path into alert_config_layers. The handler also runs DNS-aware
// webhook URL checks; this Validate is the storage-layer backstop that
// guarantees regardless of where the layer originates (HTTP handler,
// provisioning path, admin tool, future endpoint), the persisted bytes
// satisfy the contract.
//
// What is NOT checked here: list-cardinality and per-string max length
// (already enforced by the `binding:"max=N"` tags via go-playground/validator
// at HTTP bind time), and DNS resolution of webhook hostnames (deliberate —
// kept at the handler level to avoid network calls during DB writes).
func (c *AlertingConfigLayer) Validate() error {
	for i, w := range c.WebhookReceivers {
		if err := validateStaticWebhookURL(w.URL); err != nil {
			return fmt.Errorf("webhook_receivers[%d] %q: %w", i, w.Name, err)
		}
	}
	for i, e := range c.MailAddresses {
		if err := validateEmailFormat(e); err != nil {
			return fmt.Errorf("mail_addresses[%d]: %w", i, err)
		}
	}
	for i, sv := range c.Severities {
		if _, ok := validSeverities[sv.Severity]; !ok {
			return fmt.Errorf("severities[%d]: invalid severity %q", i, sv.Severity)
		}
		for j, w := range sv.WebhookReceivers {
			if err := validateStaticWebhookURL(w.URL); err != nil {
				return fmt.Errorf("severities[%d].webhook_receivers[%d] %q: %w", i, j, w.Name, err)
			}
		}
		for j, e := range sv.MailAddresses {
			if err := validateEmailFormat(e); err != nil {
				return fmt.Errorf("severities[%d].mail_addresses[%d]: %w", i, j, err)
			}
		}
	}
	for i, sys := range c.Systems {
		if !systemKeyValidationPattern.MatchString(sys.SystemKey) {
			return fmt.Errorf("systems[%d]: system_key %q contains disallowed characters", i, sys.SystemKey)
		}
		for j, w := range sys.WebhookReceivers {
			if err := validateStaticWebhookURL(w.URL); err != nil {
				return fmt.Errorf("systems[%d].webhook_receivers[%d] %q: %w", i, j, w.Name, err)
			}
		}
		for j, e := range sys.MailAddresses {
			if err := validateEmailFormat(e); err != nil {
				return fmt.Errorf("systems[%d].mail_addresses[%d]: %w", i, j, err)
			}
		}
	}
	if c.EmailTemplateLang != "" {
		if !regexp.MustCompile(`^[a-zA-Z]{2,8}$`).MatchString(c.EmailTemplateLang) {
			return fmt.Errorf("email_template_lang must be a short language code")
		}
	}
	return nil
}

// validateStaticWebhookURL runs every check that does NOT require name
// resolution: scheme is http/https, no userinfo, host is non-empty and
// well-formed (IP literal or canonical FQDN). Network-aware checks
// (denylist resolution, IP private/loopback rejection of resolved DNS
// answers) are run by the handler.
func validateStaticWebhookURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid webhook url: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("webhook url must use http or https, got %q", u.Scheme)
	}
	if u.User != nil {
		return fmt.Errorf("webhook url must not contain credentials")
	}
	if u.Hostname() == "" {
		return fmt.Errorf("webhook url is missing a host")
	}
	return nil
}

func validateEmailFormat(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("email is empty")
	}
	if _, err := mail.ParseAddress(s); err != nil {
		return fmt.Errorf("invalid email %q: %w", s, err)
	}
	return nil
}

// AlertingConfig is the main configuration structure for alerting
type AlertingConfig struct {
	// Global settings
	MailEnabled       bool               `json:"mail_enabled"`
	WebhookEnabled    bool               `json:"webhook_enabled"`
	TelegramEnabled   bool               `json:"telegram_enabled"`
	MailAddresses     []string           `json:"mail_addresses" binding:"max=50,dive,email,max=320"`
	WebhookReceivers  []WebhookReceiver  `json:"webhook_receivers" binding:"max=20"`
	TelegramReceivers []TelegramReceiver `json:"telegram_receivers" binding:"max=20"`
	// Per-severity overrides
	Severities []SeverityOverride `json:"severities,omitempty" binding:"max=10"`
	// Per-system_key overrides
	Systems []SystemOverride `json:"systems,omitempty" binding:"max=500"`
	// Email template language: "en" (default) or "it"
	EmailTemplateLang string `json:"email_template_lang,omitempty" binding:"max=8"`
}

// AlertStatus represents the status metadata for an active alert from Alertmanager.
type AlertStatus struct {
	State       string   `json:"state"`
	SilencedBy  []string `json:"silencedBy"`
	InhibitedBy []string `json:"inhibitedBy"`
}

// ActiveAlert represents an active alert returned by Alertmanager.
type ActiveAlert struct {
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	Status       AlertStatus       `json:"status"`
	StartsAt     string            `json:"startsAt"`
	EndsAt       string            `json:"endsAt"`
	Fingerprint  string            `json:"fingerprint"`
	GeneratorURL string            `json:"generatorURL,omitempty"`
}

// CreateSystemAlertSilenceRequest identifies the active alert to silence.
// If EndAt is provided it takes precedence over DurationMinutes.
type CreateSystemAlertSilenceRequest struct {
	Fingerprint     string `json:"fingerprint" binding:"required"`
	Comment         string `json:"comment"`
	DurationMinutes int    `json:"duration_minutes" binding:"omitempty,min=1,max=10080"`
	EndAt           string `json:"end_at"` // optional RFC3339 datetime; overrides duration_minutes
}

// UpdateSystemAlertSilenceRequest is the payload for changing a silence's end time or comment.
type UpdateSystemAlertSilenceRequest struct {
	Comment string `json:"comment"`
	EndAt   string `json:"end_at" binding:"required"` // RFC3339 datetime
}

// AlertmanagerMatcher represents a single Alertmanager silence matcher.
type AlertmanagerMatcher struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	IsRegex bool   `json:"isRegex"`
}

// AlertmanagerSilenceRequest is the payload sent to Alertmanager when creating or updating a silence.
// When ID is non-empty, Alertmanager updates the existing silence instead of creating a new one.
type AlertmanagerSilenceRequest struct {
	ID        string                `json:"id,omitempty"`
	Matchers  []AlertmanagerMatcher `json:"matchers"`
	StartsAt  string                `json:"startsAt"`
	EndsAt    string                `json:"endsAt"`
	Comment   string                `json:"comment"`
	CreatedBy string                `json:"createdBy"`
}

// AlertmanagerSilenceResponse is the Alertmanager response for a created silence.
type AlertmanagerSilenceResponse struct {
	SilenceID string `json:"silenceID"`
}

// AlertmanagerSilenceStatus is the runtime state of a silence as reported by Alertmanager.
type AlertmanagerSilenceStatus struct {
	State string `json:"state"` // active | expired | pending
}

// AlertmanagerSilence represents a silence returned by Alertmanager.
type AlertmanagerSilence struct {
	ID        string                     `json:"id"`
	Matchers  []AlertmanagerMatcher      `json:"matchers"`
	StartsAt  string                     `json:"startsAt,omitempty"`
	EndsAt    string                     `json:"endsAt,omitempty"`
	UpdatedAt string                     `json:"updatedAt,omitempty"`
	CreatedBy string                     `json:"createdBy,omitempty"`
	Comment   string                     `json:"comment,omitempty"`
	Status    *AlertmanagerSilenceStatus `json:"status,omitempty"`
}
