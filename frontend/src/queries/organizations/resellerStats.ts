//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { canReadResellers } from '@/lib/permissions'
import { getResellerStats } from '@/lib/organizations/resellerDetail'
import { RESELLERS_KEY } from '@/lib/organizations/resellers'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'

export const useResellerStats = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [RESELLERS_KEY, 'stats', route.params.companyId],
    enabled: () => !!loginStore.jwtToken && canReadResellers() && !!route.params.companyId,
    query: () => getResellerStats(route.params.companyId as string),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
