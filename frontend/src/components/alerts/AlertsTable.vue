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
  NeTooltip,
  type FilterOption,
  type NeDropdownItem,
} from '@nethesis/vue-components'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAlerts } from '@/queries/alerts/alerts'
import { useAlertFilters } from '@/queries/alerts/alertFilters'
import {
  deleteAlertSilence,
  getAlertSilenceIds,
  getAlertSummary,
  getSeverityBadgeKind,
  isAlertSilenced,
  type Alert,
} from '@/lib/alerts'
import { useNotificationsStore } from '@/stores/notifications'
import { formatDateTime, formatTimeAgo } from '@/lib/dateTime'
import { canManageSystems } from '@/lib/permissions'
import UpdatingSpinner from '@/components/UpdatingSpinner.vue'
import SystemLogo from '@/components/systems/SystemLogo.vue'
import OrganizationIcon from '@/components/organizations/OrganizationIcon.vue'
import OrganizationLink from '@/components/applications/OrganizationLink.vue'
import MuteAlertDrawer from '@/components/alerts/MuteAlertDrawer.vue'
import AlertDetailsDrawer from '@/components/alerts/AlertDetailsDrawer.vue'
import capitalize from 'lodash/capitalize'

const { t, locale } = useI18n()
const notificationsStore = useNotificationsStore()

const {
  state,
  asyncStatus,
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
  resetFilters,
  refetch,
} = useAlerts()

const { state: alertFiltersState } = useAlertFilters()

const alerts = computed(() => state.value.data?.alerts ?? [])
const pagination = computed(() => state.value.data?.pagination)

const isNoDataEmptyStateShown = computed(
  () => !alerts.value.length && state.value.status === 'success' && areDefaultFiltersApplied(),
)

const isNoMatchEmptyStateShown = computed(
  () => !alerts.value.length && state.value.status === 'success' && !areDefaultFiltersApplied(),
)

const noEmptyStateShown = computed(
  () => !isNoDataEmptyStateShown.value && !isNoMatchEmptyStateShown.value,
)

// ── Filter options ─────────────────────────────────────────────────────────────

const alertFiltersData = computed(() => alertFiltersState.value.data)

const severityFilterOptions = computed<FilterOption[]>(() => {
  const severities = alertFiltersData.value?.severities ?? []
  return severities.map((s) => ({ id: s, label: capitalize(s) }))
})

const alertNameFilterOptions = computed<FilterOption[]>(() => {
  const alerts = alertFiltersData.value?.alerts ?? []
  return alerts.map((a) => ({ id: a.name, label: a.name }))
})

const systemFilterOptions = computed<FilterOption[]>(() => {
  const systems = alertFiltersData.value?.systems ?? []
  return systems.map((s) => ({ id: s.key, label: s.name }))
})

const organizationFilterOptions = computed<FilterOption[]>(() => {
  const orgs = alertFiltersData.value?.organizations ?? []
  return orgs.map((o) => ({ id: o.logto_id, label: o.name }))
})

