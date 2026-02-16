//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { canReadDistributors } from '@/lib/permissions'
import { getDistributorDetail } from '@/lib/organizations/distributorDetail'
import { DISTRIBUTORS_KEY } from '@/lib/organizations/distributors'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'

export const useDistributorDetail = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [DISTRIBUTORS_KEY, route.params.distributorId],
    enabled: () => !!loginStore.jwtToken && canReadDistributors() && !!route.params.distributorId,
    query: () => getDistributorDetail(route.params.distributorId as string),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
