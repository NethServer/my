//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getAlertsStats, ALERTS_STATS_KEY } from '@/lib/dashboard/widgetData'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref } from 'vue'

// Shared on purpose: the severity breakdown and the noisiest-alerts list are two
// widgets over one response, so a board showing both still makes one request.
export const useAlertsStats = defineQuery(() => {
  const loginStore = useLoginStore()
  const top = ref(5)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [ALERTS_STATS_KEY, top.value],
    enabled: () => !!loginStore.jwtToken,
    query: () => getAlertsStats(top.value),
  })

  return {
    ...rest,
    state,
    asyncStatus,
    top,
  }
})
