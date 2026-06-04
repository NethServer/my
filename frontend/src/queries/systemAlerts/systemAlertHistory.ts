//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  getSystemAlertHistory,
  SYSTEM_ALERT_HISTORY_KEY,
  SYSTEM_ALERT_HISTORY_TABLE_ID,
} from '@/lib/alerts'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'

export const useSystemAlertHistory = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const sortBy = ref('starts_at')
  const sortDescending = ref(true)
  const severityFilters = ref<string[]>([])
  const alertnameFilters = ref<string[]>([])
  const statusFilters = ref<string[]>([])

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      SYSTEM_ALERT_HISTORY_KEY,
      route.params.systemId,
      pageNum.value,
      pageSize.value,
      sortBy.value,
      sortDescending.value,
      severityFilters.value.join(','),
      alertnameFilters.value.join(','),
      statusFilters.value.join(','),
    ],
    enabled: () => !!loginStore.jwtToken && !!route.params.systemId,
    query: () =>
      getSystemAlertHistory(
        route.params.systemId as string,
        pageNum.value,
        pageSize.value,
        sortBy.value,
        sortDescending.value,
        severityFilters.value.length > 0 ? severityFilters.value : undefined,
        alertnameFilters.value.length > 0 ? alertnameFilters.value : undefined,
        statusFilters.value.length > 0 ? statusFilters.value : undefined,
      ),
  })

  const areDefaultFiltersApplied = () =>
    !severityFilters.value.length && !alertnameFilters.value.length && !statusFilters.value.length

  const resetFilters = () => {
    severityFilters.value = []
    alertnameFilters.value = []
    statusFilters.value = []
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
    pageNum,
    pageSize,
    sortBy,
    sortDescending,
    severityFilters,
    alertnameFilters,
    statusFilters,
    areDefaultFiltersApplied,
    resetFilters,
  }
})
