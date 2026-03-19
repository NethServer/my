/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package diagnostics

import "time"

// DiagnosticStatus represents the health status of a diagnostic check
type DiagnosticStatus string

const (
	StatusOK       DiagnosticStatus = "ok"
	StatusWarning  DiagnosticStatus = "warning"
	StatusCritical DiagnosticStatus = "critical"
	StatusError    DiagnosticStatus = "error"
	StatusTimeout  DiagnosticStatus = "timeout"
)

// DiagnosticCheck is a single check within a plugin result
type DiagnosticCheck struct {
	Name    string           `json:"name"`
	Status  DiagnosticStatus `json:"status"`
	Value   string           `json:"value,omitempty"`
	Details string           `json:"details,omitempty"`
}

// PluginResult is the output of a single diagnostic plugin
type PluginResult struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Status  DiagnosticStatus  `json:"status"`
	Summary string            `json:"summary,omitempty"`
	Checks  []DiagnosticCheck `json:"checks,omitempty"`
}

// DiagnosticsReport is the full aggregated report sent to the support service
type DiagnosticsReport struct {
	CollectedAt   time.Time        `json:"collected_at"`
	DurationMs    int64            `json:"duration_ms"`
	OverallStatus DiagnosticStatus `json:"overall_status"`
	Plugins       []PluginResult   `json:"plugins"`
}
