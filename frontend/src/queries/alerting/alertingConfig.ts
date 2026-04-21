//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getAlertingConfig, ALERTING_CONFIG_KEY } from '@/lib/alerting'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref } from 'vue'

export const useAlertingConfig = defineQuery(() => {
  const loginStore = useLoginStore()
  const organizationId = ref(loginStore.userInfo?.organization_id || '')
  const format = ref<'yaml' | undefined>(undefined)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [ALERTING_CONFIG_KEY, organizationId.value, format.value ?? 'json'],
    enabled: () => !!loginStore.jwtToken && !!organizationId.value,
    query: () => getAlertingConfig(organizationId.value, format.value),
  })

  return {
    ...rest,
    state,
    asyncStatus,
    organizationId,
    format,
  }
})
