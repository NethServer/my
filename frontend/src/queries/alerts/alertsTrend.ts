//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getAlertsTrend, ALERTS_TREND_KEY } from '@/lib/alerts'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref } from 'vue'

export const useAlertsTrend = defineQuery(() => {
  const loginStore = useLoginStore()
  const organizationIds = ref<string[]>([])
  const includeDescendants = ref(false)
  const period = ref<7 | 30 | 180 | 365>(7)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      ALERTS_TREND_KEY,
      organizationIds.value.join(','),
      includeDescendants.value,
      period.value,
    ],
    enabled: () => !!loginStore.jwtToken,
    query: () =>
      getAlertsTrend(
        organizationIds.value.length > 0 ? organizationIds.value : undefined,
        includeDescendants.value ? 'descendants' : undefined,
        period.value,
      ),
  })

  return {
    ...rest,
    state,
    asyncStatus,
    organizationIds,
    includeDescendants,
    period,
  }
})
