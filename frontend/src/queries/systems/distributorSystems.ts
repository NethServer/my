//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { canReadDistributors } from '@/lib/permissions'
import { getSystems, SYSTEMS_KEY } from '@/lib/systems/systems'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'
import { DISTRIBUTORS_KEY } from '@/lib/organizations/distributors'

export const useDistributorSystems = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [SYSTEMS_KEY, DISTRIBUTORS_KEY, route.params.companyId],
    enabled: () => !!loginStore.jwtToken && canReadDistributors() && !!route.params.companyId,
    query: () =>
      getSystems(
        1,
        5, // retrieve only a few systems for the distributor detail view
        '',
        [],
        [],
        [],
        [],
        route.params.companyId ? [route.params.companyId as string] : [],
        'created_at',
        true,
      ),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
