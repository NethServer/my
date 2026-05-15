//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getAlertsHistory, ALERTS_HISTORY_KEY } from '@/lib/alerts'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref } from 'vue'

export const useAlertsHistory = defineQuery(() => {
  const loginStore = useLoginStore()
  const organizationIds = ref<string[]>([])
  const pageNum = ref(1)
  const pageSize = ref(50)
  const sortBy = ref('starts_at')
  const sortDirection = ref<'asc' | 'desc'>('desc')
  const includeDescendants = ref(false)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      ALERTS_HISTORY_KEY,
      organizationIds.value.join(','),
      pageNum.value,
      pageSize.value,
      sortBy.value,
      sortDirection.value,
      includeDescendants.value,
    ],
    enabled: () => !!loginStore.jwtToken,
    query: () =>
      getAlertsHistory(
        organizationIds.value.length > 0 ? organizationIds.value : undefined,
        pageNum.value,
        pageSize.value,
        sortBy.value,
        sortDirection.value,
        includeDescendants.value ? 'descendants' : undefined,
      ),
  })

  return {
    ...rest,
    state,
    asyncStatus,
    organizationIds,
    pageNum,
    pageSize,
    sortBy,
    sortDirection,
    includeDescendants,
  }
})
