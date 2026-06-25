<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  faArrowRight,
  faBell,
  faBellSlash,
  faCircleCheck,
  faEye,
  faMagnifyingGlass,
  faServer,
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
  NeSpinner,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
  type FilterOption,
  type NeDropdownItem,
} from '@nethesis/vue-components'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { PAGE_SIZE_OPTIONS } from '@/lib/tablePageSize'
import { useAlerts } from '@/queries/alerts/alerts'
import { useAlertFilters } from '@/queries/alerts/alertFilters'
import { useSystems } from '@/queries/systems/systems'
import {
  ALERTS_TABLE_ID,
  deleteAlertSilence,
  getAlertSilenceIds,
  getAlertSummary,
  getSeverityBadgeKind,
  isAlertSilenced,
  SEVERITY_FILTER_OPTIONS,
  type Alert,
} from '@/lib/alerts'
import { setPendingAlertState, isProcessing } from '@/lib/alertPendingStates'
import { type AlertFilterAlert } from '@/lib/alertFilters'
import { useNotificationsStore } from '@/stores/notifications'
import { formatDateTime, formatTimeAgo } from '@/lib/dateTime'
import { canManageSystems } from '@/lib/permissions'
import OrganizationIconAndLink from '@/components/organizations/OrganizationIconAndLink.vue'
import MuteAlertDrawer from '@/components/alerts/MuteAlertDrawer.vue'
import AlertDetailsDrawer from '@/components/alerts/AlertDetailsDrawer.vue'
import ProcessingAlertBadge from '@/components/alerts/ProcessingAlertBadge.vue'
import capitalize from 'lodash/capitalize'
import SystemDropdownFilter from '@/components/systems/SystemDropdownFilter.vue'
import OrganizationDropdownFilter from '@/components/organizations/OrganizationDropdownFilter.vue'
import { savePageSizeToStorage } from '@/lib/tablePageSize'
import { isUserCustomer } from '@/lib/organizations/organizations'
import SystemLogoAndLink from '../systems/SystemLogoAndLink.vue'

const { t, locale } = useI18n()
const router = useRouter()
const notificationsStore = useNotificationsStore()

const { state: systemsState } = useSystems()

const {
  state: alertsState,
  asyncStatus: alertsAsyncStatus,
  pageNum,
  pageSize,
  sortBy,
  sortDirection,
  statusFilters,
  severityFilters,
  alertnameFilters,
  systemKeyFilters,
  organizationIds,
  areDefaultFiltersApplied,
  clearFilters,
  refetch,
} = useAlerts()

const { state: alertFiltersState } = useAlertFilters()

const alerts = computed(() => alertsState.value.data?.alerts ?? [])

const pagination = computed(() => alertsState.value.data?.pagination)

const isNoDataEmptyStateShown = computed(
  () =>
    !alerts.value.length && alertsState.value.status === 'success' && areDefaultFiltersApplied(),
)

const isNoMatchEmptyStateShown = computed(
  () =>
    !alerts.value.length && alertsState.value.status === 'success' && !areDefaultFiltersApplied(),
)

const isNoSystemsEmptyStateShown = computed(
  () =>
    !systemsState.value.data?.systems?.length &&
    systemsState.value.status === 'success' &&
    areDefaultFiltersApplied(),
)

const noEmptyStateShown = computed(
  () =>
    !isNoDataEmptyStateShown.value &&
    !isNoMatchEmptyStateShown.value &&
    !isNoSystemsEmptyStateShown.value,
)

// ── Filter options ─────────────────────────────────────────────────────────────

const alertNameFilterOptions = computed<FilterOption[]>(() => {
  const alerts = alertFiltersState.value.data?.alerts ?? []
  return alerts.map((a: AlertFilterAlert) => ({ id: a.name, label: a.name }))
})

const statusFilterOptions: FilterOption[] = [
  { id: 'active', label: t('alerts.unmuted') },
  { id: 'suppressed', label: t('alerts.muted') },
]

