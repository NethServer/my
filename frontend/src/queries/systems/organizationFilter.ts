//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getOrganizationFilter, ORGANIZATION_FILTER_KEY } from '@/lib/systems/organizationFilter'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useOrganizationFilter = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [ORGANIZATION_FILTER_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getOrganizationFilter(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
