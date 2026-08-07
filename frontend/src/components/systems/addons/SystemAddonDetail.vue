<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Where one add-on stands on this system: a row per application instance on a
  NethServer cluster, a single row for a NethSecurity service, which covers the
  whole firewall and so shows the system itself in the Application column.

  The grant endpoint takes no page, search or sort parameters and returns the
  system's whole list at once, so the filtering, sorting and paging below all
  happen here.
-->

<script setup lang="ts">
import {
  faArrowLeft,
  faArrowUpRightFromSquare,
  faBan,
  faCircleCheck,
  faMagnifyingGlass,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeButton,
  NeDropdown,
  NeDropdownFilterV2,
  NeEmptyState,
  NePaginator,
  NeSkeleton,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
  NeTextInput,
  type NeDropdownFilterV2Option,
  type NeDropdownItem,
} from '@nethesis/vue-components'
import { useDebounceFn } from '@vueuse/core'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ApplicationLogo from '@/components/applications/ApplicationLogo.vue'
import UpdatingSpinner from '@/components/common/UpdatingSpinner.vue'
import SystemLogo from '@/components/systems/SystemLogo.vue'
import UserAvatar from '@/components/users/UserAvatar.vue'
import {
  ADDON_ROW_STATUSES,
  SYSTEM_ADDONS_TABLE_ID,
  getBuyUrl,
  getOrderNumber,
  getOrderUrl,
  getRowStatus,
  type SystemAddonRow,
} from '@/lib/addons/systemAddons'
import { MIN_SEARCH_LENGTH } from '@/lib/common'
import { formatDateNoTime } from '@/lib/dateTime'
import { canBuyAddons, isAddonAdmin } from '@/lib/permissions'
import {
  DEFAULT_PAGE_SIZE,
  PAGE_SIZE_OPTIONS,
  loadPageSizeFromStorage,
  savePageSizeToStorage,
} from '@/lib/tablePageSize'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { useLoginStore } from '@/stores/login'
import AddonActionModal, { type AddonAction } from './AddonActionModal.vue'
import AddonStatusIcon from './AddonStatusIcon.vue'

const { rows, loading, refreshing } = defineProps<{
  addonId: string
  rows: SystemAddonRow[]
  loading: boolean
  refreshing: boolean
}>()

const emit = defineEmits<{ back: [] }>()

const { t, locale } = useI18n()
const loginStore = useLoginStore()
const { state: systemDetail } = useSystemDetail()

const textFilter = ref('')
const debouncedTextFilter = ref('')
const purchaserFilter = ref<NeDropdownFilterV2Option[]>([])
const statusFilter = ref<NeDropdownFilterV2Option[]>([])
const pageNum = ref(1)
const pageSize = ref(DEFAULT_PAGE_SIZE)

const currentRow = ref<SystemAddonRow | undefined>()
const currentAction = ref<AddonAction>('activate')
const isShownActionModal = ref(false)

const addonName = computed(() => rows[0]?.addon.display_name ?? '')

// A NethSecurity service covers the whole firewall rather than any one
// application, so the first column names the system instead. Decided from the
// rows rather than from the product, because an nsec service granted to a
// cluster reads the same way.
const isSystemScoped = computed(() => rows.every((row) => !row.applicationId))

const firstColumnLabel = computed(() =>
  isSystemScoped.value ? t('systems.system') : t('addons.application'),
)

// Filtering one row is busywork: a firewall service has exactly one, and so
// does a cluster running a single instance of the application.
const areFiltersShown = computed(() => rows.length > 1)

// A suspended or deleted system cannot use anything it holds — collect turns
// its credentials away before it ever looks at the grants — so buying more is
// pointless while it stays that way.
const isSystemBlocked = computed(() =>
  ['suspended', 'deleted'].includes(systemDetail.value.data?.status ?? ''),
)

const getPurchaserName = (row: SystemAddonRow) => {
  const purchaser = row.grant?.purchased_by

  if (!purchaser || purchaser.out_of_scope) {
    return ''
  }
  return purchaser.name ?? purchaser.email ?? ''
}

const purchaserFilterOptions = computed((): NeDropdownFilterV2Option[] => {
  const names = new Set(rows.map(getPurchaserName).filter((name) => !!name))

  return [...names].sort().map((name) => ({ id: name, label: name }))
})

