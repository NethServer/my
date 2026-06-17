//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import { type Pagination } from './common'
import type { NeBadgeV2Kind, FilterOption } from '@nethesis/vue-components'
import * as v from 'valibot'

export const ALERTS_CONFIG_KEY = 'alertsConfig'
export const ALERTS_ALERTS_KEY = 'alertsAlerts'
export const ALERTS_TOTALS_KEY = 'alertsTotals'
export const ALERT_ACTIVITY_KEY = 'alertActivity'
export const ALERTS_SILENCES_KEY = 'alertsSilences'
export const ALERTS_TABLE_ID = 'alertsTable'
export const ALERTS_REFETCH_INTERVAL_SECONDS = 10
export const SYSTEM_ALERT_HISTORY_TABLE_ID = 'systemAlertHistoryTable'
export const SYSTEM_ALERTS_TABLE_ID = 'systemAlertsTable'
export const SYSTEM_ALERT_SILENCES_KEY = 'systemAlertSilences'
export const SYSTEM_ALERTS_KEY = 'systemAlerts'
export const SYSTEM_ALERT_HISTORY_KEY = 'systemAlertHistory'
export const SEVERITY_FILTER_OPTIONS: FilterOption[] = [
  { id: 'critical', label: 'Critical' },
  { id: 'warning', label: 'Warning' },
  { id: 'info', label: 'Info' },
]
export const MIN_ESTIMATED_COUNT = 50

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

export interface AlertsResponse {
  code: number
  message: string
  data: {
    alerts: Alert[]
    pagination?: Pagination
    warnings?: string[]
  }
}

export type AlertStatusEnum = 'active' | 'suppressed'

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
    v.number('alerts.telegram_chat_id_required'),
    v.check((n) => n !== 0, 'alerts.telegram_chat_id_invalid'),
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

export interface AlertAnnotations {
  summary_en?: string
  summary_it?: string
  description_en?: string
  description_it?: string
  summary?: string
  description?: string
}

export type AlertAnnotationsWithExtensions = AlertAnnotations & Record<string, string | undefined>

export interface AlertStatus {
  state: AlertState
  silencedBy: string[]
  inhibitedBy: string[]
}

export interface ActiveAlert {
  fingerprint: string
  labels: Record<string, string>
  annotations: AlertAnnotationsWithExtensions
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
  annotations: AlertAnnotationsWithExtensions
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

export const getAlertsTotals = (
  organizationIds?: string | string[],
  include: 'descendants' = 'descendants',
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

  return axios
    .get<AlertsTotalsResponse>(`${API_URL}/alerts/totals?${params}`, {
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
  include: 'descendants' = 'descendants',
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
  severityFilters?: string | string[],
  alertnameFilters?: string | string[],
  statusFilters?: string | string[],
) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams({
    page: page.toString(),
    page_size: Math.min(pageSize, 100).toString(),
    sort_by: sortBy,
    sort_direction: sortDescending ? 'desc' : 'asc',
  })

  if (severityFilters) {
    const severities = Array.isArray(severityFilters) ? severityFilters : [severityFilters]
    severities.forEach((s) => params.append('severity', s))
  }
  if (alertnameFilters) {
    const names = Array.isArray(alertnameFilters) ? alertnameFilters : [alertnameFilters]
    names.forEach((n) => params.append('alertname', n))
  }
  if (statusFilters) {
    const statuses = Array.isArray(statusFilters) ? statusFilters : [statusFilters]
    statuses.forEach((s) => params.append('status', s))
  }

  return axios
    .get<AlertHistoryResponse>(`${API_URL}/systems/${systemId}/alerts/history?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
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

type AlertAnnotationKey = 'summary' | 'description'
type AlertWithAnnotations = {
  annotations?: AlertAnnotations | null | undefined
}

const DEFAULT_ALERT_LOCALE = 'en'

function getAlertAnnotation(
  alert: AlertWithAnnotations,
  annotationKey: AlertAnnotationKey,
  locale: string,
) {
  const annotations = (alert.annotations ?? {}) as Record<string, string | undefined>
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

export const deleteSystemAlertSilence = (systemId: string, silenceId: string) => {
  const loginStore = useLoginStore()

  return axios.delete(
    `${API_URL}/systems/${systemId}/alerts/silences/${encodeURIComponent(silenceId)}`,
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const isAlertSilenced = (alert: Alert) => {
  return getAlertSilenceIds(alert).length > 0
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
