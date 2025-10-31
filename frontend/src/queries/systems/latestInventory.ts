//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { canReadSystems } from '@/lib/permissions'
import { getLatestInventory, LATEST_INVENTORY_KEY } from '@/lib/systems/inventory'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'

export const useLatestInventory = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [LATEST_INVENTORY_KEY, route.params.systemId],
    enabled: () => !!loginStore.jwtToken && canReadSystems() && !!route.params.systemId,
    query: () => getLatestInventory(route.params.systemId as string),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
