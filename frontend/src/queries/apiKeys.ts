//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { API_KEYS_KEY, getApiKeys } from '@/lib/apiKeys'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useApiKeys = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [API_KEYS_KEY],
    // Owner has no local user row and key management is disabled while
    // impersonating, so the endpoint is unavailable in those cases.
    enabled: () => !!loginStore.jwtToken && !loginStore.isOwner && !loginStore.isImpersonating,
    query: () => getApiKeys(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
