<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  faArrowsRotate,
  faBell,
  faBellSlash,
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
import { PAGE_SIZE_OPTIONS } from '@/lib/tablePageSize'
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
  SYSTEM_ALERTS_TABLE_ID,
  type Alert,
} from '@/lib/systemAlerts'
import { savePageSizeToStorage } from '@/lib/tablePageSize'
import { useSystemAlerts } from '@/queries/systemAlerts/systemAlerts'
import { useAlertFilters } from '@/queries/alerts/alertFilters'
import { type AlertFilterAlert } from '@/lib/alertFilters'
import { formatDateTime, formatTimeAgo } from '@/lib/dateTime'
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

// ── Active alerts query ───────────────────────────────────────────────────────

const {
  state: alertsState,
  asyncStatus: alertsAsyncStatus,
  pageNum: alertsPageNum,
  pageSize: alertsPageSize,
  sortBy: alertsSortBy,
  sortDirection: alertsSortDirection,
  severityFilters: alertsSeverityFilters,
  alertnameFilters: alertsAlertNameFilters,
  statusFilters: alertsStatusFilters,
  areDefaultFiltersApplied: alertsAreDefaultFiltersApplied,
  resetFilters: alertsResetFilters,
  refetch: alertsRefetch,
} = useSystemAlerts()

// ── Alert filters query ───────────────────────────────────────────────────────

const { state: alertFiltersState } = useAlertFilters()

// ── Status filter options ─────────────────────────────────────────────────────

const statusFilterOptions = computed(() => [
  { id: 'suppressed', label: t('alerts.muted') },
  { id: 'active', label: t('alerts.unmuted') },
])

// ── Computed data ─────────────────────────────────────────────────────────────

const alerts = computed(() => alertsState.value.data?.alerts ?? [])
const alertsPagination = computed(() => alertsState.value.data?.pagination)

// ── Sort helper ───────────────────────────────────────────────────────────────

const alertsSortDescending = computed({
  get: () => alertsSortDirection.value === 'desc',
  set: (val: boolean) => {
    alertsSortDirection.value = val ? 'desc' : 'asc'
  },
})

// ── Filter options ────────────────────────────────────────────────────────────

const alertsAlertNameOptions = computed<FilterOption[]>(() => {
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
    !alerts.value.length &&
    alertsState.value.status === 'success' &&
    alertsAreDefaultFiltersApplied(),
)

const isNoMatchShown = computed(
  () =>
    !alerts.value.length &&
    alertsState.value.status === 'success' &&
    !alertsAreDefaultFiltersApplied(),
)

const isTableShown = computed(() => !isNoDataShown.value && !isNoMatchShown.value)

// ── Mute / unmute ─────────────────────────────────────────────────────────────

const selectedAlert = ref<Alert | undefined>(undefined)
const isMuteDrawerShown = ref(false)
const unmuteError = ref<string | null>(null)

function showMuteDrawer(alert: Alert): void {
  selectedAlert.value = alert
  isMuteDrawerShown.value = true
}

async function handleUnmuteAlert(alert: Alert): Promise<void> {
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

function showDetails(alert: Alert): void {
  detailsAlert.value = alert
  isDetailsDrawerShown.value = true
}

// ── Mute drawer close ─────────────────────────────────────────────────────────

function onMuteDrawerClose(): void {
  isMuteDrawerShown.value = false
  queryCache.invalidateQueries({ key: [SYSTEM_ALERTS_KEY] })
  queryCache.invalidateQueries({ key: [SYSTEM_ALERT_SILENCES_KEY] })
}
</script>

<template>
  <div class="space-y-4">
    <!-- Load error -->
    <NeInlineNotification
      v-if="alertsState.status === 'error'"
      kind="error"
      :title="$t('alerts.cannot_retrieve_system_alerts')"
      :description="(alertsState.error as Error)?.message"
    />

    <!-- Unmute error -->
    <NeInlineNotification
      v-if="unmuteError"
      kind="error"
      :title="unmuteError"
      @close="unmuteError = null"
    />

    <!-- Toolbar -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex flex-wrap items-center gap-3">
        <!-- Severity filter -->
        <NeDropdownFilter
          v-model="alertsSeverityFilters"
          kind="checkbox"
          :label="t('alerts.severity')"
          :options="SEVERITY_FILTER_OPTIONS"
          :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
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
          show-options-filter
          :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
          :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
          :no-options-label="t('ne_dropdown_filter.no_options')"
          :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
          :clear-search-label="t('ne_dropdown_filter.clear_search')"
          @update:model-value="() => (alertsPageNum = 1)"
        />
        <!-- Status filter -->
        <NeDropdownFilter
          v-model="alertsStatusFilters"
          kind="checkbox"
          :label="t('common.status')"
          :options="statusFilterOptions"
          :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
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
      <div class="flex items-center gap-4">
        <UpdatingSpinner
          v-if="alertsAsyncStatus === 'loading' && alertsState.status !== 'pending'"
        />
        <NeButton
          kind="secondary"
          size="md"
          :disabled="alertsAsyncStatus === 'loading'"
          @click="alertsRefetch()"
        >
          <template #prefix>
            <FontAwesomeIcon :icon="faArrowsRotate" class="h-4 w-4" aria-hidden="true" />
          </template>
          {{ $t('alerts.reload_alerts') }}
        </NeButton>
      </div>
    </div>

    <!-- Empty: no active alerts -->
    <NeEmptyState
      v-if="isNoDataShown"
      :title="$t('alerts.no_active_alerts')"
      :description="$t('alerts.no_active_alerts_description')"
      :icon="faCircleCheck"
      class="bg-white dark:bg-gray-950"
    />

    <!-- Empty: no matches -->
    <NeEmptyState
      v-else-if="isNoMatchShown"
      :title="$t('alerts.no_alerts_found')"
      :description="$t('common.try_changing_search_filters')"
      :icon="faMagnifyingGlass"
      class="bg-white dark:bg-gray-950"
    >
      <NeButton kind="tertiary" @click="alertsResetFilters">
        {{ $t('common.reset_filters') }}
      </NeButton>
    </NeEmptyState>

    <!-- Active alerts table -->
    <NeTable
      v-if="isTableShown"
      :sort-key="alertsSortBy"
      :sort-descending="alertsSortDirection === 'desc'"
      :aria-label="$t('system_detail.active_alerts_title')"
      card-breakpoint="2xl"
      :loading="alertsState.status === 'pending'"
      :skeleton-columns="4"
      :skeleton-rows="5"
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
                <span
                  class="cursor-pointer font-medium hover:underline"
                  @click="() => showDetails(alert)"
                  >{{ alert.labels?.alertname || '-' }}</span
                >
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
            <div>
              <p>{{ formatTimeAgo(alert.startsAt, $t) }}</p>
              <p class="text-tertiary-neutral dark:text-tertiary-neutral mt-0.5">
                {{ formatDateTime(new Date(alert.startsAt), locale) }}
              </p>
            </div>
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
          :page-sizes="PAGE_SIZE_OPTIONS"
          :nav-pagination-label="$t('ne_table.pagination')"
          :next-label="$t('ne_table.go_to_next_page')"
          :previous-label="$t('ne_table.go_to_previous_page')"
          :range-of-total-label="$t('ne_table.of')"
          :page-size-label="$t('ne_table.show')"
          @select-page="(page: number) => (alertsPageNum = page)"
          @select-page-size="
            (size: number) => {
              alertsPageSize = size
              savePageSizeToStorage(SYSTEM_ALERTS_TABLE_ID, size)
            }
          "
        />
      </template>
    </NeTable>

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
