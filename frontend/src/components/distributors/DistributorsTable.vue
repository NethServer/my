<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { DISTRIBUTORS_TABLE_ID, type Distributor } from '@/lib/distributors'
import {
  faCircleInfo,
  faCirclePlus,
  faGlobe,
  faPenToSquare,
  faTrash,
  faCirclePause,
  faCirclePlay,
  faCircleCheck,
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
} from '@nethesis/vue-components'
import { computed, ref, watch } from 'vue'
import CreateOrEditDistributorDrawer from './CreateOrEditDistributorDrawer.vue'
import { useI18n } from 'vue-i18n'
import DeleteDistributorModal from './DeleteDistributorModal.vue'
import SuspendDistributorModal from './SuspendDistributorModal.vue'
import ReactivateDistributorModal from './ReactivateDistributorModal.vue'
import { savePageSizeToStorage } from '@/lib/tablePageSize'
import { useDistributors } from '@/queries/distributors'
import { canManageDistributors } from '@/lib/permissions'

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
  debouncedTextFilter,
  sortBy,
  sortDescending,
} = useDistributors()

const currentDistributor = ref<Distributor | undefined>()
const isShownCreateOrEditDistributorDrawer = ref(false)
const isShownDeleteDistributorDrawer = ref(false)
const isShownSuspendDistributorModal = ref(false)
const isShownReactivateDistributorModal = ref(false)

const distributorsPage = computed(() => {
  return state.value.data?.distributors
})

const pagination = computed(() => {
  return state.value.data?.pagination
})

const isNoDataEmptyStateShown = computed(() => {
  return (
    !distributorsPage.value?.length &&
    !debouncedTextFilter.value &&
    state.value.status === 'success'
  )
})

const isNoMatchEmptyStateShown = computed(() => {
  return !distributorsPage.value?.length && !!debouncedTextFilter.value
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

function clearFilters() {
  textFilter.value = ''
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

function showSuspendDistributorModal(distributor: Distributor) {
  currentDistributor.value = distributor
  isShownSuspendDistributorModal.value = true
}

function showReactivateDistributorModal(distributor: Distributor) {
  currentDistributor.value = distributor
  isShownReactivateDistributorModal.value = true
}

function onCloseDrawer() {
  isShownCreateOrEditDistributorDrawer.value = false
  emit('close-drawer')
}

function getKebabMenuItems(distributor: Distributor) {
  const items = []

  if (canManageDistributors()) {
    if (distributor.suspended_at) {
      items.push({
        id: 'reactivateDistributor',
        label: t('common.reactivate'),
        icon: faCirclePlay,
        action: () => showReactivateDistributorModal(distributor),
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
    }

    items.push({
      id: 'deleteDistributor',
      label: t('common.delete'),
      icon: faTrash,
      danger: true,
      action: () => showDeleteDistributorDrawer(distributor),
      disabled: asyncStatus.value === 'loading',
    })
  }
  return items
}

const onSort = (payload: SortEvent) => {
  sortBy.value = payload.key as keyof Distributor
  sortDescending.value = payload.descending
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
    <!-- empty state -->
    <NeEmptyState
      v-if="isNoDataEmptyStateShown"
      :title="$t('distributors.no_distributor')"
      :icon="faGlobe"
      class="bg-white dark:bg-gray-950"
    >
      <!-- create distributor -->
      <NeButton
        v-if="canManageDistributors()"
        kind="primary"
        size="lg"
        class="shrink-0"
        @click="showCreateDistributorDrawer()"
      >
        <template #prefix>
          <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
        </template>
        {{ $t('distributors.create_distributor') }}
      </NeButton>
    </NeEmptyState>
    <template v-if="!isNoDataEmptyStateShown">
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
            <NeSortDropdown
              v-model:sort-key="sortBy"
              v-model:sort-descending="sortDescending"
              :label="t('sort.sort')"
              :options="[
                { id: 'name', label: t('organizations.name') },
                { id: 'description', label: t('organizations.description') },
                { id: 'suspended_at', label: t('common.status') },
              ]"
              :open-menu-aria-label="t('ne_dropdown.open_menu')"
              :sort-by-label="t('sort.sort_by')"
              :sort-direction-label="t('sort.direction')"
              :ascending-label="t('sort.ascending')"
              :descending-label="t('sort.descending')"
            />
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
      <!-- no distributor matching filter -->
      <NeEmptyState
        v-if="isNoMatchEmptyStateShown"
        :title="$t('distributors.no_distributor_found')"
        :description="$t('common.try_changing_search_filters')"
        :icon="faCircleInfo"
        class="bg-white dark:bg-gray-950"
      >
        <NeButton kind="tertiary" @click="clearFilters"> {{ $t('common.clear_filters') }}</NeButton>
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
          <NeTableHeadCell sortable column-key="description" @sort="onSort">{{
            $t('organizations.description')
          }}</NeTableHeadCell>
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
              {{ item.name }}
            </NeTableCell>
            <NeTableCell :data-label="$t('organizations.description')">
              {{ item.description || '-' }}
            </NeTableCell>
            <NeTableCell :data-label="$t('common.status')">
              <div class="flex items-center gap-2">
                <template v-if="item.suspended_at">
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
              <div v-if="canManageDistributors()" class="-ml-2.5 flex gap-2 xl:ml-0 xl:justify-end">
                <NeButton
                  kind="tertiary"
                  @click="showEditDistributorDrawer(item)"
                  :disabled="asyncStatus === 'loading'"
                >
                  <template #prefix>
                    <FontAwesomeIcon :icon="faPenToSquare" class="h-4 w-4" aria-hidden="true" />
                  </template>
                  {{ $t('common.edit') }}
                </NeButton>
                <!-- kebab menu -->
                <NeDropdown :items="getKebabMenuItems(item)" :align-to-right="true" />
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
    </template>
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
  </div>
</template>
