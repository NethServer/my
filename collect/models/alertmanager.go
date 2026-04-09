/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package models

import "time"

// AlertmanagerWebhookPayload is the top-level payload sent by Alertmanager webhooks.
type AlertmanagerWebhookPayload struct {
	Receiver          string              `json:"receiver"`
	Status            string              `json:"status"`
	Alerts            []AlertmanagerAlert `json:"alerts"`
	GroupLabels       map[string]string   `json:"groupLabels"`
	CommonLabels      map[string]string   `json:"commonLabels"`
	CommonAnnotations map[string]string   `json:"commonAnnotations"`
	ExternalURL       string              `json:"externalURL"`
	Version           string              `json:"version"`
	GroupKey          string              `json:"groupKey"`
	TruncatedAlerts   int                 `json:"truncatedAlerts"`
}

// AlertmanagerAlert represents a single alert inside the webhook payload.
type AlertmanagerAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}
