<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useInventoryTimeline } from '@/queries/systems/inventoryTimeline'
import {
  INVENTORY_DIFFS_KEY,
  getInventoryDiffs,
  type InventoryDiff,
  type InventoryDiffCategory,
  type InventoryDiffSeverity,
  type InventoryDiffType,
} from '@/lib/systems/inventoryDiffs'
import {
  INVENTORY_MOCK_ENABLED,
  mockInventoryDiffs,
  mockDiffsPagination,
} from '@/lib/systems/inventoryMocks'
import { formatDateTimeNoSeconds } from '@/lib/dateTime'
import { canReadSystems } from '@/lib/permissions'
import { useLoginStore } from '@/stores/login'
import { useQuery } from '@pinia/colada'
import { computed, onWatcherCleanup, ref, useTemplateRef, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  NeButton,
  NeBadgeV2,
  NeDropdownFilter,
  NeInlineNotification,
  NeSkeleton,
  NeTextInput,
  type FilterOption,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  faChevronDown,
  faChevronUp,
  faPen,
  faPlus,
  faMinus,
  faAngleDown,
  faAngleUp,
  faArrowRight,
} from '@fortawesome/free-solid-svg-icons'

const { t, locale } = useI18n()
const route = useRoute()
const loginStore = useLoginStore()

// ── Timeline infinite query ──────────────────────────────────────────────────
const {
  state: timelineState,
  asyncStatus: timelineAsyncStatus,
  hasNextPage,
  loadNextPage,
  severityFilter,
  categoryFilter,
  diffTypeFilter,
  fromDate,
  toDate,
  areDefaultFiltersApplied,
  resetFilters: resetTimelineFilters,
  allInventoryIds,
  allGroups,
} = useInventoryTimeline()

// ── Local state ──────────────────────────────────────────────────────────────
const textFilter = ref('')
const expandedGroups = ref<Set<string>>(new Set())
const expandedDiffs = ref<Set<number>>(new Set())

// ── Diffs query (reactive to allInventoryIds from timeline) ──────────────────
const { state: diffsState, asyncStatus: diffsAsyncStatus } = useQuery({
  key: () => [
    INVENTORY_DIFFS_KEY,
    'timeline',
    {
      systemId: route.params.systemId,
      inventoryIds: allInventoryIds.value,
      severityFilter: severityFilter.value,
      categoryFilter: categoryFilter.value,
      diffTypeFilter: diffTypeFilter.value,
      fromDate: fromDate.value,
      toDate: toDate.value,
    },
  ],
  enabled: () =>
    !!loginStore.jwtToken &&
    canReadSystems() &&
    !!route.params.systemId &&
    allInventoryIds.value.length > 0,
  query: () => {
    const apiCall = getInventoryDiffs(
      route.params.systemId as string,
      1,
      500,
      severityFilter.value,
      categoryFilter.value,
      diffTypeFilter.value,
      allInventoryIds.value,
      fromDate.value,
      toDate.value,
    )
    if (INVENTORY_MOCK_ENABLED) {
      apiCall.catch(() => {})
      return Promise.resolve({ diffs: mockInventoryDiffs, pagination: mockDiffsPagination })
    }
    return apiCall
  },
})

// ── Auto-expand all groups when they load ─────────────────────────────────────
watch(
  () => allGroups.value,
  (groups) => {
    groups.forEach((g) => expandedGroups.value.add(g.date))
  },
  { immediate: true, deep: true },
)

// ── Filter options ────────────────────────────────────────────────────────────
const severityFilterOptions = computed<FilterOption[]>(() => [
  { id: 'critical', label: t('system_detail.severity_critical') },
  { id: 'high', label: t('system_detail.severity_high') },
  { id: 'medium', label: t('system_detail.severity_medium') },
  { id: 'low', label: t('system_detail.severity_low') },
])

const categoryFilterOptions = computed<FilterOption[]>(() => [
  { id: 'os', label: t('system_detail.category_os') },
  { id: 'hardware', label: t('system_detail.category_hardware') },
  { id: 'network', label: t('system_detail.category_network') },
  { id: 'security', label: t('system_detail.category_security') },
  { id: 'backup', label: t('system_detail.category_backup') },
  { id: 'features', label: t('system_detail.category_features') },
  { id: 'modules', label: t('system_detail.category_modules') },
  { id: 'cluster', label: t('system_detail.category_cluster') },
  { id: 'nodes', label: t('system_detail.category_nodes') },
  { id: 'system', label: t('system_detail.category_system') },
])

