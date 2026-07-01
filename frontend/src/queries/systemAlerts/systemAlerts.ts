//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { ALERTS_REFETCH_INTERVAL_SECONDS, type AlertStatusEnum } from '@/lib/alerts'
import { syncWithBackend } from '@/lib/alertPendingStates'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import type { NeDropdownFilterV2Option } from '@nethesis/vue-components'
import { getSystemActiveAlerts, SYSTEM_ALERTS_KEY, SYSTEM_ALERTS_TABLE_ID } from '@/lib/alerts'

export const useSystemAlerts = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const sortBy = ref<'starts_at' | 'severity' | 'alertname' | 'status'>('starts_at')
  const sortDirection = ref<'asc' | 'desc'>('desc')
  const severityFilters = ref<NeDropdownFilterV2Option[]>([])
  const alertnameFilters = ref<NeDropdownFilterV2Option[]>([])
  const statusFilters = ref<NeDropdownFilterV2Option[]>([])
  const shouldAutoRefetch = () => document.visibilityState === 'visible'

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      SYSTEM_ALERTS_KEY,
      route.params.systemId,
      pageNum.value,
      pageSize.value,
      sortBy.value,
      sortDirection.value,
      severityFilters.value.map((o) => o.id).join(','),
      alertnameFilters.value.map((o) => o.id).join(','),
      statusFilters.value.map((o) => o.id).join(','),
    ],
    enabled: () => !!loginStore.jwtToken && !!route.params.systemId,
    query: () =>
      getSystemActiveAlerts(
        route.params.systemId as string,
        pageNum.value,
        pageSize.value,
        sortBy.value,
        sortDirection.value,
        statusFilters.value.length > 0
          ? (statusFilters.value.map((o) => o.id) as AlertStatusEnum[])
          : undefined,
        severityFilters.value.length > 0 ? severityFilters.value.map((o) => o.id) : undefined,
        alertnameFilters.value.length > 0 ? alertnameFilters.value.map((o) => o.id) : undefined,
      ),
    staleTime: ALERTS_REFETCH_INTERVAL_SECONDS * 1000,
    autoRefetch: shouldAutoRefetch,
  })

  const areDefaultFiltersApplied = () =>
    !severityFilters.value.length && !alertnameFilters.value.length && !statusFilters.value.length

  const clearFilters = () => {
    severityFilters.value = []
    alertnameFilters.value = []
    clearStatusFilter()
    pageNum.value = 1
  }

  const clearStatusFilter = () => {
    statusFilters.value = []
  }

  // load table page size from storage
  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(SYSTEM_ALERTS_TABLE_ID)
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

  // When the backend returns fresh data, clean up pending states that are now confirmed
  watch(
    () => state.value.data?.alerts,
    (alerts) => {
      if (alerts) syncWithBackend(alerts)
    },
  )

  return {
    ...rest,
    state,
    asyncStatus,
    pageNum,
    pageSize,
    sortBy,
    sortDirection,
    severityFilters,
    alertnameFilters,
    statusFilters,
    areDefaultFiltersApplied,
    clearFilters,
    clearStatusFilter,
  }
})
