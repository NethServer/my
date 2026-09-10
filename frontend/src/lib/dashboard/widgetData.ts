//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '@/lib/config'
import { useLoginStore } from '@/stores/login'

export const ALERTS_STATS_KEY = 'alertsStats'
export const REBRANDING_SUMMARY_KEY = 'rebrandingSummary'
export const EXPIRING_ADDONS_KEY = 'expiringAddons'
export const TOP_APPLICATIONS_KEY = 'topApplications'
export const RECENT_SYSTEMS_KEY = 'recentSystems'

const authHeaders = () => {
  const loginStore = useLoginStore()
  return { Authorization: `Bearer ${loginStore.jwtToken}` }
}

interface Envelope<T> {
  code: number
  message: string
  data: T
}

// ---------------------------------------------------------------------------
// Alerts statistics — /alerts/stats
// ---------------------------------------------------------------------------

export interface AlertsStatsEntry {
  alertname?: string
  system_key?: string
  count: number
}

export interface AlertsStats {
  total: number
  by_severity: Record<string, number>
  top_alertnames: AlertsStatsEntry[]
  top_systems: AlertsStatsEntry[]
  mttr_seconds?: number
  mtbf_seconds?: number
}

export const getAlertsStats = (top: number = 5) => {
  const loginStore = useLoginStore()

  return axios
    .get<Envelope<AlertsStats>>(`${API_URL}/alerts/stats?top=${top}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

// ---------------------------------------------------------------------------
// Rebranding summary — /rebranding/summary
// ---------------------------------------------------------------------------

export interface RebrandingSummary {
  total: number
  distributors: number
  resellers: number
  customers: number
}

export const getRebrandingSummary = () =>
  axios
    .get<Envelope<RebrandingSummary>>(`${API_URL}/rebranding/summary`, { headers: authHeaders() })
    .then((res) => res.data.data)

// ---------------------------------------------------------------------------
// Add-on grants expiring soon — /entitlements/grants
// ---------------------------------------------------------------------------

export interface ExpiringGrant {
  id: string
  entitlement: string
  display_name?: string
  system_id?: string
  system_name?: string
  system_key?: string
  organization_name?: string
  expires_at?: string
}

interface ExpiringGrantsData {
  grants: ExpiringGrant[]
  total: number
}

export const EXPIRING_ADDONS_WINDOW_DAYS = 30

export const getExpiringGrants = (
  withinDays: number = EXPIRING_ADDONS_WINDOW_DAYS,
  pageSize: number = 5,
) => {
  const params = new URLSearchParams({
    active: 'true',
    expiring_before: new Date(Date.now() + withinDays * 24 * 60 * 60 * 1000).toISOString(),
    page_size: String(pageSize),
  })

  return axios
    .get<Envelope<ExpiringGrantsData>>(`${API_URL}/entitlements/grants?${params}`, {
      headers: authHeaders(),
    })
    .then((res) => res.data.data)
}

// ---------------------------------------------------------------------------
// Most installed applications — /applications/summary
// ---------------------------------------------------------------------------

export interface ApplicationSummaryEntry {
  instance_of: string
  name: string
  count: number
}

interface ApplicationsSummaryData {
  total: number
  by_type: ApplicationSummaryEntry[]
}

export const getTopApplications = (pageSize: number = 5) => {
  const params = new URLSearchParams({
    page: '1',
    page_size: String(pageSize),
    sort_by: 'count',
    sort_direction: 'desc',
  })

  return axios
    .get<Envelope<ApplicationsSummaryData>>(`${API_URL}/applications/summary?${params}`, {
      headers: authHeaders(),
    })
    .then((res) => res.data.data)
}
