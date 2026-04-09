/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import "time"

// AlertHistoryRecord represents a single resolved or inactive alert stored in alert_history.
type AlertHistoryRecord struct {
	ID          int64             `json:"id"`
	SystemKey   string            `json:"system_key"`
	Alertname   string            `json:"alertname"`
	Severity    *string           `json:"severity"`
	Status      string            `json:"status"`
	Fingerprint string            `json:"fingerprint"`
	StartsAt    time.Time         `json:"starts_at"`
	EndsAt      *time.Time        `json:"ends_at"`
	Summary     *string           `json:"summary"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Receiver    *string           `json:"receiver"`
	CreatedAt   time.Time         `json:"created_at"`
}
