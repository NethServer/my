<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { RESELLERS_TABLE_ID, type Reseller } from '@/lib/organizations/resellers'
import {
  faCircleInfo,
  faCity,
  faPenToSquare,
  faBoxArchive,
  faCirclePause,
  faCirclePlay,
  faCircleCheck,
  faRotateLeft,
  faBomb,
  faServer,
  faBuilding,
  faEye,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeTable,
  NeTableHead,
  NeTableHeadCell,
  NeTableBody,
  NeTableRow,
  NeTableCell,
  NePaginator,
  NeButton,
  NeEmptyState,
  NeInlineNotification,
  NeTextInput,
  NeSpinner,
  NeDropdown,
  type SortEvent,
  NeSortDropdown,
  NeDropdownFilter,
  type FilterOption,
  type NeDropdownItem,
} from '@nethesis/vue-components'
import { computed, ref, watch } from 'vue'
import CreateOrEditResellerDrawer from './CreateOrEditResellerDrawer.vue'
import { useI18n } from 'vue-i18n'
import DeleteResellerModal from './DeleteResellerModal.vue'
import SuspendResellerModal from './SuspendResellerModal.vue'
import ReactivateResellerModal from './ReactivateResellerModal.vue'
import RestoreResellerModal from './RestoreResellerModal.vue'
import DestroyResellerModal from './DestroyResellerModal.vue'
import { savePageSizeToStorage } from '@/lib/tablePageSize'
import { useResellers } from '@/queries/organizations/resellers'
import { canManageResellers, canDestroyResellers } from '@/lib/permissions'
import router from '@/router'

const { isShownCreateResellerDrawer = false } = defineProps<{
  isShownCreateResellerDrawer: boolean
}>()

const emit = defineEmits(['close-drawer'])

const { t } = useI18n()
const {
  state,
  asyncStatus,
  pageNum,
  pageSize,
  textFilter,
  statusFilter,
  sortBy,
  sortDescending,
  areDefaultFiltersApplied,
  resetFilters,
} = useResellers()

const currentReseller = ref<Reseller | undefined>()
const isShownCreateOrEditResellerDrawer = ref(false)
const isShownDeleteResellerDrawer = ref(false)
const isShownSuspendResellerModal = ref(false)
const isShownReactivateResellerModal = ref(false)
const isShownRestoreResellerModal = ref(false)
const isShownDestroyResellerModal = ref(false)

const statusFilterOptions = ref<FilterOption[]>([
  {
    id: 'enabled',
    label: t('common.enabled'),
  },
  {
    id: 'suspended',
    label: t('common.suspended'),
  },
  {
    id: 'deleted',
    label: t('common.archived'),
  },
])

const resellersPage = computed(() => {
  return state.value.data?.resellers
})

const pagination = computed(() => {
  return state.value.data?.pagination
})

const isNoDataEmptyStateShown = computed(() => {
  return (
    !resellersPage.value?.length &&
    state.value.status === 'success' &&
    areDefaultFiltersApplied.value
  )
})

const isNoMatchEmptyStateShown = computed(() => {
  return (
    !resellersPage.value?.length &&
    state.value.status === 'success' &&
    !areDefaultFiltersApplied.value
  )
})

const noEmptyStateShown = computed(() => {
  return !isNoDataEmptyStateShown.value && !isNoMatchEmptyStateShown.value
})

watch(
  () => isShownCreateResellerDrawer,
  () => {
    if (isShownCreateResellerDrawer) {
      showCreateResellerDrawer()
    }
  },
  { immediate: true },
)

function showCreateResellerDrawer() {
  currentReseller.value = undefined
  isShownCreateOrEditResellerDrawer.value = true
}

function showEditResellerDrawer(reseller: Reseller) {
  currentReseller.value = reseller
  isShownCreateOrEditResellerDrawer.value = true
}

function showDeleteResellerDrawer(reseller: Reseller) {
  currentReseller.value = reseller
  isShownDeleteResellerDrawer.value = true
}

function showRestoreResellerModal(reseller: Reseller) {
  currentReseller.value = reseller
  isShownRestoreResellerModal.value = true
}

function showSuspendResellerModal(reseller: Reseller) {
  currentReseller.value = reseller
  isShownSuspendResellerModal.value = true
}

function showReactivateResellerModal(reseller: Reseller) {
  currentReseller.value = reseller
  isShownReactivateResellerModal.value = true
}

function showDestroyResellerModal(reseller: Reseller) {
  currentReseller.value = reseller
  isShownDestroyResellerModal.value = true
}

function onCloseDrawer() {
  isShownCreateOrEditResellerDrawer.value = false
  emit('close-drawer')
}

