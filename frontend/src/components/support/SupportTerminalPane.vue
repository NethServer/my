<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { API_URL } from '@/lib/config'
import { getTerminalTicket } from '@/lib/support/support'

export type TerminalPhase = 'connecting' | 'connected' | 'error'

const FRAME_DATA = 0
const FRAME_RESIZE = 1
const FRAME_ERROR = 3

const props = defineProps<{
  sessionId: string
  active: boolean
}>()

const emit = defineEmits<{
  phase: [phase: TerminalPhase]
  close: []
}>()

const { t } = useI18n()

const containerRef = ref<HTMLElement | null>(null)
const phase = ref<TerminalPhase>('connecting')
const errorMessage = ref('')

// xterm objects stay out of Vue reactivity: addons break on proxies
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let resizeObserver: ResizeObserver | null = null

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

function setPhase(value: TerminalPhase) {
  phase.value = value
  emit('phase', value)
}

function sendFrame(type: number, payload: Uint8Array) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    const frame = new Uint8Array(1 + payload.length)
    frame[0] = type
    frame.set(payload, 1)
    ws.send(frame.buffer)
  }
}

function sendResize(cols: number, rows: number) {
  sendFrame(FRAME_RESIZE, new TextEncoder().encode(JSON.stringify({ cols, rows })))
}

function fit() {
  // a hidden pane has no size: fitting it would shrink the remote tty
  if (fitAddon && terminal && containerRef.value?.offsetWidth) {
    fitAddon.fit()
  }
}

async function start() {
  const el = containerRef.value
  if (!el) return
  setPhase('connecting')
  errorMessage.value = ''

  // A one-time ticket opens the WebSocket, so the long-lived JWT never ends
  // up in a URL
  let ticket: string
  try {
    ticket = await getTerminalTicket(props.sessionId)
  } catch {
    errorMessage.value = t('support.terminal_connection_error')
    setPhase('error')
    return
  }

  terminal = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: terminalTheme,
  })
  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(new WebLinksAddon())
  terminal.open(el)
  fit()

  terminal.onData((data: string) => sendFrame(FRAME_DATA, new TextEncoder().encode(data)))
  terminal.onResize(({ cols, rows }) => sendResize(cols, rows))

  resizeObserver = new ResizeObserver(() => fit())
  resizeObserver.observe(el)

  const base = API_URL.replace(/^http/, 'ws')
  ws = new WebSocket(
    `${base}/support-sessions/${encodeURIComponent(props.sessionId)}/terminal?ticket=${encodeURIComponent(ticket)}`,
  )
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    setPhase('connected')
    nextTick(() => {
      fit()
      if (props.active) terminal?.focus()
    })
    sendResize(terminal?.cols || 80, terminal?.rows || 24)
  }

  ws.onmessage = (event: MessageEvent) => {
    const data = new Uint8Array(event.data as ArrayBuffer)
    if (data.length < 1) return
    const payload = data.slice(1)
    switch (data[0]) {
      case FRAME_DATA:
        terminal?.write(payload)
        break
      case FRAME_ERROR: {
        const msg = new TextDecoder().decode(payload)
        terminal?.write(`\r\n\x1b[31m${msg}\x1b[0m\r\n`)
        errorMessage.value = msg
        setPhase('error')
        break
      }
    }
  }

  ws.onclose = () => {
    if (phase.value === 'connected') {
      terminal?.write('\r\n\x1b[33mConnection closed.\x1b[0m\r\n')
      errorMessage.value = t('support.terminal_connection_closed')
      setPhase('error')
    }
  }

  ws.onerror = () => {
    errorMessage.value = t('support.terminal_connection_error')
    setPhase('error')
  }
}

function cleanup() {
  resizeObserver?.disconnect()
  resizeObserver = null
  if (ws) {
    ws.onopen = null
    ws.onmessage = null
    ws.onclose = null
    ws.onerror = null
    ws.close()
    ws = null
  }
  if (terminal) {
    try {
      terminal.dispose()
    } catch {
      // xterm may throw while disposing addons
    }
    terminal = null
  }
  fitAddon = null
}

function retry() {
  cleanup()
  nextTick(() => start())
}

watch(
  () => props.active,
  (active) => {
    if (active) {
      nextTick(() => {
        fit()
        terminal?.focus()
      })
    }
  },
)

defineExpose({ fit })

onMounted(() => start())
onBeforeUnmount(() => cleanup())
</script>

<template>
  <div class="absolute inset-0 flex flex-col bg-[#1e1e2e]">
    <div v-if="phase === 'connecting'" class="absolute inset-0 flex items-center justify-center">
      <p class="text-gray-400">{{ t('support.terminal_connecting') }}</p>
    </div>
    <div ref="containerRef" class="min-h-0 flex-1 p-1"></div>
    <div
      v-if="phase === 'error'"
      class="absolute inset-x-0 bottom-0 flex items-center justify-between bg-red-900/90 px-4 py-3"
    >
      <span class="text-sm text-red-200">{{ errorMessage }}</span>
      <!-- light buttons: the library's ones are blue on this red bar -->
      <div class="flex gap-2">
        <button
          class="rounded-md bg-white px-3 py-1.5 text-sm font-medium text-red-900 hover:bg-red-50"
          @click="retry"
        >
          {{ t('support.terminal_retry') }}
        </button>
        <button
          class="rounded-md border border-white/60 px-3 py-1.5 text-sm font-medium text-white hover:bg-white/10"
          @click="emit('close')"
        >
          {{ t('common.close') }}
        </button>
      </div>
    </div>
  </div>
</template>
