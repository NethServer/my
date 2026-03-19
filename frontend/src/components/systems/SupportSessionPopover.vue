<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faHeadset, faUser } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeBadge, NeSpinner } from '@nethesis/vue-components'
import { ref, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  getSystemActiveSessions,
  getSupportSessionLogs,
  getSupportSessionDiagnostics,
  type SystemSessionGroup,
  type SessionDiagnostics,
} from '@/lib/support/support'
function formatDateWithMonth(date: Date, loc: string): string {
  return date.toLocaleString(loc, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

const props = defineProps<{
  systemId: string
}>()

const { t, locale } = useI18n()

// An operator connection with node context
interface OperatorConnection {
  accessType: string
  nodeId: string | null
}

// An operator grouped across all sessions/nodes
interface OperatorEntry {
  operatorId: string
  operatorName: string
  connections: OperatorConnection[]
}

interface PopoverData {
  group: SystemSessionGroup
  operators: OperatorEntry[]
}

const data = ref<PopoverData | null>(null)
const diagnostics = ref<SessionDiagnostics | null>(null)
const loading = ref(false)
const error = ref(false)
const isOpen = ref(false)
const triggerEl = ref<HTMLButtonElement | null>(null)
const panelStyle = ref({ top: '0px', left: '0px' })

async function fetchData() {
  loading.value = true
  error.value = false
  try {
    const groups = await getSystemActiveSessions(props.systemId)
    if (groups.length === 0) {
      data.value = null
      return
    }

    // Use the first group (one system = one group with potentially multiple session refs)
    const group = groups[0]

    // Fetch logs for ALL session refs and associate node_id
    const operatorMap = new Map<string, OperatorEntry>()

    for (const sessionRef of group.sessions || []) {
      try {
        const logsData = await getSupportSessionLogs(sessionRef.id, 1, 100)
        for (const log of logsData.access_logs || []) {
          // Skip disconnected and ui_proxy logs
          if (log.disconnected_at || log.access_type === 'ui_proxy') continue

          let entry = operatorMap.get(log.operator_id)
          if (!entry) {
            entry = {
              operatorId: log.operator_id,
              operatorName: log.operator_name,
              connections: [],
            }
            operatorMap.set(log.operator_id, entry)
          }
          entry.connections.push({
            accessType: log.access_type,
            nodeId: sessionRef.node_id,
          })
        }
      } catch {
        // ignore log fetch errors for individual sessions
      }
    }

    data.value = {
      group,
      operators: Array.from(operatorMap.values()),
    }

    // Fetch diagnostics from the most recently started session
    if (group.sessions && group.sessions.length > 0) {
      const latestSession = [...group.sessions].sort(
        (a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime(),
      )[0]
      try {
        diagnostics.value = await getSupportSessionDiagnostics(latestSession.id)
      } catch {
        diagnostics.value = null
      }
    }
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}

function toggle() {
  if (isOpen.value) {
    isOpen.value = false
    return
  }
  if (triggerEl.value) {
    const rect = triggerEl.value.getBoundingClientRect()
    panelStyle.value = {
      top: `${rect.top + rect.height / 2}px`,
      left: `${rect.right + 8}px`,
    }
  }
  isOpen.value = true
  fetchData()
}

function close() {
  isOpen.value = false
}

function onClickOutside(e: MouseEvent) {
  if (triggerEl.value?.contains(e.target as Node)) return
  close()
}

document.addEventListener('click', onClickOutside)
onBeforeUnmount(() => {
  document.removeEventListener('click', onClickOutside)
})

function getStatusBadgeKind(status: string) {
  switch (status) {
    case 'active':
      return 'success'
    case 'pending':
      return 'warning'
    default:
      return 'secondary'
  }
}

function formatConnectionBadge(conn: OperatorConnection): string {
  const label = conn.accessType
  if (conn.nodeId) {
    return `${label} (${t('systems.node')} ${conn.nodeId})`
  }
  return label
}

function diagnosticStatusDotClass(status: string): string {
  switch (status) {
    case 'ok':
      return 'bg-green-500'
    case 'warning':
      return 'bg-amber-400'
    case 'critical':
      return 'bg-red-500'
    default:
      return 'bg-gray-400'
  }
}

function diagnosticStatusTextClass(status: string): string {
  switch (status) {
    case 'ok':
      return 'text-green-600 dark:text-green-400'
    case 'warning':
      return 'text-amber-500 dark:text-amber-400'
    case 'critical':
      return 'text-red-600 dark:text-red-400'
    default:
      return 'text-gray-500 dark:text-gray-400'
  }
}
</script>

<template>
  <button
    ref="triggerEl"
    class="cursor-pointer focus:outline-none"
    :title="t('systems.support_session_active')"
    @click.stop="toggle"
  >
    <FontAwesomeIcon
      :icon="faHeadset"
      class="h-4 w-4 text-amber-500 dark:text-amber-400"
      aria-hidden="true"
    />
  </button>
  <Teleport to="body">
    <div
      v-if="isOpen"
      class="fixed z-50 w-[28rem] -translate-y-1/2 rounded-lg border border-gray-200 bg-white p-4 shadow-lg dark:border-gray-700 dark:bg-gray-900"
      :style="panelStyle"
      @click.stop
    >
      <div v-if="loading" class="flex items-center justify-center py-2">
        <NeSpinner />
      </div>
      <div v-else-if="error" class="text-sm text-red-600 dark:text-red-400">
        {{ t('systems.cannot_load_support_session') }}
      </div>
      <div v-else-if="data" class="space-y-2 text-sm">
        <div class="mb-1 flex items-center gap-2 font-medium">
          <FontAwesomeIcon
            :icon="faHeadset"
            class="h-4 w-4 text-amber-500 dark:text-amber-400"
            aria-hidden="true"
          />
          {{ t('systems.support_session') }}
        </div>
        <!-- Session info (single session per system) -->
        <div class="flex items-center justify-between">
          <span class="font-medium">{{ t('support.status') }}</span>
          <NeBadge
            :text="t(`support.status_${data.group.status}`)"
            :kind="getStatusBadgeKind(data.group.status)"
            size="sm"
          />
        </div>
        <div class="flex items-center justify-between">
          <span class="text-gray-500 dark:text-gray-400">{{ t('support.started_at') }}</span>
          <span>{{ formatDateWithMonth(new Date(data.group.started_at), locale) }}</span>
        </div>
        <div class="flex items-center justify-between">
          <span class="text-gray-500 dark:text-gray-400">{{ t('support.expires_at') }}</span>
          <span>{{ formatDateWithMonth(new Date(data.group.expires_at), locale) }}</span>
        </div>
        <!-- Connected operators -->
        <div v-if="data.operators.length > 0">
          <div class="mb-1.5 border-t border-gray-200 pt-2 font-medium dark:border-gray-700">
            {{ t('systems.connected_operators') }}
          </div>
          <div class="space-y-2">
            <div v-for="op in data.operators" :key="op.operatorId">
              <div class="flex items-start justify-between gap-3">
                <div class="flex min-w-[40%] items-center gap-1.5 pt-0.5">
                  <FontAwesomeIcon
                    :icon="faUser"
                    class="h-3 w-3 shrink-0 text-gray-400"
                    aria-hidden="true"
                  />
                  <span class="truncate">{{ op.operatorName }}</span>
                </div>
                <div class="flex flex-col items-end gap-1">
                  <NeBadge
                    v-for="(conn, idx) in op.connections"
                    :key="idx"
                    :text="formatConnectionBadge(conn)"
                    kind="secondary"
                    size="sm"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
        <!-- Diagnostics -->
        <div
          v-if="diagnostics?.diagnostics"
          class="border-t border-gray-200 pt-2 dark:border-gray-700"
        >
          <div class="mb-1.5 flex items-center gap-1.5 font-medium">
            <span>{{ t('support.diagnostics') }}</span>
            <span
              class="inline-block h-2 w-2 rounded-full"
              :class="diagnosticStatusDotClass(diagnostics.diagnostics.overall_status)"
            />
          </div>
          <div class="space-y-1">
            <div
              v-for="plugin in diagnostics.diagnostics.plugins"
              :key="plugin.id"
              class="flex items-center justify-between"
            >
              <span class="text-gray-500 dark:text-gray-400">{{ plugin.name }}</span>
              <span class="text-xs font-medium" :class="diagnosticStatusTextClass(plugin.status)">
                {{ plugin.summary || plugin.status }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