function getKebabMenuItems(reseller: Reseller) {
  const items: NeDropdownItem[] = []

  if (canManageResellers()) {
    if (!reseller.deleted_at) {
      items.push({
        id: 'editReseller',
        label: t('common.edit'),
        icon: faPenToSquare,
        action: () => showEditResellerDrawer(reseller),
        disabled: asyncStatus.value === 'loading',
      })
    }

    if (reseller.suspended_at) {
      items.push({
        id: 'reactivateReseller',
        label: t('common.reactivate'),
        icon: faCirclePlay,
        action: () => showReactivateResellerModal(reseller),
        disabled: asyncStatus.value === 'loading',
      })

      items.push({
        id: 'deleteReseller',
        label: t('common.archive'),
        icon: faBoxArchive,
        danger: true,
        action: () => showDeleteResellerDrawer(reseller),
        disabled: asyncStatus.value === 'loading',
      })
    } else if (reseller.deleted_at) {
      items.push({
        id: 'restoreReseller',
        label: t('common.restore'),
        icon: faRotateLeft,
        action: () => showRestoreResellerModal(reseller),
        disabled: asyncStatus.value === 'loading',
      })
    } else {
      items.push({
        id: 'suspendReseller',
        label: t('common.suspend'),
        icon: faCirclePause,
        action: () => showSuspendResellerModal(reseller),
        disabled: asyncStatus.value === 'loading',
      })

      items.push({
        id: 'deleteReseller',
        label: t('common.archive'),
        icon: faBoxArchive,
        danger: true,
        action: () => showDeleteResellerDrawer(reseller),
        disabled: asyncStatus.value === 'loading',
      })
    }
  }

  if (canDestroyResellers()) {
    items.push({
      id: 'destroyReseller',
      label: t('common.destroy'),
      icon: faBomb,
      danger: true,
      action: () => showDestroyResellerModal(reseller),
      disabled: asyncStatus.value === 'loading',
    })
  }
  return items
}

const onSort = (payload: SortEvent) => {
  sortBy.value = payload.key as keyof Reseller
  sortDescending.value = payload.descending
}

const goToResellerDetails = (reseller: Reseller) => {
  router.push({ name: 'reseller_detail', params: { companyId: reseller.logto_id } })
}
</script>

