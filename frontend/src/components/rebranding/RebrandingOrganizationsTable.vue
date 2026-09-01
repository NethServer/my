<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  formatDateTimeNoSeconds,
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
import { faMagnifyingGlass, faPalette, faTrash } from '@fortawesome/free-solid-svg-icons'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import UpdatingSpinner from '@/components/common/UpdatingSpinner.vue'
import OrganizationIconAndLink from '@/components/organizations/OrganizationIconAndLink.vue'
import { isRebrandingAdmin } from '@/lib/permissions'
import { getRebrandingProductBadgeClasses, REBRANDING_TABLE_ID } from '@/lib/rebranding/rebranding'
import {
  REBRANDING_ORGANIZATION_TYPES,
  type RebrandingOrganization,
  type RebrandingSortBy,
} from '@/lib/rebranding/rebrandingOrganizations'
import { PAGE_SIZE_OPTIONS, savePageSizeToStorage } from '@/lib/tablePageSize'
import { useRebrandingOrganizations } from '@/queries/rebranding/rebrandingOrganizations'
import RemoveFromRebrandingModal from './RemoveFromRebrandingModal.vue'

const { t, locale } = useI18n()
const {
  state,
  asyncStatus,
  pageNum,
  pageSize,
  textFilter,
  typeFilter,
  sortBy,
  sortDescending,
  areDefaultFiltersApplied,
  resetFilters,
} = useRebrandingOrganizations()

const currentOrganization = ref<RebrandingOrganization | undefined>()
const isShownRemoveFromRebrandingModal = ref(false)

const typeFilterOptions = computed<NeDropdownFilterV2Option[]>(() =>
  REBRANDING_ORGANIZATION_TYPES.map((type) => ({
    id: type,
    label: t(`organizations.${type}`),
  })),
)

const organizationsPage = computed(() => state.value.data?.organizations)
const pagination = computed(() => state.value.data?.pagination)

const isNoDataEmptyStateShown = computed(
  () =>
    !organizationsPage.value?.length &&
    state.value.status === 'success' &&
    areDefaultFiltersApplied.value,
)

const isNoMatchEmptyStateShown = computed(
  () =>
    !organizationsPage.value?.length &&
    state.value.status === 'success' &&
    !areDefaultFiltersApplied.value,
)

const noEmptyStateShown = computed(
  () => !isNoDataEmptyStateShown.value && !isNoMatchEmptyStateShown.value,
)

const onSort = (payload: SortEvent) => {
  sortBy.value = payload.key as RebrandingSortBy
  sortDescending.value = payload.descending
}

function showRemoveFromRebrandingModal(organization: RebrandingOrganization) {
  currentOrganization.value = organization
  isShownRemoveFromRebrandingModal.value = true
}

function getKebabMenuItems(organization: RebrandingOrganization): NeDropdownItem[] {
  return [
    {
      id: 'removeFromRebranding',
      label: t('rebranding.remove_from_rebranding'),
      icon: faTrash,
      danger: true,
      action: () => showRemoveFromRebrandingModal(organization),
      disabled: asyncStatus.value === 'loading',
    },
  ]
}
</script>

