//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getDistributors } from '@/lib/distributors'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useDistributors = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => ['distributors'],
    enabled: () => !!loginStore.jwtToken,
    query: getDistributors,
  })
  return { ...rest, distributors: state, distributorsAsyncStatus: asyncStatus }
})
