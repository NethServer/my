//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { canReadResellers } from '@/lib/permissions'
import { getResellerDetail } from '@/lib/organizations/resellerDetail'
import { RESELLERS_KEY } from '@/lib/organizations/resellers'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'

export const useResellerDetail = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [RESELLERS_KEY, route.params.companyId],
    enabled: () => !!loginStore.jwtToken && canReadResellers() && !!route.params.companyId,
    query: () => getResellerDetail(route.params.companyId as string),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
