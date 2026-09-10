//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getSystems } from '@/lib/systems/systems'
import { RECENT_SYSTEMS_KEY } from '@/lib/dashboard/widgetData'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref } from 'vue'

export const useRecentSystems = defineQuery(() => {
  const loginStore = useLoginStore()
  const pageSize = ref(5)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [RECENT_SYSTEMS_KEY, pageSize.value],
    enabled: () => !!loginStore.jwtToken,
    query: () => getSystems(1, pageSize.value, '', [], [], [], [], [], true, 'created_at', true),
  })

  return {
    ...rest,
    state,
    asyncStatus,
    pageSize,
  }
})
