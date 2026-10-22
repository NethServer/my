//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getAlertsConfig, ALERTS_CONFIG_KEY } from '@/lib/alerts'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref } from 'vue'

const PROPAGATION_REFETCH_INTERVAL_SECONDS = 3

export const useAlertsConfig = defineQuery(() => {
  const loginStore = useLoginStore()
  const format = ref<'yaml' | undefined>(undefined)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [ALERTS_CONFIG_KEY, format.value ?? 'json'],
    enabled: () => !!loginStore.jwtToken,
    query: () => getAlertsConfig(format.value),
    // Poll only while a background propagation of the layer is running.
    autoRefetch: (current) =>
      current.data?.propagation?.state === 'running'
        ? PROPAGATION_REFETCH_INTERVAL_SECONDS * 1000
        : false,
  })

  return {
    ...rest,
    state,
    asyncStatus,
    format,
  }
})
