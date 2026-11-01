//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { getSystemActiveSessions, type SessionRef } from '@/lib/support/support'
import { decodeTabSpecs, type WorkspaceTabSpec } from '@/lib/support/workspace'
import { useLoginStore } from '@/stores/login'
import { getManagedBy } from '@/lib/organizations/organizations'
import type { SystemSessionGroup } from '@/lib/support/support'

export const SUPPORT_WORKSPACES_STORAGE_KEY = 'supportWorkspaces'

// Names shown next to the system: the company it is assigned to and the one
// that manages it
export interface WorkspaceOrganizations {
  organizationName: string
  organizationType: string
  managedByName: string
  managedByType: string
}

const pickOrganizations = (o: Partial<WorkspaceOrganizations>): Partial<WorkspaceOrganizations> =>
  Object.fromEntries(
    Object.entries({
      organizationName: o.organizationName,
      organizationType: o.organizationType,
      managedByName: o.managedByName,
      managedByType: o.managedByType,
    }).filter(([, v]) => typeof v === 'string' && v),
  )

export const workspaceOrganizations = (group: SystemSessionGroup): WorkspaceOrganizations => {
  const managedBy = group.organization ? getManagedBy(group.organization) : undefined
  return {
    organizationName: group.organization?.name || '',
    organizationType: group.organization?.type || '',
    managedByName: managedBy?.name || '',
    managedByType: managedBy?.type || '',
  }
}

export interface SupportWorkspaceWindow extends WorkspaceOrganizations {
  systemId: string
  systemName: string
  systemType: string | null
  sessions: SessionRef[]
  minimized: boolean
  // tabs to open, handed over by another window or by a page reload
  requestedTabs: WorkspaceTabSpec[]
  // the open tabs and the side panel, saved to survive a page reload
  tabs: WorkspaceTabSpec[]
  panelOpen: boolean | null
  // restored after a reload and not reopened yet: no connection until then
  dormant: boolean
  waking: boolean
}

interface SavedWindow extends WorkspaceOrganizations {
  systemId: string
  systemName: string
  systemType: string | null
  minimized: boolean
  tabs: WorkspaceTabSpec[]
  panelOpen: boolean | null
  // node of each session, to move the tabs to the node's current session
  sessionNodes: Record<string, string | null>
}

