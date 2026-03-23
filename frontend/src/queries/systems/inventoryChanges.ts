//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getInventoryChanges, INVENTORY_CHANGES_KEY } from '@/lib/systems/inventoryChanges'
import { INVENTORY_MOCK_ENABLED, mockInventoryChanges } from '@/lib/systems/inventoryMocks'
import { canReadSystems } from '@/lib/permissions'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'

export const useInventoryChanges = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [INVENTORY_CHANGES_KEY, route.params.systemId],
    enabled: () => !!loginStore.jwtToken && canReadSystems() && !!route.params.systemId,
    query: () => {
      const apiCall = getInventoryChanges(route.params.systemId as string)
      if (INVENTORY_MOCK_ENABLED) {
        apiCall.catch(() => {})
        return Promise.resolve(mockInventoryChanges)
      }
      return apiCall
    },
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
