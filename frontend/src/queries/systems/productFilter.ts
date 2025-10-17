//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { PRODUCT_FILTER_KEY, getProductFilter } from '@/lib/systems/productFilter'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useProductFilter = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [PRODUCT_FILTER_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getProductFilter(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
