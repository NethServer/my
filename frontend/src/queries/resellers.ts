//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getResellers, RESELLERS_KEY } from '@/lib/resellers'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useResellers = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [RESELLERS_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: getResellers,
  })
  return { ...rest, resellers: state, resellersAsyncStatus: asyncStatus }
})