// Support windows live in the app shell, so a minimized one keeps its
// terminals and web UIs open while the operator moves around the app. Their
// layout (never a token or a credential) is kept in sessionStorage, so a page
// reload brings the windows back, their connections reopened on demand.
export const useSupportWorkspacesStore = defineStore('supportWorkspaces', () => {
  const loginStore = useLoginStore()
  const windows = ref<SupportWorkspaceWindow[]>([])
  const hasMinimized = computed(() => windows.value.some((w) => w.minimized))
  const sessionNodes = new Map<string, Record<string, string | null>>()
  let hydratedFor = ''

  function find(systemId: string) {
    return windows.value.find((w) => w.systemId === systemId)
  }

  function minimizeOthers(systemId: string) {
    for (const w of windows.value) {
      if (w.systemId !== systemId) {
        w.minimized = true
      }
    }
  }

  function open(
    system: {
      systemId: string
      systemName: string
      systemType: string | null
    } & Partial<WorkspaceOrganizations>,
    sessions: SessionRef[],
    tabs: WorkspaceTabSpec[] = [],
  ) {
    minimizeOthers(system.systemId)
    const existing = find(system.systemId)
    if (existing) {
      // the caller's sessions are the freshest: a node may have restarted its
      // session since the window was opened or saved
      if (sessions.length) existing.sessions = sessions
      Object.assign(existing, pickOrganizations(system))
      existing.requestedTabs = existing.dormant ? tabs : [...existing.requestedTabs, ...tabs]
      existing.dormant = false
      existing.minimized = false
      return
    }
    windows.value.push({
      organizationName: '',
      organizationType: '',
      managedByName: '',
      managedByType: '',
      ...system,
      sessions,
      minimized: false,
      requestedTabs: tabs,
      tabs: [],
      panelOpen: null,
      dormant: false,
      waking: false,
    })
  }

  // a workspace found new live sessions for its system: chips and saved layout
  // follow them
  function updateSessions(systemId: string, sessions: SessionRef[]) {
    const w = find(systemId)
    if (w) w.sessions = sessions
  }

  function tabsConsumed(systemId: string) {
    const w = find(systemId)
    if (w) {
      w.requestedTabs = []
    }
  }

  function updateLayout(systemId: string, tabs: WorkspaceTabSpec[], panelOpen: boolean) {
    const w = find(systemId)
    if (w) {
      w.tabs = tabs
      w.panelOpen = panelOpen
    }
  }

  function minimize(systemId: string) {
    const w = find(systemId)
    if (w) {
      w.minimized = true
    }
  }

  // Reopens a window restored after a reload: its sessions may have changed
  // (closed, or restarted on a node) since the layout was saved
  async function wake(w: SupportWorkspaceWindow) {
    w.waking = true
    try {
      const groups = await getSystemActiveSessions(w.systemId)
      const group = groups.find(
        (g) => g.system_id === w.systemId && g.sessions.some((s) => s.status === 'active'),
      )
      const sessions = group?.sessions || []
      const active = sessions.filter((s) => s.status === 'active')
      if (!active.length) {
        close(w.systemId)
        return
      }
      const nodes = sessionNodes.get(w.systemId) || {}
      const remap = (sessionId: string) => {
        if (active.some((s) => s.id === sessionId)) return sessionId
        // unknown session: no guessing, a tab must not jump to another node
        if (!Object.hasOwn(nodes, sessionId)) return undefined
        const node = nodes[sessionId] ?? null
        return active.find((s) => (s.node_id ?? null) === node)?.id
      }
      const tabs: WorkspaceTabSpec[] = []
      for (const tab of w.tabs) {
        const sessionId = remap(tab.sessionId)
        if (sessionId) tabs.push({ ...tab, sessionId })
      }
      if (group) Object.assign(w, workspaceOrganizations(group))
      w.sessions = sessions
      w.requestedTabs = tabs
      w.dormant = false
      minimizeOthers(w.systemId)
      w.minimized = false
    } catch (error) {
      console.error('Cannot reopen the support window:', error)
    } finally {
      w.waking = false
    }
  }

  // Windows saved or opened without the company names get them once, from
  // the same read that reopens them, without opening any connection
  const namesRequested = new Set<string>()

  async function fillOrganizations(systemId: string) {
    if (namesRequested.has(systemId)) return
    namesRequested.add(systemId)
    try {
      const groups = await getSystemActiveSessions(systemId)
      const w = find(systemId)
      const group = groups.find((g) => g.system_id === systemId)
      if (w && group) Object.assign(w, workspaceOrganizations(group))
    } catch {
      // names are a nicety: the window works without them
    } finally {
      // a later change (a backend that now knows the manager) may ask again
      namesRequested.delete(systemId)
    }
  }

  function restore(systemId: string) {
    const w = find(systemId)
    if (!w) return
    if (w.dormant) {
      if (!w.waking) wake(w)
      return
    }
    minimizeOthers(systemId)
    w.minimized = false
  }

  function close(systemId: string) {
    windows.value = windows.value.filter((w) => w.systemId !== systemId)
    sessionNodes.delete(systemId)
  }

  // On logout or a lost login every window goes, connections included, and
  // so does the saved layout of this tab
  function reset() {
    windows.value = []
    sessionNodes.clear()
    hydratedFor = ''
    try {
      for (const key of Object.keys(sessionStorage)) {
        if (key.startsWith(SUPPORT_WORKSPACES_STORAGE_KEY)) sessionStorage.removeItem(key)
      }
    } catch {
      // storage unavailable: nothing was saved
    }
  }

  watch(
    () => loginStore.isAuthenticated,
    (authenticated, wasAuthenticated) => {
      if (!authenticated && wasAuthenticated) reset()
    },
  )

  function storageKey(): string {
    const email = loginStore.userInfo?.email
    return email ? `${SUPPORT_WORKSPACES_STORAGE_KEY}:${email}` : ''
  }

  function save() {
    const key = storageKey()
    if (!key || hydratedFor !== key) return
    const saved: SavedWindow[] = windows.value.map((w) => ({
      systemId: w.systemId,
      systemName: w.systemName,
      systemType: w.systemType,
      organizationName: w.organizationName,
      organizationType: w.organizationType,
      managedByName: w.managedByName,
      managedByType: w.managedByType,
      minimized: w.minimized,
      tabs: w.tabs,
      panelOpen: w.panelOpen,
      sessionNodes: w.dormant
        ? sessionNodes.get(w.systemId) || {}
        : Object.fromEntries(w.sessions.map((s) => [s.id, s.node_id ?? null])),
    }))
    try {
      if (saved.length) {
        sessionStorage.setItem(key, JSON.stringify(saved))
      } else {
        sessionStorage.removeItem(key)
      }
    } catch {
      // storage unavailable: the layout just does not survive a reload
    }
  }

  // Brings back the windows saved before a reload, all dormant: the one that
  // was in front reopens right away, the minimized ones when clicked
  function hydrate() {
    const key = storageKey()
    if (!key || hydratedFor === key) return
    hydratedFor = key
    let saved: SavedWindow[] = []
    try {
      const parsed = JSON.parse(sessionStorage.getItem(key) || '[]')
      saved = Array.isArray(parsed) ? parsed : []
    } catch {
      saved = []
    }
    let front: SupportWorkspaceWindow | undefined
    for (const s of saved.slice(0, 20)) {
      if (!s || typeof s.systemId !== 'string' || find(s.systemId)) continue
      const w: SupportWorkspaceWindow = {
        systemId: s.systemId,
        systemName: typeof s.systemName === 'string' ? s.systemName : s.systemId,
        systemType: typeof s.systemType === 'string' ? s.systemType : null,
        organizationName: typeof s.organizationName === 'string' ? s.organizationName : '',
        organizationType: typeof s.organizationType === 'string' ? s.organizationType : '',
        managedByName: typeof s.managedByName === 'string' ? s.managedByName : '',
        managedByType: typeof s.managedByType === 'string' ? s.managedByType : '',
        sessions: [],
        minimized: true,
        requestedTabs: [],
        tabs: decodeTabSpecs(s.tabs),
        panelOpen: typeof s.panelOpen === 'boolean' ? s.panelOpen : null,
        dormant: true,
        waking: false,
      }
      const nodes = s.sessionNodes && typeof s.sessionNodes === 'object' ? s.sessionNodes : {}
      sessionNodes.set(w.systemId, nodes)
      windows.value.push(w)
      if (!s.minimized) front = w
    }
    // the reactive copy, not the plain object pushed above
    const frontWindow = front && find(front.systemId)
    if (frontWindow) wake(frontWindow)
  }

  watch(windows, save, { deep: true })

  return {
    reset,
    windows,
    hasMinimized,
    open,
    tabsConsumed,
    updateSessions,
    updateLayout,
    minimize,
    restore,
    close,
    hydrate,
    fillOrganizations,
  }
})
