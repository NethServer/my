<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { faArrowsRotate, faTerminal } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeButton,
  NeInlineNotification,
  NeSpinner,
  formatDateTimeNoSeconds,
} from '@nethesis/vue-components'
import {
  getSystemDiagnostics,
  type SessionRef,
  type SystemDiagnostics,
} from '@/lib/support/support'

const props = defineProps<{
  systemId: string
  sessions: SessionRef[]
  terminalLabel: (sessionId: string) => string
  nodeName: (nodeId: string) => string
}>()

const emit = defineEmits<{
  openTerminal: [session: SessionRef]
}>()

const { t, te, locale } = useI18n()

const diagnostics = ref<SystemDiagnostics | null>(null)
const loading = ref(false)
const error = ref('')

const activeSessions = computed(() => props.sessions.filter((s) => s.status === 'active'))
const multiNode = computed(() => (diagnostics.value?.nodes.length || 0) > 1)

// One card per node and plugin, the same plugin of each node side by side
const cards = computed(() => {
  const nodes = diagnostics.value?.nodes || []
  const pluginIds = [
    ...new Set(nodes.flatMap((n) => n.diagnostics?.plugins.map((p) => p.id) || [])),
  ]
  return pluginIds.flatMap((id) =>
    nodes.flatMap((node) => {
      const plugin = node.diagnostics?.plugins.find((p) => p.id === id)
      return plugin ? [{ key: `${node.session_id}:${id}`, node, plugin }] : []
    }),
  )
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    diagnostics.value = await getSystemDiagnostics(props.systemId)
  } catch {
    error.value = t('support.diagnostics_load_error')
  } finally {
    loading.value = false
  }
}

const statusClasses: Record<string, string> = {
  ok: 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300',
  warning: 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300',
  critical: 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300',
  error: 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300',
  timeout: 'bg-gray-200 text-gray-700 dark:bg-gray-700 dark:text-gray-300',
}

function statusClass(status: string | undefined): string {
  return statusClasses[status || ''] || statusClasses.timeout
}

function statusLabel(status: string | undefined): string {
  const key = `support.diagnostics_status_${status}`
  return te(key) ? t(key) : status || '-'
}

// A check or detail name from the appliance, translated when known
function label(name: string): string {
  const key = `support.diagnostics_check_${name}`
  if (te(key)) return t(key)
  const words = name.replace(/[_-]+/g, ' ').trim()
  return words.charAt(0).toUpperCase() + words.slice(1)
}

function formatBytes(bytes: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit++
  }
  return `${value.toFixed(value < 10 && unit > 0 ? 1 : 0)} ${units[unit]}`
}

function formatDuration(seconds: number): string {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  return [d && `${d}d`, (d || h) && `${h}h`, `${m}m`].filter(Boolean).join(' ')
}

// Readable value of a detail, from the unit its key carries
function formatValue(key: string, value: unknown): string {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'boolean')
    return value ? t('support.diagnostics_yes') : t('support.diagnostics_no')
  if (typeof value === 'number') {
    if (key.endsWith('_kb')) return formatBytes(value * 1024)
    if (key.endsWith('_bytes')) return formatBytes(value)
    if (key.endsWith('_pct')) return `${value.toFixed(1)}%`
    if (key === 'total_seconds' || key.endsWith('_seconds')) return formatDuration(value)
    return Number.isInteger(value) ? String(value) : value.toFixed(2)
  }
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

const goodStates = ['active', 'running', 'ok', 'up', 'enabled', 'connected']
const badStates = ['failed', 'stopped', 'inactive', 'down', 'error', 'critical', 'dead']

function stateClass(value: unknown): string {
  const v = String(value ?? '').toLowerCase()
  if (goodStates.includes(v)) return 'text-green-700 dark:text-green-400'
  if (badStates.includes(v)) return 'text-red-700 dark:text-red-400'
  return ''
}

type DetailsView =
  | { kind: 'none' }
  | { kind: 'text'; text: string }
  | { kind: 'chips'; items: string[] }
  | { kind: 'table'; columns: string[]; rows: Record<string, unknown>[] }
  | {
      kind: 'pairs'
      pairs: { key: string; value: string }[]
      lists: { key: string; items: string[] }[]
    }

const isRecord = (v: unknown): v is Record<string, unknown> =>
  !!v && typeof v === 'object' && !Array.isArray(v)

