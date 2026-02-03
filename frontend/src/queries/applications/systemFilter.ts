//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { APPLICATION_SYSTEM_FILTER_KEY, getSystemFilter } from '@/lib/applications/systemFilter'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useSystemFilter = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [APPLICATION_SYSTEM_FILTER_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getSystemFilter(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
