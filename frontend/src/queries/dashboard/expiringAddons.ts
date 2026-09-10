//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  getExpiringGrants,
  EXPIRING_ADDONS_KEY,
  EXPIRING_ADDONS_WINDOW_DAYS,
} from '@/lib/dashboard/widgetData'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref } from 'vue'

export const useExpiringAddons = defineQuery(() => {
  const loginStore = useLoginStore()
  const withinDays = ref(EXPIRING_ADDONS_WINDOW_DAYS)
  const pageSize = ref(5)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [EXPIRING_ADDONS_KEY, withinDays.value, pageSize.value],
    enabled: () => !!loginStore.jwtToken,
    query: () => getExpiringGrants(withinDays.value, pageSize.value),
  })

  return {
    ...rest,
    state,
    asyncStatus,
    withinDays,
    pageSize,
  }
})
