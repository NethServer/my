/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

// WebhookConfig represents a named webhook receiver
type WebhookConfig struct {
	Name string `json:"name" binding:"required"`
	URL  string `json:"url" binding:"required,url"`
}

// SeverityConfig defines email and webhook receivers for a specific severity level,
// plus optional system_key exceptions that should be excluded from notifications
type SeverityConfig struct {
	Emails     []string        `json:"emails" binding:"required,min=1,dive,email"`
	Webhooks   []WebhookConfig `json:"webhooks,omitempty"`
	Exceptions []string        `json:"exceptions,omitempty"`
}

// AlertingConfigRequest is the JSON body for POST /api/alerting/config.
// Keys are severity levels: "critical", "warning", "info".
type AlertingConfigRequest map[string]SeverityConfig

// AlertQueryParams holds optional query filters for GET /api/alerting/alerts
type AlertQueryParams struct {
	State     string `form:"state"`      // e.g. "firing", "pending"
	Severity  string `form:"severity"`   // e.g. "critical", "warning", "info"
	SystemKey string `form:"system_key"` // filter by system_key label
}