// ── Sort ────────────────────────────────────────────────────────────────────────

const sortDescending = computed({
  get: () => sortDirection.value === 'desc',
  set: (val: boolean) => {
    sortDirection.value = val ? 'desc' : 'asc'
  },
})

// ── Mute alert drawer ────────────────────────────────────────────────────────────

const selectedAlert = ref<Alert | undefined>(undefined)
const isMuteDrawerShown = ref(false)

function showMuteDrawer(alert: Alert) {
  selectedAlert.value = alert
  isMuteDrawerShown.value = true
}

// ── Unmute alert ────────────────────────────────────────────────────────────

const unmuteError = ref<string | null>(null)

async function handleUnmuteAlert(alert: Alert) {
  unmuteError.value = null
  try {
    const organizationId = alert.labels?.organization_id
    const silenceIds = getAlertSilenceIds(alert)
    if (!silenceIds.length) return

    // Delete all silences for this alert
    for (const silenceId of silenceIds) {
      await deleteAlertSilence(silenceId, organizationId)
    }

    // Record the target state so the alert shows as "processing" until the
    // backend reflects the unmute.
    setPendingAlertState(alert.fingerprint, organizationId ?? '', false)

    // Show success notification with delay (per frontend conventions)
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('alerts.alert_unmuted_successfully'),
      })
    }, 500)

    // Refetch alerts to reflect changes
    refetch()
  } catch (error) {
    console.error('Error unmuting alert:', error)
    unmuteError.value = t('alerts.cannot_unmute_alert')
  }
}

// ── Alert details drawer ──────────────────────────────────────────────────────

const detailsDrawerAlert = ref<Alert | undefined>(undefined)
const isDetailsDrawerShown = ref(false)

function showDetailsDrawer(alert: Alert) {
  detailsDrawerAlert.value = alert
  isDetailsDrawerShown.value = true
}

function getKebabMenuItems(alert: Alert): NeDropdownItem[] {
  const items: NeDropdownItem[] = []
  if (canManageSystems()) {
    if (isProcessing(alert)) {
      // In transition: disable both actions until the backend catches up.
      items.push(
        {
          id: 'muteAlert',
          label: t('alerts.mute_alert'),
          icon: faBellSlash,
          disabled: true,
          action: () => {},
        },
        {
          id: 'unmuteAlert',
          label: t('alerts.unmute_alert'),
          icon: faBell,
          disabled: true,
          action: () => {},
        },
      )
    } else if (isAlertSilenced(alert)) {
      items.push({
        id: 'unmuteAlert',
        label: t('alerts.unmute_alert'),
        icon: faBell,
        action: () => handleUnmuteAlert(alert),
      })
    } else {
      items.push({
        id: 'muteAlert',
        label: t('alerts.mute_alert'),
        icon: faBellSlash,
        action: () => showMuteDrawer(alert),
      })
    }
  }
  return items
}

// ── Navigation ──────────────────────────────────────────────────────────────────────

function goToSystems() {
  router.push({ name: 'systems' })
}
</script>

