<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { onClickOutside } from '@vueuse/core'
import {
  faGlobe,
  faHeartPulse,
  faPlus,
  faTerminal,
  faUpRightFromSquare,
  faWindowMinimize,
  faWindowRestore,
  faXmark,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeModal, NeTextInput } from '@nethesis/vue-components'
import {
  buildSupportProxyUrl,
  generateSupportProxyToken,
  getSystemActiveSessions,
  logSupportWorkspaceEvent,
  removeSupportSessionService,
  type SessionRef,
} from '@/lib/support/support'
import {
  APPLIANCE_UI_SERVICES,
  loadWorkspaceData,
  nodeDisplayName,
  serviceDisplayName,
  serviceHostLabel,
  WORKSPACE_PANEL_SECTIONS,
  type WorkspaceData,
  type WorkspacePanelSection,
  type WorkspaceLogin,
  type WorkspaceService,
  type WorkspaceTabSpec,
} from '@/lib/support/workspace'
import SystemLogo from '@/components/systems/SystemLogo.vue'
import SupportTerminalPane, {
  type TerminalPhase,
} from '@/components/support/SupportTerminalPane.vue'
import SupportWebPane from '@/components/support/SupportWebPane.vue'
import SupportWorkspacePanel from '@/components/support/SupportWorkspacePanel.vue'
import SupportAddServiceModal from '@/components/support/SupportAddServiceModal.vue'
import SupportDiagnosticsPane from '@/components/support/SupportDiagnosticsPane.vue'

interface WorkspaceTab {
  id: number
  spec: WorkspaceTabSpec
  label: string
  phase: TerminalPhase | null
}

const props = defineProps<{
  systemId: string
  systemName: string
  systemType?: string | null
  organizationName?: string
  managedByName?: string
  sessions: SessionRef[]
  mode: 'app' | 'detached'
  minimized?: boolean
  requestedTabs?: WorkspaceTabSpec[]
  // the panel state saved before a page reload
  initialPanelOpen?: boolean | null
}>()

const emit = defineEmits<{
  minimize: []
  detach: [tabs: WorkspaceTabSpec[]]
  attach: [tabs: WorkspaceTabSpec[]]
  close: []
  tabsConsumed: []
  layout: [tabs: WorkspaceTabSpec[], panelOpen: boolean]
  // the system's live sessions changed (a node restarted its session)
  sessionsChanged: [sessions: SessionRef[]]
}>()

const { t } = useI18n()

const tabs = ref<WorkspaceTab[]>([])
const activeTabId = ref(0)
let nextTabId = 1

const data = ref<WorkspaceData | null>(null)
const dataLoading = ref(false)
const panelOpen = ref(props.initialPanelOpen ?? true)
const panelSection = ref<WorkspacePanelSection>('services')
const notesCount = ref(0)

// the rail is always there: its active entry closes the panel, any other
// one opens it on that section
function togglePanel(section: WorkspacePanelSection) {
  if (panelOpen.value && panelSection.value === section) {
    panelOpen.value = false
    return
  }
  panelSection.value = section
  panelOpen.value = true
}

const menuOpen = ref(false)
const menuSearch = ref('')
const menuRef = ref<HTMLElement | null>(null)
onClickOutside(menuRef, () => {
  menuOpen.value = false
})

// the + menu opens on a fresh search, ready to type
watch(menuOpen, (open) => {
  if (!open) return
  menuSearch.value = ''
  menuIndex.value = 0
  nextTick(() => menuRef.value?.querySelector('input')?.focus())
})

const showCloseConfirm = ref(false)
const pendingMove = ref<'detach' | 'attach' | null>(null)

// The sessions follow the system: a node that lost its tunnel comes back with
// a new session, which must replace the closed one in the menu and the panel
const SESSIONS_POLL_INTERVAL_MS = 30_000
const sessions = ref<SessionRef[]>(props.sessions)
const sessionsKey = (list: SessionRef[]) =>
  list
    .map((s) => `${s.id}:${s.status}`)
    .sort()
    .join(',')
watch(
  () => props.sessions,
  (list) => {
    if (sessionsKey(list) !== sessionsKey(sessions.value)) sessions.value = list
  },
)
let sessionsTimer: ReturnType<typeof setInterval> | null = null

