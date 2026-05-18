/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import "sort"

// AlertCatalogEntry describes a single known alert type produced by NethServer
// (NS8) and NethSecurity systems.
type AlertCatalogEntry struct {
	Name     string `json:"name"`
	Severity string `json:"severity"`          // critical | warning | info
	Service  string `json:"service,omitempty"` // sub-service, when applicable
}

// alertCatalog is the static, authoritative list of alert names the platform
// can receive. It is intentionally NOT derived from alert_history: the filters
// dropdown must offer every alert a system can raise, not only the ones already
// seen. Severities/organizations/systems remain data-driven; alert names do not.
//
// Source of truth — kept in sync with the vmalert rules shipped by:
//   - NethServer / NS8 metrics: NethServer/ns8-metrics#71
//   - NethSecurity:             NethServer/nethsecurity#1633
//
// Plus alerts synthesised by the collect service itself (not raised by a
// system): LinkFailed, emitted by the heartbeat monitor cron when a system
// stops communicating (see collect/cron/linkfailed_monitor.go).
//
// See also the unified Alert Catalog in services/mimir/README.md. The
// placeholder `MyAlert` template rule from nethsecurity is intentionally
// excluded.
var alertCatalog = []AlertCatalogEntry{
	// --- Synthesised by collect (heartbeat monitor) ---
	{Name: "LinkFailed", Severity: "critical"},

	// --- NethServer / NS8 (ns8-metrics#71) ---
	{Name: "BackupFailed", Severity: "critical", Service: "backup"},
	{Name: "CertExpired", Severity: "critical"},
	{Name: "CertExpiringCritical", Severity: "critical"},
	{Name: "CertExpiringSoon", Severity: "warning"},
	{Name: "DiskSpaceCritical", Severity: "critical", Service: "storage"},
	{Name: "DiskSpaceLow", Severity: "warning", Service: "storage"},
	{Name: "LokiOffline", Severity: "warning", Service: "loki"},
	{Name: "NodeOffline", Severity: "critical"},
	{Name: "RaidDiskFailed", Severity: "critical", Service: "storage"},
	{Name: "RaidDriveMissing", Severity: "critical", Service: "storage"},
	{Name: "SwapFull", Severity: "warning"},
	{Name: "SwapNotPresent", Severity: "critical"},

	// --- NethSecurity (nethsecurity#1633) ---
	{Name: "BackupEncryptionDisabled", Severity: "warning", Service: "backup"},
	{Name: "CriticalCpuUsage", Severity: "warning", Service: "host"},
	{Name: "CriticalMemoryUsage", Severity: "warning", Service: "host"},
	{Name: "DiskSpaceWarning", Severity: "warning", Service: "storage"},
	{Name: "HighCpuUsage", Severity: "info", Service: "host"},
	{Name: "HighMemoryUsage", Severity: "info", Service: "host"},
	{Name: "HighSystemLoad", Severity: "warning", Service: "host"},
	{Name: "ServiceDown", Severity: "critical"},
	{Name: "StorageStatus", Severity: "critical", Service: "storage"},
	{Name: "WanDown", Severity: "critical", Service: "network"},
}

// AlertCatalog returns the static alert catalog sorted by alert name. The
// returned slice is a fresh copy so callers cannot mutate the package state.
func AlertCatalog() []AlertCatalogEntry {
	out := make([]AlertCatalogEntry, len(alertCatalog))
	copy(out, alertCatalog)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
