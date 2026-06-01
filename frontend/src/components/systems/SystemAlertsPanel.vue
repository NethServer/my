<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  faBell,
  faBellSlash,
  faChevronDown,
  faChevronUp,
  faCircleCheck,
  faEye,
  faMagnifyingGlass,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeBadgeV2,
  NeButton,
  NeDropdown,
  NeDropdownFilter,
  NeEmptyState,
  NeInlineNotification,
  NePaginator,
  NeSortDropdown,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
  type FilterOption,
  type NeDropdownItem,
} from '@nethesis/vue-components'
import capitalize from 'lodash/capitalize'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import {
  deleteSystemAlertSilence,
  getAlertSilenceIds,
  getAlertSummary,
  getSeverityBadgeKind,
  isAlertSilenced,
  SYSTEM_ALERTS_KEY,
  SYSTEM_ALERT_SILENCES_KEY,
  type Alert,
  type AlertHistoryRecord,
} from '@/lib/systemAlerts'
import { useSystemAlerts } from '@/queries/systemAlerts/systemAlerts'
import { useSystemAlertHistory } from '@/queries/systemAlerts/systemAlertHistory'
import { useAlertFilters } from '@/queries/alerts/alertFilters'
import { formatDateTime } from '@/lib/dateTime'
import { canManageSystems } from '@/lib/permissions'
import MuteAlertDrawer from '@/components/alerts/MuteAlertDrawer.vue'
import AlertDetailsDrawer from '@/components/alerts/AlertDetailsDrawer.vue'
import UpdatingSpinner from '@/components/UpdatingSpinner.vue'
import { useRoute } from 'vue-router'
import { SEVERITY_FILTER_OPTIONS } from '@/lib/alerts'

const { t, locale } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()
const route = useRoute()

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

// ── Active alerts query ────────────────────────────────────────────────────────

const {
  state: alertsState,
  asyncStatus: alertsAsyncStatus,
  pageNum: alertsPageNum,
  pageSize: alertsPageSize,
  sortBy: alertsSortBy,
  sortDirection: alertsSortDirection,
  severityFilters: alertsSeverityFilters,
  alertnameFilters: alertsAlertNameFilters,
  areDefaultFiltersApplied: alertsAreDefaultFiltersApplied,
  resetFilters: alertsResetFilters,
} = useSystemAlerts()

// ── Alert history query ────────────────────────────────────────────────────────

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
  resetFilters: historyResetFilters,
} = useSystemAlertHistory()

// ── Alert filters query ────────────────────────────────────────────────────────

const { state: alertFiltersState } = useAlertFilters()

// ── Alert history collapsed state ──────────────────────────────────────────────

const isHistoryExpanded = ref(false)

// ── Computed data ─────────────────────────────────────────────────────────────

const alerts = computed(() => alertsState.value.data?.alerts ?? [])
const alertsPagination = computed(() => alertsState.value.data?.pagination)

const historyAlerts = computed(() => historyState.value.data?.alerts ?? [])
const historyPagination = computed(() => historyState.value.data?.pagination)

// ── Sort helpers ──────────────────────────────────────────────────────────────

const alertsSortDescending = computed({
  get: () => alertsSortDirection.value === 'desc',
  set: (val: boolean) => {
    alertsSortDirection.value = val ? 'desc' : 'asc'
  },
})

// ── Filter options ────────────────────────────────────────────────────────────

const alertsAlertNameOptions = computed<FilterOption[]>(() => {
  const alerts = alertFiltersState.value.data?.alerts ?? []
  const names = new Set<string>()
  alerts.forEach((a) => {
    if (a.name) names.add(a.name)
  })
  return Array.from(names)
    .sort()
    .map((n) => ({ id: n, label: n }))
})

const historyAlertNameOptions = computed<FilterOption[]>(() => {
  const names = new Set<string>()
  historyAlerts.value.forEach((a) => {
    if (a.alertname) names.add(a.alertname)
  })
  return Array.from(names).map((n) => ({ id: n, label: n }))
})

// ── Empty states ──────────────────────────────────────────────────────────────

const isAlertsNoDataShown = computed(
  () =>
    !alerts.value.length &&
    alertsState.value.status === 'success' &&
    alertsAreDefaultFiltersApplied(),
)

const isAlertsNoMatchShown = computed(
  () =>
    !alerts.value.length &&
    alertsState.value.status === 'success' &&
    !alertsAreDefaultFiltersApplied(),
)

const alertsTableShown = computed(() => !isAlertsNoDataShown.value && !isAlertsNoMatchShown.value)

const isHistoryNoDataShown = computed(
  () =>
    !historyAlerts.value.length &&
    historyState.value.status === 'success' &&
    historyAreDefaultFiltersApplied(),
)

