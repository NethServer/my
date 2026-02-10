//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  getOrganizationFilter,
  USERS_ORGANIZATION_FILTER_KEY,
} from '@/lib/users/organizationFilter'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useOrganizationFilter = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [USERS_ORGANIZATION_FILTER_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getOrganizationFilter(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
