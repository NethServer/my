//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  INVENTORY_DIFFS_KEY,
  getInventoryDiffs,
  type InventoryDiff,
} from '@/lib/systems/inventoryDiffs'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useInventoryTimeline } from './inventoryTimeline'

export const useInventoryDiffs = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const {
    allInventoryIds,
    severityFilter,
    categoryFilter,
    diffTypeFilter,
    fromDate,
    toDate,
    debouncedTextFilter,
  } = useInventoryTimeline()

  // ── Stable accumulated state ──────────────────────────────────────────────
  const stableDiffs = ref<InventoryDiff[]>([])
  const lastFetchedInventoryIds = ref<Set<number>>(new Set())
  const diffsHaveEverLoaded = ref(false)

  // Reset accumulated diffs when filter conditions (or system) change so the
  // subsequent diffs fetch starts fresh rather than merging into stale data.
  // Using a serialized computed key prevents spurious resets when resetFilters()
  // assigns new empty-array instances that are equal in content to the old ones.
  const filterResetKey = computed(() =>
    JSON.stringify({
      systemId: route.params.systemId,
      severityFilter: severityFilter.value,
      categoryFilter: categoryFilter.value,
      diffTypeFilter: diffTypeFilter.value,
      fromDate: fromDate.value,
      toDate: toDate.value,
      debouncedTextFilter: debouncedTextFilter.value,
    }),
  )

  watch(filterResetKey, () => {
    stableDiffs.value = []
    lastFetchedInventoryIds.value = new Set()
    diffsHaveEverLoaded.value = false
  })

  // ── Diffs query — key tracks the full ID set; query fetches only the delta ─
  const { state, asyncStatus } = useQuery({
    key: () => [
      INVENTORY_DIFFS_KEY,
      'timeline',
      {
        systemId: route.params.systemId,
        inventoryIds: allInventoryIds.value,
        severityFilter: severityFilter.value,
        categoryFilter: categoryFilter.value,
        diffTypeFilter: diffTypeFilter.value,
        fromDate: fromDate.value,
        toDate: toDate.value,
        search: debouncedTextFilter.value,
      },
    ],
    // Only run when there are IDs in the current timeline page that haven't been fetched yet.
    // Using allInventoryIds ensures the key and the enabled guard are always in sync: when
    // filters change the infinite query resets to no-data, allInventoryIds becomes [] and the
    // query is automatically disabled until the first timeline page for the new filters loads.
    enabled: () =>
      !!loginStore.jwtToken &&
      !!route.params.systemId &&
      allInventoryIds.value.some((id) => !lastFetchedInventoryIds.value.has(id)),
    query: () => {
      // Fetch only the IDs we haven't retrieved yet so each loadNextPage() sends a
      // minimal request instead of re-requesting every previously loaded page.
      const idsToFetch = allInventoryIds.value.filter(
        (id) => !lastFetchedInventoryIds.value.has(id),
      )
      return getInventoryDiffs(
        route.params.systemId as string,
        1,
        100,
        severityFilter.value,
        categoryFilter.value,
        diffTypeFilter.value,
        idsToFetch,
        fromDate.value,
        toDate.value,
        debouncedTextFilter.value,
      ).then((result) => ({ ...result, requestedInventoryIds: idsToFetch }))
    },
  })

  watch(
    () => state.value.data,
    (data) => {
      if (data !== undefined) {
        // Merge: replace any existing diffs for these IDs (idempotent on refetch),
        // then append the new ones. This way each loadNextPage() only fetches the
        // delta instead of re-requesting all previously loaded pages.
        const fetchedIds = new Set(data.requestedInventoryIds)
        stableDiffs.value = [
          ...stableDiffs.value.filter((d) => !fetchedIds.has(d.inventory_id)),
          ...data.diffs,
        ]
        fetchedIds.forEach((id) => lastFetchedInventoryIds.value.add(id))
        diffsHaveEverLoaded.value = true
      }
    },
  )

  // allDiffs always returns stable data — never empty during a refetch
  const allDiffs = computed<InventoryDiff[]>(() => stableDiffs.value)

  const diffsIsLoading = computed(
    () =>
      !diffsHaveEverLoaded.value &&
      state.value.status === 'pending' &&
      allInventoryIds.value.length > 0,
  )

  // True while allInventoryIds has IDs not yet covered by the last completed diffs fetch
  const diffsIsRefetching = computed(
    () =>
      diffsHaveEverLoaded.value &&
      allInventoryIds.value.some((id) => !lastFetchedInventoryIds.value.has(id)),
  )

  function getDiffsForGroup(group: { inventory_ids: number[] }): InventoryDiff[] {
    const idSet = new Set(group.inventory_ids)
    return allDiffs.value.filter((d) => idSet.has(d.inventory_id))
  }

  // Returns true when this group's diffs haven't been fetched yet (new page, still loading)
  function isGroupPendingDiffs(group: { inventory_ids: number[] }): boolean {
    if (!diffsIsRefetching.value) return false
    return group.inventory_ids.some((id) => !lastFetchedInventoryIds.value.has(id))
  }

  return {
    state,
    asyncStatus,
    allDiffs,
    diffsIsLoading,
    diffsIsRefetching,
    getDiffsForGroup,
    isGroupPendingDiffs,
  }
})
