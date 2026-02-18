<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { DISTRIBUTORS_TABLE_ID, type Distributor } from '@/lib/organizations/distributors'
import {
  faCircleInfo,
  faGlobe,
  faPenToSquare,
  faBoxArchive,
  faCirclePause,
  faCirclePlay,
  faCircleCheck,
  faRotateLeft,
  faBomb,
  faServer,
  faCity,
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
import CreateOrEditDistributorDrawer from './CreateOrEditDistributorDrawer.vue'
import { useI18n } from 'vue-i18n'
import DeleteDistributorModal from './DeleteDistributorModal.vue'
import SuspendDistributorModal from './SuspendDistributorModal.vue'
import ReactivateDistributorModal from './ReactivateDistributorModal.vue'
import RestoreDistributorModal from './RestoreDistributorModal.vue'
import DestroyDistributorModal from './DestroyDistributorModal.vue'
import { savePageSizeToStorage } from '@/lib/tablePageSize'
import { useDistributors } from '@/queries/organizations/distributors'
import { canDestroyDistributors, canManageDistributors } from '@/lib/permissions'
import router from '@/router'

const { isShownCreateDistributorDrawer = false } = defineProps<{
  isShownCreateDistributorDrawer: boolean
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
} = useDistributors()

const currentDistributor = ref<Distributor | undefined>()
const isShownCreateOrEditDistributorDrawer = ref(false)
const isShownDeleteDistributorDrawer = ref(false)
const isShownSuspendDistributorModal = ref(false)
const isShownReactivateDistributorModal = ref(false)
const isShownRestoreDistributorModal = ref(false)
const isShownDestroyDistributorModal = ref(false)

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

const distributorsPage = computed(() => {
  return state.value.data?.distributors
})

const pagination = computed(() => {
  return state.value.data?.pagination
})

const isNoDataEmptyStateShown = computed(() => {
  return (
    !distributorsPage.value?.length &&
    state.value.status === 'success' &&
    areDefaultFiltersApplied.value
  )
})

const isNoMatchEmptyStateShown = computed(() => {
  return (
    !distributorsPage.value?.length &&
    state.value.status === 'success' &&
    !areDefaultFiltersApplied.value
  )
})

const noEmptyStateShown = computed(() => {
  return !isNoDataEmptyStateShown.value && !isNoMatchEmptyStateShown.value
})

watch(
  () => isShownCreateDistributorDrawer,
  () => {
    if (isShownCreateDistributorDrawer) {
      showCreateDistributorDrawer()
    }
  },
  { immediate: true },
)

function resetFilters() {
  textFilter.value = ''
  statusFilter.value = ['enabled', 'suspended']
}

function showCreateDistributorDrawer() {
  currentDistributor.value = undefined
  isShownCreateOrEditDistributorDrawer.value = true
}

function showEditDistributorDrawer(distributor: Distributor) {
  currentDistributor.value = distributor
  isShownCreateOrEditDistributorDrawer.value = true
}

function showDeleteDistributorDrawer(distributor: Distributor) {
  currentDistributor.value = distributor
  isShownDeleteDistributorDrawer.value = true
}

function showRestoreDistributorModal(distributor: Distributor) {
  currentDistributor.value = distributor
  isShownRestoreDistributorModal.value = true
}

function showSuspendDistributorModal(distributor: Distributor) {
  currentDistributor.value = distributor
  isShownSuspendDistributorModal.value = true
}

function showReactivateDistributorModal(distributor: Distributor) {
  currentDistributor.value = distributor
  isShownReactivateDistributorModal.value = true
}

function showDestroyDistributorModal(distributor: Distributor) {
  currentDistributor.value = distributor
  isShownDestroyDistributorModal.value = true
}

function onCloseDrawer() {
  isShownCreateOrEditDistributorDrawer.value = false
  emit('close-drawer')
}

function getKebabMenuItems(distributor: Distributor) {
  const items: NeDropdownItem[] = []

  if (canManageDistributors()) {
    if (!distributor.deleted_at) {
      items.push({
        id: 'editDistributor',
        label: t('common.edit'),
        icon: faPenToSquare,
        action: () => showEditDistributorDrawer(distributor),
        disabled: asyncStatus.value === 'loading',
      })
    }

    if (distributor.suspended_at) {
      items.push({
        id: 'reactivateDistributor',
        label: t('common.reactivate'),
        icon: faCirclePlay,
        action: () => showReactivateDistributorModal(distributor),
        disabled: asyncStatus.value === 'loading',
      })

      items.push({
        id: 'deleteDistributor',
        label: t('common.archive'),
        icon: faBoxArchive,
        danger: true,
        action: () => showDeleteDistributorDrawer(distributor),
        disabled: asyncStatus.value === 'loading',
      })
    } else if (distributor.deleted_at) {
      items.push({
        id: 'restoreSystem',
        label: t('common.restore'),
        icon: faRotateLeft,
        action: () => showRestoreDistributorModal(distributor),
        disabled: asyncStatus.value === 'loading',
      })
    } else {
      items.push({
        id: 'suspendDistributor',
        label: t('common.suspend'),
        icon: faCirclePause,
        action: () => showSuspendDistributorModal(distributor),
        disabled: asyncStatus.value === 'loading',
      })

      items.push({
        id: 'deleteDistributor',
        label: t('common.archive'),
        icon: faBoxArchive,
        danger: true,
        action: () => showDeleteDistributorDrawer(distributor),
        disabled: asyncStatus.value === 'loading',
      })
    }
  }

  if (canDestroyDistributors()) {
    items.push({
      id: 'destroyDistributor',
      label: t('common.destroy'),
      icon: faBomb,
      danger: true,
      action: () => showDestroyDistributorModal(distributor),
      disabled: asyncStatus.value === 'loading',
    })
  }
  return items
}

const onSort = (payload: SortEvent) => {
  sortBy.value = payload.key as keyof Distributor
  sortDescending.value = payload.descending
}

const goToDistributorDetails = (distributor: Distributor) => {
  router.push({ name: 'distributor_detail', params: { distributorId: distributor.logto_id } })
}
</script>

<template>
  <div>
    <!-- get distributors error notification -->
    <NeInlineNotification
      v-if="state.status === 'error'"
      kind="error"
      :title="$t('distributors.cannot_retrieve_distributors')"
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
            :placeholder="$t('distributors.filter_distributors')"
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
      :title="$t('distributors.no_distributor')"
      :icon="faGlobe"
      class="bg-white dark:bg-gray-950"
    />
    <!-- no distributor matching filter -->
    <NeEmptyState
      v-else-if="isNoMatchEmptyStateShown"
      :title="$t('distributors.no_distributor_found')"
      :description="$t('common.try_changing_search_filters')"
      :icon="faCircleInfo"
      class="bg-white dark:bg-gray-950"
    >
      <NeButton kind="tertiary" @click="resetFilters"> {{ $t('common.reset_filters') }}</NeButton>
    </NeEmptyState>
    <NeTable
      v-if="noEmptyStateShown"
      :sort-key="sortBy"
      :sort-descending="sortDescending"
      :aria-label="$t('distributors.title')"
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
        <NeTableHeadCell>
          {{ $t('resellers.title') }}
        </NeTableHeadCell>
        <NeTableHeadCell>
          {{ $t('distributors.total_customers') }}
        </NeTableHeadCell>
        <NeTableHeadCell>
          {{ $t('distributors.total_systems') }}
        </NeTableHeadCell>
        <NeTableHeadCell sortable column-key="suspended_at" @sort="onSort">{{
          $t('common.status')
        }}</NeTableHeadCell>
        <NeTableHeadCell>
          <!-- no header for actions -->
        </NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <NeTableRow v-for="(item, index) in distributorsPage" :key="index">
          <NeTableCell :data-label="$t('organizations.name')">
            <router-link
              v-if="!item.deleted_at"
              :to="{ name: 'distributor_detail', params: { distributorId: item.logto_id } }"
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
          <NeTableCell :data-label="$t('resellers.title')">
            <div class="flex items-center gap-2" :class="{ 'opacity-50': item.deleted_at }">
              <FontAwesomeIcon
                :icon="faCity"
                class="size-4 text-gray-700 dark:text-gray-400"
                aria-hidden="true"
              />
              {{ item.resellers_count }}
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('distributors.total_customers')">
            <div class="flex items-center gap-2" :class="{ 'opacity-50': item.deleted_at }">
              <FontAwesomeIcon
                :icon="faBuilding"
                class="size-4 text-gray-700 dark:text-gray-400"
                aria-hidden="true"
              />
              {{ item.customers_count }}
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('distributors.total_systems')">
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
              <NeButton
                v-if="!item.deleted_at"
                kind="tertiary"
                @click="goToDistributorDetails(item)"
              >
                <template #prefix>
                  <FontAwesomeIcon :icon="faEye" class="h-4 w-4" aria-hidden="true" />
                </template>
                {{ $t('common.view') }}
              </NeButton>
              <!-- kebab menu -->
              <NeDropdown
                v-if="canManageDistributors() || canDestroyDistributors()"
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
              savePageSizeToStorage(DISTRIBUTORS_TABLE_ID, size)
            }
          "
        />
      </template>
    </NeTable>
    <!-- side drawer -->
    <CreateOrEditDistributorDrawer
      :is-shown="isShownCreateOrEditDistributorDrawer"
      :current-distributor="currentDistributor"
      @close="onCloseDrawer"
    />
    <!-- delete distributor modal -->
    <DeleteDistributorModal
      :visible="isShownDeleteDistributorDrawer"
      :distributor="currentDistributor"
      @close="isShownDeleteDistributorDrawer = false"
    />
    <!-- suspend distributor modal -->
    <SuspendDistributorModal
      :visible="isShownSuspendDistributorModal"
      :distributor="currentDistributor"
      @close="isShownSuspendDistributorModal = false"
    />
    <!-- reactivate distributor modal -->
    <ReactivateDistributorModal
      :visible="isShownReactivateDistributorModal"
      :distributor="currentDistributor"
      @close="isShownReactivateDistributorModal = false"
    />
    <!-- restore distributor modal -->
    <RestoreDistributorModal
      :visible="isShownRestoreDistributorModal"
      :distributor="currentDistributor"
      @close="isShownRestoreDistributorModal = false"
    />
    <!-- destroy distributor modal -->
    <DestroyDistributorModal
      :visible="isShownDestroyDistributorModal"
      :distributor="currentDistributor"
      @close="isShownDestroyDistributorModal = false"
    />
  </div>
</template>
