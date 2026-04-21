//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getSystemAlertHistory, ALERT_HISTORY_KEY } from '@/lib/alerting'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import { useLoginStore } from '@/stores/login'
import { ALERT_HISTORY_TABLE_ID } from '@/lib/alerting'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'

export const useAlertHistory = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()
  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const sortBy = ref('created_at')
  const sortDescending = ref(true)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      ALERT_HISTORY_KEY,
      route.params.systemId,
      pageNum.value,
      pageSize.value,
      sortBy.value,
      sortDescending.value,
    ],
    enabled: () => !!loginStore.jwtToken && !!route.params.systemId,
    query: () =>
      getSystemAlertHistory(
        route.params.systemId as string,
        pageNum.value,
        pageSize.value,
        sortBy.value,
        sortDescending.value,
      ),
  })

  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(ALERT_HISTORY_TABLE_ID)
      }
    },
    { immediate: true },
  )

  return {
    ...rest,
    state,
    asyncStatus,
    pageNum,
    pageSize,
    sortBy,
    sortDescending,
  }
})
