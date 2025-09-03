//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getConsent } from '@/lib/impersonationConsent'
import { IMPERSONATION_CONSENT_KEY } from '@/lib/impersonationConsent'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useImpersonationConsent = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [IMPERSONATION_CONSENT_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getConsent(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
