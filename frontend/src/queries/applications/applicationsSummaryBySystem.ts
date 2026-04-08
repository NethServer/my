//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  APPLICATIONS_SUMMARY_KEY,
  getApplicationsSummaryBySystem,
} from '@/lib/applications/applicationsSummary'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'

export const useApplicationsSummaryBySystem = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [APPLICATIONS_SUMMARY_KEY, route.params.systemId],
    enabled: () => !!loginStore.jwtToken && !!route.params.systemId,
    query: () =>
      getApplicationsSummaryBySystem(route.params.systemId as string, 1, 5, 'count', true),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