const statusFilterOptions: FilterOption[] = [
  { id: 'muted', label: t('alerts.filter_status_muted') },
  { id: 'unmuted', label: t('alerts.filter_status_unmuted') },
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
    const silenceIds = getAlertSilenceIds(alert)
    if (!silenceIds.length) return

    const organizationId = alert.labels?.organization_id

    // Delete all silences for this alert
    for (const silenceId of silenceIds) {
      await deleteAlertSilence(silenceId, organizationId)
    }

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
    if (isAlertSilenced(alert)) {
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

// ── Reload ──────────────────────────────────────────────────────────────────────

function handleReload() {
  refetch()
}
</script>

<template>
  <div>
    <!-- Error notification: data load -->
    <NeInlineNotification
      v-if="state.status === 'error'"
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
      <div class="flex w-full items-end justify-between gap-4">
        <!-- Filters -->
        <div class="flex flex-wrap items-center gap-4">
          <!-- Severity filter -->
          <NeDropdownFilter
            v-model="severityFilters"
            kind="checkbox"
            :label="t('alerts.filter_severity')"
            :options="severityFilterOptions"
            :show-clear-filter="false"
            :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
            @update:model-value="() => (pageNum = 1)"
          />
          <!-- Alert name filter -->
          <NeDropdownFilter
            v-model="alertnameFilters"
            kind="checkbox"
            :label="t('alerts.filter_alert')"
            :options="alertNameFilterOptions"
            :show-clear-filter="false"
            :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
            @update:model-value="() => (pageNum = 1)"
          />
          <!-- System filter -->
          <NeDropdownFilter
            v-model="systemKeyFilters"
            kind="checkbox"
            :label="t('alerts.filter_system')"
            :options="systemFilterOptions"
            :show-clear-filter="false"
            :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
            @update:model-value="() => (pageNum = 1)"
          />
          <!-- Organization filter -->
          <NeDropdownFilter
            v-model="organizationIds"
            kind="checkbox"
            :label="t('alerts.filter_organization')"
            :options="organizationFilterOptions"
            :show-clear-filter="false"
            :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
            @update:model-value="() => (pageNum = 1)"
          />
          <!-- Status filter -->
          <NeDropdownFilter
            v-model="statusFilters"
            kind="checkbox"
            :label="t('alerts.filter_status')"
            :options="statusFilterOptions"
            :show-clear-filter="false"
            :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
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
          <!-- Reset filters -->
          <NeButton kind="tertiary" @click="resetFilters">
            {{ t('common.reset_filters') }}
          </NeButton>
        </div>
        <!-- Right-side actions -->
        <div class="flex items-center gap-3">
          <!-- Update indicator -->
          <UpdatingSpinner v-if="asyncStatus === 'loading' && state.status !== 'pending'" />
          <!-- Reload button -->
          <NeButton kind="secondary" size="md" @click="handleReload">
            <template #prefix>
              <FontAwesomeIcon :icon="faArrowsRotate" class="h-4 w-4" aria-hidden="true" />
            </template>
            {{ t('alerts.reload_alerts') }}
          </NeButton>
        </div>
      </div>
    </div>

    <!-- Empty state: no data -->
    <NeEmptyState
      v-if="isNoDataEmptyStateShown"
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
      <NeButton kind="tertiary" @click="resetFilters">
        {{ $t('common.reset_filters') }}
      </NeButton>
    </NeEmptyState>

    <!-- Alerts table -->
    <NeTable
      v-if="noEmptyStateShown"
      :sort-key="sortBy"
      :sort-descending="sortDirection === 'desc'"
      :aria-label="$t('alerts.title')"
      card-breakpoint="2xl"
      :loading="state.status === 'pending'"
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
                <p class="font-medium">{{ alert.labels?.alertname || '-' }}</p>
                <p
                  v-if="getAlertSummary(alert, locale)"
                  class="text-tertiary-neutral dark:text-tertiary-neutral mt-0.5 break-all"
                >
                  {{ getAlertSummary(alert, locale) }}
                </p>
              </div>
              <NeBadgeV2 v-if="alert.status.state === 'suppressed'" kind="gray">
                <FontAwesomeIcon :icon="faBellSlash" class="size-4" />
                {{ t('alerts.muted') }}
              </NeBadgeV2>
            </div>
          </NeTableCell>
          <!-- System -->
          <NeTableCell :data-label="$t('alerts.system')">
            <router-link
              v-if="alert.labels?.system_id"
              :to="{ name: 'system_detail', params: { systemId: alert.labels.system_id } }"
              class="cursor-pointer font-medium hover:underline"
            >
              <div class="flex items-center gap-2">
                <SystemLogo :system="alert.labels?.system_type" />
                {{ alert.labels.system_name || alert.labels.system_key }}
              </div>
            </router-link>
            <span v-else>-</span>
          </NeTableCell>
          <!-- Organization -->
          <NeTableCell :data-label="$t('alerts.organization')">
            <div
              v-if="alert.labels?.organization_name || alert.labels?.organization_id"
              class="flex items-center gap-2"
            >
              <NeTooltip
                v-if="alert.labels?.organization_type"
                placement="top"
                trigger-event="mouseenter focus"
                class="shrink-0"
              >
                <template #trigger>
                  <OrganizationIcon :org-type="alert.labels.organization_type" size="sm" />
                </template>
                <template #content>
                  {{ t(`organizations.${alert.labels.organization_type}`) }}
                </template>
              </NeTooltip>
              <OrganizationLink
                :organization="{
                  logto_id: alert.labels?.organization_id,
                  name: alert.labels?.organization_name || alert.labels?.organization_id || '-',
                  type: alert.labels?.organization_type || '',
                }"
              />
            </div>
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
              <NeDropdown :items="getKebabMenuItems(alert)" :align-to-right="true" />
            </div>
          </NeTableCell>
        </NeTableRow>
      </NeTableBody>
      <template v-if="pagination" #paginator>
        <NePaginator
          :current-page="pageNum"
          :total-rows="pagination.total_count"
          :page-size="pageSize"
          :page-sizes="[10, 25, 50, 100]"
          :nav-pagination-label="$t('ne_table.pagination')"
          :next-label="$t('ne_table.go_to_next_page')"
          :previous-label="$t('ne_table.go_to_previous_page')"
          :range-of-total-label="$t('ne_table.of')"
          :page-size-label="$t('ne_table.show')"
          @select-page="(page: number) => (pageNum = page)"
          @select-page-size="
            (size: number) => {
              pageSize = size
              pageNum = 1
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
