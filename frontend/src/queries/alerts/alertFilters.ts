//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { ALERT_FILTERS_KEY, getAlertFilters } from '@/lib/alertFilters'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useAlertFilters = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [ALERT_FILTERS_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getAlertFilters(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
