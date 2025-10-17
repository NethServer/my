//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { VERSION_FILTER_KEY, getVersionFilter } from '@/lib/systems/versionFilter'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useVersionFilter = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [VERSION_FILTER_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getVersionFilter(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
