<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { searchStringInDistributor, type Distributor } from '@/lib/distributors'
import { useLoginStore } from '@/stores/login'
import {
  faCircleInfo,
  faCirclePlus,
  faGlobe,
  faPenToSquare,
  faTrash,
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
  useItemPagination,
  NeButton,
  NeEmptyState,
  NeInlineNotification,
  NeTextInput,
  NeSpinner,
  NeDropdown,
  useSort,
  type SortEvent,
  NeSortDropdown,
} from '@nethesis/vue-components'
import { computed, ref, watch } from 'vue'
import CreateOrEditDistributorDrawer from './CreateOrEditDistributorDrawer.vue'
import { useI18n } from 'vue-i18n'
import DeleteDistributorModal from './DeleteDistributorModal.vue'
import { loadPageSizeFromStorage, savePageSizeToStorage } from '@/lib/tablePageSize'
import { useDistributors } from '@/queries/distributors'
import { canManageDistributors } from '@/lib/permissions'

const { isShownCreateDistributorDrawer = false } = defineProps<{
  isShownCreateDistributorDrawer: boolean
}>()

const emit = defineEmits(['close-drawer'])

const { t } = useI18n()
const loginStore = useLoginStore()
const { distributors, distributorsAsyncStatus } = useDistributors()

const currentDistributor = ref<Distributor | undefined>()
const textFilter = ref('')
const isShownCreateOrEditDistributorDrawer = ref(false)
const isShownDeleteDistributorDrawer = ref(false)
const tableId = 'distributorsTable'
const pageSize = ref(10)
const sortKey = ref<keyof Distributor>('name')
const sortDescending = ref(false)

const filteredDistributors = computed(() => {
  if (!distributors.value.data?.length) {
    return []
  }

  if (!textFilter.value.trim()) {
    return distributors.value.data
  } else {
    return distributors.value.data.filter((distributor) =>
      searchStringInDistributor(textFilter.value, distributor),
    )
  }
})

const { sortedItems } = useSort(filteredDistributors, sortKey, sortDescending)

const { currentPage, paginatedItems } = useItemPagination(() => sortedItems.value, {
  itemsPerPage: pageSize,
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

watch(
  () => loginStore.userInfo?.email,
  (email) => {
    if (email) {
      pageSize.value = loadPageSizeFromStorage(tableId)
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

function onCloseDrawer() {
  isShownCreateOrEditDistributorDrawer.value = false
  emit('close-drawer')
}

function getKebabMenuItems(distributor: Distributor) {
  return [
    {
      id: 'deleteDistributor',
      label: t('common.delete'),
      icon: faTrash,
      danger: true,
      action: () => showDeleteDistributorDrawer(distributor),
      disabled: distributorsAsyncStatus.value === 'loading',
    },
  ]
}

const onSort = (payload: SortEvent) => {
  sortKey.value = payload.key as keyof Distributor
  sortDescending.value = payload.descending
}
</script>

<template>
  <div>
    <!-- get distributors error notification -->
    <NeInlineNotification
      v-if="distributors.status === 'error'"
      kind="error"
      :title="$t('distributors.cannot_retrieve_distributors')"
      :description="distributors.error.message"
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
          <!-- //// check dropdown options -->
          <NeSortDropdown
            v-model:sort-key="sortKey"
            v-model:sort-descending="sortDescending"
            :label="t('sort.sort')"
            :options="[
              { id: 'name', label: t('organizations.name') },
              { id: 'description', label: t('organizations.description') },
            ]"
            :open-menu-aria-label="t('ne_dropdown.open_menu')"
            :sort-by-label="t('sort.sort_by')"
            :sort-direction-label="t('sort.direction')"
            :ascending-label="t('sort.ascending')"
            :descending-label="t('sort.descending')"
            class="xl:hidden"
          />
        </div>
        <!-- update indicator -->
        <div
          v-if="distributorsAsyncStatus === 'loading' && distributors.status !== 'pending'"
          class="flex items-center gap-2"
        >
          <NeSpinner color="white" />
          <div class="text-gray-500 dark:text-gray-400">
            {{ $t('common.updating') }}
          </div>
        </div>
      </div>
    </div>
    <!-- //// check breakpoint, skeleton-columns -->
    <NeTable
      :sort-key="sortKey"
      :sort-descending="sortDescending"
      :aria-label="$t('distributors.title')"
      card-breakpoint="xl"
      :loading="distributors.status === 'pending'"
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
        <NeTableHeadCell>
          <!-- no header for actions -->
        </NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <!-- empty state -->
        <NeTableRow v-if="!distributors.data?.length">
          <NeTableCell colspan="5">
            <NeEmptyState
              :title="$t('distributors.no_distributor')"
              :icon="faGlobe"
              class="bg-white dark:bg-gray-950"
            >
              <!-- create distributor -->
              <NeButton
                v-if="canManageDistributors()"
                kind="secondary"
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
          </NeTableCell>
        </NeTableRow>
        <!-- no distributor matching filter -->
        <NeTableRow v-else-if="!filteredDistributors.length">
          <NeTableCell colspan="4">
            <NeEmptyState
              :title="$t('distributors.no_distributor_found')"
              :description="$t('common.try_changing_search_filters')"
              :icon="faCircleInfo"
              class="bg-white dark:bg-gray-950"
            >
              <NeButton kind="tertiary" @click="clearFilters">
                {{ $t('common.clear_filters') }}</NeButton
              >
            </NeEmptyState>
          </NeTableCell>
        </NeTableRow>
        <NeTableRow v-for="(item, index) in paginatedItems" v-else :key="index">
          <NeTableCell :data-label="$t('organizations.name')">
            {{ item.name }}
          </NeTableCell>
          <NeTableCell :data-label="$t('organizations.description')">
            {{ item.description || '-' }}
          </NeTableCell>
          <NeTableCell :data-label="$t('common.actions')">
            <div v-if="canManageDistributors()" class="-ml-2.5 flex gap-2 xl:ml-0 xl:justify-end">
              <NeButton
                kind="tertiary"
                @click="showEditDistributorDrawer(item)"
                :disabled="distributorsAsyncStatus === 'loading'"
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
          :current-page="currentPage"
          :total-rows="sortedItems.length"
          :page-size="pageSize"
          :nav-pagination-label="$t('ne_table.pagination')"
          :next-label="$t('ne_table.go_to_next_page')"
          :previous-label="$t('ne_table.go_to_previous_page')"
          :range-of-total-label="$t('ne_table.of')"
          :page-size-label="$t('ne_table.show')"
          @select-page="
            (page: number) => {
              currentPage = page
            }
          "
          @select-page-size="
            (size: number) => {
              pageSize = size
              savePageSizeToStorage(tableId, size)
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
  </div>
</template>
