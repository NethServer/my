//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getAlertsStats, ALERTS_STATS_KEY } from '@/lib/alerts'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref } from 'vue'

export const useAlertsStats = defineQuery(() => {
  const loginStore = useLoginStore()
  const organizationIds = ref<string[]>([])
  const includeDescendants = ref(false)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [ALERTS_STATS_KEY, organizationIds.value.join(','), includeDescendants.value],
    enabled: () => !!loginStore.jwtToken,
    query: () =>
      getAlertsStats(
        organizationIds.value.length > 0 ? organizationIds.value : undefined,
        includeDescendants.value ? 'descendants' : undefined,
      ),
  })

  return {
    ...rest,
    state,
    asyncStatus,
    organizationIds,
    includeDescendants,
  }
})