const isHistoryNoMatchShown = computed(
  () =>
    !historyAlerts.value.length &&
    historyState.value.status === 'success' &&
    !historyAreDefaultFiltersApplied(),
)

const historyTableShown = computed(
  () => !isHistoryNoDataShown.value && !isHistoryNoMatchShown.value,
)

// ── Mute / unmute ─────────────────────────────────────────────────────────────

const selectedAlert = ref<Alert | undefined>(undefined)
const isMuteDrawerShown = ref(false)
const unmuteError = ref<string | null>(null)

function showMuteDrawer(alert: Alert) {
  selectedAlert.value = alert
  isMuteDrawerShown.value = true
}

async function handleUnmuteAlert(alert: Alert) {
  unmuteError.value = null
  try {
    const silenceIds = getAlertSilenceIds(alert)
    if (!silenceIds.length) return

    const systemId = route.params.systemId as string
    for (const silenceId of silenceIds) {
      await deleteSystemAlertSilence(systemId, silenceId)
    }

    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('alerts.alert_unmuted_successfully'),
      })
    }, 500)

    queryCache.invalidateQueries({ key: [SYSTEM_ALERTS_KEY] })
    queryCache.invalidateQueries({ key: [SYSTEM_ALERT_SILENCES_KEY] })
  } catch {
    unmuteError.value = t('alerts.cannot_unmute_alert')
  }
}

function getAlertKebabItems(alert: Alert): NeDropdownItem[] {
  if (!canManageSystems()) return []
  if (isAlertSilenced(alert)) {
    return [
      {
        id: 'unmute',
        label: t('alerts.unmute_alert'),
        icon: faBell,
        action: () => handleUnmuteAlert(alert),
      },
    ]
  }
  return [
    {
      id: 'mute',
      label: t('alerts.mute_alert'),
      icon: faBellSlash,
      action: () => showMuteDrawer(alert),
    },
  ]
}

// ── Alert details drawer ──────────────────────────────────────────────────────

const detailsAlert = ref<Alert | undefined>(undefined)
const isDetailsDrawerShown = ref(false)

function showDetails(alert: Alert) {
  detailsAlert.value = alert
  isDetailsDrawerShown.value = true
}

// ── Reload ────────────────────────────────────────────────────────────────────

function onMuteDrawerClose() {
  isMuteDrawerShown.value = false
  // Invalidate system-scoped caches so the table reflects the new silence state
  queryCache.invalidateQueries({ key: [SYSTEM_ALERTS_KEY] })
  queryCache.invalidateQueries({ key: [SYSTEM_ALERT_SILENCES_KEY] })
}
</script>

