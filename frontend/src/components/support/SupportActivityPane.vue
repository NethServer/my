<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  faChevronDown,
  faChevronRight,
  faCirclePlay,
  faCircleStop,
  faClockRotateLeft,
  faEye,
  faGlobe,
  faKey,
  faRightToBracket,
  faMinus,
  faNoteSticky,
  faPlus,
  faTerminal,
  faUser,
  type IconDefinition,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeInlineNotification, NeSpinner } from '@nethesis/vue-components'
import {
  type SupportActivity,
  type SupportActivityLog,
  type SupportEpisode,
  type SupportNote,
} from '@/lib/support/support'
import { nodeDisplayName } from '@/lib/support/workspace'

const props = defineProps<{
  activity: SupportActivity | null
  notes: SupportNote[]
  episodes: SupportEpisode[]
  loading: boolean
  error: string
  // when the data was fetched: "lately" is measured from it
  fetchedAt: number
  // readable service names, by service name and by proxy host label
  serviceNames: Map<string, string>
  nodeLabels?: Record<string, string>
}>()

const { t, locale } = useI18n()

// an operator active within this window counts as working on the system
const RECENT_WINDOW_MS = 30 * 60 * 1000
// accesses that stay open until the operator disconnects
const LIVE_ACCESS_TYPES = ['web_terminal', 'ssh']

interface TimelineItem {
  id: string
  at: Date
  icon: IconDefinition
  tone: 'neutral' | 'start' | 'end' | 'note'
  text: string
  detail?: string
}

interface PresenceActivity {
  kind: 'terminal' | 'web' | 'recent'
  text: string
}

interface Presence {
  operatorId: string
  name: string
  activities: PresenceActivity[]
}

// an operator card lists this many activities before folding the rest
const PRESENCE_FOLD = 3

function addActivity(entry: Presence, activity: PresenceActivity) {
  if (!entry.activities.some((a) => a.kind === activity.kind && a.text === activity.text)) {
    entry.activities.push(activity)
  }
}

const activity = computed(() => props.activity)
const now = computed(() => props.fetchedAt)

const sessionsById = computed(() => new Map((activity.value?.sessions || []).map((s) => [s.id, s])))
const multiNode = computed(
  () => new Set((activity.value?.sessions || []).map((s) => s.node_id || '')).size > 1,
)

function nodeOf(sessionId: string): string {
  const nodeId = sessionsById.value.get(sessionId)?.node_id
  return multiNode.value && nodeId ? nodeDisplayName(t, nodeId, props.nodeLabels) : ''
}

function operatorName(log: SupportActivityLog): string {
  return log.operator_name || t('support.notes_unknown_author')
}