<template>
  <div>
    <!-- get rebranding organizations error notification -->
    <NeInlineNotification
      v-if="state.status === 'error'"
      kind="error"
      :title="$t('rebranding.cannot_retrieve_rebranding_organizations')"
      :description="state.error.message"
      class="mb-6"
    />
    <!-- table toolbar -->
    <div class="mb-6 flex items-center gap-4">
      <div class="flex w-full items-end justify-between gap-4">
        <!-- filters -->
        <div class="flex flex-wrap items-center gap-4">
          <!-- text filter -->
          <NeTextInput
            v-model="textFilter"
            is-search
            :placeholder="$t('rebranding.filter_companies')"
            class="max-w-48 sm:max-w-sm"
            @blur="textFilter = textFilter.trim()"
          />
          <!-- company type filter -->
          <NeDropdownFilterV2
            v-model="typeFilter"
            kind="checkbox"
            :label="t('rebranding.type')"
            :options="typeFilterOptions"
            :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
            :options-filter-placeholder="t('ne_dropdown_filter.options_filter_placeholder')"
          />
          <NeButton kind="tertiary" @click="resetFilters">
            {{ t('common.reset_filters') }}
          </NeButton>
        </div>
        <!-- update indicator -->
        <UpdatingSpinner v-if="asyncStatus === 'loading' && state.status !== 'pending'" />
      </div>
    </div>
    <!-- no company has rebranding yet -->
    <NeEmptyState
      v-if="isNoDataEmptyStateShown"
      :title="$t('rebranding.no_company_with_rebranding')"
      :description="$t('rebranding.no_company_with_rebranding_description')"
      :icon="faPalette"
      class="bg-white dark:bg-gray-950"
    >
      <slot name="empty-state-action"></slot>
    </NeEmptyState>
    <!-- no company matching filters -->
    <NeEmptyState
      v-else-if="isNoMatchEmptyStateShown"
      :title="$t('rebranding.no_company_found')"
      :description="$t('common.try_changing_search_filters')"
      :icon="faMagnifyingGlass"
      class="bg-white dark:bg-gray-950"
    >
      <NeButton kind="tertiary" @click="resetFilters">{{ $t('common.reset_filters') }}</NeButton>
    </NeEmptyState>
    <NeTable
      v-if="noEmptyStateShown"
      :sort-key="sortBy"
      :sort-descending="sortDescending"
      :aria-label="$t('rebranding.title')"
      card-breakpoint="2xl"
      :loading="state.status === 'pending'"
      :skeleton-columns="4"
      :skeleton-rows="7"
    >
      <NeTableHead>
        <NeTableHeadCell sortable column-key="name" @sort="onSort">
          {{ $t('rebranding.company') }}
        </NeTableHeadCell>
        <NeTableHeadCell>{{ $t('rebranding.branded_products') }}</NeTableHeadCell>
        <NeTableHeadCell sortable column-key="updated_at" @sort="onSort">
          {{ $t('rebranding.last_updated') }}
        </NeTableHeadCell>
        <NeTableHeadCell>
          <!-- no header for actions -->
        </NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <NeTableRow v-for="item in organizationsPage" :key="item.logto_id">
          <NeTableCell :data-label="$t('rebranding.company')">
            <OrganizationIconAndLink
              :organization="{
                logto_id: item.logto_id,
                name: item.name,
                type: item.organization_type,
              }"
              icon-size="sm"
            />
          </NeTableCell>
          <NeTableCell :data-label="$t('rebranding.branded_products')">
            <div v-if="item.products.length" class="flex flex-wrap gap-2">
              <NeBadgeV2
                v-for="product in item.products"
                :key="product.product_id"
                kind="custom"
                :custom-kind-classes="getRebrandingProductBadgeClasses(product.product_id)"
                size="xs"
              >
                {{ product.product_display_name }}
              </NeBadgeV2>
            </div>
            <span v-else class="text-tertiary-neutral">-</span>
          </NeTableCell>
          <NeTableCell :data-label="$t('rebranding.last_updated')">
            <!-- null until the company has configured at least one product -->
            <span v-if="item.updated_at">
              {{ formatDateTimeNoSeconds(new Date(item.updated_at), locale) }}
            </span>
            <span v-else class="text-tertiary-neutral">-</span>
          </NeTableCell>
          <NeTableCell :data-label="$t('common.actions')">
            <div class="-ml-2.5 flex gap-2 2xl:ml-0 2xl:justify-end">
              <!-- kebab menu -->
              <NeDropdown
                v-if="isRebrandingAdmin()"
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
              savePageSizeToStorage(REBRANDING_TABLE_ID, size)
            }
          "
        />
      </template>
    </NeTable>
    <!-- remove from rebranding modal -->
    <RemoveFromRebrandingModal
      :visible="isShownRemoveFromRebrandingModal"
      :organization="currentOrganization"
      @close="isShownRemoveFromRebrandingModal = false"
    />
  </div>
</template>
