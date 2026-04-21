/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

// WebhookReceiver represents a generic webhook receiver with name and URL
type WebhookReceiver struct {
	Name string `json:"name" binding:"required"`
	URL  string `json:"url" binding:"required,url"`
}

// TelegramReceiver represents a Telegram bot notification target.
// BotToken is the secret token obtained from @BotFather.
// ChatID is the numeric identifier of the target chat (user, group, or channel).
type TelegramReceiver struct {
	BotToken string `json:"bot_token" binding:"required"`
	ChatID   int64  `json:"chat_id" binding:"required"`
}

// SeverityOverride defines mail/webhook/telegram settings for a specific severity level
type SeverityOverride struct {
	Severity          string             `json:"severity" binding:"required,oneof=critical warning info"`
	MailEnabled       *bool              `json:"mail_enabled"`
	WebhookEnabled    *bool              `json:"webhook_enabled"`
	TelegramEnabled   *bool              `json:"telegram_enabled"`
	MailAddresses     []string           `json:"mail_addresses,omitempty"`
	WebhookReceivers  []WebhookReceiver  `json:"webhook_receivers,omitempty"`
	TelegramReceivers []TelegramReceiver `json:"telegram_receivers,omitempty"`
}

// SystemOverride defines mail/webhook/telegram settings for a specific system_key
type SystemOverride struct {
	SystemKey         string             `json:"system_key" binding:"required"`
	MailEnabled       *bool              `json:"mail_enabled"`
	WebhookEnabled    *bool              `json:"webhook_enabled"`
	TelegramEnabled   *bool              `json:"telegram_enabled"`
	MailAddresses     []string           `json:"mail_addresses,omitempty"`
	WebhookReceivers  []WebhookReceiver  `json:"webhook_receivers,omitempty"`
	TelegramReceivers []TelegramReceiver `json:"telegram_receivers,omitempty"`
}

// AlertingConfig is the main configuration structure for alerting
type AlertingConfig struct {
	// Global settings
	MailEnabled       bool               `json:"mail_enabled"`
	WebhookEnabled    bool               `json:"webhook_enabled"`
	TelegramEnabled   bool               `json:"telegram_enabled"`
	MailAddresses     []string           `json:"mail_addresses"`
	WebhookReceivers  []WebhookReceiver  `json:"webhook_receivers"`
	TelegramReceivers []TelegramReceiver `json:"telegram_receivers"`
	// Per-severity overrides
	Severities []SeverityOverride `json:"severities,omitempty"`
	// Per-system_key overrides
	Systems []SystemOverride `json:"systems,omitempty"`
	// Email template language: "en" (default) or "it"
	EmailTemplateLang string `json:"email_template_lang,omitempty"`
}

// AlertQueryParams holds optional query filters for GET /api/alerts
type AlertQueryParams struct {
	State     string `form:"state"`      // e.g. "firing", "pending"
	Severity  string `form:"severity"`   // e.g. "critical", "warning", "info"
	SystemKey string `form:"system_key"` // filter by system_key label
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
