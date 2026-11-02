//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import type { ParentOrganization } from '@/lib/organizations/organizations'
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
    logto_id: string
    name: string
    type: string
    // the company that manages the organization's systems ("Managed by")
    parent?: ParentOrganization
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

export type DiagnosticStatus = 'ok' | 'warning' | 'critical' | 'error' | 'timeout'

export interface DiagnosticCheck {
  name: string
  status: DiagnosticStatus
  value?: string
  // free-form: an object of measures, a list of rows (services, units) or a
  // plain message, as each appliance plugin reports it
  details?: unknown
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

export interface NodeDiagnostics {
  node_id: string | null
  session_id: string
  diagnostics: DiagnosticsReport | null
  diagnostics_at: string | null
}

export interface SystemDiagnostics {
  system_id: string
  overall_status: string
  nodes: NodeDiagnostics[]
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

export type SupportSessionSortBy =
  'name' | 'type' | 'organization_name' | 'started_at' | 'expires_at' | 'status'

export interface SupportSessionsFilters {
  search: string
  types: string[]
  organizationIds: string[]
  statuses: SupportSessionStatus[]
}

export const getSupportSessionsQueryStringParams = (
  pageNum: number,
  pageSize: number,
  filters: SupportSessionsFilters,
  sortBy: string | null,
  sortDescending: boolean,
) => {
  const searchParams = new URLSearchParams({
    page: pageNum.toString(),
    page_size: pageSize.toString(),
    sort_by: sortBy || '',
    sort_direction: sortDescending ? 'desc' : 'asc',
  })

  if (filters.search) {
    searchParams.append('search', filters.search)
  }
  filters.types.forEach((type) => searchParams.append('type', type))
  filters.organizationIds.forEach((id) => searchParams.append('organization_id', id))
  filters.statuses.forEach((status) => searchParams.append('status', status))

  return searchParams.toString()
}

export const getSupportSessions = (
  pageNum: number,
  pageSize: number,
  filters: SupportSessionsFilters,
  sortBy: string,
  sortDescending: boolean,
) => {
  const loginStore = useLoginStore()
  const params = getSupportSessionsQueryStringParams(
    pageNum,
    pageSize,
    filters,
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
    .get<SupportSessionResponse>(`${API_URL}/support-sessions/${encodeURIComponent(id)}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const extendSupportSession = (id: string, hours: number) => {
  const loginStore = useLoginStore()

  return axios
    .patch(
      `${API_URL}/support-sessions/${encodeURIComponent(id)}/extend`,
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
    .delete(`${API_URL}/support-sessions/${encodeURIComponent(id)}`, {
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
  // the node's label set in the cluster UI
  nodeLabel: string
  // 'cluster' when the NS8 cluster login guards the service instead of the app
  auth: string
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
        node_label?: string
        auth?: string
      }
    >
  }
}

interface ProxyTokenResponse {
  code: number
  message: string
  data: {
    url: string
    // a one-time code the proxy swaps for its access cookie
    token: string
    expires_at?: string
  }
}

export const getSupportSessionServices = (sessionId: string): Promise<SupportServiceGroup[]> => {
  const loginStore = useLoginStore()

  return axios
    .get<SupportServicesRawResponse>(
      `${API_URL}/support-sessions/${encodeURIComponent(sessionId)}/services`,
      {
        headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
      },
    )
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
        nodeLabel: svc.node_label || '',
        auth: svc.auth || '',
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

/**
 * Builds the URL that opens a proxied service. `path` comes from the appliance,
 * so the result must stay on the proxy host returned by the backend: anything
 * else would hand the token to another site. Returns null when it does not.
 */
const SUPPORT_PROXY_HOST = /^[a-z0-9-]+--[0-9a-f]{32}\.support\./

export const buildSupportProxyUrl = (
  proxyUrl: string,
  token: string,
  path?: string,
  // a page address (a link followed inside the session), not a route prefix
  exact = false,
): string | null => {
  let base: URL
  let target: URL
  try {
    base = new URL(proxyUrl)
    target = path && path !== '/' ? new URL('/' + path.replace(/^[/\\]+/, ''), base) : new URL(base)
  } catch {
    return null
  }
  if (base.protocol !== 'https:' && base.protocol !== 'http:') {
    return null
  }
  // only a support proxy host may be framed with scripts and same-origin
  // rights: never my itself
  if (base.origin === window.location.origin || !SUPPORT_PROXY_HOST.test(base.hostname)) {
    return null
  }
  if (
    target.protocol !== base.protocol ||
    target.host !== base.host ||
    target.username ||
    target.password
  ) {
    return null
  }
  if (!exact && !target.pathname.endsWith('/')) {
    target.pathname += '/'
  }
  target.searchParams.set('token', token)
  return target.toString()
}

export const generateSupportProxyToken = (sessionId: string, service: string) => {
  const loginStore = useLoginStore()

  return axios
    .post<ProxyTokenResponse>(
      `${API_URL}/support-sessions/${encodeURIComponent(sessionId)}/proxy-token`,
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
      `${API_URL}/support-sessions/${encodeURIComponent(sessionId)}/terminal-ticket`,
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
    .get<SupportAccessLogsResponse>(
      `${API_URL}/support-sessions/${encodeURIComponent(sessionId)}/logs?${params}`,
      {
        headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
      },
    )
    .then((res) => res.data.data)
}

export interface AddSessionServiceItem {
  name: string
  target: string
  label?: string
  tls?: boolean
}

export const removeSupportSessionService = (sessionId: string, serviceName: string) => {
  const loginStore = useLoginStore()
  return axios
    .delete(
      `${API_URL}/support-sessions/${encodeURIComponent(sessionId)}/services/${encodeURIComponent(serviceName)}`,
      {
        headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
      },
    )
    .then((res) => res.data)
}

export const addSupportSessionServices = (sessionId: string, services: AddSessionServiceItem[]) => {
  const loginStore = useLoginStore()
  return axios
    .post(
      `${API_URL}/support-sessions/${encodeURIComponent(sessionId)}/services`,
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
    }>(`${API_URL}/support-sessions/${encodeURIComponent(sessionId)}/diagnostics`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

// Ephemeral support users

export interface SessionUserCredential {
  username: string
  password: string
}

export interface SessionDomainUser {
  domain: string
  module: string
  username: string
  password: string
}

export interface SessionAppConfig {
  id: string
  name: string
  url?: string
  notes?: string
}

export interface SessionUsersData {
  session_id: string
  platform: string
  cluster_admin?: SessionUserCredential
  domain_users?: SessionDomainUser[]
  local_users?: SessionUserCredential[]
  apps?: SessionAppConfig[]
  module_domains?: Record<string, string> // moduleID → user domain
}

export interface SessionUsersReport {
  created_at: string
  duration_ms: number
  users: SessionUsersData
}

export interface SessionUsersResponse {
  session_id: string
  users: SessionUsersReport | null
  users_at: string | null
}

export const getSupportSessionUsers = (sessionId: string): Promise<SessionUsersResponse> => {
  const loginStore = useLoginStore()
  return axios
    .get<{
      code: number
      message: string
      data: SessionUsersResponse
    }>(`${API_URL}/support-sessions/${encodeURIComponent(sessionId)}/users`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getSystemDiagnostics = (systemId: string): Promise<SystemDiagnostics> => {
  const loginStore = useLoginStore()
  return axios
    .get<{
      code: number
      message: string
      data: SystemDiagnostics
    }>(`${API_URL}/support-sessions/diagnostics?system_id=${encodeURIComponent(systemId)}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

// Support history of a system: its sessions and the operator accesses and
// actions across all of them, newest first
export interface SupportActivitySession {
  id: string
  node_id?: string
  status: SupportSessionStatus
  started_at: string
  expires_at: string
  closed_at?: string
  closed_by?: string
}

export interface SupportActivityLog {
  id: string
  session_id: string
  operator_id: string
  operator_name?: string
  access_type: string
  connected_at: string
  disconnected_at?: string
  // JSON object, as stored
  metadata?: string
}

export interface SupportWebPresence {
  session_id: string
  operator_id: string
  operator_name: string
  service: string
  last_seen: string
}

export interface SupportActivity {
  system_id: string
  sessions: SupportActivitySession[]
  access_logs: SupportActivityLog[]
  // operators on the web interfaces of the live sessions in the last minutes
  web_presence?: SupportWebPresence[]
}

export interface SupportNote {
  id: string
  session_id: string
  system_id: string
  author_id: string
  author_name?: string
  body: string
  created_at: string
  updated_at: string
}

export const SUPPORT_NOTE_MAX_LENGTH = 10000

export const getSupportActivity = (systemId: string) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams({ system_id: systemId })

  return axios
    .get<{ data: SupportActivity }>(`${API_URL}/support-sessions/activity?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getSupportNotes = (systemId: string) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams({ system_id: systemId })

  return axios
    .get<{ data: { notes: SupportNote[] } }>(`${API_URL}/support-notes?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.notes)
}

export const createSupportNote = (systemId: string, body: string) => {
  const loginStore = useLoginStore()

  return axios
    .post<{ data: SupportNote }>(
      `${API_URL}/support-notes`,
      { system_id: systemId, body },
      { headers: { Authorization: `Bearer ${loginStore.jwtToken}` } },
    )
    .then((res) => res.data.data)
}

export const updateSupportNote = (noteId: string, body: string) => {
  const loginStore = useLoginStore()

  return axios
    .put<{ data: SupportNote }>(
      `${API_URL}/support-notes/${encodeURIComponent(noteId)}`,
      { body },
      { headers: { Authorization: `Bearer ${loginStore.jwtToken}` } },
    )
    .then((res) => res.data.data)
}

export const deleteSupportNote = (noteId: string) => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/support-notes/${encodeURIComponent(noteId)}`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

// Splits text into plain parts and web links, so links render as anchors and
// everything else stays text (never HTML)
export function linkifyText(text: string): { text: string; href?: string }[] {
  const parts: { text: string; href?: string }[] = []
  const re = /\bhttps?:\/\/[^\s<>"'`]+/gi
  let last = 0
  for (const match of text.matchAll(re)) {
    let url = match[0]
    // punctuation closing a sentence is not part of the link
    const trailing = url.match(/[.,;:!?)\]}]+$/)
    if (trailing) url = url.slice(0, -trailing[0].length)
    const start = match.index ?? 0
    let href: string | undefined
    try {
      const parsed = new URL(url)
      if (parsed.protocol === 'https:' || parsed.protocol === 'http:') href = parsed.href
    } catch {
      href = undefined
    }
    if (!href) continue
    if (start > last) parts.push({ text: text.slice(last, start) })
    parts.push({ text: url, href })
    last = start + url.length
  }
  if (last < text.length) parts.push({ text: text.slice(last) })
  return parts
}