const diffTypeFilterOptions = computed<FilterOption[]>(() => [
  { id: 'create', label: t('system_detail.diff_type_create') },
  { id: 'update', label: t('system_detail.diff_type_update') },
  { id: 'delete', label: t('system_detail.diff_type_delete') },
])

// ── Helpers ───────────────────────────────────────────────────────────────────
const DAY_MS = 24 * 60 * 60 * 1000

function todayDateString(): string {
  return new Date().toISOString().slice(0, 10)
}

function formatGroupDate(dateStr: string): string {
  const date = new Date(dateStr + 'T00:00:00')
  return date.toLocaleDateString(locale.value, { day: 'numeric', month: 'short', year: 'numeric' })
}

function gapDaysBetween(newerDateStr: string, olderDateStr: string, newerIsToday: boolean): number {
  const newer = new Date(newerDateStr + 'T00:00:00')
  const older = new Date(olderDateStr + 'T00:00:00')
  const diffDays = Math.round((newer.getTime() - older.getTime()) / DAY_MS)
  // If the newer date is "today" with no changes it counts as a gap day itself
  return newerIsToday ? diffDays : diffDays - 1
}

function gapBadgeText(days: number): string {
  if (days === 1) return t('system_detail.one_day_no_changes')
  return t('system_detail.n_days_no_changes', { n: days })
}

// ── Display groups (today + all groups) ───────────────────────────────────────
interface DisplayGroup {
  date: string
  isToday: boolean
  change_count: number
  inventory_ids: number[]
  gapDaysAfter: number // gap to the NEXT (older) entry in the timeline
}

const today = todayDateString()

const displayGroups = computed<DisplayGroup[]>(() => {
  const groups = allGroups.value
  const result: DisplayGroup[] = []

  // Always show Today as the first entry
  const todayGroup = groups.find((g) => g.date === today)
  const firstRealGroup = todayGroup ? groups[1] : groups[0]

  const gapAfterToday = firstRealGroup ? gapDaysBetween(today, firstRealGroup.date, !todayGroup) : 0

  result.push({
    date: today,
    isToday: true,
    change_count: todayGroup?.change_count ?? 0,
    inventory_ids: todayGroup?.inventory_ids ?? [],
    gapDaysAfter: gapAfterToday > 0 ? gapAfterToday : 0,
  })

  // Add the remaining groups (skip today's group if it was in allGroups)
  const otherGroups = todayGroup ? groups.slice(1) : groups
  otherGroups.forEach((group, idx) => {
    const nextGroup = otherGroups[idx + 1]
    const gapAfter = nextGroup ? gapDaysBetween(group.date, nextGroup.date, false) : 0
    result.push({
      date: group.date,
      isToday: false,
      change_count: group.change_count,
      inventory_ids: group.inventory_ids,
      gapDaysAfter: gapAfter > 0 ? gapAfter : 0,
    })
  })

  return result
})

const isTimelineEmpty = computed(() => {
  if (timelineState.value.status !== 'success') return false
  return allGroups.value.filter((g) => g.change_count > 0).length === 0
})

// ── Diffs helpers ─────────────────────────────────────────────────────────────
const allDiffs = computed<InventoryDiff[]>(() => diffsState.value.data?.diffs ?? [])

const timelineIsPending = computed(() => timelineState.value.status === 'pending')
const timelineError = computed(() =>
  timelineState.value.status === 'error' ? timelineState.value.error : null,
)
const diffsError = computed(() =>
  diffsState.value.status === 'error' ? diffsState.value.error : null,
)
const diffsIsLoading = computed(
  () => diffsState.value.status === 'pending' || diffsAsyncStatus.value === 'loading',
)
function getDiffsForGroup(group: DisplayGroup): InventoryDiff[] {
  const idSet = new Set(group.inventory_ids)
  return allDiffs.value.filter((d) => idSet.has(d.inventory_id))
}

