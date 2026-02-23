//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { SYSTEM_FILTERS_KEY, getSystemFilters } from '@/lib/systems/systemFilters'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useSystemFilters = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [SYSTEM_FILTERS_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getSystemFilters(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
