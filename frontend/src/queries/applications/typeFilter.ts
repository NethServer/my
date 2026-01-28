//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { APPLICATION_TYPE_FILTER_KEY, getTypeFilter } from '@/lib/applications/typeFilter'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useTypeFilter = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [APPLICATION_TYPE_FILTER_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getTypeFilter(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
