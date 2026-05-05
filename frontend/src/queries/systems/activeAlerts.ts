//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getSystemActiveAlerts } from '@/lib/alerting'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'

export const SYSTEM_ACTIVE_ALERTS_KEY = 'systemActiveAlerts'

export const useSystemActiveAlerts = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [SYSTEM_ACTIVE_ALERTS_KEY, route.params.systemId],
    enabled: () => !!loginStore.jwtToken && !!route.params.systemId,
    query: () => getSystemActiveAlerts(route.params.systemId as string),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
