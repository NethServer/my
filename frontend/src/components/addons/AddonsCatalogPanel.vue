<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The catalog endpoint returns the whole list in one unpaginated, unfiltered
  response, so searching, filtering, sorting and paging all happen here rather
  than in the query composable as on the other list pages.
-->

<script setup lang="ts">
import {
  faCirclePlus,
  faMagnifyingGlass,
  faPenToSquare,
  faPuzzlePiece,
  faTrash,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeBadgeV2,
  NeButton,
  NeDropdown,
  NeDropdownFilterV2,
  NeEmptyState,
  NeInlineNotification,
  NePaginator,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
  NeTextInput,
  type NeDropdownFilterV2Option,
  type NeDropdownItem,
  type SortEvent,
} from '@nethesis/vue-components'
import { useDebounceFn } from '@vueuse/core'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ApplicationLogo from '@/components/applications/ApplicationLogo.vue'
import UpdatingSpinner from '@/components/common/UpdatingSpinner.vue'
import SystemLogo from '@/components/systems/SystemLogo.vue'
import {
  ADDON_PRODUCTS,
  ADDONS_TABLE_ID,
  getAddonApplication,
  getApplicationDisplayName,
  type Addon,
} from '@/lib/addons/addons'
import { MIN_SEARCH_LENGTH } from '@/lib/common'
import { canManageAddons } from '@/lib/permissions'
import { getProductName } from '@/lib/systems/systems'
import {
  DEFAULT_PAGE_SIZE,
  PAGE_SIZE_OPTIONS,
  loadPageSizeFromStorage,
  savePageSizeToStorage,
} from '@/lib/tablePageSize'
import { useAddons } from '@/queries/addons/addons'
import { useLoginStore } from '@/stores/login'
import AddonInUseModal from './AddonInUseModal.vue'
import CreateOrEditAddonDrawer from './CreateOrEditAddonDrawer.vue'
import DeleteAddonModal from './DeleteAddonModal.vue'

type AddonSortKey = 'display_name' | 'product' | 'application'

const { t } = useI18n()
const loginStore = useLoginStore()
const { state, asyncStatus } = useAddons()

const textFilter = ref('')
const debouncedTextFilter = ref('')
const productFilter = ref<NeDropdownFilterV2Option[]>([])
const applicationFilter = ref<NeDropdownFilterV2Option[]>([])
const sortBy = ref<AddonSortKey>('display_name')
const sortDescending = ref(false)
const pageNum = ref(1)
const pageSize = ref(DEFAULT_PAGE_SIZE)

const currentAddon = ref<Addon | undefined>()
const isShownCreateOrEditAddonDrawer = ref(false)
const isShownDeleteAddonModal = ref(false)
const isShownAddonInUseModal = ref(false)

const addons = computed(() => state.value.data ?? [])

// The row shape the table renders: the product and the application are derived
// from the add-on, not stored on it, and both filtering and sorting need them.
const addonRows = computed(() =>
  addons.value.map((addon) => {
    const application = getAddonApplication(addon)

    return {
      addon,
      product: addon.system_type ?? '',
      productLabel: addon.system_type ? getProductName(addon.system_type) : '',
      application,
      applicationLabel: application ? getApplicationDisplayName(application) : '',
    }
  }),
)

const productFilterOptions = computed((): NeDropdownFilterV2Option[] =>
  ADDON_PRODUCTS.map((product) => ({ id: product, label: getProductName(product) })),
)

// Only the applications actually present in the catalog, so the filter can
// never select an empty set.
const applicationFilterOptions = computed(() => {
  const applications = new Set(
    addonRows.value.map((row) => row.application).filter((application) => !!application),
  )

  return [...applications]
    .map((application) => ({ id: application, label: getApplicationDisplayName(application) }))
    .sort((a, b) => a.label.localeCompare(b.label))
})

const filteredRows = computed(() => {
  const search = debouncedTextFilter.value.trim().toLowerCase()
  const products = productFilter.value.map((option) => option.id)
  const applications = applicationFilter.value.map((option) => option.id)

  return addonRows.value.filter((row) => {
    if (products.length && !products.includes(row.product)) {
      return false
    }
    if (applications.length && !applications.includes(row.application)) {
      return false
    }
    if (
      search &&
      !`${row.addon.display_name} ${row.addon.description} ${row.addon.id}`
        .toLowerCase()
        .includes(search)
    ) {
      return false
    }
    return true
  })
})

const sortedRows = computed(() => {
  const direction = sortDescending.value ? -1 : 1

  return [...filteredRows.value].sort((a, b) => {
    switch (sortBy.value) {
      case 'product':
        return a.productLabel.localeCompare(b.productLabel) * direction
      case 'application':
        return a.applicationLabel.localeCompare(b.applicationLabel) * direction
      default:
        return a.addon.display_name.localeCompare(b.addon.display_name) * direction
    }
  })
})