<template>
  <div>
    <!-- Error notification: data load -->
    <NeInlineNotification
      v-if="alertsState.status === 'error'"
      kind="error"
      :title="$t('alerts.cannot_retrieve_alerts')"
      class="mb-6"
    />

    <!-- Error notification: unmute -->
    <NeInlineNotification
      v-if="unmuteError"
      kind="error"
      :title="unmuteError"
      class="mb-6"
      @close="unmuteError = null"
    />

    <!-- Toolbar -->
    <div class="mb-6 flex items-center gap-4">
      <div class="flex w-full items-start justify-between gap-4">
        <!-- Filters -->
        <div class="flex flex-wrap items-center gap-4">
          <!-- Severity filter -->
          <NeDropdownFilter
            v-model="severityFilters"
            kind="checkbox"
            :label="t('alerts.severity')"
            :options="SEVERITY_FILTER_OPTIONS"
            :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
            :options-filter-placeholder="t('ne_dropdown_filter.options_filter_placeholder')"
            @update:model-value="() => (pageNum = 1)"
          />
          <!-- Alert name filter -->
          <NeDropdownFilter
            v-model="alertnameFilters"
            kind="checkbox"
            :label="t('alerts.alert')"
            :options="alertNameFilterOptions"
            show-options-filter
            :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
            :options-filter-placeholder="t('ne_dropdown_filter.options_filter_placeholder')"
            @update:model-value="() => (pageNum = 1)"
          />
          <!-- System filter -->
          <SystemDropdownFilter
            v-model="systemKeyFilters"
            id-field="system_key"
            @update:model-value="() => (pageNum = 1)"
          />
          <!-- Organization filter -->
          <OrganizationDropdownFilter
            v-if="!isUserCustomer()"
            v-model="organizationIds"
            @update:model-value="() => (pageNum = 1)"
          />
          <!-- Status filter -->
          <NeDropdownFilter
            v-model="statusFilters"
            kind="checkbox"
            :label="t('common.status')"
            :options="statusFilterOptions"
            :show-clear-filter="true"
            :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
            :options-filter-placeholder="t('ne_dropdown_filter.options_filter_placeholder')"
            @update:model-value="() => (pageNum = 1)"
          />
          <!-- Sort -->
          <NeSortDropdown
            v-model:sort-key="sortBy"
            v-model:sort-descending="sortDescending"
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
          <!-- Clear filters -->
          <NeButton kind="tertiary" @click="clearFilters">
            {{ t('common.clear_filters') }}
          </NeButton>
        </div>
        <!-- Data updated every X seconds -->
        <div class="flex items-center gap-2">
          <NeSpinner
            color="white"
            v-if="alertsAsyncStatus === 'loading' && alertsState.status !== 'pending'"
          />
          <div class="text-tertiary-neutral">
            {{ t('common.data_updated_every_seconds', { seconds: 10 }) }}
          </div>
        </div>
      </div>
    </div>

    <!-- Empty state: no systems configured -->
    <NeEmptyState
      v-if="isNoSystemsEmptyStateShown"
      :title="$t('alerts.no_systems_configured')"
      :icon="faServer"
      class="bg-white dark:bg-gray-950"
    >
      <NeButton kind="tertiary" @click="goToSystems">
        <template #prefix>
          <FontAwesomeIcon :icon="faArrowRight" aria-hidden="true" />
        </template>
        {{ $t('common.go_to_page', { page: $t('systems.title') }) }}
      </NeButton>
    </NeEmptyState>

    <!-- Empty state: no data -->
    <NeEmptyState
      v-else-if="isNoDataEmptyStateShown"
      :title="$t('alerts.no_active_alerts')"
      :description="$t('alerts.no_active_alerts_description')"
      :icon="faCircleCheck"
      class="bg-white dark:bg-gray-950"
    />

    <!-- Empty state: no matches -->
    <NeEmptyState
      v-else-if="isNoMatchEmptyStateShown"
      :title="$t('alerts.no_alerts_found')"
      :description="$t('common.try_changing_search_filters')"
      :icon="faMagnifyingGlass"
      class="bg-white dark:bg-gray-950"
    >
      <NeButton kind="tertiary" @click="clearFilters">
        {{ $t('common.clear_filters') }}
      </NeButton>
    </NeEmptyState>

    <!-- Alerts table -->
    <NeTable
      v-if="noEmptyStateShown"
      :sort-key="sortBy"
      :sort-descending="sortDirection === 'desc'"
      :aria-label="$t('alerts.title')"
      card-breakpoint="2xl"
      :loading="alertsState.status === 'pending'"
      :skeleton-columns="5"
      :skeleton-rows="7"
    >
      <NeTableHead>
        <NeTableHeadCell>{{ $t('alerts.severity') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('alerts.alertname') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('alerts.system') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('alerts.organization') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('alerts.started') }}</NeTableHeadCell>
        <NeTableHeadCell>
          <!-- no header for actions -->
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
          <!-- Alert -->
          <NeTableCell :data-label="$t('alerts.alertname')">
            <div class="flex items-start gap-2">
              <div>
                <span
                  class="cursor-pointer font-medium hover:underline"
                  @click="() => showDetailsDrawer(alert)"
                  >{{ alert.labels?.alertname || '-' }}</span
                >
                <p
                  v-if="getAlertSummary(alert, locale)"
                  class="text-tertiary-neutral dark:text-tertiary-neutral mt-0.5 break-all"
                >
                  {{ getAlertSummary(alert, locale) }}
                </p>
              </div>
              <ProcessingAlertBadge v-if="isProcessing(alert)" />
              <NeBadgeV2 v-else-if="isAlertSilenced(alert)" kind="gray">
                <FontAwesomeIcon :icon="faBellSlash" class="size-4" />
                {{ t('alerts.muted') }}
              </NeBadgeV2>
            </div>
          </NeTableCell>
          <!-- System -->
          <NeTableCell :data-label="$t('alerts.system')">
            <SystemLogoAndLink
              :system-id="alert.labels?.system_id"
              :system-name="alert.labels?.system_name"
              :system-type="alert.labels?.system_type"
            ></SystemLogoAndLink>
          </NeTableCell>
          <!-- Organization -->
          <NeTableCell :data-label="$t('alerts.organization')">
            <OrganizationIconAndLink
              v-if="alert.labels?.organization_name || alert.labels?.organization_id"
              :organization="{
                logto_id: alert.labels?.organization_id,
                name: alert.labels?.organization_name || alert.labels?.organization_id,
                type: alert.labels?.organization_type || '',
              }"
            />
            <span v-else>-</span>
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
            <div class="-ml-2.5 flex gap-2 2xl:ml-0 2xl:justify-end">
              <NeButton kind="tertiary" size="sm" @click="() => showDetailsDrawer(alert)">
                <template #prefix>
                  <FontAwesomeIcon :icon="faEye" class="h-4 w-4" aria-hidden="true" />
                </template>
                {{ $t('alerts.view_details') }}
              </NeButton>
              <!-- kebab menu -->
              <NeDropdown
                v-if="canManageSystems()"
                :items="getKebabMenuItems(alert)"
                :align-to-right="true"
              />
            </div>
          </NeTableCell>
        </NeTableRow>
      </NeTableBody>
      <template v-if="pagination" #paginator>
        <NePaginator
          :current-page="pageNum"
          :total-rows="pagination.total_count"
          :page-size="pageSize"
          :page-sizes="PAGE_SIZE_OPTIONS"
          :nav-pagination-label="$t('ne_table.pagination')"
          :next-label="$t('ne_table.go_to_next_page')"
          :previous-label="$t('ne_table.go_to_previous_page')"
          :range-of-total-label="$t('ne_table.of')"
          :page-size-label="$t('ne_table.show')"
          @select-page="(page: number) => (pageNum = page)"
          @select-page-size="
            (size: number) => {
              pageSize = size
              savePageSizeToStorage(ALERTS_TABLE_ID, size)
            }
          "
        />
      </template>
    </NeTable>

    <!-- Mute alert drawer -->
    <MuteAlertDrawer
      :is-shown="isMuteDrawerShown"
      :alert="selectedAlert"
      @close="isMuteDrawerShown = false"
    />

    <!-- Alert details drawer -->
    <AlertDetailsDrawer
      :is-shown="isDetailsDrawerShown"
      :alert="detailsDrawerAlert"
      @close="isDetailsDrawerShown = false"
    />
  </div>
</template>
