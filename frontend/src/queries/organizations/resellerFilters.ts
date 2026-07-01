//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { RESELLER_FILTERS_KEY, getResellerFilters } from '@/lib/organizations/resellerFilters'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useResellerFilters = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [RESELLER_FILTERS_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getResellerFilters(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
