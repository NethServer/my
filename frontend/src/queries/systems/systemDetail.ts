//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { canReadSystems } from '@/lib/permissions'
import { getSystemDetail } from '@/lib/systems/systemDetail'
import { SYSTEMS_KEY } from '@/lib/systems/systems'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'

export const useSystemDetail = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [SYSTEMS_KEY, route.params.systemId],
    enabled: () => !!loginStore.jwtToken && canReadSystems(),
    query: () => getSystemDetail(route.params.systemId as string),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
