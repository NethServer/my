//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getAlertsSilences, ALERTS_SILENCES_KEY } from '@/lib/alerts'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref } from 'vue'

//// delete if unused

export const useAlertsSilences = defineQuery(() => {
  const loginStore = useLoginStore()
  const organizationIds = ref<string[]>([])
  const includeDescendants = ref(false)
  const systemKeyFilters = ref<string[]>([])

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      ALERTS_SILENCES_KEY,
      organizationIds.value.join(','),
      includeDescendants.value,
      systemKeyFilters.value.join(','),
    ],
    enabled: () => !!loginStore.jwtToken,
    query: () =>
      getAlertsSilences(
        organizationIds.value.length > 0 ? organizationIds.value : undefined,
        includeDescendants.value ? 'descendants' : undefined,
        systemKeyFilters.value.length > 0 ? systemKeyFilters.value : undefined,
      ),
  })

  return {
    ...rest,
    state,
    asyncStatus,
    organizationIds,
    includeDescendants,
    systemKeyFilters,
  }
})
