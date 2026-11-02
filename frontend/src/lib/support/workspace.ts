//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  faClockRotateLeft,
  faKey,
  faNoteSticky,
  type IconDefinition,
} from '@fortawesome/free-solid-svg-icons'
import {
  getSupportSessionServices,
  getSupportSessionUsers,
  type SessionDomainUser,
  type SessionRef,
  type SessionUserCredential,
} from '@/lib/support/support'

export const SUPPORT_WORKSPACE_ATTACH_MESSAGE = 'my-support-workspace-attach'
export const SUPPORT_WORKSPACE_ATTACHED_MESSAGE = 'my-support-workspace-attached'
// sessionStorage key prefix that hands the tabs to a detached window: a popup
// starts with a copy of its opener's sessionStorage, so they never go in a URL
export const SUPPORT_WORKSPACE_HANDOFF_PREFIX = 'supportWorkspaceHandoff:'
export const SUPPORT_OPEN_LINK_MESSAGE = 'my-support-open'

// The side panel of a workspace: services and credentials, the system's
// support history, the operators' notes
export type WorkspacePanelSection = 'services' | 'activity' | 'notes'

export const WORKSPACE_PANEL_SECTIONS: {
  id: WorkspacePanelSection
  icon: IconDefinition
  label: string
}[] = [
  { id: 'services', icon: faKey, label: 'support.services_and_credentials' },
  { id: 'activity', icon: faClockRotateLeft, label: 'support.activity_title' },
  { id: 'notes', icon: faNoteSticky, label: 'support.notes_title' },
]

// What a workspace tab shows: enough to reopen it in another window, never a
// token or a credential
export type WorkspaceTabSpec =
  | { kind: 'terminal'; sessionId: string }
  | {
      kind: 'web'
      sessionId: string
      service: string
      path: string
      label: string
      // path is a page followed from a link, not the service's route prefix
      exact?: boolean
    }

export interface WorkspaceLogin {
  username: string
  password: string
}

export const SUPPORT_AUTOFILL_MESSAGE = 'my-support-autofill'
export const SUPPORT_AUTOFILL_RESULT_MESSAGE = 'my-support-autofill-result'

export interface WorkspaceCredentials {
  clusterAdmin: SessionUserCredential | null
  domainUsers: SessionDomainUser[]
  localUsers: SessionUserCredential[]
}

export interface WorkspaceService {
  name: string
  label: string
  session: SessionRef
  host: string
  path: string
  clusterAuth: boolean
}

export interface WorkspaceModule {
  moduleId: string
  moduleLabel: string
  nodeId: string
  domain: string
  password: string
  services: WorkspaceService[]
}

export interface WorkspaceData {
  credentials: WorkspaceCredentials | null
  // the labels given to the cluster nodes, by node ID
  nodeLabels: Record<string, string>
  modules: WorkspaceModule[]
  // services that belong to no module: the appliance UIs and the custom ones
  ungrouped: WorkspaceService[]
}

// The appliance's own UIs, shown next to the credentials that open them
export const APPLIANCE_UI_SERVICES = ['cluster-admin', 'nethsecurity-ui']

export const serviceDisplayName = (svc: WorkspaceService): string => {
  if (svc.label && !svc.host) return svc.label
  return svc.host ? `${svc.host}${svc.path || ''}` : svc.label || svc.name
}

// "Node 2 · Milan" when the node has a label in the cluster, "Node 2" otherwise
export const nodeDisplayName = (
  t: (key: string, values: Record<string, unknown>) => string,
  nodeId: string,
  labels?: Record<string, string>,
): string => {
  const name = t('support.activity_node', { node: nodeId })
  const label = labels?.[nodeId]
  return label ? `${name} · ${label}` : name
}

/**
 * Loads the services and the support credentials of every active session of a
 * system (one per cluster node) and merges them by module.
 */
