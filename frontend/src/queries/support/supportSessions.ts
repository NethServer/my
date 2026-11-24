//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { MIN_SEARCH_LENGTH } from '@/lib/common'
import { canConnectSystems } from '@/lib/permissions'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import {
  getSupportSessions,
  SUPPORT_SESSIONS_KEY,
  SUPPORT_SESSIONS_TABLE_ID,
  type SupportSessionSortBy,
  type SupportSessionStatus,
} from '@/lib/support/support'
import { useLoginStore } from '@/stores/login'
import type { NeDropdownFilterV2Option } from '@nethesis/vue-components'
import { defineQuery, useQuery } from '@pinia/colada'
import { useDebounceFn } from '@vueuse/core'
import { computed, ref, watch } from 'vue'

// The page splits the sessions in two lists: the live ones, always in view,
// and the history (closed or expired), loaded only once it is opened
export type SupportSessionsScope = 'live' | 'history'

export const SUPPORT_SESSIONS_SCOPE_STATUSES: Record<SupportSessionsScope, SupportSessionStatus[]> =
  {
    live: ['active', 'pending'],
    history: ['closed', 'expired'],
  }

const supportSessionsList = (scope: SupportSessionsScope) => () => {
  const loginStore = useLoginStore()
  // the history list stays idle until its section is expanded
  const isEnabled = ref(scope === 'live')

  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const textFilter = ref('')
  const debouncedTextFilter = ref('')
  const productFilter = ref<NeDropdownFilterV2Option[]>([])
  const organizationFilter = ref<NeDropdownFilterV2Option[]>([])
  // narrows within the scope: no selection means every status of the scope
  const statusFilter = ref<NeDropdownFilterV2Option[]>([])
  const sortBy = ref<SupportSessionSortBy>('started_at')
  const sortDescending = ref(true)

  const filters = computed(() => ({
    search: debouncedTextFilter.value,
    types: productFilter.value.map((o) => o.id),
    organizationIds: organizationFilter.value.map((o) => o.id),
    statuses: (statusFilter.value.length
      ? statusFilter.value.map((o) => o.id)
      : SUPPORT_SESSIONS_SCOPE_STATUSES[scope]) as SupportSessionStatus[],
  }))

  const { state, asyncStatus, refetch, ...rest } = useQuery({
    key: () => [
      SUPPORT_SESSIONS_KEY,
      scope,
      {
        pageNum: pageNum.value,
        pageSize: pageSize.value,
        ...filters.value,
        sortBy: sortBy.value,
        sortDirection: sortDescending.value,
      },
    ],
    enabled: () => !!loginStore.jwtToken && canConnectSystems() && isEnabled.value,
    query: () =>
      getSupportSessions(
        pageNum.value,
        pageSize.value,
        filters.value,
        sortBy.value,
        sortDescending.value,
      ),
    refetchOnWindowFocus: true,
  })

  const areDefaultFiltersApplied = computed(() => {
    return (
      !debouncedTextFilter.value &&
      productFilter.value.length === 0 &&
      organizationFilter.value.length === 0 &&
      statusFilter.value.length === 0
    )
  })

  // load table page size from storage
  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(supportSessionsTableId(scope))
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
        pageNum.value = 1
      }
    }, 500),
  )

  // reset to first page when page size or a filter changes
  watch(
    () => [
      pageSize.value,
      productFilter.value.map((o) => o.id).join(','),
      organizationFilter.value.map((o) => o.id).join(','),
      statusFilter.value.map((o) => o.id).join(','),
    ],
    () => {
      pageNum.value = 1
    },
  )

  const resetFilters = () => {
    textFilter.value = ''
    productFilter.value = []
    organizationFilter.value = []
    statusFilter.value = []
  }

  return {
    ...rest,
    state,
    asyncStatus,
    refetch,
    isEnabled,
    pageNum,
    pageSize,
    textFilter,
    productFilter,
    organizationFilter,
    statusFilter,
    sortBy,
    sortDescending,
    areDefaultFiltersApplied,
    resetFilters,
  }
}

// each list keeps its own page size
export const supportSessionsTableId = (scope: SupportSessionsScope) =>
  scope === 'live' ? SUPPORT_SESSIONS_TABLE_ID : `${SUPPORT_SESSIONS_TABLE_ID}History`

export const useLiveSupportSessions = defineQuery(supportSessionsList('live'))
export const useSupportSessionsHistory = defineQuery(supportSessionsList('history'))