function formatDiffValue(value: unknown): string {
  if (value === null || value === undefined) return '—'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value) || '—'
}

function objectToLines(value: unknown): string[] {
  if (value !== null && typeof value === 'object' && !Array.isArray(value)) {
    return Object.entries(value as Record<string, unknown>).map(
      ([k, v]) => `"${k}": ${JSON.stringify(v)}`,
    )
  }
  return [formatDiffValue(value)]
}

function getFilteredDiffsForGroup(group: DisplayGroup): InventoryDiff[] {
  const diffs = getDiffsForGroup(group)
  if (!textFilter.value.trim()) return diffs
  const search = textFilter.value.trim().toLowerCase()
  return diffs.filter((d) => d.field_path.toLowerCase().includes(search))
}

// ── Group expand/collapse ─────────────────────────────────────────────────────
function toggleGroup(date: string) {
  if (expandedGroups.value.has(date)) {
    expandedGroups.value.delete(date)
  } else {
    expandedGroups.value.add(date)
  }
}

// ── Diff expand/collapse ──────────────────────────────────────────────────────
function toggleDiff(diffId: number) {
  if (expandedDiffs.value.has(diffId)) {
    expandedDiffs.value.delete(diffId)
  } else {
    expandedDiffs.value.add(diffId)
  }
}

// ── Diff type styling ─────────────────────────────────────────────────────────
function getDiffTypeIcon(type: InventoryDiffType) {
  if (type === 'create') return faPlus
  if (type === 'delete') return faMinus
  return faPen
}

function getDiffTypeIconBg(type: InventoryDiffType): string {
  if (type === 'create') return 'bg-green-600'
  if (type === 'delete') return 'bg-red-500'
  return 'bg-blue-700'
}

function getDiffTypeBorder(type: InventoryDiffType): string {
  if (type === 'create') return 'border-l-green-500'
  if (type === 'delete') return 'border-l-red-500'
  return 'border-l-blue-700'
}

//// use NeBadgeV2Kind as return type instead of string literal union

// ── Severity badge styling ────────────────────────────────────────────────────
function getSeverityKind(
  severity: InventoryDiffSeverity,
): 'rose' | 'amber' | 'blue' | 'custom' | 'primary' | 'indigo' | 'gray' | 'green' {
  if (severity === 'critical') return 'rose'
  if (severity === 'high') return 'amber'
  if (severity === 'low') return 'blue'
  return 'custom'
}

function getSeverityCustomKindClasses(severity: InventoryDiffSeverity): string | undefined {
  if (severity === 'medium')
    return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-400'
  return undefined
}

function getSeverityLabel(severity: InventoryDiffSeverity): string {
  return t(`system_detail.severity_${severity}`)
}

function getCategoryLabel(category: InventoryDiffCategory): string {
  return t(`system_detail.category_${category}`)
}

// ── Reset all filters ─────────────────────────────────────────────────────────
function resetAllFilters() {
  textFilter.value = ''
  resetTimelineFilters()
}

// ── Infinite scroll (IntersectionObserver) ────────────────────────────────────
const loadMoreTrigger = useTemplateRef<HTMLElement>('loadMoreTrigger')

watch(loadMoreTrigger, (el) => {
  if (!el) return
  const observer = new IntersectionObserver(
    (entries) => {
      if (entries[0]?.isIntersecting) {
        loadNextPage()
      }
    },
    { rootMargin: '300px', threshold: [0] },
  )
  observer.observe(el)
  onWatcherCleanup(() => observer.disconnect())
})

// ── Computed filter state for NeDropdownFilter (need arrays of string IDs) ───
const severityFilterModel = computed<string[]>({
  get: () => severityFilter.value as string[],
  set: (val) => {
    severityFilter.value = val as InventoryDiffSeverity[]
  },
})

const categoryFilterModel = computed<string[]>({
  get: () => categoryFilter.value as string[],
  set: (val) => {
    categoryFilter.value = val as InventoryDiffCategory[]
  },
})

const diffTypeFilterModel = computed<string[]>({
  get: () => diffTypeFilter.value as string[],
  set: (val) => {
    diffTypeFilter.value = val as InventoryDiffType[]
  },
})
</script>

