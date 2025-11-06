//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { CREATED_BY_FILTER_KEY, getCreatedByFilter } from '@/lib/systems/createdByFilter'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useCreatedByFilter = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [CREATED_BY_FILTER_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getCreatedByFilter(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
