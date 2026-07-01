//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { CUSTOMER_FILTERS_KEY, getCustomerFilters } from '@/lib/organizations/customerFilters'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useCustomerFilters = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [CUSTOMER_FILTERS_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getCustomerFilters(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