<template>
  <div>
    <!-- get resellers error notification -->
    <NeInlineNotification
      v-if="state.status === 'error'"
      kind="error"
      :title="$t('resellers.cannot_retrieve_resellers')"
      :description="state.error.message"
      class="mb-6"
    />
    <!-- table toolbar -->
    <div class="mb-6 flex items-center gap-4">
      <div class="flex w-full items-center justify-between gap-4">
        <!-- filters -->
        <div class="flex flex-wrap items-center gap-4">
          <!-- text filter -->
          <NeTextInput
            v-model.trim="textFilter"
            is-search
            :placeholder="$t('resellers.filter_resellers')"
            class="max-w-48 sm:max-w-sm"
          />
          <!-- status filter -->
          <NeDropdownFilter
            v-model="statusFilter"
            kind="checkbox"
            :label="t('common.status')"
            :options="statusFilterOptions"
            :show-clear-filter="false"
            :clear-filter-label="t('ne_dropdown_filter.reset_filter')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
          />
          <NeSortDropdown
            v-model:sort-key="sortBy"
            v-model:sort-descending="sortDescending"
            :label="t('sort.sort')"
            :options="[
              { id: 'name', label: t('organizations.name') },
              { id: 'suspended_at', label: t('common.status') },
            ]"
            :open-menu-aria-label="t('ne_dropdown.open_menu')"
            :sort-by-label="t('sort.sort_by')"
            :sort-direction-label="t('sort.direction')"
            :ascending-label="t('sort.ascending')"
            :descending-label="t('sort.descending')"
          />
          <NeButton kind="tertiary" @click="resetFilters">
            {{ t('common.reset_filters') }}
          </NeButton>
        </div>
        <!-- update indicator -->
        <div
          v-if="asyncStatus === 'loading' && state.status !== 'pending'"
          class="flex items-center gap-2"
        >
          <NeSpinner color="white" />
          <div class="text-gray-500 dark:text-gray-400">
            {{ $t('common.updating') }}
          </div>
        </div>
      </div>
    </div>
    <!-- empty state -->
    <NeEmptyState
      v-if="isNoDataEmptyStateShown"
      :title="$t('resellers.no_reseller')"
      :icon="faCity"
      class="bg-white dark:bg-gray-950"
    />
    <!-- no reseller matching filter -->
    <NeEmptyState
      v-else-if="isNoMatchEmptyStateShown"
      :title="$t('resellers.no_reseller_found')"
      :description="$t('common.try_changing_search_filters')"
      :icon="faCircleInfo"
      class="bg-white dark:bg-gray-950"
    >
      <NeButton kind="tertiary" @click="resetFilters">
        {{ $t('common.reset_filters') }}
      </NeButton>
    </NeEmptyState>
    <NeTable
      v-if="noEmptyStateShown"
      :sort-key="sortBy"
      :sort-descending="sortDescending"
      :aria-label="$t('resellers.title')"
      card-breakpoint="xl"
      :loading="state.status === 'pending'"
      :skeleton-columns="5"
      :skeleton-rows="7"
    >
      <NeTableHead>
        <NeTableHeadCell sortable column-key="name" @sort="onSort">{{
          $t('organizations.name')
        }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('organizations.vat_number') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('customers.title') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('resellers.total_systems') }}</NeTableHeadCell>
        <NeTableHeadCell sortable column-key="suspended_at" @sort="onSort">{{
          $t('common.status')
        }}</NeTableHeadCell>
        <NeTableHeadCell>
          <!-- no header for actions -->
        </NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <NeTableRow v-for="(item, index) in resellersPage" :key="index">
          <NeTableCell :data-label="$t('organizations.name')">
            <router-link
              v-if="!item.deleted_at"
              :to="{ name: 'reseller_detail', params: { companyId: item.logto_id } }"
              class="cursor-pointer font-medium hover:underline"
            >
              {{ item.name }}
            </router-link>
            <span v-else class="opacity-50">
              {{ item.name }}
            </span>
          </NeTableCell>
          <NeTableCell
            :data-label="$t('organizations.vat_number')"
            :class="{ 'opacity-50': item.deleted_at }"
          >
            {{ item.custom_data?.vat || '-' }}
          </NeTableCell>
          <NeTableCell :data-label="$t('customers.title')">
            <div class="flex items-center gap-2" :class="{ 'opacity-50': item.deleted_at }">
              <FontAwesomeIcon
                :icon="faBuilding"
                class="size-4 text-gray-700 dark:text-gray-400"
                aria-hidden="true"
              />
              {{ item.customers_count }}
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('resellers.total_systems')">
            <div class="flex items-center gap-2" :class="{ 'opacity-50': item.deleted_at }">
              <FontAwesomeIcon
                :icon="faServer"
                class="size-4 text-gray-700 dark:text-gray-400"
                aria-hidden="true"
              />
              {{ item.systems_count }}
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('common.status')">
            <div class="flex items-center gap-2">
              <template v-if="item.deleted_at">
                <FontAwesomeIcon
                  :icon="faBoxArchive"
                  class="size-4 text-gray-700 dark:text-gray-400"
                  aria-hidden="true"
                />
                <span>
                  {{ t('common.archived') }}
                </span>
              </template>
              <template v-else-if="item.suspended_at">
                <FontAwesomeIcon
                  :icon="faCirclePause"
                  class="size-4 text-gray-700 dark:text-gray-400"
                  aria-hidden="true"
                />
                <span>
                  {{ t('common.suspended') }}
                </span>
              </template>
              <template v-else>
                <FontAwesomeIcon
                  :icon="faCircleCheck"
                  class="size-4 text-green-600 dark:text-green-400"
                  aria-hidden="true"
                />
                <span>
                  {{ t('common.enabled') }}
                </span>
              </template>
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('common.actions')">
            <div class="-ml-2.5 flex gap-2 xl:ml-0 xl:justify-end">
              <NeButton v-if="!item.deleted_at" kind="tertiary" @click="goToResellerDetails(item)">
                <template #prefix>
                  <FontAwesomeIcon :icon="faEye" class="h-4 w-4" aria-hidden="true" />
                </template>
                {{ $t('common.view') }}
              </NeButton>
              <!-- kebab menu -->
              <NeDropdown
                v-if="canManageResellers() || canDestroyResellers()"
                :items="getKebabMenuItems(item)"
                :align-to-right="true"
              />
            </div>
          </NeTableCell>
        </NeTableRow>
      </NeTableBody>
      <template #paginator>
        <NePaginator
          :current-page="pageNum"
          :total-rows="pagination?.total_count || 0"
          :page-size="pageSize"
          :page-sizes="[5, 10, 25, 50, 100]"
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
              savePageSizeToStorage(RESELLERS_TABLE_ID, size)
            }
          "
        />
      </template>
    </NeTable>
    <!-- side drawer -->
    <CreateOrEditResellerDrawer
      :is-shown="isShownCreateOrEditResellerDrawer"
      :current-reseller="currentReseller"
      @close="onCloseDrawer"
    />
    <!-- delete reseller modal -->
    <DeleteResellerModal
      :visible="isShownDeleteResellerDrawer"
      :reseller="currentReseller"
      @close="isShownDeleteResellerDrawer = false"
    />
    <!-- suspend reseller modal -->
    <SuspendResellerModal
      :visible="isShownSuspendResellerModal"
      :reseller="currentReseller"
      @close="isShownSuspendResellerModal = false"
    />
    <!-- reactivate reseller modal -->
    <ReactivateResellerModal
      :visible="isShownReactivateResellerModal"
      :reseller="currentReseller"
      @close="isShownReactivateResellerModal = false"
    />
    <!-- restore reseller modal -->
    <RestoreResellerModal
      :visible="isShownRestoreResellerModal"
      :reseller="currentReseller"
      @close="isShownRestoreResellerModal = false"
    />
    <!-- destroy reseller modal -->
    <DestroyResellerModal
      :visible="isShownDestroyResellerModal"
      :reseller="currentReseller"
      @close="isShownDestroyResellerModal = false"
    />
  </div>
</template>
