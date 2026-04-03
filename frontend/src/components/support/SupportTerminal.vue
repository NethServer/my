<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { NeButton, NeModal } from '@nethesis/vue-components'
import { faXmark, faPlus, faServer } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { API_URL } from '@/lib/config'
import { getTerminalTicket, type SessionRef } from '@/lib/support/support'

const FRAME_DATA = 0
const FRAME_RESIZE = 1
const FRAME_ERROR = 3

interface TerminalTab {
  id: number
  label: string
  sessionId: string
  nodeId: string | null
  phase: 'connecting' | 'connected' | 'error'
  errorMessage: string
  terminal: Terminal | null
  fitAddon: FitAddon | null
  ws: WebSocket | null
  resizeObserver: ResizeObserver | null
  containerRef: HTMLElement | null
}

const props = defineProps<{
  sessions: SessionRef[]
  systemName: string
}>()

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()

let nextTabId = 1
const tabs = ref<TerminalTab[]>([])
const activeTabId = ref<number>(0)
const showCloseConfirm = ref(false)
const showNodePicker = ref(false)

const isMultiNode = computed(() => {
  return props.sessions.length > 1 && props.sessions.some((s) => s.node_id)
})

const activeSessions = computed(() => {
  return props.sessions.filter((s) => s.status === 'active')
})

const terminalTheme = {
  background: '#1e1e2e',
  foreground: '#cdd6f4',
  cursor: '#f5e0dc',
  selectionBackground: '#585b70',
  black: '#45475a',
  red: '#f38ba8',
  green: '#a6e3a1',
  yellow: '#f9e2af',
  blue: '#89b4fa',
  magenta: '#f5c2e7',
  cyan: '#94e2d5',
  white: '#bac2de',
  brightBlack: '#585b70',
  brightRed: '#f38ba8',
  brightGreen: '#a6e3a1',
  brightYellow: '#f9e2af',
  brightBlue: '#89b4fa',
  brightMagenta: '#f5c2e7',
  brightCyan: '#94e2d5',
  brightWhite: '#a6adc8',
}

function getWebSocketUrl(sessionId: string, ticket: string): string {
  const base = API_URL.replace(/^http/, 'ws')
  return `${base}/support-sessions/${sessionId}/terminal?ticket=${encodeURIComponent(ticket)}`
}

function createTabForSession(session: SessionRef): TerminalTab {
  const id = nextTabId++
  const nodeLabel = session.node_id ? `Node ${session.node_id}` : ''
  const tab: TerminalTab = {
    id,
    label: isMultiNode.value ? `${nodeLabel} #${id}` : `${id}`,
    sessionId: session.id,
    nodeId: session.node_id,
    phase: 'connecting',
    errorMessage: '',
    terminal: null,
    fitAddon: null,
    ws: null,
    resizeObserver: null,
    containerRef: null,
  }
  tabs.value.push(tab)
  activeTabId.value = id
  nextTick(() => startTab(id))
  return tab
}

function handleNewTab() {
  if (isMultiNode.value && activeSessions.value.length > 1) {
    showNodePicker.value = true
  } else {
    createTabForSession(activeSessions.value[0])
  }
}

function pickNode(session: SessionRef) {
  showNodePicker.value = false
  createTabForSession(session)
}

function cancelNodePicker() {
  showNodePicker.value = false
  // If no tabs were opened yet, close the terminal entirely
  if (tabs.value.length === 0) {
    emit('close')
  }
}

function getTab(id: number): TerminalTab | undefined {
  return tabs.value.find((tab) => tab.id === id)
}