function metadataOf(log: SupportActivityLog): Record<string, string> {
  if (!log.metadata) return {}
  try {
    const parsed = JSON.parse(log.metadata)
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function formatDuration(ms: number): string {
  const minutes = Math.max(1, Math.round(ms / 60000))
  if (minutes < 60) return t('support.activity_minutes', { n: minutes })
  return t('support.activity_hours', { h: Math.floor(minutes / 60), m: minutes % 60 })
}

function isLiveSession(sessionId: string) {
  const status = sessionsById.value.get(sessionId)?.status
  return status === 'active' || status === 'pending'
}

function isOpen(log: SupportActivityLog) {
  return (
    LIVE_ACCESS_TYPES.includes(log.access_type) &&
    !log.disconnected_at &&
    isLiveSession(log.session_id)
  )
}

// Who is on the system now (an open terminal) and who worked on it lately
const presence = computed(() => {
  const connected = new Map<string, Presence>()
  const recent = new Map<string, Presence>()
  for (const log of activity.value?.access_logs || []) {
    if (!isOpen(log)) continue
    const entry = connected.get(log.operator_id) || {
      operatorId: log.operator_id,
      name: operatorName(log),
      activities: [],
    }
    const node = nodeOf(log.session_id)
    addActivity(entry, {
      kind: 'terminal',
      text: node ? `${t('support.activity_terminal')} · ${node}` : t('support.activity_terminal'),
    })
    connected.set(log.operator_id, entry)
  }
  // web interfaces: every page request refreshes the operator's mark, until
  // the operator closes that interface (another tab still open marks it again)
  const closedAt = new Map<string, number>()
  for (const log of activity.value?.access_logs || []) {
    if (log.access_type !== 'ui_close') continue
    const key = `${log.operator_id}|${serviceLabel(metadataOf(log).service || '')}`
    const at = new Date(log.connected_at).getTime()
    if (at > (closedAt.get(key) || 0)) closedAt.set(key, at)
  }
  for (const p of activity.value?.web_presence || []) {
    const closed = closedAt.get(`${p.operator_id}|${serviceLabel(p.service)}`)
    if (closed && closed >= new Date(p.last_seen).getTime()) continue
    const entry = connected.get(p.operator_id) || {
      operatorId: p.operator_id,
      name: p.operator_name || t('support.notes_unknown_author'),
      activities: [],
    }
    addActivity(entry, { kind: 'web', text: serviceLabel(p.service) })
    connected.set(p.operator_id, entry)
  }
  for (const log of activity.value?.access_logs || []) {
    if (connected.has(log.operator_id) || recent.has(log.operator_id)) continue
    const at = new Date(log.disconnected_at || log.connected_at).getTime()
    if (now.value - at > RECENT_WINDOW_MS) continue
    recent.set(log.operator_id, {
      operatorId: log.operator_id,
      name: operatorName(log),
      activities: [
        { kind: 'recent', text: t('support.activity_last_seen', { time: timeOf(new Date(at)) }) },
      ],
    })
  }
  return { connected: [...connected.values()], recent: [...recent.values()] }
})

// the appliance UIs read better by product name, the others by their address;
// a service the session no longer has keeps its name (a host label without
// its hash suffix)
function serviceLabel(service: string) {
  const name = service.replace(/-[0-9a-f]{8}$/, '')
  if (name === 'cluster-admin') return t('support.cluster_admin')
  if (name === 'nethsecurity-ui') return 'NethSecurity'
  return props.serviceNames.get(service) || props.serviceNames.get(name) || name
}

// a log row is one event, a terminal two: opened, and closed when it ended
function logItems(log: SupportActivityLog): TimelineItem[] {
  const opened = logItem(log)
  if (!LIVE_ACCESS_TYPES.includes(log.access_type) || !log.disconnected_at) return [opened]
  const end = new Date(log.disconnected_at)
  const node = nodeOf(log.session_id)
  return [
    opened,
    {
      id: `${log.id}-end`,
      at: end,
      icon: faTerminal,
      tone: 'end',
      text: t('support.activity_closed_terminal', { name: operatorName(log) }),
      detail:
        [node, formatDuration(end.getTime() - opened.at.getTime())].filter(Boolean).join(' · ') ||
        undefined,
    },
  ]
}

function logItem(log: SupportActivityLog): TimelineItem {
  const name = operatorName(log)
  const meta = metadataOf(log)
  const node = nodeOf(log.session_id)
  const at = new Date(log.connected_at)
  const base = { id: log.id, at, tone: 'neutral' as const, detail: node || undefined }
  switch (log.access_type) {
    case 'web_terminal':
    case 'ssh': {
      // the end of a terminal is an event of its own, see logItems
      const duration = isOpen(log) ? t('support.activity_in_progress') : ''
      return {
        ...base,
        icon: faTerminal,
        text: t('support.activity_opened_terminal', { name }),
        detail: [node, duration].filter(Boolean).join(' · ') || undefined,
      }
    }
    case 'autofill':
      return {
        ...base,
        icon: faKey,
        text: t('support.activity_autofill', { name, service: serviceLabel(meta.service || '') }),
      }
    case 'ui_close':
      return {
        ...base,
        icon: faGlobe,
        tone: 'end',
        text: t('support.activity_closed_service', {
          name,
          service: serviceLabel(meta.service || ''),
        }),
      }
    case 'view':
      return { ...base, icon: faEye, text: t('support.activity_viewed_credentials', { name }) }
    case 'ui_proxy':
      return {
        ...base,
        icon: faGlobe,
        text: t('support.activity_opened_service', {
          name,
          service: serviceLabel(meta.service || ''),
        }),
      }
    case 'cluster_login':
      return {
        ...base,
        icon: faRightToBracket,
        text: t('support.activity_cluster_login', { name, service: meta.service || '' }),
      }
    case 'session_extend':
      return {
        ...base,
        icon: faClockRotateLeft,
        text: t('support.activity_extended', { name, hours: meta.hours || '?' }),
      }
    case 'session_close':
      return {
        ...base,
        icon: faCircleStop,
        tone: 'end',
        text: t('support.activity_closed_by', { name }),
      }
    case 'service_add':
      return {
        ...base,
        icon: faPlus,
        text: t('support.activity_service_added', { name, service: meta.service || '' }),
      }
    case 'service_remove':
      return {
        ...base,
        icon: faMinus,
        text: t('support.activity_service_removed', { name, service: meta.service || '' }),
      }
    default:
      return { ...base, icon: faUser, text: `${name} · ${log.access_type}` }
  }
}

// operator cards opened past their first activities
const unfolded = ref(new Set<string>())

function toggleFold(operatorId: string) {
  if (unfolded.value.has(operatorId)) {
    unfolded.value.delete(operatorId)
  } else {
    unfolded.value.add(operatorId)
  }
}

function visibleActivities(op: Presence) {
  return unfolded.value.has(op.operatorId) ? op.activities : op.activities.slice(0, PRESENCE_FOLD)
}

const timeline = computed(() => {
  const items: (TimelineItem & { sessionId: string })[] = []
  for (const s of activity.value?.sessions || []) {
    const node = multiNode.value && s.node_id ? nodeDisplayName(t, s.node_id, props.nodeLabels) : ''
    items.push({
      id: `start-${s.id}`,
      sessionId: s.id,
      at: new Date(s.started_at),
      icon: faCirclePlay,
      tone: 'start',
      text: t('support.activity_session_started'),
      detail: node || undefined,
    })
    if (s.closed_at || s.status === 'expired') {
      const reason = s.status === 'expired' ? 'timeout' : s.closed_by || ''
      items.push({
        id: `end-${s.id}`,
        sessionId: s.id,
        at: new Date(s.closed_at || s.expires_at),
        icon: faCircleStop,
        tone: 'end',
        text: ['client', 'operator', 'timeout', 'system'].includes(reason)
          ? t(`support.activity_session_ended_${reason}`)
          : t('support.activity_session_ended'),
        detail: node || undefined,
      })
    }
  }
  for (const log of activity.value?.access_logs || []) {
    for (const item of logItems(log)) items.push({ ...item, sessionId: log.session_id })
  }
  for (const note of props.notes) {
    items.push({
      id: `note-${note.id}`,
      sessionId: note.session_id,
      at: new Date(note.created_at),
      icon: faNoteSticky,
      tone: 'note',
      text: t('support.activity_note_added', {
        name: note.author_name || t('support.notes_unknown_author'),
      }),
      detail: note.body.length > 80 ? `${note.body.slice(0, 80)}…` : note.body,
    })
  }
  items.sort((a, b) => b.at.getTime() - a.at.getTime())

  // one block per intervention, the current one first
  const episodeOf = new Map<string, string>()
  for (const ep of props.episodes) for (const id of ep.sessionIds) episodeOf.set(id, ep.id)
  return props.episodes.map((ep) => {
    const epItems = items.filter((item) => episodeOf.get(item.sessionId) === ep.id)
    const operators = new Set(
      (activity.value?.access_logs || [])
        .filter((log) => episodeOf.get(log.session_id) === ep.id)
        .map((log) => operatorName(log)),
    )
    return {
      episode: ep,
      items: epItems,
      operators: [...operators],
      notes: props.notes.filter((n) => episodeOf.get(n.session_id) === ep.id).length,
    }
  })
})

// the current intervention is open, past ones on request
const expanded = ref(new Set<string>())
watch(
  () => props.episodes.map((ep) => ep.id).join(','),
  () => {
    const first = props.episodes[0]
    if (first && !expanded.value.size) expanded.value.add(first.id)
  },
  { immediate: true },
)

function toggle(id: string) {
  if (expanded.value.has(id)) {
    expanded.value.delete(id)
  } else {
    expanded.value.add(id)
  }
}

function dateOf(date: Date) {
  return date.toLocaleDateString(locale.value, { day: 'numeric', month: 'short', year: 'numeric' })
}

function episodeTitle(ep: SupportEpisode) {
  return ep.live
    ? t('support.activity_current_intervention')
    : t('support.activity_intervention_of', { date: dateOf(ep.startedAt) })
}

// a timestamp shows its day only when it is not today
function stampOf(date: Date) {
  return date.toDateString() === new Date().toDateString()
    ? timeOf(date)
    : `${date.toLocaleDateString(locale.value, { day: 'numeric', month: 'short' })}, ${timeOf(date)}`
}

function timeOf(date: Date) {
  return date.toLocaleTimeString(locale.value, { hour: '2-digit', minute: '2-digit' })
}

function toneClass(tone: TimelineItem['tone']) {
  switch (tone) {
    case 'start':
      return 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400'
    case 'end':
      return 'bg-gray-200 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
    case 'note':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-400'
    default:
      return 'bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-400'
  }
}
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4">
    <div v-if="loading" class="flex justify-center py-4">
      <NeSpinner />
    </div>
    <NeInlineNotification v-else-if="error && !activity" kind="error" :title="error" />
    <template v-else-if="activity">
      <!-- who is on the system now and who worked on it lately -->
      <section class="shrink-0 rounded-md border border-gray-200 p-3 dark:border-gray-700">
        <h4 class="mb-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
          {{ t('support.activity_connected_now') }}
        </h4>
        <p
          v-if="!presence.connected.length && !presence.recent.length"
          class="text-sm text-gray-500 dark:text-gray-400"
        >
          {{ t('support.activity_nobody') }}
        </p>
        <ul class="flex flex-col gap-3">
          <li
            v-for="op in [...presence.connected, ...presence.recent]"
            :key="op.operatorId"
            class="text-sm"
          >
            <div class="flex items-center gap-2">
              <span v-if="presence.connected.includes(op)" class="relative flex h-2 w-2 shrink-0">
                <span
                  class="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75"
                />
                <span class="relative inline-flex h-2 w-2 rounded-full bg-green-500" />
              </span>
              <span v-else class="inline-flex h-2 w-2 shrink-0 rounded-full bg-amber-400" />
              <span class="truncate font-medium text-gray-900 dark:text-gray-100">{{
                op.name
              }}</span>
            </div>
            <!-- one line per terminal or web interface -->
            <ul class="mt-1 ml-4 flex flex-col gap-1">
              <li
                v-for="activity in visibleActivities(op)"
                :key="activity.kind + activity.text"
                class="flex items-center gap-2 text-xs text-gray-600 dark:text-gray-300"
              >
                <FontAwesomeIcon
                  :icon="
                    activity.kind === 'terminal'
                      ? faTerminal
                      : activity.kind === 'web'
                        ? faGlobe
                        : faClockRotateLeft
                  "
                  class="h-3 w-3 shrink-0 text-gray-400"
                  aria-hidden="true"
                />
                <span class="min-w-0 truncate" :title="activity.text">{{ activity.text }}</span>
              </li>
            </ul>
            <button
              v-if="op.activities.length > PRESENCE_FOLD"
              class="text-primary-600 dark:text-primary-400 mt-1 ml-9 text-xs hover:underline"
              @click="toggleFold(op.operatorId)"
            >
              {{
                unfolded.has(op.operatorId)
                  ? t('support.activity_show_less')
                  : t('support.activity_show_more', { n: op.activities.length - PRESENCE_FOLD })
              }}
            </button>
          </li>
        </ul>
      </section>

      <!-- every intervention on the system, past ones too -->
      <section
        v-for="group in timeline"
        :key="group.episode.id"
        class="shrink-0 overflow-hidden rounded-md border border-gray-200 dark:border-gray-700"
      >
        <button
          class="flex w-full items-start gap-2 px-3 py-2.5 text-left hover:bg-gray-50 dark:hover:bg-gray-800"
          @click="toggle(group.episode.id)"
        >
          <FontAwesomeIcon
            :icon="expanded.has(group.episode.id) ? faChevronDown : faChevronRight"
            class="mt-1 h-3 w-3 shrink-0 text-gray-500 dark:text-gray-400"
          />
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="truncate text-sm font-semibold text-gray-900 dark:text-gray-100">
                {{ episodeTitle(group.episode) }}
              </span>
              <span
                v-if="group.episode.live"
                class="inline-flex h-2 w-2 shrink-0 rounded-full bg-amber-400"
              />
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              {{ stampOf(group.episode.startedAt) }}
              <template v-if="group.episode.endedAt">
                → {{ stampOf(group.episode.endedAt) }}</template
              >
              <template v-if="group.notes">
                · {{ t('support.activity_notes_count', { n: group.notes }, group.notes) }}</template
              >
            </div>
            <div
              v-if="group.operators.length"
              class="truncate text-xs text-gray-500 dark:text-gray-400"
            >
              {{ group.operators.join(', ') }}
            </div>
          </div>
        </button>
        <ol
          v-if="expanded.has(group.episode.id)"
          class="relative mx-3 mt-1 mb-3 ml-6 border-l border-gray-200 dark:border-gray-700"
        >
          <li v-for="item in group.items" :key="item.id" class="relative mb-3 pl-5 last:mb-0">
            <span
              class="absolute top-0 -left-3 flex h-6 w-6 items-center justify-center rounded-full ring-4 ring-white dark:ring-gray-900"
              :class="toneClass(item.tone)"
            >
              <FontAwesomeIcon :icon="item.icon" class="h-2.5 w-2.5" aria-hidden="true" />
            </span>
            <div class="flex items-baseline justify-between gap-2">
              <p class="min-w-0 text-sm text-gray-900 dark:text-gray-100">{{ item.text }}</p>
              <time class="shrink-0 text-xs text-gray-500 tabular-nums dark:text-gray-400">
                {{ stampOf(item.at) }}
              </time>
            </div>
            <p
              v-if="item.detail"
              class="text-xs break-words whitespace-pre-line text-gray-500 dark:text-gray-400"
            >
              {{ item.detail }}
            </p>
          </li>
        </ol>
      </section>
    </template>
  </div>
</template>
