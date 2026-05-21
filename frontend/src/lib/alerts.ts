//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import { type Pagination } from './common'
import type { NeBadgeV2Kind } from '@nethesis/vue-components/components/NeBadgeV2.vue.js'
import * as v from 'valibot'

export const ALERTS_CONFIG_KEY = 'alertsConfig'
export const ALERTS_ALERTS_KEY = 'alertsAlerts'
export const ALERTS_TOTALS_KEY = 'alertsTotals'
export const ALERTS_TREND_KEY = 'alertsTrend'
export const ALERTS_STATS_KEY = 'alertsStats'
export const ALERTS_HISTORY_KEY = 'alertsHistory'
export const ALERT_HISTORY_KEY = 'alertHistory'
export const ALERT_ACTIVITY_KEY = 'alertActivity'
export const ALERTS_SILENCES_KEY = 'alertsSilences'
export const ALERT_SILENCES_KEY = 'alertSilences'
export const ALERT_HISTORY_TABLE_ID = 'alertHistoryTable'
export const SYSTEM_ALERT_SILENCES_KEY = 'systemAlertSilences'

// ── Types ─────────────────────────────────────────────────────────────────────

export interface ChannelToggles {
  email?: boolean | null
  webhook?: boolean | null
  telegram?: boolean | null
}

export interface EmailRecipient {
  address: string
  severities?: string[]
  language?: 'en' | 'it'
  format?: 'html' | 'plain'
}

export interface WebhookRecipient {
  name: string
  url: string
  severities?: string[]
}

export interface TelegramRecipient {
  bot_token: string
  chat_id: number
  severities?: string[]
}

export interface AlertingConfigLayer {
  enabled: ChannelToggles
  email_recipients: EmailRecipient[]
  webhook_recipients: WebhookRecipient[]
  telegram_recipients: TelegramRecipient[]
}

// ── Valibot schemas ───────────────────────────────────────────────────────────

const SeveritySchema = v.picklist(['critical', 'warning', 'info'])

export const EmailRecipientPayloadSchema = v.object({
  address: v.pipe(
    v.string(),
    v.nonEmpty('alerts.email_address_required'),
    v.email('alerts.email_address_invalid_format'),
    v.maxLength(320, 'alerts.email_address_invalid_format'),
  ),
  severities: v.optional(v.array(SeveritySchema)),
  language: v.optional(v.picklist(['en', 'it'])),
  format: v.optional(v.picklist(['html', 'plain'])),
})

export const WebhookRecipientPayloadSchema = v.object({
  url: v.pipe(
    v.string(),
    v.nonEmpty('alerts.webhook_url_required'),
    v.url('alerts.webhook_url_invalid'),
    v.maxLength(2048, 'alerts.webhook_url_invalid'),
  ),
  name: v.optional(v.string()),
  severities: v.optional(v.array(SeveritySchema)),
})

export const TelegramRecipientPayloadSchema = v.object({
  bot_token: v.pipe(
    v.string(),
    v.nonEmpty('alerts.telegram_bot_token_required'),
    v.maxLength(256, 'alerts.telegram_bot_token_required'),
  ),
  chat_id: v.pipe(
    v.number(),
    v.check((n) => n !== 0, 'alerts.telegram_chat_id_required'),
  ),
  severities: v.optional(v.array(SeveritySchema)),
})

export const EmailNotificationsPayloadSchema = v.object({
  email_recipients: v.array(EmailRecipientPayloadSchema),
})

export const WebhookNotificationsPayloadSchema = v.object({
  webhook_recipients: v.array(WebhookRecipientPayloadSchema),
})

export const TelegramNotificationsPayloadSchema = v.object({
  telegram_recipients: v.array(TelegramRecipientPayloadSchema),
})

export type AlertState = 'active' | 'suppressed' | 'unprocessed'

export interface AlertStatus {
  state: AlertState
  silencedBy: string[]
  inhibitedBy: string[]
}

export interface ActiveAlert {
  fingerprint: string
  labels: Record<string, string>
  annotations: Record<string, string>
  status: AlertStatus
  startsAt: string
  endsAt: string
  generatorURL?: string
}

