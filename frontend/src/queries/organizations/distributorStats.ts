//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { canReadDistributors } from '@/lib/permissions'
import { getDistributorStats } from '@/lib/organizations/distributorDetail'
import { DISTRIBUTORS_KEY } from '@/lib/organizations/distributors'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'

export const useDistributorStats = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [DISTRIBUTORS_KEY, 'stats', route.params.companyId],
    enabled: () => !!loginStore.jwtToken && canReadDistributors() && !!route.params.companyId,
    query: () => getDistributorStats(route.params.companyId as string),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
