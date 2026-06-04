//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

// Re-exports for the system-scoped alerts domain.
// The underlying API functions and types live in @/lib/alerts; this module
// provides the canonical query-key constants and a single import point for
// components in the system-detail context.

////

export {
  // API functions — all 7 per-system endpoints
  getSystemActiveAlerts,
  getSystemAlertHistory,
  getSystemAlertSilences,
  getSystemAlertSilence,
  createSystemAlertSilence,
  updateSystemAlertSilence,
  deleteSystemAlertSilence,
  // Shared helpers
  getSeverityBadgeKind,
  getAlertSummary,
  getAlertDescription,
  getAlertSilenceIds,
  isAlertSilenced,
  // Types
  type Alert,
  type AlertHistoryRecord,
  type AlertmanagerSilence,
  // Table IDs
  SYSTEM_ALERT_HISTORY_TABLE_ID,
  SYSTEM_ALERTS_TABLE_ID,
} from '@/lib/alerts'

export {
  SYSTEM_ALERTS_KEY,
  SYSTEM_ALERT_HISTORY_KEY,
  SYSTEM_ALERT_SILENCES_KEY,
} from '@/lib/alerts'