export type Alert = ActiveAlert

export interface AlertmanagerSilenceStatus {
  state: 'active' | 'expired' | 'pending'
}

export interface AlertmanagerMatcher {
  name: string
  value: string
  isRegex: boolean
}

export interface AlertmanagerSilence {
  id: string
  organization_id?: string
  system_key?: string
  matchers: AlertmanagerMatcher[]
  startsAt: string
  endsAt: string
  updatedAt: string
  createdBy: string
  comment: string
  status: AlertmanagerSilenceStatus
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

interface AlertsConfigResponse {
  code: number
  message: string
  data: AlertingConfigLayer & {
    updated_by_name?: string | null
    updated_at?: string | null
  }
}

interface AlertsResponse {
  code: number
  message: string
  data: {
    alerts: Alert[]
    pagination?: Pagination
    warnings?: string[]
  }
}

interface AlertsTotalsResponse {
  code: number
  message: string
  data: {
    active: number
    critical: number
    warning: number
    info: number
    muted: number
    history: number
    warnings?: string[]
  }
}

interface AlertNameCount {
  alertname: string
  count: number
}

interface SystemKeyCount {
  system_key: string
  count: number
}

interface AlertStats {
  total: number
  by_severity: Record<string, number>
  top_alertnames: AlertNameCount[]
  top_systems: SystemKeyCount[]
  mttr_seconds?: number
  mtbf_seconds?: number
}

interface AlertsStatsResponse {
  code: number
  message: string
  data: AlertStats
}

interface TrendDataPoint {
  date: string
  count: number
}

interface TrendResponse {
  period: number
  period_label: string
  current_total: number
  previous_total: number
  delta: number
  delta_percentage: number
  trend: 'up' | 'down' | 'stable'
  data_points: TrendDataPoint[]
}

interface AlertsTrendResponse {
  code: number
  message: string
  data: TrendResponse
}

interface AlertHistoryResponse {
  code: number
  message: string
  data: {
    alerts: AlertHistoryRecord[]
    pagination: Pagination
  }
}

interface AlertActivityEntry {
  id: number
  organization_id: string
  fingerprint: string
  action: 'silenced' | 'silence_updated' | 'unsilenced'
  actor_user_id?: string
  actor_name?: string
  silence_id?: string
  details: Record<string, unknown>
  created_at: string
}

interface AlertActivityResponse {
  code: number
  message: string
  data: {
    events: AlertActivityEntry[]
  }
}

interface CreateSystemAlertSilenceResponse {
  code: number
  message: string
  data: {
    silence_id: string
  }
}

interface SystemAlertSilencesResponse {
  code: number
  message: string
  data: {
    silences: AlertmanagerSilence[]
    warnings?: string[]
  }
}

export const getAlertsConfig = (format?: 'yaml') => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams()
  if (format) {
    params.append('format', format)
  }