async function startTab(tabId: number) {
  const tab = getTab(tabId)
  if (!tab) return

  const el = document.getElementById(`terminal-container-${tab.id}`)
  if (!el) return
  tab.containerRef = el

  // Obtain a one-time ticket before opening the WebSocket,
  // so the long-lived JWT is never exposed in the URL
  let ticket: string
  try {
    ticket = await getTerminalTicket(tab.sessionId)
  } catch {
    tab.phase = 'error'
    tab.errorMessage = t('support.terminal_connection_error')
    return
  }

  tab.terminal = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: terminalTheme,
  })

  tab.fitAddon = new FitAddon()
  tab.terminal.loadAddon(tab.fitAddon)
  tab.terminal.loadAddon(new WebLinksAddon())

  tab.terminal.open(el)
  tab.fitAddon.fit()

  tab.terminal.onData((data: string) => {
    if (tab.ws && tab.ws.readyState === WebSocket.OPEN) {
      const encoded = new TextEncoder().encode(data)
      const frame = new Uint8Array(1 + encoded.length)
      frame[0] = FRAME_DATA
      frame.set(encoded, 1)
      tab.ws.send(frame.buffer)
    }
  })

  tab.resizeObserver = new ResizeObserver(() => {
    if (tab.fitAddon && tab.terminal) {
      tab.fitAddon.fit()
    }
  })
  tab.resizeObserver.observe(el)

  tab.terminal.onResize(({ cols, rows }) => {
    if (tab.ws && tab.ws.readyState === WebSocket.OPEN) {
      const payload = JSON.stringify({ cols, rows })
      const encoded = new TextEncoder().encode(payload)
      const frame = new Uint8Array(1 + encoded.length)
      frame[0] = FRAME_RESIZE
      frame.set(encoded, 1)
      tab.ws.send(frame.buffer)
    }
  })

  const url = getWebSocketUrl(tab.sessionId, ticket)
  tab.ws = new WebSocket(url)
  tab.ws.binaryType = 'arraybuffer'

  tab.ws.onopen = () => {
    tab.phase = 'connected'
    nextTick(() => {
      tab.fitAddon?.fit()
      if (tab.id === activeTabId.value) {
        tab.terminal?.focus()
      }
    })

    const cols = tab.terminal?.cols || 80
    const rows = tab.terminal?.rows || 24
    const payload = JSON.stringify({ cols, rows })
    const encoded = new TextEncoder().encode(payload)
    const frame = new Uint8Array(1 + encoded.length)
    frame[0] = FRAME_RESIZE
    frame.set(encoded, 1)
    tab.ws!.send(frame.buffer)
  }

  tab.ws.onmessage = (event: MessageEvent) => {
    const data = new Uint8Array(event.data as ArrayBuffer)
    if (data.length < 1) return

    const frameType = data[0]
    const payload = data.slice(1)

    switch (frameType) {
      case FRAME_DATA:
        tab.terminal?.write(payload)
        break
      case FRAME_ERROR: {
        const msg = new TextDecoder().decode(payload)
        tab.terminal?.write(`\r\n\x1b[31m${msg}\x1b[0m\r\n`)
        tab.errorMessage = msg
        tab.phase = 'error'
        break
      }
    }
  }

  tab.ws.onclose = () => {
    if (tab.phase === 'connected') {
      tab.terminal?.write('\r\n\x1b[33mConnection closed.\x1b[0m\r\n')
      tab.phase = 'error'
      tab.errorMessage = t('support.terminal_connection_closed')
    }
  }

  tab.ws.onerror = () => {
    tab.phase = 'error'
    tab.errorMessage = t('support.terminal_connection_error')
  }
}

function cleanupTab(tab: TerminalTab) {
  if (tab.resizeObserver) {
    tab.resizeObserver.disconnect()
    tab.resizeObserver = null
  }
  if (tab.ws) {
    tab.ws.onopen = null
    tab.ws.onmessage = null
    tab.ws.onclose = null
    tab.ws.onerror = null
    tab.ws.close()
    tab.ws = null
  }
  if (tab.terminal) {
    try {
      tab.terminal.dispose()
    } catch {
      // xterm may throw if addons were loaded on a reactive proxy
    }
    tab.terminal = null
  }
  tab.fitAddon = null
}

function closeTab(tabId: number) {
  const idx = tabs.value.findIndex((t) => t.id === tabId)
  if (idx === -1) return

  cleanupTab(tabs.value[idx])
  tabs.value.splice(idx, 1)

  if (tabs.value.length === 0) {
    emit('close')
    return
  }

  if (activeTabId.value === tabId) {
    const newIdx = Math.min(idx, tabs.value.length - 1)
    activeTabId.value = tabs.value[newIdx].id
  }
}

function switchTab(tabId: number) {
  activeTabId.value = tabId
  nextTick(() => {
    const tab = getTab(tabId)
    if (tab?.fitAddon && tab.terminal) {
      tab.fitAddon.fit()
      tab.terminal.focus()
    }
  })
}

function retryTab(tabId: number) {
  const tab = getTab(tabId)
  if (!tab) return
  cleanupTab(tab)
  tab.phase = 'connecting'
  tab.errorMessage = ''
  nextTick(() => startTab(tab.id))
}

function handleCloseAll() {
  if (tabs.value.some((tab) => tab.phase === 'connected')) {
    showCloseConfirm.value = true
    return
  }
  doCloseAll()
}

function doCloseAll() {
  showCloseConfirm.value = false
  for (const tab of tabs.value) {
    cleanupTab(tab)
  }
  tabs.value = []
  emit('close')
}

watch(
  () => props.sessions,
  () => {
    for (const tab of tabs.value) {
      cleanupTab(tab)
    }
    tabs.value = []
    nextTabId = 1
    // Auto-open first session or show picker
    if (isMultiNode.value && activeSessions.value.length > 1) {
      showNodePicker.value = true
    } else if (activeSessions.value.length > 0) {
      createTabForSession(activeSessions.value[0])
    }
  },
)

onMounted(() => {
  if (isMultiNode.value && activeSessions.value.length > 1) {
    showNodePicker.value = true
  } else if (activeSessions.value.length > 0) {
    createTabForSession(activeSessions.value[0])
  }
})

onBeforeUnmount(() => {
  for (const tab of tabs.value) {
    cleanupTab(tab)
  }
})
</script>

