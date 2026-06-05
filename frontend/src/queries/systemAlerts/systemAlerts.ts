//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import type { AlertStatusEnum } from '@/lib/alerts'
import {
  getSystemActiveAlerts,
  SYSTEM_ALERTS_KEY,
  SYSTEM_ALERTS_TABLE_ID,
} from '@/lib/systemAlerts'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'

export const useSystemAlerts = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const sortBy = ref<'starts_at' | 'severity' | 'alertname' | 'status'>('starts_at')
  const sortDirection = ref<'asc' | 'desc'>('desc')
  const severityFilters = ref<string[]>([])
  const alertnameFilters = ref<string[]>([])
  const statusFilters = ref<AlertStatusEnum[]>(['active'])

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      SYSTEM_ALERTS_KEY,
      route.params.systemId,
      pageNum.value,
      pageSize.value,
      sortBy.value,
      sortDirection.value,
      severityFilters.value.join(','),
      alertnameFilters.value.join(','),
      statusFilters.value.join(','),
    ],
    enabled: () => !!loginStore.jwtToken && !!route.params.systemId,
    query: () =>
      getSystemActiveAlerts(
        route.params.systemId as string,
        pageNum.value,
        pageSize.value,
        sortBy.value,
        sortDirection.value,
        statusFilters.value.length > 0 ? statusFilters.value : undefined,
        severityFilters.value.length > 0 ? severityFilters.value : undefined,
        alertnameFilters.value.length > 0 ? alertnameFilters.value : undefined,
      ),
  })

  const areDefaultFiltersApplied = () =>
    !severityFilters.value.length &&
    !alertnameFilters.value.length &&
    statusFilters.value.length === 1 &&
    statusFilters.value.includes('active') &&
    !statusFilters.value.includes('suppressed')

  const resetFilters = () => {
    severityFilters.value = []
    alertnameFilters.value = []
    resetStatusFilter()
    pageNum.value = 1
  }

  const resetStatusFilter = () => {
    statusFilters.value = ['active']
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
    resetFilters,
    resetStatusFilter,
  }
})
