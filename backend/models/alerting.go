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

// SeverityOverride defines mail/webhook settings for a specific severity level
type SeverityOverride struct {
	Severity         string            `json:"severity" binding:"required,oneof=critical warning info"`
	MailEnabled      *bool             `json:"mail_enabled"`
	WebhookEnabled   *bool             `json:"webhook_enabled"`
	MailAddresses    []string          `json:"mail_addresses,omitempty"`
	WebhookReceivers []WebhookReceiver `json:"webhook_receivers,omitempty"`
}

// SystemOverride defines mail/webhook settings for a specific system_key
type SystemOverride struct {
	SystemKey        string            `json:"system_key" binding:"required"`
	MailEnabled      *bool             `json:"mail_enabled"`
	WebhookEnabled   *bool             `json:"webhook_enabled"`
	MailAddresses    []string          `json:"mail_addresses,omitempty"`
	WebhookReceivers []WebhookReceiver `json:"webhook_receivers,omitempty"`
}

// AlertingConfig is the main configuration structure for alerting
type AlertingConfig struct {
	// Global settings
	MailEnabled      bool              `json:"mail_enabled"`
	WebhookEnabled   bool              `json:"webhook_enabled"`
	MailAddresses    []string          `json:"mail_addresses"`
	WebhookReceivers []WebhookReceiver `json:"webhook_receivers"`
	// Per-severity overrides
	Severities []SeverityOverride `json:"severities,omitempty"`
	// Per-system_key overrides
	Systems []SystemOverride `json:"systems,omitempty"`
	// Email template language: "en" (default) or "it"
	EmailTemplateLang string `json:"email_template_lang,omitempty"`
}

// AlertQueryParams holds optional query filters for GET /api/alerting/alerts
type AlertQueryParams struct {
	State     string `form:"state"`      // e.g. "firing", "pending"
	Severity  string `form:"severity"`   // e.g. "critical", "warning", "info"
	SystemKey string `form:"system_key"` // filter by system_key label
}
