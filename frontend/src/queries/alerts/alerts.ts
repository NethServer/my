//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  getAlerts,
  ALERTS_ALERTS_KEY,
  ALERTS_TABLE_ID,
  type AlertStatusEnum,
  ALERTS_REFETCH_INTERVAL_SECONDS,
} from '@/lib/alerts'
import { syncWithBackend } from '@/lib/alertPendingStates'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref, watch } from 'vue'
import type { NeDropdownFilterV2Option } from '@nethesis/vue-components'

export const useAlerts = defineQuery(() => {
  const loginStore = useLoginStore()
  const organizationIds = ref<NeDropdownFilterV2Option[]>([])
  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const sortBy = ref<'starts_at' | 'severity' | 'alertname' | 'status'>('starts_at')
  const sortDirection = ref<'asc' | 'desc'>('desc')
  const statusFilters = ref<NeDropdownFilterV2Option[]>([])
  const severityFilters = ref<NeDropdownFilterV2Option[]>([])
  const systemKeyFilters = ref<NeDropdownFilterV2Option[]>([])
  const alertnameFilters = ref<NeDropdownFilterV2Option[]>([])
  const shouldAutoRefetch = () => document.visibilityState === 'visible'

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      ALERTS_ALERTS_KEY,
      organizationIds.value.map((o) => o.id).join(','),
      pageNum.value,
      pageSize.value,
      sortBy.value,
      sortDirection.value,
      statusFilters.value.map((o) => o.id).join(','),
      severityFilters.value.map((o) => o.id).join(','),
      systemKeyFilters.value.map((o) => o.id).join(','),
      alertnameFilters.value.map((o) => o.id).join(','),
    ],
    enabled: () => !!loginStore.jwtToken,
    query: () =>
      getAlerts(
        organizationIds.value.length > 0 ? organizationIds.value.map((o) => o.id) : undefined,
        pageNum.value,
        pageSize.value,
        sortBy.value,
        sortDirection.value,
        statusFilters.value.length > 0
          ? (statusFilters.value.map((o) => o.id) as AlertStatusEnum[])
          : undefined,
        severityFilters.value.length > 0 ? severityFilters.value.map((o) => o.id) : undefined,
        systemKeyFilters.value.length > 0 ? systemKeyFilters.value.map((o) => o.id) : undefined,
        alertnameFilters.value.length > 0 ? alertnameFilters.value.map((o) => o.id) : undefined,
      ),
    staleTime: ALERTS_REFETCH_INTERVAL_SECONDS * 1000,
    autoRefetch: shouldAutoRefetch,
  })

  const clearFilters = () => {
    organizationIds.value = []
    severityFilters.value = []
    systemKeyFilters.value = []
    alertnameFilters.value = []
    resetStatusFilter()
    pageNum.value = 1
  }

  const resetStatusFilter = () => {
    statusFilters.value = []
  }

  const areDefaultFiltersApplied = () => {
    return (
      !organizationIds.value.length &&
      !statusFilters.value.length &&
      !severityFilters.value.length &&
      !systemKeyFilters.value.length &&
      !alertnameFilters.value.length
    )
  }

  // load table page size from storage
  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(ALERTS_TABLE_ID)
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

  // reset to first page when status filter changes
  watch(
    () => statusFilters.value,
    () => {
      pageNum.value = 1
    },
    { deep: true },
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
    organizationIds,
    pageNum,
    pageSize,
    sortBy,
    sortDirection,
    statusFilters,
    severityFilters,
    systemKeyFilters,
    alertnameFilters,
    clearFilters,
    resetStatusFilter,
    areDefaultFiltersApplied,
  }
})