<template>
  <!-- Filters bar -->
  <div class="mb-6 flex flex-wrap items-center gap-4">
    <!-- Text filter -->
    <NeTextInput
      v-model.trim="textFilter"
      is-search
      :placeholder="$t('common.filter')"
      class="max-w-xs"
    />
    <!-- Severity filter -->
    <NeDropdownFilter
      v-model="severityFilterModel"
      kind="checkbox"
      :label="t('system_detail.severity')"
      :options="severityFilterOptions"
      :show-clear-filter="false"
      :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
      :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
      :no-options-label="t('ne_dropdown_filter.no_options')"
      :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
      :clear-search-label="t('ne_dropdown_filter.clear_search')"
    />
    <!-- Category filter -->
    <NeDropdownFilter
      v-model="categoryFilterModel"
      kind="checkbox"
      :label="t('system_detail.category')"
      :options="categoryFilterOptions"
      show-options-filter
      :show-clear-filter="false"
      :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
      :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
      :no-options-label="t('ne_dropdown_filter.no_options')"
      :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
      :clear-search-label="t('ne_dropdown_filter.clear_search')"
    />
    <!-- Change type filter -->
    <NeDropdownFilter
      v-model="diffTypeFilterModel"
      kind="checkbox"
      :label="t('system_detail.change_type')"
      :options="diffTypeFilterOptions"
      :show-clear-filter="false"
      :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
      :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
      :no-options-label="t('ne_dropdown_filter.no_options')"
      :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
      :clear-search-label="t('ne_dropdown_filter.clear_search')"
    />
    <!-- Date range: from -->
    <div class="flex items-center gap-2">
      <label class="text-sm text-gray-500 dark:text-gray-400">{{
        t('system_detail.from_date')
      }}</label>
      <input
        v-model="fromDate"
        type="date"
        class="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-700 focus:ring-2 focus:ring-sky-500 focus:outline-none dark:border-gray-600 dark:bg-gray-900 dark:text-gray-300"
      />
    </div>
    <!-- Date range: to -->
    <div class="flex items-center gap-2">
      <label class="text-sm text-gray-500 dark:text-gray-400">{{
        t('system_detail.to_date')
      }}</label>
      <input
        v-model="toDate"
        type="date"
        class="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-700 focus:ring-2 focus:ring-sky-500 focus:outline-none dark:border-gray-600 dark:bg-gray-900 dark:text-gray-300"
      />
    </div>
    <!-- Reset filters -->
    <NeButton kind="tertiary" @click="resetAllFilters">
      {{ t('common.reset_filters') }}
    </NeButton>
  </div>

  <!-- Error notifications -->
  <NeInlineNotification
    v-if="timelineError"
    kind="error"
    :title="t('system_detail.cannot_retrieve_inventory_timeline')"
    :description="timelineError?.message"
    class="mb-6"
  />
  <NeInlineNotification
    v-if="diffsError"
    kind="error"
    :title="t('system_detail.cannot_retrieve_inventory_diffs')"
    :description="diffsError?.message"
    class="mb-6"
  />

  <!-- Loading skeleton (initial load) -->
  <div v-if="timelineIsPending" class="space-y-6">
    <div v-for="i in 3" :key="i" class="flex gap-12">
      <NeSkeleton class="h-5 w-24" />
      <div class="flex-1 space-y-3">
        <NeSkeleton class="h-5 w-32" />
        <NeSkeleton class="h-14 w-full" />
        <NeSkeleton class="h-14 w-full" />
      </div>
    </div>
  </div>

  <!-- Empty state -->
  <div
    v-else-if="isTimelineEmpty"
    class="flex flex-col items-center justify-center py-16 text-center text-gray-500 dark:text-gray-400"
  >
    <p class="text-base font-medium">{{ t('system_detail.no_changes_in_timeline') }}</p>
    <p class="mt-1 text-sm">
      {{
        areDefaultFiltersApplied
          ? t('system_detail.no_changes_in_timeline_no_filters_description')
          : t('system_detail.no_changes_in_timeline_description')
      }}
    </p>
    <NeButton
      v-if="!areDefaultFiltersApplied"
      kind="tertiary"
      class="mt-4"
      @click="resetAllFilters"
    >
      {{ t('common.reset_filters') }}
    </NeButton>
  </div>

  <!-- Timeline -->
  <div v-else class="relative mt-2">
    <!-- Vertical timeline line -->
    <div
      class="absolute top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-700"
      style="left: 144px"
    ></div>

    <div v-for="group in displayGroups" :key="group.date">
      <!-- Date header row -->
      <div class="relative mb-2 flex items-start">
        <!-- Date label column (right-aligned) -->
        <div class="w-36 flex-shrink-0 pt-0.5 pr-6 text-right">
          <span
            class="text-base font-medium"
            :class="
              group.isToday
                ? 'text-indigo-600 dark:text-indigo-400'
                : 'text-gray-500 dark:text-gray-400'
            "
          >
            {{ group.isToday ? t('system_detail.today') : formatGroupDate(group.date) }}
          </span>
        </div>

        <!-- Timeline dot (centered on the vertical line at left: 144px, dot size 10px → left: 139px) -->
        <div
          class="absolute z-10 size-2.5 rounded-full"
          :class="
            group.isToday ? 'bg-indigo-700 dark:bg-indigo-500' : 'bg-gray-300 dark:bg-gray-600'
          "
          style="left: 139px; top: 7px"
        ></div>

        <!-- Content -->
        <div class="flex-1 pl-10">
          <!-- Today with no changes -->
          <template v-if="group.isToday && group.change_count === 0">
            <span class="text-base font-medium text-indigo-600 dark:text-indigo-400">
              {{ t('system_detail.no_changes_today') }}
            </span>
          </template>

          <!-- Group with changes: toggle header -->
          <template v-else-if="group.change_count > 0">
            <button
              class="flex items-center gap-2 text-base font-medium text-gray-600 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200"
              @click="toggleGroup(group.date)"
            >
              <FontAwesomeIcon
                :icon="expandedGroups.has(group.date) ? faChevronUp : faChevronDown"
                class="size-3.5"
              />
              <span>
                {{
                  group.change_count === 1
                    ? t('system_detail.one_change')
                    : t('system_detail.n_changes', { n: group.change_count })
                }}
              </span>
            </button>

            <!-- Diffs list -->
            <div v-if="expandedGroups.has(group.date)" class="mt-3 space-y-3">
              <!-- Loading diffs state -->
              <template v-if="diffsIsLoading">
                <NeSkeleton v-for="j in group.change_count" :key="j" class="h-14 w-full" />
              </template>

              <template v-else>
                <div
                  v-for="diff in getFilteredDiffsForGroup(group)"
                  :key="diff.id"
                  class="overflow-hidden rounded-lg border-l-4 bg-white shadow-sm dark:bg-gray-950"
                  :class="getDiffTypeBorder(diff.diff_type)"
                >
                  <!-- Diff header row -->
                  <div
                    class="flex cursor-pointer items-center justify-between px-5 py-3"
                    @click="toggleDiff(diff.id)"
                  >
                    <div class="flex items-center gap-4">
                      <!-- Change type icon -->
                      <div
                        class="flex size-6 flex-shrink-0 items-center justify-center rounded-full"
                        :class="getDiffTypeIconBg(diff.diff_type)"
                      >
                        <FontAwesomeIcon
                          :icon="getDiffTypeIcon(diff.diff_type)"
                          class="size-3 text-white"
                        />
                      </div>
                      <!-- Category -->
                      <span
                        class="w-20 flex-shrink-0 text-sm font-medium text-gray-700 uppercase dark:text-gray-50"
                      >
                        {{ getCategoryLabel(diff.category) }}
                      </span>
                      <!-- Field path -->
                      <span class="text-sm text-gray-500 dark:text-gray-400">
                        {{ diff.field_path }}
                      </span>
                      <!-- Severity badge -->
                      <NeBadgeV2
                        :kind="getSeverityKind(diff.severity)"
                        :custom-kind-classes="getSeverityCustomKindClasses(diff.severity)"
                      >
                        {{ getSeverityLabel(diff.severity) }}
                      </NeBadgeV2>
                    </div>
                    <!-- Expand chevron -->
                    <FontAwesomeIcon
                      :icon="expandedDiffs.has(diff.id) ? faAngleUp : faAngleDown"
                      class="size-4 flex-shrink-0 text-gray-400 dark:text-gray-500"
                    />
                  </div>

                  <!-- Expanded diff detail -->
                  <div
                    v-if="expandedDiffs.has(diff.id)"
                    class="border-t border-gray-100 px-5 py-3 dark:border-gray-800"
                  >
                    <!-- Update: inline strikethrough → arrow → new value -->
                    <div
                      v-if="diff.diff_type === 'update'"
                      class="flex items-center gap-4 rounded-sm bg-blue-50 px-1.5 py-0.5 dark:bg-blue-950"
                    >
                      <FontAwesomeIcon
                        :icon="faPen"
                        class="size-3 shrink-0 text-blue-700 dark:text-blue-400"
                      />
                      <span class="font-mono text-sm text-gray-700 line-through dark:text-gray-300">
                        {{ formatDiffValue(diff.previous_value) }}
                      </span>
                      <FontAwesomeIcon
                        :icon="faArrowRight"
                        class="size-4 shrink-0 text-gray-500 dark:text-gray-400"
                      />
                      <span class="font-mono text-sm text-gray-700 dark:text-gray-300">
                        {{ formatDiffValue(diff.current_value) }}
                      </span>
                    </div>
                    <!-- Create: green list of added values -->
                    <div
                      v-else-if="diff.diff_type === 'create'"
                      class="flex flex-col gap-0.5 rounded-sm bg-green-50 px-1.5 py-0.5 dark:bg-green-950"
                    >
                      <div
                        v-for="(line, idx) in objectToLines(diff.current_value)"
                        :key="idx"
                        class="flex items-center gap-4"
                      >
                        <FontAwesomeIcon
                          :icon="faPlus"
                          class="size-3 shrink-0 text-green-600 dark:text-green-400"
                        />
                        <span class="font-mono text-sm text-gray-700 dark:text-gray-300">{{
                          line
                        }}</span>
                      </div>
                    </div>
                    <!-- Delete: rose list of removed values with strikethrough -->
                    <div
                      v-else
                      class="flex flex-col gap-0.5 rounded-sm bg-rose-50 px-1.5 py-0.5 dark:bg-rose-950"
                    >
                      <div
                        v-for="(line, idx) in objectToLines(diff.previous_value)"
                        :key="idx"
                        class="flex items-center gap-4"
                      >
                        <FontAwesomeIcon
                          :icon="faMinus"
                          class="size-3 shrink-0 text-rose-700 dark:text-rose-400"
                        />
                        <span
                          class="font-mono text-sm text-gray-700 line-through dark:text-gray-300"
                          >{{ line }}</span
                        >
                      </div>
                    </div>
                    <!-- Timestamp -->
                    <p class="mt-2 text-[10px] text-gray-500 dark:text-gray-400">
                      {{ formatDateTimeNoSeconds(new Date(diff.created_at), locale) }}
                    </p>
                  </div>
                </div>

                <!-- No diffs found -->
                <p
                  v-if="getFilteredDiffsForGroup(group).length === 0"
                  class="text-sm text-gray-400 dark:text-gray-500"
                >
                  {{ t('common.try_changing_search_filters') }}
                </p>
              </template>
            </div>
          </template>
        </div>
      </div>

      <!-- Gap badge (days without changes between this group and the next) -->
      <div v-if="group.gapDaysAfter > 0" class="my-8 flex items-start">
        <div class="w-36 flex-shrink-0"></div>
        <div class="flex-1 pl-10">
          <span
            class="inline-block rounded bg-gray-200 px-3 py-1 text-sm font-medium text-gray-800 dark:bg-gray-600 dark:text-gray-100"
          >
            {{ gapBadgeText(group.gapDaysAfter) }}
          </span>
        </div>
      </div>
    </div>

    <!-- Load more trigger (IntersectionObserver target) -->
    <div v-if="hasNextPage" ref="loadMoreTrigger" class="flex items-start py-4">
      <div class="w-36 flex-shrink-0"></div>
      <div class="flex-1 pl-10">
        <p
          v-if="timelineAsyncStatus === 'loading'"
          class="text-sm text-gray-400 dark:text-gray-500"
        >
          {{ t('common.loading') }}
        </p>
      </div>
    </div>
  </div>
</template>
