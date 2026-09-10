//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getAlerts, ALERTS_REFETCH_INTERVAL_SECONDS } from '@/lib/alerts'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref } from 'vue'

export const FIRING_ALERTS_KEY = 'dashboardFiringAlerts'

// The only widget on the board that polls. A hidden tab is not watching, so it
// does not refetch — the same guard the alerts page uses.
const shouldAutoRefetch = () => document.visibilityState === 'visible'

export const useFiringAlerts = defineQuery(() => {
  const loginStore = useLoginStore()
  const pageSize = ref(5)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [FIRING_ALERTS_KEY, pageSize.value],
    enabled: () => !!loginStore.jwtToken,
    query: () => getAlerts(undefined, 1, pageSize.value, 'severity', 'desc'),
    staleTime: ALERTS_REFETCH_INTERVAL_SECONDS * 1000,
    autoRefetch: shouldAutoRefetch,
  })

  return {
    ...rest,
    state,
    asyncStatus,
    pageSize,
  }
})
