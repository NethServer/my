//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  INVENTORY_DIFFS_KEY,
  INVENTORY_DIFFS_TABLE_ID,
  getInventoryDiffs,
  type InventoryDiffCategory,
  type InventoryDiffSeverity,
  type InventoryDiffType,
} from '@/lib/systems/inventoryDiffs'
import {
  INVENTORY_MOCK_ENABLED,
  mockInventoryDiffs,
  mockDiffsPagination,
} from '@/lib/systems/inventoryMocks'
import { canReadSystems } from '@/lib/permissions'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'

//// currently unused?
export const useInventoryDiffs = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()
  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const severityFilter = ref<InventoryDiffSeverity[]>([])
  const categoryFilter = ref<InventoryDiffCategory[]>([])
  const diffTypeFilter = ref<InventoryDiffType[]>([])
  const inventoryIdFilter = ref<number[]>([])
  const fromDate = ref('')
  const toDate = ref('')

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      INVENTORY_DIFFS_KEY,
      {
        systemId: route.params.systemId,
        pageNum: pageNum.value,
        pageSize: pageSize.value,
        severityFilter: severityFilter.value,
        categoryFilter: categoryFilter.value,
        diffTypeFilter: diffTypeFilter.value,
        inventoryIdFilter: inventoryIdFilter.value,
        fromDate: fromDate.value,
        toDate: toDate.value,
      },
    ],
    enabled: () => !!loginStore.jwtToken && canReadSystems() && !!route.params.systemId,
    query: () => {
      const apiCall = getInventoryDiffs(
        route.params.systemId as string,
        pageNum.value,
        pageSize.value,
        severityFilter.value,
        categoryFilter.value,
        diffTypeFilter.value,
        inventoryIdFilter.value,
        fromDate.value,
        toDate.value,
      )
      if (INVENTORY_MOCK_ENABLED) {
        apiCall.catch(() => {})
        return Promise.resolve({ diffs: mockInventoryDiffs, pagination: mockDiffsPagination })
      }
      return apiCall
    },
  })

  const areDefaultFiltersApplied = computed(() => {
    return (
      severityFilter.value.length === 0 &&
      categoryFilter.value.length === 0 &&
      diffTypeFilter.value.length === 0 &&
      inventoryIdFilter.value.length === 0 &&
      !fromDate.value &&
      !toDate.value
    )
  })

  // load table page size from storage
  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(INVENTORY_DIFFS_TABLE_ID)
      }
    },
    { immediate: true },
  )

  // reset to first page when page size changes
  watch(
    () => pageSize.value,
    () => {
      pageNum.value = 1
    },
  )

  // reset to first page when any filter changes
  watch(
    () => severityFilter.value,
    () => {
      pageNum.value = 1
    },
    { deep: true },
  )

  watch(
    () => categoryFilter.value,
    () => {
      pageNum.value = 1
    },
    { deep: true },
  )

  watch(
    () => diffTypeFilter.value,
    () => {
      pageNum.value = 1
    },
    { deep: true },
  )

  watch(
    () => inventoryIdFilter.value,
    () => {
      pageNum.value = 1
    },
    { deep: true },
  )

  watch(
    () => fromDate.value,
    () => {
      pageNum.value = 1
    },
  )

  watch(
    () => toDate.value,
    () => {
      pageNum.value = 1
    },
  )

  const resetFilters = () => {
    severityFilter.value = []
    categoryFilter.value = []
    diffTypeFilter.value = []
    inventoryIdFilter.value = []
    fromDate.value = ''
    toDate.value = ''
  }

  return {
    ...rest,
    state,
    asyncStatus,
    pageNum,
    pageSize,
    severityFilter,
    categoryFilter,
    diffTypeFilter,
    inventoryIdFilter,
    fromDate,
    toDate,
    areDefaultFiltersApplied,
    resetFilters,
  }
})