const paginatedRows = computed(() =>
  sortedRows.value.slice((pageNum.value - 1) * pageSize.value, pageNum.value * pageSize.value),
)

const isFiltered = computed(() => {
  return (
    !!debouncedTextFilter.value || !!productFilter.value.length || !!applicationFilter.value.length
  )
})

const isNoDataEmptyStateShown = computed(() => {
  return !addons.value.length && state.value.status === 'success' && !isFiltered.value
})

const isNoMatchEmptyStateShown = computed(() => {
  return !sortedRows.value.length && state.value.status === 'success' && !!isFiltered.value
})

const noEmptyStateShown = computed(() => {
  return !isNoDataEmptyStateShown.value && !isNoMatchEmptyStateShown.value
})

// load table page size from storage
watch(
  () => loginStore.userInfo?.email,
  (email) => {
    if (email) {
      pageSize.value = loadPageSizeFromStorage(ADDONS_TABLE_ID)
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
watch([() => pageSize.value, () => productFilter.value, () => applicationFilter.value], () => {
  pageNum.value = 1
})

// a deletion or a narrower filter can leave the current page past the end
watch([sortedRows, pageSize], () => {
  const lastPage = Math.max(1, Math.ceil(sortedRows.value.length / pageSize.value))

  if (pageNum.value > lastPage) {
    pageNum.value = lastPage
  }
})

function clearFilters() {
  textFilter.value = ''
  debouncedTextFilter.value = ''
  productFilter.value = []
  applicationFilter.value = []
}

function showCreateAddonDrawer() {
  currentAddon.value = undefined
  isShownCreateOrEditAddonDrawer.value = true
}

function showEditAddonDrawer(addon: Addon) {
  currentAddon.value = addon
  isShownCreateOrEditAddonDrawer.value = true
}

function showDeleteAddonModal(addon: Addon) {
  currentAddon.value = addon
  isShownDeleteAddonModal.value = true
}

function showAddonInUseModal(addon: Addon) {
  currentAddon.value = addon
  isShownAddonInUseModal.value = true
}

function getKebabMenuItems(addon: Addon): NeDropdownItem[] {
  return [
    {
      id: 'delete',
      label: t('common.delete'),
      icon: faTrash,
      danger: true,
      // an add-on any system was ever granted cannot be deleted: explain the
      // refusal instead of asking to confirm a delete that would fail
      action: () => (addon.in_use ? showAddonInUseModal(addon) : showDeleteAddonModal(addon)),
    },
  ]
}

const onSort = (payload: SortEvent) => {
  sortBy.value = payload.key as AddonSortKey
  sortDescending.value = payload.descending
}
</script>

<template>
  <div>
    <!-- page description -->
    <div class="text-tertiary-neutral mb-8 max-w-2xl">
      {{ $t('addons.page_description') }}
    </div>
    <!-- get add-ons error notification -->
    <NeInlineNotification
      v-if="state.status === 'error'"
      kind="error"
      :title="$t('addons.cannot_retrieve_addons')"
      :description="state.error.message"
      class="mb-6"
    />
    <!-- empty state -->
    <NeEmptyState
      v-if="isNoDataEmptyStateShown"
      :title="$t('addons.no_addons')"
      :description="$t('addons.no_addons_description')"
      :icon="faPuzzlePiece"
      class="bg-white dark:bg-gray-950"
    >
      <NeButton v-if="canManageAddons()" kind="primary" size="lg" @click="showCreateAddonDrawer">
        <template #prefix>
          <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
        </template>
        {{ $t('addons.create_addon') }}
      </NeButton>
    </NeEmptyState>
    <template v-if="!isNoDataEmptyStateShown">
      <!-- table toolbar -->
      <div class="mb-6 flex items-center gap-4">
        <div class="flex w-full items-end justify-between gap-4">
          <!-- filters -->
          <div class="flex flex-wrap items-center gap-4">
            <!-- text filter -->
            <NeTextInput
              v-model="textFilter"
              @blur="textFilter = textFilter.trim()"
              is-search
              :placeholder="$t('addons.filter_addons')"
              class="max-w-48 sm:max-w-sm"
            />
            <NeDropdownFilterV2
              v-model="productFilter"
              kind="checkbox"
              :label="t('addons.product')"
              :options="productFilterOptions"
              :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
              :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
              :no-options-label="t('ne_dropdown_filter.no_options')"
              :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
              :clear-search-label="t('ne_dropdown_filter.clear_search')"
              :options-filter-placeholder="t('ne_dropdown_filter.options_filter_placeholder')"
            />
            <NeDropdownFilterV2
              v-model="applicationFilter"
              kind="checkbox"
              :label="t('addons.application')"
              :options="applicationFilterOptions"
              show-options-filter
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
          <div class="flex items-center gap-4">
            <!-- update indicator -->
            <UpdatingSpinner v-if="asyncStatus === 'loading' && state.status !== 'pending'" />
            <NeButton
              v-if="canManageAddons()"
              kind="primary"
              class="shrink-0"
              @click="showCreateAddonDrawer"
            >
              <template #prefix>
                <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
              </template>
              {{ $t('addons.create_addon') }}
            </NeButton>
          </div>
        </div>
      </div>
      <!-- no add-on matching filter -->
      <NeEmptyState
        v-if="isNoMatchEmptyStateShown"
        :title="$t('addons.no_addons_found')"
        :description="$t('common.try_changing_search_filters')"
        :icon="faMagnifyingGlass"
        class="bg-white dark:bg-gray-950"
      >
        <NeButton kind="tertiary" @click="clearFilters">
          {{ $t('common.clear_filters') }}
        </NeButton>
      </NeEmptyState>
      <NeTable
        v-if="noEmptyStateShown"
        :sort-key="sortBy"
        :sort-descending="sortDescending"
        :aria-label="$t('addons.title')"
        card-breakpoint="2xl"
        :loading="state.status === 'pending'"
        :skeleton-columns="4"
        :skeleton-rows="7"
      >
        <NeTableHead>
          <NeTableHeadCell sortable column-key="display_name" @sort="onSort">{{
            $t('addons.addon_name')
          }}</NeTableHeadCell>
          <NeTableHeadCell sortable column-key="product" @sort="onSort">{{
            $t('addons.product')
          }}</NeTableHeadCell>
          <NeTableHeadCell sortable column-key="application" @sort="onSort">{{
            $t('addons.application')
          }}</NeTableHeadCell>
          <NeTableHeadCell>
            <!-- no header for actions -->
          </NeTableHeadCell>
        </NeTableHead>
        <NeTableBody>
          <NeTableRow v-for="row in paginatedRows" :key="row.addon.id">
            <NeTableCell :data-label="$t('addons.addon_name')">
              <div class="flex items-center gap-2">
                <span class="font-medium">{{ row.addon.display_name }}</span>
                <NeBadgeV2 v-if="!row.addon.in_use" kind="gray" size="xs">
                  {{ $t('addons.unused') }}
                </NeBadgeV2>
              </div>
              <div v-if="row.addon.description" class="text-tertiary-neutral truncate">
                {{ row.addon.description }}
              </div>
            </NeTableCell>
            <NeTableCell :data-label="$t('addons.product')">
              <div v-if="row.product" class="flex items-center gap-2">
                <SystemLogo :system="row.product" />
                <span>{{ row.productLabel }}</span>
              </div>
              <span v-else>-</span>
            </NeTableCell>
            <NeTableCell :data-label="$t('addons.application')">
              <div v-if="row.application" class="flex items-center gap-2">
                <ApplicationLogo :app="row.application" />
                <span>{{ row.applicationLabel }}</span>
              </div>
              <span v-else>-</span>
            </NeTableCell>
            <NeTableCell :data-label="$t('common.actions')">
              <div class="-ml-2.5 flex gap-2 2xl:ml-0 2xl:justify-end">
                <NeButton
                  v-if="canManageAddons()"
                  kind="tertiary"
                  @click="showEditAddonDrawer(row.addon)"
                >
                  <template #prefix>
                    <FontAwesomeIcon :icon="faPenToSquare" class="h-4 w-4" aria-hidden="true" />
                  </template>
                  {{ $t('common.edit') }}
                </NeButton>
                <!-- kebab menu -->
                <NeDropdown
                  v-if="canManageAddons()"
                  :items="getKebabMenuItems(row.addon)"
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
            :total-rows="sortedRows.length"
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
                savePageSizeToStorage(ADDONS_TABLE_ID, size)
              }
            "
          />
        </template>
      </NeTable>
    </template>
    <!-- create/edit add-on drawer -->
    <CreateOrEditAddonDrawer
      :is-shown="isShownCreateOrEditAddonDrawer"
      :current-addon="currentAddon"
      @close="isShownCreateOrEditAddonDrawer = false"
    />
    <!-- delete add-on modal -->
    <DeleteAddonModal
      :visible="isShownDeleteAddonModal"
      :addon="currentAddon"
      @close="isShownDeleteAddonModal = false"
    />
    <!-- add-on in use modal -->
    <AddonInUseModal
      :visible="isShownAddonInUseModal"
      :addon="currentAddon"
      @close="isShownAddonInUseModal = false"
    />
  </div>
</template>