async function syncSessions() {
  try {
    const groups = await getSystemActiveSessions(props.systemId)
    const list = groups.find((g) => g.system_id === props.systemId)?.sessions || []
    if (list.length && sessionsKey(list) !== sessionsKey(sessions.value)) {
      sessions.value = list
      emit('sessionsChanged', list)
    }
  } catch {
    // the next tick retries; a lost login is handled by the API layer
  }
}

function onVisible() {
  if (document.visibilityState === 'visible') syncSessions()
}

const activeSessions = computed(() => sessions.value.filter((s) => s.status === 'active'))
const isMultiNode = computed(() => activeSessions.value.filter((s) => s.node_id).length > 1)

function terminalPhaseLabel(phase: TerminalPhase): string {
  if (phase === 'connected') return t('support.terminal_connected')
  if (phase === 'connecting') return t('support.terminal_connecting')
  return t('support.terminal_connection_error')
}

function terminalLabel(sessionId: string): string {
  const nodeId = sessions.value.find((s) => s.id === sessionId)?.node_id
  return isMultiNode.value && nodeId
    ? `${t('support.workspace_terminal')} · ${nodeName(nodeId)}`
    : t('support.workspace_terminal')
}

function nodeName(nodeId: string): string {
  return nodeDisplayName(t, nodeId, data.value?.nodeLabels)
}

// Each tab holds a live connection: a page asking for many must not exhaust
// the browser
const MAX_TABS = 20
const tabsLimitNotice = ref(false)

function addTab(spec: WorkspaceTabSpec) {
  if (!activeSessions.value.some((s) => s.id === spec.sessionId)) return
  if (tabs.value.length >= MAX_TABS) {
    tabsLimitNotice.value = true
    setTimeout(() => (tabsLimitNotice.value = false), 4000)
    return
  }
  if (spec.kind === 'web') {
    // one tab per page: a second one would only share its login
    const existing = tabs.value.find(
      (tab) =>
        tab.spec.kind === 'web' && tab.spec.service === spec.service && tab.spec.path === spec.path,
    )
    if (existing) {
      activeTabId.value = existing.id
      return
    }
  }
  const id = nextTabId++
  const sameKind = tabs.value.filter(
    (tab) => tab.spec.kind === 'terminal' && tab.spec.sessionId === spec.sessionId,
  ).length
  // a terminal keeps only its number: the node name may arrive later
  const label = spec.kind === 'terminal' ? (sameKind ? ` #${sameKind + 1}` : '') : spec.label
  tabs.value.push({ id, spec, label, phase: spec.kind === 'terminal' ? 'connecting' : null })
  activeTabId.value = id
  menuOpen.value = false
}

function openTerminal(session: SessionRef) {
  addTab({ kind: 'terminal', sessionId: session.id })
}

function openTerminalById(sessionId: string) {
  addTab({ kind: 'terminal', sessionId })
}

const panelTerminals = computed(() =>
  activeSessions.value
    .slice()
    .sort((a, b) => (a.node_id || '').localeCompare(b.node_id || '', undefined, { numeric: true }))
    .map((s) => ({ sessionId: s.id, node: s.node_id ? nodeName(s.node_id) : '' })),
)

function openService(svc: WorkspaceService) {
  addTab({
    kind: 'web',
    sessionId: svc.session.id,
    service: svc.name,
    path: svc.path,
    label: svc.label || svc.host || svc.name,
  })
}

// A link inside a web tab that opens another page of the session (e.g. the
// wizard's Advanced, FreePBX): it becomes a tab here. The label is matched
// against the session's own services only, so no other host can be opened.
const hostLabels = new Map<string, string>()

