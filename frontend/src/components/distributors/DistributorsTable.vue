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

const distributorsPage = computed(() => {
  return state.value.data?.distributors
})

const pagination = computed(() => {
  return state.value.data?.pagination
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
      disabled: asyncStatus.value === 'loading',
    },
  ]
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
    <NeTable
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
        <NeTableHeadCell>
          <!-- no header for actions -->
        </NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <!-- empty state -->
        <NeTableRow v-if="!distributorsPage?.length && !debouncedTextFilter">
          <NeTableCell colspan="5">
            <NeEmptyState
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
          </NeTableCell>
        </NeTableRow>
        <!-- no distributor matching filter -->
        <NeTableRow v-else-if="!distributorsPage?.length && debouncedTextFilter">
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
        <NeTableRow v-for="(item, index) in distributorsPage" v-else :key="index">
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
