//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  APPLICATION_FILTERS_KEY,
  getApplicationFilters,
} from '@/lib/applications/applicationFilters'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useApplicationFilters = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [APPLICATION_FILTERS_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getApplicationFilters(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