async function openLink(label: string, slug: string, path: string) {
  // only a service of the session the link points to (its host carries it)
  const services = [
    ...(data.value?.modules.flatMap((mg) => mg.services) || []),
    ...(data.value?.ungrouped || []),
  ].filter((svc) => svc.session.id.replace(/-/g, '') === slug)
  for (const svc of services) {
    if (!hostLabels.has(svc.name)) hostLabels.set(svc.name, await serviceHostLabel(svc.name))
    if (hostLabels.get(svc.name) !== label) continue
    const page = path.split(/[?#]/)[0].replace(/\/+$/, '').split('/').pop()
    addTab({
      kind: 'web',
      sessionId: svc.session.id,
      service: svc.name,
      path,
      label: `${svc.label || svc.host || svc.name}${page ? ` · ${page}` : ''}`,
      exact: true,
    })
    return
  }
}

async function openServiceExternal(svc: WorkspaceService) {
  try {
    const result = await generateSupportProxyToken(svc.session.id, svc.name)
    const url = buildSupportProxyUrl(result.url, result.token, svc.path)
    if (url) {
      window.open(url, '_blank', 'noopener,noreferrer')
    }
  } catch (error) {
    console.error('Cannot generate proxy token:', error)
  }
}

async function removeService(svc: WorkspaceService) {
  try {
    await removeSupportSessionService(svc.session.id, svc.name)
    tabs.value = tabs.value.filter(
      (tab) => !(tab.spec.kind === 'web' && tab.spec.service === svc.name),
    )
    await refreshData()
  } catch (error) {
    console.error('Cannot remove service:', error)
  }
}

// the session history tells when the operator left a web interface; a
// terminal's end is recorded by the server when its connection closes
function logWebClose(closing: WorkspaceTab[]) {
  const seen = new Set<string>()
  for (const tab of closing) {
    const spec = tab.spec
    if (spec.kind !== 'web') continue
    const key = `${spec.sessionId}|${spec.service}`
    // another tab still shows the service: the operator has not left it
    if (seen.has(key)) continue
    seen.add(key)
    const stillOpen = tabs.value.some(
      (other) =>
        !closing.includes(other) &&
        other.spec.kind === 'web' &&
        other.spec.sessionId === spec.sessionId &&
        other.spec.service === spec.service,
    )
    if (!stillOpen) logSupportWorkspaceEvent(spec.sessionId, 'ui_close', spec.service)
  }
}

function closeTab(id: number) {
  const idx = tabs.value.findIndex((tab) => tab.id === id)
  if (idx === -1) return
  logWebClose([tabs.value[idx]])
  const barIdx = tabOrder.value.indexOf(id)
  tabs.value.splice(idx, 1)
  if (activeTabId.value === id) {
    const order = tabOrder.value.filter((tabId) => tabId !== id)
    activeTabId.value = order[Math.min(barIdx, order.length - 1)] ?? 0
  }
}

// The tab bar has its own order: the panes keep theirs, since moving an
// iframe in the page reloads it
const tabOrder = ref<number[]>([])

watch(
  () => tabs.value.map((tab) => tab.id),
  (ids) => {
    const kept = tabOrder.value.filter((id) => ids.includes(id))
    tabOrder.value = [...kept, ...ids.filter((id) => !kept.includes(id))]
  },
  { immediate: true },
)

const orderedTabs = computed(() =>
  tabOrder.value
    .map((id) => tabs.value.find((tab) => tab.id === id))
    .filter((tab): tab is WorkspaceTab => !!tab),
)

const draggedTabId = ref<number | null>(null)
// where the dragged tab lands: before the tab with this id, or at the end (-1)
const dropBeforeId = ref<number | null>(null)

function onTabDragStart(event: DragEvent, id: number) {
  draggedTabId.value = id
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', String(id))
  }
}

function onTabDragOver(event: DragEvent, id: number) {
  if (draggedTabId.value === null) return
  event.preventDefault()
  const el = event.currentTarget as HTMLElement
  const rect = el.getBoundingClientRect()
  const after = event.clientX > rect.left + rect.width / 2
  if (!after) {
    dropBeforeId.value = id
  } else {
    const idx = tabOrder.value.indexOf(id)
    dropBeforeId.value = tabOrder.value[idx + 1] ?? -1
  }
}

function onTabDrop() {
  const dragged = draggedTabId.value
  const before = dropBeforeId.value
  if (dragged !== null && before !== null && before !== dragged) {
    const order = tabOrder.value.filter((id) => id !== dragged)
    const at = before === -1 ? order.length : order.indexOf(before)
    order.splice(at === -1 ? order.length : at, 0, dragged)
    tabOrder.value = order
  }
  onTabDragEnd()
}

function onTabDragEnd() {
  draggedTabId.value = null
  dropBeforeId.value = null
}

const addServiceOpen = ref(false)

// the tunnel announces the new service a moment after accepting it
function onServiceAdded() {
  refreshData()
  setTimeout(refreshData, 2000)
}

async function refreshData() {
  dataLoading.value = true
  try {
    data.value = await loadWorkspaceData(sessions.value)
  } finally {
    dataLoading.value = false
  }
}

// The + menu: terminals first, then the appliance UI, the modules and the
// custom services, filtered by the search box
const menuQuery = computed(() => menuSearch.value.trim().toLowerCase())

function matches(...values: string[]): boolean {
  return !menuQuery.value || values.some((v) => v.toLowerCase().includes(menuQuery.value))
}

const menuApplianceServices = computed(
  () =>
    data.value?.ungrouped.filter(
      (s) => APPLIANCE_UI_SERVICES.includes(s.name) && matches(s.name, s.label),
    ) || [],
)
const menuModules = computed(
  () =>
    data.value?.modules
      .map((mg) => ({
        ...mg,
        services: mg.services.filter((s) =>
          matches(mg.moduleId, mg.moduleLabel, s.name, serviceDisplayName(s)),
        ),
      }))
      .filter((mg) => mg.services.length) || [],
)
const menuCustomServices = computed(
  () =>
    data.value?.ungrouped.filter(
      (s) => !APPLIANCE_UI_SERVICES.includes(s.name) && matches(s.name, s.label),
    ) || [],
)

// Keyboard navigation of the + menu: the items in display order, the arrows
// move the highlight, Enter opens it
const menuItems = computed(() => {
  const items: { key: string; run: () => void }[] = []
  if (!menuQuery.value) {
    for (const session of activeSessions.value) {
      items.push({ key: `t:${session.id}`, run: () => openTerminal(session) })
    }
  }
  if (!dataLoading.value) {
    const services = [
      ...menuApplianceServices.value,
      ...menuModules.value.flatMap((mg) => mg.services),
      ...menuCustomServices.value,
    ]
    for (const svc of services) {
      items.push({ key: `s:${svc.name}`, run: () => openService(svc) })
    }
  }
  return items
})
const menuIndex = ref(0)

function menuItemIndex(key: string): number {
  return menuItems.value.findIndex((item) => item.key === key)
}

function onMenuKeydown(event: KeyboardEvent) {
  const count = menuItems.value.length
  if (event.key === 'Escape') {
    menuOpen.value = false
    return
  }
  if (!count) return
  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    const step = event.key === 'ArrowDown' ? 1 : -1
    menuIndex.value = (menuIndex.value + step + count) % count
    nextTick(() =>
      menuRef.value
        ?.querySelector(`[data-menu-index="${menuIndex.value}"]`)
        ?.scrollIntoView({ block: 'nearest' }),
    )
  } else if (event.key === 'Enter') {
    event.preventDefault()
    menuItems.value[menuIndex.value]?.run()
  }
}

