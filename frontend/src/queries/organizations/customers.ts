//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { MIN_SEARCH_LENGTH } from '@/lib/common'
import {
  CUSTOMERS_KEY,
  CUSTOMERS_TABLE_ID,
  getCustomers,
  type Customer,
  type CustomerStatus,
} from '@/lib/organizations/customers'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useDebounceFn } from '@vueuse/core'
import { computed, ref, watch } from 'vue'
import type { NeDropdownFilterV2Option } from '@nethesis/vue-components'

export const useCustomers = defineQuery(() => {
  const loginStore = useLoginStore()
  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const textFilter = ref('')
  const debouncedTextFilter = ref('')
  const statusFilter = ref<NeDropdownFilterV2Option[]>([
    { id: 'enabled', label: 'enabled' },
    { id: 'suspended', label: 'suspended' },
  ])
  const createdByFilter = ref<NeDropdownFilterV2Option[]>([])
  // parent company: the reseller or distributor the customer belongs to
  const organizationFilter = ref<NeDropdownFilterV2Option[]>([])
  // when true, the customers of every company in the hierarchy of the selected
  // organization are shown (organizationFilter holds that single organization)
  const includeHierarchy = ref(false)
  const sortBy = ref<keyof Customer>('name')
  const sortDescending = ref(false)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      CUSTOMERS_KEY,
      {
        pageNum: pageNum.value,
        pageSize: pageSize.value,
        textFilter: debouncedTextFilter.value,
        statusFilter: statusFilter.value.map((o) => o.id),
        createdByFilter: createdByFilter.value.map((o) => o.id),
        organizationFilter: organizationFilter.value.map((o) => o.id),
        includeHierarchy: includeHierarchy.value,
        sortBy: sortBy.value,
        sortDirection: sortDescending.value,
      },
    ],
    enabled: () => !!loginStore.jwtToken,
    query: () =>
      getCustomers(
        pageNum.value,
        pageSize.value,
        debouncedTextFilter.value,
        statusFilter.value.map((o) => o.id) as CustomerStatus[],
        createdByFilter.value.map((o) => o.id),
        organizationFilter.value.map((o) => o.id),
        includeHierarchy.value,
        sortBy.value,
        sortDescending.value,
      ),
  })

  const areDefaultFiltersApplied = computed(() => {
    return (
      !debouncedTextFilter.value &&
      statusFilter.value.length === 2 &&
      statusFilter.value.some((o) => o.id === 'enabled') &&
      statusFilter.value.some((o) => o.id === 'suspended') &&
      !statusFilter.value.some((o) => o.id === 'deleted') &&
      createdByFilter.value.length === 0 &&
      organizationFilter.value.length === 0
    )
  })

  // load table page size from storage
  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(CUSTOMERS_TABLE_ID)
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

  // reset to first page when page size changes
  watch(
    () => pageSize.value,
    () => {
      pageNum.value = 1
    },
  )

  // reset to first page when status filter changes
  watch(
    () => statusFilter.value,
    () => {
      pageNum.value = 1
    },
  )

  // reset to first page when createdBy filter changes
  watch(
    () => createdByFilter.value,
    () => {
      pageNum.value = 1
    },
    { deep: true },
  )

  // the organization hierarchy mode is scoped to; lets us tell a genuine user
  // change apart from OrganizationDropdownFilter re-emitting the same selection
  // as a fresh array on mount (which must not exit hierarchy mode)
  const hierarchyOrgId = ref<string | null>(null)

  // watch the selected org ids by value (not the array reference): reset to the
  // first page when the parent company selection changes, and exit hierarchy
  // mode whenever the selection moves away from the organization it was applied to
  watch(
    () => organizationFilter.value.map((o) => o.id).join(','),
    (ids) => {
      pageNum.value = 1

      if (includeHierarchy.value && ids !== (hierarchyOrgId.value ?? '')) {
        includeHierarchy.value = false
        hierarchyOrgId.value = null
      }
    },
  )

  // filter customers by the given organization and every company in its hierarchy
  const applyHierarchyFilter = (organization: NeDropdownFilterV2Option) => {
    resetFilters()
    organizationFilter.value = [organization]
    includeHierarchy.value = true
    hierarchyOrgId.value = organization.id
  }

  const resetFilters = () => {
    textFilter.value = ''
    createdByFilter.value = []
    organizationFilter.value = []
    includeHierarchy.value = false
    hierarchyOrgId.value = null
    resetStatusFilter()
  }

  const resetStatusFilter = () => {
    statusFilter.value = [
      { id: 'enabled', label: 'enabled' },
      { id: 'suspended', label: 'suspended' },
    ]
  }

  return {
    ...rest,
    state,
    asyncStatus,
    pageNum,
    pageSize,
    textFilter,
    debouncedTextFilter,
    statusFilter,
    createdByFilter,
    organizationFilter,
    includeHierarchy,
    sortBy,
    sortDescending,
    areDefaultFiltersApplied,
    applyHierarchyFilter,
    resetFilters,
    resetStatusFilter,
  }
})
