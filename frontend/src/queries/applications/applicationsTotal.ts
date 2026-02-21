//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { APPLICATIONS_TOTAL_KEY, getApplicationsTotal } from '@/lib/applications/applications'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useApplicationsTotal = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [APPLICATIONS_TOTAL_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: getApplicationsTotal,
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
