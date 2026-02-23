//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { USER_FILTERS_KEY, getUserFilters } from '@/lib/users/userFilters'
import { canReadUsers } from '@/lib/permissions'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useUserFilters = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [USER_FILTERS_KEY],
    enabled: () => !!loginStore.jwtToken && canReadUsers(),
    query: () => getUserFilters(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