const statusFilterOptions = computed((): NeDropdownFilterV2Option[] =>
  ADDON_ROW_STATUSES.map((status) => ({ id: status, label: t(`addons.status_${status}`) })),
)

const filteredRows = computed(() => {
  const search = debouncedTextFilter.value.trim().toLowerCase()
  const purchasers = purchaserFilter.value.map((option) => option.id)
  const statuses = statusFilter.value.map((option) => option.id)

  return rows.filter((row) => {
    if (purchasers.length && !purchasers.includes(getPurchaserName(row))) {
      return false
    }
    if (statuses.length && !statuses.includes(getRowStatus(row))) {
      return false
    }
    if (search && !row.applicationLabel.toLowerCase().includes(search)) {
      return false
    }
    return true
  })
})

const paginatedRows = computed(() =>
  filteredRows.value.slice((pageNum.value - 1) * pageSize.value, pageNum.value * pageSize.value),
)

const isFiltered = computed(
  () =>
    !!debouncedTextFilter.value || !!purchaserFilter.value.length || !!statusFilter.value.length,
)

const isNoMatchEmptyStateShown = computed(
  () => !loading && !filteredRows.value.length && isFiltered.value,
)

// load table page size from storage
watch(
  () => loginStore.userInfo?.email,
  (email) => {
    if (email) {
      pageSize.value = loadPageSizeFromStorage(SYSTEM_ADDONS_TABLE_ID)
    }
  },
  { immediate: true },
)

watch(
  () => textFilter.value,
  useDebounceFn(() => {
    // debounce and ignore if text filter is too short
    if (textFilter.value.length === 0 || textFilter.value.length >= MIN_SEARCH_LENGTH) {
      debouncedTextFilter.value = textFilter.value

      // reset to first page when filter changes
      pageNum.value = 1
    }
  }, 500),
)

// reset to first page when page size or any filter changes
watch([() => pageSize.value, () => purchaserFilter.value, () => statusFilter.value], () => {
  pageNum.value = 1
})

// a narrower filter can leave the current page past the end
watch([filteredRows, pageSize], () => {
  const lastPage = Math.max(1, Math.ceil(filteredRows.value.length / pageSize.value))

  if (pageNum.value > lastPage) {
    pageNum.value = lastPage
  }
})

function clearFilters() {
  textFilter.value = ''
  debouncedTextFilter.value = ''
  purchaserFilter.value = []
  statusFilter.value = []
}

function showActionModal(row: SystemAddonRow, action: AddonAction) {
  currentRow.value = row
  currentAction.value = action
  isShownActionModal.value = true
}

// Nothing is offered on a blocked system, and nothing can be bought that the
// company is not allowed to have in the first place.
function canBuy(row: SystemAddonRow) {
  if (!canBuyAddons() || isSystemBlocked.value) {
    return false
  }
  if (!row.grant) {
    return true
  }
  // a shop-side revocation (cancelled subscription, failed payment) can be
  // undone by buying again; a deliberate one cannot
  return (
    row.grant.status === 'expired' ||
    (row.grant.status === 'revoked' && row.grant.revoked_source === 'shop')
  )
}

function getKebabMenuItems(row: SystemAddonRow): NeDropdownItem[] {
  const items: NeDropdownItem[] = []

  if (!isAddonAdmin() || isSystemBlocked.value) {
    return items
  }

  if (!row.grant) {
    items.push({
      id: 'activate',
      label: t('addons.activate'),
      icon: faCircleCheck,
      action: () => showActionModal(row, 'activate'),
    })
  } else if (row.grant.status === 'active') {
    items.push({
      id: 'revoke',
      label: t('addons.revoke'),
      icon: faBan,
      danger: true,
      action: () => showActionModal(row, 'revoke'),
    })
  } else if (row.grant.status === 'revoked') {
    // restoring only clears the revocation: on an expired grant it would
    // change nothing, so it is not offered there
    items.push({
      id: 'reactivate',
      label: t('addons.reactivate'),
      icon: faCircleCheck,
      action: () => showActionModal(row, 'reactivate'),
    })
  }

  return items
}

function openShop(row: SystemAddonRow) {
  window.open(getBuyUrl(systemDetail.value.data?.system_key ?? '', row), '_blank')
}

