// Copyright (C) 2026 Nethesis S.r.l.
// SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Pending-state layer for the alert mute/unmute "processing" badge.
 *
 * The alert's silenced status (status.silencedBy) takes 1–5 minutes to
 * propagate from Alertmanager. After a mute or unmute action, this module
 * remembers the target muted state in localStorage so the UI can show a
 * "processing" indicator until the backend reflects it.
 *
 * An entry is dropped when the backend catches up (syncWithBackend) or after a
 * 10-minute TTL.
 *
 * localStorage key: "my.alert_pending_states"
 * Entry map key:    `${fingerprint}:${organizationId}`
 */

import { reactive } from 'vue'
import type { Alert } from './alerts'

export const ALERT_PENDING_STATES_STORAGE_KEY = 'my.alert_pending_states'
const TTL_MS = 10 * 60 * 1000 // 10 minutes

export interface AlertPendingEntry {
  /** The muted state the user expects the alert to reach. */
  muted: boolean
  /** Timestamp when this entry was created (ms). */
  createdAt: number
}

// Keyed by `${fingerprint}:${organizationId}`
type PendingStatesMap = Record<string, AlertPendingEntry>

function makeKey(fingerprint: string, organizationId: string): string {
  return `${fingerprint}:${organizationId}`
}

function loadFromStorage(): PendingStatesMap {
  try {
    const raw = localStorage.getItem(ALERT_PENDING_STATES_STORAGE_KEY)
    return raw ? (JSON.parse(raw) as PendingStatesMap) : {}
  } catch {
    return {}
  }
}

function saveToStorage(): void {
  try {
    localStorage.setItem(ALERT_PENDING_STATES_STORAGE_KEY, JSON.stringify(pendingAlertStates))
  } catch {
    // quota exceeded or private browsing — fail silently
  }
}

function isExpired(entry: AlertPendingEntry): boolean {
  return Date.now() - entry.createdAt > TTL_MS
}

function isBackendMuted(alert: Alert): boolean {
  return (alert.status?.silencedBy ?? []).filter(Boolean).length > 0
}

/** Reactive map — components re-render automatically when it changes. */
export const pendingAlertStates = reactive<PendingStatesMap>(loadFromStorage())

/** Record the target muted state after a mute or unmute action. */
export function setPendingAlertState(
  fingerprint: string,
  organizationId: string,
  muted: boolean,
): void {
  pendingAlertStates[makeKey(fingerprint, organizationId)] = {
    muted,
    createdAt: Date.now(),
  }
  saveToStorage()
}

/** Return the (non-expired) pending entry for an alert, or null. */
export function getPendingAlertEntry(
  fingerprint: string,
  organizationId: string,
): AlertPendingEntry | null {
  const entry = pendingAlertStates[makeKey(fingerprint, organizationId)]
  return entry && !isExpired(entry) ? entry : null
}

/**
 * Whether an alert is in transition: a mute/unmute action was performed but the
 * backend hasn't reflected the target state yet. Use this to show a
 * "processing" badge and disable mute/unmute actions.
 */
export function isProcessing(alert: Alert): boolean {
  const entry = getPendingAlertEntry(alert.fingerprint, alert.labels?.organization_id ?? '')
  return entry ? entry.muted !== isBackendMuted(alert) : false
}

/**
 * Call after each alerts fetch. Drops pending entries that the backend now
 * agrees with, plus any expired entries.
 */
export function syncWithBackend(alerts: Alert[]): void {
  let changed = false

  for (const key of Object.keys(pendingAlertStates)) {
    if (isExpired(pendingAlertStates[key])) {
      delete pendingAlertStates[key]
      changed = true
    }
  }

  for (const alert of alerts) {
    const key = makeKey(alert.fingerprint, alert.labels?.organization_id ?? '')
    const entry = pendingAlertStates[key]
    if (entry && entry.muted === isBackendMuted(alert)) {
      delete pendingAlertStates[key]
      changed = true
    }
  }

  if (changed) saveToStorage()
}
