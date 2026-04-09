//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getAlerts, ALERTING_ALERTS_KEY } from '@/lib/alerting'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref } from 'vue'

export const useAlerts = defineQuery(() => {
  const loginStore = useLoginStore()
  const organizationId = ref(loginStore.userInfo?.organization_id || '')
  const stateFilter = ref<string[]>([])
  const severityFilter = ref<string[]>([])
  const systemKeyFilter = ref('')

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      ALERTING_ALERTS_KEY,
      organizationId.value,
      stateFilter.value.join(','),
      severityFilter.value.join(','),
      systemKeyFilter.value,
    ],
    enabled: () => !!loginStore.jwtToken && !!organizationId.value,
    query: () =>
      getAlerts(
        organizationId.value,
        stateFilter.value[0] || undefined,
        severityFilter.value[0] || undefined,
        systemKeyFilter.value || undefined,
      ),
  })

  const resetFilters = () => {
    stateFilter.value = []
    severityFilter.value = []
    systemKeyFilter.value = ''
  }

  const areDefaultFiltersApplied = () => {
    return !stateFilter.value.length && !severityFilter.value.length && !systemKeyFilter.value
  }

  return {
    ...rest,
    state,
    asyncStatus,
    organizationId,
    stateFilter,
    severityFilter,
    systemKeyFilter,
    resetFilters,
    areDefaultFiltersApplied,
  }
})
