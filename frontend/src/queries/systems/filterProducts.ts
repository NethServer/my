//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { FILTER_PRODUCTS_KEY, getFilterProducts } from '@/lib/systems/filterProducts'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useFilterProducts = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [FILTER_PRODUCTS_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getFilterProducts(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