<template>
  <div class="space-y-8">
    <!-- ── Active alerts section ──────────────────────────────────────────── -->
    <section>
      <!-- Section header -->
      <div class="mb-1">
        <h4 class="text-base font-medium text-gray-900 dark:text-gray-100">
          {{ $t('system_detail.active_alerts_title') }}
        </h4>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ $t('system_detail.active_alerts_description') }}
        </p>
      </div>

      <!-- Load error -->
      <NeInlineNotification
        v-if="alertsState.status === 'error'"
        kind="error"
        :title="$t('alerts.cannot_retrieve_system_alerts')"
        :description="(alertsState.error as Error)?.message"
        class="mt-4"
      />

      <!-- Unmute error -->
      <NeInlineNotification
        v-if="unmuteError"
        kind="error"
        :title="unmuteError"
        class="mt-4"
        @close="unmuteError = null"
      />

      <!-- Toolbar -->
      <div class="mt-4 flex flex-wrap items-center justify-between gap-3">
        <div class="flex flex-wrap items-center gap-3">
          <!-- Severity filter -->
          <NeDropdownFilter
            v-model="alertsSeverityFilters"
            kind="checkbox"
            :label="t('alerts.severity')"
            :options="SEVERITY_FILTER_OPTIONS"
            :show-clear-filter="false"
            :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
            @update:model-value="() => (alertsPageNum = 1)"
          />
          <!-- Alert name filter -->
          <NeDropdownFilter
            v-model="alertsAlertNameFilters"
            kind="checkbox"
            :label="t('alerts.alert')"
            :options="alertsAlertNameOptions"
            :show-clear-filter="false"
            :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
            @update:model-value="() => (alertsPageNum = 1)"
          />
          <!-- Sort -->
          <NeSortDropdown
            v-model:sort-key="alertsSortBy"
            v-model:sort-descending="alertsSortDescending"
            :label="t('sort.sort')"
            :options="[
              { id: 'starts_at', label: t('alerts.started') },
              { id: 'severity', label: t('alerts.severity') },
              { id: 'alertname', label: t('alerts.alertname') },
            ]"
            :open-menu-aria-label="t('ne_dropdown.open_menu')"
            :sort-by-label="t('sort.sort_by')"
            :sort-direction-label="t('sort.direction')"
            :ascending-label="t('sort.ascending')"
            :descending-label="t('sort.descending')"
          />
          <!-- Reset filters -->
          <NeButton kind="tertiary" @click="alertsResetFilters">
            {{ t('common.reset_filters') }}
          </NeButton>
        </div>
        <!-- Right-side actions -->
        <div class="flex items-center gap-2">
          <UpdatingSpinner
            v-if="alertsAsyncStatus === 'loading' && alertsState.status !== 'pending'"
          />
        </div>
      </div>

      <!-- Empty: no active alerts -->
      <NeEmptyState
        v-if="isAlertsNoDataShown"
        :title="$t('alerts.no_active_alerts')"
        :description="$t('alerts.no_active_alerts_description')"
        :icon="faCircleCheck"
        class="mt-4 bg-white dark:bg-gray-950"
      />

      <!-- Empty: no matches -->
      <NeEmptyState
        v-else-if="isAlertsNoMatchShown"
        :title="$t('alerts.no_alerts_found')"
        :description="$t('common.try_changing_search_filters')"
        :icon="faMagnifyingGlass"
        class="mt-4 bg-white dark:bg-gray-950"
      >
        <NeButton kind="tertiary" @click="alertsResetFilters">
          {{ $t('common.reset_filters') }}
        </NeButton>
      </NeEmptyState>

      <!-- Active alerts table -->
      <NeTable
        v-if="alertsTableShown"
        :sort-key="alertsSortBy"
        :sort-descending="alertsSortDirection === 'desc'"
        :aria-label="$t('system_detail.active_alerts_title')"
        card-breakpoint="2xl"
        :loading="alertsState.status === 'pending'"
        :skeleton-columns="4"
        :skeleton-rows="5"
        class="mt-4"
      >
        <NeTableHead>
          <NeTableHeadCell>{{ $t('alerts.severity') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('alerts.alertname') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('alerts.started') }}</NeTableHeadCell>
          <NeTableHeadCell>
            <!-- actions — no header -->
          </NeTableHeadCell>
        </NeTableHead>
        <NeTableBody>
          <NeTableRow v-for="alert in alerts" :key="alert.fingerprint">
            <!-- Severity -->
            <NeTableCell :data-label="$t('alerts.severity')">
              <NeBadgeV2 :kind="getSeverityBadgeKind(alert.labels?.severity)">
                {{ capitalize(alert.labels?.severity) }}
              </NeBadgeV2>
            </NeTableCell>
            <!-- Alert name + summary + muted badge -->
            <NeTableCell :data-label="$t('alerts.alertname')">
              <div class="flex items-start gap-2">
                <div>
                  <p class="font-medium">{{ alert.labels?.alertname || '-' }}</p>
                  <p
                    v-if="getAlertSummary(alert, locale)"
                    class="mt-0.5 text-sm break-all text-gray-500 dark:text-gray-400"
                  >
                    {{ getAlertSummary(alert, locale) }}
                  </p>
                </div>
                <NeBadgeV2 v-if="alert.status.state === 'suppressed'" kind="gray">
                  <FontAwesomeIcon :icon="faBellSlash" class="size-4" aria-hidden="true" />
                  {{ t('alerts.muted') }}
                </NeBadgeV2>
              </div>
            </NeTableCell>
            <!-- Started at -->
            <NeTableCell :data-label="$t('alerts.started')">
              {{ formatDateTime(new Date(alert.startsAt), locale) }}
            </NeTableCell>
            <!-- Actions -->
            <NeTableCell :data-label="$t('common.actions')">
              <div class="-ml-2.5 flex items-center gap-2 2xl:ml-0 2xl:justify-end">
                <NeButton kind="tertiary" size="sm" @click="showDetails(alert)">
                  <template #prefix>
                    <FontAwesomeIcon :icon="faEye" class="h-4 w-4" aria-hidden="true" />
                  </template>
                  {{ $t('alerts.view_details') }}
                </NeButton>
                <NeDropdown
                  v-if="canManageSystems()"
                  :items="getAlertKebabItems(alert)"
                  :align-to-right="true"
                />
              </div>
            </NeTableCell>
          </NeTableRow>
        </NeTableBody>
        <template v-if="alertsPagination" #paginator>
          <NePaginator
            :current-page="alertsPageNum"
            :total-rows="alertsPagination.total_count"
            :page-size="alertsPageSize"
            :page-sizes="[10, 25, 50, 100]"
            :nav-pagination-label="$t('ne_table.pagination')"
            :next-label="$t('ne_table.go_to_next_page')"
            :previous-label="$t('ne_table.go_to_previous_page')"
            :range-of-total-label="$t('ne_table.of')"
            :page-size-label="$t('ne_table.show')"
            @select-page="(page: number) => (alertsPageNum = page)"
            @select-page-size="
              (size: number) => {
                alertsPageSize = size
                alertsPageNum = 1
              }
            "
          />
        </template>
      </NeTable>
    </section>

    <!-- ── Alert history section (collapsible) ────────────────────────────── -->
    <section>
      <!-- Collapsible header -->
      <button
        class="flex w-full items-center gap-3 py-2 text-left"
        :aria-expanded="isHistoryExpanded"
        @click="isHistoryExpanded = !isHistoryExpanded"
      >
        <FontAwesomeIcon
          :icon="isHistoryExpanded ? faChevronUp : faChevronDown"
          class="h-4 w-4 shrink-0 text-gray-500 dark:text-gray-400"
          aria-hidden="true"
        />
        <div>
          <h4 class="text-base font-medium text-gray-900 dark:text-gray-100">
            {{ $t('system_detail.alert_history_title') }}
          </h4>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ $t('system_detail.alert_history_description') }}
          </p>
        </div>
      </button>

      <!-- Expanded content -->
      <div v-if="isHistoryExpanded" class="mt-4 space-y-4">
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
              :label="t('alerts.filter_severity')"
              :options="SEVERITY_FILTER_OPTIONS"
              :show-clear-filter="false"
              :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
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
              :label="t('alerts.filter_alert')"
              :options="historyAlertNameOptions"
              :show-clear-filter="false"
              :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
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
            <!-- Reset filters -->
            <NeButton kind="tertiary" @click="historyResetFilters">
              {{ t('common.reset_filters') }}
            </NeButton>
          </div>
          <div class="flex items-center gap-2">
            <UpdatingSpinner
              v-if="historyAsyncStatus === 'loading' && historyState.status !== 'pending'"
            />
          </div>
        </div>

        <!-- Empty: no history -->
        <NeEmptyState
          v-if="isHistoryNoDataShown"
          :title="$t('alerts.no_alert_history')"
          :description="$t('alerts.no_alert_history_description')"
          :icon="faCircleCheck"
          class="bg-white dark:bg-gray-950"
        />

        <!-- Empty: no match -->
        <NeEmptyState
          v-else-if="isHistoryNoMatchShown"
          :title="$t('alerts.no_alert_history_found')"
          :description="$t('common.try_changing_search_filters')"
          :icon="faMagnifyingGlass"
          class="bg-white dark:bg-gray-950"
        >
          <NeButton kind="tertiary" @click="historyResetFilters">
            {{ $t('common.reset_filters') }}
          </NeButton>
        </NeEmptyState>

        <!-- History table -->
        <NeTable
          v-if="historyTableShown"
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
                <p class="font-medium">{{ record.alertname || '-' }}</p>
                <p
                  v-if="record.summary"
                  class="mt-0.5 text-sm break-all text-gray-500 dark:text-gray-400"
                >
                  {{ record.summary }}
                </p>
              </NeTableCell>
              <!-- Started at -->
              <NeTableCell :data-label="$t('alerts.started')">
                {{ record.starts_at ? formatDateTime(new Date(record.starts_at), locale) : '-' }}
              </NeTableCell>
              <!-- Ended at -->
              <NeTableCell :data-label="$t('alerts.ends_at')">
                {{ record.ends_at ? formatDateTime(new Date(record.ends_at), locale) : '-' }}
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
              :page-sizes="[10, 25, 50, 100]"
              :nav-pagination-label="$t('ne_table.pagination')"
              :next-label="$t('ne_table.go_to_next_page')"
              :previous-label="$t('ne_table.go_to_previous_page')"
              :range-of-total-label="$t('ne_table.of')"
              :page-size-label="$t('ne_table.show')"
              @select-page="(page: number) => (historyPageNum = page)"
              @select-page-size="
                (size: number) => {
                  historyPageSize = size
                  historyPageNum = 1
                }
              "
            />
          </template>
        </NeTable>
      </div>
    </section>

    <!-- Mute alert drawer -->
    <MuteAlertDrawer
      :is-shown="isMuteDrawerShown"
      :alert="selectedAlert"
      @close="onMuteDrawerClose"
    />

    <!-- Alert details drawer -->
    <AlertDetailsDrawer
      :is-shown="isDetailsDrawerShown"
      :alert="detailsAlert"
      @close="isDetailsDrawerShown = false"
    />
  </div>
</template>
