//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getSessions, SESSIONS_KEY, SESSIONS_TABLE_ID } from '@/lib/impersonationSessions'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref, watch } from 'vue'

export const useImpersonationSessions = defineQuery(() => {
  const loginStore = useLoginStore()
  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      SESSIONS_KEY,
      {
        pageNum: pageNum.value,
        pageSize: pageSize.value,
      },
    ],
    enabled: () => !!loginStore.jwtToken,
    query: () => getSessions(pageNum.value, pageSize.value),
  })

  // load table page size from storage
  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(SESSIONS_TABLE_ID)
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
  }
})
