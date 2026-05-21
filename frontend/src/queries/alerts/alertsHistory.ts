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
  const pageSize = ref(20)
  const sortBy = ref('created_at')
  const sortDirection = ref<'asc' | 'desc'>('desc')
  const includeDescendants = ref(false)
  const fromDate = ref<string | undefined>(undefined)
  const toDate = ref<string | undefined>(undefined)
  const systemKeyFilters = ref<string[]>([])
  const alertnameFilters = ref<string[]>([])
  const severityFilters = ref<string[]>([])
  const statusFilters = ref<string[]>([])

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      ALERTS_HISTORY_KEY,
      organizationIds.value.join(','),
      pageNum.value,
      pageSize.value,
      sortBy.value,
      sortDirection.value,
      includeDescendants.value,
      fromDate.value ?? '',
      toDate.value ?? '',
      systemKeyFilters.value.join(','),
      alertnameFilters.value.join(','),
      severityFilters.value.join(','),
      statusFilters.value.join(','),
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
        fromDate.value,
        toDate.value,
        systemKeyFilters.value.length > 0 ? systemKeyFilters.value : undefined,
        alertnameFilters.value.length > 0 ? alertnameFilters.value : undefined,
        severityFilters.value.length > 0 ? severityFilters.value : undefined,
        statusFilters.value.length > 0 ? statusFilters.value : undefined,
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
    fromDate,
    toDate,
    systemKeyFilters,
    alertnameFilters,
    severityFilters,
    statusFilters,
  }
})
