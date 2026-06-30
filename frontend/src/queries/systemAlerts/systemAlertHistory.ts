//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  getSystemAlertHistory,
  ALERTS_REFETCH_INTERVAL_SECONDS,
  SYSTEM_ALERT_HISTORY_KEY,
  SYSTEM_ALERT_HISTORY_TABLE_ID,
} from '@/lib/alerts'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import type { NeDropdownFilterV2Option } from '@nethesis/vue-components'

export const useSystemAlertHistory = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const isHistoryEnabled = ref(false)
  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const sortBy = ref('starts_at')
  const sortDescending = ref(true)
  const severityFilters = ref<NeDropdownFilterV2Option[]>([])
  const alertnameFilters = ref<NeDropdownFilterV2Option[]>([])

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      SYSTEM_ALERT_HISTORY_KEY,
      route.params.systemId,
      pageNum.value,
      pageSize.value,
      sortBy.value,
      sortDescending.value,
      severityFilters.value.map((o) => o.id).join(','),
      alertnameFilters.value.map((o) => o.id).join(','),
    ],
    enabled: () => !!loginStore.jwtToken && !!route.params.systemId && isHistoryEnabled.value,
    query: () =>
      getSystemAlertHistory(
        route.params.systemId as string,
        pageNum.value,
        pageSize.value,
        sortBy.value,
        sortDescending.value,
        severityFilters.value.length > 0 ? severityFilters.value.map((o) => o.id) : undefined,
        alertnameFilters.value.length > 0 ? alertnameFilters.value.map((o) => o.id) : undefined,
      ),
    staleTime: ALERTS_REFETCH_INTERVAL_SECONDS * 1000,
    autoRefetch: true,
  })

  const areDefaultFiltersApplied = () =>
    !severityFilters.value.length && !alertnameFilters.value.length

  const clearFilters = () => {
    severityFilters.value = []
    alertnameFilters.value = []
    pageNum.value = 1
  }

  // load table page size from storage
  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(SYSTEM_ALERT_HISTORY_TABLE_ID)
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
    isHistoryEnabled,
    pageNum,
    pageSize,
    sortBy,
    sortDescending,
    severityFilters,
    alertnameFilters,
    areDefaultFiltersApplied,
    clearFilters,
  }
})
