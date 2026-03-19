//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import { getQueryStringParams, type Pagination } from '../common'

export const SUPPORT_SESSIONS_KEY = 'supportSessions'
export const SUPPORT_SESSIONS_TABLE_ID = 'supportSessionsTable'

export type SupportSessionStatus = 'pending' | 'active' | 'expired' | 'closed'

// SessionRef is a lightweight reference to an individual session within a system group
export interface SessionRef {
  id: string
  node_id: string | null
  status: SupportSessionStatus
  started_at: string
  expires_at: string
}

// SystemSessionGroup represents a system with its aggregated support session info
// (returned by the backend with server-side grouping and pagination)
export interface SystemSessionGroup {
  system_id: string
  system_name: string
  system_type: string
  system_key: string
  organization: {
    id: string
    name: string
    type: string
  }
  started_at: string
  expires_at: string
  status: SupportSessionStatus
  session_count: number
  node_count: number
  sessions: SessionRef[]
}

// SupportSession is the full session model (used for single-session detail)
export interface SupportSession {
  id: string
  system_id: string
  node_id: string | null
  session_token: string
  started_at: string
  expires_at: string
  status: SupportSessionStatus
  closed_at: string | null
  closed_by: string | null
  created_at: string
  updated_at: string
  system_name: string
  system_type: string
  system_key: string
  organization: {
    id: string
    name: string
    type: string
  }
}

export interface SupportAccessLog {
  id: string
  session_id: string
  operator_id: string
  operator_name: string
  access_type: string
  connected_at: string
  disconnected_at: string | null
  metadata: Record<string, unknown> | null
}

export interface DiagnosticCheck {
  name: string
  status: 'ok' | 'warning' | 'critical' | 'error' | 'timeout'
  value?: string
  details?: string
}

export interface DiagnosticPlugin {
  id: string
  name: string
  status: 'ok' | 'warning' | 'critical' | 'error' | 'timeout'
  summary?: string
  checks?: DiagnosticCheck[]
}

export interface DiagnosticsReport {
  collected_at: string
  duration_ms: number
  overall_status: 'ok' | 'warning' | 'critical' | 'error' | 'timeout'
  plugins: DiagnosticPlugin[]
}

export interface SessionDiagnostics {
  session_id: string
  diagnostics: DiagnosticsReport | null
  diagnostics_at: string | null
}

interface SupportSessionsResponse {
  code: number
  message: string
  data: {
    support_sessions: SystemSessionGroup[]
    pagination: Pagination
  }
}

interface SupportSessionResponse {
  code: number
  message: string
  data: SupportSession
}

interface SupportAccessLogsResponse {
  code: number
  message: string
  data: {
    access_logs: SupportAccessLog[]
    pagination: Pagination
  }
}

export const getSupportSessionsQueryStringParams = (
  pageNum: number,
  pageSize: number,
  statusFilter: SupportSessionStatus[],
  sortBy: string | null,
  sortDescending: boolean,
) => {
  const searchParams = new URLSearchParams({
    page: pageNum.toString(),
    page_size: pageSize.toString(),
    sort_by: sortBy || '',
    sort_direction: sortDescending ? 'desc' : 'asc',
  })

  statusFilter.forEach((status) => {
    searchParams.append('status', status)
  })

  return searchParams.toString()
}

export const getSupportSessions = (
  pageNum: number,
  pageSize: number,
  statusFilter: SupportSessionStatus[],
  sortBy: string,
  sortDescending: boolean,
) => {
  const loginStore = useLoginStore()
  const params = getSupportSessionsQueryStringParams(
    pageNum,
    pageSize,
    statusFilter,
    sortBy,
    sortDescending,
  )

  return axios
    .get<SupportSessionsResponse>(`${API_URL}/support-sessions?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getSystemActiveSessions = (systemId: string) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams({
    page: '1',
    page_size: '100',
    system_id: systemId,
    status: 'active',
  })
  params.append('status', 'pending')

  return axios
    .get<SupportSessionsResponse>(`${API_URL}/support-sessions?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.support_sessions)
}

export const getSupportSession = (id: string) => {
  const loginStore = useLoginStore()

  return axios
    .get<SupportSessionResponse>(`${API_URL}/support-sessions/${id}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const extendSupportSession = (id: string, hours: number) => {
  const loginStore = useLoginStore()

  return axios
    .patch(
      `${API_URL}/support-sessions/${id}/extend`,
      { hours },
      {
        headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
      },
    )
    .then((res) => res.data)
}

export const closeSupportSession = (id: string) => {
  const loginStore = useLoginStore()

  return axios
    .delete(`${API_URL}/support-sessions/${id}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data)
}