<template>
  <div class="fixed inset-0 z-50 flex flex-col bg-gray-900">
    <!-- Header bar -->
    <div class="flex items-center justify-between bg-gray-800 px-4 py-2">
      <div class="flex items-center gap-3">
        <span class="text-sm font-medium text-gray-200">
          {{ t('support.terminal') }} &mdash; {{ systemName }}
        </span>
      </div>

      <button
        class="rounded p-1 text-gray-400 hover:bg-gray-700 hover:text-gray-200"
        :title="t('common.close')"
        @click="handleCloseAll"
      >
        <FontAwesomeIcon :icon="faXmark" class="h-5 w-5" aria-hidden="true" />
      </button>
    </div>

    <!-- Tab bar -->
    <div class="flex items-center gap-0 border-b border-gray-700 bg-gray-800 px-2">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        class="group relative flex items-center gap-1.5 px-3 py-1.5 text-xs transition-colors"
        :class="
          tab.id === activeTabId
            ? 'rounded-t bg-gray-900 text-gray-100'
            : 'text-gray-400 hover:bg-gray-700/50 hover:text-gray-200'
        "
        @click="switchTab(tab.id)"
      >
        <span
          class="h-1.5 w-1.5 rounded-full"
          :class="{
            'bg-green-400': tab.phase === 'connected',
            'animate-pulse bg-yellow-400': tab.phase === 'connecting',
            'bg-red-400': tab.phase === 'error',
          }"
        ></span>
        <span>{{ t('support.terminal') }} {{ tab.label }}</span>
        <button
          v-if="tabs.length > 1"
          class="ml-1 rounded p-0.5 opacity-0 group-hover:opacity-100 hover:bg-gray-600"
          @click.stop="closeTab(tab.id)"
        >
          <FontAwesomeIcon :icon="faXmark" class="h-3 w-3" aria-hidden="true" />
        </button>
      </button>
      <button
        class="ml-1 rounded p-1 text-gray-500 hover:bg-gray-700 hover:text-gray-300"
        :title="t('support.terminal_new_tab')"
        @click="handleNewTab"
      >
        <FontAwesomeIcon :icon="faPlus" class="h-3.5 w-3.5" aria-hidden="true" />
      </button>
    </div>

    <!-- Tab content -->
    <div class="relative flex-1">
      <div
        v-for="tab in tabs"
        :key="tab.id"
        class="absolute inset-0 flex flex-col"
        :class="tab.id === activeTabId ? 'z-10' : 'invisible z-0'"
      >
        <!-- Connecting phase -->
        <div v-if="tab.phase === 'connecting'" class="flex flex-1 items-center justify-center">
          <p class="text-gray-400">{{ t('support.terminal_connecting') }}</p>
        </div>

        <!-- Terminal container -->
        <div
          v-show="tab.phase === 'connected' || tab.phase === 'error'"
          :id="`terminal-container-${tab.id}`"
          class="flex-1 p-1"
        ></div>

        <!-- Error overlay -->
        <div
          v-if="tab.phase === 'error'"
          class="absolute inset-x-0 bottom-0 flex items-center justify-between bg-red-900/90 px-4 py-3"
        >
          <span class="text-sm text-red-200">{{ tab.errorMessage }}</span>
          <div class="flex gap-2">
            <NeButton kind="secondary" size="sm" @click="retryTab(tab.id)">
              {{ t('support.terminal_retry') }}
            </NeButton>
            <NeButton kind="secondary" size="sm" @click="closeTab(tab.id)">
              {{ t('common.close') }}
            </NeButton>
          </div>
        </div>
      </div>
    </div>

    <!-- Node picker for multi-node clusters -->
    <Teleport to="body">
      <NeModal
        :visible="showNodePicker"
        :title="t('support.terminal_select_node')"
        kind="info"
        :cancel-label="t('common.cancel')"
        :close-aria-label="t('common.close')"
        class="hide-primary-button"
        @close="cancelNodePicker"
      >
        <p class="mb-4 text-sm text-gray-300">
          {{ t('support.terminal_select_node_description') }}
        </p>
        <div class="flex flex-col gap-2">
          <NeButton
            v-for="session in activeSessions"
            :key="session.id"
            kind="secondary"
            size="lg"
            class="w-full justify-center"
            @click="pickNode(session)"
          >
            <template #prefix>
              <FontAwesomeIcon :icon="faServer" aria-hidden="true" />
            </template>
            Node {{ session.node_id }}
          </NeButton>
        </div>
      </NeModal>
    </Teleport>

    <!-- Close all confirmation modal -->
    <Teleport to="body">
      <NeModal
        :visible="showCloseConfirm"
        :title="t('support.terminal_close_all')"
        kind="warning"
        :primary-label="t('support.terminal_close_all')"
        :cancel-label="t('common.cancel')"
        primary-button-kind="danger"
        :close-aria-label="t('common.close')"
        @close="showCloseConfirm = false"
        @primary-click="doCloseAll"
      >
        <p>{{ t('support.terminal_close_all_confirm', { count: tabs.length }) }}</p>
      </NeModal>
    </Teleport>
  </div>
</template>

<style>
/* NeModal always renders the primary button (no v-if on primaryLabel).
   Hide it in the node-picker modal where selection happens via node buttons. */
.hide-primary-button button[type='submit'] {
  display: none;
}
</style>