watch(menuQuery, () => {
  menuIndex.value = 0
})

// The login a web UI takes: the cluster admin for cluster-admin, the local
// user for the NethSecurity UI, the domain user for the apps of a module.
// Services behind the cluster login ask for it in a browser dialog instead.
function credentialsFor(service: string): WorkspaceLogin | null {
  const creds = data.value?.credentials
  if (!creds) return null
  if (service === APPLIANCE_UI_SERVICES[0]) return creds.clusterAdmin
  if (service === APPLIANCE_UI_SERVICES[1]) return creds.localUsers[0] ?? null
  const mg = data.value?.modules.find((m) => m.services.some((s) => s.name === service))
  const svc = mg?.services.find((s) => s.name === service)
  const username = creds.domainUsers[0]?.username
  if (!mg || !svc || svc.clusterAuth || !mg.password || !username) return null
  return { username, password: mg.password }
}

function tabSpecs(): WorkspaceTabSpec[] {
  return orderedTabs.value.map((tab) => tab.spec)
}

function handleMove(move: 'detach' | 'attach') {
  // terminals cannot follow: their shells end here
  if (tabs.value.some((tab) => tab.spec.kind === 'terminal')) {
    pendingMove.value = move
    return
  }
  doMove(move)
}

function doMove(move: 'detach' | 'attach') {
  pendingMove.value = null
  const specs = tabSpecs()
  tabs.value = []
  if (move === 'detach') {
    emit('detach', specs)
  } else {
    emit('attach', specs)
  }
}

function handleClose() {
  if (tabs.value.length) {
    showCloseConfirm.value = true
    return
  }
  emit('close')
}

function doClose() {
  showCloseConfirm.value = false
  logWebClose(tabs.value)
  tabs.value = []
  emit('close')
}

