<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { formatTimeAgo } from '@/lib/dateTime'
import { PAGE_SIZE_OPTIONS } from '@/lib/tablePageSize'
import { faCircleCheck, faEye, faMagnifyingGlass } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeBadgeV2,
  NeButton,
  NeDropdownFilter,
  NeEmptyState,
  NeInlineNotification,
  NePaginator,
  NeSortDropdown,
  NeSpinner,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
  type FilterOption,
} from '@nethesis/vue-components'
import capitalize from 'lodash/capitalize'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  getSeverityBadgeKind,
  type Alert,
  type AlertHistoryRecord,
  SYSTEM_ALERT_HISTORY_TABLE_ID,
  SEVERITY_FILTER_OPTIONS,
} from '@/lib/alerts'
import { useSystemAlertHistory } from '@/queries/systemAlerts/systemAlertHistory'
import { useAlertFilters } from '@/queries/alerts/alertFilters'
import { type AlertFilterAlert } from '@/lib/alertFilters'
import { savePageSizeToStorage } from '@/lib/tablePageSize'
import { formatDateTime } from '@/lib/dateTime'
import AlertDetailsDrawer from '@/components/alerts/AlertDetailsDrawer.vue'

const { t, locale } = useI18n()

// ── Alert history query ───────────────────────────────────────────────────────

const {
  state: historyState,
  asyncStatus: historyAsyncStatus,
  pageNum: historyPageNum,
  pageSize: historyPageSize,
  sortBy: historySortBy,
  sortDescending: historySortDescending,
  severityFilters: historySeverityFilters,
  alertnameFilters: historyAlertNameFilters,
  areDefaultFiltersApplied: historyAreDefaultFiltersApplied,
  clearFilters: historyClearFilters,
} = useSystemAlertHistory()

// ── Alert filters query ───────────────────────────────────────────────────────

const { state: alertFiltersState } = useAlertFilters()

// ── Computed data ─────────────────────────────────────────────────────────────

const historyAlerts = computed(() => historyState.value.data?.alerts ?? [])
const historyPagination = computed(() => historyState.value.data?.pagination)

// ── Filter options ────────────────────────────────────────────────────────────

const historyAlertNameOptions = computed<FilterOption[]>(() => {
  const filterAlerts = (alertFiltersState.value.data?.alerts ?? []) as AlertFilterAlert[]
  const names = new Set<string>()
  filterAlerts.forEach((a: AlertFilterAlert) => {
    if (a.name) names.add(a.name)
  })
  return Array.from(names)
    .sort()
    .map((n) => ({ id: n, label: n }))
})

// ── Empty states ──────────────────────────────────────────────────────────────

const isNoDataShown = computed(
  () =>
    !historyAlerts.value.length &&
    historyState.value.status === 'success' &&
    historyAreDefaultFiltersApplied(),
)

const isNoMatchShown = computed(
  () =>
    !historyAlerts.value.length &&
    historyState.value.status === 'success' &&
    !historyAreDefaultFiltersApplied(),
)

const isTableShown = computed(() => !isNoDataShown.value && !isNoMatchShown.value)

// ── Adapter: history record → Alert shape for AlertDetailsDrawer ──────────────

function toAlert(record: AlertHistoryRecord): Alert {
  return {
    fingerprint: record.fingerprint,
    labels: record.labels,
    annotations: record.annotations,
    status: {
      state: 'active',
      silencedBy: [],
      inhibitedBy: [],
    },
    startsAt: record.starts_at,
    endsAt: record.ends_at ?? record.starts_at,
  }
}

// ── Alert details drawer ──────────────────────────────────────────────────────

const detailsAlert = ref<Alert | undefined>(undefined)
const isDetailsDrawerShown = ref(false)

function showDetails(alert: Alert): void {
  detailsAlert.value = alert
  isDetailsDrawerShown.value = true
}
</script>

