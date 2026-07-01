//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  DISTRIBUTOR_FILTERS_KEY,
  getDistributorFilters,
} from '@/lib/organizations/distributorFilters'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useDistributorFilters = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [DISTRIBUTOR_FILTERS_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getDistributorFilters(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
