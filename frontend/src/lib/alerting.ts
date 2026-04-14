//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import { type Pagination } from './common'

export const ALERTING_CONFIG_KEY = 'alertingConfig'
export const ALERTING_ALERTS_KEY = 'alertingAlerts'
export const ALERT_HISTORY_KEY = 'alertHistory'
export const ALERT_HISTORY_TABLE_ID = 'alertHistoryTable'

// ── Types ─────────────────────────────────────────────────────────────────────

export interface WebhookReceiver {
  name: string
  url: string
}

export interface SeverityOverride {
  severity: 'critical' | 'warning' | 'info'
  mail_enabled?: boolean
  webhook_enabled?: boolean
  mail_addresses?: string[]
  webhook_receivers?: WebhookReceiver[]
}

export interface SystemOverride {
  system_key: string
  mail_enabled?: boolean
  webhook_enabled?: boolean
  mail_addresses?: string[]
  webhook_receivers?: WebhookReceiver[]
}

export interface AlertingConfig {
  mail_enabled: boolean
  webhook_enabled: boolean
  mail_addresses: string[]
  webhook_receivers: WebhookReceiver[]
  severities?: SeverityOverride[]
  systems?: SystemOverride[]
  email_template_lang?: string
}

export interface AlertStatus {
  state: string
  silencedBy: string[]
  inhibitedBy: string[]
}

export interface Alert {
  labels: Record<string, string>
  annotations: Record<string, string>
  status: AlertStatus
  startsAt: string
  endsAt: string
  fingerprint: string
  generatorURL?: string
  receivers?: { name: string }[]
}

export interface AlertHistoryRecord {
  id: number
  system_key: string
  alertname: string
  severity: string | null
  status: string
  fingerprint: string
  starts_at: string
  ends_at: string | null
  summary: string | null
  labels: Record<string, string>
  annotations: Record<string, string>
  receiver: string | null
  created_at: string
}

// ── API functions ─────────────────────────────────────────────────────────────

interface AlertingConfigResponse {
  code: number
  message: string
  data: {
    config: AlertingConfig | string
  }
}

interface AlertsResponse {
  code: number
  message: string
  data: {
    alerts: Alert[]
  }
}

interface AlertHistoryResponse {
  code: number
  message: string
  data: {
    alerts: AlertHistoryRecord[]
    pagination: Pagination
  }
}

interface CreateSystemAlertSilenceResponse {
  code: number
  message: string
  data: {
    silence_id: string
  }
}

export const getAlertingConfig = (organizationId: string, format?: 'yaml') => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams({ organization_id: organizationId })
  if (format) {
    params.append('format', format)
  }

  return axios
    .get<AlertingConfigResponse>(`${API_URL}/alerts/config?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.config)
}

export const postAlertingConfig = (organizationId: string, config: AlertingConfig) => {
  const loginStore = useLoginStore()

  return axios.post(`${API_URL}/alerts/config?organization_id=${organizationId}`, config, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const deleteAlertingConfig = (organizationId: string) => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/alerts/config?organization_id=${organizationId}`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const getAlerts = (
  organizationId: string,
  state?: string,
  severity?: string,
  systemKey?: string,
) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams({ organization_id: organizationId })
  if (state) params.append('state', state)
  if (severity) params.append('severity', severity)
  if (systemKey) params.append('system_key', systemKey)

  return axios
    .get<AlertsResponse>(`${API_URL}/alerts?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.alerts)
}

export const getSystemAlertHistory = (
  systemId: string,
  page: number,
  pageSize: number,
  sortBy: string,
  sortDescending: boolean,
) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams({
    page: page.toString(),
    page_size: pageSize.toString(),
    sort_by: sortBy,
    sort_direction: sortDescending ? 'desc' : 'asc',
  })

  return axios
    .get<AlertHistoryResponse>(`${API_URL}/systems/${systemId}/alerts/history?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

// Get active alerts for a specific system via the dedicated endpoint
export const getSystemActiveAlerts = (systemId: string) => {
  const loginStore = useLoginStore()
  return axios
    .get<AlertsResponse>(`${API_URL}/systems/${systemId}/alerts`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.alerts)
}

export const createSystemAlertSilence = (
  systemId: string,
  fingerprint: string,
  comment?: string,
) => {
  const loginStore = useLoginStore()
  const payload = {
    fingerprint,
    comment: comment?.trim() || undefined,
  }

  return axios
    .post<CreateSystemAlertSilenceResponse>(
      `${API_URL}/systems/${systemId}/alerts/silences`,
      payload,
      {
        headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
      },
    )
    .then((res) => res.data.data)
}

type AlertAnnotationKey = 'summary' | 'description'
type AlertWithAnnotations = {
  annotations?: Record<string, string | null | undefined>
}

const DEFAULT_ALERT_LOCALE = 'en'

function getAlertAnnotation(
  alert: AlertWithAnnotations,
  annotationKey: AlertAnnotationKey,
  locale: string,
) {
  const annotations = alert.annotations ?? {}
  const normalizedLocale = locale.split('-')[0].toLowerCase() || DEFAULT_ALERT_LOCALE
  const candidateKeys = Array.from(
    new Set([
      `${annotationKey}_${normalizedLocale}`,
      annotationKey,
      `${annotationKey}_${DEFAULT_ALERT_LOCALE}`,
    ]),
  )

  for (const key of candidateKeys) {
    const value = annotations[key]
    if (typeof value === 'string' && value.trim()) {
      return value.trim()
    }
  }

  return ''
}

export const getAlertSummary = (alert: AlertWithAnnotations, locale: string) => {
  return getAlertAnnotation(alert, 'summary', locale)
}

export const getAlertDescription = (alert: AlertWithAnnotations, locale: string) => {
  return getAlertAnnotation(alert, 'description', locale)
}
