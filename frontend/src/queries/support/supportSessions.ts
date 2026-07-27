//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { canConnectSystems } from '@/lib/permissions'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import {
  getSupportSessions,
  SUPPORT_SESSIONS_KEY,
  SUPPORT_SESSIONS_TABLE_ID,
} from '@/lib/support/support'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref, watch } from 'vue'

export const useSupportSessions = defineQuery(() => {
  const loginStore = useLoginStore()

  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const sortBy = ref('started_at')
  const sortDescending = ref(true)

  const { state, asyncStatus, refetch, ...rest } = useQuery({
    key: () => [
      SUPPORT_SESSIONS_KEY,
      {
        pageNum: pageNum.value,
        pageSize: pageSize.value,
        sortBy: sortBy.value,
        sortDirection: sortDescending.value,
      },
    ],
    enabled: () => !!loginStore.jwtToken && canConnectSystems(),
    query: () =>
      getSupportSessions(pageNum.value, pageSize.value, [], sortBy.value, sortDescending.value),
    refetchOnWindowFocus: true,
  })

  // load table page size from storage
  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(SUPPORT_SESSIONS_TABLE_ID)
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
    refetch,
    pageNum,
    pageSize,
    sortBy,
    sortDescending,
  }
})