export interface SupportServiceItem {
  name: string
  label: string
  host: string
  path: string
  pathPrefix: string
  moduleId: string
  nodeId: string
}

export interface SupportServiceGroup {
  moduleId: string
  moduleLabel: string
  nodeId: string
  services: SupportServiceItem[]
}

interface SupportServicesRawResponse {
  code: number
  message: string
  data: {
    services: Record<
      string,
      {
        target: string
        host: string
        tls: boolean
        label: string
        path?: string
        path_prefix?: string
        module_id?: string
        node_id?: string
      }
    >
  }
}

interface ProxyTokenResponse {
  code: number
  message: string
  data: {
    url: string
    token: string
  }
}

export const getSupportSessionServices = (sessionId: string): Promise<SupportServiceGroup[]> => {
  const loginStore = useLoginStore()

  return axios
    .get<SupportServicesRawResponse>(`${API_URL}/support-sessions/${sessionId}/services`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => {
      const servicesMap = res.data.data?.services || {}

      // Build flat list of service items
      const items: SupportServiceItem[] = Object.entries(servicesMap).map(([name, svc]) => ({
        name,
        label: svc.label || '',
        host: svc.host || '',
        path: svc.path || '',
        pathPrefix: svc.path_prefix || '',
        moduleId: svc.module_id || '',
        nodeId: svc.node_id || '',
      }))

      // Group by nodeId + moduleId
      // Key: "nodeId:moduleId" for grouped, "nodeId:" for ungrouped
      const groupMap = new Map<string, SupportServiceGroup>()
      const ungrouped: SupportServiceItem[] = []

      for (const item of items) {
        if (!item.moduleId) {
          ungrouped.push(item)
          continue
        }
        const key = `${item.nodeId}:${item.moduleId}`
        let group = groupMap.get(key)
        if (!group) {
          group = {
            moduleId: item.moduleId,
            moduleLabel: item.label,
            nodeId: item.nodeId,
            services: [],
          }
          groupMap.set(key, group)
        }
        // Use the first non-empty label as the module label
        if (!group.moduleLabel && item.label) {
          group.moduleLabel = item.label
        }
        group.services.push(item)
      }

      // Sort groups by nodeId then moduleId, services within groups by name
      const groups = Array.from(groupMap.values()).sort((a, b) => {
        const nodeCompare = a.nodeId.localeCompare(b.nodeId, undefined, { numeric: true })
        if (nodeCompare !== 0) return nodeCompare
        return a.moduleId.localeCompare(b.moduleId)
      })
      for (const g of groups) {
        g.services.sort((a, b) => a.name.localeCompare(b.name))
      }

      // Add ungrouped services as individual groups
      ungrouped.sort((a, b) => a.name.localeCompare(b.name))
      for (const item of ungrouped) {
        groups.push({
          moduleId: '',
          moduleLabel: '',
          nodeId: item.nodeId,
          services: [item],
        })
      }

      return groups
    })
}

export const generateSupportProxyToken = (sessionId: string, service: string) => {
  const loginStore = useLoginStore()

  return axios
    .post<ProxyTokenResponse>(
      `${API_URL}/support-sessions/${sessionId}/proxy-token`,
      { service },
      {
        headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
      },
    )
    .then((res) => res.data.data)
}

interface TerminalTicketResponse {
  code: number
  message: string
  data: {
    ticket: string
  }
}

export const getTerminalTicket = (sessionId: string): Promise<string> => {
  const loginStore = useLoginStore()

  return axios
    .post<TerminalTicketResponse>(
      `${API_URL}/support-sessions/${sessionId}/terminal-ticket`,
      {},
      {
        headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
      },
    )
    .then((res) => res.data.data.ticket)
}

export const getSupportSessionLogs = (sessionId: string, pageNum: number, pageSize: number) => {
  const loginStore = useLoginStore()
  const params = getQueryStringParams(pageNum, pageSize, null, null, false)

  return axios
    .get<SupportAccessLogsResponse>(`${API_URL}/support-sessions/${sessionId}/logs?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export interface AddSessionServiceItem {
  name: string
  target: string
  label?: string
  tls?: boolean
}

export const addSupportSessionServices = (sessionId: string, services: AddSessionServiceItem[]) => {
  const loginStore = useLoginStore()
  return axios
    .post(
      `${API_URL}/support-sessions/${sessionId}/services`,
      { services },
      { headers: { Authorization: `Bearer ${loginStore.jwtToken}` } },
    )
    .then((res) => res.data)
}

export const getSupportSessionDiagnostics = (sessionId: string): Promise<SessionDiagnostics> => {
  const loginStore = useLoginStore()
  return axios
    .get<{
      code: number
      message: string
      data: SessionDiagnostics
    }>(`${API_URL}/support-sessions/${sessionId}/diagnostics`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
