//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { FILTER_CREATED_BY_KEY, getFilterCreatedBy } from '@/lib/systems/filterCreatedBy'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useFilterCreatedBy = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [FILTER_CREATED_BY_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getFilterCreatedBy(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
