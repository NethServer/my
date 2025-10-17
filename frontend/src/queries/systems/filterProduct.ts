//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { FILTER_PRODUCT_KEY, getFilterProduct } from '@/lib/systems/filterProduct'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useFilterProduct = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [FILTER_PRODUCT_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getFilterProduct(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
