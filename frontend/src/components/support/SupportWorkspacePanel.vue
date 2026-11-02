<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { faArrowsRotate } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  getSupportActivity,
  getSupportNotes,
  groupSupportEpisodes,
  type SupportActivity,
  type SupportNote,
} from '@/lib/support/support'
import {
  serviceDisplayName,
  serviceHostLabel,
  WORKSPACE_PANEL_SECTIONS,
  type WorkspaceData,
  type WorkspacePanelSection,
  type WorkspaceService,
} from '@/lib/support/workspace'
import SupportServicesPane, {
  type WorkspaceTerminal,
} from '@/components/support/SupportServicesPane.vue'
import SupportActivityPane from '@/components/support/SupportActivityPane.vue'
import SupportNotesPane from '@/components/support/SupportNotesPane.vue'

const props = defineProps<{
  systemId: string
  systemName: string
  data: WorkspaceData | null
  loading: boolean
  section: WorkspacePanelSection
  terminals: WorkspaceTerminal[]
}>()

const emit = defineEmits<{
  open: [svc: WorkspaceService]
  openExternal: [svc: WorkspaceService]
  remove: [svc: WorkspaceService]
  addService: []
  openTerminal: [sessionId: string]
  refresh: []
  // the section rail shows how many notes the system has
  notesCount: [count: number]
  'update:section': [section: WorkspacePanelSection]
}>()

const { t } = useI18n()

// how often the panel follows who is working on the system: closely while
// the timeline is in front
const POLL_INTERVAL_MS = 30_000
const ACTIVITY_POLL_INTERVAL_MS = 10_000

const activity = ref<SupportActivity | null>(null)
const notes = ref<SupportNote[]>([])
const historyLoading = ref(false)
const historyError = ref('')
const fetchedAt = ref(Date.now())
let timer: ReturnType<typeof setInterval> | null = null

// readable names of the session's services, by name and by proxy host label
// (the web presence knows a service by the host it was reached on)
const serviceNames = ref(new Map<string, string>())
watch(
  () => props.data,
  async (data) => {
    const names = new Map<string, string>()
    const services = [
      ...(data?.modules.flatMap((mg) => mg.services) || []),
      ...(data?.ungrouped || []),
    ]
    for (const svc of services) {
      const display = serviceDisplayName(svc)
      names.set(svc.name, display)
      names.set(await serviceHostLabel(svc.name), display)
    }
    serviceNames.value = names
  },
  { immediate: true },
)

const episodes = computed(() => groupSupportEpisodes(activity.value?.sessions || []))
const pastInterventions = computed(() => episodes.value.filter((ep) => !ep.live).length)

async function loadHistory() {
  historyLoading.value = !activity.value
  try {
    const [act, list] = await Promise.all([
      getSupportActivity(props.systemId),
      getSupportNotes(props.systemId),
    ])
    activity.value = act
    notes.value = list
    historyError.value = ''
  } catch {
    historyError.value = t('support.activity_load_error')
  } finally {
    historyLoading.value = false
    fetchedAt.value = Date.now()
  }
}

function refresh() {
  if (props.section === 'services') {
    emit('refresh')
  } else {
    loadHistory()
  }
}

function schedule() {
  if (timer) clearInterval(timer)
  timer = setInterval(
    loadHistory,
    props.section === 'activity' ? ACTIVITY_POLL_INTERVAL_MS : POLL_INTERVAL_MS,
  )
}

// back on the page or on the timeline: no waiting for the next tick
function onVisible() {
  if (document.visibilityState === 'visible') loadHistory()
}

watch(() => props.systemId, loadHistory)
watch(
  () => props.section,
  (section) => {
    if (section === 'activity') loadHistory()
    schedule()
  },
)
watch(
  () => notes.value.length,
  (count) => emit('notesCount', count),
)
onMounted(() => {
  loadHistory()
  schedule()
  document.addEventListener('visibilitychange', onVisible)
})
onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
  document.removeEventListener('visibilitychange', onVisible)
})
</script>

<template>
  <div class="flex h-full flex-col bg-white dark:bg-gray-900">
    <div
      class="flex h-11 shrink-0 items-center justify-between gap-2 border-b border-gray-200 px-4 dark:border-gray-700"
    >
      <h3 class="min-w-0 truncate text-sm font-semibold text-gray-900 dark:text-gray-100">
        {{ t(WORKSPACE_PANEL_SECTIONS.find((item) => item.id === section)?.label || '') }}
      </h3>
      <div class="flex shrink-0 items-center gap-1">
        <button
          class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-800 dark:hover:text-gray-200"
          :title="t('support.workspace_reload')"
          :disabled="loading"
          @click="refresh"
        >
          <FontAwesomeIcon :icon="faArrowsRotate" class="h-3.5 w-3.5" aria-hidden="true" />
        </button>
      </div>
    </div>
    <!-- the system's history spans its sessions: say how far it goes -->
    <button
      v-if="section !== 'services' && pastInterventions"
      class="shrink-0 border-b border-gray-200 bg-gray-50 px-4 py-2 text-left text-xs text-gray-600 hover:bg-gray-100 dark:border-gray-700 dark:bg-gray-800/60 dark:text-gray-300 dark:hover:bg-gray-800"
      @click="emit('update:section', 'activity')"
    >
      {{ t('support.activity_past_interventions', { n: pastInterventions }, pastInterventions) }}
    </button>
    <SupportServicesPane
      v-if="section === 'services'"
      :data="data"
      :loading="loading"
      :terminals="terminals"
      :system-id="systemId"
      :system-name="systemName"
      @open="(svc) => emit('open', svc)"
      @open-external="(svc) => emit('openExternal', svc)"
      @remove="(svc) => emit('remove', svc)"
      @add="emit('addService')"
      @open-terminal="(id: string) => emit('openTerminal', id)"
    />
    <SupportActivityPane
      v-else-if="section === 'activity'"
      :activity="activity"
      :notes="notes"
      :episodes="episodes"
      :loading="historyLoading"
      :error="historyError"
      :fetched-at="fetchedAt"
      :service-names="serviceNames"
      :node-labels="data?.nodeLabels"
    />
    <SupportNotesPane
      v-else
      :system-id="systemId"
      :notes="notes"
      :episodes="episodes"
      :loading="historyLoading"
      :error="historyError"
      @changed="loadHistory"
    />
  </div>
</template>
