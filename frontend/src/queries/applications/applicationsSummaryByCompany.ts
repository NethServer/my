//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  APPLICATIONS_SUMMARY_KEY,
  getApplicationsSummaryByCompany,
} from '@/lib/applications/applicationsSummary'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'

export const useApplicationsSummaryByCompany = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [APPLICATIONS_SUMMARY_KEY, route.params.companyId],
    enabled: () => !!loginStore.jwtToken && !!route.params.companyId,
    query: () =>
      getApplicationsSummaryByCompany(route.params.companyId as string, 1, 5, 'count', true),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
