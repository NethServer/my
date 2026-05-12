/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"fmt"
	"net/mail"
	"net/url"
	"strings"
)

// validSeverities is the canonical Alertmanager severity set we route on.
// Recipients carry a `severities[]` field; empty means "all severities".
var validSeverities = map[string]struct{}{
	"critical": {},
	"warning":  {},
	"info":     {},
}

// validLanguages restricts EmailRecipient.Language to the set of templates
// shipped under services/alerting/templates/.
var validLanguages = map[string]struct{}{
	"en": {},
	"it": {},
}

// validFormats restricts EmailRecipient.Format. "html" emits only `html:`
// in the rendered email_configs entry (Alertmanager generates an
// equivalent text/plain alternative automatically but we keep the wire
// behavior explicit). "plain" emits only `text:` so spartan mailboxes get
// a stripped-down body.
var validFormats = map[string]struct{}{
	"html":  {},
	"plain": {},
}

// ChannelToggles is the per-layer enable/disable triplet. *bool keeps
// "not set at this layer / inherit from above" distinguishable from
// "explicitly false". The handler normalises any explicit `false` from
// non-Owner layers to nil before persisting (additive contract: only
// Owner can disable a channel globally).
type ChannelToggles struct {
	Email    *bool `json:"email"`
	Webhook  *bool `json:"webhook"`
	Telegram *bool `json:"telegram"`
}

// EmailRecipient is a single email destination with its own routing scope.
// Severities=[] means "all severities" — the recipient lands on the global
// receiver in Alertmanager. A non-empty subset narrows the route to those
// severities via an extra matcher (`severity="X"` or `severity=~"X|Y"`).
// Language/Format pick the email template variant: at least one recipient
// in the merged config can pin one language, another a different one
// (we render one email_configs per recipient with template overrides).
type EmailRecipient struct {
	Address    string   `json:"address" binding:"required,email,max=320"`
	Severities []string `json:"severities" binding:"max=3,dive,oneof=critical warning info"`
	Language   string   `json:"language,omitempty" binding:"omitempty,oneof=en it"`
	Format     string   `json:"format,omitempty" binding:"omitempty,oneof=html plain"`
}

// WebhookRecipient is a generic outbound HTTP receiver with optional
// severity narrowing. Name is purely descriptive (rendered into the
// receiver name in Alertmanager YAML). URL is validated against a
// denylist of private/loopback/metadata destinations at the handler.
type WebhookRecipient struct {
	Name       string   `json:"name" binding:"required,max=100"`
	URL        string   `json:"url" binding:"required,url,max=2048"`
	Severities []string `json:"severities" binding:"max=3,dive,oneof=critical warning info"`
}

// TelegramRecipient is a Telegram bot destination with optional severity
// narrowing. BotToken is bearer-equivalent and never returned outside the
// owning org (irrelevant in the current API since /alerts/config returns
// only the caller's own layer, but the storage layer still treats it as
// sensitive: encrypted-at-rest by the database, redacted from any future
// admin-only inspection path).
type TelegramRecipient struct {
	BotToken   string   `json:"bot_token" binding:"required,max=256"`
	ChatID     int64    `json:"chat_id" binding:"required"`
	Severities []string `json:"severities" binding:"max=3,dive,oneof=critical warning info"`
}

// AlertingConfigLayer is the per-organization layer persisted in
// alert_config_layers.config_json. It doubles as the internal type
// produced by MergeForRender when the renderer builds the effective
// per-tenant Mimir YAML — the merged result is conceptually "a layer
// where each entry knows its own scope (severities[]) and per-recipient
// rendering hints (language/format)".
//
// The API surface (POST/GET /alerts/config) is exactly this struct.
// Nothing about the layered model — neither inherited ancestor recipients
// nor the merged effective preview — ever leaves the owning org. Server
// performs the merge at render time only.
type AlertingConfigLayer struct {
	Enabled            ChannelToggles      `json:"enabled"`
	EmailRecipients    []EmailRecipient    `json:"email_recipients" binding:"max=50"`
	WebhookRecipients  []WebhookRecipient  `json:"webhook_recipients" binding:"max=20"`
	TelegramRecipients []TelegramRecipient `json:"telegram_recipients" binding:"max=20"`
}

// Validate runs stateless format/structure checks that must hold for every
// write path into alert_config_layers. The handler also runs DNS-aware
// webhook URL checks; this Validate is the storage-layer backstop that
// guarantees regardless of where the layer originates (HTTP handler,
// provisioning path, admin tool, future endpoint), the persisted bytes
// satisfy the contract.
func (c *AlertingConfigLayer) Validate() error {
	for i, r := range c.EmailRecipients {
		if err := validateEmailFormat(r.Address); err != nil {
			return fmt.Errorf("email_recipients[%d]: %w", i, err)
		}
		if err := validateSeverities(r.Severities); err != nil {
			return fmt.Errorf("email_recipients[%d]: %w", i, err)
		}
		if r.Language != "" {
			if _, ok := validLanguages[r.Language]; !ok {
				return fmt.Errorf("email_recipients[%d]: unknown language %q", i, r.Language)
			}
		}
		if r.Format != "" {
			if _, ok := validFormats[r.Format]; !ok {
				return fmt.Errorf("email_recipients[%d]: unknown format %q", i, r.Format)
			}
		}
	}
	for i, r := range c.WebhookRecipients {
		if err := validateStaticWebhookURL(r.URL); err != nil {
			return fmt.Errorf("webhook_recipients[%d] %q: %w", i, r.Name, err)
		}
		if err := validateSeverities(r.Severities); err != nil {
			return fmt.Errorf("webhook_recipients[%d]: %w", i, err)
		}
	}
	for i, r := range c.TelegramRecipients {
		if strings.TrimSpace(r.BotToken) == "" {
			return fmt.Errorf("telegram_recipients[%d]: bot_token is empty", i)
		}
		if err := validateSeverities(r.Severities); err != nil {
			return fmt.Errorf("telegram_recipients[%d]: %w", i, err)
		}
	}
	return nil
}

func validateSeverities(s []string) error {
	for _, v := range s {
		if _, ok := validSeverities[v]; !ok {
			return fmt.Errorf("invalid severity %q", v)
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