// Dates only: a licence period is counted in days, so the time of day is noise.
function formatValidity(row: SystemAddonRow) {
  // A grant awaiting payment has no period yet — the backend marks the stub by
  // setting valid_until to valid_from, which would otherwise read as a licence
  // that expired the day it started.
  if (!row.grant || row.grant.status === 'pending') {
    return '-'
  }
  const from = formatDateNoTime(new Date(row.grant.valid_from), locale.value)
  const until = row.grant.valid_until
    ? formatDateNoTime(new Date(row.grant.valid_until), locale.value)
    : t('addons.never_expires')

  return `${from} - ${until}`
}
</script>

<template>
  <div>
    <!-- back link + add-on name -->
    <div class="mb-8 flex flex-col items-start gap-2">
      <NeButton kind="tertiary" class="-ml-2.5" @click="emit('back')">
        <template #prefix>
          <FontAwesomeIcon :icon="faArrowLeft" class="size-4" aria-hidden="true" />
        </template>
        {{ $t('addons.back_to_addons') }}
      </NeButton>
      <div class="flex w-full items-center justify-between gap-4">
        <NeSkeleton v-if="loading" size="lg" class="w-xs" />
        <h4 v-else class="text-lg font-medium text-gray-900 dark:text-gray-100">{{ addonName }}</h4>
        <!-- with no filter row to host it, the indicator sits next to the title:
             it is shorter than the heading, so appearing shifts nothing -->
        <UpdatingSpinner v-if="!areFiltersShown && refreshing" />
      </div>
    </div>
    <!-- filters: dropped for a single row, where they would be busywork -->
    <div v-if="areFiltersShown" class="mb-6 flex w-full items-end justify-between gap-4">
      <div class="flex flex-wrap items-center gap-4">
        <NeTextInput
          v-model="textFilter"
          is-search
          :placeholder="$t('addons.filter_applications')"
          class="max-w-48 sm:max-w-sm"
          @blur="textFilter = textFilter.trim()"
        />
        <NeDropdownFilterV2
          v-model="purchaserFilter"
          kind="checkbox"
          :label="t('addons.purchased_by')"
          :options="purchaserFilterOptions"
          show-options-filter
          :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
          :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
          :no-options-label="t('ne_dropdown_filter.no_options')"
          :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
          :clear-search-label="t('ne_dropdown_filter.clear_search')"
          :options-filter-placeholder="t('ne_dropdown_filter.options_filter_placeholder')"
        />
        <NeDropdownFilterV2
          v-model="statusFilter"
          kind="checkbox"
          :label="t('addons.status')"
          :options="statusFilterOptions"
          :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
          :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
          :no-options-label="t('ne_dropdown_filter.no_options')"
          :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
          :clear-search-label="t('ne_dropdown_filter.clear_search')"
          :options-filter-placeholder="t('ne_dropdown_filter.options_filter_placeholder')"
        />
        <NeButton kind="tertiary" @click="clearFilters">
          {{ t('common.clear_filters') }}
        </NeButton>
      </div>
      <UpdatingSpinner v-if="refreshing" />
    </div>
    <!-- no application matching filter -->
    <NeEmptyState
      v-if="isNoMatchEmptyStateShown"
      :title="$t('addons.no_applications_found')"
      :description="$t('common.try_changing_search_filters')"
      :icon="faMagnifyingGlass"
      class="bg-white dark:bg-gray-950"
    >
      <NeButton kind="tertiary" @click="clearFilters">
        {{ $t('common.clear_filters') }}
      </NeButton>
    </NeEmptyState>
    <NeTable
      v-else
      :aria-label="addonName"
      card-breakpoint="2xl"
      :loading="loading"
      :skeleton-columns="6"
      :skeleton-rows="7"
    >
      <NeTableHead>
        <NeTableHeadCell>{{ firstColumnLabel }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('addons.order') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('addons.purchased_by') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('addons.validity') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('addons.status') }}</NeTableHeadCell>
        <NeTableHeadCell>
          <!-- no header for actions -->
        </NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <NeTableRow v-for="row in paginatedRows" :key="`${row.addon.id} ${row.scope}`">
          <NeTableCell :data-label="firstColumnLabel">
            <div class="flex items-center gap-2">
              <ApplicationLogo v-if="row.applicationId" :app="row.applicationId" />
              <SystemLogo v-else :system="systemDetail.data?.type ?? 'nsec'" />
              <span>{{ row.applicationLabel }}</span>
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('addons.order')">
            <!-- with no grant there is no order to name: the Status column is
                 where "not purchased" is said -->
            <span v-if="!row.grant">-</span>
            <a
              v-else-if="getOrderNumber(row.grant)"
              :href="getOrderUrl(row.grant, isAddonAdmin())"
              target="_blank"
              rel="noopener noreferrer"
              class="text-primary-700 dark:text-primary-500 hover:underline"
            >
              #{{ getOrderNumber(row.grant) }}
            </a>
            <span v-else class="text-tertiary-neutral italic">
              {{ $t('addons.manually_created') }}
            </span>
          </NeTableCell>
          <NeTableCell :data-label="$t('addons.purchased_by')">
            <div v-if="getPurchaserName(row)" class="flex items-center gap-2">
              <UserAvatar
                size="sm"
                :is-owner="false"
                :name="getPurchaserName(row)"
                :logto-id="row.grant?.purchased_by?.logto_id ?? ''"
              />
              <span>{{ getPurchaserName(row) }}</span>
            </div>
            <span v-else-if="row.grant?.purchased_by?.out_of_scope" class="text-tertiary-neutral">
              {{ $t('addons.purchaser_not_visible') }}
            </span>
            <div v-else-if="row.grant?.created_by?.user_name" class="flex items-center gap-2">
              <UserAvatar
                size="sm"
                :is-owner="false"
                :name="row.grant.created_by.user_name"
                :logto-id="row.grant.created_by.user_id ?? ''"
              />
              <div>
                <div class="text-tertiary-neutral text-xs">{{ $t('addons.created_by') }}</div>
                <div>{{ row.grant.created_by.user_name }}</div>
              </div>
            </div>
            <span v-else>-</span>
          </NeTableCell>
          <NeTableCell :data-label="$t('addons.validity')">
            {{ formatValidity(row) }}
          </NeTableCell>
          <NeTableCell :data-label="$t('addons.status')">
            <AddonStatusIcon :status="getRowStatus(row)" />
          </NeTableCell>
          <NeTableCell :data-label="$t('common.actions')">
            <div class="-ml-2.5 flex gap-2 2xl:ml-0 2xl:justify-end">
              <!-- when buying is the only thing on offer it gets a button of its
                   own rather than hiding inside a one-item kebab -->
              <NeButton
                v-if="canBuy(row) && !getKebabMenuItems(row).length"
                kind="tertiary"
                @click="openShop(row)"
              >
                <template #prefix>
                  <FontAwesomeIcon
                    :icon="faArrowUpRightFromSquare"
                    class="size-4"
                    aria-hidden="true"
                  />
                </template>
                {{ $t('addons.buy') }}
              </NeButton>
              <NeDropdown
                v-else-if="getKebabMenuItems(row).length || canBuy(row)"
                :items="[
                  ...getKebabMenuItems(row),
                  ...(canBuy(row)
                    ? [
                        {
                          id: 'buy',
                          label: t('addons.buy'),
                          icon: faArrowUpRightFromSquare,
                          action: () => openShop(row),
                        },
                      ]
                    : []),
                ]"
                :align-to-right="true"
                :open-menu-aria-label="$t('ne_dropdown.open_menu')"
              />
            </div>
          </NeTableCell>
        </NeTableRow>
      </NeTableBody>
      <template #paginator>
        <NePaginator
          :current-page="pageNum"
          :total-rows="filteredRows.length"
          :page-size="pageSize"
          :page-sizes="PAGE_SIZE_OPTIONS"
          :nav-pagination-label="$t('ne_table.pagination')"
          :next-label="$t('ne_table.go_to_next_page')"
          :previous-label="$t('ne_table.go_to_previous_page')"
          :range-of-total-label="$t('ne_table.of')"
          :page-size-label="$t('ne_table.show')"
          @select-page="
            (page: number) => {
              pageNum = page
            }
          "
          @select-page-size="
            (size: number) => {
              pageSize = size
              savePageSizeToStorage(SYSTEM_ADDONS_TABLE_ID, size)
            }
          "
        />
      </template>
    </NeTable>
    <!-- activate / revoke / reactivate modal -->
    <AddonActionModal
      :visible="isShownActionModal"
      :action="currentAction"
      :row="currentRow"
      @close="isShownActionModal = false"
    />
  </div>
</template>
