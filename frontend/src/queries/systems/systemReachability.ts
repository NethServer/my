// Copyright (C) 2026 Nethesis S.r.l.
// SPDX-License-Identifier: GPL-3.0-or-later

import { checkSystemReachability, SYSTEM_REACHABILITY_KEY } from '@/lib/systems/systems'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'

export const useSystemReachability = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [SYSTEM_REACHABILITY_KEY, route.params.systemId],
    enabled: () => !!loginStore.jwtToken && !!route.params.systemId,
    query: () => checkSystemReachability(route.params.systemId as string),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
