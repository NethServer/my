/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

import (
	"encoding/json"
	"time"
)

// DiagnosticCheck is a single named check within a plugin result.
type DiagnosticCheck struct {
	Name    string          `json:"name"`
	Status  string          `json:"status"`
	Value   string          `json:"value,omitempty"`
	Details json.RawMessage `json:"details,omitempty"`
}

// DiagnosticPlugin is the result from a single diagnostics plugin.
type DiagnosticPlugin struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Status  string            `json:"status"`
	Summary string            `json:"summary,omitempty"`
	Checks  []DiagnosticCheck `json:"checks,omitempty"`
}

// DiagnosticsReport is the full diagnostics report collected by a tunnel-client at connect time.
type DiagnosticsReport struct {
	CollectedAt   time.Time          `json:"collected_at"`
	DurationMs    int64              `json:"duration_ms"`
	OverallStatus string             `json:"overall_status"`
	Plugins       []DiagnosticPlugin `json:"plugins"`
}
