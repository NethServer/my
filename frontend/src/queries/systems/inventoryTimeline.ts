//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  type InventoryDiffCategory,
  type InventoryDiffSeverity,
  type InventoryDiffType,
} from '@/lib/systems/inventoryDiffs'
import { INVENTORY_TIMELINE_KEY, getInventoryTimeline } from '@/lib/systems/inventoryTimeline'
import { canReadSystems } from '@/lib/permissions'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useInfiniteQuery } from '@pinia/colada'
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'

const TIMELINE_PAGE_SIZE = 20

export const useInventoryTimeline = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()
  const severityFilter = ref<InventoryDiffSeverity[]>([])
  const categoryFilter = ref<InventoryDiffCategory[]>([])
  const diffTypeFilter = ref<InventoryDiffType[]>([])
  const fromDate = ref('')
  const toDate = ref('')

  const { state, asyncStatus, hasNextPage, loadNextPage } = useInfiniteQuery({
    key: () => [
      INVENTORY_TIMELINE_KEY,
      {
        systemId: route.params.systemId,
        severityFilter: severityFilter.value,
        categoryFilter: categoryFilter.value,
        diffTypeFilter: diffTypeFilter.value,
        fromDate: fromDate.value,
        toDate: toDate.value,
      },
    ],
    enabled: () => !!loginStore.jwtToken && canReadSystems() && !!route.params.systemId,
    initialPageParam: 1,
    query: ({ pageParam }) => {
      const apiCall = getInventoryTimeline(
        route.params.systemId as string,
        pageParam,
        TIMELINE_PAGE_SIZE,
        severityFilter.value,
        categoryFilter.value,
        diffTypeFilter.value,
        fromDate.value,
        toDate.value,
      )
      return apiCall
    },
    getNextPageParam: (lastPage) =>
      lastPage.pagination.has_next ? lastPage.pagination.page + 1 : null,
  })

  const allInventoryIds = computed(() =>
    (state.value.data?.pages ?? []).flatMap((page) =>
      page.groups.flatMap((group) => group.inventory_ids),
    ),
  )

  const allGroups = computed(() => (state.value.data?.pages ?? []).flatMap((page) => page.groups))

  const areDefaultFiltersApplied = computed(() => {
    return (
      severityFilter.value.length === 0 &&
      categoryFilter.value.length === 0 &&
      diffTypeFilter.value.length === 0 &&
      !fromDate.value &&
      !toDate.value
    )
  })

  const resetFilters = () => {
    severityFilter.value = []
    categoryFilter.value = []
    diffTypeFilter.value = []
    fromDate.value = ''
    toDate.value = ''
  }

  return {
    state,
    asyncStatus,
    hasNextPage,
    loadNextPage,
    severityFilter,
    categoryFilter,
    diffTypeFilter,
    fromDate,
    toDate,
    areDefaultFiltersApplied,
    resetFilters,
    allInventoryIds,
    allGroups,
  }
})
