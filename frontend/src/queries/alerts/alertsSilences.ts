//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getAlertsSilences, ALERTS_SILENCES_KEY } from '@/lib/alerts'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref } from 'vue'

export const useAlertsSilences = defineQuery(() => {
  const loginStore = useLoginStore()
  const organizationIds = ref<string[]>([])
  const pageNum = ref(1)
  const pageSize = ref(50)
  const includeDescendants = ref(false)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      ALERTS_SILENCES_KEY,
      organizationIds.value.join(','),
      pageNum.value,
      pageSize.value,
      includeDescendants.value,
    ],
    enabled: () => !!loginStore.jwtToken,
    query: () =>
      getAlertsSilences(
        organizationIds.value.length > 0 ? organizationIds.value : undefined,
        pageNum.value,
        pageSize.value,
        includeDescendants.value ? 'descendants' : undefined,
      ),
  })

  return {
    ...rest,
    state,
    asyncStatus,
    organizationIds,
    pageNum,
    pageSize,
    includeDescendants,
  }
})
