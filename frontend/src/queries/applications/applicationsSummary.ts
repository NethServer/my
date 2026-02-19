//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  APPLICATIONS_SUMMARY_KEY,
  getApplicationsSummary,
} from '@/lib/applications/applicationsSummary'
import { canReadApplications } from '@/lib/permissions'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'

export const useApplicationsSummary = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [APPLICATIONS_SUMMARY_KEY, route.params.companyId],
    enabled: () => !!loginStore.jwtToken && canReadApplications() && !!route.params.companyId,
    query: () => getApplicationsSummary(route.params.companyId as string, 1, 5, 'count', true),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
