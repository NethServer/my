<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { CUSTOMERS_TABLE_ID, type Customer } from '@/lib/organizations/customers'
import {
  faCircleInfo,
  faPenToSquare,
  faBoxArchive,
  faBuilding,
  faCirclePause,
  faCirclePlay,
  faCircleCheck,
  faRotateLeft,
  faBomb,
  faServer,
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
import CreateOrEditCustomerDrawer from './CreateOrEditCustomerDrawer.vue'
import { useI18n } from 'vue-i18n'
import DeleteCustomerModal from './DeleteCustomerModal.vue'
import SuspendCustomerModal from './SuspendCustomerModal.vue'
import ReactivateCustomerModal from './ReactivateCustomerModal.vue'
import RestoreCustomerModal from './RestoreCustomerModal.vue'
import DestroyCustomerModal from './DestroyCustomerModal.vue'
import { savePageSizeToStorage } from '@/lib/tablePageSize'
import { useCustomers } from '@/queries/organizations/customers'
import { canManageCustomers, canDestroyCustomers } from '@/lib/permissions'
import router from '@/router'

const { isShownCreateCustomerDrawer = false } = defineProps<{
  isShownCreateCustomerDrawer: boolean
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
} = useCustomers()

const currentCustomer = ref<Customer | undefined>()
const isShownCreateOrEditCustomerDrawer = ref(false)
const isShownDeleteCustomerDrawer = ref(false)
const isShownSuspendCustomerModal = ref(false)
const isShownReactivateCustomerModal = ref(false)
const isShownRestoreCustomerModal = ref(false)
const isShownDestroyCustomerModal = ref(false)

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

const customersPage = computed(() => {
  return state.value.data?.customers
})

const pagination = computed(() => {
  return state.value.data?.pagination
})

const isNoDataEmptyStateShown = computed(() => {
  return (
    !customersPage.value?.length &&
    state.value.status === 'success' &&
    areDefaultFiltersApplied.value
  )
})

const isNoMatchEmptyStateShown = computed(() => {
  return (
    !customersPage.value?.length &&
    state.value.status === 'success' &&
    !areDefaultFiltersApplied.value
  )
})

const noEmptyStateShown = computed(() => {
  return !isNoDataEmptyStateShown.value && !isNoMatchEmptyStateShown.value
})

watch(
  () => isShownCreateCustomerDrawer,
  () => {
    if (isShownCreateCustomerDrawer) {
      showCreateCustomerDrawer()
    }
  },
  { immediate: true },
)

function showCreateCustomerDrawer() {
  currentCustomer.value = undefined
  isShownCreateOrEditCustomerDrawer.value = true
}

function showEditCustomerDrawer(customer: Customer) {
  currentCustomer.value = customer
  isShownCreateOrEditCustomerDrawer.value = true
}

function showDeleteCustomerDrawer(customer: Customer) {
  currentCustomer.value = customer
  isShownDeleteCustomerDrawer.value = true
}

function showRestoreCustomerModal(customer: Customer) {
  currentCustomer.value = customer
  isShownRestoreCustomerModal.value = true
}

function showSuspendCustomerModal(customer: Customer) {
  currentCustomer.value = customer
  isShownSuspendCustomerModal.value = true
}

function showReactivateCustomerModal(customer: Customer) {
  currentCustomer.value = customer
  isShownReactivateCustomerModal.value = true
}

function showDestroyCustomerModal(customer: Customer) {
  currentCustomer.value = customer
  isShownDestroyCustomerModal.value = true
}

function onCloseDrawer() {
  isShownCreateOrEditCustomerDrawer.value = false
  emit('close-drawer')
}

function getKebabMenuItems(customer: Customer) {
  const items: NeDropdownItem[] = []

  if (canManageCustomers()) {
    if (!customer.deleted_at) {
      items.push({
        id: 'editCustomer',
        label: t('common.edit'),
        icon: faPenToSquare,
        action: () => showEditCustomerDrawer(customer),
        disabled: asyncStatus.value === 'loading',
      })
    }

    if (customer.suspended_at) {
      items.push({
        id: 'reactivateCustomer',
        label: t('common.reactivate'),
        icon: faCirclePlay,
        action: () => showReactivateCustomerModal(customer),
        disabled: asyncStatus.value === 'loading',
      })

      items.push({
        id: 'deleteCustomer',
        label: t('common.archive'),
        icon: faBoxArchive,
        danger: true,
        action: () => showDeleteCustomerDrawer(customer),
        disabled: asyncStatus.value === 'loading',
      })
    } else if (customer.deleted_at) {
      items.push({
        id: 'restoreCustomer',
        label: t('common.restore'),
        icon: faRotateLeft,
        action: () => showRestoreCustomerModal(customer),
        disabled: asyncStatus.value === 'loading',
      })
    } else {
      items.push({
        id: 'suspendCustomer',
        label: t('common.suspend'),
        icon: faCirclePause,
        action: () => showSuspendCustomerModal(customer),
        disabled: asyncStatus.value === 'loading',
      })

      items.push({
        id: 'deleteCustomer',
        label: t('common.archive'),
        icon: faBoxArchive,
        danger: true,
        action: () => showDeleteCustomerDrawer(customer),
        disabled: asyncStatus.value === 'loading',
      })
    }
  }

  if (canDestroyCustomers()) {
    items.push({
      id: 'destroyCustomer',
      label: t('common.destroy'),
      icon: faBomb,
      danger: true,
      action: () => showDestroyCustomerModal(customer),
      disabled: asyncStatus.value === 'loading',
    })
  }
  return items
}

const onSort = (payload: SortEvent) => {
  sortBy.value = payload.key as keyof Customer
  sortDescending.value = payload.descending
}

const goToCustomerDetails = (customer: Customer) => {
  router.push({ name: 'customer_detail', params: { companyId: customer.logto_id } })
}
</script>

<template>
  <div>
    <!-- get customers error notification -->
    <NeInlineNotification
      v-if="state.status === 'error'"
      kind="error"
      :title="$t('customers.cannot_retrieve_customers')"
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
            :placeholder="$t('customers.filter_customers')"
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
      :title="$t('customers.no_customer')"
      :icon="faBuilding"
      class="bg-white dark:bg-gray-950"
    />
    <!-- no customer matching filter -->
    <NeEmptyState
      v-else-if="isNoMatchEmptyStateShown"
      :title="$t('customers.no_customer_found')"
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
      :aria-label="$t('customers.title')"
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
          {{ $t('systems.title') }}
        </NeTableHeadCell>
        <NeTableHeadCell sortable column-key="suspended_at" @sort="onSort">{{
          $t('common.status')
        }}</NeTableHeadCell>
        <NeTableHeadCell>
          <!-- no header for actions -->
        </NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <NeTableRow v-for="(item, index) in customersPage" :key="index">
          <NeTableCell :data-label="$t('organizations.name')">
            <router-link
              v-if="!item.deleted_at"
              :to="{ name: 'customer_detail', params: { companyId: item.logto_id } }"
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
          <NeTableCell :data-label="$t('systems.title')">
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
              <NeButton v-if="!item.deleted_at" kind="tertiary" @click="goToCustomerDetails(item)">
                <template #prefix>
                  <FontAwesomeIcon :icon="faEye" class="h-4 w-4" aria-hidden="true" />
                </template>
                {{ $t('common.view') }}
              </NeButton>
              <!-- kebab menu -->
              <NeDropdown
                v-if="canManageCustomers() || canDestroyCustomers()"
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
              savePageSizeToStorage(CUSTOMERS_TABLE_ID, size)
            }
          "
        />
      </template>
    </NeTable>
    <!-- side drawer -->
    <CreateOrEditCustomerDrawer
      :is-shown="isShownCreateOrEditCustomerDrawer"
      :current-customer="currentCustomer"
      @close="onCloseDrawer"
    />
    <!-- delete customer modal -->
    <DeleteCustomerModal
      :visible="isShownDeleteCustomerDrawer"
      :customer="currentCustomer"
      @close="isShownDeleteCustomerDrawer = false"
    />
    <!-- suspend customer modal -->
    <SuspendCustomerModal
      :visible="isShownSuspendCustomerModal"
      :customer="currentCustomer"
      @close="isShownSuspendCustomerModal = false"
    />
    <!-- reactivate customer modal -->
    <ReactivateCustomerModal
      :visible="isShownReactivateCustomerModal"
      :customer="currentCustomer"
      @close="isShownReactivateCustomerModal = false"
    />
    <!-- restore customer modal -->
    <RestoreCustomerModal
      :visible="isShownRestoreCustomerModal"
      :customer="currentCustomer"
      @close="isShownRestoreCustomerModal = false"
    />
    <!-- destroy customer modal -->
    <DestroyCustomerModal
      :visible="isShownDestroyCustomerModal"
      :customer="currentCustomer"
      @close="isShownDestroyCustomerModal = false"
    />
  </div>
</template>