export const loadWorkspaceData = async (sessions: SessionRef[]): Promise<WorkspaceData> => {
  const activeSessions = sessions.filter((s) => s.status === 'active')

  const [servicesResults, usersResults] = await Promise.all([
    Promise.all(activeSessions.map((s) => getSupportSessionServices(s.id).catch(() => []))),
    Promise.all(activeSessions.map((s) => getSupportSessionUsers(s.id).catch(() => null))),
  ])

  const credentials: WorkspaceCredentials = { clusterAdmin: null, domainUsers: [], localUsers: [] }
  const seenDomains = new Set<string>()
  const seenLocal = new Set<string>()
  const moduleDomains = new Map<string, string>()
  for (const r of usersResults) {
    const u = r?.users?.users
    if (!u) continue
    if (u.cluster_admin && !credentials.clusterAdmin) credentials.clusterAdmin = u.cluster_admin
    for (const du of u.domain_users || []) {
      if (!seenDomains.has(du.domain)) {
        seenDomains.add(du.domain)
        credentials.domainUsers.push(du)
      }
    }
    for (const lu of u.local_users || []) {
      if (!seenLocal.has(lu.username)) {
        seenLocal.add(lu.username)
        credentials.localUsers.push(lu)
      }
    }
    for (const [modId, domain] of Object.entries(u.module_domains || {})) {
      moduleDomains.set(modId, domain)
    }
  }
  const hasCredentials =
    credentials.clusterAdmin ||
    credentials.domainUsers.length > 0 ||
    credentials.localUsers.length > 0

  const domainPasswords = new Map<string, string>()
  for (const du of credentials.domainUsers) {
    domainPasswords.set(du.domain, du.password)
  }

  const moduleMap = new Map<string, WorkspaceModule>()
  const ungrouped: WorkspaceService[] = []
  const seenServices = new Set<string>()
  const nodeLabels: Record<string, string> = {}

  activeSessions.forEach((session, idx) => {
    for (const sg of servicesResults[idx] || []) {
      for (const svc of sg.services) {
        if (svc.nodeId && svc.nodeLabel && !nodeLabels[svc.nodeId]) {
          nodeLabels[svc.nodeId] = svc.nodeLabel
        }
        if (seenServices.has(svc.name)) continue
        seenServices.add(svc.name)

        const item: WorkspaceService = {
          name: svc.name,
          label: svc.label,
          session,
          host: svc.host,
          path: svc.path,
          clusterAuth: svc.auth === 'cluster',
        }
        if (!svc.moduleId) {
          ungrouped.push(item)
          continue
        }

        let mg = moduleMap.get(svc.moduleId)
        if (!mg) {
          const domain = moduleDomains.get(svc.moduleId) || ''
          mg = {
            moduleId: svc.moduleId,
            moduleLabel: svc.label || sg.moduleLabel || '',
            nodeId: svc.nodeId || sg.nodeId || '',
            domain,
            password: domain ? domainPasswords.get(domain) || '' : '',
            services: [],
          }
          moduleMap.set(svc.moduleId, mg)
        }
        if (!mg.moduleLabel && (svc.label || sg.moduleLabel)) {
          mg.moduleLabel = svc.label || sg.moduleLabel
        }
        mg.services.push(item)
      }
    }
  })

  for (const mg of moduleMap.values()) {
    mg.services.sort((a, b) => {
      const ak = a.host ? `${a.host}${a.path || ''}` : a.name
      const bk = b.host ? `${b.host}${b.path || ''}` : b.name
      return ak.localeCompare(bk)
    })
  }

  const modules = Array.from(moduleMap.values()).sort((a, b) => {
    const nc = (a.nodeId || '').localeCompare(b.nodeId || '', undefined, { numeric: true })
    if (nc !== 0) return nc
    return a.moduleId.localeCompare(b.moduleId)
  })

  return {
    credentials: hasCredentials ? credentials : null,
    nodeLabels,
    modules,
    ungrouped: ungrouped.sort((a, b) => a.name.localeCompare(b.name)),
  }
}

export const encodeTabSpecs = (tabs: WorkspaceTabSpec[]): string =>
  btoa(encodeURIComponent(JSON.stringify(tabs)))

// Tab specs come from a URL or another window: keep only well-formed entries
export const decodeTabSpecs = (raw: unknown): WorkspaceTabSpec[] => {
  let value = raw
  if (typeof raw === 'string') {
    try {
      value = JSON.parse(decodeURIComponent(atob(raw)))
    } catch {
      return []
    }
  }
  if (!Array.isArray(value)) return []
  const tabs: WorkspaceTabSpec[] = []
  for (const t of value.slice(0, 20)) {
    if (!t || typeof t !== 'object' || typeof t.sessionId !== 'string') continue
    if (t.kind === 'terminal') {
      tabs.push({ kind: 'terminal', sessionId: t.sessionId })
    } else if (t.kind === 'web' && typeof t.service === 'string') {
      tabs.push({
        kind: 'web',
        sessionId: t.sessionId,
        service: t.service,
        path: typeof t.path === 'string' ? t.path : '',
        label: typeof t.label === 'string' ? t.label : t.service,
        exact: t.exact === true,
      })
    }
  }
  return tabs
}

const SERVICE_HOST_LABEL_MAX_PREFIX = 20

/**
 * The DNS label a service gets in its proxy host: a readable prefix plus 8 hex
 * chars of sha256(name). Must stay identical to the backend's
 * helpers.ServiceHostLabel.
 */
export const serviceHostLabel = async (name: string): Promise<string> => {
  let prefix = ''
  let dash = false
  for (const ch of name.toLowerCase()) {
    if ((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')) {
      prefix += ch
      dash = false
    } else if (!dash && prefix.length > 0) {
      prefix += '-'
      dash = true
    }
  }
  prefix = prefix.slice(0, SERVICE_HOST_LABEL_MAX_PREFIX).replace(/-+$/, '') || 'svc'
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(name))
  const hex = Array.from(new Uint8Array(digest).slice(0, 4))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')
  return `${prefix}-${hex}`
}