<template>
  <div class="space-y-4">
    <!-- Load error -->
    <NeInlineNotification
      v-if="historyState.status === 'error'"
      kind="error"
      :title="$t('alerts.cannot_retrieve_alert_history')"
      :description="(historyState.error as Error)?.message"
    />

    <!-- Toolbar -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex flex-wrap items-center gap-3">
        <!-- Severity filter -->
        <NeDropdownFilter
          v-model="historySeverityFilters"
          kind="checkbox"
          :label="t('alerts.severity')"
          :options="SEVERITY_FILTER_OPTIONS"
          :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
          :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
          :no-options-label="t('ne_dropdown_filter.no_options')"
          :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
          :clear-search-label="t('ne_dropdown_filter.clear_search')"
          @update:model-value="() => (historyPageNum = 1)"
        />
        <!-- Alert name filter -->
        <NeDropdownFilter
          v-model="historyAlertNameFilters"
          kind="checkbox"
          :label="t('alerts.alert')"
          :options="historyAlertNameOptions"
          show-options-filter
          :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
          :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
          :no-options-label="t('ne_dropdown_filter.no_options')"
          :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
          :clear-search-label="t('ne_dropdown_filter.clear_search')"
          @update:model-value="() => (historyPageNum = 1)"
        />
        <!-- Sort -->
        <NeSortDropdown
          v-model:sort-key="historySortBy"
          v-model:sort-descending="historySortDescending"
          :label="t('sort.sort')"
          :options="[
            { id: 'starts_at', label: t('alerts.started') },
            { id: 'ends_at', label: t('alerts.ends_at') },
            { id: 'severity', label: t('alerts.severity') },
            { id: 'alertname', label: t('alerts.alertname') },
          ]"
          :open-menu-aria-label="t('ne_dropdown.open_menu')"
          :sort-by-label="t('sort.sort_by')"
          :sort-direction-label="t('sort.direction')"
          :ascending-label="t('sort.ascending')"
          :descending-label="t('sort.descending')"
        />
        <!-- Clear filters -->
        <NeButton kind="tertiary" @click="historyClearFilters">
          {{ t('common.clear_filters') }}
        </NeButton>
      </div>
      <!-- Data updated every X seconds -->
      <div class="flex items-center gap-2">
        <NeSpinner
          color="white"
          v-if="historyAsyncStatus === 'loading' && historyState.status !== 'pending'"
        />
        <div class="text-tertiary-neutral">
          {{ t('common.data_updated_every_seconds', { seconds: 10 }) }}
        </div>
      </div>
    </div>

    <!-- Empty: no history -->
    <NeEmptyState
      v-if="isNoDataShown"
      :title="$t('alerts.no_alert_history')"
      :description="$t('alerts.no_alert_history_description')"
      :icon="faCircleCheck"
      class="bg-white dark:bg-gray-950"
    />

    <!-- Empty: no match -->
    <NeEmptyState
      v-else-if="isNoMatchShown"
      :title="$t('alerts.no_alert_history_found')"
      :description="$t('common.try_changing_search_filters')"
      :icon="faMagnifyingGlass"
      class="bg-white dark:bg-gray-950"
    >
      <NeButton kind="tertiary" @click="historyClearFilters">
        {{ $t('common.clear_filters') }}
      </NeButton>
    </NeEmptyState>

    <!-- History table -->
    <NeTable
      v-if="isTableShown"
      :sort-key="historySortBy"
      :sort-descending="historySortDescending"
      :aria-label="$t('system_detail.alert_history_title')"
      card-breakpoint="2xl"
      :loading="historyState.status === 'pending'"
      :skeleton-columns="5"
      :skeleton-rows="5"
    >
      <NeTableHead>
        <NeTableHeadCell>{{ $t('alerts.severity') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('alerts.alertname') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('alerts.started') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('alerts.ends_at') }}</NeTableHeadCell>
        <NeTableHeadCell>
          <!-- actions -->
        </NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <NeTableRow v-for="record in historyAlerts" :key="record.id">
          <!-- Severity -->
          <NeTableCell :data-label="$t('alerts.severity')">
            <NeBadgeV2 :kind="getSeverityBadgeKind(record.severity ?? undefined)">
              {{ capitalize(record.severity ?? '') }}
            </NeBadgeV2>
          </NeTableCell>
          <!-- Alert name + summary -->
          <NeTableCell :data-label="$t('alerts.alertname')">
            <span
              class="cursor-pointer font-medium hover:underline"
              @click="() => showDetails(toAlert(record))"
              >{{ record.alertname || '-' }}</span
            >
            <p
              v-if="record.summary"
              class="mt-0.5 text-sm break-all text-gray-500 dark:text-gray-400"
            >
              {{ record.summary }}
            </p>
          </NeTableCell>
          <!-- Started at -->
          <NeTableCell :data-label="$t('alerts.started')">
            <div v-if="record.starts_at">
              <p>{{ formatTimeAgo(record.starts_at, $t) }}</p>
              <p class="text-tertiary-neutral dark:text-tertiary-neutral mt-0.5">
                {{ formatDateTime(new Date(record.starts_at), locale) }}
              </p>
            </div>
            <div v-else>-</div>
          </NeTableCell>
          <!-- Ended at -->
          <NeTableCell :data-label="$t('alerts.ends_at')">
            <div v-if="record.ends_at">
              <p>{{ formatTimeAgo(record.ends_at, $t) }}</p>
              <p class="text-tertiary-neutral dark:text-tertiary-neutral mt-0.5">
                {{ formatDateTime(new Date(record.ends_at), locale) }}
              </p>
            </div>
            <div v-else>-</div>
          </NeTableCell>
          <!-- Actions -->
          <NeTableCell :data-label="$t('common.actions')">
            <div class="-ml-2.5 flex items-center gap-2 2xl:ml-0 2xl:justify-end">
              <NeButton kind="tertiary" size="sm" @click="showDetails(toAlert(record))">
                <template #prefix>
                  <FontAwesomeIcon :icon="faEye" class="h-4 w-4" aria-hidden="true" />
                </template>
                {{ $t('alerts.view_details') }}
              </NeButton>
            </div>
          </NeTableCell>
        </NeTableRow>
      </NeTableBody>
      <template v-if="historyPagination" #paginator>
        <NePaginator
          :current-page="historyPageNum"
          :total-rows="historyPagination.total_count"
          :page-size="historyPageSize"
          :page-sizes="PAGE_SIZE_OPTIONS"
          :nav-pagination-label="$t('ne_table.pagination')"
          :next-label="$t('ne_table.go_to_next_page')"
          :previous-label="$t('ne_table.go_to_previous_page')"
          :range-of-total-label="$t('ne_table.of')"
          :page-size-label="$t('ne_table.show')"
          @select-page="(page: number) => (historyPageNum = page)"
          @select-page-size="
            (size: number) => {
              historyPageSize = size
              savePageSizeToStorage(SYSTEM_ALERT_HISTORY_TABLE_ID, size)
            }
          "
        />
      </template>
    </NeTable>

    <!-- Alert details drawer -->
    <AlertDetailsDrawer
      :is-shown="isDetailsDrawerShown"
      :alert="detailsAlert"
      @close="isDetailsDrawerShown = false"
    />
  </div>
</template>