watch(
  () => props.requestedTabs,
  (requested) => {
    if (!requested?.length) return
    for (const spec of requested) addTab(spec)
    if (tabs.value.length && props.initialPanelOpen == null) panelOpen.value = false
    emit('tabsConsumed')
  },
  { immediate: true },
)

watch(
  [() => orderedTabs.value.map((tab) => tab.spec), panelOpen],
  ([specs, open]) => emit('layout', specs, open),
  { deep: true },
)

watch(
  () => sessionsKey(sessions.value),
  () => refreshData(),
)

onMounted(() => {
  refreshData()
  sessionsTimer = setInterval(syncSessions, SESSIONS_POLL_INTERVAL_MS)
  document.addEventListener('visibilitychange', onVisible)
})
onBeforeUnmount(() => {
  if (sessionsTimer) clearInterval(sessionsTimer)
  document.removeEventListener('visibilitychange', onVisible)
})
</script>

<template>
  <div class="fixed inset-0 z-50 flex flex-col bg-gray-900">
    <!-- header -->
    <div class="flex items-center justify-between gap-4 bg-gray-800 px-4 py-2">
      <div class="flex min-w-0 items-center gap-3">
        <SystemLogo v-if="systemType" :system="systemType" size="sm" />
        <span class="truncate text-sm font-medium text-gray-200">
          {{ t('support.workspace_title') }} &mdash; {{ systemName }}
        </span>
        <span v-if="organizationName" class="hidden truncate text-xs text-gray-400 md:inline">
          {{ t('support.workspace_assigned_to', { name: organizationName }) }}
          <template v-if="managedByName && managedByName !== organizationName">
            · {{ t('support.workspace_managed_by', { name: managedByName }) }}
          </template>
        </span>
      </div>
      <div class="flex shrink-0 items-center gap-1">
        <button
          v-if="mode === 'app'"
          class="rounded p-1.5 text-gray-400 hover:bg-gray-700 hover:text-gray-200"
          :title="t('support.workspace_minimize')"
          :aria-label="t('support.workspace_minimize')"
          @click="emit('minimize')"
        >
          <FontAwesomeIcon :icon="faWindowMinimize" class="h-4 w-4" aria-hidden="true" />
        </button>
        <button
          v-if="mode === 'app'"
          class="rounded p-1.5 text-gray-400 hover:bg-gray-700 hover:text-gray-200"
          :title="t('support.workspace_detach')"
          :aria-label="t('support.workspace_detach')"
          @click="handleMove('detach')"
        >
          <FontAwesomeIcon :icon="faUpRightFromSquare" class="h-4 w-4" aria-hidden="true" />
        </button>
        <button
          v-else
          class="flex items-center gap-2 rounded px-2 py-1 text-sm text-gray-400 hover:bg-gray-700 hover:text-gray-200"
          @click="handleMove('attach')"
        >
          <FontAwesomeIcon :icon="faWindowRestore" class="h-4 w-4" aria-hidden="true" />
          {{ t('support.workspace_attach') }}
        </button>
        <button
          class="rounded p-1.5 text-gray-400 hover:bg-gray-700 hover:text-gray-200"
          :title="t('support.workspace_close')"
          :aria-label="t('support.workspace_close')"
          @click="handleClose"
        >
          <FontAwesomeIcon :icon="faXmark" class="h-5 w-5" aria-hidden="true" />
        </button>
      </div>
    </div>

    <!-- tab bar: fixed height, the panel key comes and goes -->
    <div class="flex h-9 shrink-0 items-center border-b border-gray-700 bg-gray-800 px-2">
      <div class="flex min-w-0 items-center overflow-x-auto">
        <!-- diagnostics: always first, never closed or moved -->
        <button
          type="button"
          class="flex shrink-0 items-center gap-1.5 px-3 py-1.5 text-xs"
          :class="
            activeTabId === 0
              ? 'rounded-t bg-gray-900 text-gray-100'
              : 'text-gray-400 hover:bg-gray-700/50 hover:text-gray-200'
          "
          :aria-pressed="activeTabId === 0"
          @click="activeTabId = 0"
        >
          <FontAwesomeIcon
            :icon="faHeartPulse"
            class="h-3 w-3 shrink-0 text-sky-400"
            aria-hidden="true"
          />
          <span>{{ t('support.diagnostics') }}</span>
        </button>
        <div
          v-for="tab in orderedTabs"
          :key="tab.id"
          draggable="true"
          class="group relative flex max-w-60 shrink-0 items-center gap-1 pr-1 text-xs"
          :class="[
            tab.id === activeTabId
              ? 'rounded-t bg-gray-900 text-gray-100'
              : 'text-gray-400 hover:bg-gray-700/50 hover:text-gray-200',
            { 'opacity-40': tab.id === draggedTabId },
          ]"
          @dragstart="onTabDragStart($event, tab.id)"
          @dragover="onTabDragOver($event, tab.id)"
          @drop.prevent="onTabDrop"
          @dragend="onTabDragEnd"
        >
          <!-- drop marker -->
          <span
            v-if="draggedTabId !== null && dropBeforeId === tab.id"
            class="bg-primary-500 absolute inset-y-1 -left-px w-0.5 rounded"
            aria-hidden="true"
          ></span>
          <span
            v-if="
              draggedTabId !== null &&
              dropBeforeId === -1 &&
              tab.id === tabOrder[tabOrder.length - 1]
            "
            class="bg-primary-500 absolute inset-y-1 -right-px w-0.5 rounded"
            aria-hidden="true"
          ></span>
          <button
            type="button"
            class="flex min-w-0 items-center gap-1.5 py-1.5 pl-3"
            :aria-pressed="tab.id === activeTabId"
            @click="activeTabId = tab.id"
          >
            <span
              v-if="tab.spec.kind === 'terminal'"
              class="h-1.5 w-1.5 shrink-0 rounded-full"
              :class="{
                'bg-green-400': tab.phase === 'connected',
                'animate-pulse bg-yellow-400': tab.phase === 'connecting',
                'bg-red-400': tab.phase === 'error',
              }"
              aria-hidden="true"
            ></span>
            <FontAwesomeIcon
              v-else
              :icon="faGlobe"
              class="h-3 w-3 shrink-0 text-amber-500"
              aria-hidden="true"
            />
            <span class="truncate">{{
              tab.spec.kind === 'terminal'
                ? terminalLabel(tab.spec.sessionId) + tab.label
                : tab.label
            }}</span>
            <span v-if="tab.spec.kind === 'terminal' && tab.phase" class="sr-only">{{
              terminalPhaseLabel(tab.phase)
            }}</span>
          </button>
          <button
            type="button"
            class="shrink-0 rounded p-0.5 opacity-0 group-hover:opacity-100 hover:bg-gray-600 focus-visible:opacity-100"
            :class="{ 'opacity-100': tab.id === activeTabId }"
            :title="t('support.close_tab')"
            :aria-label="t('support.close_tab')"
            @click.stop="closeTab(tab.id)"
          >
            <FontAwesomeIcon :icon="faXmark" class="h-3 w-3" aria-hidden="true" />
          </button>
        </div>
      </div>
      <!-- + menu: a terminal or one of the system's web UIs -->
      <div ref="menuRef" class="relative shrink-0">
        <button
          class="ml-1 rounded p-1 text-gray-500 hover:bg-gray-700 hover:text-gray-300"
          :title="t('support.workspace_new_tab')"
          :aria-label="t('support.workspace_new_tab')"
          :aria-expanded="menuOpen"
          @click="menuOpen = !menuOpen"
        >
          <FontAwesomeIcon :icon="faPlus" class="h-3.5 w-3.5" aria-hidden="true" />
        </button>
        <div
          v-if="menuOpen"
          @keydown="onMenuKeydown"
          class="absolute top-full left-0 z-20 mt-1 flex max-h-[70vh] w-80 flex-col overflow-hidden rounded-md border border-gray-200 bg-white text-sm shadow-xl dark:border-gray-700 dark:bg-gray-900"
        >
          <div class="border-b border-gray-200 p-2 dark:border-gray-700">
            <NeTextInput
              v-model="menuSearch"
              is-search
              :placeholder="t('support.workspace_search_services')"
            />
          </div>
          <div class="min-h-0 overflow-y-auto py-1">
            <template v-if="!menuQuery">
              <button
                v-for="session in activeSessions"
                :key="session.id"
                class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-gray-700 dark:text-gray-200"
                :data-menu-index="menuItemIndex(`t:${session.id}`)"
                :class="{
                  'bg-gray-100 dark:bg-gray-800': menuItemIndex(`t:${session.id}`) === menuIndex,
                }"
                @mouseenter="menuIndex = menuItemIndex(`t:${session.id}`)"
                @click="openTerminal(session)"
              >
                <FontAwesomeIcon
                  :icon="faTerminal"
                  class="h-3 w-3 text-green-600"
                  aria-hidden="true"
                />
                {{ terminalLabel(session.id) }}
              </button>
            </template>
            <div v-if="dataLoading" class="px-3 py-2 text-gray-500">
              {{ t('support.workspace_loading') }}
            </div>
            <template v-else>
              <button
                v-for="svc in menuApplianceServices"
                :key="svc.name"
                class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-gray-700 dark:text-gray-200"
                :data-menu-index="menuItemIndex(`s:${svc.name}`)"
                :class="{
                  'bg-gray-100 dark:bg-gray-800': menuItemIndex(`s:${svc.name}`) === menuIndex,
                }"
                @mouseenter="menuIndex = menuItemIndex(`s:${svc.name}`)"
                @click="openService(svc)"
              >
                <FontAwesomeIcon
                  :icon="faGlobe"
                  class="h-3 w-3 text-amber-500"
                  aria-hidden="true"
                />
                {{ svc.label || svc.name }}
              </button>
              <div v-for="mg in menuModules" :key="mg.moduleId">
                <div
                  class="px-3 pt-2 pb-1 text-xs font-medium tracking-wide text-gray-500 uppercase dark:text-gray-400"
                >
                  {{ mg.moduleLabel || mg.moduleId }}
                  <span v-if="mg.nodeId" class="font-normal normal-case"
                    >· {{ nodeName(mg.nodeId) }}</span
                  >
                </div>
                <button
                  v-for="svc in mg.services"
                  :key="svc.name"
                  class="flex w-full items-center gap-2 px-3 py-1.5 pl-5 text-left text-gray-700 dark:text-gray-200"
                  :data-menu-index="menuItemIndex(`s:${svc.name}`)"
                  :class="{
                    'bg-gray-100 dark:bg-gray-800': menuItemIndex(`s:${svc.name}`) === menuIndex,
                  }"
                  @mouseenter="menuIndex = menuItemIndex(`s:${svc.name}`)"
                  @click="openService(svc)"
                >
                  <FontAwesomeIcon
                    :icon="faGlobe"
                    class="h-3 w-3 shrink-0 text-amber-500"
                    aria-hidden="true"
                  />
                  <span class="truncate">{{ serviceDisplayName(svc) }}</span>
                </button>
              </div>
              <template v-if="menuCustomServices.length">
                <div
                  class="px-3 pt-2 pb-1 text-xs font-medium tracking-wide text-gray-500 uppercase dark:text-gray-400"
                >
                  {{ t('support.custom_services') }}
                </div>
                <button
                  v-for="svc in menuCustomServices"
                  :key="svc.name"
                  class="flex w-full items-center gap-2 px-3 py-1.5 pl-5 text-left text-gray-700 dark:text-gray-200"
                  :data-menu-index="menuItemIndex(`s:${svc.name}`)"
                  :class="{
                    'bg-gray-100 dark:bg-gray-800': menuItemIndex(`s:${svc.name}`) === menuIndex,
                  }"
                  @mouseenter="menuIndex = menuItemIndex(`s:${svc.name}`)"
                  @click="openService(svc)"
                >
                  <FontAwesomeIcon
                    :icon="faGlobe"
                    class="h-3 w-3 shrink-0 text-amber-500"
                    aria-hidden="true"
                  />
                  <span class="truncate">{{ svc.label || svc.name }}</span>
                </button>
              </template>
            </template>
          </div>
        </div>
      </div>
      <span v-if="tabsLimitNotice" class="ml-3 shrink-0 text-xs text-amber-400">
        {{ t('support.workspace_tabs_limit') }}
      </span>
    </div>

    <!-- body -->
    <div class="flex min-h-0 flex-1">
      <div class="relative min-w-0 flex-1">
        <!-- the fixed first tab: the system's diagnostics -->
        <div class="absolute inset-0" :class="activeTabId === 0 ? 'z-10' : 'invisible z-0'">
          <SupportDiagnosticsPane :system-id="systemId" :node-name="nodeName" />
        </div>
        <!-- inactive tabs stay mounted and sized: shells and web UIs keep their state -->
        <div
          v-for="tab in tabs"
          :key="tab.id"
          class="absolute inset-0"
          :class="tab.id === activeTabId ? 'z-10' : 'invisible z-0'"
        >
          <SupportTerminalPane
            v-if="tab.spec.kind === 'terminal'"
            :session-id="tab.spec.sessionId"
            :active="tab.id === activeTabId && !minimized"
            @phase="(phase: TerminalPhase) => (tab.phase = phase)"
            @close="closeTab(tab.id)"
          />
          <SupportWebPane
            v-else
            :session-id="tab.spec.sessionId"
            :service="tab.spec.service"
            :path="tab.spec.path"
            :system-name="systemName"
            :credentials="credentialsFor(tab.spec.service)"
            :exact="tab.spec.exact"
            :active="tab.id === activeTabId && !minimized"
            @open-link="openLink"
          />
        </div>
      </div>
      <SupportWorkspacePanel
        v-if="panelOpen"
        v-model:section="panelSection"
        class="w-96 shrink-0 border-l border-gray-200 dark:border-gray-700"
        :system-id="systemId"
        :data="data"
        :loading="dataLoading"
        :terminals="panelTerminals"
        @open="openService"
        @open-external="openServiceExternal"
        @remove="removeService"
        @add-service="addServiceOpen = true"
        @open-terminal="openTerminalById"
        @refresh="refreshData"
        @notes-count="(count: number) => (notesCount = count)"
      />
      <!-- the panel's sections, always at hand like the CTI's side bar -->
      <nav
        class="flex w-12 shrink-0 flex-col items-center gap-1 border-l border-gray-200 bg-white py-2 dark:border-gray-700 dark:bg-gray-900"
        :aria-label="t('support.workspace_panel')"
      >
        <button
          v-for="item in WORKSPACE_PANEL_SECTIONS"
          :key="item.id"
          class="relative flex h-9 w-9 items-center justify-center rounded-md"
          :class="
            panelOpen && panelSection === item.id
              ? 'bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-50'
              : 'text-gray-400 hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-gray-800 dark:hover:text-gray-200'
          "
          :title="t(item.label)"
          :aria-label="
            item.id === 'notes' && notesCount ? `${t(item.label)} (${notesCount})` : t(item.label)
          "
          :aria-pressed="panelOpen && panelSection === item.id"
          @click="togglePanel(item.id)"
        >
          <FontAwesomeIcon :icon="item.icon" class="h-4 w-4" aria-hidden="true" />
          <span
            v-if="panelOpen && panelSection === item.id"
            class="bg-primary-600 dark:bg-primary-400 absolute top-1.5 -right-1.5 h-6 w-0.5 rounded-full"
            aria-hidden="true"
          />
          <span
            v-if="item.id === 'notes' && notesCount"
            class="absolute -top-1 -right-1 min-w-4 rounded-full bg-amber-400 px-1 text-xs leading-4 font-medium text-gray-900"
            aria-hidden="true"
            >{{ notesCount > 99 ? '99+' : notesCount }}</span
          >
        </button>
      </nav>
    </div>

    <!-- moving to another window: terminals restart -->
    <Teleport to="body">
      <NeModal
        :visible="!!pendingMove"
        :title="
          t(pendingMove === 'attach' ? 'support.workspace_attach' : 'support.workspace_detach')
        "
        kind="warning"
        :primary-label="
          t(pendingMove === 'attach' ? 'support.workspace_attach' : 'support.workspace_detach')
        "
        :cancel-label="t('common.cancel')"
        :close-aria-label="t('common.close')"
        @close="pendingMove = null"
        @primary-click="doMove(pendingMove ?? 'detach')"
      >
        <p>{{ t('support.workspace_move_confirm') }}</p>
      </NeModal>
    </Teleport>

    <Teleport to="body">
      <NeModal
        :visible="showCloseConfirm"
        :title="t('support.workspace_close')"
        kind="warning"
        :primary-label="t('common.close')"
        :cancel-label="t('common.cancel')"
        primary-button-kind="danger"
        :close-aria-label="t('common.close')"
        @close="showCloseConfirm = false"
        @primary-click="doClose"
      >
        <p>{{ t('support.workspace_close_confirm', { n: tabs.length }, tabs.length) }}</p>
      </NeModal>
      <SupportAddServiceModal
        :visible="addServiceOpen"
        :sessions="sessions"
        :node-name="nodeName"
        @close="addServiceOpen = false"
        @added="onServiceAdded"
      />
    </Teleport>
  </div>
</template>