// One support intervention on a system: its sessions overlapping in time (a
// multi-node cluster opens one per node), newest first
export interface SupportEpisode {
  id: string
  startedAt: Date
  endedAt: Date | null
  live: boolean
  sessionIds: string[]
}

export function groupSupportEpisodes(sessions: SupportActivitySession[]): SupportEpisode[] {
  const sorted = [...sessions].sort(
    (a, b) => new Date(a.started_at).getTime() - new Date(b.started_at).getTime(),
  )
  const episodes: SupportEpisode[] = []
  let current: SupportEpisode | null = null
  let currentEnd = 0
  for (const s of sorted) {
    const start = new Date(s.started_at).getTime()
    const live = s.status === 'active' || s.status === 'pending'
    const end = live ? Infinity : new Date(s.closed_at || s.expires_at).getTime()
    if (!current || start > currentEnd) {
      current = { id: s.id, startedAt: new Date(start), endedAt: null, live, sessionIds: [] }
      episodes.push(current)
      currentEnd = end
    }
    current.sessionIds.push(s.id)
    current.live = current.live || live
    currentEnd = Math.max(currentEnd, end)
    current.endedAt = current.live ? null : new Date(currentEnd)
  }
  return episodes.reverse()
}

// What only the browser sees goes to the session's access log: a login form
// filled with the session credentials, a web interface tab closed. Best
// effort: the history is an aid, the operator's work goes on without it.
export const logSupportWorkspaceEvent = (
  sessionId: string,
  type: 'autofill' | 'ui_close',
  service: string,
) => {
  const loginStore = useLoginStore()

  return axios
    .post(
      `${API_URL}/support-sessions/${encodeURIComponent(sessionId)}/events`,
      { type, service },
      { headers: { Authorization: `Bearer ${loginStore.jwtToken}` } },
    )
    .catch(() => undefined)
}
