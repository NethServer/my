//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { FILTER_VERSION_KEY, getFilterVersion } from '@/lib/systems/filterVersion'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useFilterVersion = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [FILTER_VERSION_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getFilterVersion(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