// The details shape varies per plugin: a list of rows becomes a table, a list
// of values chips, an object label/value pairs
function detailsView(details: unknown): DetailsView {
  if (details === null || details === undefined || details === '') return { kind: 'none' }
  if (typeof details !== 'object') return { kind: 'text', text: String(details) }
  if (Array.isArray(details)) {
    if (!details.length) return { kind: 'none' }
    if (details.every(isRecord)) {
      const columns = [...new Set(details.flatMap((row) => Object.keys(row)))]
      return { kind: 'table', columns, rows: details }
    }
    return { kind: 'chips', items: details.map((item) => formatValue('', item)) }
  }
  const pairs: { key: string; value: string }[] = []
  const lists: { key: string; items: string[] }[] = []
  for (const [key, value] of Object.entries(details as Record<string, unknown>)) {
    if (Array.isArray(value) && value.every((item) => !isRecord(item))) {
      lists.push({ key, items: value.map((item) => formatValue('', item)) })
    } else {
      pairs.push({ key, value: formatValue(key, value) })
    }
  }
  return { kind: 'pairs', pairs, lists }
}

onMounted(() => load())

defineExpose({ reload: load })
</script>

<template>
  <div class="absolute inset-0 overflow-y-auto bg-gray-50 dark:bg-gray-950">
    <div class="flex flex-col gap-4 p-4 lg:px-6">
      <!-- header: overall state, nodes, quick terminals, refresh -->
      <div class="flex flex-wrap items-center gap-x-6 gap-y-3">
        <div class="flex items-center gap-3">
          <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100">
            {{ t('support.diagnostics') }}
          </h3>
          <span
            v-if="diagnostics"
            class="rounded-full px-2.5 py-0.5 text-xs font-medium"
            :class="statusClass(diagnostics.overall_status)"
          >
            {{ statusLabel(diagnostics.overall_status) }}
          </span>
        </div>
        <div
          v-for="node in diagnostics?.nodes || []"
          :key="node.session_id"
          class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400"
        >
          <span v-if="multiNode" class="font-medium text-gray-700 dark:text-gray-300">
            {{
              node.node_id ? nodeName(node.node_id) : t('support.diagnostics_node', { node: '-' })
            }}
          </span>
          <span
            v-if="node.diagnostics && multiNode"
            class="rounded-full px-2 py-0.5 font-medium"
            :class="statusClass(node.diagnostics.overall_status)"
          >
            {{ statusLabel(node.diagnostics.overall_status) }}
          </span>
          <span v-if="node.diagnostics">
            {{
              t('support.diagnostics_collected', {
                time: formatDateTimeNoSeconds(new Date(node.diagnostics.collected_at), locale),
                duration: (node.diagnostics.duration_ms / 1000).toFixed(1),
              })
            }}
          </span>
          <span v-else>{{ t('support.diagnostics_not_available') }}</span>
        </div>
        <div class="ml-auto flex flex-wrap items-center gap-2">
          <NeButton
            v-for="session in activeSessions"
            :key="session.id"
            kind="secondary"
            size="sm"
            @click="emit('openTerminal', session)"
          >
            <template #prefix>
              <FontAwesomeIcon :icon="faTerminal" aria-hidden="true" />
            </template>
            {{ terminalLabel(session.id) }}
          </NeButton>
          <NeButton kind="tertiary" size="sm" :disabled="loading" @click="load">
            <template #prefix>
              <FontAwesomeIcon :icon="faArrowsRotate" aria-hidden="true" />
            </template>
            {{ t('support.workspace_reload') }}
          </NeButton>
        </div>
      </div>

      <div v-if="loading && !diagnostics" class="flex justify-center py-10">
        <NeSpinner />
      </div>
      <NeInlineNotification v-else-if="error" kind="error" :title="error" />
      <p v-else-if="diagnostics && !cards.length" class="text-sm text-gray-500 dark:text-gray-400">
        {{ t('support.diagnostics_not_available') }}
      </p>

      <div
        v-if="cards.length"
        class="grid items-start gap-4 lg:grid-cols-2"
        :class="{ '2xl:grid-cols-3': (diagnostics?.nodes.length || 0) > 2 }"
      >
        <div
          v-for="card in cards"
          :key="card.key"
          class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900"
        >
          <div
            class="flex items-center justify-between gap-3 border-b border-gray-200 px-4 py-2.5 dark:border-gray-700"
          >
            <span class="font-medium text-gray-900 dark:text-gray-100">
              {{ card.plugin.name }}
              <span v-if="multiNode" class="font-normal text-gray-500 dark:text-gray-400">
                ·
                {{
                  card.node.node_id
                    ? nodeName(card.node.node_id)
                    : t('support.diagnostics_node', { node: '-' })
                }}
              </span>
            </span>
            <div class="flex items-center gap-2">
              <span v-if="card.plugin.summary" class="text-xs text-gray-500 dark:text-gray-400">
                {{ card.plugin.summary }}
              </span>
              <span
                class="rounded-full px-2 py-0.5 text-xs font-medium"
                :class="statusClass(card.plugin.status)"
              >
                {{ statusLabel(card.plugin.status) }}
              </span>
            </div>
          </div>
          <ul class="divide-y divide-gray-100 dark:divide-gray-800">
            <li v-for="check in card.plugin.checks || []" :key="check.name" class="px-4 py-2">
              <div class="flex items-center gap-3 text-sm">
                <span
                  class="size-2 shrink-0 rounded-full"
                  :class="{
                    'bg-green-500': check.status === 'ok',
                    'bg-amber-500': check.status === 'warning',
                    'bg-red-500': check.status === 'critical' || check.status === 'error',
                    'bg-gray-400': check.status === 'timeout',
                  }"
                  :title="statusLabel(check.status)"
                ></span>
                <span class="text-gray-700 dark:text-gray-300">{{ label(check.name) }}</span>
                <span
                  class="ml-auto min-w-0 truncate text-right font-medium text-gray-900 dark:text-gray-100"
                  :title="check.value"
                >
                  {{ check.value || '-' }}
                </span>
              </div>
              <!-- details, shaped by what the plugin reports -->
              <template v-for="view in [detailsView(check.details)]" :key="view.kind">
                <p
                  v-if="view.kind === 'text'"
                  class="mt-1 pl-5 text-xs break-words text-gray-500 dark:text-gray-400"
                >
                  {{ view.text }}
                </p>
                <div v-else-if="view.kind === 'chips'" class="mt-1.5 flex flex-wrap gap-1 pl-5">
                  <span
                    v-for="item in view.items"
                    :key="item"
                    class="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-700 dark:bg-gray-800 dark:text-gray-300"
                  >
                    {{ item }}
                  </span>
                </div>
                <div v-else-if="view.kind === 'table'" class="mt-1 overflow-x-auto pl-5">
                  <table class="w-full text-xs">
                    <thead>
                      <tr class="text-left text-gray-500 dark:text-gray-400">
                        <th v-for="col in view.columns" :key="col" class="py-0.5 pr-4 font-medium">
                          {{ label(col) }}
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="(row, idx) in view.rows" :key="idx">
                        <td
                          v-for="col in view.columns"
                          :key="col"
                          class="py-0.5 pr-4 text-gray-700 dark:text-gray-300"
                          :class="stateClass(row[col])"
                        >
                          {{ formatValue(col, row[col]) }}
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <div v-else-if="view.kind === 'pairs'" class="mt-1 pl-5 text-xs">
                  <div v-if="view.pairs.length" class="flex flex-wrap gap-x-4 gap-y-0.5">
                    <span v-for="pair in view.pairs" :key="pair.key">
                      <span class="text-gray-500 dark:text-gray-400">{{ label(pair.key) }}</span>
                      <span class="ml-1.5 text-gray-700 dark:text-gray-300">{{ pair.value }}</span>
                    </span>
                  </div>
                  <div
                    v-for="list in view.lists"
                    :key="list.key"
                    class="mt-1.5 flex flex-wrap items-center gap-1"
                  >
                    <span class="mr-1 text-gray-500 dark:text-gray-400">{{ label(list.key) }}</span>
                    <span
                      v-for="item in list.items"
                      :key="item"
                      class="rounded bg-gray-100 px-1.5 py-0.5 text-gray-700 dark:bg-gray-800 dark:text-gray-300"
                    >
                      {{ item }}
                    </span>
                  </div>
                </div>
              </template>
            </li>
          </ul>
        </div>
      </div>
    </div>
  </div>
</template>