  return axios
    .get<AlertsConfigResponse>(`${API_URL}/alerts/config?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getAlertsTotals = (organizationIds?: string | string[], include?: 'descendants') => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams()

  if (organizationIds) {
    const orgIds = Array.isArray(organizationIds) ? organizationIds : [organizationIds]
    orgIds.forEach((id) => params.append('organization_id', id))
  }

  if (include === 'descendants') {
    params.append('include', 'descendants')
  }

  return axios
    .get<AlertsTotalsResponse>(`${API_URL}/alerts/totals?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getAlertsTrend = (
  organizationIds?: string | string[],
  include?: 'descendants',
  period: 7 | 30 | 180 | 365 = 7,
) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams()

  if (organizationIds) {
    const orgIds = Array.isArray(organizationIds) ? organizationIds : [organizationIds]
    orgIds.forEach((id) => params.append('organization_id', id))
  }

  if (include === 'descendants') {
    params.append('include', 'descendants')
  }

  params.append('period', period.toString())

  return axios
    .get<AlertsTrendResponse>(`${API_URL}/alerts/trend?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getAlertsStats = (
  organizationIds?: string | string[],
  include?: 'descendants',
  fromDate?: string,
  toDate?: string,
  top?: number,
) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams()

  if (organizationIds) {
    const orgIds = Array.isArray(organizationIds) ? organizationIds : [organizationIds]
    orgIds.forEach((id) => params.append('organization_id', id))
  }

  if (include === 'descendants') {
    params.append('include', 'descendants')
  }
  if (fromDate) {
    params.append('from_date', fromDate)
  }
  if (toDate) {
    params.append('to_date', toDate)
  }
  if (top !== undefined) {
    params.append('top', Math.min(top, 50).toString())
  }

  return axios
    .get<AlertsStatsResponse>(`${API_URL}/alerts/stats?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getAlertsHistory = (
  organizationIds?: string | string[],
  page: number = 1,
  pageSize: number = 20,
  sortBy: string = 'created_at',
  sortDirection: 'asc' | 'desc' = 'desc',
  include?: 'descendants',
  fromDate?: string,
  toDate?: string,
  systemKeyFilters?: string | string[],
  alertnameFilters?: string | string[],
  severityFilters?: string | string[],
  statusFilters?: string | string[],
) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams()

  if (organizationIds) {
    const orgIds = Array.isArray(organizationIds) ? organizationIds : [organizationIds]
    orgIds.forEach((id) => params.append('organization_id', id))
  }

  if (include === 'descendants') {
    params.append('include', 'descendants')
  }

  params.append('page', page.toString())
  params.append('page_size', Math.min(pageSize, 200).toString())
  params.append('sort_by', sortBy)
  params.append('sort_direction', sortDirection)

  if (fromDate) {
    params.append('from_date', fromDate)
  }
  if (toDate) {
    params.append('to_date', toDate)
  }
  if (systemKeyFilters) {
    const keys = Array.isArray(systemKeyFilters) ? systemKeyFilters : [systemKeyFilters]
    keys.forEach((key) => params.append('system_key', key))
  }
  if (alertnameFilters) {
    const names = Array.isArray(alertnameFilters) ? alertnameFilters : [alertnameFilters]
    names.forEach((name) => params.append('alertname', name))
  }
  if (severityFilters) {
    const severities = Array.isArray(severityFilters) ? severityFilters : [severityFilters]
    severities.forEach((severity) => params.append('severity', severity))
  }
  if (statusFilters) {
    const statuses = Array.isArray(statusFilters) ? statusFilters : [statusFilters]
    statuses.forEach((status) => params.append('status', status))
  }

  return axios
    .get<AlertHistoryResponse>(`${API_URL}/alerts/history?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getAlertActivity = (
  fingerprint: string,
  organizationId: string,
  limit: number = 100,
) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams()

  params.append('organization_id', organizationId)
  params.append('limit', Math.min(limit, 500).toString())

  return axios
    .get<AlertActivityResponse>(
      `${API_URL}/alerts/activity/${encodeURIComponent(fingerprint)}?${params}`,
      {
        headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
      },
    )
    .then((res) => res.data.data)
}

export const getAlertsSilences = (
  organizationIds?: string | string[],
  include?: 'descendants',
  systemKeyFilters?: string | string[],
) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams()

  if (organizationIds) {
    const orgIds = Array.isArray(organizationIds) ? organizationIds : [organizationIds]
    orgIds.forEach((id) => params.append('organization_id', id))
  }

  if (include === 'descendants') {
    params.append('include', 'descendants')
  }

  if (systemKeyFilters) {
    const keys = Array.isArray(systemKeyFilters) ? systemKeyFilters : [systemKeyFilters]
    keys.forEach((key) => params.append('system_key', key))
  }

  return axios
    .get<SystemAlertSilencesResponse>(`${API_URL}/alerts/silences?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const createAlertSilence = (
  fingerprint: string,
  organizationId?: string,
  comment?: string,
  endAt?: string,
  durationMinutes?: number,
) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams()
  if (organizationId) {
    params.append('organization_id', organizationId)
  }

  const payload: Record<string, unknown> = { fingerprint }
  if (comment?.trim()) {
    payload.comment = comment.trim()
  }
  if (endAt) {
    payload.end_at = endAt
  }
  if (durationMinutes !== undefined) {
    payload.duration_minutes = durationMinutes
  }

  return axios
    .post<CreateSystemAlertSilenceResponse>(`${API_URL}/alerts/silences?${params}`, payload, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getAlertSilence = (silenceId: string, organizationId?: string) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams()
  if (organizationId) {
    params.append('organization_id', organizationId)
  }

  return axios
    .get<{
      code: number
      message: string
      data: { silence: AlertmanagerSilence }
    }>(`${API_URL}/alerts/silences/${encodeURIComponent(silenceId)}?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.silence)
}

export const updateAlertSilence = (
  silenceId: string,
  organizationId?: string,
  comment?: string,
  endAt?: string,
) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams()
  if (organizationId) {
    params.append('organization_id', organizationId)
  }

  const payload: Record<string, unknown> = {}
  if (comment !== undefined) {
    payload.comment = comment
  }
  if (endAt) {
    payload.end_at = endAt
  }

  return axios.put(
    `${API_URL}/alerts/silences/${encodeURIComponent(silenceId)}?${params}`,
    payload,
    { headers: { Authorization: `Bearer ${loginStore.jwtToken}` } },
  )
}

export const deleteAlertSilence = (silenceId: string, organizationId?: string) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams()
  if (organizationId) {
    params.append('organization_id', organizationId)
  }

  return axios.delete(`${API_URL}/alerts/silences/${encodeURIComponent(silenceId)}?${params}`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const postAlertsConfig = (config: AlertingConfigLayer) => {
  const loginStore = useLoginStore()

  return axios.post(`${API_URL}/alerts/config`, config, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const deleteAlertsConfig = () => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/alerts/config`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const getAlerts = (
  organizationIds?: string | string[],
  page: number = 1,
  pageSize: number = 50,
  sortBy: 'starts_at' | 'severity' | 'alertname' | 'status' = 'starts_at',
  sortDirection: 'asc' | 'desc' = 'desc',
  statusFilters?: string | string[],
  severityFilters?: string | string[],
  systemKeyFilters?: string | string[],
  alertnameFilters?: string | string[],
  include?: 'descendants',
) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams()

  // Add organization_id(s)
  if (organizationIds) {
    const orgIds = Array.isArray(organizationIds) ? organizationIds : [organizationIds]
    orgIds.forEach((id) => params.append('organization_id', id))
  }

  // Add include parameter
  if (include === 'descendants') {
    params.append('include', 'descendants')
  }

  // Add pagination
  params.append('page', page.toString())
  params.append('page_size', Math.min(pageSize, 100).toString())

  // Add sorting
  params.append('sort_by', sortBy)
  params.append('sort_direction', sortDirection)

  // Add status filters (renamed from state)
  if (statusFilters) {
    const statuses = Array.isArray(statusFilters) ? statusFilters : [statusFilters]
    statuses.forEach((status) => params.append('status', status))
  }

  // Add severity filters
  if (severityFilters) {
    const severities = Array.isArray(severityFilters) ? severityFilters : [severityFilters]
    severities.forEach((severity) => params.append('severity', severity))
  }

  // Add system_key filters
  if (systemKeyFilters) {
    const keys = Array.isArray(systemKeyFilters) ? systemKeyFilters : [systemKeyFilters]
    keys.forEach((key) => params.append('system_key', key))
  }

  // Add alertname filters
  if (alertnameFilters) {
    const names = Array.isArray(alertnameFilters) ? alertnameFilters : [alertnameFilters]
    names.forEach((name) => params.append('alertname', name))
  }

  return axios
    .get<AlertsResponse>(`${API_URL}/alerts?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getSystemAlertHistory = (
  systemId: string,
  page: number = 1,
  pageSize: number = 50,
  sortBy: string = 'starts_at',
  sortDescending: boolean = true,
) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams({
    page: page.toString(),
    page_size: Math.min(pageSize, 100).toString(),
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
export const getSystemActiveAlerts = (
  systemId: string,
  page: number = 1,
  pageSize: number = 50,
  sortBy: 'starts_at' | 'severity' | 'alertname' | 'status' = 'starts_at',
  sortDirection: 'asc' | 'desc' = 'desc',
  statusFilters?: string | string[],
  severityFilters?: string | string[],
  alertnameFilters?: string | string[],
) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams()

  params.append('page', page.toString())
  params.append('page_size', Math.min(pageSize, 100).toString())
  params.append('sort_by', sortBy)
  params.append('sort_direction', sortDirection)

  if (statusFilters) {
    const statuses = Array.isArray(statusFilters) ? statusFilters : [statusFilters]
    statuses.forEach((status) => params.append('status', status))
  }
  if (severityFilters) {
    const severities = Array.isArray(severityFilters) ? severityFilters : [severityFilters]
    severities.forEach((severity) => params.append('severity', severity))
  }
  if (alertnameFilters) {
    const names = Array.isArray(alertnameFilters) ? alertnameFilters : [alertnameFilters]
    names.forEach((name) => params.append('alertname', name))
  }

  return axios
    .get<AlertsResponse>(`${API_URL}/systems/${systemId}/alerts?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getSystemAlertSilences = (systemId: string) => {
  const loginStore = useLoginStore()
  return axios
    .get<SystemAlertSilencesResponse>(`${API_URL}/systems/${systemId}/alerts/silences`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.silences)
}

export const getSystemAlertSilence = (systemId: string, silenceId: string) => {
  const loginStore = useLoginStore()

  return axios
    .get<{
      code: number
      message: string
      data: { silence: AlertmanagerSilence }
    }>(`${API_URL}/systems/${systemId}/alerts/silences/${encodeURIComponent(silenceId)}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.silence)
}

export const createSystemAlertSilence = (
  systemId: string,
  fingerprint: string,
  comment?: string,
  endAt?: string,
  durationMinutes?: number,
) => {
  const loginStore = useLoginStore()
  const payload: Record<string, unknown> = { fingerprint }
  if (comment?.trim()) {
    payload.comment = comment.trim()
  }
  if (endAt) {
    payload.end_at = endAt
  }
  if (durationMinutes !== undefined) {
    payload.duration_minutes = durationMinutes
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

export const updateSystemAlertSilence = (
  systemId: string,
  silenceId: string,
  comment?: string,
  endAt?: string,
) => {
  const loginStore = useLoginStore()
  const payload: Record<string, unknown> = {}

  if (comment !== undefined) {
    payload.comment = comment
  }
  if (endAt) {
    payload.end_at = endAt
  }

  return axios.put(
    `${API_URL}/systems/${systemId}/alerts/silences/${encodeURIComponent(silenceId)}`,
    payload,
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const deleteSystemAlertSilence = (systemId: string, silenceId: string) => {
  const loginStore = useLoginStore()

  return axios.delete(
    `${API_URL}/systems/${systemId}/alerts/silences/${encodeURIComponent(silenceId)}`,
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
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

export const getAlertSilenceIds = (alert: Alert) => {
  return Array.from(new Set((alert.status?.silencedBy || []).filter((silenceId) => !!silenceId)))
}

export const isAlertSilenced = (alert: Alert) => {
  return getAlertSilenceIds(alert).length > 0
}

export const getStatusBadgeColor = (status?: string) => {
  switch (status?.toLowerCase()) {
    case 'active':
      return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
    case 'suppressed':
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
    case 'resolved':
      return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200'
  }
}

export const getSeverityBadgeColor = (severity?: string) => {
  switch (severity?.toLowerCase()) {
    case 'critical':
      return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
    case 'warning':
      return 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200'
    case 'info':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200'
  }
}

export const getSeverityBadgeKind = (severity?: string): NeBadgeV2Kind => {
  switch (severity?.toLowerCase()) {
    case 'critical':
      return 'rose'
    case 'warning':
      return 'amber'
    case 'info':
      return 'blue'
    default:
      return 'gray'
  }
}
